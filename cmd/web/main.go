// Command web is the entry point for the Order Packs Calculator server.
//
// Usage:
//
//	web   Start the HTTP server (PORT, DATABASE_URL, WEB_DIR from env).
package main

import (
	"log/slog"
	"os"

	"challenge/internal/app"
	"challenge/internal/config"
)

func main() {
	cfg := config.Load()

	a, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}
	defer a.Close()

	if err := a.Run(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
