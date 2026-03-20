import { useState, FormEvent } from 'react'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const resp = await fetch('/v1/govportal/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      if (!resp.ok) throw new Error(`خطای ${resp.status}`)
      const data = await resp.json() as { token: string }
      localStorage.setItem('gov_token', data.token)
      window.location.href = '/'
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  // Dev bypass
  function devLogin() {
    const token = prompt('توکن JWT dev را وارد کنید:')
    if (token) {
      localStorage.setItem('gov_token', token)
      window.location.href = '/'
    }
  }

  return (
    <div
      style={{
        minHeight: '100dvh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#1c2437',
      }}
    >
      <div
        style={{
          background: '#fff',
          borderRadius: 12,
          padding: 32,
          width: '100%',
          maxWidth: 400,
          boxShadow: '0 8px 32px rgba(0,0,0,0.2)',
        }}
      >
        <h1 style={{ fontSize: 24, marginBottom: 4 }}>پرتال دولتی ایندیس</h1>
        <p style={{ fontSize: 13, color: '#666', marginBottom: 24 }}>
          ورود برای کارکنان وزارتخانه
        </p>

        <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>نام کاربری</label>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              required
              dir="ltr"
              style={{ width: '100%', padding: '10px 14px', border: '1px solid #d1d5db', borderRadius: 8, fontSize: 14 }}
            />
          </div>
          <div>
            <label style={{ display: 'block', marginBottom: 6, fontSize: 14 }}>رمز عبور</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
              style={{ width: '100%', padding: '10px 14px', border: '1px solid #d1d5db', borderRadius: 8, fontSize: 14 }}
            />
          </div>
          {error && <p style={{ color: '#c23030', fontSize: 13 }}>{error}</p>}
          <button
            type="submit"
            disabled={loading}
            style={{ background: '#1a56db', color: '#fff', border: 'none', borderRadius: 8, padding: '12px', fontSize: 16, cursor: 'pointer' }}
          >
            {loading ? 'در حال ورود…' : 'ورود'}
          </button>
        </form>

        {import.meta.env.DEV && (
          <button
            onClick={devLogin}
            style={{ marginTop: 16, background: 'transparent', color: '#666', border: '1px solid #d1d5db', borderRadius: 8, padding: '8px', width: '100%', fontSize: 12, cursor: 'pointer' }}
          >
            ورود توسعه‌دهنده
          </button>
        )}
      </div>
    </div>
  )
}
