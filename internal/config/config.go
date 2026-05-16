package config

import "os"

type Config struct {
	Port string
	DatabaseURL string
	WebDir string
}

// Reads configuration from env vars if set, or takes defaults.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/challenge?sslmode=disable"
	}

	webDir := os.Getenv("WEB_DIR")
	if webDir == "" {
		webDir = "./web"
	}

	return Config{
		Port:        port,
		DatabaseURL: dbURL,
		WebDir:      webDir,
	}
}
