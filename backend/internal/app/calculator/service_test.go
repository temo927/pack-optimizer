package calculator

import (
	"testing"
)

func TestCompute_StandardPackSizes(t *testing.T) {
	sizes := []int{250, 500, 1000, 2000, 5000}
	
	tests := []struct {
		name           string
		amount         int
		expectedItems  int
		expectedPacks  int
		expectedBreakdown map[int]int
	}{
		{
			name:           "Amount 1 - should use smallest pack",
			amount:         1,
			expectedItems:  250,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{250: 1},
		},
		{
			name:           "Amount 250 - exact match",
			amount:         250,
			expectedItems:  250,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{250: 1},
		},
		{
			name:           "Amount 251 - should use 1x500 (minimal items, minimal packs)",
			amount:         251,
			expectedItems:  500,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{500: 1},
		},
		{
			name:           "Amount 500 - exact match",
			amount:         500,
			expectedItems:  500,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{500: 1},
		},
		{
			name:           "Amount 501 - should use 1x500 + 1x250",
			amount:         501,
			expectedItems:  750,
			expectedPacks:  2,
			expectedBreakdown: map[int]int{500: 1, 250: 1},
		},
		{
			name:           "Amount 750 - exact combination",
			amount:         750,
			expectedItems:  750,
			expectedPacks:  2,
			expectedBreakdown: map[int]int{500: 1, 250: 1},
		},
		{
			name:           "Amount 1000 - exact match",
			amount:         1000,
			expectedItems:  1000,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{1000: 1},
		},
		{
			name:           "Amount 10000 - should use 2x5000",
			amount:         10000,
			expectedItems:  10000,
			expectedPacks:  2,
			expectedBreakdown: map[int]int{5000: 2},
		},
		{
			name:           "Amount 12001 - from requirements example",
			amount:         12001,
			expectedItems:  12250,
			expectedPacks:  4,
			expectedBreakdown: map[int]int{5000: 2, 2000: 1, 250: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Compute(tt.amount, sizes)
			if res.TotalItems != tt.expectedItems {
				t.Errorf("TotalItems: expected %d, got %d", tt.expectedItems, res.TotalItems)
			}
			if res.TotalPacks != tt.expectedPacks {
				t.Errorf("TotalPacks: expected %d, got %d", tt.expectedPacks, res.TotalPacks)
			}
			if len(res.Counts) != len(tt.expectedBreakdown) {
				t.Errorf("Breakdown length: expected %d, got %d", len(tt.expectedBreakdown), len(res.Counts))
			}
			for size, count := range tt.expectedBreakdown {
				if res.Counts[size] != count {
					t.Errorf("Breakdown[%d]: expected %d, got %d", size, count, res.Counts[size])
				}
			}
		})
	}
}

func TestCompute_EdgeCasePackSizes(t *testing.T) {
	sizes := []int{23, 31, 53}
	
	tests := []struct {
		name           string
		amount         int
		expectedItems  int
		expectedPacks  int
		expectedBreakdown map[int]int
	}{
		{
			name:           "Amount 500000 - critical edge case",
			amount:         500000,
			expectedItems:  500000,
			expectedPacks:  9438,
			expectedBreakdown: map[int]int{23: 2, 31: 7, 53: 9429},
		},
		{
			name:           "Amount 1 - should use smallest pack",
			amount:         1,
			expectedItems:  23,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{23: 1},
		},
		{
			name:           "Amount 23 - exact match",
			amount:         23,
			expectedItems:  23,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{23: 1},
		},
		{
			name:           "Amount 24 - should use 1x31 (minimal items)",
			amount:         24,
			expectedItems:  31,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{31: 1},
		},
		{
			name:           "Amount 31 - exact match",
			amount:         31,
			expectedItems:  31,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{31: 1},
		},
		{
			name:           "Amount 53 - exact match",
			amount:         53,
			expectedItems:  53,
			expectedPacks:  1,
			expectedBreakdown: map[int]int{53: 1},
		},
		{
			name:           "Amount 54 - should use exact combination",
			amount:         54,
			expectedItems:  54, // 31 + 23 (exact match!)
			expectedPacks:  2,
			expectedBreakdown: map[int]int{31: 1, 23: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Compute(tt.amount, sizes)
			if res.TotalItems != tt.expectedItems {
				t.Errorf("TotalItems: expected %d, got %d", tt.expectedItems, res.TotalItems)
			}
			if res.TotalPacks != tt.expectedPacks {
				t.Errorf("TotalPacks: expected %d, got %d", tt.expectedPacks, res.TotalPacks)
			}
			for size, count := range tt.expectedBreakdown {
				if res.Counts[size] != count {
					t.Errorf("Breakdown[%d]: expected %d, got %d", size, count, res.Counts[size])
				}
			}
		})
	}
}

