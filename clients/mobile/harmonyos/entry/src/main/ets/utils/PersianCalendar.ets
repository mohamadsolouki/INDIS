/**
 * Solar Hijri (Jalali) calendar utilities for HarmonyOS.
 *
 * HarmonyOS / ArkTS does not have a built-in Persian calendar formatter,
 * so we implement the Gregorian-to-Solar-Hijri conversion algorithm directly.
 * This mirrors pkg/i18n Solar Hijri helpers and the iOS/Android equivalents.
 */
export class PersianCalendar {

  /**
   * Converts a Gregorian Date to a Solar Hijri date string (YYYY/MM/DD).
   */
  static format(date: Date): string {
    const [y, m, d] = PersianCalendar.toSolarHijri(
      date.getFullYear(), date.getMonth() + 1, date.getDate()
    )
    return `${y}/${String(m).padStart(2, '0')}/${String(d).padStart(2, '0')}`
  }

  /**
   * Parses an ISO 8601 string and returns a formatted Solar Hijri date.
   */
  static formatISO(iso8601: string): string {
    if (!iso8601) return ''
    try {
      return PersianCalendar.format(new Date(iso8601))
    } catch {
      return iso8601
    }
  }

  /**
   * Returns the current Solar Hijri year.
   */
  static currentYear(): number {
    return PersianCalendar.toSolarHijri(
      new Date().getFullYear(), new Date().getMonth() + 1, new Date().getDate()
    )[0]
  }

  // ── Conversion algorithm ──────────────────────────────────────────────────
  // Based on the standard Gregorian ↔ Jalali conversion algorithm.

  private static toSolarHijri(gy: number, gm: number, gd: number): [number, number, number] {
    const g_d_no = 365 * gy + PersianCalendar.div(gy + 3, 4) - PersianCalendar.div(gy + 99, 100)
      + PersianCalendar.div(gy + 399, 400)
    let offset = [0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334]
    let jy = gy - 1600
    let jd_no = 365 * jy + PersianCalendar.div(jy, 4) - PersianCalendar.div(jy, 100)
      + PersianCalendar.div(jy, 400)
    let j_d_no2 = jd_no + 79 + 365 * 1595 + 388 + 97
    let g_d_no2 = 365 * 1600 + 388 + 97 + 79
    let i = gy - 621
    let j1p = i - PersianCalendar.div(i, 33) * 33
    let jl = j1p + (PersianCalendar.div(j1p, 4) + 1) * 8
    let leap = jl % 4 === 0 ? 1 : 0
    let g_d_no3 = g_d_no + offset[gm - 1] + gd - 1
    if (gm > 2) g_d_no3 += (gy % 4 === 0 && (gy % 100 !== 0 || gy % 400 === 0)) ? 1 : 0
    const jn = g_d_no3 - (365 * (gy - 622) + PersianCalendar.div(gy - 621 + 3, 4)
      - PersianCalendar.div(gy - 621 + 99, 100) + PersianCalendar.div(gy - 621 + 399, 400) + 20)
    let jy2 = gy - 622
    let days = jn
    if (days < 0) { jy2--; days += 365 + leap }
    let jm = 0
    const jml = [31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30, 29 + leap]
    for (jm = 0; jm < 12; jm++) {
      if (days < jml[jm]) break
      days -= jml[jm]
    }
    return [jy2, jm + 1, days + 1]
  }

  private static div(a: number, b: number): number {
    return Math.floor(a / b)
  }
}
