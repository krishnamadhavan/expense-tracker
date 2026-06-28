package domain

import (
	"testing"
	"time"
)

func TestFinancialYearBounds_IndiaApril(t *testing.T) {
	loc := MustLoadLocation(DefaultTimezone)
	start, end := FinancialYearBounds(2025, time.April, loc)

	wantStart := time.Date(2025, time.April, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, time.April, 1, 0, 0, 0, 0, loc)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %v, want %v", end, wantEnd)
	}
	if FYLabel(2025, time.April) != "FY 2025-26" {
		t.Fatalf("label = %q", FYLabel(2025, time.April))
	}
}

func TestFYContaining_AsiaKolkata_Table(t *testing.T) {
	loc := MustLoadLocation("Asia/Kolkata")
	april := time.April

	cases := []struct {
		name        string
		instant     time.Time
		wantFYStart int
	}{
		{
			name:        "April 1 2025 is start of FY 2025-26",
			instant:     time.Date(2025, time.April, 1, 0, 0, 0, 0, loc),
			wantFYStart: 2025,
		},
		{
			name:        "March 31 2025 still FY 2024-25",
			instant:     time.Date(2025, time.March, 31, 23, 59, 59, 0, loc),
			wantFYStart: 2024,
		},
		{
			name:        "Jan 15 2026 in FY 2025-26",
			instant:     time.Date(2026, time.January, 15, 12, 0, 0, 0, loc),
			wantFYStart: 2025,
		},
		{
			name:        "UTC evening that is next calendar day in Kolkata",
			// 2025-03-31 20:00 UTC = 2025-04-01 01:30 IST -> FY 2025
			instant:     time.Date(2025, time.March, 31, 20, 0, 0, 0, time.UTC),
			wantFYStart: 2025,
		},
		{
			name:        "UTC morning still March 31 in Kolkata",
			// 2025-03-31 10:00 UTC = 2025-03-31 15:30 IST -> FY 2024
			instant:     time.Date(2025, time.March, 31, 10, 0, 0, 0, time.UTC),
			wantFYStart: 2024,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fy, start, end := FYContaining(tc.instant, april, loc)
			if fy != tc.wantFYStart {
				t.Fatalf("fyStartYear = %d, want %d (local=%v)", fy, tc.wantFYStart, tc.instant.In(loc))
			}
			if !InHalfOpen(tc.instant.In(loc), start, end) && !InHalfOpen(tc.instant, start, end) {
				// instant may be UTC; membership uses absolute instants — compare instants properly
			}
			// Half-open check on the same timeline: convert instant to loc for date logic already done in FYContaining.
			// Verify bounds consistency:
			s2, e2 := FinancialYearBounds(fy, april, loc)
			if !start.Equal(s2) || !end.Equal(e2) {
				t.Fatalf("bounds mismatch: got [%v, %v) want [%v, %v)", start, end, s2, e2)
			}
			// Instant's local time must fall in [start, end)
			local := tc.instant.In(loc)
			if local.Before(start) || !local.Before(end) {
				// For UTC instants, FYContaining uses .In(loc) for year/month — verify local date is in range
				if local.Before(start) || !local.Before(end) {
					t.Fatalf("local %v not in [%v, %v)", local, start, end)
				}
			}
		})
	}
}

func TestFYLabel_CalendarYearStart(t *testing.T) {
	if got := FYLabel(2025, time.January); got != "FY 2025" {
		t.Fatalf("got %q", got)
	}
}

func TestMonthBounds(t *testing.T) {
	loc := MustLoadLocation("Asia/Kolkata")
	d := time.Date(2025, time.June, 15, 10, 0, 0, 0, loc)
	start, end := MonthBounds(d, loc)
	wantStart := time.Date(2025, time.June, 1, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2025, time.July, 1, 0, 0, 0, 0, loc)
	if !start.Equal(wantStart) || !end.Equal(wantEnd) {
		t.Fatalf("got [%v, %v)", start, end)
	}
}

func TestDateInLocation(t *testing.T) {
	loc := MustLoadLocation("Asia/Kolkata")
	// UTC noon June 15 -> still June 15 in Kolkata (17:30)
	d := time.Date(2025, time.June, 15, 12, 0, 0, 0, time.UTC)
	got := DateInLocation(d, loc)
	want := time.Date(2025, time.June, 15, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
