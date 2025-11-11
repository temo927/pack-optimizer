// Package domain defines the core business models and interfaces (ports) for hexagonal architecture.
// This layer is framework-agnostic and contains no external dependencies.
// It defines contracts that adapters must implement, following the Dependency Inversion Principle.
package domain

import "context"

// CalculationRequest represents a request to calculate optimal pack distribution.
type CalculationRequest struct {
	Amount int   `json:"amount"`           // Number of items to fulfill
	Sizes  []int `json:"sizes,omitempty"`  // Optional custom pack sizes
}

// CalculationResult represents the result of a pack calculation.
type CalculationResult struct {
	Amount     int         `json:"amount"`     // Original requested amount
	TotalItems int         `json:"totalItems"` // Total items in solution (may exceed amount)
	Overage    int         `json:"overage"`    // Difference between totalItems and amount
	TotalPacks int         `json:"totalPacks"` // Total number of packs needed
	Breakdown  map[int]int `json:"breakdown"`  // Map of pack size -> quantity needed
}

// Ports (hexagonal architecture)
// These interfaces define contracts that adapters must implement.
// The domain layer depends on abstractions, not concrete implementations.

// PackRepository is the port for pack size persistence.
// Implementations can use PostgreSQL, MongoDB, or any other storage.
type PackRepository interface {
	// GetAllActive returns the current active pack sizes.
	GetAllActive() ([]int, error)
	
	// ReplaceActive replaces all pack sizes with a new set.
	// Returns the normalized (sorted, deduplicated) sizes.
	ReplaceActive(sizes []int) ([]int, error)
	
	// CurrentVersion returns the highest version number.
	// Used for cache key generation in versioned storage.
	CurrentVersion() (int64, error)
}

// Cache is the port for caching operations.
// Implementations can use Redis, Memcached, or in-memory cache.
type Cache interface {
	// Get retrieves a value from cache by key.
	// Returns nil if key doesn't exist.
	Get(key string) ([]byte, error)
	
	// Set stores a value in cache with a time-to-live.
	Set(key string, value []byte, ttlSeconds int) error
	
	// DeleteByPrefix removes all keys matching the given prefix.
	// Used for cache invalidation when data changes.
	DeleteByPrefix(prefix string) error
}

// Calculator is the port for pack calculation operations.
// This defines the application service interface for computing optimal pack distributions.
type Calculator interface {
	// Compute calculates the optimal pack distribution for a given amount.
	// Uses the provided pack sizes, or active sizes if not specified.
	// Returns a result with breakdown showing how many packs of each size are needed.
	Compute(ctx context.Context, amount int, sizes []int) (CalculationResult, error)
}
