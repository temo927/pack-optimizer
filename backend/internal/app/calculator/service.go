package calculator

import (
	"context"
	"sort"

	"github.com/temo/pack-optimizer/backend/internal/domain"
)

type Result struct {
	TotalItems int
	TotalPacks int
	Counts     map[int]int
}

// Compute uses DP to find minimal total items >= amount, then minimal number of packs.
func Compute(amount int, sizes []int) Result {
	if amount <= 0 || len(sizes) == 0 {
		return Result{TotalItems: 0, TotalPacks: 0, Counts: map[int]int{}}
	}
	// sanitize sizes
	unique := make(map[int]struct{})
	for _, s := range sizes {
		if s > 0 {
			unique[s] = struct{}{}
		}
	}
	sizes = sizes[:0]
	for s := range unique {
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)
	maxS := sizes[len(sizes)-1]
	targetUpper := amount + maxS - 1

	const inf = int(^uint(0) >> 1) / 2
	dp := make([]int, targetUpper+1)
	prev := make([]int, targetUpper+1) // store chosen size
	for i := 1; i <= targetUpper; i++ {
		dp[i] = inf
		prev[i] = -1
	}
	// bottom-up
	for t := 1; t <= targetUpper; t++ {
		best := inf
		bestS := -1
		for _, s := range sizes {
			if t >= s && dp[t-s] != inf {
				if dp[t-s]+1 < best {
					best = dp[t-s] + 1
					bestS = s
				}
			}
		}
		dp[t] = best
		prev[t] = bestS
	}
	bestT := -1
	for t := amount; t <= targetUpper; t++ {
		if dp[t] != inf {
			bestT = t
			break
		}
	}
	if bestT == -1 {
		return Result{TotalItems: 0, TotalPacks: 0, Counts: map[int]int{}}
	}
	// reconstruct
	counts := map[int]int{}
	for t := bestT; t > 0; {
		s := prev[t]
		if s <= 0 {
			break
		}
		counts[s]++
		t -= s
	}
	totalPacks := 0
	for _, c := range counts {
		totalPacks += c
	}
	return Result{
		TotalItems: bestT,
		TotalPacks: totalPacks,
		Counts:     counts,
	}
}

// Service implements the domain.Calculator port.
type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Compute(ctx context.Context, amount int, sizes []int) (domain.CalculationResult, error) {
	res := Compute(amount, sizes)
	return domain.CalculationResult{
		Amount:     amount,
		TotalItems: res.TotalItems,
		Overage:    res.TotalItems - amount,
		TotalPacks: res.TotalPacks,
		Breakdown:  res.Counts,
	}, nil
}


