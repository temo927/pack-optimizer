//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	pg "github.com/temo/pack-optimizer/backend/internal/adapters/postgres"
)

func TestPostgresRepository(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Skip("docker not available")
		return
	}
	// Postgres
	pgRes, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres", Tag: "16",
		Env: []string{"POSTGRES_PASSWORD=postgres", "POSTGRES_DB=packs"},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("pg start: %v", err)
	}
	defer pool.Purge(pgRes)
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:%s/packs?sslmode=disable", pgRes.GetPort("5432/tcp"))

	var db *pgxpool.Pool
	if err := pool.Retry(func() error {
		var e error
		db, e = pgxpool.New(context.Background(), dsn)
		if e != nil {
			return e
		}
		return db.Ping(context.Background())
	}); err != nil {
		t.Fatalf("pg ping: %v", err)
	}
	defer db.Close()
	// create schema
	_, _ = db.Exec(context.Background(), `
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE IF NOT EXISTS pack_sets (version BIGSERIAL PRIMARY KEY, sizes INTEGER[] NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
`)
	repo := pg.New(db)
	_, err = repo.ReplaceActive([]int{10, 20, 50})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	out, err := repo.GetAllActive()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(out) != 3 || out[0] != 10 || out[2] != 50 {
		t.Fatalf("unexpected sizes: %+v", out)
	}
}


