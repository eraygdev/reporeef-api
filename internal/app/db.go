package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

// Connect veritabanına bağlanır ve havuzu db'ye atar.
func Connect() {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		slog.Error("DATABASE_URL environment variable is not set")
		os.Exit(1)
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		slog.Error("failed to parse database config", "err", err)
		os.Exit(1)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	db, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		slog.Error("failed to create connection pool", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}

	slog.Info("database connected")
}

// Close graceful shutdown için.
func Close() {
	if db != nil {
		db.Close()
	}
}
