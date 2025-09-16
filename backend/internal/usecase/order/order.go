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

	ModifyOrder(ctx context.Context, modify model.OrderModify, orderType model.OrderType, ticker string) ([]*model.Trade, error)

	OrderSize(ctx context.Context, ticker string) int

	GetTopOfBook(ctx context.Context, ticker string) *model.TopOfBook

	GetOrderInfos(ctx context.Context, ticker string) *model.MarketDepth

	RegisterTradeHandler(handler TradeHandler)
	GetOrderByUserId(ctx context.Context, userId int64, isOnlyActive bool) (*[]orderRepository.OrderRecord, error)
}
type tickerType string
type orderUseCaseImpl struct {
	orderBookEngineMap map[tickerType]*engine.OrderBookEngine // hold interface by value, not pointer to interface

	tbClient *tb.Client

	ledgerID uint32 // TigerBeetle ledger identifier you choose (e.g., 1)

	escrowAccount Uint128

	tradeHandler TradeHandler
	orderRepo    *orderRepository.OrderRepository
	ledgerRepo   *ledgerRepository.LedgerRepository
	db           *sqlx.DB
}

// NewOrderUseCase constructs with an already-initialized TB client.
type TradeHandler func(model.Trade)

type OrderUseCaseOpts struct {
	TBLedgerID    uint32
	EscrowAccount Uint128 // (optional global escrow, but we'll use per-ticker escrow accounts)
	OrderRepo     *orderRepository.OrderRepository
	LedgerRepo    *ledgerRepository.LedgerRepository
	Db            *sqlx.DB
	TbClient      *tb.Client
}

func NewOrderUseCase(ctx context.Context, opts OrderUseCaseOpts) OrderUseCase {
	orderbookMap := make(map[tickerType]*engine.OrderBookEngine, 3)
	return &orderUseCaseImpl{
		orderBookEngineMap: orderbookMap,
		tbClient:           opts.TbClient,
		ledgerID:           opts.TBLedgerID,
		escrowAccount:      opts.EscrowAccount,
		orderRepo:          opts.OrderRepo,
		ledgerRepo:         opts.LedgerRepo,
		db:                 opts.Db,
	}
}

func (ou *orderUseCaseImpl) RegisterTradeHandler(handler TradeHandler) {
	ou.tradeHandler = handler
}

