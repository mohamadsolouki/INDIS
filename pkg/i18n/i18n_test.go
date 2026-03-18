package i18n

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ToSolarHijri
// ---------------------------------------------------------------------------

func TestToSolarHijri_KnownDates(t *testing.T) {
	// Known Gregorian → Solar Hijri conversion vectors.
	//
	// 2026-03-18 → 1404/12/27  (27 Esfand 1404)
	// 2025-03-20 → 1404/01/01  (Nowruz, 1 Farvardin 1404)
	// 2024-01-01 → 1402/10/11  (11 Dey 1402)
	tests := []struct {
		name    string
		greg    time.Time
		wantY   int
		wantM   int
		wantD   int
	}{
		{
			name:  "2026-03-18 → 1404/12/27",
			greg:  time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC),
			wantY: 1404, wantM: 12, wantD: 27,
		},
		{
			name:  "2025-03-20 → 1404/01/01 (Nowruz)",
			greg:  time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
			wantY: 1404, wantM: 1, wantD: 1,
		},
		{
			name:  "2024-01-01 → 1402/10/11",
			greg:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantY: 1402, wantM: 10, wantD: 11,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ToSolarHijri(tc.greg)
			if got.Year != tc.wantY || got.Month != tc.wantM || got.Day != tc.wantD {
				t.Errorf("ToSolarHijri(%v) = %04d/%02d/%02d, want %04d/%02d/%02d",
					tc.greg.Format("2006-01-02"),
					got.Year, got.Month, got.Day,
					tc.wantY, tc.wantM, tc.wantD)
			}
		})
	}
}

func TestToSolarHijri_StructuralConstraints(t *testing.T) {
	// Month must be 1–12; day must be 1–31 for any valid modern Gregorian date.
	dates := []time.Time{
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2010, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC),
	}
	for _, d := range dates {
		got := ToSolarHijri(d)
		if got.Month < 1 || got.Month > 12 {
			t.Errorf("ToSolarHijri(%v).Month = %d, want 1–12", d.Format("2006-01-02"), got.Month)
		}
		if got.Day < 1 || got.Day > 31 {
			t.Errorf("ToSolarHijri(%v).Day = %d, want 1–31", d.Format("2006-01-02"), got.Day)
		}
		if got.Year < 1370 || got.Year > 1410 {
			t.Errorf("ToSolarHijri(%v).Year = %d, outside expected modern range", d.Format("2006-01-02"), got.Year)
		}
	}
}

func TestToSolarHijri_TimezoneSideEffect(t *testing.T) {
	// Same Gregorian date in UTC and in a +5:30 zone should both produce the same
	// Solar Hijri date when the underlying UTC date is the same.
	utc := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	tehran := time.Date(2026, 3, 18, 12, 0, 0, 0, time.FixedZone("IRST", 3*3600+30*60))
	gotUTC := ToSolarHijri(utc)
	gotTehran := ToSolarHijri(tehran)
	if gotUTC != gotTehran {
		t.Errorf("ToSolarHijri different for UTC vs Tehran zone on same nominal date: %v vs %v", gotUTC, gotTehran)
	}
}

// ---------------------------------------------------------------------------
// FormatSolarHijriDate
// ---------------------------------------------------------------------------

func TestFormatSolarHijriDate_PersianFalse_ASCIIOnly(t *testing.T) {
	t.Parallel()
	d := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	s := FormatSolarHijriDate(d, false)
	for _, r := range s {
		if r > 127 {
			t.Errorf("FormatSolarHijriDate(persian=false) contains non-ASCII rune %U in %q", r, s)
		}
	}
	// Must contain "/" separators
	if !strings.Contains(s, "/") {
		t.Errorf("FormatSolarHijriDate(persian=false) = %q, expected slash-separated date", s)
	}
}

