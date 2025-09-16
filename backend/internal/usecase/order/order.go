package order

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/engine"
	ledgerRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/ledger"
	orderRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/order"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/router/middleware"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
	"github.com/jmoiron/sqlx"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	. "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

type OrderUseCase interface {
	AddOrder(ctx context.Context, ticker string, side model.Side, price model.Price, quantity model.Quantity, orderType model.OrderType) (trades []*model.Trade, orderID model.OrderId, err error)

	CancelOrder(ctx context.Context, orderID model.OrderId) error

	ModifyOrder(ctx context.Context, modify model.OrderModify, orderType model.OrderType) (trades []*model.Trade, err error)

	OrderSize(ctx context.Context) int

	GetTopOfBook(ctx context.Context) *model.TopOfBook

	GetOrderInfos(ctx context.Context) *model.MarketDepth

	RegisterTradeHandler(handler TradeHandler)
	GetOrderByUserId(ctx context.Context, userId int64) (*[]orderRepository.OrderRecord, error)
}

type orderUseCaseImpl struct {
	orderBookEngine engine.OrderBookEngine // hold interface by value, not pointer to interface

	tbClient *tb.Client

	// config

	ledgerID uint32 // TigerBeetle ledger identifier you choose (e.g., 1)

	// account IDs: typically 128-bit IDs you decide. Here we assume you know how to map user/account IDs.

	// For demo, we’ll assume a single “exchange escrow” account.

	escrowAccount Uint128

	tradeHandler TradeHandler
	orderRepo    *orderRepository.OrderRepository
	ledgerRepo   *ledgerRepository.LedgerRepository
	db           *sqlx.DB
}

// NewOrderUseCase constructs with an already-initialized TB client.
type TradeHandler func(model.Trade)

type OrderUseCaseOpts struct {
	OrderBookEngine engine.OrderBookEngine
	TBLedgerID      uint32
	EscrowAccount   Uint128 // (optional global escrow, but we'll use per-ticker escrow accounts)
	OrderRepo       *orderRepository.OrderRepository
	LedgerRepo      *ledgerRepository.LedgerRepository
	Db              *sqlx.DB
	TbClient        *tb.Client
}

func NewOrderUseCase(ctx context.Context, opts OrderUseCaseOpts) OrderUseCase {
	return &orderUseCaseImpl{
		orderBookEngine: opts.OrderBookEngine,
		tbClient:        opts.TbClient,
		ledgerID:        opts.TBLedgerID,
		escrowAccount:   opts.EscrowAccount,
		orderRepo:       opts.OrderRepo,
		ledgerRepo:      opts.LedgerRepo,
		db:              opts.Db,
	}
}

func (ou *orderUseCaseImpl) RegisterTradeHandler(handler TradeHandler) {
	ou.tradeHandler = handler
}

