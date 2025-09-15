package order

import (
	"context"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
)

// --- Models corresponding to DB tables ---
type OrderRecord struct {
	ID             uint64  `db:"id"`
	UserID         int64   `db:"user_id"`
	TickerID       int64   `db:"ticker_id"`
	Side           int8    `db:"side"` // 0 = BID, 1 = ASK
	TickerLedgerID int64   `db:"ticker_ledger_id"`
	Type           uint8   `db:"type"` // 0 = IOC (Fill-And-Kill), 1 = GTC
	Quantity       uint64  `db:"quantity"`
	Price          uint64  `db:"price"`
	IsActive       bool    `db:"is_active"`
	CreatedAt      string  `db:"created_at"` // use time.Time in real code
	ClosedAt       *string `db:"closed_at"`
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
	CloseOrder(ctx context.Context, tx *sqlx.Tx, orderID uint64, closedAt time.Time, canceled bool) error
	GetOrderByID(ctx context.Context, tx *sqlx.Tx, orderID uint64) (*OrderRecord, error)
	ListOrdersByUser(ctx context.Context, tx *sqlx.Tx, userID int64, onlyActive bool) ([]OrderRecord, error)
	CreateTrade(ctx context.Context, tx *sqlx.Tx, trade TradeRecord) error
}

// --- Implementation ---
type orderRepositoryImpl struct{}

func NewOrderRepository(db *sqlx.DB) OrderRepository {
	return &orderRepositoryImpl{}
}

func (r *orderRepositoryImpl) CreateOrder(ctx context.Context, tx *sqlx.Tx, order OrderRecord) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO orders (id, user_id, ticker_id, side, ticker_ledger_id, type, quantity, price, is_active)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		order.ID, order.UserID, order.TickerID, order.Side, order.TickerLedgerID, order.Type, order.Quantity, order.Price, order.IsActive)
	return err
}

func (r *orderRepositoryImpl) UpdateOrder(ctx context.Context, tx *sqlx.Tx, order OrderRecord) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET price=$1, quantity=$2, type=$3, side=$4, ticker_ledger_id=$5 
         WHERE id=$6`,
		order.Price, order.Quantity, order.Type, order.Side, order.TickerLedgerID, order.ID)
	return err
}

func (r *orderRepositoryImpl) CloseOrder(ctx context.Context, tx *sqlx.Tx, orderID uint64, closedAt time.Time, canceled bool) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE orders SET is_active=false, closed_at=$1 WHERE id=$2`,
		closedAt, orderID)
	return err
}

func (r *orderRepositoryImpl) GetOrderByID(ctx context.Context, tx *sqlx.Tx, orderID uint64) (*OrderRecord, error) {
	var ord OrderRecord
	err := tx.GetContext(ctx, &ord,
		`SELECT id, user_id, ticker_id, side, ticker_ledger_id, type, quantity, price, is_active, created_at, closed_at
         FROM orders WHERE id=$1`,
		orderID)
	if err != nil {
		return nil, err
	}
	return &ord, nil
}

func (r *orderRepositoryImpl) ListOrdersByUser(ctx context.Context, tx *sqlx.Tx, userID int64, onlyActive bool) ([]OrderRecord, error) {
	var orders []OrderRecord
	var err error
	if onlyActive {
		err = tx.SelectContext(ctx, &orders,
			`SELECT id, user_id, ticker_id, side, ticker_ledger_id, type, quantity, price, is_active, created_at, closed_at
             FROM orders WHERE user_id=$1 AND is_active=true ORDER BY created_at DESC`, userID)
	} else {
		err = tx.SelectContext(ctx, &orders,
			`SELECT id, user_id, ticker_id, side, ticker_ledger_id, type, quantity, price, is_active, created_at, closed_at
             FROM orders WHERE user_id=$1 ORDER BY created_at DESC`, userID)
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
