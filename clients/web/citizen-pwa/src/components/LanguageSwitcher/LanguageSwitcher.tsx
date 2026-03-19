import { useTranslation } from 'react-i18next';
import { CheckIcon, LanguageIcon } from '@heroicons/react/24/outline';
import { LOCALE_CONFIGS, applyLocale } from '../../i18n';
import { useSettings } from '../../hooks/useSettings';
import { cn } from '../../lib/cn';
import type { SupportedLocale } from '../../types';

interface Props {
  compact?: boolean;
}

export default function LanguageSwitcher({ compact = false }: Props) {
  const { i18n } = useTranslation();
  const setLocale = useSettings((s) => s.setLocale);

  const current = i18n.language as SupportedLocale;

  const handleChange = async (locale: SupportedLocale) => {
    await i18n.changeLanguage(locale);
    applyLocale(locale);
    setLocale(locale);
  };

  if (compact) {
    return (
      <div className="relative inline-block">
        <select
          value={current}
          onChange={(e) => void handleChange(e.target.value as SupportedLocale)}
          className="appearance-none bg-transparent pr-6 rtl:pr-0 rtl:pl-6 text-sm font-medium text-indis-primary focus:outline-none cursor-pointer"
          aria-label="Select language"
        >
          {Object.values(LOCALE_CONFIGS).map(({ code, nativeName }) => (
            <option key={code} value={code}>{nativeName}</option>
          ))}
        </select>
        <LanguageIcon className="w-4 h-4 text-indis-primary absolute start-0 top-0.5 pointer-events-none" />
      </div>
    );
  }

  return (
    <div className="space-y-1" role="radiogroup" aria-label="Language selection">
      {Object.values(LOCALE_CONFIGS).map(({ code, name, nativeName, dir }) => {
        const isSelected = current === code;
        return (
          <button
            key={code}
            type="button"
            role="radio"
            aria-checked={isSelected}
            onClick={() => void handleChange(code)}
            className={cn(
              'w-full flex items-center justify-between px-4 py-3 rounded-xl border transition-all',
              isSelected
                ? 'border-indis-primary bg-indis-primary/5 text-indis-primary'
                : 'border-gray-100 bg-white text-gray-700 hover:border-gray-200',
            )}
          >
            <div className="flex items-center gap-3">
              <div className="text-xs text-gray-400 uppercase font-mono w-6">{code}</div>
              <div className="text-right rtl:text-right ltr:text-left">
                <p className="font-medium text-sm" dir={dir}>{nativeName}</p>
                <p className="text-xs text-gray-400">{name}</p>
              </div>
            </div>
            {isSelected && <CheckIcon className="w-5 h-5 text-indis-primary flex-shrink-0" />}
          </button>
        );
      })}
    </div>
  );
}
