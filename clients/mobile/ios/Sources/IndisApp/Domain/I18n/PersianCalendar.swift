import Foundation

/// Solar Hijri (Jalali) calendar utilities for the INDIS iOS app.
///
/// Wraps Foundation's `Calendar(identifier: .persian)` — available natively on iOS 16+.
/// For iOS 14/15 the calendar is still supported; Persian era names may differ slightly.
///
/// Mirrors `pkg/i18n` Solar Hijri helpers from the backend.
struct PersianCalendar {

    private static let calendar: Calendar = {
        var cal = Calendar(identifier: .persian)
        cal.locale = Locale(identifier: "fa_IR")
        return cal
    }()

    private static let formatter: DateFormatter = {
        let fmt = DateFormatter()
        fmt.calendar = Calendar(identifier: .persian)
        fmt.locale = Locale(identifier: "fa_IR")
        fmt.dateStyle = .medium
        fmt.timeStyle = .none
        return fmt
    }()

    private static let fullFormatter: DateFormatter = {
        let fmt = DateFormatter()
        fmt.calendar = Calendar(identifier: .persian)
        fmt.locale = Locale(identifier: "fa_IR")
        fmt.dateFormat = "yyyy/MM/dd"
        return fmt
    }()

    /// Formats a Date as a Persian (Solar Hijri) date string in the locale's medium style.
    static func format(_ date: Date) -> String {
        formatter.string(from: date)
    }

    /// Formats a Date as `yyyy/MM/dd` in Solar Hijri.
    static func formatShort(_ date: Date) -> String {
        fullFormatter.string(from: date)
    }

    /// Parses an ISO 8601 string and returns a formatted Solar Hijri date.
    static func formatISO(_ iso8601: String) -> String {
        guard let date = ISO8601DateFormatter().date(from: iso8601) else { return iso8601 }
        return format(date)
    }

    /// Returns the current Solar Hijri year.
    static var currentYear: Int {
        calendar.component(.year, from: Date())
    }
}