func (ou *orderUseCaseImpl) getOrderbook(ticker tickerType) *engine.OrderBookEngine {
	orderbook, ok := ou.orderBookEngineMap[ticker]
	if ok {
		return orderbook
	}
	createOrderbook := engine.NewOrderBookEngine()
	createOrderbook.Initialize()
	ou.orderBookEngineMap[ticker] = &createOrderbook

	return &createOrderbook
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
		TickerLedgerID: int64(assetTicker.ID),
		Filled:         0,
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
	matchedTrades, matchErr := (*ou.getOrderbook(tickerType(ticker))).AddOrder(engineOrder)
	if matchErr != nil {
		return nil, orderID, matchErr
	}

	if err := tx.Commit(); err != nil {
		return nil, orderID, err
	}

	err = ou.settleTrades(ctx, matchedTrades, tickerType(ticker))
	if err != nil {
		return nil, orderID, err
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
		quoteRec, _ := (*ou.ledgerRepo).GetLedgerByTicker(ctx, tx, model.CASH_TICKER)
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
			err = ou.releaseReservationBestEffort(
				ctx,
				model.BID,
				userCashAcctLedgerId,
				BigIntToUint128(*big.NewInt(0)),
				model.Quantity(ord.Quantity),
				model.Price(ord.Price),
				uint32(quoteRec.TBLedgerID),
				fiatEscrow,
			)
			if err != nil {
				return err
			}
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
	err = (*ou.orderRepo).CloseOrder(ctx, tx, uint64(orderID), time.Now())
	if err != nil {
		return err
	}
	ledger, err := (*ou.ledgerRepo).GetLedgerByID(ctx, tx, ord.TickerID)
	if err != nil {
		return err
	}
	err = (*ou.getOrderbook(tickerType(ledger.Ticker))).CancelOrder(orderID)
	if err != nil {
		return err
	}

	tx.Commit()
	return err

}

func (ou *orderUseCaseImpl) ModifyOrder(ctx context.Context, modify model.OrderModify, orderType model.OrderType, ticker string) ([]*model.Trade, error) {
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

	err = ou.CancelOrder(ctx, modify.ID)
	if err != nil {
		return nil, err
	}

	_, _, err = ou.AddOrder(ctx, ticker, modify.Side, modify.Price, modify.Quantity, orderType)
	if err != nil {
		return nil, err
	}
	// 3. Submit the updated order to engine
	trades, err := (*ou.getOrderbook(tickerType(ticker))).ModifyOrder(modify, orderType)
	if err != nil {
		return nil, err
	}
	tx.Commit()

	err = ou.settleTrades(ctx, trades, tickerType(ticker))
	if err != nil {
		return nil, err
	}

	return make([]*model.Trade, 0), nil
}

func (ou *orderUseCaseImpl) OrderSize(ctx context.Context, ticker string) int {

	return (*ou.getOrderbook(tickerType(ticker))).OrderSize()

}

func (ou *orderUseCaseImpl) GetTopOfBook(ctx context.Context, ticker string) *model.TopOfBook {

	return (*ou.getOrderbook(tickerType(ticker))).GetTopOfBook()

}

func (ou *orderUseCaseImpl) GetOrderInfos(ctx context.Context, ticker string) *model.MarketDepth {

	return (*ou.getOrderbook(tickerType(ticker))).GetOrderInfos()

}

func (ou *orderUseCaseImpl) GetOrderByUserId(ctx context.Context, userId int64, isOnlyActive bool) (*[]orderRepository.OrderRecord, error) {
	tx := ou.db.MustBeginTx(ctx, nil)
	orderRecord, err := (*ou.orderRepo).ListOrdersByUser(ctx, tx, userId, isOnlyActive)
	return &orderRecord, err
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

// settleTrades posts settlement transfers

func (ou *orderUseCaseImpl) settleTrades(ctx context.Context, matchedTrades []*model.Trade, ticker tickerType) error {
	tx := ou.db.MustBeginTx(ctx, nil)
	defer tx.Rollback()
	assetTicker, err := (*ou.ledgerRepo).GetLedgerByTicker(ctx, tx, string(ticker))
	if err != nil {
		log.Printf("settlement error: did not found ledger ticker %v", err)
		return err
	}
	quoteTicker, err := (*ou.ledgerRepo).GetLedgerByTicker(ctx, tx, model.CASH_TICKER)
	if err != nil {
		log.Printf("settlement error: did not found cash ticker %v", err)
		return err
	}

	tbTransfer := make([]Transfer, 0, 2*len(matchedTrades))
	createTrades := make([]orderRepository.TradeRecord, 0, 2*len(matchedTrades))
	closeOrders := make([]uint64, 0, 2*len(matchedTrades))
	for _, tr := range matchedTrades {
		takerOrderID := tr.TakerID
		makerOrderID := tr.MakerID
		var buyerAssetAcct, sellerCashAcct *ledgerRepository.UserLedger

		makerOrderRec, err := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(makerOrderID))
		if err != nil {
			log.Printf("settlement error: get order maker %v", err)

			return err
		}
		makerUserId := makerOrderRec.UserID
		takerOrderRec, err := (*ou.orderRepo).GetOrderByID(ctx, tx, uint64(takerOrderID))
		if err != nil {
			log.Printf("settlement error: get order taker %v", err)

			return err
		}
		takerUserID := takerOrderRec.UserID

		buyerAssetAcct, err = (*ou.ledgerRepo).GetUserLedger(ctx, tx, takerUserID, assetTicker.ID)
		if err != nil {
			log.Printf("settlement error: get user ledger asset %v", err)

			return err
		}
		sellerCashAcct, err = (*ou.ledgerRepo).GetUserLedger(ctx, tx, makerUserId, quoteTicker.ID)
		if err != nil {
			log.Printf("settlement error: get user ledger cash %v", err)

			return err
		}

		// TigerBeetle settlement transfers (escrow -> buyer/seller accounts)
		// Prepare transfer amounts
		cashAmount := big.NewInt(0).Mul(big.NewInt(int64(tr.Price)), big.NewInt(int64(tr.Quantity)))
		assetAmount := big.NewInt(int64(tr.Quantity))
		// Prepare escrow accounts (from ticker records)

		assetEscrow, err := stringToUint128(assetTicker.EscrowAccountID)
		if err != nil {
			return err
		}

		cashEscrow, err := stringToUint128(quoteTicker.EscrowAccountID)

		if err != nil {
			return err
		}

		sellerCashTbId, err := stringToUint128(sellerCashAcct.TBAccountID)
		if err != nil {
			return err
		}
		buyerAssetTbId, err := stringToUint128(buyerAssetAcct.TBAccountID)
		if err != nil {
			return err
		}

		// Create two transfers:
		transfer1 := Transfer{
			ID:              ID(),           // generate unique transfer ID:contentReference[oaicite:8]{index=8}
			DebitAccountID:  cashEscrow,     // debit quote currency escrow
			CreditAccountID: sellerCashTbId, // credit seller's fiat account
			Amount:          BigIntToUint128(*cashAmount),
			Ledger:          model.CASH_LEDGER,
			Code:            3001,
		}
		tbTransfer = append(tbTransfer, transfer1)
		transfer2 := Transfer{
			ID:              ID(),
			DebitAccountID:  assetEscrow,    // debit asset escrow
			CreditAccountID: buyerAssetTbId, // credit buyer's asset account
			Amount:          BigIntToUint128(*assetAmount),
			Ledger:          uint32(assetTicker.TBLedgerID),
			Code:            3002,
		}
		tbTransfer = append(tbTransfer, transfer2)
		// Use transfer1's ID as the ledger_transfer_id to record in trade (represents fiat movement)
		transferTbId := transfer1.ID.BigInt()

		// Record the trade in the database
		tradeRecord := orderRepository.TradeRecord{
			TickerID:         assetTicker.ID, // asset ticker ID
			OrderTakerID:     uint64(tr.TakerID),
			OrderMakerID:     uint64(tr.MakerID),
			LedgerTransferID: &transferTbId,
			UserLedgerID:     sellerCashAcct.ID,
			TickerLedgerID:   buyerAssetAcct.ID,
			Quantity:         uint64(tr.Quantity),
			Price:            uint64(tr.Price),
		}
		createTrades = append(createTrades, tradeRecord)

		// Close fully-filled orders: if the taker or maker order is now completely filled, mark as closed
		// (Check in-memory: engine already removed filled orders. We can also compare filled qty vs initial)

		isMakerOrderFilled := makerOrderRec.GetRemaining() == tradeRecord.Quantity
		if !isMakerOrderFilled {
			err = (*ou.orderRepo).UpdateFilled(ctx, tx, makerOrderRec.ID, tradeRecord.Quantity)
			if err != nil {
				return err
			}
		} else {
			closeOrders = append(closeOrders, makerOrderRec.ID)
		}
		// return err
		isTakerOrderFilled := takerOrderRec.GetRemaining() == tradeRecord.Quantity
		if !isTakerOrderFilled {
			err = (*ou.orderRepo).UpdateFilled(ctx, tx, takerOrderRec.ID, tradeRecord.Quantity)
			if err != nil {
				return err
			}
		} else {
			closeOrders = append(closeOrders, takerOrderRec.ID)
		}
	}

	results, err := (*ou.tbClient).CreateTransfers(tbTransfer)
	if err != nil {
		log.Printf("settlement error: %v", err)
		return err
	} else if len(results) > 0 {
		log.Printf("settlement transfer failures: %+v", results)
		return fmt.Errorf("settlement transfer failures: %+v", results)
	}

	closeTradeErrs := (*ou.orderRepo).CloseOrders(ctx, tx, closeOrders, time.Now())
	if closeTradeErrs != nil {
		return fmt.Errorf("inserting trade: %w", closeTradeErrs)
	}
	err = (*ou.orderRepo).CreateTrades(ctx, tx, createTrades)
	if err != nil {
		return fmt.Errorf("inserting trade: %w", err)
	}

	tx.Commit()
	for _, tr := range matchedTrades {
		if ou.tradeHandler != nil {
			ou.tradeHandler(*tr)
		}
	}
	return nil

}
