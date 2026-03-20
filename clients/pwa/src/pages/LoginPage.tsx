import { useState, FormEvent } from 'react'
import { useAuth } from '../hooks/useAuth'
import { api } from '../lib/api'

/**
 * Login page — accepts a DID and PIN combination.
 *
 * In the dev environment the backend issues a JWT on POST /v1/auth/login.
 * For local dev testing use `make dev-token` to generate a token and paste
 * it into the "token" field that appears when the dev-bypass is enabled.
 */
export default function LoginPage() {
  const { login } = useAuth()
  const [did, setDid] = useState('')
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [devToken, setDevToken] = useState('')
  const showDevBypass = import.meta.env.DEV

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const resp = await api.post<{ token: string; did: string }>('/auth/login', { did, pin })
      login(resp.token, resp.did)
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  function handleDevBypass() {
    if (!devToken.trim()) return
    // Extract sub (DID) from JWT payload without verification (dev only).
    try {
      const payload = JSON.parse(atob(devToken.split('.')[1]))
      login(devToken.trim(), payload.sub ?? 'did:indis:dev')
    } catch {
      setError('توکن نامعتبر است')
    }
  }

  return (
    <div
      style={{
        minHeight: '100dvh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 24,
      }}
    >
      <div className="card" style={{ width: '100%', maxWidth: 420 }}>
        <h1 className="text-center" style={{ marginBottom: 8, fontSize: 28 }}>ایندیس</h1>
        <p className="text-center text-muted" style={{ marginBottom: 32 }}>
          سامانه هویت دیجیتال ملی ایران
        </p>

        <form onSubmit={handleSubmit} className="flex-col gap-4">
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>
              شناسه دیجیتال (DID)
            </label>
            <input
              type="text"
              value={did}
              onChange={e => setDid(e.target.value)}
              placeholder="did:indis:…"
              required
              dir="ltr"
              style={{
                width: '100%',
                padding: '10px 14px',
                border: '1px solid var(--color-border)',
                borderRadius: 8,
                fontSize: 14,
              }}
            />
          </div>

          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>
              رمز عبور / کد شخصی
            </label>
            <input
              type="password"
              value={pin}
              onChange={e => setPin(e.target.value)}
              required
              style={{
                width: '100%',
                padding: '10px 14px',
                border: '1px solid var(--color-border)',
                borderRadius: 8,
                fontSize: 14,
              }}
            />
          </div>

          {error && (
            <p style={{ color: 'var(--color-error)', fontSize: 13 }}>{error}</p>
          )}

          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? 'در حال ورود…' : 'ورود'}
          </button>
        </form>

        {showDevBypass && (
          <details style={{ marginTop: 24 }}>
            <summary style={{ cursor: 'pointer', fontSize: 12, color: 'var(--color-text-muted)' }}>
              ورود توسعه‌دهنده (dev only)
            </summary>
            <div style={{ marginTop: 10 }}>
              <textarea
                value={devToken}
                onChange={e => setDevToken(e.target.value)}
                placeholder="JWT از make dev-token"
                rows={3}
                dir="ltr"
                style={{ width: '100%', fontSize: 11, padding: 8, borderRadius: 6, border: '1px solid var(--color-border)' }}
              />
              <button
                type="button"
                className="btn-ghost"
                style={{ marginTop: 8, width: '100%' }}
                onClick={handleDevBypass}
              >
                ورود با توکن
              </button>
            </div>
          </details>
        )}
      </div>
    </div>
  )
}
