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
