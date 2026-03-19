import { useTranslation } from 'react-i18next';
import { CheckBadgeIcon, XCircleIcon, ClockIcon, ShieldExclamationIcon } from '@heroicons/react/24/outline';
import { cn } from '../../lib/cn';
import { formatSolarHijriLong } from '../../lib/solarHijri';
import type { WalletCredential, CredentialType } from '../../types';

interface CredentialCardProps {
  credential: WalletCredential;
  onClick?: () => void;
}

const CRED_DISPLAY: Record<CredentialType, { label: string; labelFa: string; color: string; bgColor: string }> = {
  CitizenshipCredential:       { label: 'Citizenship',        labelFa: 'شهروندی',        color: 'text-green-700',  bgColor: 'bg-green-50 border-green-200' },
  AgeRangeCredential:          { label: 'Age Range',           labelFa: 'محدوده سنی',      color: 'text-blue-700',   bgColor: 'bg-blue-50 border-blue-200' },
  VoterEligibilityCredential:  { label: 'Voter Eligibility',   labelFa: 'رأی‌دهنده',       color: 'text-indigo-700', bgColor: 'bg-indigo-50 border-indigo-200' },
  ResidencyCredential:         { label: 'Residency',           labelFa: 'اقامت',           color: 'text-purple-700', bgColor: 'bg-purple-50 border-purple-200' },
  MilitaryServiceCredential:   { label: 'Military Service',    labelFa: 'خدمت نظامی',     color: 'text-orange-700', bgColor: 'bg-orange-50 border-orange-200' },
  EmploymentCredential:        { label: 'Employment',          labelFa: 'اشتغال',          color: 'text-yellow-700', bgColor: 'bg-yellow-50 border-yellow-200' },
  HealthInsuranceCredential:   { label: 'Health Insurance',    labelFa: 'بیمه سلامت',     color: 'text-pink-700',   bgColor: 'bg-pink-50 border-pink-200' },
  DisabilityCredential:        { label: 'Disability',          labelFa: 'معلولیت',         color: 'text-teal-700',   bgColor: 'bg-teal-50 border-teal-200' },
  TemporaryEnrollmentReceipt:  { label: 'Enrollment Receipt',  labelFa: 'رسید ثبت‌نام',    color: 'text-gray-700',   bgColor: 'bg-gray-50 border-gray-200' },
  GuardianCredential:          { label: 'Guardian',            labelFa: 'سرپرست',          color: 'text-rose-700',   bgColor: 'bg-rose-50 border-rose-200' },
  SocialAttestationCredential: { label: 'Social Attestation',  labelFa: 'تأیید اجتماعی',  color: 'text-cyan-700',   bgColor: 'bg-cyan-50 border-cyan-200' },
};

export default function CredentialCard({ credential, onClick }: CredentialCardProps) {
  const { t } = useTranslation();

  // Parse raw VC to extract claims
  let parsed: { issuedAt?: string; expiresAt?: string; status?: string } = {};
  try {
    const raw = JSON.parse(credential.raw) as Record<string, unknown>;
    parsed = {
      issuedAt:  raw.issuanceDate as string | undefined,
      expiresAt: raw.expirationDate as string | undefined,
      status:    raw.credentialStatus as string | undefined,
    };
  } catch {
    parsed = { issuedAt: credential.syncedAt };
  }

  const display = CRED_DISPLAY[credential.type] ?? {
    label: credential.type, labelFa: credential.type, color: 'text-gray-700', bgColor: 'bg-gray-50 border-gray-200',
  };

  const statusIcon = {
    active:    <CheckBadgeIcon className="w-5 h-5 text-green-500" />,
    revoked:   <XCircleIcon className="w-5 h-5 text-red-500" />,
    expired:   <ClockIcon className="w-5 h-5 text-yellow-500" />,
    suspended: <ShieldExclamationIcon className="w-5 h-5 text-orange-500" />,
  };

  const isExpired = parsed.expiresAt ? new Date(parsed.expiresAt) < new Date() : false;
  const effectiveStatus = isExpired ? 'expired' : (parsed.status ?? 'active');

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'w-full text-right rtl:text-right ltr:text-left rounded-xl border p-4 space-y-2 transition-all hover:shadow-md active:scale-[0.99]',
        display.bgColor,
        onClick ? 'cursor-pointer' : 'cursor-default',
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className={cn('font-bold text-base', display.color)}>{display.labelFa}</p>
          <p className="text-xs text-gray-500">{display.label}</p>
        </div>
        {statusIcon[effectiveStatus as keyof typeof statusIcon] ?? statusIcon.active}
      </div>

      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500">
        {parsed.issuedAt && (
          <span>{t('wallet.issued')}: {formatSolarHijriLong(new Date(parsed.issuedAt))}</span>
        )}
        {parsed.expiresAt && (
          <span className={isExpired ? 'text-red-500 font-medium' : ''}>
            {t('wallet.expires')}: {formatSolarHijriLong(new Date(parsed.expiresAt))}
          </span>
        )}
      </div>

      {!credential.isOfflineAvailable && (
        <span className="inline-block text-[10px] bg-yellow-100 text-yellow-700 px-1.5 py-0.5 rounded">
          {t('wallet.offline_badge')} — نیاز به اتصال
        </span>
      )}
    </button>
  );
}
