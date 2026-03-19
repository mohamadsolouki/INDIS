import { toPersianNumerals } from './persianNumerals';

export interface SolarHijriDate {
  year: number;
  month: number;
  day: number;
}

const G_MONTH_DAYS = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
const J_MONTH_DAYS = [31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30];

function isGregorianLeap(year: number): boolean {
  return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
}

export function toSolarHijri(date: Date): SolarHijriDate {
  // Work in UTC
  let gy = date.getUTCFullYear();
  let gm = date.getUTCMonth() + 1; // 1-based
  let gd = date.getUTCDate();

  gy -= 1600;
  gm -= 1;
  gd -= 1;

  let gDayNo = 365 * gy + Math.floor((gy + 3) / 4) - Math.floor((gy + 99) / 100) + Math.floor((gy + 399) / 400);
  for (let i = 0; i < gm; i++) {
    gDayNo += G_MONTH_DAYS[i];
  }
  if (gm > 1 && isGregorianLeap(gy + 1600)) {
    gDayNo++;
  }
  gDayNo += gd;

  let jDayNo = gDayNo - 79;

  const jnp = Math.floor(jDayNo / 12053);
  jDayNo = jDayNo % 12053;

  let jy = 979 + 33 * jnp + 4 * Math.floor(jDayNo / 1461);
  jDayNo = jDayNo % 1461;

  if (jDayNo >= 366) {
    jy += Math.floor((jDayNo - 1) / 365);
    jDayNo = (jDayNo - 1) % 365;
  }

  let i = 0;
  for (; i < 11; i++) {
    const jmi = J_MONTH_DAYS[i];
    if (jDayNo >= jmi) {
      jDayNo -= jmi;
    } else {
      break;
    }
  }

  return { year: jy, month: i + 1, day: jDayNo + 1 };
}

export function formatSolarHijri(date: Date, persian = true): string {
  const { year, month, day } = toSolarHijri(date);
  const y = String(year).padStart(4, '0');
  const m = String(month).padStart(2, '0');
  const d = String(day).padStart(2, '0');
  if (!persian) return `${y}/${m}/${d}`;
  return `${toPersianNumerals(year)}/${toPersianNumerals(month)}/${toPersianNumerals(day)}`;
}

const PERSIAN_MONTH_NAMES = [
  'فروردین', 'اردیبهشت', 'خرداد', 'تیر', 'مرداد', 'شهریور',
  'مهر', 'آبان', 'آذر', 'دی', 'بهمن', 'اسفند',
];

export function formatSolarHijriLong(date: Date): string {
  const { year, month, day } = toSolarHijri(date);
  return `${toPersianNumerals(day)} ${PERSIAN_MONTH_NAMES[month - 1]} ${toPersianNumerals(year)}`;
}
