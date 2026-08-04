package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"buffetflow/internal/database"
	"buffetflow/internal/handlers"
	"buffetflow/internal/repositories"
	"buffetflow/internal/services"
)

func main() {
	var address, databasePath, timezone string
	var migrateOnly bool
	flag.StringVar(&address, "address", ":8080", "HTTP listen address")
	flag.StringVar(&databasePath, "database", "data/buffetflow.db", "SQLite database path")
	flag.StringVar(&timezone, "timezone", "America/Sao_Paulo", "business timezone")
	flag.BoolVar(&migrateOnly, "migrate-only", false, "apply migrations and exit")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	location, err := time.LoadLocation(timezone)
	if err != nil {
		logger.Error("load timezone", "error", err)
		os.Exit(1)
	}
	time.Local = location

	if directory := filepath.Dir(databasePath); directory != "." {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			logger.Error("create data directory", "error", err)
			os.Exit(1)
		}
	}
	db, err := database.Open(databasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("apply migrations", "error", err)
		os.Exit(1)
	}
	store := repositories.New(db)
	authService := services.NewAuthService(store)
	if err := authService.EnsureDemoAdmin(ctx); err != nil {
		logger.Error("seed administrator", "error", err)
		os.Exit(1)
	}
	checklistService := services.NewChecklistService(store)
	if err := checklistService.EnsureDemoChecklist(ctx); err != nil {
		logger.Error("seed demo checklist", "error", err)
		os.Exit(1)
	}
	if migrateOnly {
		fmt.Println("Migrations and demonstration data are up to date.")
		return
	}

	app := handlers.New(store, authService, checklistService, logger, location)
	server := &http.Server{
		Addr:              address,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("BuffetFlow ready", "address", "http://localhost"+address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}
