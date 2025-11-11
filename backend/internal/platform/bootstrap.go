// Package platform provides dependency injection and application bootstrapping.
// It wires together all adapters and services following the hexagonal architecture pattern.
package platform

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
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
// 1. Connecting to PostgreSQL with retry logic (waits for DB to be ready)
// 2. Connecting to Redis with retry logic (waits for cache to be ready)
// 3. Creating repository and cache adapters
// 4. Wrapping repository with caching layer
// 5. Creating calculator service
// 6. Returning configured App and cleanup function
//
// The retry logic (30 attempts with 1 second intervals) ensures the application
// can start even if dependencies aren't immediately available (useful in Docker Compose).
func Bootstrap(cfg Config) (*App, func(context.Context) error) {
	ctx := context.Background()
	
	// Connect to PostgreSQL with retry logic
	var pool *pgxpool.Pool
	var err error
	for i := 0; i < 30; i++ {
		pool, err = pgxpool.New(ctx, cfg.PostgresURL)
		if err == nil && pool != nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
		}
		log.Warn().Err(err).Int("retry", i+1).Msg("waiting for postgres")
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatal().Err(err).Msg("pg not ready")
	}

	// Connect to Redis with retry logic
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       0,
	})
	for i := 0; i < 30; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			break
		} else {
			log.Warn().Err(err).Int("retry", i+1).Msg("waiting for redis")
			time.Sleep(1 * time.Second)
		}
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
func MountRoutes(r *chi.Mux, app *App) {
	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/", httpad.NewRouter(app.PacksSvc, app.Calc))
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
