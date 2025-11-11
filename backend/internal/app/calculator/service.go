// Package calculator implements the core pack optimization algorithm using dynamic programming.
// The algorithm finds the optimal combination of packs to fulfill an order while minimizing
// total items (Rule 2) and then minimizing number of packs (Rule 3).
package calculator

import (
	"context"
	"sort"

	"github.com/temo/pack-optimizer/backend/internal/domain"
)

// Result represents the output of a pack calculation.
type Result struct {
	TotalItems int         // Total number of items in the solution
	TotalPacks int         // Total number of packs needed
	Counts     map[int]int // Map of pack size -> quantity needed
}

// Compute uses dynamic programming to find the minimal total items >= amount,
// then minimizes the number of packs. This implements the business rules:
// Rule 1: Only whole packs (no partial packs)
// Rule 2: Minimize total items (minimize overage) - takes precedence
// Rule 3: Minimize number of packs (when items are equal)
//
// Algorithm:
// 1. Sanitize and sort pack sizes
// 2. Build DP table where dp[i] = minimum packs needed for i items
// 3. For each target amount, try all pack sizes and choose optimal combination
// 4. Find the best target >= amount with minimum items, then minimum packs
// 5. Reconstruct solution by backtracking through choices
//
// Time Complexity: O(amount × pack_sizes)
// Space Complexity: O(amount)
func Compute(amount int, sizes []int) Result {
	// Handle edge cases
	if amount <= 0 || len(sizes) == 0 {
		return Result{TotalItems: 0, TotalPacks: 0, Counts: map[int]int{}}
	}
	
	// Sanitize sizes: remove duplicates, filter invalid values, and sort
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
	
	// Calculate upper bound for DP table
	// We need to search up to amount + maxSize - 1 to find optimal solution
	maxS := sizes[len(sizes)-1]
	targetUpper := amount + maxS - 1
	
	// Initialize DP table with infinity (representing impossible states)
	const inf = int(^uint(0) >> 1) / 2
	dp := make([]int, targetUpper+1)      // dp[i] = minimum packs needed for i items
	prev := make([]int, targetUpper+1)    // prev[i] = pack size used to reach i items
	
	// Initialize all states as impossible
	for i := 1; i <= targetUpper; i++ {
		dp[i] = inf
		prev[i] = -1
	}
	
	// Base case: 0 items requires 0 packs
	dp[0] = 0
	
	// Bottom-up DP: fill the table for all possible item counts
	for t := 1; t <= targetUpper; t++ {
		best := inf      // Best (minimum) number of packs found so far
		bestS := -1     // Pack size that gives the best result
		
		// Try each pack size to see if we can improve the solution
		for _, s := range sizes {
			// Check if we can use this pack size (target >= size)
			// and if we have a valid solution for (target - size)
			if t >= s && dp[t-s] != inf {
				// If using this pack size gives fewer total packs, update best
				if dp[t-s]+1 < best {
					best = dp[t-s] + 1
					bestS = s
				}
			}
		}
		
		dp[t] = best
		prev[t] = bestS
	}
	
	// Find the best target >= amount with minimum items (Rule 2)
	// If multiple targets have same items, choose one with minimum packs (Rule 3)
	bestT := -1
	for t := amount; t <= targetUpper; t++ {
		if dp[t] != inf {
			bestT = t
			break // First valid solution has minimum items (since we search in order)
		}
	}
	
	// If no solution found, return empty result
	if bestT == -1 {
		return Result{TotalItems: 0, TotalPacks: 0, Counts: map[int]int{}}
	}
	
	// Reconstruct the solution by backtracking through prev array
	counts := map[int]int{}
	for t := bestT; t > 0; {
		s := prev[t]
		if s <= 0 {
			break
		}
		counts[s]++
		t -= s
	}
	
	// Calculate total packs
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
// This is the application service that wraps the Compute function
// and converts it to the domain interface format.
type Service struct{}

// NewService creates a new calculator service instance.
func NewService() *Service { return &Service{} }

// Compute implements the domain.Calculator interface.
// It calls the core Compute function and converts the result to domain format,
// including calculating the overage (difference between total items and requested amount).
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
