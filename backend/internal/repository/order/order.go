package order

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// --- Models corresponding to DB tables ---
type OrderRecord struct {
	ID             uint64  `db:"id"`
	UserID         int64   `db:"user_id"`
	TickerID       int64   `db:"ticker_id"`
	Side           int8    `db:"side"`
	TickerLedgerID int64   `db:"ticker_ledger_id"`
	Type           uint8   `db:"type"`
	Quantity       uint64  `db:"quantity"`
	Filled         uint64  `db:"filled"`
	Price          uint64  `db:"price"`
	IsActive       bool    `db:"is_active"`
	CreatedAt      string  `db:"created_at"`
	ClosedAt       *string `db:"closed_at"`
}

func (rec *OrderRecord) GetRemaining() uint64 {
	return rec.Quantity - rec.Filled
}

type TradeRecord struct {
	ID               int64    `db:"id"`
	TickerID         int64    `db:"ticker_id"`
	OrderTakerID     uint64   `db:"order_taker_id"`
	OrderMakerID     uint64   `db:"order_maker_id"`
	LedgerTransferID *big.Int `db:"ledger_transfer_id"` // using big.Int to handle NUMERIC(38,0)
	UserLedgerID     int64    `db:"user_ledger_id"`
	TickerLedgerID   int64    `db:"ticker_ledger_id"`
	Type             uint8    `db:"type"`
	Quantity         uint64   `db:"quantity"`
	Price            uint64   `db:"price"`
	TradedAt         string   `db:"traded_at"`
}

// --- Repository Interface ---
type OrderRepository interface {
	CreateOrder(ctx context.Context, tx *sqlx.Tx, order OrderRecord) error
	UpdateOrder(ctx context.Context, tx *sqlx.Tx, order OrderRecord) error
	CloseOrder(ctx context.Context, tx *sqlx.Tx, orderID uint64, closedAt time.Time) error
	CloseOrders(ctx context.Context, tx *sqlx.Tx, orderID []uint64, closedAt time.Time) error
	UpdateFilled(ctx context.Context, tx *sqlx.Tx, orderID uint64, filled uint64) error
	GetOrderByID(ctx context.Context, tx *sqlx.Tx, orderID uint64) (*OrderRecord, error)
	ListOrdersByUser(ctx context.Context, tx *sqlx.Tx, userID int64, onlyActive bool) ([]OrderRecordWithTicker, error)
	CreateTrade(ctx context.Context, tx *sqlx.Tx, trade TradeRecord) error
	CreateTrades(ctx context.Context, tx *sqlx.Tx, trade []TradeRecord) error
}

// --- Implementation ---
type orderRepositoryImpl struct{}

func NewOrderRepository(db *sqlx.DB) OrderRepository {
	return &orderRepositoryImpl{}
}

func (r *orderRepositoryImpl) CreateOrder(ctx context.Context, tx *sqlx.Tx, order OrderRecord) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO orders (id, user_id, ticker_id, side, ticker_ledger_id, type, quantity,filled, price, is_active)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		order.ID, order.UserID, order.TickerID, order.Side, order.TickerLedgerID, order.Type, order.Quantity, order.Filled, order.Price, order.IsActive)
	return err
}

func (r *orderRepositoryImpl) UpdateOrder(ctx context.Context, tx *sqlx.Tx, order OrderRecord) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET price=$1, quantity=$2, type=$3, side=$4, ticker_ledger_id=$5,filled=$6
         WHERE id=$7`,
		order.Price, order.Quantity, order.Type, order.Side, order.TickerLedgerID, order.Filled, order.ID)
	return err
}
func (r *orderRepositoryImpl) UpdateFilled(ctx context.Context, tx *sqlx.Tx, orderID uint64, filled uint64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET filled=$1
         WHERE id=$2`,
		filled, orderID)
	return err
}

