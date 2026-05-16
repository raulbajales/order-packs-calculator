package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"challenge/internal/config"
	appdb "challenge/internal/db"
	apphttp "challenge/internal/http"
	"challenge/internal/repository"
)

// Holds resources for the application
type App struct {
	db     *sql.DB
	server *nethttp.Server
}

// Creates a new App instance: Connects to the DB and runs migrations
func New(cfg config.Config) (*App, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL) // DB driver
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if the DB is live
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// Run migrations.
	// This is here just for simplicity: We run the migration every time 
	// the application starts, NOT A GOOD PRACTICE at all, just for this exercise,
	// in a productive environment, migrations should be run either manually or 
	// by a CI/CD pipeline.
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	repo := repository.New(db)

	// Serve web assets from the filesystem directory specified in config.
	webFS := os.DirFS(cfg.WebDir)

	handlers := apphttp.NewHandlers(repo, webFS)
	router := apphttp.NewRouter(handlers, webFS)

	// Create the HTTP server
	server := &nethttp.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{db: db, server: server}, nil
}

// Common pattern here to start the server and handle graceful shutdown.
// The server runs until the process receives SIGINT or SIGTERM, or the server 
// returns an unexpected error.
func (a *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1) // Channel for errors from server
	go func() { 
		slog.Info("server listening", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("Shutting down server...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.server.Shutdown(shutCtx)
	}
}

// Releases the database connection.
func (a *App) Close() {
	if err := a.db.Close(); err != nil {
		slog.Error("close db", "error", err)
	}
}

// Applies all pending migrations using the SQL files embedded in appdb.MigrationsFS.
func runMigrations(db *sql.DB) error {
	goose.SetBaseFS(appdb.MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	return goose.Up(db, "migrations")
}
