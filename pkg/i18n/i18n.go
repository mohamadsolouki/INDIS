// Package i18n provides internationalisation and RTL support for INDIS.
//
// Persian is the PRIMARY language — all interfaces are designed RTL-first.
// All other language interfaces are derived from the Persian/RTL design (PRD §FR-006).
//
// Supported languages (launch):
//   - فارسی (Persian) — Primary, RTL
//   - English — Co-Primary, LTR
//   - کردی سورانی (Kurdish Sorani) — Tier 1, RTL
//   - کردی کرمانجی (Kurdish Kurmanji) — Tier 1, LTR
//   - آذربایجانی (Azerbaijani Turkish) — Tier 1, LTR/RTL
//   - عربی (Arabic) — Tier 1, RTL
//
// Localisation requirements:
//   - Solar Hijri (Shamsi) calendar default
//   - Persian numerals (۰۱۲۳۴۵۶۷۸۹) default
//   - Vazirmatn typography
//   - Persian alphabetical sort order
package i18n

import (
	"fmt"
	"strings"
	"time"
)

// Language represents a supported UI language.
type Language string

const (
	LangPersian    Language = "fa"
	LangEnglish    Language = "en"
	LangKurdSorani Language = "ckb"
	LangKurdKurm   Language = "kmr"
	LangAzeri      Language = "az"
	LangArabic     Language = "ar"
)

// Direction indicates text direction.
type Direction string

const (
	DirectionRTL Direction = "rtl"
	DirectionLTR Direction = "ltr"
)

// TextDirection returns the text direction for the given language.
func TextDirection(lang Language) Direction {
	switch lang {
	case LangPersian, LangKurdSorani, LangArabic:
		return DirectionRTL
	default:
		return DirectionLTR
	}
}

// SolarHijriDate represents a date in the Solar Hijri (Shamsi) calendar.
// Ref: https://en.wikipedia.org/wiki/Solar_Hijri_calendar
type SolarHijriDate struct {
	Year  int
	Month int // 1–12
	Day   int
}

// String formats the date as YYYY/MM/DD in Persian numerals.
func (d SolarHijriDate) String() string {
	return fmt.Sprintf("%s/%s/%s",
		ToPersianNumerals(d.Year),
		ToPersianNumerals(d.Month),
		ToPersianNumerals(d.Day),
	)
}

// StringLatin formats the date as YYYY/MM/DD in Latin numerals.
func (d SolarHijriDate) StringLatin() string {
	return fmt.Sprintf("%04d/%02d/%02d", d.Year, d.Month, d.Day)
}

// ToSolarHijri converts a Gregorian time.Time to Solar Hijri (Shamsi) date.
//
// Algorithm: Borkowski (1996) — "The Persian calendar for 3000 years"
// https://www.fourmilab.ch/documents/calendar/
func ToSolarHijri(t time.Time) SolarHijriDate {
	// Work in UTC to avoid timezone side-effects.
	t = t.UTC()
	y, m, d := t.Date()

	// Gregorian to Julian Day Number
	jdn := gregorianToJDN(y, int(m), d)

	// Julian Day Number to Solar Hijri
	return jdnToSolarHijri(jdn)
}

// gregorianToJDN converts a Gregorian date to Julian Day Number.
func gregorianToJDN(year, month, day int) int {
	a := (14 - month) / 12
	y := year + 4800 - a
	mo := month + 12*a - 3
	return day + (153*mo+2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
}

// jdnToSolarHijri converts a Julian Day Number to Solar Hijri date.
// Algorithm from https://www.fourmilab.ch/documents/calendar/ (Persian Calendar section).
func jdnToSolarHijri(jdn int) SolarHijriDate {
	// Solar Hijri epoch JDN: 1948320 (Nawruz 1 Farvardin 1 SH = March 22, 622 CE Julian)
	// Using the algorithmic calendar (Borkowski 1996).
	depoch := jdn - persianEpochJDN()
	cycle, cyear := divmod(depoch, 2820)
	if cyear < 0 {
		cycle--
		cyear += 2820
	}
	ycycle := 474 + 2820*cycle
	aux, _ := divmod(cyear, 474)
	if aux < 0 {
		aux = 0
	}
	year := cyear + aux*474 + ycycle - 474

	yday := jdn - solarHijriToJDN(year, 1, 1) + 1
	var month int
	if yday <= 186 {
		month = (yday-1)/31 + 1
	} else {
		month = (yday-7)/30 + 1
	}
	day := jdn - solarHijriToJDN(year, month, 1) + 1
	return SolarHijriDate{Year: year, Month: month, Day: day}
}

// persianEpochJDN returns the Julian Day Number of the Persian calendar epoch (1 Farvardin 1 SH).
func persianEpochJDN() int {
	return 1948320
}

// solarHijriToJDN converts a Solar Hijri date to Julian Day Number.
func solarHijriToJDN(year, month, day int) int {
	epbase := year - 474
	if year < 474 {
		epbase = year - 473
	}
	epyear := 474 + mod(epbase, 2820)
	var monthDays int
	if month <= 6 {
		monthDays = 31 * (month - 1)
	} else {
		monthDays = 30*(month-1) + 6
	}
	return day + monthDays +
		(epyear*682-110)/2816 +
		(epyear-1)*365 +
		epbase/2820*1029983 +
		persianEpochJDN() - 1
}

func divmod(a, b int) (int, int) {
	return a / b, a % b
}

func mod(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}

// persianDigits maps ASCII digits 0-9 to their Persian/Arabic-Indic equivalents.
var persianDigits = [10]rune{'۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'}

// ToPersianNumerals converts an integer to its Persian numeral string.
// Negative numbers are prefixed with a minus sign.
func ToPersianNumerals(n int) string {
	if n < 0 {
		return "-" + ToPersianNumerals(-n)
	}
	s := fmt.Sprintf("%d", n)
	var b strings.Builder
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(persianDigits[ch-'0'])
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

// FormatSolarHijriDate formats a time.Time as a Solar Hijri date string.
// If persian is true, uses Persian numerals; otherwise Latin.
func FormatSolarHijriDate(t time.Time, persian bool) string {
	d := ToSolarHijri(t)
	if persian {
		return d.String()
	}
	return d.StringLatin()
}
