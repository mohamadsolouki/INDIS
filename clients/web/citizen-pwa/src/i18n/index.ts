import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import fa from './locales/fa.json';
import en from './locales/en.json';
import ckb from './locales/ckb.json';
import kmr from './locales/kmr.json';
import ar from './locales/ar.json';
import az from './locales/az.json';
import type { LocaleConfig, SupportedLocale, TextDirection } from '../types';

export const LOCALE_CONFIGS: Record<SupportedLocale, LocaleConfig> = {
  fa:  { code: 'fa',  name: 'Persian',         nativeName: 'فارسی',             dir: 'rtl' },
  en:  { code: 'en',  name: 'English',          nativeName: 'English',            dir: 'ltr' },
  ckb: { code: 'ckb', name: 'Kurdish Sorani',   nativeName: 'کوردی سۆرانی',      dir: 'rtl' },
  kmr: { code: 'kmr', name: 'Kurdish Kurmanji', nativeName: 'Kurdî Kurmancî',     dir: 'ltr' },
  ar:  { code: 'ar',  name: 'Arabic',           nativeName: 'عربی',               dir: 'rtl' },
  az:  { code: 'az',  name: 'Azerbaijani',      nativeName: 'آذربایجانجا',        dir: 'ltr' },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: { fa: { translation: fa }, en: { translation: en }, ckb: { translation: ckb }, kmr: { translation: kmr }, ar: { translation: ar }, az: { translation: az } },
    fallbackLng: 'fa',
    supportedLngs: ['fa', 'en', 'ckb', 'kmr', 'ar', 'az'],
    detection: { order: ['localStorage', 'navigator'], caches: ['localStorage'] },
    interpolation: { escapeValue: false },
  });

export function getDir(locale: string): TextDirection {
  return LOCALE_CONFIGS[locale as SupportedLocale]?.dir ?? 'rtl';
}

export function applyLocale(locale: string): void {
  const dir = getDir(locale);
  document.documentElement.lang = locale;
  document.documentElement.dir = dir;
}

export default i18n;