// AddOrder writes any necessary pre-commit ledger entries (e.g., reserve funds), then submits to engine.
func (ou *orderUseCaseImpl) AddOrder(ctx context.Context, ticker string, side model.Side, price model.Price, quantity model.Quantity, orderType model.OrderType) ([]*model.Trade, model.OrderId, error) {

	orderID := model.OrderId(time.Now().UnixMilli())
	userID := *(ctx.Value(middleware.AuthKey{}).(*middleware.UserClaims))

	// Reserve funds for GTC orders (if needed)
	tx := ou.db.MustBeginTx(ctx, nil)
	defer tx.Rollback()

	assetTicker, err := (*ou.ledgerRepo).GetLedgerByTicker(ctx, tx, ticker)
	log.Printf("tickerledger %v err %v", assetTicker, err)
	if err != nil {
		return nil, 0, err
	}
	tickerID := assetTicker.ID

	if orderType == model.ORDER_GOOD_TILL_CANCEL {
		quoteTicker, err := (*ou.ledgerRepo).GetLedgerByTicker(ctx, tx, model.CASH_TICKER) // replace "USD" with your quote currency
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get quote ticker: %w", err)
		}
		// Fetch user's ledger accounts for reservation
		switch side {
		case model.BID:
			// Reserve fiat: debit user's currency account, credit currency escrow
			userCashAcct, err := (*ou.ledgerRepo).GetUserLedger(ctx, tx, userID.UserId, quoteTicker.ID)
			if err != nil {
				return nil, 0, err
			}
			// Amount = price * quantity (in smallest currency units)
			cashAmount := big.NewInt(1).Mul(big.NewInt(int64(price)), big.NewInt(int64(quantity)))
			userCashTb, err := stringToUint128(userCashAcct.TBAccountID)
			if err != nil {
				return nil, 0, err
			}
			tickerEscrow, err := stringToUint128(quoteTicker.EscrowAccountID)
			if err != nil {
				return nil, 0, err
			}
			err = ou.reserveFunds(ctx,
				userCashTb,   // user's fiat account (debit)
				tickerEscrow, // quote currency escrow account (credit)
				BigIntToUint128(*cashAmount),
				1001,
				model.CASH_LEDGER,
			)
			if err != nil {
				return nil, 0, fmt.Errorf("fund reservation failed: %w", err)
			}
		case model.ASK:
			userAssetAcct, err := (*ou.ledgerRepo).GetUserLedger(ctx, tx, userID.UserId, assetTicker.ID)
			if err != nil {
				return nil, 0, err
			}
			assetQty := big.NewInt(1).SetUint64(uint64(quantity))
			userAssetb, err := stringToUint128(userAssetAcct.TBAccountID)
			if err != nil {
				return nil, 0, err
			}
			tickerEscrow, err := stringToUint128(assetTicker.EscrowAccountID)
			if err != nil {
				return nil, 0, err
			}
			log.Printf("THIS IS RESULT OF ALL %v,%v", userAssetAcct, assetTicker)
			err = ou.reserveFunds(ctx,
				userAssetb,
				tickerEscrow,
				BigIntToUint128(*assetQty),
				1002,
				uint32(userAssetAcct.LedgerTbId),
			)
			if err != nil {
				return nil, 0, fmt.Errorf("asset reservation failed: %w", err)
			}
		}
	}

	// 3. Persist the new order in the database
	newOrderRecord := orderRepository.OrderRecord{
		ID:             uint64(orderID),
		UserID:         userID.UserId,
		TickerID:       tickerID,
		Side:           int8(side),
		TickerLedgerID: int64(ou.ledgerID),
		Type:           uint8(orderType),
		Quantity:       uint64(quantity),
		Price:          uint64(price),
		IsActive:       true,
	}

	err = (*ou.orderRepo).CreateOrder(ctx, tx, newOrderRecord)
	if err != nil {
		return nil, 0, fmt.Errorf("inserting order: %w", err)
	}

	// 4. Submit order to matching engine
	engineOrder := model.NewOrder(orderID, side, price, quantity, orderType)
	matchedTrades, matchErr := ou.orderBookEngine.AddOrder(engineOrder)
	if matchErr != nil {
		return nil, orderID, matchErr
	}

	// 5. Handle each resulting trade
	for _, tr := range matchedTrades {
		// Determine roles: identify which order was taker vs maker
		takerOrderID := tr.TakerID
		makerOrderID := tr.MakerID
		takerOrderType := orderType // by default assume the current order is taker
		var buyerID, sellerID int64
		var buyerAssetAcct, sellerCashAcct *ledgerRepository.UserLedger

		// Check if our new order was the taker or maker in this trade
		if takerOrderID == orderID {
			// Our new order is the taker
			if side == model.BID {
				// New order is a BUY taker -> buyer is current user, seller is maker order’s user
				buyerID = userID.UserId
				// fetch maker order's user (seller)
				makerOrderRec, _ := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(makerOrderID))
				sellerID = makerOrderRec.UserID
			} else {
				// New order is a SELL taker -> seller is current user, buyer is maker order’s user
				sellerID = userID.UserId
				makerOrderRec, _ := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(makerOrderID))
				buyerID = makerOrderRec.UserID
			}
			takerOrderType = orderType
		} else {
			// Our new order was placed into book and the other order was taker (scenario: our order was GTC and got matched by an incoming IOC from opposite side).
			// In this case, the trade’s taker is the other order.
			// Determine buyer/seller based on the trade Side flag.
			if tr.Side == model.ASK {
				// Trade side ASK => taker was a sell order (the other user is seller, current user is buyer)
				buyerID = userID.UserId
				// taker (seller) is other user:
				takerOrderRec, _ := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(takerOrderID))
				sellerID = takerOrderRec.UserID
				takerOrderType = model.OrderType(takerOrderRec.Type)
			} else {
				// Trade side BID => taker was a buy order (the other user is buyer, current user is seller)
				sellerID = userID.UserId
				takerOrderRec, _ := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(takerOrderID))
				buyerID = takerOrderRec.UserID
				takerOrderType = model.OrderType(takerOrderRec.Type)
			}
		}

		// Fetch buyer’s asset account and seller’s cash account (for settlement)
		assetLedgerID := int64(ou.ledgerID)       //TODO fetch ticker ledger id
		quoteLedgerID := int64(model.CASH_LEDGER) /* quote currency ticker ID, e.g., quoteTicker.ID */
		buyerAssetAcct, err = (*ou.ledgerRepo).GetUserLedger(ctx, tx, buyerID, assetLedgerID)
		if err != nil {
			return nil, orderID, err
		}
		sellerCashAcct, err = (*ou.ledgerRepo).GetUserLedger(ctx, tx, sellerID, quoteLedgerID)
		if err != nil {
			return nil, orderID, err
		}

		// TigerBeetle settlement transfers (escrow -> buyer/seller accounts)
		// Prepare transfer amounts
		cashAmount := big.NewInt(0).Mul(big.NewInt(int64(tr.Price)), big.NewInt(int64(tr.Quantity)))
		assetAmount := big.NewInt(int64(tr.Quantity))
		// Prepare escrow accounts (from ticker records)
		assetTicker, _ := (*ou.ledgerRepo).GetLedgerByID(ctx, tx, assetLedgerID)
		quoteTicker, _ := (*ou.ledgerRepo).GetLedgerByID(ctx, tx, quoteLedgerID)
		assetEscrow, err := stringToUint128(assetTicker.EscrowAccountID)
		if err != nil {
			return nil, orderID, err
		}

		quoteEscrow, err := stringToUint128(quoteTicker.EscrowAccountID)

		if err != nil {
			return nil, orderID, err
		}

		sellerCashTbId, err := stringToUint128(sellerCashAcct.TBAccountID)
		if err != nil {
			return nil, orderID, err
		}
		buyerAssetTbId, err := stringToUint128(buyerAssetAcct.TBAccountID)
		if err != nil {
			return nil, orderID, err
		}

		// Create two transfers:
		transfer1 := Transfer{
			ID:              ID(),           // generate unique transfer ID:contentReference[oaicite:8]{index=8}
			DebitAccountID:  quoteEscrow,    // debit quote currency escrow
			CreditAccountID: sellerCashTbId, // credit seller's fiat account
			Amount:          BigIntToUint128(*cashAmount),
			Ledger:          ou.ledgerID,
			Code:            3001,
			Timestamp:       uint64(time.Now().UnixNano()),
		}
		transfer2 := Transfer{
			ID:              ID(),
			DebitAccountID:  assetEscrow,    // debit asset escrow
			CreditAccountID: buyerAssetTbId, // credit buyer's asset account
			Amount:          BigIntToUint128(*assetAmount),
			Ledger:          ou.ledgerID,
			Code:            3002,
			Timestamp:       uint64(time.Now().UnixNano()),
		}
		// Execute the transfers as a batch
		results, err := (*ou.tbClient).CreateTransfers([]Transfer{transfer1, transfer2})
		if err != nil {
			log.Printf("settlement error: %v", err)
			// Decide how to handle partial failure (compensation or rollback steps if needed)
		} else if len(results) > 0 {
			log.Printf("settlement transfer failures: %+v", results)
		}
		// Use transfer1's ID as the ledger_transfer_id to record in trade (represents fiat movement)
		transferTbId := transfer1.ID.BigInt()

		// Record the trade in the database
		tradeRecord := orderRepository.TradeRecord{
			TickerID:         tickerID, // asset ticker ID
			OrderTakerID:     uint64(tr.TakerID),
			OrderMakerID:     uint64(tr.MakerID),
			LedgerTransferID: &transferTbId,     // store as big.Int (will be saved as NUMERIC string)
			UserLedgerID:     sellerCashAcct.ID, // seller's fiat account record ID
			TickerLedgerID:   buyerAssetAcct.ID, // buyer's asset account record ID
			Type:             uint8(takerOrderType),
			Quantity:         uint64(tr.Quantity),
			Price:            uint64(tr.Price),
		}
		err = (*ou.orderRepo).CreateTrade(ctx, tx, tradeRecord)
		if err != nil {
			return nil, orderID, fmt.Errorf("inserting trade: %w", err)
		}

		// Close fully-filled orders: if the taker or maker order is now completely filled, mark as closed
		// (Check in-memory: engine already removed filled orders. We can also compare filled qty vs initial)
		if tr.MakerID != 0 {
			makerOrder, err := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(tr.MakerID))
			isMakerOrderFilled := (*makerOrder).Quantity == tradeRecord.Quantity
			if err == nil && makerOrder != nil && makerOrder.IsActive && isMakerOrderFilled {
				_ = (*ou.orderRepo).CloseOrder(ctx, tx, makerOrder.ID, time.Now(), false)
			}
			return nil, 0, err
		}
		if tr.TakerID != 0 {
			takerOrder, err := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(tr.TakerID))
			isTakerOrderFilled := (*takerOrder).Quantity == tradeRecord.Quantity
			if err == nil && takerOrder != nil && takerOrder.IsActive && isTakerOrderFilled {
				_ = (*ou.orderRepo).CloseOrder(ctx, tx, takerOrder.ID, time.Now(), false)
			}
			return nil, 0, err
		}
	}

	// 6. Commit transaction and notify via WebSocket
	if err := tx.Commit(); err != nil {
		return nil, orderID, err
	}
	// Trigger WebSocket notifications for each trade (already registered in main)
	for _, tr := range matchedTrades {
		if ou.tradeHandler != nil {
			ou.tradeHandler(*tr)
		}
	}
	return matchedTrades, orderID, nil
}

