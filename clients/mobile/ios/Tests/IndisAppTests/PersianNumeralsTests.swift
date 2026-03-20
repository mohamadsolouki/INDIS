import XCTest
@testable import IndisApp

final class PersianNumeralsTests: XCTestCase {

    func testDigitConversion() {
        XCTAssertEqual(PersianNumerals.convert("0123456789"), "۰۱۲۳۴۵۶۷۸۹")
    }

    func testMixedString() {
        XCTAssertEqual(PersianNumerals.convert("نسخه 1.0"), "نسخه ۱.۰")
    }

    func testNoChangeForNonDigits() {
        let input = "Hello World"
        XCTAssertEqual(PersianNumerals.convert(input), input)
    }

    func testFormatInt() {
        XCTAssertEqual(PersianNumerals.format(2026), "۲۰۲۶")
    }

    func testStringExtension() {
        XCTAssertEqual("99".toPersian(), "۹۹")
    }
}
