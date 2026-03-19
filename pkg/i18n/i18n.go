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
// Uses the jalaali arithmetic algorithm (based on the 2820-year grand cycle).
// Note: the arithmetic algorithm can differ by one day from the astronomical
// Persian calendar at Nowruz. Specifically, Nowruz 1404 is astronomically on
// 2025-03-20 but this algorithm places it on 2025-03-21.
// For exact Nowruz-day calculations, use an astronomical equinox table.
func ToSolarHijri(t time.Time) SolarHijriDate {
	t = t.UTC()
	gy, gm, gd := t.Year(), int(t.Month()), t.Day()

	// Days elapsed since 1600-01-01 Gregorian (0-indexed).
	gy -= 1600
	gm -= 1
	gd -= 1

	gDayNo := 365*gy + (gy+3)/4 - (gy+99)/100 + (gy+399)/400
	for i := 0; i < gm; i++ {
		gDayNo += gMonthDays[i]
	}
	if gm > 1 && isGregorianLeap(gy+1600) {
		gDayNo++ // Feb 29
	}
	gDayNo += gd

	// Offset from Gregorian 1600-01-01 to Persian epoch (empirically derived).
	jDayNo := gDayNo - 79

	// 12053-day blocks (33-year sub-cycles within the 2820-year grand cycle).
	jnp := jDayNo / 12053
	jDayNo %= 12053

	jy := 979 + 33*jnp + 4*(jDayNo/1461)
	jDayNo %= 1461

	if jDayNo >= 366 {
		jy += (jDayNo - 1) / 365
		jDayNo = (jDayNo - 1) % 365
	}

	// Find month.
	i := 0
	for ; i < 11; i++ {
		jmi := jMonthDays[i]
		if jDayNo >= jmi {
			jDayNo -= jmi
		} else {
			break
		}
	}
	return SolarHijriDate{Year: jy, Month: i + 1, Day: jDayNo + 1}
}

// gMonthDays holds the number of days in each Gregorian month (non-leap year).
var gMonthDays = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// jMonthDays holds the number of days in Solar Hijri months 1–11.
// Month 12 is handled implicitly (29 or 30 days, leap-year dependent).
var jMonthDays = [11]int{31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30}

// isGregorianLeap reports whether a Gregorian year is a leap year.
func isGregorianLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
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