func (ou *orderUseCaseImpl) CancelOrder(ctx context.Context, orderID model.OrderId) error {

	// Start a transaction to update DB and possibly release funds
	tx := ou.db.MustBeginTx(ctx, nil)
	defer tx.Rollback()
	// Get order details (to know side, remaining quantity, user, etc.)
	ord, err := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(orderID))
	if err != nil {
		return err
	}
	// If order is not already closed and has remaining quantity, release any escrow reservation
	if ord.IsActive && ord.Quantity > 0 {
		// Identify asset vs currency based on side
		tickerRec, _ := (*ou.ledgerRepo).GetLedgerByID(ctx, tx, ord.TickerID)
		// Assume one quote currency as before
		quoteRec, _ := (*ou.ledgerRepo).GetLedgerByTicker(ctx, tx, "USD")
		userID := ord.UserID
		if model.Side(ord.Side) == model.BID {
			// Release reserved currency: transfer from escrow back to user's cash account
			userCashAcct, _ := (*ou.ledgerRepo).GetUserLedger(ctx, tx, userID, quoteRec.ID)
			userCashAcctLedgerId, err := stringToUint128(userCashAcct.TBAccountID)
			if err != nil {
				return err
			}
			fiatEscrow, err := stringToUint128(quoteRec.EscrowAccountID)
			if err != nil {
				return err
			}
			_ = ou.releaseReservationBestEffort(
				ctx,
				model.BID,
				userCashAcctLedgerId,
				BigIntToUint128(*big.NewInt(0)),
				model.Quantity(ord.Quantity),
				model.Price(ord.Price),
				uint32(quoteRec.TBLedgerID),
				fiatEscrow,
			) // (debit from escrow to user)
		} else {
			// Release reserved asset: transfer from escrow back to user's asset account
			userAssetAcct, err := (*ou.ledgerRepo).GetUserLedger(ctx, tx, userID, tickerRec.ID)
			if err != nil {
				return err
			}
			userAssetAccountId, err := stringToUint128(userAssetAcct.TBAccountID)
			if err != nil {
				return err
			}
			assetEscrowAccount, err := stringToUint128(tickerRec.EscrowAccountID)
			if err != nil {
				return err
			}
			_ = ou.releaseReservationBestEffort(
				ctx,
				model.ASK,
				BigIntToUint128(*big.NewInt(0)),
				userAssetAccountId,
				model.Quantity(ord.Quantity),
				model.Price(ord.Price),
				uint32(quoteRec.TBLedgerID),
				assetEscrowAccount,
			)
		}
	}
	// Cancel in the matching engine
	err = ou.orderBookEngine.CancelOrder(orderID)
	if err != nil {
		return err
	}
	// Mark order as canceled in DB
	_ = (*ou.orderRepo).CloseOrder(ctx, tx, uint64(orderID), time.Now(), true)
	return ou.orderBookEngine.CancelOrder(orderID)

}

