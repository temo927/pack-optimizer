package platform

import (
	"os"
)

type Config struct {
	HTTPPort     string
	PostgresURL  string
	RedisAddr    string
	RedisDB      int
	RedisPass    string
	CORSOrigin   string
	CacheTTLSecs int
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func LoadConfig() Config {
	return Config{
		HTTPPort:     getenv("HTTP_PORT", "8080"),
		PostgresURL:  getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/packs?sslmode=disable"),
		RedisAddr:    getenv("REDIS_ADDR", "localhost:6379"),
		RedisPass:    os.Getenv("REDIS_PASSWORD"),
		CORSOrigin:   getenv("CORS_ORIGIN", "*"),
		CacheTTLSecs: 600,
	}
}


