import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  QrCodeIcon, CheckCircleIcon, XCircleIcon, ShieldCheckIcon, InformationCircleIcon,
} from '@heroicons/react/24/outline';
import QRDisplay from '../../components/QRDisplay/QRDisplay';
import { useAuthStore } from '../../auth/store';
import { cn } from '../../lib/cn';

// In a real implementation, pending verification requests would come from
// a WebSocket/SSE push from the gateway (notification service).
// For now, we show a mock pending request to demonstrate the UI.
interface VerificationRequest {
  id: string;
  verifierName: string;
  verifierDid: string;
  requestedCredentials: string[];
  purpose: string;
  timestamp: string;
  expiresAt: string;
}

const MOCK_REQUEST: VerificationRequest = {
  id: 'req_demo_001',
  verifierName: 'بانک ملی ایران',
  verifierDid: 'did:indis:bank:melli',
  requestedCredentials: ['CitizenshipCredential', 'AgeRangeCredential'],
  purpose: 'احراز هویت برای افتتاح حساب',
  timestamp: new Date().toISOString(),
  expiresAt: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
};

export default function Verify() {
  const { t } = useTranslation();
  const did = useAuthStore((s) => s.did);
  const [showQR, setShowQR] = useState(false);
  const [pendingRequest] = useState<VerificationRequest | null>(
    // Show mock request only in demo — in production, comes from push notification
    process.env.NODE_ENV === 'development' ? MOCK_REQUEST : null,
  );
  const [requestResult, setRequestResult] = useState<'approved' | 'denied' | null>(null);

  const qrData = did
    ? JSON.stringify({ type: 'INDIS_CREDENTIAL_PRESENTATION', did, ts: Date.now() })
    : '';

  return (
    <div className="max-w-lg mx-auto px-4 py-6 space-y-6 animate-fade-in">
      <div>
        <h1 className="text-xl font-bold text-gray-900">{t('verify.title')}</h1>
      </div>

      {/* Pending Verification Request */}
      {pendingRequest && !requestResult && (
        <div className="bg-yellow-50 border-2 border-yellow-400 rounded-2xl p-4 space-y-4 animate-slide-up">
          <div className="flex items-start gap-3">
            <div className="w-10 h-10 rounded-full bg-yellow-100 flex items-center justify-center flex-shrink-0">
              <ShieldCheckIcon className="w-6 h-6 text-yellow-600" />
            </div>
            <div>
              <p className="font-bold text-gray-900">{t('verify.pending_request')}</p>
              <p className="text-gray-600 text-sm mt-0.5">{pendingRequest.verifierName}</p>
            </div>
          </div>

          {/* What's being requested */}
          <div className="bg-white rounded-xl p-3 space-y-2">
            <p className="text-xs font-medium text-gray-500 uppercase tracking-wide">درخواست مدرک</p>
            {pendingRequest.requestedCredentials.map((c) => (
              <div key={c} className="flex items-center gap-2 text-sm text-gray-700">
                <InformationCircleIcon className="w-4 h-4 text-blue-500 flex-shrink-0" />
                <span>{c}</span>
              </div>
            ))}
          </div>

          {/* Purpose */}
          <div className="bg-white rounded-xl p-3">
            <p className="text-xs font-medium text-gray-500 mb-1">هدف درخواست</p>
            <p className="text-sm text-gray-700">{pendingRequest.purpose}</p>
          </div>

          {/* Important: ZK proof — verifier gets boolean only */}
          <div className="bg-blue-50 rounded-xl p-3 flex gap-2">
            <InformationCircleIcon className="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" />
            <p className="text-xs text-blue-700">
              با تأیید، فقط یک اثبات ریاضی (بله/خیر) به تأیید‌کننده ارسال می‌شود. هیچ داده شخصی منتقل نمی‌شود.
            </p>
          </div>

          {/* Action buttons */}
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setRequestResult('denied')}
              className="flex-1 flex items-center justify-center gap-2 border-2 border-red-300 text-red-600 rounded-xl py-3 font-medium hover:bg-red-50 transition-colors"
            >
              <XCircleIcon className="w-5 h-5" />
              {t('verify.decline')}
            </button>
            <button
              type="button"
              onClick={() => setRequestResult('approved')}
              className="flex-1 flex items-center justify-center gap-2 bg-indis-primary text-white rounded-xl py-3 font-medium hover:bg-indis-primary-dark transition-colors"
            >
              <CheckCircleIcon className="w-5 h-5" />
              {t('verify.approve')}
            </button>
          </div>
        </div>
      )}

      {/* Result feedback */}
      {requestResult && (
        <div
          className={cn(
            'rounded-2xl p-6 text-center space-y-3 animate-slide-up',
            requestResult === 'approved' ? 'bg-green-50 border-2 border-green-300' : 'bg-gray-50 border-2 border-gray-200',
          )}
        >
          {requestResult === 'approved' ? (
            <>
              <CheckCircleIcon className="w-16 h-16 text-green-500 mx-auto" />
              <p className="font-bold text-green-800">تأیید انجام شد</p>
              <p className="text-green-700 text-sm">اثبات ZK با موفقیت به تأیید‌کننده ارسال شد</p>
            </>
          ) : (
            <>
              <XCircleIcon className="w-16 h-16 text-gray-400 mx-auto" />
              <p className="font-bold text-gray-700">درخواست رد شد</p>
              <p className="text-gray-500 text-sm">هیچ اطلاعاتی به تأیید‌کننده ارسال نشد</p>
            </>
          )}
        </div>
      )}

      {/* No pending request */}
      {!pendingRequest && !requestResult && (
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
