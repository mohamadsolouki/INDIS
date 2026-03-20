import XCTest
@testable import IndisApp

final class PersianCalendarTests: XCTestCase {

    func testFormatISO() {
        // A known Gregorian date → expected Solar Hijri year prefix
        let result = PersianCalendar.formatISO("2026-03-20T00:00:00Z")
        // 2026-03-20 Gregorian ≈ 1404/12/29 Solar Hijri
        XCTAssertTrue(result.contains("۱۴۰۴") || result.contains("1404"),
                      "Expected year 1404 in Solar Hijri, got: \(result)")
    }

    func testCurrentYearIsReasonable() {
        let year = PersianCalendar.currentYear
        XCTAssertTrue(year >= 1400 && year < 1500, "Solar Hijri year should be in range 1400–1500")
    }
}
