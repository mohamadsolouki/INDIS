import { useState, type ElementType, type FormEvent, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import {
  ClockIcon, ShareIcon, ShieldCheckIcon, ArrowDownTrayIcon,
  TrashIcon, PlusIcon, ArrowPathIcon,
} from '@heroicons/react/24/outline';
import { cn } from '../../lib/cn';
import { formatSolarHijriLong } from '../../lib/solarHijri';
import { usePrivacyHistory, useConsentRules, useDataExport } from '../../hooks/usePrivacy';
import type { PrivacyEvent, ConsentRule, ConsentRuleRequest } from '../../types';

type Tab = 'history' | 'sharing' | 'consent' | 'export';

const TABS: { id: Tab; labelKey: string; Icon: ElementType }[] = [
  { id: 'history', labelKey: 'privacy.history',  Icon: ClockIcon },
  { id: 'sharing', labelKey: 'privacy.sharing',  Icon: ShareIcon },
  { id: 'consent', labelKey: 'privacy.consent',  Icon: ShieldCheckIcon },
  { id: 'export',  labelKey: 'privacy.export',   Icon: ArrowDownTrayIcon },
];

export default function Privacy() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<Tab>('history');

  return (
    <div className="max-w-lg mx-auto animate-fade-in">
      {/* Header */}
      <div className="px-4 py-5 bg-white border-b border-gray-100">
        <h1 className="text-xl font-bold text-gray-900">{t('privacy.title')}</h1>
        <p className="text-sm text-gray-500 mt-1">{t('privacy.subtitle')}</p>
      </div>

      {/* Tab bar */}
      <div className="bg-white border-b border-gray-100 overflow-x-auto hide-scrollbar">
        <div className="flex min-w-max">
          {TABS.map(({ id, labelKey, Icon }) => (
            <button
              key={id}
              type="button"
              onClick={() => setActiveTab(id)}
              className={cn(
                'flex items-center gap-1.5 px-4 py-3 text-sm font-medium border-b-2 whitespace-nowrap transition-colors',
                activeTab === id
                  ? 'border-indis-primary text-indis-primary'
                  : 'border-transparent text-gray-500 hover:text-gray-700',
              )}
            >
              <Icon className="w-4 h-4" />
              {t(labelKey)}
            </button>
          ))}
        </div>
      </div>

      {/* Tab content */}
      <div className="px-4 py-4">
        {activeTab === 'history' && <HistoryTab />}
        {activeTab === 'sharing' && <SharingTab />}
        {activeTab === 'consent' && <ConsentTab />}
        {activeTab === 'export'  && <ExportTab />}
      </div>
    </div>
  );
}

// ── History Tab ───────────────────────────────────────────────────────────────

function HistoryTab() {
  const { t } = useTranslation();
  const { events, loading, error, hasMore, loadMore, reload } = usePrivacyHistory();

  if (loading && events.length === 0) return <LoadingSpinner />;

  if (error) return <ErrorBanner message={error} onRetry={reload} />;

  if (events.length === 0) {
    return (
      <EmptyState icon={<ClockIcon className="w-12 h-12 text-gray-300" />} message={t('privacy.no_history')} />
    );
  }

  return (
    <div className="space-y-2">
      {events.map((event) => <EventRow key={event.eventId} event={event} />)}
      {hasMore && (
        <button
          type="button"
          onClick={() => void loadMore()}
          disabled={loading}
          className="w-full py-2 text-indis-primary text-sm hover:underline"
        >
          {loading ? '...' : 'بیشتر نمایش بده'}
        </button>
      )}
    </div>
  );
}

// ── Sharing Tab ───────────────────────────────────────────────────────────────

function SharingTab() {
  const { t } = useTranslation();
  // Sharing tab uses the same structure as history — just calls getSharing endpoint.
  // We reuse the history hook pattern here for brevity.
  const { events, loading, error, reload } = usePrivacyHistory(); // TODO: use sharing endpoint

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorBanner message={error} onRetry={reload} />;
  if (events.length === 0) {
    return (
      <EmptyState icon={<ShareIcon className="w-12 h-12 text-gray-300" />} message={t('privacy.no_history')} />
    );
  }

  return (
    <div className="space-y-2">
      {events.map((event) => <EventRow key={event.eventId} event={event} />)}
    </div>
  );
}

function EventRow({ event }: { event: PrivacyEvent }) {
  const { t } = useTranslation();
  const resultColors: Record<string, string> = {
    approved:      'text-green-600 bg-green-50',
    denied:        'text-red-600 bg-red-50',
    auto_approved: 'text-blue-600 bg-blue-50',
  };
  const resultLabels: Record<string, string> = {
    approved:      t('privacy.result_approved'),
    denied:        t('privacy.result_denied'),
    auto_approved: t('privacy.result_auto'),
  };

  return (
    <div className="bg-white rounded-xl border border-gray-100 p-3 space-y-1.5">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-medium text-gray-800 text-sm truncate">{event.verifierName || event.verifierDid}</p>
          <p className="text-xs text-gray-500">{event.credentialType}</p>
        </div>
        <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium flex-shrink-0', resultColors[event.result] ?? 'text-gray-500 bg-gray-50')}>
          {resultLabels[event.result] ?? event.result}
        </span>
      </div>
      <p className="text-xs text-gray-400">{formatSolarHijriLong(new Date(event.timestamp))}</p>
    </div>
  );
}

