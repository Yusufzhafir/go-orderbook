package ledger

import (
	"context"
	"math/big"
	"time"

	"github.com/jmoiron/sqlx"
)

// --- Models ---
type Ticker struct {
	ID              int64     `db:"id"`
	Ticker          string    `db:"ticker"`
	TBLedgerID      int64     `db:"tb_ledger_id"`      // TB ledger is uint32, safe as int64 in Go/PG BIGINT
	EscrowAccountID string    `db:"escrow_account_id"` // TB account (uint128) stored as NUMERIC(38,0)
	CreatedAt       time.Time `db:"created_at"`
}

type UserLedger struct {
	ID          int64     `db:"id"`
	UserID      int64     `db:"user_id"`
	LedgerID    int64     `db:"ledger_id"`     // references ticker.id
	TBAccountID string    `db:"tb_account_id"` // TB account (uint128) stored as NUMERIC(38,0)
	IsEscrow    bool      `db:"is_escrow"`
	CreatedAt   time.Time `db:"created_at"`
}

// --- Interface ---
type LedgerRepository interface {
	// Ledger (Ticker) operations
	CreateLedger(ctx context.Context, tx *sqlx.Tx, ticker string, tbLedgerID int64, escrowAccountId string) (CreateLedgerResult, error)
	GetLedgerByID(ctx context.Context, tx *sqlx.Tx, id int64) (*Ticker, error)
	GetLedgerByTicker(ctx context.Context, tx *sqlx.Tx, ticker string) (*Ticker, error)
	ListLedgers(ctx context.Context, tx *sqlx.Tx) ([]Ticker, error)
	UpdateEscrowAccount(ctx context.Context, tx *sqlx.Tx, ledgerID int64, escrowAccountID int64) error

	// UserLedger operations
	CreateUserLedger(ctx context.Context, tx *sqlx.Tx, userID int64, ledgerID int64, tbAccountID *big.Int, isEscrow bool) (int64, error)
	GetUserLedger(ctx context.Context, tx *sqlx.Tx, userID int64, ledgerID int64) (*UserLedger, error)
	ListUserLedgers(ctx context.Context, tx *sqlx.Tx, userID int64) ([]UserLedger, error)
}

// --- Implementation ---
type ledgerRepositoryImpl struct {
}

func NewLedgerRepository(db *sqlx.DB) LedgerRepository {
	return &ledgerRepositoryImpl{}
}

// --- Ledger (Ticker) methods ---
type CreateLedgerResult struct {
	LedgerId   int64
	LedgerTbId int64
}

// CreateLedger inserts a new asset/currency ledger (ticker) with a TigerBeetle ledger ID.
func (r *ledgerRepositoryImpl) CreateLedger(ctx context.Context, tx *sqlx.Tx, ticker string, tbLedgerID int64, escrowAccountId string) (CreateLedgerResult, error) {

	// Note: tbLedgerID is a 128-bit numeric. TigerBeetle ledger IDs are 32-bit,
	// so ensure tbLedgerID fits in uint32 if TigerBeetle requires that.
	var id int64
	err := tx.QueryRowContext(ctx,
		`INSERT INTO ticker (ticker, tb_ledger_id,escrow_account_id) VALUES ($1, $2, $3) RETURNING id`,
		ticker, tbLedgerID, escrowAccountId,
	).Scan(&id)
	return CreateLedgerResult{
		LedgerId:   id,
		LedgerTbId: tbLedgerID,
	}, err
}

func (r *ledgerRepositoryImpl) GetLedgerByID(ctx context.Context, tx *sqlx.Tx, id int64) (*Ticker, error) {
	var t Ticker
	err := tx.GetContext(ctx, &t,
		`SELECT id, ticker, tb_ledger_id, escrow_account_id, created_at FROM ticker WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ledgerRepositoryImpl) GetLedgerByTicker(ctx context.Context, tx *sqlx.Tx, ticker string) (*Ticker, error) {
	var t Ticker
	err := tx.GetContext(ctx, &t,
		`SELECT id, ticker, tb_ledger_id, escrow_account_id, created_at FROM ticker WHERE ticker=$1`, ticker)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ledgerRepositoryImpl) ListLedgers(ctx context.Context, tx *sqlx.Tx) ([]Ticker, error) {
	var list []Ticker
	err := tx.SelectContext(ctx, &list,
		`SELECT id, ticker, tb_ledger_id, escrow_account_id, created_at FROM ticker ORDER BY id`)
	return list, err
}

func (r *ledgerRepositoryImpl) UpdateEscrowAccount(ctx context.Context, tx *sqlx.Tx, ledgerID int64, escrowAccountID int64) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE ticker SET escrow_account_id=$1 WHERE id=$2`, escrowAccountID, ledgerID)
	return err
}

// --- UserLedger (user account in a ledger) methods ---

func (r *ledgerRepositoryImpl) CreateUserLedger(ctx context.Context, tx *sqlx.Tx, userID int64, ledgerID int64, tbAccountID *big.Int, isEscrow bool) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		`INSERT INTO users_ledger (user_id, ledger_id, tb_account_id, is_escrow)
         VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, ledgerID, tbAccountID.String(), isEscrow,
	).Scan(&id)
	return id, err
}

func (r *ledgerRepositoryImpl) GetUserLedger(ctx context.Context, tx *sqlx.Tx, userID int64, ledgerID int64) (*UserLedger, error) {
	var ul UserLedger
	err := tx.GetContext(ctx, &ul,
		`SELECT id, user_id, ledger_id, tb_account_id, is_escrow, created_at
         FROM users_ledger 
         WHERE user_id=$1 AND ledger_id=$2`,
		userID, ledgerID)
	if err != nil {
		return nil, err
	}
	return &ul, nil
}

func (r *ledgerRepositoryImpl) ListUserLedgers(ctx context.Context, tx *sqlx.Tx, userID int64) ([]UserLedger, error) {
	var list []UserLedger
	err := tx.SelectContext(ctx, &list,
		`SELECT id, user_id, ledger_id, tb_account_id, is_escrow, created_at
         FROM users_ledger 
         WHERE user_id=$1 
         ORDER BY ledger_id`,
		userID)
	return list, err
}
