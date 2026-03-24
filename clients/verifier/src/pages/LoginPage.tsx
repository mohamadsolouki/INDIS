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
      className="verifier-screen verifier-screen--center"
      dir="rtl"
    >
      <div
        className="verifier-panel login-card"
      >
        {/* Header */}
        <div className="login-header">
          <div className="login-icon">🛡️</div>
          <h1 className="login-title">پایانه تأیید INDIS</h1>
          <p className="login-subtitle">
            ورود تأیید‌کننده‌های مجاز
          </p>
        </div>

        {/* Tabs */}
        <div className="login-tabs">
          {(['login', 'register'] as const).map((t) => (
            <button
              key={t}
              onClick={() => { setTab(t); setError('') }}
              className={`verifier-tab ${tab === t ? 'verifier-tab--active' : ''}`}
            >
              {t === 'login' ? 'ورود' : 'ثبت پایانه'}
            </button>
          ))}
        </div>

        {/* Login form */}
        {tab === 'login' && (
          <form onSubmit={(e) => void handleLogin(e)} className="login-form">
            <div>
              <label className="login-field-label">
                شناسه تأیید‌کننده
              </label>
              <input
                type="text"
                value={verifierId}
                onChange={(e) => setVerifierId(e.target.value)}
                placeholder="verifier-xxxxx"
                dir="ltr"
                className="verifier-input"
              />
            </div>
            {error && <p role="alert" className="login-error">{error}</p>}
            <button
              type="submit"
              disabled={loading}
              className="login-submit login-submit--primary"
            >
              {loading ? 'در حال ورود…' : 'ورود'}
            </button>
          </form>
        )}

        {/* Register form */}
        {tab === 'register' && (
          <form onSubmit={(e) => void handleRegister(e)} className="login-form">
            <div>
              <label className="login-field-label">
                نام پایانه
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="مثلاً: پایانه درب ورودی"
                className="verifier-input"
              />
            </div>
            <div>
              <label className="login-field-label">
                سازمان
              </label>
              <input
                type="text"
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
                placeholder="مثلاً: وزارت کشور"
                className="verifier-input"
              />
            </div>
            {error && <p role="alert" className="login-error">{error}</p>}
            <button
              type="submit"
              disabled={loading}
              className="login-submit login-submit--success"
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
            className="login-dev-btn"
          >
            ورود توسعه‌دهنده (dev-verifier)
          </button>
        )}
      </div>
    </div>
  )
}