// ── Consent Tab ───────────────────────────────────────────────────────────────

function ConsentTab() {
  const { t } = useTranslation();
  const { rules, loading, error, reload, addRule, removeRule } = useConsentRules();
  const [showForm, setShowForm] = useState(false);

  if (loading && rules.length === 0) return <LoadingSpinner />;
  if (error) return <ErrorBanner message={error} onRetry={reload} />;

  return (
    <div className="space-y-3">
      {/* Description */}
      <p className="text-xs text-gray-500">
        قوانین اشتراک‌گذاری خودکار مدارک با تأیید‌کنندگان مختلف را مدیریت کنید.
      </p>

      {/* Rules list */}
      {rules.length === 0 && !showForm && (
        <EmptyState icon={<ShieldCheckIcon className="w-12 h-12 text-gray-300" />} message="هیچ قانونی تعریف نشده" />
      )}

      {rules.map((rule) => (
        <ConsentRuleRow
          key={rule.id}
          rule={rule}
          onDelete={() => void removeRule(rule.id)}
          t={t}
        />
      ))}

      {/* Add rule form or trigger */}
      {showForm ? (
        <AddConsentRuleForm
          onAdd={async (req) => { await addRule(req); setShowForm(false); }}
          onCancel={() => setShowForm(false)}
          t={t}
        />
      ) : (
        <button
          type="button"
          onClick={() => setShowForm(true)}
          className="w-full flex items-center justify-center gap-2 border-2 border-dashed border-gray-300 rounded-xl py-3 text-gray-500 hover:border-indis-primary hover:text-indis-primary transition-colors text-sm"
        >
          <PlusIcon className="w-4 h-4" />
          {t('privacy.add_rule')}
        </button>
      )}
    </div>
  );
}

function ConsentRuleRow({
  rule,
  onDelete,
  t,
}: {
  rule: ConsentRule;
  onDelete: () => void;
  t: (k: string) => string;
}) {
  const ruleColor: Record<string, string> = {
    always: 'text-green-700 bg-green-50',
    ask:    'text-yellow-700 bg-yellow-50',
    never:  'text-red-700 bg-red-50',
  };
  const ruleLabel: Record<string, string> = {
    always: t('privacy.consent_always'),
    ask:    t('privacy.consent_ask'),
    never:  t('privacy.consent_never'),
  };

  return (
    <div className="bg-white rounded-xl border border-gray-100 p-3 flex items-center gap-3">
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-gray-800 truncate">{rule.verifierCategory}</p>
        <p className="text-xs text-gray-500">{rule.credentialType}</p>
      </div>
      <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', ruleColor[rule.rule] ?? 'bg-gray-50 text-gray-500')}>
        {ruleLabel[rule.rule] ?? rule.rule}
      </span>
      <button
        type="button"
        onClick={onDelete}
        className="text-gray-400 hover:text-red-500 transition-colors p-1"
        aria-label={t('privacy.delete_rule')}
      >
        <TrashIcon className="w-4 h-4" />
      </button>
    </div>
  );
}

interface AddRuleFormProps {
  onAdd: (req: ConsentRuleRequest) => Promise<void>;
  onCancel: () => void;
  t: (k: string) => string;
}

function AddConsentRuleForm({ onAdd, onCancel, t }: AddRuleFormProps) {
  const [category, setCategory] = useState('');
  const [credType, setCredType] = useState('CitizenshipCredential');
  const [rule, setRule] = useState<'always' | 'ask' | 'never'>('ask');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!category.trim()) return;
    setSaving(true);
    try {
      await onAdd({ verifier_category: category, credential_type: credType, rule });
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={(e) => void handleSubmit(e)} className="bg-gray-50 rounded-xl p-4 space-y-3 border border-gray-200">
      <div>
        <label className="block text-xs text-gray-500 mb-1">دسته تأیید‌کننده</label>
        <input
          type="text"
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          placeholder="مثال: بانک، بیمارستان"
          className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indis-primary"
          required
        />
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">نوع مدرک</label>
        <select
          value={credType}
          onChange={(e) => setCredType(e.target.value)}
          className="w-full border border-gray-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indis-primary bg-white"
        >
          <option value="CitizenshipCredential">شهروندی</option>
          <option value="VoterEligibilityCredential">رأی‌دهنده</option>
          <option value="HealthInsuranceCredential">بیمه سلامت</option>
          <option value="AgeRangeCredential">محدوده سنی</option>
          <option value="ResidencyCredential">اقامت</option>
        </select>
      </div>
      <div>
        <label className="block text-xs text-gray-500 mb-1">قانون</label>
        <div className="flex gap-2">
          {(['always', 'ask', 'never'] as const).map((r) => (
            <button
              key={r}
              type="button"
              onClick={() => setRule(r)}
              className={cn(
                'flex-1 py-2 rounded-lg text-xs font-medium border transition-colors',
                rule === r
                  ? 'bg-indis-primary text-white border-indis-primary'
                  : 'bg-white text-gray-600 border-gray-200',
              )}
            >
              {t(`privacy.consent_${r}`)}
            </button>
          ))}
        </div>
      </div>
      <div className="flex gap-2 pt-1">
        <button
          type="button"
          onClick={onCancel}
          className="flex-1 py-2 text-gray-600 border border-gray-200 rounded-lg text-sm hover:bg-gray-100"
        >
          {t('common.cancel')}
        </button>
        <button
          type="submit"
          disabled={saving}
          className="flex-1 py-2 bg-indis-primary text-white rounded-lg text-sm hover:bg-indis-primary-dark disabled:opacity-60"
        >
          {saving ? '...' : t('common.save')}
        </button>
      </div>
    </form>
  );
}