func TestCompute_BoundaryConditions(t *testing.T) {
	sizes := []int{250, 500, 1000}
	
	tests := []struct {
		name          string
		amount        int
		shouldSucceed bool
	}{
		{
			name:          "Amount 0 - should return empty result",
			amount:        0,
			shouldSucceed: true, // Returns empty but valid
		},
		{
			name:          "Amount 1 - should succeed",
			amount:        1,
			shouldSucceed: true,
		},
		{
			name:          "Amount 1000000 - large amount",
			amount:        1000000,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Compute(tt.amount, sizes)
			if tt.shouldSucceed {
				if tt.amount > 0 && res.TotalItems < tt.amount {
					t.Errorf("TotalItems %d should be >= amount %d", res.TotalItems, tt.amount)
				}
				if tt.amount > 0 && res.TotalPacks == 0 {
					t.Errorf("Should have at least 1 pack for amount %d", tt.amount)
				}
			}
		})
	}
}

func TestCompute_OptimizationRules(t *testing.T) {
	t.Run("Rule 2: Minimal items takes precedence", func(t *testing.T) {
		// Amount 251 with [250, 500] should choose 1x500 (not 2x250)
		res := Compute(251, []int{250, 500})
		if res.TotalItems != 500 {
			t.Errorf("Expected 500 items (1x500), got %d", res.TotalItems)
		}
		if res.TotalPacks != 1 {
			t.Errorf("Expected 1 pack, got %d", res.TotalPacks)
		}
		if res.Counts[500] != 1 {
			t.Errorf("Expected 1 pack of 500, got %d", res.Counts[500])
		}
	})

	t.Run("Rule 3: Minimal packs when items are equal", func(t *testing.T) {
		// Amount 1000 with [500, 1000] should choose 1x1000 (not 2x500)
		res := Compute(1000, []int{500, 1000})
		if res.TotalItems != 1000 {
			t.Errorf("Expected 1000 items, got %d", res.TotalItems)
		}
		if res.TotalPacks != 1 {
			t.Errorf("Expected 1 pack, got %d", res.TotalPacks)
		}
		if res.Counts[1000] != 1 {
			t.Errorf("Expected 1 pack of 1000, got %d", res.Counts[1000])
		}
	})

	t.Run("Rule 1: Only whole packs", func(t *testing.T) {
		res := Compute(263, []int{250, 500, 1000, 2000, 5000})
		// Verify all pack counts are integers (they always are, but verify total)
		totalFromBreakdown := 0
		for size, count := range res.Counts {
			if count < 0 {
				t.Errorf("Pack count cannot be negative: %d x %d", count, size)
			}
			totalFromBreakdown += size * count
		}
		if totalFromBreakdown != res.TotalItems {
			t.Errorf("Breakdown total %d doesn't match TotalItems %d", totalFromBreakdown, res.TotalItems)
		}
		if res.TotalItems < 263 {
			t.Errorf("TotalItems %d must be >= requested amount 263", res.TotalItems)
		}
	})
}

func TestCompute_EmptyAndInvalidInputs(t *testing.T) {
	t.Run("Empty sizes", func(t *testing.T) {
		res := Compute(100, []int{})
		if res.TotalItems != 0 || res.TotalPacks != 0 {
			t.Errorf("Expected empty result, got %+v", res)
		}
	})

	t.Run("Zero amount", func(t *testing.T) {
		res := Compute(0, []int{250, 500})
		if res.TotalItems != 0 || res.TotalPacks != 0 {
			t.Errorf("Expected empty result for zero amount, got %+v", res)
		}
	})

	t.Run("Negative amount", func(t *testing.T) {
		res := Compute(-1, []int{250, 500})
		if res.TotalItems != 0 || res.TotalPacks != 0 {
			t.Errorf("Expected empty result for negative amount, got %+v", res)
		}
	})

	t.Run("Invalid sizes filtered out", func(t *testing.T) {
		res := Compute(250, []int{0, -1, 250, 500})
		if res.TotalItems != 250 {
			t.Errorf("Expected 250, got %d", res.TotalItems)
		}
		if res.Counts[250] != 1 {
			t.Errorf("Expected 1 pack of 250")
		}
	})
}

func TestCompute_SpecialScenarios(t *testing.T) {
	t.Run("Single pack size", func(t *testing.T) {
		res := Compute(263, []int{250})
		if res.TotalItems < 263 {
			t.Errorf("TotalItems %d must be >= 263", res.TotalItems)
		}
		if res.Counts[250] == 0 {
			t.Errorf("Should use pack size 250")
		}
	})

	t.Run("Two pack sizes", func(t *testing.T) {
		res := Compute(750, []int{250, 500})
		if res.TotalItems != 750 {
			t.Errorf("Expected 750, got %d", res.TotalItems)
		}
		if res.TotalPacks != 2 {
			t.Errorf("Expected 2 packs, got %d", res.TotalPacks)
		}
	})

	t.Run("Large amount with small packs", func(t *testing.T) {
		res := Compute(100000, []int{23, 31, 53})
		if res.TotalItems < 100000 {
			t.Errorf("TotalItems %d must be >= 100000", res.TotalItems)
		}
		if res.TotalPacks == 0 {
			t.Errorf("Should have packs")
		}
	})
}
