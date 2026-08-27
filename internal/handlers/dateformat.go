// Package handlers — date formatting helpers.
//
// Two strategies for blog post dates, used by the templates:
//
//   - FormatDate: long format localised on language.
//       EN: "July 22, 2026"
//       TH: "22 กรกฎาคม พ.ศ. 2569" (Buddhist Era = year + 543)
//
//   - RelativeDate: human-readable age in the active language.
//       EN: "3 weeks ago"
//       TH: "3 สัปดาห์ที่แล้ว"
package handlers

import (
	"strings"
	"time"
)

// thaiMonths are the 12 Gregorian months rendered in Thai.
var thaiMonths = [12]string{
	"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน",
	"พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม",
	"กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
}

// FormatDate returns the long, locale-specific date for the active
// language. English uses the standard Go format; Thai converts to
// the Buddhist Era (year + 543) and renders the month name in Thai.
func FormatDate(t time.Time, lang string) string {
	if lang == "th" {
		return formatThaiDate(t)
	}
	return t.Format("January 2, 2006")
}

// formatThaiDate renders "22 กรกฎาคม พ.ศ. 2569".
func formatThaiDate(t time.Time) string {
	day := t.Day()
	month := thaiMonths[t.Month()-1]
	yearBE := t.Year() + 543 // Buddhist Era
	return formatThaiNumber(day) + " " + month + " พ.ศ. " + formatThaiNumber(yearBE)
}

// formatThaiNumber renders a non-negative integer in Thai numerals
// (๐, ๑, ๒, …, ๙). Used for day and year in long Thai dates.
func formatThaiNumber(n int) string {
	if n == 0 {
		return "๐"
	}
	digits := []rune{'๐', '๑', '๒', '๓', '๔', '๕', '๖', '๗', '๘', '๙'}
	if n < 0 {
		return "-" + formatThaiNumber(-n)
	}
	out := ""
	for n > 0 {
		out = string(digits[n%10]) + out
		n /= 10
	}
	return out
}

// RelativeDate returns "3 weeks ago" / "3 สัปดาห์ที่แล้ว" — the
// largest unit that's >= 1 is used.
//
// Past times only — future times just return "just now".
func RelativeDate(t time.Time, lang string) string {
	now := time.Now()
	diff := now.Sub(t)
	if diff < time.Minute {
		if lang == "th" {
			return "เมื่อสักครู่"
		}
		return "just now"
	}
	if diff < 0 {
		if lang == "th" {
			return "เร็วๆ นี้"
		}
		return "soon"
	}

	n, unit := relativeUnit(diff, lang)
	if lang == "th" {
		return formatThaiNumber(n) + " " + unit + "ที่แล้ว"
	}
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return formatInt(n) + " " + unit + "s ago"
}

// relativeUnit picks the largest unit >= 1 second.
func relativeUnit(d time.Duration, lang string) (int, string) {
	seconds := int(d.Seconds())
	type unitEntry struct {
		secs int
		en   string
		th   string
	}
	units := []unitEntry{
		{60, "second", "วินาที"},
		{60, "minute", "นาที"},
		{24, "hour", "ชั่วโมง"},
		{7, "day", "วัน"},
		{4, "week", "สัปดาห์"},
		{12, "month", "เดือน"},
		{0, "year", "ปี"},
	}
	n := seconds
	for _, u := range units {
		if u.secs == 0 {
			break
		}
		if n < u.secs {
			unit := u.en
			if lang == "th" {
				unit = strings.ToLower(u.th)
			}
			return n, unit
		}
		n = n / u.secs
	}
	unit := "year"
	if lang == "th" {
		unit = "ปี"
	}
	return n, unit
}

// formatInt is a tiny helper (avoids pulling in fmt just for %d).
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt(-n)
	}
	digits := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	out := ""
	for n > 0 {
		out = string(digits[n%10]) + out
		n /= 10
	}
	return out
}