// ── Export Tab ────────────────────────────────────────────────────────────────

function ExportTab() {
  const { t } = useTranslation();
  const { request, loading, error, requestExport, checkStatus } = useDataExport();

  return (
    <div className="space-y-4">
      <p className="text-xs text-gray-500">
        یک نسخه کامل از تمام داده‌های هویتی خود را درخواست دهید. فایل خروجی رمزنگاری‌شده و امضاشده خواهد بود.
      </p>

      {error && <ErrorBanner message={error} />}

      {!request ? (
        <button
          type="button"
          onClick={() => void requestExport()}
          disabled={loading}
          className="w-full flex items-center justify-center gap-2 bg-indis-primary text-white rounded-xl py-3 font-medium hover:bg-indis-primary-dark transition-colors disabled:opacity-60"
        >
          {loading ? (
            <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          ) : (
            <ArrowDownTrayIcon className="w-5 h-5" />
          )}
          {t('privacy.export_request')}
        </button>
      ) : (
        <div className="bg-white rounded-xl border border-gray-100 p-4 space-y-3">
          <div className="flex items-center justify-between">
            <p className="font-medium text-gray-800">درخواست خروجی</p>
            <StatusBadge status={request.status} t={t} />
          </div>
          <p className="text-xs text-gray-500">{formatSolarHijriLong(new Date(request.requestedAt))}</p>

          {request.status === 'completed' && request.downloadUrl && (
            <a
              href={request.downloadUrl}
              className="flex items-center gap-2 bg-green-600 text-white rounded-lg px-4 py-2 text-sm hover:bg-green-700"
              download
            >
              <ArrowDownTrayIcon className="w-4 h-4" />
              {t('privacy.export_download')}
            </a>
          )}

          {(request.status === 'pending' || request.status === 'processing') && (
            <button
              type="button"
              onClick={() => void checkStatus(request.requestId)}
              disabled={loading}
              className="flex items-center gap-1 text-indis-primary text-xs hover:underline"
            >
              <ArrowPathIcon className={cn('w-3 h-3', loading && 'animate-spin')} />
              بروزرسانی وضعیت
            </button>
          )}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status, t }: { status: string; t: (k: string) => string }) {
  const colors: Record<string, string> = {
    pending:    'bg-yellow-100 text-yellow-700',
    processing: 'bg-blue-100 text-blue-700',
    completed:  'bg-green-100 text-green-700',
    failed:     'bg-red-100 text-red-700',
  };
  const labels: Record<string, string> = {
    pending:    t('privacy.export_pending'),
    processing: t('privacy.export_pending'),
    completed:  t('privacy.export_completed'),
    failed:     t('common.error'),
  };
  return (
    <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', colors[status] ?? 'bg-gray-100 text-gray-600')}>
      {labels[status] ?? status}
    </span>
  );
}

// ── Shared UI components ──────────────────────────────────────────────────────

function LoadingSpinner() {
  return (
    <div className="flex justify-center py-8">
      <div className="w-8 h-8 border-2 border-indis-primary border-t-transparent rounded-full animate-spin" />
    </div>
  );
}

function ErrorBanner({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="bg-red-50 border border-red-200 rounded-lg px-4 py-3 flex items-center justify-between gap-3">
      <span className="text-red-700 text-sm">{message}</span>
      {onRetry && (
        <button type="button" onClick={onRetry} className="text-red-600 text-xs hover:underline flex items-center gap-1">
          <ArrowPathIcon className="w-3 h-3" />
          {t('common.retry')}
        </button>
      )}
    </div>
  );
}

function EmptyState({ icon, message }: { icon: ReactNode; message: string }) {
  return (
    <div className="flex flex-col items-center py-12 gap-3 text-center">
      {icon}
      <p className="text-gray-500 text-sm">{message}</p>
    </div>
  );
}
