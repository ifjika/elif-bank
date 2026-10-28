package money

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// ==============================================================================
// Package money: Fixed-Point Arithmetic & Formatting
// Dokumen Terkait: docs/TSD.md (Seksi 2.3)
// Fungsi: Mencegah floating-point precision loss dan overflow pada int64
// ==============================================================================

var (
	ErrOverflow  = errors.New("arithmetic overflow detected")
	ErrUnderflow = errors.New("arithmetic underflow detected")
)

// Add menjumlahkan dua nominal int64 dengan proteksi integer overflow.
func Add(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b {
		return 0, ErrOverflow
	}
	if b < 0 && a < math.MinInt64-b {
		return 0, ErrUnderflow
	}
	return a + b, nil
}

// Subtract mengurangkan nominal a dengan b dengan proteksi integer underflow.
func Subtract(a, b int64) (int64, error) {
	if b < 0 && a > math.MaxInt64+b {
		return 0, ErrOverflow
	}
	if b > 0 && a < math.MinInt64+b {
		return 0, ErrUnderflow
	}
	return a - b, nil
}

// FormatRupiah memformat integer Rupiah menjadi string yang mudah dibaca (misal: 50000000 -> "Rp 50.000.000").
func FormatRupiah(amount int64) string {
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		if isNegative {
			return "-Rp " + str
		}
		return "Rp " + str
	}

	var sb strings.Builder
	reminder := n % 3
	if reminder > 0 {
		sb.WriteString(str[:reminder])
		sb.WriteString(".")
	}

	for i := reminder; i < n; i += 3 {
		sb.WriteString(str[i : i+3])
		if i+3 < n {
			sb.WriteString(".")
		}
	}

	res := sb.String()
	if isNegative {
		return "-Rp " + res
	}
	return "Rp " + res
}
