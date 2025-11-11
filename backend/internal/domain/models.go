package domain

import "context"

type CalculationRequest struct {
	Amount int   `json:"amount"`
	Sizes  []int `json:"sizes,omitempty"`
}

type CalculationResult struct {
	Amount     int            `json:"amount"`
	TotalItems int            `json:"totalItems"`
	Overage    int            `json:"overage"`
	TotalPacks int            `json:"totalPacks"`
	Breakdown  map[int]int    `json:"breakdown"`
}

// Ports (hexagonal)
// Persistence
type PackRepository interface {
	GetAllActive() ([]int, error)
	ReplaceActive(sizes []int) ([]int, error)
	CurrentVersion() (int64, error)
}

// Caching
type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttlSeconds int) error
	DeleteByPrefix(prefix string) error
}

// Application
type Calculator interface {
	Compute(ctx context.Context, amount int, sizes []int) (CalculationResult, error)
}


