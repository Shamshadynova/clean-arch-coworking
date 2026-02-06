package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	bookinghttp "github.com/example/coworking/internal/booking/adapters/http"
	"github.com/example/coworking/internal/booking/application"
	busdummy "github.com/example/coworking/internal/booking/infrastructure/bus/dummy"
	"github.com/example/coworking/internal/booking/infrastructure/memory"
	"github.com/example/coworking/internal/booking/infrastructure/outbox"
	policydummy "github.com/example/coworking/internal/booking/infrastructure/policy/dummy"
	"github.com/example/coworking/internal/booking/infrastructure/transaction"
	"github.com/example/coworking/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up structured logging.
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Wire dependencies.
	repo := memory.NewBookingRepository()
	bus := busdummy.NewEventBus()
	availabilityChecker := policydummy.NewAvailabilityChecker()
	priceCalculator := policydummy.NewPriceCalculator()
	eventStore := outbox.NewEventStore(bus)
	uow := transaction.NewUnitOfWork(repo, eventStore)

	svc := application.NewService(repo, bus, availabilityChecker, priceCalculator, uow, logger)
	handler := bookinghttp.NewRouter(svc, logger)

	// Create HTTP server.
	srv := &http.Server{
		Addr:         ":" + cfg.AppPort,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting booking service", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
