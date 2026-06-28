package domain

import (
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Money is a non-negative amount with exactly 2 decimal places (INR / NUMERIC(19,2)).
// Internally stored as integer minor units (paise for INR) to avoid float drift.
type Money struct {
	// Minor is amount * 100 (e.g. ₹10.50 => 1050). Always >= 0 when valid.
	Minor int64
}

// ZeroMoney is 0.00.
var ZeroMoney = Money{Minor: 0}

// MoneyFromMinor constructs Money from minor units (must be >= 0).
func MoneyFromMinor(minor int64) (Money, error) {
	if minor < 0 {
		return Money{}, fmt.Errorf("%w: minor units must be >= 0", ErrInvalidAmount)
	}
	return Money{Minor: minor}, nil
}

// ParseMoney parses a decimal string with up to 2 fractional digits (e.g. "10", "10.5", "10.50").
// More than 2 fractional digits is rejected (no silent rounding).
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Money{}, fmt.Errorf("%w: empty amount", ErrInvalidAmount)
	}
	if strings.HasPrefix(s, "-") {
		return Money{}, fmt.Errorf("%w: amount must be >= 0", ErrInvalidAmount)
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return Money{}, fmt.Errorf("%w: not a decimal number", ErrInvalidAmount)
	}
	if r.Sign() < 0 {
		return Money{}, fmt.Errorf("%w: amount must be >= 0", ErrInvalidAmount)
	}
	// Scale to minor units with exact 2 dp.
	scaled := new(big.Rat).Mul(r, big.NewRat(100, 1))
	if !scaled.IsInt() {
		return Money{}, fmt.Errorf("%w: at most 2 decimal places allowed", ErrInvalidAmount)
	}
	num := scaled.Num()
	if !num.IsInt64() {
		return Money{}, fmt.Errorf("%w: amount out of range", ErrInvalidAmount)
	}
	minor := num.Int64()
	if minor < 0 {
		return Money{}, fmt.Errorf("%w: amount must be >= 0", ErrInvalidAmount)
	}
	// NUMERIC(19,2) major part is 17 digits max => minor <= 10^19 - 1 conceptually;
	// use a practical int64-safe ceiling (max int64 is fine for personal finance).
	return Money{Minor: minor}, nil
}

// MustParseMoney panics on error (tests / constants only).
func MustParseMoney(s string) Money {
	m, err := ParseMoney(s)
	if err != nil {
		panic(err)
	}
	return m
}

// IsZero reports whether the amount is 0.00.
func (m Money) IsZero() bool { return m.Minor == 0 }

// String renders with exactly 2 decimal places.
func (m Money) String() string {
	sign := ""
	minor := m.Minor
	if minor < 0 {
		// Domain forbids negative; still render defensively.
		sign = "-"
		minor = -minor
	}
	major := minor / 100
	frac := minor % 100
	return fmt.Sprintf("%s%d.%02d", sign, major, frac)
}

// Float64 is for display/tests only — do not use for accumulation.
func (m Money) Float64() float64 {
	return float64(m.Minor) / 100.0
}

// Add returns m + o; errors if the result would overflow int64.
func (m Money) Add(o Money) (Money, error) {
	if o.Minor > 0 && m.Minor > math.MaxInt64-o.Minor {
		return Money{}, fmt.Errorf("%w: overflow", ErrInvalidAmount)
	}
	return Money{Minor: m.Minor + o.Minor}, nil
}
