package service_test

import (
	"testing"

	"challenge/internal/service"
)

var defaultSizes = []int{250, 500, 1000, 2000, 5000}

// Tests the exact examples from the coding challenge.
func TestCalculatePDFExamples(t *testing.T) {
	tests := []struct {
		name      string
		order     int
		wantTotal int // total items shipped
		wantPacks int // total number of packs
	}{
		{"order 1", 1, 250, 1},
		{"order 250", 250, 250, 1},
		{"order 251", 251, 500, 1},
		{"order 501", 501, 750, 2},
		{"order 12001", 12001, 12250, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Calculate(tt.order, defaultSizes)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			total, packs := sumResult(result)

			if total < tt.order {
				t.Errorf("total %d < order %d: under-shipped", total, tt.order)
			}
			if total != tt.wantTotal {
				t.Errorf("total items = %d, want %d", total, tt.wantTotal)
			}
			if packs != tt.wantPacks {
				t.Errorf("pack count = %d, want %d", packs, tt.wantPacks)
			}
		})
	}
}

// Tests zero excess when the order is an exact multiple.
func TestCalculateExactFit(t *testing.T) {
	cases := []int{250, 500, 1000, 2000, 5000, 7500, 10000}
	for _, order := range cases {
		result, err := service.Calculate(order, defaultSizes)
		if err != nil {
			t.Fatalf("order %d: unexpected error: %v", order, err)
		}
		total, _ := sumResult(result)
		if total != order {
			t.Errorf("order %d: expected zero excess but got total %d", order, total)
		}
	}
}

// Tests the algorithm finds minimum excess
// Sizes [3, 7]; order 11.
func TestCalculateMinimizesExcess(t *testing.T) {
	result, err := service.Calculate(11, []int{3, 7})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	total, _ := sumResult(result)
	if total != 12 {
		t.Errorf("total = %d, want 12 (excess should be minimized to 1)", total)
	}
}

// Tests that for equal excess the fewest packs are chosen.
func TestCalculateMinimizesPacks(t *testing.T) {
	result, err := service.Calculate(251, []int{250, 500})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	_, packs := sumResult(result)
	if packs != 1 {
		t.Errorf("pack count = %d, want 1 (should minimise packs for same excess)", packs)
	}
}

// Tests error handling.
func TestCalculateErrors(t *testing.T) {
	t.Run("zero order", func(t *testing.T) {
		if _, err := service.Calculate(0, defaultSizes); err == nil {
			t.Fatal("error for zero order")
		}
	})
	t.Run("negative order", func(t *testing.T) {
		if _, err := service.Calculate(-5, defaultSizes); err == nil {
			t.Fatal("error for negative order")
		}
	})
	t.Run("no sizes", func(t *testing.T) {
		if _, err := service.Calculate(100, nil); err == nil {
			t.Fatal("error for empty sizes")
		}
	})
	t.Run("exceeds max", func(t *testing.T) {
		if _, err := service.Calculate(service.MaxOrder+1, defaultSizes); err == nil {
			t.Fatal("error when order exceeds max")
		}
	})
}

// Tests shipped >= ordered for many inputs.
func TestCalculateResultNeverUnderShips(t *testing.T) {
	orders := []int{1, 99, 100, 249, 250, 251, 999, 1000, 1001, 9999, 10000, 10001, 50000}
	for _, order := range orders {
		result, err := service.Calculate(order, defaultSizes)
		if err != nil {
			t.Fatalf("order %d: error: %v", order, err)
		}
		total, _ := sumResult(result)
		if total < order {
			t.Errorf("order %d: shipped %d < ordered %d", order, total, order)
		}
	}
}

// Sums the total and number of packs in the result.
func sumResult(result []service.PackResult) (total, packs int) {
	for _, r := range result {
		total += r.Size * r.Quantity
		packs += r.Quantity
	}
	return
}
