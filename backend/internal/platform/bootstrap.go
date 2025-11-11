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

type App struct {
	PacksSvc packsServiceFacade
	Calc     domain.Calculator
}

// packsServiceFacade defines the minimal methods handlers need; implemented below.
type packsServiceFacade interface {
	GetActiveSizes(ctx context.Context) ([]int, error)
	ReplaceActive(ctx context.Context, sizes []int) ([]int, error)
}

func Bootstrap(cfg Config) (*App, func(context.Context) error) {
	ctx := context.Background()
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

	repo := pg.New(pool)
	cache := redisad.New(rdb)
	ps := &packsService{repo: repo, cache: cache, ttl: cfg.CacheTTLSecs}
	calc := calculator.NewService()

	return &App{PacksSvc: ps, Calc: calc}, func(ctx context.Context) error {
		rdb.Close()
		pool.Close()
		return nil
	}
}

func MountRoutes(r *chi.Mux, app *App) {
	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/", httpad.NewRouter(app.PacksSvc, app.Calc))
	})
}

// service implementation with caching around repo
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
	ttl int
}

func (p *packsService) GetActiveSizes(ctx context.Context) ([]int, error) {
	ver, _ := p.repo.CurrentVersion()
	key := "packlist:v1:" + strconv.FormatInt(ver, 10)
	if b, _ := p.cache.Get(key); b != nil {
		var out []int
		_ = json.Unmarshal(b, &out)
		return out, nil
	}
	sizes, err := p.repo.GetAllActive()
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(sizes); err == nil {
		_ = p.cache.Set(key, b, p.ttl)
	}
	return sizes, nil
}

func (p *packsService) ReplaceActive(ctx context.Context, sizes []int) ([]int, error) {
	out, err := p.repo.ReplaceActive(sizes)
	if err != nil {
		return nil, err
	}
	_ = p.cache.DeleteByPrefix("packlist:v1:")
	_ = p.cache.DeleteByPrefix("calc:v1:")
	return out, nil
}


