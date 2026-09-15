package config

import (
	"testing"
	"time"
)

func TestCostWindowTakesTheDefaultForAnUnknownValue(t *testing.T) {
	cases := map[string]string{
		"":     DefaultCostWindow,
		"  ":   DefaultCostWindow,
		"1y":   DefaultCostWindow,
		"d1":   DefaultCostWindow,
		"0d":   DefaultCostWindow,
		"-1d":  DefaultCostWindow,
		"d":    DefaultCostWindow,
		"1D":   "1d",
		" 2w ": "2w",
		"7d":   "7d",
		"1m":   "1m",
		"all":  CostWindowAll,
		"ALL":  CostWindowAll,
	}
	for value, want := range cases {
		if got := CostWindow(value); got != want {
			t.Errorf("CostWindow(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestWindowStartTakesTheUTCBucket(t *testing.T) {
	// A Wednesday.
	now := time.Date(2026, 9, 16, 14, 30, 0, 0, time.UTC)
	cases := []struct {
		value string
		want  time.Time
	}{
		{"1d", time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)},
		{"7d", time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
		{"1w", time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)},
		{"2w", time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)},
		{"1m", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
		{"3m", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got, ok := WindowStart(c.value, now)
		if !ok {
			t.Fatalf("WindowStart(%q) reports no window", c.value)
		}
		if !got.Equal(c.want) {
			t.Errorf("WindowStart(%q) = %s, want %s", c.value, got, c.want)
		}
	}
}

func TestWindowStartOnASundayTakesTheMondayBefore(t *testing.T) {
	now := time.Date(2026, 9, 20, 23, 59, 0, 0, time.UTC)
	got, _ := WindowStart("1w", now)
	want := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("WindowStart on a Sunday = %s, want the Monday %s", got, want)
	}
}

func TestWindowStartRollsOverTheYear(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	day, _ := WindowStart("1d", now)
	if want := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC); !day.Equal(want) {
		t.Fatalf("the day = %s, want %s", day, want)
	}
	week, _ := WindowStart("2d", now)
	if want := time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC); !week.Equal(want) {
		t.Fatalf("two days = %s, want %s", week, want)
	}
	month, _ := WindowStart("2m", now)
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !month.Equal(want) {
		t.Fatalf("two months = %s, want %s", month, want)
	}
}

func TestWindowStartIgnoresTheZoneOfTheClock(t *testing.T) {
	zone := time.FixedZone("far", 13*60*60)
	now := time.Date(2026, 9, 16, 14, 30, 0, 0, time.UTC)
	utc, _ := WindowStart("1d", now)
	elsewhere, _ := WindowStart("1d", now.In(zone))
	if !utc.Equal(elsewhere) {
		t.Fatalf("the same instant gave %s in UTC and %s in %s", utc, elsewhere, zone)
	}
}

func TestWindowStartReportsNoWindowForAll(t *testing.T) {
	if _, ok := WindowStart("all", time.Now()); ok {
		t.Fatal("all must report no window")
	}
}

func TestAnAbsentSettingTakesTheDay(t *testing.T) {
	now := time.Date(2026, 9, 16, 0, 30, 0, 0, time.UTC)
	got, ok := WindowStart("", now)
	if !ok {
		t.Fatal("an absent setting must take a window")
	}
	if want := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("WindowStart(\"\") = %s, want %s", got, want)
	}
}
