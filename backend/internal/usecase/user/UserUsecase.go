package user

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/repository/ledger"
	repository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/user"
	"github.com/jmoiron/sqlx"
	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbTypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

type UserUseCase interface {
	Register(ctx context.Context, username, password string) (int64, error)
	Login(ctx context.Context, username, password string) (*repository.User, error)
	GetProfile(ctx context.Context, userID int64) (*repository.User, error)
	GetUserLedger(ctx context.Context, userID int64, ledgerID int64) (*ledger.UserLedger, error)
}

type userUseCaseImpl struct {
	repo       *repository.UserRepository
	ledgerRepo *ledger.LedgerRepository
	tbClient   *tb.Client
	db         *sqlx.DB
}

type UserUseCaseOpts struct {
	UserRepo   *repository.UserRepository
	LedgerRepo *ledger.LedgerRepository
	TbClient   *tb.Client
	Db         *sqlx.DB
}

func NewUserUseCase(opts UserUseCaseOpts) UserUseCase {
	return &userUseCaseImpl{
		repo:       opts.UserRepo,
		ledgerRepo: opts.LedgerRepo,
		tbClient:   opts.TbClient,
		db:         opts.Db,
	}
}
func (uc *userUseCaseImpl) GetUserLedger(ctx context.Context, userID int64, ledgerID int64) (*ledger.UserLedger, error) {
	tx := uc.db.MustBeginTx(ctx, nil)
	return (*uc.ledgerRepo).GetUserLedger(ctx, tx, userID, ledgerID)
}

func (uc *userUseCaseImpl) Register(ctx context.Context, username, password string) (int64, error) {
	// Prevent duplicate usernames
	existing, _ := (*uc.repo).GetByUsername(ctx, username)
	if existing != nil {
		return 0, errors.New("username already exists")
	}

	tx := uc.db.MustBeginTx(ctx, nil)
	// Create the new user in the users table
	newUserID, err := (*uc.repo).Create(ctx, tx, username, password)
	defer tx.Rollback()
	if err != nil {
		return 0, err
	}

	// Fetch all existing ledgers (tickers) from DB
	ledgers, err := (*uc.ledgerRepo).ListLedgers(ctx, tx)
	if err != nil {
		return 0, err
	}

	// Prepare TigerBeetle accounts batch (for efficiency, create all at once)
	tbAccounts := make([]tbTypes.Account, 0, len(ledgers))

	for i, ledger := range ledgers {
		log.Printf("this is ledgers %v", ledger)
		// Generate a new TigerBeetle account ID (128-bit) for this user & ledger
		accountID := tbTypes.ID()           // returns a new tbTypes.Uint128 unique ID
		accountBigInt := accountID.BigInt() // convert to *big.Int for DB storage

		// Determine TigerBeetle ledger numeric ID (must fit in uint32)
		ledgerNum := uint32(ledger.TBLedgerID)

		// Configure account flags: ensure user cannot go negative (debits <= credits)
		isLinked := true
		if i == len(ledgers)-1 {
			isLinked = false
		}
		flags := tbTypes.AccountFlags{DebitsMustNotExceedCredits: true, History: true, Linked: isLinked}.ToUint16()
		// You might also set other fields like Code (e.g., currency code) if needed
		tbAccounts = append(tbAccounts, tbTypes.Account{
			ID:     accountID, // 128-bit account identifier
			Ledger: ledgerNum, // TigerBeetle ledger (asset/currency identifier)
			Code:   1,         // optional account code, 0 if not used
			Flags:  flags,     // enforce no overdraft on this account:contentReference[oaicite:3]{index=3}
		})

		// Record the UserLedger mapping in our database
		_, err = (*uc.ledgerRepo).CreateUserLedger(ctx, tx, newUserID, ledger.ID, &accountBigInt, false)
		if err != nil {
			// If DB insert fails, handle cleanup if necessary (e.g., remove any created accounts)
			return 0, err
		}
	}

	tbclient := *uc.tbClient
	// Create all the new accounts in TigerBeetle in one batch operation
	accountError, err := tbclient.CreateAccounts(tbAccounts)
	if err != nil {
		// If TigerBeetle account creation fails, rollback the DB inserts to avoid dangling records
		// (This might involve deleting the UserLedger records created above)
		return 0, err
	}
	for _, err := range accountError {
		switch err.Index {
		case uint32(tbTypes.AccountExists):
			log.Printf("Batch account at %d already exists.", err.Index)
		default:
			log.Printf("Batch account at %d failed to create: %s", err.Index, err.Result)
		}
	}
	if len(accountError) != 0 {
		return 0, fmt.Errorf("error creating accounts in tigerbeetle got %d errors", len(accountError))
	}

	tx.Commit()
	return newUserID, nil
}

func (uc *userUseCaseImpl) Login(ctx context.Context, username, password string) (*repository.User, error) {
	return (*uc.repo).VerifyPassword(ctx, username, password)
}

func (uc *userUseCaseImpl) GetProfile(ctx context.Context, userID int64) (*repository.User, error) {
	return (*uc.repo).GetByID(ctx, userID)
}
