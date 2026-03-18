package org.indis.app.domain.i18n

object PersianNumerals {
    private val map = mapOf(
        '0' to '۰',
        '1' to '۱',
        '2' to '۲',
        '3' to '۳',
        '4' to '۴',
        '5' to '۵',
        '6' to '۶',
        '7' to '۷',
        '8' to '۸',
        '9' to '۹'
    )

    fun localize(input: String): String = buildString(input.length) {
        input.forEach { append(map[it] ?: it) }
    }
}
