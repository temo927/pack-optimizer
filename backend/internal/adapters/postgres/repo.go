// Package postgres implements the PostgreSQL adapter for pack size persistence.
// Uses a versioned, append-only storage strategy where each change creates a new version.
// This provides an audit trail and enables rollback capabilities.
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

// Repository implements the pack size persistence layer using PostgreSQL.
// It uses an append-only versioning strategy where each update creates a new row.
type Repository struct {
	db *pgxpool.Pool // PostgreSQL connection pool
}

// New creates a new PostgreSQL repository instance.
func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetAllActive retrieves the latest version of pack sizes from the database.
// Returns the most recent pack_sets row ordered by version (descending).
// If no rows exist, returns an empty array instead of an error.
func (r *Repository) GetAllActive() ([]int, error) {
	const q = `SELECT sizes FROM pack_sets ORDER BY version DESC LIMIT 1`
	var arr []int32
	err := r.db.QueryRow(context.Background(), q).Scan(&arr)
	if err != nil {
		// Handle case where no rows exist (fresh database)
		if errors.Is(err, pgx.ErrNoRows) {
			return []int{}, nil
		}
		return nil, err
	}
	
	// Convert PostgreSQL int32 array to Go int slice
	sizes := make([]int, len(arr))
	for i, v := range arr {
		sizes[i] = int(v)
	}
	
	// Sort for consistency
	sort.Ints(sizes)
	return sizes, nil
}

// ReplaceActive creates a new version of pack sizes by inserting a new row.
// This implements the append-only versioning strategy - old versions are preserved.
// The function normalizes the input by:
// - Removing duplicates
// - Filtering out invalid (non-positive) values
// - Sorting the result
//
// Note: Empty arrays are allowed - validation happens at the API layer.
func (r *Repository) ReplaceActive(sizes []int) ([]int, error) {
	// Allow empty arrays - validation happens at API layer
	// Normalize: remove duplicates and invalid values
	uniq := make(map[int]struct{})
	for _, s := range sizes {
		if s > 0 {
			uniq[s] = struct{}{}
		}
	}
	
	// Rebuild sorted slice from unique values
	sizes = sizes[:0]
	for s := range uniq {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	
	// Convert to PostgreSQL int32 array format
	arr := make([]int32, len(sizes))
	for i, v := range sizes {
		arr[i] = int32(v)
	}
	
	// Insert new version with current timestamp
	const q = `INSERT INTO pack_sets (sizes, created_at) VALUES ($1, $2)`
	_, err := r.db.Exec(context.Background(), q, arr, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	
	// Return a copy of the normalized sizes
	return slices.Clone(sizes), nil
}

// CurrentVersion returns the highest version number from the pack_sets table.
// Returns 0 if no versions exist. Used for cache key generation.
func (r *Repository) CurrentVersion() (int64, error) {
	const q = `SELECT COALESCE(MAX(version),0) FROM pack_sets`
	var v int64
	err := r.db.QueryRow(context.Background(), q).Scan(&v)
	return v, err
}
