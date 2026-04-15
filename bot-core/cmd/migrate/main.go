package main

import (
	"context"
	"fmt"
	"os"

	"sukoon/bot-core/internal/config"
	"sukoon/bot-core/internal/observability"
	"sukoon/bot-core/internal/persistence/postgres"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.AppEnv)
	store, err := postgres.New(context.Background(), cfg.EffectiveMigrateDatabaseURL(), logger, postgres.Options{
		MaxConns:        cfg.DatabaseMaxConns,
		MinConns:        cfg.DatabaseMinConns,
		MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		MaxConnIdleTime: cfg.DatabaseMaxConnIdleTime,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "database error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Migrate(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "migration error: %v\n", err)
		os.Exit(1)
	}
}
