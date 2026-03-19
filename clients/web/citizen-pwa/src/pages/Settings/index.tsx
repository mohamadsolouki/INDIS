import { useEffect, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import {
  SunIcon, MoonIcon, ComputerDesktopIcon,
  ArrowRightStartOnRectangleIcon, InformationCircleIcon,
  AdjustmentsHorizontalIcon,
} from '@heroicons/react/24/outline';
import LanguageSwitcher from '../../components/LanguageSwitcher/LanguageSwitcher';
import { useSettings } from '../../hooks/useSettings';
import { useAuthStore } from '../../auth/store';
import { cn } from '../../lib/cn';
import type { AppSettings } from '../../types';

export default function Settings() {
  const { t } = useTranslation();
  const {
    usePersianNumerals, useSolarHijri, fontSize, theme,
    setPersianNumerals, setSolarHijri, setFontSize, setTheme,
  } = useSettings();
  const { isAuthenticated, logout } = useAuthStore();

  // Apply font size to <html> element
  useEffect(() => {
    const html = document.documentElement;
    html.classList.remove('font-large', 'font-xlarge');
    if (fontSize === 'large') html.classList.add('font-large');
    if (fontSize === 'xlarge') html.classList.add('font-xlarge');
  }, [fontSize]);

  // Apply theme
  useEffect(() => {
    const html = document.documentElement;
    if (theme === 'dark') {
      html.classList.add('dark');
    } else if (theme === 'light') {
      html.classList.remove('dark');
    } else {
      // system
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      if (prefersDark) html.classList.add('dark'); else html.classList.remove('dark');
    }
  }, [theme]);

  return (
    <div className="max-w-lg mx-auto animate-fade-in">
      {/* Header */}
      <div className="px-4 py-5 bg-white border-b border-gray-100">
        <h1 className="text-xl font-bold text-gray-900">{t('settings.title')}</h1>
      </div>

      <div className="px-4 py-4 space-y-6">
        {/* Language */}
        <Section title={t('settings.language')} icon={<AdjustmentsHorizontalIcon className="w-5 h-5" />}>
          <LanguageSwitcher />
        </Section>

        {/* Localisation */}
        <Section title="محلی‌سازی" icon={<AdjustmentsHorizontalIcon className="w-5 h-5" />}>
          <Toggle
            label={t('settings.persian_numerals')}
            sublabel="۱۲۳ به جای 123"
            checked={usePersianNumerals}
            onChange={setPersianNumerals}
          />
          <Toggle
            label={t('settings.solar_hijri')}
            sublabel="تاریخ شمسی به جای میلادی"
            checked={useSolarHijri}
            onChange={setSolarHijri}
          />
        </Section>

        {/* Font size — FR-017.4: adjustable font sizes */}
        <Section title={t('settings.font_size')} icon={<AdjustmentsHorizontalIcon className="w-5 h-5" />}>
          <div className="flex gap-2">
            {(['normal', 'large', 'xlarge'] as const).map((size) => (
              <button
                key={size}
                type="button"
                onClick={() => setFontSize(size)}
                className={cn(
                  'flex-1 py-2.5 rounded-xl border text-sm font-medium transition-colors',
                  fontSize === size
                    ? 'bg-indis-primary text-white border-indis-primary'
                    : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300',
                )}
                style={{ fontSize: size === 'normal' ? 14 : size === 'large' ? 16 : 20 }}
              >
                {t(`settings.font_${size}`)}
              </button>
            ))}
          </div>
        </Section>

        {/* Theme */}
        <Section title={t('settings.theme')} icon={<SunIcon className="w-5 h-5" />}>
          <div className="flex gap-2">
            {([
              { value: 'light',  label: t('settings.theme_light'),  Icon: SunIcon },
              { value: 'dark',   label: t('settings.theme_dark'),   Icon: MoonIcon },
              { value: 'system', label: t('settings.theme_system'), Icon: ComputerDesktopIcon },
            ] as const).map(({ value, label, Icon }) => (
              <button
                key={value}
                type="button"
                onClick={() => setTheme(value as AppSettings['theme'])}
                className={cn(
                  'flex-1 flex flex-col items-center gap-1 py-3 rounded-xl border text-xs font-medium transition-colors',
                  theme === value
                    ? 'bg-indis-primary text-white border-indis-primary'
                    : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300',
                )}
              >
                <Icon className="w-5 h-5" />
                {label}
              </button>
            ))}
          </div>
        </Section>

        {/* About */}
        <Section title={t('settings.about')} icon={<InformationCircleIcon className="w-5 h-5" />}>
          <div className="bg-gray-50 rounded-xl p-4 space-y-2 text-sm text-gray-600">
            <div className="flex justify-between">
              <span>{t('settings.version')}</span>
              <span className="font-mono text-gray-800">1.0.0</span>
            </div>
            <div className="flex justify-between">
              <span>استاندارد</span>
              <span className="text-gray-500 text-xs">W3C DID Core 1.0 + VC 2.0</span>
            </div>
            <div className="flex justify-between">
              <span>رمزنگاری</span>
              <span className="text-gray-500 text-xs">Ed25519 + ZK-SNARK</span>
            </div>
          </div>
          <p className="text-xs text-gray-400 text-center mt-2">
            سامانه هویت دیجیتال ملی ایران — INDIS v1.0
          </p>
        </Section>

        {/* Logout */}
        {isAuthenticated && (
          <button
            type="button"
            onClick={logout}
            className="w-full flex items-center justify-center gap-2 border-2 border-red-200 text-red-600 rounded-xl py-3 font-medium hover:bg-red-50 transition-colors"
          >
            <ArrowRightStartOnRectangleIcon className="w-5 h-5" />
            {t('settings.logout')}
          </button>
        )}
      </div>
    </div>
  );
}

// Shared Section component

function Section({ title, icon, children }: { title: string; icon?: ReactNode; children: ReactNode }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 text-gray-700">
        {icon && <span className="text-gray-400">{icon}</span>}
        <h2 className="font-semibold text-sm">{title}</h2>
      </div>
      <div className="space-y-2">
        {children}
      </div>
    </div>
  );
}

// Toggle component

interface ToggleProps {
  label: string;
  sublabel?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}

function Toggle({ label, sublabel, checked, onChange }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className="w-full flex items-center justify-between bg-white rounded-xl border border-gray-100 px-4 py-3 hover:border-gray-200 transition-colors"
    >
      <div className="text-right rtl:text-right ltr:text-left">
        <p className="text-sm font-medium text-gray-800">{label}</p>
        {sublabel && <p className="text-xs text-gray-500 mt-0.5">{sublabel}</p>}
      </div>
      <div
        className={cn(
          'relative w-11 h-6 rounded-full transition-colors duration-200 flex-shrink-0',
          checked ? 'bg-indis-primary' : 'bg-gray-200',
        )}
      >
        <div
          className={cn(
            'absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all duration-200',
            checked ? 'start-5' : 'start-0.5',
          )}
        />
      </div>
    </button>
  );
}
