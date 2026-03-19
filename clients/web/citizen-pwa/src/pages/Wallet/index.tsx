import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { WalletIcon, ArrowPathIcon, FunnelIcon } from '@heroicons/react/24/outline';
import CredentialCard from '../../components/CredentialCard/CredentialCard';
import { useWallet } from '../../hooks/useWallet';
import { cn } from '../../lib/cn';
import type { CredentialType } from '../../types';

const FILTER_OPTIONS: { value: CredentialType | 'all'; label: string }[] = [
  { value: 'all',                        label: 'همه' },
  { value: 'CitizenshipCredential',      label: 'شهروندی' },
  { value: 'VoterEligibilityCredential', label: 'رأی‌دهنده' },
  { value: 'HealthInsuranceCredential',  label: 'بیمه سلامت' },
  { value: 'AgeRangeCredential',         label: 'سنی' },
  { value: 'ResidencyCredential',        label: 'اقامت' },
];

export default function Wallet() {
  const { t } = useTranslation();
  const [filter, setFilter] = useState<CredentialType | 'all'>('all');
  const { credentials, loading, error, reload } = useWallet(filter === 'all' ? undefined : filter);

  return (
    <div className="max-w-lg mx-auto animate-fade-in">
      {/* Header */}
      <div className="px-4 py-5 bg-white border-b border-gray-100">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold text-gray-900">{t('wallet.title')}</h1>
          <button
            type="button"
            onClick={() => void reload()}
            className="text-gray-400 hover:text-indis-primary p-1 rounded-lg transition-colors"
            aria-label={t('common.retry')}
          >
            <ArrowPathIcon className={cn('w-5 h-5', loading && 'animate-spin')} />
          </button>
        </div>
      </div>

      {/* Filter chips */}
      <div className="bg-white border-b border-gray-100 px-4 py-2 overflow-x-auto hide-scrollbar">
        <div className="flex gap-2 min-w-max">
          <FunnelIcon className="w-4 h-4 text-gray-400 flex-shrink-0 self-center" />
          {FILTER_OPTIONS.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              onClick={() => setFilter(value)}
              className={cn(
                'px-3 py-1 rounded-full text-xs font-medium border transition-colors whitespace-nowrap',
                filter === value
                  ? 'bg-indis-primary text-white border-indis-primary'
                  : 'bg-white text-gray-600 border-gray-200 hover:border-gray-300',
              )}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      <div className="px-4 py-4">
        {/* Error */}
        {error && (
          <div className="mb-4 bg-red-50 border border-red-200 rounded-lg px-4 py-3">
            <p className="text-red-700 text-sm">{error}</p>
          </div>
        )}

        {/* Loading */}
        {loading && credentials.length === 0 && (
          <div className="flex justify-center py-12">
            <div className="w-8 h-8 border-2 border-indis-primary border-t-transparent rounded-full animate-spin" />
          </div>
        )}

        {/* Empty state */}
        {!loading && credentials.length === 0 && (
          <div className="flex flex-col items-center py-16 gap-3 text-center">
            <WalletIcon className="w-16 h-16 text-gray-300" />
            <p className="text-gray-500">{t('wallet.empty')}</p>
            <p className="text-gray-400 text-xs">
              پس از ثبت‌نام، مدارک دیجیتال به‌طور خودکار اضافه می‌شوند
            </p>
          </div>
        )}

        {/* Credentials grid */}
        {credentials.length > 0 && (
          <div className="space-y-3">
            <p className="text-xs text-gray-500">{credentials.length} مدرک</p>
            {credentials.map((cred) => (
              <CredentialCard key={cred.id} credential={cred} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
