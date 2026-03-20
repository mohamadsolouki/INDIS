import { FormEvent, useState } from 'react'
import { useNavigate } from 'react-router-dom'

const GATEWAY = import.meta.env.VITE_GATEWAY_URL ?? 'http://localhost:8080'

/**
 * LoginPage — verifier terminal registration and login.
 *
 * First-time use: registers the terminal with the gateway (POST /v1/verifier/register),
 * receiving a verifier ID and a one-time cert seed, which are stored in localStorage.
 *
 * Subsequent logins: the stored verifier ID is accepted directly (no network call
 * needed when operating offline — the ZK verification itself still calls the gateway).
 *
 * PRD FR-013: verifier terminals are registered entities with issued certificates.
 */
export default function LoginPage() {
  const navigate = useNavigate()
  const [tab, setTab] = useState<'login' | 'register'>('login')
  const [verifierId, setVerifierId] = useState('')
  const [name, setName] = useState('')
  const [organization, setOrganization] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleLogin(e: FormEvent) {
    e.preventDefault()
    if (!verifierId.trim()) { setError('شناسه تأیید‌کننده الزامی است'); return }
    setLoading(true)
    setError('')
    try {
      const resp = await fetch(`${GATEWAY}/v1/verifier/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ verifier_id: verifierId.trim() }),
      })
      if (!resp.ok) throw new Error(`خطای ${resp.status}: ${await resp.text()}`)
      const data = await resp.json() as { token: string; verifier_id?: string }
      localStorage.setItem('verifier_id', data.verifier_id ?? verifierId.trim())
      localStorage.setItem('verifier_token', data.token)
      navigate('/')
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  async function handleRegister(e: FormEvent) {
    e.preventDefault()
    if (!name.trim() || !organization.trim()) { setError('نام و سازمان الزامی هستند'); return }
    setLoading(true)
    setError('')
    try {
      const resp = await fetch(`${GATEWAY}/v1/verifier/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name.trim(), organization: organization.trim() }),
      })
      if (!resp.ok) {
        const body = await resp.text()
        throw new Error(`خطای ثبت: ${resp.status} — ${body}`)
      }
      const data = await resp.json() as { verifier_id: string; token?: string; certificate_b64?: string }
      localStorage.setItem('verifier_id', data.verifier_id)
      if (data.token) localStorage.setItem('verifier_token', data.token)
      if (data.certificate_b64) localStorage.setItem('verifier_cert', data.certificate_b64)
      navigate('/')
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100dvh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#111',
        color: '#fff',
        padding: 24,
        direction: 'rtl',
      }}
    >
      <div
        style={{
          background: '#1a1a2e',
          borderRadius: 16,
          padding: 32,
          width: '100%',
          maxWidth: 400,
          boxShadow: '0 8px 40px rgba(0,0,0,0.5)',
        }}
      >
        {/* Header */}
        <div style={{ textAlign: 'center', marginBottom: 24 }}>
          <div style={{ fontSize: 40, marginBottom: 8 }}>🛡️</div>
          <h1 style={{ fontSize: 20, fontWeight: 700 }}>پایانه تأیید INDIS</h1>
          <p style={{ fontSize: 13, color: '#aaa', marginTop: 4 }}>
            ورود تأیید‌کننده‌های مجاز
          </p>
        </div>

        {/* Tabs */}
        <div style={{ display: 'flex', marginBottom: 24, borderRadius: 10, background: '#0d0d1a', padding: 4 }}>
          {(['login', 'register'] as const).map((t) => (
            <button
              key={t}
              onClick={() => { setTab(t); setError('') }}
              style={{
                flex: 1,
                padding: '8px 0',
                borderRadius: 8,
                border: 'none',
                fontSize: 14,
                cursor: 'pointer',
                background: tab === t ? '#1a56db' : 'transparent',
                color: tab === t ? '#fff' : '#aaa',
                transition: 'background 0.2s',
              }}
            >
              {t === 'login' ? 'ورود' : 'ثبت پایانه'}
            </button>
          ))}
        </div>

        {/* Login form */}
        {tab === 'login' && (
          <form onSubmit={(e) => void handleLogin(e)} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div>
              <label style={{ display: 'block', marginBottom: 6, fontSize: 14, color: '#ccc' }}>
                شناسه تأیید‌کننده
              </label>
              <input
                type="text"
                value={verifierId}
                onChange={(e) => setVerifierId(e.target.value)}
                placeholder="verifier-xxxxx"
                dir="ltr"
                style={{
                  width: '100%',
                  padding: '10px 14px',
                  borderRadius: 8,
                  border: '1px solid #333',
                  background: '#0d0d1a',
                  color: '#fff',
                  fontSize: 14,
                  boxSizing: 'border-box',
                }}
              />
            </div>
            {error && <p style={{ color: '#ff6b6b', fontSize: 13 }}>{error}</p>}
            <button
              type="submit"
              disabled={loading}
              style={{
                background: loading ? '#555' : '#1a56db',
                color: '#fff',
                border: 'none',
                borderRadius: 8,
                padding: '12px',
                fontSize: 16,
                cursor: loading ? 'not-allowed' : 'pointer',
                fontWeight: 600,
              }}
            >
              {loading ? 'در حال ورود…' : 'ورود'}
            </button>
          </form>
        )}

        {/* Register form */}
        {tab === 'register' && (
          <form onSubmit={(e) => void handleRegister(e)} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div>
              <label style={{ display: 'block', marginBottom: 6, fontSize: 14, color: '#ccc' }}>
                نام پایانه
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="مثلاً: پایانه درب ورودی"
                style={{
                  width: '100%',
                  padding: '10px 14px',
                  borderRadius: 8,
                  border: '1px solid #333',
                  background: '#0d0d1a',
                  color: '#fff',
                  fontSize: 14,
                  boxSizing: 'border-box',
                }}
              />
            </div>
            <div>
              <label style={{ display: 'block', marginBottom: 6, fontSize: 14, color: '#ccc' }}>
                سازمان
              </label>
              <input
                type="text"
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
                placeholder="مثلاً: وزارت کشور"
                style={{
                  width: '100%',
                  padding: '10px 14px',
                  borderRadius: 8,
                  border: '1px solid #333',
                  background: '#0d0d1a',
                  color: '#fff',
                  fontSize: 14,
                  boxSizing: 'border-box',
                }}
              />
            </div>
            {error && <p style={{ color: '#ff6b6b', fontSize: 13 }}>{error}</p>}
            <button
              type="submit"
              disabled={loading}
              style={{
                background: loading ? '#555' : '#0f9960',
                color: '#fff',
                border: 'none',
                borderRadius: 8,
                padding: '12px',
                fontSize: 16,
                cursor: loading ? 'not-allowed' : 'pointer',
                fontWeight: 600,
              }}
            >
              {loading ? 'در حال ثبت…' : 'ثبت پایانه'}
            </button>
          </form>
        )}

        {/* Dev bypass */}
        {import.meta.env.DEV && (
          <button
            onClick={() => {
              localStorage.setItem('verifier_id', 'dev-verifier')
              localStorage.setItem('verifier_token', 'dev-token')
              navigate('/')
            }}
            style={{
              marginTop: 16,
              background: 'transparent',
              color: '#555',
              border: '1px solid #333',
              borderRadius: 8,
              padding: '8px',
              width: '100%',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            ورود توسعه‌دهنده (dev-verifier)
          </button>
        )}
      </div>
    </div>
  )
}
