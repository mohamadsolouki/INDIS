package org.indis.app.domain.i18n

import java.time.LocalDate

data class SolarHijriDate(val year: Int, val month: Int, val day: Int)

object PersianCalendar {
    // Baseline approximation for skeleton stage; replace with full pkg/i18n parity.
    fun toSolarHijri(date: LocalDate): SolarHijriDate {
        val year = date.year - 621
        return SolarHijriDate(year, date.monthValue, date.dayOfMonth)
    }
}
