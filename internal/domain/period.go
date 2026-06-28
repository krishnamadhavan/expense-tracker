package domain

import (
	"fmt"
	"time"
)

// DefaultFYStartMonth is April (Indian financial year).
const DefaultFYStartMonth = time.April

// DefaultTimezone is Asia/Kolkata for India-primary households.
const DefaultTimezone = "Asia/Kolkata"

// LoadLocation loads an IANA timezone or returns an error.
func LoadLocation(name string) (*time.Location, error) {
	if name == "" {
		name = DefaultTimezone
	}
	return time.LoadLocation(name)
}

// MustLoadLocation panics if the timezone is invalid (tests / bootstrap).
func MustLoadLocation(name string) *time.Location {
	loc, err := LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// FinancialYearBounds returns the half-open interval [start, endExclusive) for the
// FY that *starts* in calendar year fyStartYear at fyStartMonth in location loc.
//
// Example: fyStartYear=2025, fyStartMonth=April, Asia/Kolkata =>
//
//	start = 2025-04-01 00:00:00 IST
//	endExclusive = 2026-04-01 00:00:00 IST
//
// UI label for this FY is typically FYLabel(fyStartYear, fyStartMonth) => "FY 2025-26".
func FinancialYearBounds(fyStartYear int, fyStartMonth time.Month, loc *time.Location) (start, endExclusive time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	if fyStartMonth < 1 || fyStartMonth > 12 {
		fyStartMonth = DefaultFYStartMonth
	}
	start = time.Date(fyStartYear, fyStartMonth, 1, 0, 0, 0, 0, loc)
	endExclusive = start.AddDate(1, 0, 0)
	return start, endExclusive
}

// FYContaining returns the FY start calendar year and bounds for the FY that contains
// instant d (interpreted in loc). The FY is defined as starting on the 1st of fyStartMonth.
//
// Example (fyStartMonth=April, Kolkata): 2025-03-31 is still FY starting 2024 (FY 2024-25);
// 2025-04-01 is FY starting 2025 (FY 2025-26).
func FYContaining(d time.Time, fyStartMonth time.Month, loc *time.Location) (fyStartYear int, start, endExclusive time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	if fyStartMonth < 1 || fyStartMonth > 12 {
		fyStartMonth = DefaultFYStartMonth
	}
	local := d.In(loc)
	y, m, _ := local.Date()
	fyStartYear = y
	if m < fyStartMonth {
		fyStartYear = y - 1
	}
	start, endExclusive = FinancialYearBounds(fyStartYear, fyStartMonth, loc)
	return fyStartYear, start, endExclusive
}

// FYLabel returns a display label for the FY starting in fyStartYear.
// For April start: "FY 2025-26". For January start (calendar year): "FY 2025".
func FYLabel(fyStartYear int, fyStartMonth time.Month) string {
	if fyStartMonth == time.January {
		return fmt.Sprintf("FY %d", fyStartYear)
	}
	endYearShort := (fyStartYear + 1) % 100
	return fmt.Sprintf("FY %d-%02d", fyStartYear, endYearShort)
}

// MonthBounds returns [start, endExclusive) for the calendar month containing d in loc.
func MonthBounds(d time.Time, loc *time.Location) (start, endExclusive time.Time) {
	if loc == nil {
		loc = time.UTC
	}
	local := d.In(loc)
	y, m, _ := local.Date()
	start = time.Date(y, m, 1, 0, 0, 0, 0, loc)
	endExclusive = start.AddDate(0, 1, 0)
	return start, endExclusive
}

// DateInLocation truncates to calendar date at 00:00 in loc (useful for txn_date).
func DateInLocation(d time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := d.In(loc)
	y, m, day := local.Date()
	return time.Date(y, m, day, 0, 0, 0, 0, loc)
}

// InHalfOpen reports whether t is in [start, endExclusive).
func InHalfOpen(t, start, endExclusive time.Time) bool {
	return !t.Before(start) && t.Before(endExclusive)
}
