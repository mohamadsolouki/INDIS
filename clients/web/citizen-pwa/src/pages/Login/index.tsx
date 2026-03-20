import { useState, FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { FingerPrintIcon, KeyIcon, DevicePhoneMobileIcon } from '@heroicons/react/24/outline';
import { cn } from '../../lib/cn';
import { useAuthStore } from '../../auth/store';
import {
  isWebAuthnAvailable,
  authenticateWebAuthn,
  registerWebAuthn,
  getStoredCredentialId,
  storeCredentialId,
  bufferToBase64url,
} from '../../auth/webauthn';
import { http } from '../../api/client';

/**
 * Login page — primary entry-point for unauthenticated citizens.
 *
 * Three auth paths:
 *  1. WebAuthn (platform authenticator / biometric) — device-bound, PRD FR-001.4
 *  2. DID + PIN fallback — for devices without biometric hardware
 *  3. Dev bypass — available only in development builds
 */
export default function Login() {
  const { t } = useTranslation();
  const login = useAuthStore((s) => s.login);

  const [mode, setMode] = useState<'webauthn' | 'pin'>('webauthn');
  const [did, setDid] = useState('');
  const [pin, setPin] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [webAuthnReady, setWebAuthnReady] = useState<boolean | null>(null);
  const [devToken, setDevToken] = useState('');

  // Check WebAuthn availability once on mount
  useState(() => {
    void isWebAuthnAvailable().then((ok) => {
      setWebAuthnReady(ok);
      if (!ok) setMode('pin');
    });
  });

  // ── WebAuthn auth ─────────────────────────────────────────────────────────

  async function handleWebAuthn() {
    setError('');
    setLoading(true);
    try {
      const storedCredId = getStoredCredentialId();
      const assertion = await authenticateWebAuthn(storedCredId ?? undefined);

      // Exchange assertion for JWT from gateway.
      // The gateway verifies the authenticator signature and returns a signed JWT.
      const resp = await http.post<{ token: string }>('/v1/auth/webauthn/verify', {
        credential_id: assertion.id,
        client_data_json: bufferToBase64url(
          (assertion.response as AuthenticatorAssertionResponse).clientDataJSON,
        ),
        authenticator_data: bufferToBase64url(
          (assertion.response as AuthenticatorAssertionResponse).authenticatorData,
        ),
        signature: bufferToBase64url(
          (assertion.response as AuthenticatorAssertionResponse).signature,
        ),
      });
      login(resp.token);
    } catch (err) {
      // If no credential registered yet, try registration flow.
      if (err instanceof DOMException && err.name === 'NotAllowedError') {
        setError('تأیید بیومتریک لغو شد');
      } else if (err instanceof Error && err.message.includes('NotAllowedError')) {
        setError('تأیید بیومتریک لغو شد');
      } else {
        // Might be first-time use — show register option
        setError('احراز هویت بیومتریک ناموفق بود. آیا می‌خواهید ثبت‌نام کنید؟');
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleWebAuthnRegister() {
    if (!did.trim()) {
      setError('برای ثبت‌نام بیومتریک، ابتدا DID خود را وارد کنید');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const cred = await registerWebAuthn(did, did);
      storeCredentialId(cred.id);

      // Register credential on the gateway.
      const resp = await http.post<{ token: string }>('/v1/auth/webauthn/register', {
        did,
        credential_id: cred.id,
        public_key: bufferToBase64url(
          (cred.response as AuthenticatorAttestationResponse).getPublicKey() ??
            new ArrayBuffer(0),
        ),
        attestation_object: bufferToBase64url(
          (cred.response as AuthenticatorAttestationResponse).attestationObject,
        ),
        client_data_json: bufferToBase64url(
          (cred.response as AuthenticatorAttestationResponse).clientDataJSON,
        ),
      });
      login(resp.token);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'ثبت بیومتریک ناموفق بود');
    } finally {
      setLoading(false);
    }
  }

  // ── DID + PIN auth ────────────────────────────────────────────────────────

  async function handlePinLogin(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const resp = await http.post<{ token: string }>('/v1/auth/login', { did, pin });
      login(resp.token);
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errors.unauthorized'));
    } finally {
      setLoading(false);
    }
  }

  // ── Dev bypass ────────────────────────────────────────────────────────────

  function handleDevLogin() {
    if (!devToken.trim()) return;
    try {
      // Minimal decode to verify it's a valid JWT before accepting.
      const parts = devToken.split('.');
      if (parts.length !== 3) throw new Error('invalid');
      JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')));
      login(devToken.trim());
    } catch {
      setError('توکن JWT نامعتبر است');
    }
  }

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div className="min-h-dvh bg-gradient-to-b from-indis-primary to-indis-primary-dark flex flex-col items-center justify-center px-6 py-12">
      {/* App title */}
      <div className="text-center mb-10 space-y-2">
        <h1 className="text-4xl font-black text-white tracking-tight">ایندیس</h1>
        <p className="text-white/70 text-sm">{t('app.tagline')}</p>
      </div>

      {/* Auth card */}
      <div className="w-full max-w-sm bg-white rounded-3xl shadow-2xl overflow-hidden">
        {/* Mode tabs */}
        <div className="flex border-b border-gray-100">
          {webAuthnReady !== false && (
            <button
              type="button"
              onClick={() => setMode('webauthn')}
              className={cn(
                'flex-1 py-4 flex items-center justify-center gap-2 text-sm font-medium transition-colors',
                mode === 'webauthn'
                  ? 'text-indis-primary border-b-2 border-indis-primary -mb-px'
                  : 'text-gray-500 hover:text-gray-700',
              )}
            >
              <FingerPrintIcon className="w-4 h-4" />
              بیومتریک
            </button>
          )}
          <button
            type="button"
            onClick={() => setMode('pin')}
            className={cn(
              'flex-1 py-4 flex items-center justify-center gap-2 text-sm font-medium transition-colors',
              mode === 'pin'
                ? 'text-indis-primary border-b-2 border-indis-primary -mb-px'
                : 'text-gray-500 hover:text-gray-700',
            )}
          >
            <KeyIcon className="w-4 h-4" />
            DID + رمز
          </button>
        </div>

        <div className="p-6 space-y-5">
          {/* Error */}
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-xl p-3" role="alert">
              <p className="text-red-700 text-sm">{error}</p>
            </div>
          )}

          {/* WebAuthn mode */}
          {mode === 'webauthn' && (
            <div className="space-y-4">
              <div className="text-center py-4">
                <div className="w-20 h-20 mx-auto bg-indis-primary/10 rounded-full flex items-center justify-center mb-4">
                  <FingerPrintIcon className="w-10 h-10 text-indis-primary" />
                </div>
                <p className="text-gray-700 text-sm">
                  از اثر انگشت یا تشخیص چهره دستگاه برای ورود استفاده کنید
                </p>
              </div>

              <button
                type="button"
                onClick={() => void handleWebAuthn()}
                disabled={loading}
                className={cn(
                  'w-full flex items-center justify-center gap-3 bg-indis-primary text-white rounded-2xl py-4 font-semibold hover:bg-indis-primary-dark transition-colors',
                  loading && 'opacity-60 cursor-not-allowed',
                )}
              >
                {loading ? (
                  <span className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                ) : (
                  <FingerPrintIcon className="w-5 h-5" />
                )}
                ورود بیومتریک
              </button>

              <div className="relative flex items-center gap-3">
                <div className="flex-1 h-px bg-gray-200" />
                <span className="text-gray-400 text-xs">یا</span>
                <div className="flex-1 h-px bg-gray-200" />
              </div>

              {/* First-time registration */}
              <div className="space-y-3">
                <input
                  type="text"
                  value={did}
                  onChange={(e) => setDid(e.target.value)}
                  placeholder="did:indis:…"
                  dir="ltr"
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-indis-primary placeholder:text-gray-400"
                />
                <button
                  type="button"
                  onClick={() => void handleWebAuthnRegister()}
                  disabled={loading || !did.trim()}
                  className="w-full flex items-center justify-center gap-2 border-2 border-indis-primary text-indis-primary rounded-2xl py-3 text-sm font-medium hover:bg-indis-primary/5 transition-colors disabled:opacity-50"
                >
                  <DevicePhoneMobileIcon className="w-4 h-4" />
                  ثبت دستگاه جدید
                </button>
              </div>
            </div>
          )}

          {/* DID + PIN mode */}
          {mode === 'pin' && (
            <form onSubmit={(e) => void handlePinLogin(e)} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm text-gray-600 block">شناسه دیجیتال (DID)</label>
                <input
                  type="text"
                  value={did}
                  onChange={(e) => setDid(e.target.value)}
                  placeholder="did:indis:…"
                  required
                  dir="ltr"
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-indis-primary"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm text-gray-600 block">رمز عبور / PIN</label>
                <input
                  type="password"
                  value={pin}
                  onChange={(e) => setPin(e.target.value)}
                  required
                  className="w-full rounded-xl border border-gray-200 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-indis-primary"
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className={cn(
                  'w-full bg-indis-primary text-white rounded-2xl py-4 font-semibold hover:bg-indis-primary-dark transition-colors',
                  loading && 'opacity-60 cursor-not-allowed',
                )}
              >
                {loading ? (
                  <span className="inline-block w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                ) : (
                  'ورود'
                )}
              </button>
            </form>
          )}

          {/* Dev bypass — development builds only */}
          {import.meta.env.DEV && (
            <details className="mt-2">
              <summary className="text-xs text-gray-400 cursor-pointer hover:text-gray-500 select-none">
                حالت توسعه‌دهنده
              </summary>
              <div className="mt-3 space-y-2">
                <p className="text-xs text-gray-400">توکن از <code>make dev-token</code></p>
                <textarea
                  value={devToken}
                  onChange={(e) => setDevToken(e.target.value)}
                  rows={2}
                  dir="ltr"
                  placeholder="eyJ…"
                  className="w-full rounded-lg border border-gray-200 px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-indis-primary"
                />
                <button
                  type="button"
                  onClick={handleDevLogin}
                  className="w-full border border-gray-200 rounded-xl py-2 text-xs text-gray-600 hover:bg-gray-50 transition-colors"
                >
                  ورود با توکن
                </button>
              </div>
            </details>
          )}
        </div>
      </div>

      {/* Enrollment link */}
      <p className="mt-8 text-white/60 text-sm text-center">
        هنوز هویت دیجیتال ندارید؟{' '}
        <a href="/enrollment" className="text-white font-semibold hover:underline">
          ثبت‌نام کنید
        </a>
      </p>
    </div>
  );
}
