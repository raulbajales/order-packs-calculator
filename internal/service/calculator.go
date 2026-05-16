package service

import (
	"fmt"
	"math"
	"sort"
)

// Number of packs of a given size
type PackResult struct {
	Size     int
	Quantity int
}

// Upper bound on order quantities accepted by Calculate.
// To keep the dynamic-programming table bounded in memory.
const MaxOrder = 1_000_000

// Returns the optimal pack distribution for the given order quantity
// and available pack sizes.
//
// Optimality is defined by the following rules:
//  - Total shipped items must be >= order (no partial packs).
//  - Minimise total excess items shipped.
//  - For equal excess, minimise the number of packs used.
func Calculate(order int, sizes []int) ([]PackResult, error) {
	if order <= 0 {
		return nil, fmt.Errorf("order quantity must be positive")
	}
	if order > MaxOrder {
		return nil, fmt.Errorf("order quantity exceeds maximum of %d", MaxOrder)
	}
	if len(sizes) == 0 {
		return nil, fmt.Errorf("no pack sizes configured")
	}

	// Work with a sorted-descending copy so reconstruction produces largest-first output
	sorted := make([]int, len(sizes))
	copy(sorted, sizes)
	sort.Sort(sort.Reverse(sort.IntSlice(sorted)))

	minSize := sorted[len(sorted)-1]

	// Search targets in [order, order+minSize].
	// Any excess >= minSize can always be reduced by swapping to a smaller pack,
	// so the optimal solution is guaranteed to lie within this window
	maxTarget := order + minSize

	const inf = math.MaxInt / 2

	// dp[i] = minimum number of packs to ship exactly i items
	// from[i] = the pack size chosen to reach state i
	dp := make([]int, maxTarget+1)
	from := make([]int, maxTarget+1)
	for i := 1; i <= maxTarget; i++ {
		dp[i] = inf
	}

	for i := 1; i <= maxTarget; i++ {
		for _, s := range sorted {
			if s > i {
				continue
			}
			prev := dp[i-s]
			if prev != inf && prev+1 < dp[i] {
				dp[i] = prev + 1
				from[i] = s
			}
		}
	}

	// Find the first reachable target >= order to minimise excess
	bestT := -1
	for t := order; t <= maxTarget; t++ {
		if dp[t] < inf {
			bestT = t
			break
		}
	}

	if bestT == -1 {
		return nil, fmt.Errorf("no valid pack combination found")
	}

	// Reconstruct the distribution by tracing back through from
	counts := make(map[int]int)
	for cur := bestT; cur > 0; {
		s := from[cur]
		counts[s]++
		cur -= s
	}

	return toSlice(counts, sorted), nil
}

func toSlice(counts map[int]int, sizes []int) []PackResult {
	var out []PackResult
	for _, s := range sizes {
		if n := counts[s]; n > 0 {
			out = append(out, PackResult{Size: s, Quantity: n})
		}
	}
	return out
}
