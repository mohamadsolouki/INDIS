import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  QrCodeIcon,
  EyeIcon,
  EyeSlashIcon,
  CheckBadgeIcon,
  ShieldCheckIcon,
} from '@heroicons/react/24/outline';
import { cn } from '../../lib/cn';
import type { DigitalCard } from '../../types';

interface IdentityCardProps {
  card: DigitalCard | null;
  credentials: string[];
  loading?: boolean;
  onQRReveal?: () => void;
}

const CREDENTIAL_BADGES: { type: string; labelKey: string; color: string }[] = [
  { type: 'CitizenshipCredential',      labelKey: 'home.citizen',    color: 'bg-green-100 text-green-800' },
  { type: 'VoterEligibilityCredential', labelKey: 'home.voter',      color: 'bg-blue-100 text-blue-800' },
  { type: 'HealthInsuranceCredential',  labelKey: 'home.health',     color: 'bg-pink-100 text-pink-800' },
  { type: 'EmploymentCredential',       labelKey: 'home.employment', color: 'bg-yellow-100 text-yellow-800' },
  { type: 'ResidencyCredential',        labelKey: 'home.residence',  color: 'bg-purple-100 text-purple-800' },
  { type: 'MilitaryServiceCredential',  labelKey: 'home.military',   color: 'bg-orange-100 text-orange-800' },
];

/**
 * IdentityCard — PRD FR-007
 *
 * Renders the citizen's digital identity card with:
 * - Green gradient header with Islamic geometric pattern texture
 * - Photo, full name (Persian + Latin), masked national ID with reveal toggle
 * - Active credential badges
 * - QR code section (locked until user interaction)
 * - Status indicator in header and footer
 *
 * RTL-first layout: photo appears on the right in RTL (default) and left in LTR.
 * Accessible: WCAG 2.1 AA contrast ratios; aria-labels on all interactive elements.
 */
