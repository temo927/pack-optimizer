// Package platform provides dependency injection and application bootstrapping.
// It wires together all adapters and services following the hexagonal architecture pattern.
package platform

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	httpad "github.com/temo/pack-optimizer/backend/internal/adapters/http"
	pg "github.com/temo/pack-optimizer/backend/internal/adapters/postgres"
	redisad "github.com/temo/pack-optimizer/backend/internal/adapters/redis"
	"github.com/temo/pack-optimizer/backend/internal/app/calculator"
	"github.com/temo/pack-optimizer/backend/internal/domain"
)

// App represents the fully configured application with all its dependencies.
type App struct {
	PacksSvc packsServiceFacade // Service for managing pack sizes (with caching)
	Calc     domain.Calculator  // Service for calculating optimal pack distributions
}

// packsServiceFacade defines the minimal interface needed by HTTP handlers.
// This follows the Interface Segregation Principle - handlers only see what they need.
type packsServiceFacade interface {
	GetActiveSizes(ctx context.Context) ([]int, error)
	ReplaceActive(ctx context.Context, sizes []int) ([]int, error)
}

// Bootstrap initializes the application by:
// 1. Connecting to PostgreSQL with retry logic and circuit breaker
// 2. Connecting to Redis with retry logic and circuit breaker
// 3. Creating repository and cache adapters
// 4. Wrapping repository with caching layer
// 5. Creating calculator service
// 6. Returning configured App and cleanup function
//
// Uses exponential backoff retry and circuit breaker pattern for resilience.
func Bootstrap(cfg Config, logger *slog.Logger) (*App, func(context.Context) error) {
	ctx := context.Background()
	
	if logger == nil {
		logger = slog.Default()
	}
	
	// Configure retry with exponential backoff
	retryConfig := RetryConfig{
		MaxAttempts:       30,
		InitialDelay:      1 * time.Second,
		MaxDelay:          10 * time.Second,
		BackoffMultiplier: 1.5,
	}
	
	// Create circuit breakers for external dependencies
	dbCircuitBreaker := NewCircuitBreaker(logger, 5, 30*time.Second)
	redisCircuitBreaker := NewCircuitBreaker(logger, 5, 30*time.Second)
	
	// Connect to PostgreSQL with retry logic and circuit breaker
	pool, err := ConnectPostgresWithRetry(ctx, logger, cfg.PostgresURL, retryConfig, dbCircuitBreaker)
	if err != nil {
		logger.Error("postgres not ready after retries", "error", err)
		panic(err)
	}

	// Connect to Redis with retry logic and circuit breaker
	rdb, err := ConnectRedisWithRetry(ctx, logger, cfg.RedisAddr, cfg.RedisPass, retryConfig, redisCircuitBreaker)
	if err != nil {
		logger.Error("redis not ready after retries", "error", err)
		panic(err)
	}

	// Create adapters
	repo := pg.New(pool)              // PostgreSQL repository
	cache := redisad.New(rdb)         // Redis cache adapter
	
	// Wrap repository with caching layer
	ps := &packsService{repo: repo, cache: cache, ttl: cfg.CacheTTLSecs}
	
	// Create calculator service
	calc := calculator.NewService()

	// Return configured app and cleanup function
	return &App{PacksSvc: ps, Calc: calc}, func(ctx context.Context) error {
		rdb.Close()
		pool.Close()
		return nil
	}
}

// MountRoutes registers all API routes on the provided router.
// Routes are mounted under the /api/v1 prefix.
func MountRoutes(r *chi.Mux, app *App, errorHandler *httpad.ErrorHandler) {
	r.Route("/api/v1", func(api chi.Router) {
		// Add recovery middleware to catch panics
		api.Use(httpad.RecoveryMiddleware(errorHandler))
		// Add request ID middleware for tracing
		api.Use(httpad.RequestIDMiddleware)
		// Mount API routes
		api.Mount("/", httpad.NewRouter(app.PacksSvc, app.Calc, errorHandler))
	})
}

// packsService implements the packsServiceFacade interface.
// It wraps the repository with a caching layer to improve performance.
// Cache keys are version-based to ensure proper invalidation on updates.
type packsService struct {
	repo  interface {
		GetAllActive() ([]int, error)
		ReplaceActive(sizes []int) ([]int, error)
		CurrentVersion() (int64, error)
	}
	cache interface {
		Get(key string) ([]byte, error)
		Set(key string, value []byte, ttlSeconds int) error
		DeleteByPrefix(prefix string) error
	}
	ttl int // Cache time-to-live in seconds
}

// GetActiveSizes retrieves pack sizes with caching.
// First checks cache using version-based key, falls back to repository if cache miss.
// Caches the result for future requests.
func (p *packsService) GetActiveSizes(ctx context.Context) ([]int, error) {
	// Get current version for cache key
	ver, _ := p.repo.CurrentVersion()
	key := "packlist:v1:" + strconv.FormatInt(ver, 10)
	
	// Try cache first
	if b, _ := p.cache.Get(key); b != nil {
		var out []int
		_ = json.Unmarshal(b, &out)
		return out, nil
	}
	
	// Cache miss - fetch from repository
	sizes, err := p.repo.GetAllActive()
	if err != nil {
		return nil, err
	}
	
	// Cache the result for future requests
	if b, err := json.Marshal(sizes); err == nil {
		_ = p.cache.Set(key, b, p.ttl)
	}
	
	return sizes, nil
}

// ReplaceActive updates pack sizes and invalidates related cache entries.
// After updating the repository, it clears all pack list and calculation caches
// to ensure consistency.
func (p *packsService) ReplaceActive(ctx context.Context, sizes []int) ([]int, error) {
	// Update repository (creates new version)
	out, err := p.repo.ReplaceActive(sizes)
	if err != nil {
		return nil, err
	}
	
	// Invalidate all related caches
	_ = p.cache.DeleteByPrefix("packlist:v1:")
	_ = p.cache.DeleteByPrefix("calc:v1:")
	
	return out, nil
}
