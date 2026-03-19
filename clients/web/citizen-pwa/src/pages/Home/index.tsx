import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { PlusCircleIcon, ArrowPathIcon } from '@heroicons/react/24/outline';
import IdentityCard from '../../components/IdentityCard/IdentityCard';
import { useCard } from '../../hooks/useCard';
import { useAuthStore } from '../../auth/store';
import { cn } from '../../lib/cn';

/**
 * Home page — primary landing view for authenticated citizens.
 *
 * Displays the FR-007 IdentityCard along with quick-action links to Wallet
 * and Privacy settings. When the user is not yet enrolled, a call-to-action
 * banner links to the Enrollment wizard.
 */
export default function Home() {
  const { t } = useTranslation();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const { cardData, loading, error, refresh, generate } = useCard();

  /**
   * Derive active credential types from the card presence.
   * In a full implementation these come from the wallet/credential API.
   */
  const activeCredentials: string[] = cardData
    ? [
        'CitizenshipCredential',
        'VoterEligibilityCredential',
        'HealthInsuranceCredential',
      ]
    : [];

  return (
    <div
      className="max-w-lg mx-auto px-4 py-6 space-y-6 animate-fade-in"
      dir="rtl"
    >
      {/* ── Page title ─────────────────────────────────────────────────────── */}
      <div>
        <h1 className="text-xl font-bold text-gray-900">{t('home.title')}</h1>
        <p className="text-sm text-gray-500 mt-0.5">{t('app.tagline')}</p>
      </div>

      {/* ── Error banner ───────────────────────────────────────────────────── */}
      {error && (
        <div
          className="bg-red-50 border border-red-200 rounded-lg px-4 py-3 flex items-center justify-between gap-3"
          role="alert"
        >
          <span className="text-red-700 text-sm">{error}</span>
          <button
            type="button"
            onClick={() => void refresh()}
            className="text-red-600 hover:text-red-800 flex items-center gap-1 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500 rounded"
          >
            <ArrowPathIcon className="w-4 h-4" aria-hidden="true" />
            {t('common.retry')}
          </button>
        </div>
      )}

      {/* ── Identity Card (FR-007) ──────────────────────────────────────────── */}
      <IdentityCard
        card={cardData}
        credentials={activeCredentials}
        loading={loading}
      />

      {/* ── Actions ────────────────────────────────────────────────────────── */}
      {isAuthenticated && (
        <div className="flex gap-3">
          {!cardData && !loading && (
            <button
              type="button"
              onClick={() => void generate()}
              className="flex-1 flex items-center justify-center gap-2 bg-indis-primary text-white rounded-xl px-4 py-3 font-medium hover:bg-indis-primary-dark transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary"
            >
              <PlusCircleIcon className="w-5 h-5" aria-hidden="true" />
              {t('home.generate_card')}
            </button>
          )}
          {cardData && (
            <button
              type="button"
              onClick={() => void refresh()}
              className={cn(
                'flex items-center gap-2 text-indis-primary rounded-xl px-4 py-2 text-sm hover:bg-indis-primary/5 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary',
              )}
              aria-label={t('common.retry')}
            >
              <ArrowPathIcon
                className={cn('w-4 h-4', loading && 'animate-spin')}
                aria-hidden="true"
              />
              {t('common.retry')}
            </button>
          )}
        </div>
      )}

      {/* ── Not-enrolled / unauthenticated CTA ─────────────────────────────── */}
      {!isAuthenticated && !loading && (
        <div className="bg-indis-primary/5 rounded-2xl p-6 text-center space-y-3">
          <p className="text-gray-700 font-medium">{t('enrollment.subtitle')}</p>
          <Link
            to="/enrollment"
            className="inline-block bg-indis-primary text-white rounded-xl px-6 py-3 font-medium hover:bg-indis-primary-dark transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary"
          >
            {t('enrollment.title')}
          </Link>
        </div>
      )}

      {/* ── Quick links grid ───────────────────────────────────────────────── */}
      <nav aria-label={t('nav.quick_links')}>
        <div className="grid grid-cols-2 gap-3">
          <Link
            to="/wallet"
            className="bg-white rounded-xl p-4 shadow-sm border border-gray-100 hover:shadow-md transition-shadow focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary"
          >
            <p className="font-semibold text-gray-800 text-sm">{t('nav.wallet')}</p>
            <p className="text-gray-500 text-xs mt-1">{t('wallet.title')}</p>
          </Link>
          <Link
            to="/privacy"
            className="bg-white rounded-xl p-4 shadow-sm border border-gray-100 hover:shadow-md transition-shadow focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary"
          >
            <p className="font-semibold text-gray-800 text-sm">{t('nav.privacy')}</p>
            <p className="text-gray-500 text-xs mt-1">{t('privacy.title')}</p>
          </Link>
        </div>
      </nav>
    </div>
  );
}