export default function IdentityCard({
  card,
  credentials,
  loading = false,
  onQRReveal,
}: IdentityCardProps) {
  const { t } = useTranslation();
  const [showNationalId, setShowNationalId] = useState(false);
  const [showQR, setShowQR] = useState(false);

  // ── Loading skeleton ────────────────────────────────────────────────────────
  if (loading) {
    return (
      <div
        className="rounded-2xl shadow-xl overflow-hidden bg-white animate-pulse"
        role="progressbar"
        aria-label={t('common.loading')}
      >
        <div className="h-20 bg-indis-primary/20" />
        <div className="p-4 space-y-3">
          <div className="h-4 bg-gray-200 rounded w-3/4" />
          <div className="h-4 bg-gray-200 rounded w-1/2" />
          <div className="h-16 bg-gray-200 rounded" />
        </div>
      </div>
    );
  }

  // ── Empty / not-enrolled state ──────────────────────────────────────────────
  if (!card) {
    return (
      <div className="rounded-2xl shadow-xl overflow-hidden bg-white border-2 border-dashed border-gray-300 p-8 text-center">
        <ShieldCheckIcon className="w-16 h-16 text-gray-300 mx-auto mb-3" aria-hidden="true" />
        <p className="text-gray-500 text-sm">{t('enrollment.subtitle')}</p>
      </div>
    );
  }

  const activeBadges = CREDENTIAL_BADGES.filter((b) => credentials.includes(b.type));

  const handleQRActivate = () => {
    setShowQR(true);
    onQRReveal?.();
  };

  return (
    <article
      className="rounded-2xl shadow-xl overflow-hidden bg-white select-none"
      aria-label={t('home.title')}
      dir="rtl"
    >
      {/* ── Card Header ──────────────────────────────────────────────────────── */}
      <div
        className="relative bg-gradient-to-b from-indis-primary to-indis-primary-dark px-4 pt-4 pb-6"
        style={{
          backgroundImage:
            "url(\"data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.05'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E\")",
        }}
        aria-hidden="false"
      >
        {/* Status badge — top-start (right in RTL) */}
        <div className="absolute top-4 start-4">
          <span
            className="inline-flex items-center gap-1 bg-green-400/20 text-white text-xs px-2 py-0.5 rounded-full border border-green-300/30"
            role="status"
            aria-label={t('home.status_verified')}
          >
            <CheckBadgeIcon className="w-3 h-3" aria-hidden="true" />
            {t('home.status_verified')}
          </span>
        </div>

        {/* Bilingual title */}
        <div className="text-center mb-2">
          <p className="text-white font-bold text-sm" lang="fa">
            سیستم هویت دیجیتال ملی ایران
          </p>
          <p className="text-white/70 text-xs" lang="en" dir="ltr">
            Iran National Digital Identity System
          </p>
        </div>
      </div>

      {/* ── Card Body ────────────────────────────────────────────────────────── */}
      <div className="px-4 py-4 space-y-4">
        {/* Identity row: photo + name/ID */}
        <div className="flex items-start gap-4 flex-row">
          {/* Name & National ID (flex-1 so it fills available space) */}
          <div className="flex-1 min-w-0">
            <p className="font-bold text-gray-900 text-lg leading-snug truncate" lang="fa">
              {card.fullName}
            </p>
            {card.fullNameEn && (
              <p className="text-gray-500 text-sm truncate" lang="en" dir="ltr">
                {card.fullNameEn}
              </p>
            )}

            <div className="mt-2 flex items-center gap-2 flex-wrap">
              <span className="text-gray-500 text-xs" lang="fa">
                {t('home.national_id')}:
              </span>
              <span
                className="font-mono text-sm text-gray-800 tracking-widest"
                aria-label={showNationalId ? card.nationalId : t('home.national_id_masked')}
              >
                {showNationalId ? card.nationalId : '●●●●●●●●●●'}
              </span>
              <button
                type="button"
                onClick={() => setShowNationalId((v) => !v)}
                className="text-indis-primary text-xs flex items-center gap-0.5 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary rounded"
                aria-label={showNationalId ? t('home.hide') : t('home.show')}
                aria-pressed={showNationalId}
              >
                {showNationalId ? (
                  <>
                    <EyeSlashIcon className="w-3.5 h-3.5" aria-hidden="true" />
                    {t('home.hide')}
                  </>
                ) : (
                  <>
                    <EyeIcon className="w-3.5 h-3.5" aria-hidden="true" />
                    {t('home.show')}
                  </>
                )}
              </button>
            </div>
          </div>

          {/* Photo — appears on the left visually in RTL because flex-row with RTL dir */}
          <div
            className="flex-shrink-0 w-20 h-24 bg-gray-100 rounded-lg border-2 border-gray-200 flex items-center justify-center overflow-hidden"
            aria-label={t('home.photo_label') ?? 'عکس شخصی'}
          >
            {card.photo ? (
              <img
                src={card.photo}
                alt={card.fullName}
                className="w-full h-full object-cover"
              />
            ) : (
              <div
                className="text-gray-400 text-3xl font-bold"
                aria-hidden="true"
              >
                {card.fullName?.charAt(0) ?? '?'}
              </div>
            )}
          </div>
        </div>

        {/* Credential Badges */}
        {activeBadges.length > 0 && (
          <div
            className="flex flex-wrap gap-1.5"
            role="list"
            aria-label={t('home.credentials')}
          >
            {activeBadges.map((b) => (
              <span
                key={b.type}
                role="listitem"
                className={cn(
                  'inline-flex items-center gap-0.5 text-xs px-2 py-0.5 rounded-full font-medium',
                  b.color,
                )}
              >
                <CheckBadgeIcon className="w-3 h-3" aria-hidden="true" />
                {t(b.labelKey)}
              </span>
            ))}
          </div>
        )}

        {/* QR Section */}
        <div
          className="border border-gray-200 rounded-xl p-3 flex flex-col items-center gap-2 cursor-pointer hover:bg-gray-50 focus-within:ring-2 focus-within:ring-indis-primary transition-colors"
          onClick={handleQRActivate}
          role="button"
          tabIndex={0}
          aria-label={showQR ? 'QR Code' : t('home.qr_tap_hint')}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              handleQRActivate();
            }
          }}
        >
          {showQR && card.qrCode ? (
            <img
              src={`data:image/png;base64,${card.qrCode}`}
              alt="QR Code"
              className="w-32 h-32"
            />
          ) : (
            <>
              <QrCodeIcon className="w-10 h-10 text-gray-400" aria-hidden="true" />
              <p className="text-gray-500 text-xs text-center" lang="fa">
                {t('home.qr_tap_hint')}
              </p>
            </>
          )}
        </div>
      </div>

      {/* ── Card Footer ──────────────────────────────────────────────────────── */}
      <div className="px-4 py-2 bg-gray-50 border-t border-gray-100 flex justify-between items-center text-xs text-gray-400">
        <span
          className="font-mono text-[10px] truncate max-w-[60%]"
          title={card.did}
          aria-label={`DID: ${card.did}`}
          dir="ltr"
        >
          {card.did}
        </span>
        <span className="text-green-600 font-medium flex items-center gap-0.5">
          <CheckBadgeIcon className="w-3.5 h-3.5" aria-hidden="true" />
          {t('home.status_verified')}
        </span>
      </div>
    </article>
  );
}