func (ou *orderUseCaseImpl) ModifyOrder(ctx context.Context, modify model.OrderModify, orderType model.OrderType) ([]*model.Trade, error) {
	tx := ou.db.MustBeginTx(ctx, nil)
	defer tx.Rollback()
	// Fetch the current order record
	ordRec, err := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(modify.ID))
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// 2. Update order record with new parameters
	ordRec.Price = uint64(modify.Price)
	ordRec.Quantity = uint64(modify.Quantity)
	ordRec.Side = int8(modify.Side)
	ordRec.Type = uint8(orderType)
	// (Reset is_active in case it was closed due to partial fill; ensure it's open for the new order)
	ordRec.IsActive = true
	err = (*ou.orderRepo).UpdateOrder(ctx, tx, *ordRec)
	if err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}
	// 3. Submit the updated order to engine
	trades, err := ou.orderBookEngine.ModifyOrder(modify, orderType)
	if err != nil {
		return nil, err
	}
	// 4. Handle any trades resulting from the modify (similar to AddOrder logic above)
	for _, tr := range trades {
		if ou.tradeHandler != nil {
			ou.tradeHandler(*tr)
		}

	}
	tx.Commit()
	return trades, nil
}

func (ou *orderUseCaseImpl) OrderSize(ctx context.Context) int {

	return ou.orderBookEngine.OrderSize()

}

