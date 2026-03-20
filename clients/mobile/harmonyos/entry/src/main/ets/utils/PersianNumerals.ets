/**
 * Converts Western Arabic digits (0–9) to Eastern Arabic/Persian digits (۰–۹).
 * Mirrors pkg/i18n.ToPersianNumerals() and the iOS/Android equivalents.
 */
export class PersianNumerals {

  private static readonly MAP: Record<string, string> = {
    '0': '۰', '1': '۱', '2': '۲', '3': '۳', '4': '۴',
    '5': '۵', '6': '۶', '7': '۷', '8': '۸', '9': '۹',
  }

  static convert(input: string): string {
    return input.split('').map(ch => PersianNumerals.MAP[ch] ?? ch).join('')
  }

  static formatNumber(n: number): string {
    return PersianNumerals.convert(String(n))
  }
}
