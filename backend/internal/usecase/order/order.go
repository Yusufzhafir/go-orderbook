package order

import (
	"context"

	"fmt"

	"log"

	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/engine"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	. "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

type OrderUseCase interface {
	AddOrder(ctx context.Context, side model.Side, price model.Price, quantity model.Quantity, orderType model.OrderType) (trades []*model.Trade, orderID model.OrderId, err error)

	CancelOrder(ctx context.Context, orderID model.OrderId) error

	ModifyOrder(ctx context.Context, modify model.OrderModify, orderType model.OrderType) (trades []*model.Trade, err error)

	OrderSize(ctx context.Context) int

	GetTopOfBook(ctx context.Context) *model.TopOfBook

	GetOrderInfos(ctx context.Context) *model.MarketDepth

	RegisterTradeHandler(handler TradeHandler)
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
}

// NewOrderUseCase constructs with an already-initialized TB client.
type TradeHandler func(model.Trade)

type OrderUseCaseOpts struct {
	OrderBookEngine engine.OrderBookEngine

	TBClusterAddrs []string // e.g., []string{"3000"}

	TBLedgerID uint32 // e.g., 1

	EscrowAccount Uint128
}

func NewOrderUseCase(ctx context.Context, opts OrderUseCaseOpts) (OrderUseCase, error) {

	// In TigerBeetle, you create the client once and reuse

	client, err := tb.NewClient(ToUint128(0), opts.TBClusterAddrs)

	if err != nil {

		return nil, fmt.Errorf("tigerbeetle client init: %w", err)

	}

	// optionally: ping to ensure connection
	return &orderUseCaseImpl{
		orderBookEngine: opts.OrderBookEngine,
		tbClient:        &client,
		ledgerID:        opts.TBLedgerID,
		escrowAccount:   opts.EscrowAccount,
	}, nil

}

func (ou *orderUseCaseImpl) RegisterTradeHandler(handler TradeHandler) {
	ou.tradeHandler = handler
}

// AddOrder writes any necessary pre-commit ledger entries (e.g., reserve funds), then submits to engine.

func (ou *orderUseCaseImpl) AddOrder(ctx context.Context, side model.Side, price model.Price, quantity model.Quantity, orderType model.OrderType) ([]*model.Trade, model.OrderId, error) {

	// Generate an order ID. If your engine assigns IDs, call into it; otherwise generate here.

	// Using TigerBeetle’s ID() is fine to produce unique 128-bit values; map to your OrderId type.

	orderID := model.OrderId(model.OrderId(time.Now().UnixMilli())) // or map properly to your type

	// Example: reserve funds for a LIMIT BUY by debiting user cash -> credit escrow
	// and for a LIMIT SELL, reserve asset -> credit escrow. Adjust to your accounting model.
	// For demo, assume you can map from context to a user account:
	// userCash := uint64(10) // Uint128
	// userAsset := 10        // Uint128

	// Optional reservation for non-market/IOC orders:
	// if orderType == model.ORDER_GOOD_TILL_CANCEL {
	// 	if side == model.BID {
	// 		amount := toTigerBeetleUnitsCash(price, quantity) // compute cash = price * qty in integer smallest units
	// 		if err := ou.reserveFunds(ctx, userCash, ou.escrowAccount, amount, 1001 /*code*/); err != nil {
	// 			return nil, 0, err
	// 		}
	// 	} else if side == model.ASK {
	// 		amount := toTigerBeetleUnitsAsset(quantity)
	// 		if err := ou.reserveFunds(ctx, userAsset, ou.escrowAccount, amount, 1002); err != nil {
	// 			return nil, 0, err
	// 		}
	// 	}
	// }

	// Submit to engine
	newOrder := model.NewOrder(
		orderID,
		side,
		price,
		quantity,
		orderType,
	)
	trades, err := ou.orderBookEngine.AddOrder(newOrder)
	if err != nil {
		return nil, newOrder.GetId(), err
	}

	// For each trade, settle funds atomically in TB (batch transfers).
	// Example settlement:
	for _, trade := range trades {
		if ou.tradeHandler != nil {
			ou.tradeHandler(*trade)
		}
		if err := ou.settleTrades(ctx, trades); err != nil {
			// Settlement error handling policy is up to you:
			// - If engine already matched, you need compensating actions or design a 2-phase approach.
			log.Printf("settlement error: %v", err)
			// You might alert and reconcile.
		}
	}

	return trades, orderID, nil

}

func (ou *orderUseCaseImpl) CancelOrder(ctx context.Context, orderID model.OrderId) error {

	// If you reserved funds on AddOrder and the order is still open, release here.

	// You’ll need to fetch order details from engine to know side/price/qty remaining and user accounts.

	// Example (pseudo):

	// ord := ou.orderBookEngine.GetOrder(orderID)

	// if ord != nil && ord.RemainingQuantity > 0 { release escrow back to user }

	return ou.orderBookEngine.CancelOrder(orderID)

}

func (ou *orderUseCaseImpl) ModifyOrder(ctx context.Context, modify model.OrderModify, orderType model.OrderType) ([]*model.Trade, error) {

	// Consider adjusting reservations if price/quantity increases.

	trades, err := ou.orderBookEngine.ModifyOrder(modify, orderType)
	if err != nil {
		return nil, err
	}
	for _, trade := range trades {
		if ou.tradeHandler != nil {
			ou.tradeHandler(*trade)
		}
	}
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

// TigerBeetle helpers:

// reserveFunds creates a transfer user -> escrow (for cash or asset).
func (ou *orderUseCaseImpl) reserveFunds(ctx context.Context, debit Uint128, credit Uint128, amount Uint128, code uint16) error {

	transfers := []Transfer{

		{

			ID: ID(), // unique

			DebitAccountID: debit,

			CreditAccountID: credit,

			Amount: amount,

			Ledger: ou.ledgerID,

			Code: code,

			Flags: 0,

			Timestamp: uint64(time.Now().UnixNano()),
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

// releaseReservationBestEffort is an example; tailor as needed.

func (ou *orderUseCaseImpl) releaseReservationBestEffort(ctx context.Context, side model.Side, userCash, userAsset Uint128, qty model.Quantity, price model.Price) error {

	var debit, credit Uint128

	var amount Uint128

	switch side {

	case model.BID:

		debit = ou.escrowAccount

		credit = userCash

		amount = toTigerBeetleUnitsCash(price, qty)

	case model.ASK:

		debit = ou.escrowAccount

		credit = userAsset

		amount = toTigerBeetleUnitsAsset(qty)

	}

	transfers := []Transfer{{

		ID: ID(),

		DebitAccountID: debit,

		CreditAccountID: credit,

		Amount: amount,

		Ledger: ou.ledgerID,

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
