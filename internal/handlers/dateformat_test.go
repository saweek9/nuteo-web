package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDate_English(t *testing.T) {
	got := FormatDate(time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), "en")
	want := "July 22, 2026"
	if got != want {
		t.Errorf("EN: got %q, want %q", got, want)
	}
}

func TestFormatDate_ThaiBuddhistEra(t *testing.T) {
	got := FormatDate(time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), "th")
	// Must contain Thai month name, year 2569 (= 2026 + 543),
	// day in Thai numerals ("๒๒").
	for _, want := range []string{"กรกฎาคม", "พ.ศ.", "๒๕๖๙", "๒๒"} {
		if !strings.Contains(got, want) {
			t.Errorf("TH %q missing %q", got, want)
		}
	}
}

func TestFormatDate_January2027TH(t *testing.T) {
	// 2027-01-01 → Buddhist Era 2570 → "๑ มกราคม พ.ศ. ๒๕๗๐"
	got := FormatDate(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), "th")
	for _, want := range []string{"มกราคม", "๒๕๗๐", "๑"} {
		if !strings.Contains(got, want) {
			t.Errorf("Jan 2027 TH %q missing %q", got, want)
		}
	}
}

func TestFormatThaiNumber(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "๐"},
		{1, "๑"},
		{9, "๙"},
		{10, "๑๐"},
		{2026, "๒๐๒๖"},
		{2569, "๒๕๖๙"},
	}
	for _, c := range cases {
		if got := formatThaiNumber(c.in); got != c.want {
			t.Errorf("formatThaiNumber(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRelativeDate_JustNow(t *testing.T) {
	got := RelativeDate(time.Now().Add(-30*time.Second), "en")
	// Less than a minute → "just now" (could be a few seconds ago)
	if got != "just now" {
		t.Errorf("30s ago EN: got %q, want 'just now'", got)
	}
}

func TestRelativeDate_ThreeWeeks(t *testing.T) {
	got := RelativeDate(time.Now().Add(-21*24*time.Hour), "en")
	if got != "3 weeks ago" {
		t.Errorf("21 days ago EN: got %q, want '3 weeks ago'", got)
	}
}

func TestRelativeDate_OneWeek_TH(t *testing.T) {
	got := RelativeDate(time.Now().Add(-8*24*time.Hour), "th")
	// 8 days ≈ 1 week, "1 สัปดาห์ที่แล้ว"
	if got != "๑ สัปดาห์ที่แล้ว" {
		t.Errorf("8 days TH: got %q, want '๑ สัปดาห์ที่แล้ว'", got)
	}
}

func TestRelativeDate_OneDay_EN(t *testing.T) {
	got := RelativeDate(time.Now().Add(-26*time.Hour), "en")
	// 26 hours ≈ 1 day, "1 day ago"
	if got != "1 day ago" {
		t.Errorf("26h ago EN: got %q, want '1 day ago'", got)
	}
}
