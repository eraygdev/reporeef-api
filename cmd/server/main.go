package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"

	"github.com/eraygdev/reporeef-api/internal/app"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// .env yoksa devam — sistem env değişkenleri kullanılır.
		// Logger henüz init olmadığı için print ile geçici bildirim.
		println("Warning: .env file not found, using system environment variables")
	}

	// slog'u başlat — bundan sonra tüm log'lar structured.
	app.InitLogger()

	// Sentry başlat — DSN yoksa atla (lokal dev için)
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      os.Getenv("ENV"),
			Release:          "reporeef-api@1.0.0",
			TracesSampleRate: 0.1, // 10% tracing sample
		})
		if err != nil {
			slog.Error("sentry initialization failed", "err", err)
		} else {
			slog.Info("sentry initialized")
			defer sentry.Flush(2 * time.Second)
		}
	} else {
		slog.Info("SENTRY_DSN not set — Sentry disabled")
	}

	app.Connect()
	defer app.Close()
	app.InitOAuth()

	// OAuth state cleanup — süresi geçmiş state'leri temizler
	go app.StartOAuthCleanup()

	router := app.NewRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server started", "addr", "http://localhost:"+port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "err", err)
		os.Exit(1)
	}

	slog.Info("server exited gracefully")
}