func TestFormatSolarHijriDate_PersianTrue_ContainsPersianDigits(t *testing.T) {
	t.Parallel()
	d := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	s := FormatSolarHijriDate(d, true)
	// Must contain at least one Persian digit rune (U+06F0 – U+06F9)
	hasPersian := false
	for _, r := range s {
		if r >= '۰' && r <= '۹' {
			hasPersian = true
			break
		}
	}
	if !hasPersian {
		t.Errorf("FormatSolarHijriDate(persian=true) = %q, expected Persian digit runes", s)
	}
}

func TestFormatSolarHijriDate_KnownOutput(t *testing.T) {
	// 2026-03-18 → 1404/12/27; latin format = "1404/12/27"
	d := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	got := FormatSolarHijriDate(d, false)
	if got != "1404/12/27" {
		t.Errorf("FormatSolarHijriDate(2026-03-18, false) = %q, want \"1404/12/27\"", got)
	}
}

// ---------------------------------------------------------------------------
// ToPersianNumerals
// ---------------------------------------------------------------------------

func TestToPersianNumerals_TableDriven(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "۰"},
		{1, "۱"},
		{9, "۹"},
		{10, "۱۰"},
		{1234567890, "۱۲۳۴۵۶۷۸۹۰"},
		{-1, "-۱"},
		{-42, "-۴۲"},
	}
	for _, tc := range tests {
		got := ToPersianNumerals(tc.input)
		if got != tc.want {
			t.Errorf("ToPersianNumerals(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestToPersianNumerals_NegativeKeepsMinusSign(t *testing.T) {
	got := ToPersianNumerals(-123)
	if !strings.HasPrefix(got, "-") {
		t.Errorf("ToPersianNumerals(-123) = %q, should start with '-'", got)
	}
}

func TestToPersianNumerals_AllDigitsMapped(t *testing.T) {
	persianZero := '۰'
	for i := 0; i <= 9; i++ {
		got := []rune(ToPersianNumerals(i))
		if len(got) != 1 {
			t.Errorf("ToPersianNumerals(%d) has %d runes, want 1", i, len(got))
			continue
		}
		if got[0] != rune(persianZero)+rune(i) {
			t.Errorf("ToPersianNumerals(%d) = %q (U+%04X), want U+%04X",
				i, string(got[0]), got[0], rune(persianZero)+rune(i))
		}
	}
}

// ---------------------------------------------------------------------------
// TextDirection
// ---------------------------------------------------------------------------

func TestTextDirection_TableDriven(t *testing.T) {
	tests := []struct {
		lang Language
		want Direction
	}{
		{LangPersian, DirectionRTL},
		{LangEnglish, DirectionLTR},
		{LangArabic, DirectionRTL},
		{LangKurdSorani, DirectionRTL},
		{LangKurdKurm, DirectionLTR},
		{LangAzeri, DirectionLTR},
	}
	for _, tc := range tests {
		got := TextDirection(tc.lang)
		if got != tc.want {
			t.Errorf("TextDirection(%q) = %q, want %q", tc.lang, got, tc.want)
		}
	}
}

func TestTextDirection_UnknownLanguage_DefaultsLTR(t *testing.T) {
	got := TextDirection("xx")
	if got != DirectionLTR {
		t.Errorf("TextDirection(unknown) = %q, want %q", got, DirectionLTR)
	}
}

// ---------------------------------------------------------------------------
// SolarHijriDate.String / StringLatin
// ---------------------------------------------------------------------------

func TestSolarHijriDate_StringLatin_Format(t *testing.T) {
	d := SolarHijriDate{Year: 1404, Month: 1, Day: 1}
	got := d.StringLatin()
	if got != "1404/01/01" {
		t.Errorf("StringLatin() = %q, want \"1404/01/01\"", got)
	}
}

func TestSolarHijriDate_String_PersianNumerals(t *testing.T) {
	d := SolarHijriDate{Year: 1404, Month: 1, Day: 1}
	got := d.String()
	// Must contain Persian numerals, not ASCII digits for year/month/day.
	for _, r := range got {
		if r >= '0' && r <= '9' {
			t.Errorf("String() = %q contains ASCII digit rune %q", got, r)
		}
	}
}