func (r *orderRepositoryImpl) CloseOrder(ctx context.Context, tx *sqlx.Tx, orderID uint64, closedAt time.Time) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET is_active=false, closed_at=$1 WHERE id=$2`,
		closedAt, orderID)
	return err
}

func (r *orderRepositoryImpl) CloseOrders(ctx context.Context, tx *sqlx.Tx, orderIDs []uint64, closedAt time.Time) error {
	if tx == nil {
		return fmt.Errorf("nil transaction passed to CloseOrders")
	}
	if len(orderIDs) == 0 {
		return nil
	}

	// Build: UPDATE ... WHERE id IN (...)
	q, args, err := sqlx.In(`
		UPDATE orders
		SET is_active = FALSE, closed_at = ?
		WHERE id IN (?)
		  AND is_active = TRUE
	`, closedAt, orderIDs)
	if err != nil {
		return err
	}
	q = tx.Rebind(q) // convert '?' -> '$1,$2,...' for Postgres

	_, err = tx.ExecContext(ctx, q, args...)
	return err
}

func (r *orderRepositoryImpl) GetOrderByID(ctx context.Context, tx *sqlx.Tx, orderID uint64) (*OrderRecord, error) {
	var ord OrderRecord
	err := tx.GetContext(ctx, &ord,
		`SELECT id, user_id, ticker_id, side, ticker_ledger_id, type, quantity,filled, price, is_active, created_at, closed_at
         FROM orders WHERE id=$1 LIMIT 1`,
		orderID)
	if err != nil {
		return nil, err
	}
	return &ord, nil
}

type OrderRecordWithTicker struct {
	ID             uint64  `db:"id"`
	UserID         int64   `db:"user_id"`
	Ticker         string  `db:"ticker"`
	TickerID       int64   `db:"ticker_id"`
	Side           int8    `db:"side"`
	TickerLedgerID int64   `db:"ticker_ledger_id"`
	Type           uint8   `db:"type"`
	Quantity       uint64  `db:"quantity"`
	Filled         uint64  `db:"filled"`
	Price          uint64  `db:"price"`
	IsActive       bool    `db:"is_active"`
	CreatedAt      string  `db:"created_at"`
	ClosedAt       *string `db:"closed_at"`
}

func (r *orderRepositoryImpl) ListOrdersByUser(ctx context.Context, tx *sqlx.Tx, userID int64, onlyActive bool) ([]OrderRecordWithTicker, error) {
	var orders []OrderRecordWithTicker
	var err error
	if onlyActive {
		err = tx.SelectContext(ctx, &orders,
			`SELECT o.id, user_id, ticker_id,side, t.ticker as ticker,ticker_ledger_id, type, quantity,filled, price, is_active, o.created_at, closed_at
             FROM orders o LEFT JOIN ticker t ON o.ticker_id=t.id WHERE user_id=$1 AND is_active=true ORDER BY created_at DESC`, userID)
	} else {
		err = tx.SelectContext(ctx, &orders,
			`SELECT id, user_id, ticker_id, side, t.ticker as ticker, ticker_ledger_id, type, quantity,filled, price, is_active, created_at, closed_at
             FROM orders o LEFT JOIN ticker t ON o.ticker_id=t.id  WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	}
	return orders, err
}

func (r *orderRepositoryImpl) CreateTrade(ctx context.Context, tx *sqlx.Tx, trade TradeRecord) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO trades (ticker_id, order_taker_id, order_maker_id, ledger_transfer_id,
                              user_ledger_id, ticker_ledger_id, type, quantity, price, traded_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, NOW())`,
		trade.TickerID, trade.OrderTakerID, trade.OrderMakerID,
		trade.LedgerTransferID.String(),
		trade.UserLedgerID, trade.TickerLedgerID,
		trade.Type, trade.Quantity, trade.Price)
	return err
}

func (r *orderRepositoryImpl) CreateTrades(ctx context.Context, tx *sqlx.Tx, trades []TradeRecord) error {
	if len(trades) == 0 {
		return nil
	}

	var (
		sb    strings.Builder
		args  = make([]interface{}, 0, len(trades)*9) // 9 bind args per row (traded_at uses NOW())
		count = 0
	)

	sb.WriteString(`INSERT INTO trades (
		ticker_id, order_taker_id, order_maker_id, ledger_transfer_id,
		user_ledger_id, ticker_ledger_id, type, quantity, price, traded_at
	) VALUES `)

	for i, t := range trades {
		if i > 0 {
			sb.WriteString(",")
		}
		// Placeholders for this row
		// ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,NOW())",
			count+1, count+2, count+3, count+4, count+5, count+6, count+7, count+8, count+9,
		))
		count += 9

		ltid := "0"
		if t.LedgerTransferID != nil {
			ltid = t.LedgerTransferID.String() // decimal string for NUMERIC(38,0)
		}

		args = append(args,
			t.TickerID,
			t.OrderTakerID,
			t.OrderMakerID,
			ltid,
			t.UserLedgerID,
			t.TickerLedgerID,
			t.Type,
			t.Quantity,
			t.Price,
		)
	}

	_, err := tx.ExecContext(ctx, sb.String(), args...)
	return err
}