func (ou *orderUseCaseImpl) GetTopOfBook(ctx context.Context) *model.TopOfBook {

	return ou.orderBookEngine.GetTopOfBook()

}

func (ou *orderUseCaseImpl) GetOrderInfos(ctx context.Context) *model.MarketDepth {

	return ou.orderBookEngine.GetOrderInfos()

}

func (ou *orderUseCaseImpl) GetOrderByUserId(ctx context.Context, userId int64) (*[]orderRepository.OrderRecord, error) {
	tx := ou.db.MustBeginTx(ctx, nil)
	return (*ou.orderRepo).GetOrderByUserId(ctx, tx, userId)
}

// TigerBeetle helpers:

func (ou *orderUseCaseImpl) reserveFunds(ctx context.Context, debit Uint128, credit Uint128, amount Uint128, code uint16, ledger uint32) error {

	transfers := []Transfer{

		{

			ID: ID(), // unique

			DebitAccountID: debit,

			CreditAccountID: credit,

			Amount: amount,

			Ledger: ledger,

			Code: code,

			Flags: 0,
		},
	}

	//cannot pass context
	if results, err := (*ou.tbClient).CreateTransfers(transfers); err != nil {

		return err

	} else if len(results) > 0 {

		// TigerBeetle returns an array of errors for failed transfers

		return fmt.Errorf("reserve transfer failed: %+v", results)

	}

	return nil

}

