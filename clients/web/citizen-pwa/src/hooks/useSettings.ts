import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { AppSettings, SupportedLocale } from '../types';

const DEFAULTS: AppSettings = {
  locale: 'fa',
  usePersianNumerals: true,
  useSolarHijri: true,
  fontSize: 'normal',
  theme: 'system',
};

interface SettingsStore extends AppSettings {
  setLocale: (locale: SupportedLocale) => void;
  setPersianNumerals: (v: boolean) => void;
  setSolarHijri: (v: boolean) => void;
  setFontSize: (v: AppSettings['fontSize']) => void;
  setTheme: (v: AppSettings['theme']) => void;
  reset: () => void;
}

export const useSettings = create<SettingsStore>()(
  persist(
    (set) => ({
      ...DEFAULTS,
      setLocale:          (locale)             => set({ locale }),
      setPersianNumerals: (usePersianNumerals) => set({ usePersianNumerals }),
      setSolarHijri:      (useSolarHijri)      => set({ useSolarHijri }),
      setFontSize:        (fontSize)           => set({ fontSize }),
      setTheme:           (theme)              => set({ theme }),
      reset:              ()                   => set(DEFAULTS),
    }),
    { name: 'indis_settings' },
  ),
);
