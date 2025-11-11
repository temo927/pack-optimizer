package postgres

import (
	"context"
	"errors"
	"slices"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetAllActive() ([]int, error) {
	const q = `SELECT sizes FROM pack_sets ORDER BY version DESC LIMIT 1`
	var arr []int32
	err := r.db.QueryRow(context.Background(), q).Scan(&arr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return empty array if no rows exist
			return []int{}, nil
		}
		return nil, err
	}
	sizes := make([]int, len(arr))
	for i, v := range arr {
		sizes[i] = int(v)
	}
	sort.Ints(sizes)
	return sizes, nil
}

func (r *Repository) ReplaceActive(sizes []int) ([]int, error) {
	// Allow empty arrays - validation happens at API layer
	// normalize
	uniq := make(map[int]struct{})
	for _, s := range sizes {
		if s > 0 {
			uniq[s] = struct{}{}
		}
	}
	sizes = sizes[:0]
	for s := range uniq {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	arr := make([]int32, len(sizes))
	for i, v := range sizes {
		arr[i] = int32(v)
	}
	const q = `INSERT INTO pack_sets (sizes, created_at) VALUES ($1, $2)`
	_, err := r.db.Exec(context.Background(), q, arr, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return slices.Clone(sizes), nil
}

func (r *Repository) CurrentVersion() (int64, error) {
	const q = `SELECT COALESCE(MAX(version),0) FROM pack_sets`
	var v int64
	err := r.db.QueryRow(context.Background(), q).Scan(&v)
	return v, err
}


