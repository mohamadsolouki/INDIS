import Foundation

/// Converts Western Arabic digits (0–9) to Eastern Arabic/Persian digits (۰–۹).
///
/// Used throughout the UI when `AppState.usePersianNumerals` is true.
/// Mirrors `pkg/i18n.ToPersianNumerals()` from the backend.
struct PersianNumerals {

    private static let map: [Character: Character] = [
        "0": "۰", "1": "۱", "2": "۲", "3": "۳", "4": "۴",
        "5": "۵", "6": "۶", "7": "۷", "8": "۸", "9": "۹",
    ]

    /// Converts all ASCII digits in the string to their Persian equivalents.
    static func convert(_ input: String) -> String {
        String(input.map { map[$0] ?? $0 })
    }

    /// Formats an integer using Persian numerals.
    static func format(_ number: Int) -> String {
        convert(String(number))
    }

    /// Formats a Double with the given number of decimal places using Persian numerals.
    static func format(_ number: Double, decimals: Int = 0) -> String {
        convert(String(format: "%.\(decimals)f", number))
    }
}

extension String {
    /// Returns this string with ASCII digits replaced by Persian numerals.
    func toPersian() -> String { PersianNumerals.convert(self) }
}
