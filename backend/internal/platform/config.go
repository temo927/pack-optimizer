// Package platform provides configuration management for the application.
// Configuration is loaded from environment variables with sensible defaults.
package platform

import (
	"os"
)

// Config holds all application configuration values.
type Config struct {
	HTTPPort          string // HTTP server port
	PostgresURL       string // PostgreSQL connection string
	RedisAddr         string // Redis server address
	RedisDB           int    // Redis database number
	RedisPass         string // Redis password (optional)
	CORSOrigin        string // CORS allowed origin
	CacheTTLSecs      int    // Cache time-to-live in seconds
	RateLimitEnabled  bool   // Whether rate limiting is enabled
	RateLimitRPM      string // Rate limit requests per minute
	RateLimitBurst    string // Rate limit burst size
	DDoSProtectionEnabled bool   // Whether DDoS protection is enabled
	MaxRequestSize    string // Maximum request body size in bytes
	MaxHeaderSize     string // Maximum header size in bytes
	Environment       string // Environment (development, production)
}

// getenv retrieves an environment variable or returns a default value.
// Helper function to simplify configuration loading with defaults.
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvBool retrieves a boolean environment variable.
func getenvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v == "true" || v == "1" || v == "yes"
}

// LoadConfig loads configuration from environment variables.
// Uses sensible defaults for local development if environment variables are not set.
// This allows the application to run out-of-the-box with docker-compose.
func LoadConfig() Config {
	return Config{
		HTTPPort:              getenv("HTTP_PORT", "8080"),
		PostgresURL:           getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/packs?sslmode=disable"),
		RedisAddr:             getenv("REDIS_ADDR", "localhost:6379"),
		RedisPass:             os.Getenv("REDIS_PASSWORD"),
		CORSOrigin:            getenv("CORS_ORIGIN", "*"),
		CacheTTLSecs:          600, // 10 minutes default cache TTL
		RateLimitEnabled:      getenvBool("RATE_LIMIT_ENABLED", true),
		RateLimitRPM:          getenv("RATE_LIMIT_RPM", "100"), // 100 requests per minute default
		RateLimitBurst:        getenv("RATE_LIMIT_BURST", ""),  // Auto-calculated if empty
		DDoSProtectionEnabled: getenvBool("DDOS_PROTECTION_ENABLED", true),
		MaxRequestSize:        getenv("MAX_REQUEST_SIZE", "10485760"), // 10MB default
		MaxHeaderSize:         getenv("MAX_HEADER_SIZE", "8192"),      // 8KB default
		Environment:           getenv("ENVIRONMENT", "development"),
	}
}
