package db

import "embed"

// Holds all Goose migration files embedded at build time.
// It is consumed by goose.SetBaseFS in app.go
//
//go:embed migrations
var MigrationsFS embed.FS
