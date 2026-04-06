package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"sukoon/bot-core/internal/app"
	"sukoon/bot-core/internal/config"
	"sukoon/bot-core/internal/observability"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.AppEnv)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runtime, err := app.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("bootstrap failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer runtime.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	if runtime.HTTPServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runtime.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("http server: %w", err)
			}
		}()
	}

	if runtime.Worker != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runtime.Worker.Run(ctx); err != nil && ctx.Err() == nil {
				errCh <- fmt.Errorf("worker: %w", err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("runtime failure", slog.Any("error", err))
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if runtime.HTTPServer != nil {
		if err := runtime.HTTPServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http shutdown failed", slog.Any("error", err))
		}
	}

	wg.Wait()
}
