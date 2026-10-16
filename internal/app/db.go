package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker/v2"
)

var (
	db      *pgxpool.Pool
	cbQuery *gobreaker.CircuitBreaker[interface{}]
)

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

	// Circuit breaker — DB geçici olarak yanıt vermezse
	// yeni istekleri hızlıca reddet, sıraya dizip backend'i kilitlemesin.
	//
	// Örnek: 5 ardışık hata → 30 sn "açık" (hiç deneme) → yarım açık
	// (1 istek dene) → başarılıysa kapan, başarısızsa tekrar aç.
	cbQuery = gobreaker.NewCircuitBreaker[interface{}](gobreaker.Settings{
		Name:        "pgx-query",
		MaxRequests: 1,                // Yarım açık modda 1 istek denemesi
		Interval:    60 * time.Second, // Kapalı modda sayacı sıfırla
		Timeout:     30 * time.Second, // Açık moddan yarı açığa geçiş süresi
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state change",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})

	slog.Info("database connected")
}

// queryRowCB — circuit breaker ile sarmalanmış QueryRow.
// DB çağrıları bunu kullanmalı (mevcut db.QueryRow yerine).
func queryRowCB(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	result, err := cbQuery.Execute(func() (interface{}, error) {
		return db.QueryRow(ctx, sql, args...), nil
	})
	if err != nil {
		return &errorRow{err: err}
	}
	return result.(pgx.Row)
}

// errorRow — circuit breaker açıkken QueryRow yerine döndürülür.
type errorRow struct {
	err error
}

func (r *errorRow) Scan(dest ...interface{}) error {
	return r.err
}

// execCB — circuit breaker ile sarmalanmış Exec.
func execCB(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	result, err := cbQuery.Execute(func() (interface{}, error) {
		return db.Exec(ctx, sql, args...)
	})
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	return result.(pgconn.CommandTag), nil
}

// pingCB — circuit breaker ile sarmalanmış Ping.
func pingCB(ctx context.Context) error {
	_, err := cbQuery.Execute(func() (interface{}, error) {
		return nil, db.Ping(ctx)
	})
	return err
}

// queryCB — circuit breaker ile sarmalanmış Query.
func queryCB(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	result, err := cbQuery.Execute(func() (interface{}, error) {
		return db.Query(ctx, sql, args...)
	})
	if err != nil {
		return nil, err
	}
	return result.(pgx.Rows), nil
}

// Close graceful shutdown için.
func Close() {
	if db != nil {
		db.Close()
	}
}
