// Package platform provides configuration management for the application.
// Configuration is loaded from environment variables with sensible defaults.
package platform

import (
	"os"
)

// Config holds all application configuration values.
type Config struct {
	HTTPPort     string // HTTP server port
	PostgresURL  string // PostgreSQL connection string
	RedisAddr    string // Redis server address
	RedisDB      int    // Redis database number
	RedisPass    string // Redis password (optional)
	CORSOrigin   string // CORS allowed origin
	CacheTTLSecs int    // Cache time-to-live in seconds
}

// getenv retrieves an environment variable or returns a default value.
// Helper function to simplify configuration loading with defaults.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// LoadConfig loads configuration from environment variables.
// Uses sensible defaults for local development if environment variables are not set.
// This allows the application to run out-of-the-box with docker-compose.
func LoadConfig() Config {
	return Config{
		HTTPPort:     getenv("HTTP_PORT", "8080"),
		PostgresURL:  getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/packs?sslmode=disable"),
		RedisAddr:    getenv("REDIS_ADDR", "localhost:6379"),
		RedisPass:    os.Getenv("REDIS_PASSWORD"),
		CORSOrigin:   getenv("CORS_ORIGIN", "*"),
		CacheTTLSecs: 600, // 10 minutes default cache TTL
	}
}
