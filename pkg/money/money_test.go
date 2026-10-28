package money

import (
	"math"
	"testing"
)

func TestAdd_Success(t *testing.T) {
	res, err := Add(50000, 25000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != 75000 {
		t.Errorf("expected 75000, got %d", res)
	}
}

func TestAdd_Overflow(t *testing.T) {
	_, err := Add(math.MaxInt64, 1)
	if err != ErrOverflow {
		t.Errorf("expected ErrOverflow, got: %v", err)
	}
}

func TestSubtract_Underflow(t *testing.T) {
	_, err := Subtract(math.MinInt64, 1)
	if err != ErrUnderflow {
		t.Errorf("expected ErrUnderflow, got: %v", err)
	}
}

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "Rp 0"},
		{500, "Rp 500"},
		{1000, "Rp 1.000"},
		{50000, "Rp 50.000"},
		{2500000, "Rp 2.500.000"},
		{50000000, "Rp 50.000.000"},
		{-15000, "-Rp 15.000"},
	}

	for _, tt := range tests {
		actual := FormatRupiah(tt.input)
		if actual != tt.expected {
			t.Errorf("FormatRupiah(%d): expected %s, got %s", tt.input, tt.expected, actual)
		}
	}
}