// when user cancel order return back their assets or cash to their account
func (ou *orderUseCaseImpl) releaseReservationBestEffort(
	ctx context.Context,
	side model.Side,
	userCashAccount,
	userAssetAccount Uint128,
	qty model.Quantity,
	price model.Price,
	ledger uint32,
	escrow Uint128,
) error {

	var debit, credit Uint128

	var amount Uint128

	switch side {

	case model.BID:

		debit = escrow

		credit = userCashAccount

		amount = toTigerBeetleUnitsCash(price, qty)

	case model.ASK:

		debit = escrow

		credit = userAssetAccount

		amount = toTigerBeetleUnitsAsset(qty)

	}

	transfers := []Transfer{{

		ID: ID(),

		DebitAccountID: debit,

		CreditAccountID: credit,

		Amount: amount,

		Ledger: ledger,

		Code: 2001,

		Timestamp: uint64(time.Now().UnixNano()),
	}}

	results, err := (*ou.tbClient).CreateTransfers(transfers)

	if err != nil {

		return err

	}

	if len(results) > 0 {

		return fmt.Errorf("release transfer failed: %+v", results)

	}

	return nil

}

// settleTrades posts settlement transfers. In a real system you’d move from escrow to the counterparties.

func (ou *orderUseCaseImpl) settleTrades(ctx context.Context, trades []*model.Trade) error {

	// transfers := make([]Transfer, 0, len(trades)*2)

	for i, _ := range trades {
		i += 1
		// // Map trade maker/taker to their TB accounts
		// makerCash := accountForOrderIDCash(t.MakerID)
		// makerAsset := accountForOrderIDAsset(t.MakerID)
		// takerCash := accountForOrderIDCash(t.TakerID)
		// takerAsset := accountForOrderIDAsset(t.TakerID)

		// cashAmount := toTigerBeetleUnitsCash(t.Price, t.Quantity)
		// assetAmount := toTigerBeetleUnitsAsset(t.Quantity)

		// // Example: escrow -> seller cash, escrow -> buyer asset
		// transfers = append(transfers,
		// 	Transfer{
		// 		ID:              tb.ID()[0],
		// 		DebitAccountID:  ou.escrowAccount,
		// 		CreditAccountID: sellerCashAccount(makerCash, takerCash, t), // whichever side is seller
		// 		Amount:          cashAmount,
		// 		Ledger:          ou.ledgerID,
		// 		Code:            3001,
		// 		Timestamp:       uint64(time.Now().UnixNano()),
		// 	},
		// 	Transfer{
		// 		ID:              tb.ID()[0],
		// 		DebitAccountID:  ou.escrowAccount,
		// 		CreditAccountID: buyerAssetAccount(makerAsset, takerAsset, t),
		// 		Amount:          assetAmount,
		// 		Ledger:          ou.ledgerID,
		// 		Code:            3002,
		// 		Timestamp:       uint64(time.Now().UnixNano()),
		// 	},
		// )
	}

	// results, err := ou.tbClient.CreateTransfers(ctx, transfers)
	// if err != nil {
	// 	return err
	// }
	// if len(results) > 0 {
	// 	return fmt.Errorf("settlement failures: %+v", results)
	// }
	return nil

}
