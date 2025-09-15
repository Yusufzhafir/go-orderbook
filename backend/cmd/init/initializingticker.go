package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	userLedgerRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/ledger"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"

	// userRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/user"
	// "github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/user"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbTypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return

	}

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	// construct DSN
	pgInfo := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)
	db, err := sqlx.Connect("postgres", pgInfo)
	if err != nil {
		log.Fatalf("error connecting postgres: %v", err)
		return
	}

	client, err := tb.NewClient(tbTypes.ToUint128(1), []string{"3001"})
	if err != nil {
		log.Fatalf("error connecting tigerbeetle: %v", err)
		return
	}

	type LedgerInit struct {
		Ticker   string
		LedgerId int
	}
	ledgerList := []LedgerInit{
		{
			Ticker:   model.CASH_TICKER,
			LedgerId: model.CASH_LEDGER,
		},
		{
			Ticker:   "BBCAUSD",
			LedgerId: 20,
		},
		{
			Ticker:   "BTCUSD",
			LedgerId: 30,
		},
	}
	userLedgerRepo := userLedgerRepository.NewLedgerRepository(db)
	rootTx := db.MustBeginTx(rootCtx, nil)
	defer rootTx.Rollback()

	escrowAccounts := make([]tbTypes.Account, 0, 3)
	for i, ledgerItem := range ledgerList {
		escrowAccountId := tbTypes.ID()
		accountId := escrowAccountId.BigInt()
		_, err := userLedgerRepo.CreateLedger(rootCtx, rootTx, ledgerItem.Ticker, int64(ledgerItem.LedgerId), accountId.String())
		if err != nil {
			log.Fatalf("error creating ledger: %v", err)
			return
		}
		isLinked := true
		if i == len(ledgerList)-1 {
			isLinked = false
		}
		escrowAccounts = append(escrowAccounts, tbTypes.Account{
			ID:     escrowAccountId,
			Code:   1001,
			Ledger: uint32(ledgerItem.LedgerId),
			Flags: tbTypes.AccountFlags{
				Linked:                     isLinked,
				CreditsMustNotExceedDebits: true,
				History:                    true,
			}.ToUint16(),
		})
	}

	errList, err := client.CreateAccounts(escrowAccounts)

	if err != nil {
		log.Fatalf("error creating accounts: %v", err)
	}

	if len(errList) > 0 {
		for i, accountError := range errList {
			log.Printf("on index %d, we got this error: %v", i, accountError)
		}
		log.Fatalf("error creating accounts: %v", err)
	}

	ledgers, err := userLedgerRepo.ListLedgers(rootCtx, rootTx)
	if err != nil {
		log.Fatalf("error getting ledger list: %v", err)
		return

	}

	log.Printf("this is existing ledgers in db %v", ledgers)

	queryResult, err := client.QueryAccounts(
		tbTypes.QueryFilter{
			Code:  1001,
			Limit: 1000,
		},
	)
	if err != nil {
		log.Fatalf("error fetching accounts: %v", err)
	}
	log.Printf("this is existing accounts in ledger %v", queryResult)
	rootTx.Commit()
}

func stringToUint128(s string) (tbTypes.Uint128, error) {
	bi, ok := new(big.Int).SetString(s, 10) // parse decimal string
	if !ok {
		return tbTypes.Uint128{}, fmt.Errorf("invalid uint128 string: %s", s)
	}
	return tbTypes.BigIntToUint128(*bi), nil
}
