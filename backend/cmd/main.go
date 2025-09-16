package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	userLedgerRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/ledger"
	orderRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/order"
	userRepository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/user"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/router"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/router/middleware"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/user"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/websocket"
	"github.com/Yusufzhafir/go-orderbook/backend/pkg/model"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	tb "github.com/tigerbeetle/tigerbeetle-go"
	tbTypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"

	_ "github.com/lib/pq"
)

func mapToWsTrade(order model.Trade) websocket.Trade {
	side := "BUY"
	if order.Side == model.ASK {
		side = "SELL"
	}
	return websocket.Trade{
		Symbol: "ticker",
		Price:  order.Price,
		Qty:    order.Quantity,
		Side:   side,
		Ts:     order.Timestamp.UnixMilli(),
	}
}

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := log.Default()
	//load environment variable
	err := godotenv.Load()
	if err != nil {
		logger.Fatal("Error loading .env file")
	}

	tbAdress := os.Getenv("TB_ADDRESS")
	if tbAdress == "" {
		tbAdress = "3001"
	}

	tbClusterId, err := strconv.ParseUint(os.Getenv("TB_CLUSTER_ID"), 0, 64)

	if err != nil {
		tbClusterId = 1
	}

	tBClusterAddrs := []string{tbAdress}
	tbClient, err := tb.NewClient(tbTypes.ToUint128(tbClusterId), tBClusterAddrs)
	if err != nil {
		logger.Fatalf("tigerbeetle client init: %v", err)
		return
	}
	hub := websocket.NewHub(logger)
	go hub.Run(rootCtx)

	serveMux := http.NewServeMux()

	//start ws on servemux
	serveMux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(hub, w, r)
	})

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	jwtSecret := os.Getenv("JWT_SECRET")

	// construct DSN
	pgInfo := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)
	db, err := sqlx.Connect("postgres", pgInfo)
	if err != nil {
		logger.Fatalf("error connecting postgres: %v", err)
	}

	orderRepository := orderRepository.NewOrderRepository(db)
	userRepo := userRepository.NewUserRepository(db)
	userLedgerRepo := userLedgerRepository.NewLedgerRepository(db)
	userUseCaseOpts := user.UserUseCaseOpts{
		UserRepo:   &userRepo,
		LedgerRepo: &userLedgerRepo,
		Db:         db,
		TbClient:   &tbClient,
	}
	usecaseOpts := order.OrderUseCaseOpts{
		TBLedgerID:    uint32(0),
		EscrowAccount: tbTypes.BigIntToUint128(*big.NewInt(1)),
		TbClient:      &tbClient,
		Db:            db,
		LedgerRepo:    &userLedgerRepo,
		OrderRepo:     &orderRepository,
	}

	orderUseCase := order.NewOrderUseCase(rootCtx, usecaseOpts)
	userUsecase := user.NewUserUseCase(userUseCaseOpts)
	tokenMaker := middleware.NewJWTMaker(jwtSecret)
	//bind router
	bindRouterOpts := router.BindRouterOpts{
		ServerRouter: serveMux,
		OrderUseCase: &orderUseCase,
		TokenMaker:   tokenMaker,
		UserUseCase:  &userUsecase,
	}
	router.BindRouter(bindRouterOpts)
	logger.Println("finished binding router")

	corsServerMux := router.Cors(serveMux)
	server := http.Server{
		Addr:    ":8080",
		Handler: corsServerMux,
	}

	orderUseCase.RegisterTradeHandler(func(tr model.Trade) {
		// quick mapping + publish
		logger.Printf("Sending Trades")
		hub.PublishTrade(mapToWsTrade(tr))
	})

	// Start server in background.
	go func() {
		logger.Printf("HTTP server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("listen error: %v", err)
		}
	}()

	// Block until we get a signal (or parent context canceled).
	<-rootCtx.Done()
	logger.Println("shutdown signal received")

	// Give in-flight requests up to 10s to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		// If graceful shutdown times out, force close.
		logger.Printf("graceful shutdown failed: %v; forcing close", err)
		_ = server.Close()
	}

	logger.Println("server stopped")
}
