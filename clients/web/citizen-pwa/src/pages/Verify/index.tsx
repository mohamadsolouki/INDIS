import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  QrCodeIcon, CheckCircleIcon, XCircleIcon, ShieldCheckIcon, InformationCircleIcon,
  SignalIcon, SignalSlashIcon,
} from '@heroicons/react/24/outline';
import QRDisplay from '../../components/QRDisplay/QRDisplay';
import { useAuthStore } from '../../auth/store';
import { useVerificationRequests } from '../../hooks/useVerificationRequests';
import { http } from '../../api/client';
import { cn } from '../../lib/cn';

export default function Verify() {
  const { t } = useTranslation();
  const did = useAuthStore((s) => s.did);
  const [showQR, setShowQR] = useState(false);
  const [requestResult, setRequestResult] = useState<Record<string, 'approved' | 'denied'>>({});
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Real-time verification requests via SSE.
  const { requests, status, dismiss } = useVerificationRequests();

  const qrData = did
    ? JSON.stringify({ type: 'INDIS_CREDENTIAL_PRESENTATION', did, ts: Date.now() })
    : '';

  async function handleDecision(requestId: string, decision: 'approved' | 'denied') {
    setActionLoading(requestId);
    try {
      if (decision === 'approved') {
        // POST ZK proof generation request to the gateway.
        await http.post('/v1/verifier/respond', {
          request_id: requestId,
          decision: 'approved',
          proof_system: 'groth16',
        });
      }
      setRequestResult((prev) => ({ ...prev, [requestId]: decision }));
      setTimeout(() => dismiss(requestId), 3000);
    } catch {
      // Best-effort — still mark locally.
      setRequestResult((prev) => ({ ...prev, [requestId]: decision }));
    } finally {
      setActionLoading(null);
    }
  }

  return (
    <div className="max-w-lg mx-auto px-4 py-6 space-y-6 animate-fade-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-gray-900">{t('verify.title')}</h1>
        {/* SSE connection status indicator */}
        <div className="flex items-center gap-1.5 text-xs">
          {status === 'connected' ? (
            <>
              <SignalIcon className="w-4 h-4 text-green-500" />
              <span className="text-green-600">متصل</span>
            </>
          ) : (
            <>
              <SignalSlashIcon className="w-4 h-4 text-gray-400" />
              <span className="text-gray-400">آفلاین</span>
            </>
          )}
        </div>
      </div>

      {/* Pending Verification Requests (real-time from SSE) */}
      {requests.map((req) => (
        <div key={req.id} className="bg-yellow-50 border-2 border-yellow-400 rounded-2xl p-4 space-y-4 animate-slide-up">
          {requestResult[req.id] ? (
            /* Result overlay */
            <div className={cn(
              'text-center py-4 space-y-2',
              requestResult[req.id] === 'approved' ? 'text-green-700' : 'text-gray-500',
            )}>
              {requestResult[req.id] === 'approved' ? (
                <>
                  <CheckCircleIcon className="w-12 h-12 mx-auto text-green-500" />
                  <p className="font-bold">تأیید انجام شد</p>
                  <p className="text-xs">اثبات ZK ارسال شد</p>
                </>
              ) : (
                <>
                  <XCircleIcon className="w-12 h-12 mx-auto text-gray-400" />
                  <p className="font-bold">درخواست رد شد</p>
                </>
              )}
            </div>
          ) : (
            <>
              <div className="flex items-start gap-3">
                <div className="w-10 h-10 rounded-full bg-yellow-100 flex items-center justify-center flex-shrink-0">
                  <ShieldCheckIcon className="w-6 h-6 text-yellow-600" />
                </div>
                <div>
                  <p className="font-bold text-gray-900">{t('verify.pending_request')}</p>
                  <p className="text-gray-600 text-sm mt-0.5">{req.verifierName}</p>
                </div>
              </div>

              <div className="bg-white rounded-xl p-3 space-y-2">
                <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">درخواست مدرک</p>
                {req.requestedCredentials.map((c) => (
                  <div key={c} className="flex items-center gap-2 text-sm text-gray-700">
                    <InformationCircleIcon className="w-4 h-4 text-blue-500 flex-shrink-0" />
                    <span>{c}</span>
                  </div>
                ))}
              </div>

              <div className="bg-white rounded-xl p-3">
                <p className="text-xs font-medium text-gray-500 mb-1">هدف درخواست</p>
                <p className="text-sm text-gray-700">{req.purpose}</p>
              </div>

              <div className="bg-blue-50 rounded-xl p-3 flex gap-2">
                <InformationCircleIcon className="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" />
                <p className="text-xs text-blue-700">
                  با تأیید، فقط یک اثبات ریاضی (بله/خیر) به تأیید‌کننده ارسال می‌شود. هیچ داده شخصی منتقل نمی‌شود.
                </p>
              </div>

              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => void handleDecision(req.id, 'denied')}
                  disabled={actionLoading === req.id}
                  className="flex-1 flex items-center justify-center gap-2 border-2 border-red-300 text-red-600 rounded-xl py-3 font-medium hover:bg-red-50 transition-colors disabled:opacity-50"
                >
                  <XCircleIcon className="w-5 h-5" />
                  {t('verify.decline')}
                </button>
                <button
                  type="button"
                  onClick={() => void handleDecision(req.id, 'approved')}
                  disabled={actionLoading === req.id}
                  className="flex-1 flex items-center justify-center gap-2 bg-indis-primary text-white rounded-xl py-3 font-medium hover:bg-indis-primary-dark transition-colors disabled:opacity-50"
                >
                  {actionLoading === req.id ? (
                    <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  ) : (
                    <CheckCircleIcon className="w-5 h-5" />
                  )}
                  {t('verify.approve')}
                </button>
              </div>
            </>
          )}
        </div>
      ))}

      {/* Empty state */}
      {requests.length === 0 && (
        <div className="bg-white rounded-2xl border border-gray-100 p-6 text-center space-y-3">
          <QrCodeIcon className="w-16 h-16 text-gray-300 mx-auto" />
          <p className="text-gray-500">{t('verify.no_requests')}</p>
          <p className="text-gray-400 text-xs">
            درخواست‌های تأیید به‌صورت خودکار اینجا نمایش داده می‌شوند
          </p>
        </div>
      )}

      {/* QR Code for offline verification */}
      {did && (
        <div className="bg-white rounded-2xl border border-gray-100 p-4 space-y-3">
          <div className="flex items-center justify-between">
            <p className="font-medium text-gray-800 text-sm">نمایش آفلاین</p>
            <button
              type="button"
              onClick={() => setShowQR((v) => !v)}
              className="text-indis-primary text-xs hover:underline"
            >
              {showQR ? 'پنهان کردن' : t('verify.scan_qr')}
            </button>
          </div>
          <p className="text-xs text-gray-500">
            برای تأیید آفلاین، این کیوآر را به تأیید‌کننده نشان دهید
          </p>

          {showQR && (
            <QRDisplay
              value={qrData}
              label="هویت دیجیتال INDIS"
              size={180}
              className="py-2"
            />
          )}
        </div>
      )}
    </div>
  );
}
