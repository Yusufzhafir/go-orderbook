package main

import (
	"context"
	"errors"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Yusufzhafir/go-orderbook/backend/internal/engine"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/router"
	"github.com/Yusufzhafir/go-orderbook/backend/internal/usecase/order"
	"github.com/joho/godotenv"
	"github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := log.Default()
	//load environment variable
	err := godotenv.Load()
	if err != nil {
		logger.Fatal("Error loading .env file")
	}

	ob := engine.NewOrderBookEngine()
	ob.Initialize()
	usecaseOpts := order.OrderUseCaseOpts{
		OrderBookEngine: ob,
		TBClusterAddrs:  []string{"3000"},
		TBLedgerID:      uint32(0),
		EscrowAccount:   types.BigIntToUint128(*big.NewInt(1)),
	}
	orderUseCase, err := order.NewOrderUseCase(rootCtx, usecaseOpts)

	if err != nil {
		logger.Fatalf("error creating order usecase: %v", err)
	}
	serveMux := http.NewServeMux()
	//bind router
	bindRouterOpts := router.BindRouterOpts{
		ServerRouter: serveMux,
		OrderUseCase: &orderUseCase,
	}
	router.BindRouter(bindRouterOpts)
	logger.Println("finished binding router")

	server := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
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
