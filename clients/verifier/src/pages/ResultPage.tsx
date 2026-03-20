import { useLocation, useNavigate } from 'react-router-dom'
import { useEffect } from 'react'

/**
 * Full-screen binary result display per PRD §2.1.3 / FR-013.
 *
 * APPROVED → full-screen GREEN (تأیید شد)
 * DENIED   → full-screen RED   (رد شد)
 *
 * No citizen data is ever shown on this screen.
 * Auto-returns to scan page after 5 seconds.
 */
export default function ResultPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const state = location.state as { valid: boolean; error?: string } | null
  const valid = state?.valid ?? false

  useEffect(() => {
    const timer = setTimeout(() => navigate('/'), 5000)
    return () => clearTimeout(timer)
  }, [navigate])

  return (
    <div
      style={{
        minHeight: '100dvh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: valid ? '#0a6640' : '#8b1a1a',
        color: '#fff',
        cursor: 'pointer',
      }}
      onClick={() => navigate('/')}
    >
      <div style={{ fontSize: 120, lineHeight: 1 }}>
        {valid ? '✅' : '❌'}
      </div>
      <h1 style={{ fontSize: 48, marginTop: 24, fontWeight: 800 }}>
        {valid ? 'تأیید شد' : 'رد شد'}
      </h1>
      <p style={{ marginTop: 16, fontSize: 20, opacity: 0.8 }}>
        {valid ? 'APPROVED' : 'DENIED'}
      </p>
      <p style={{ marginTop: 40, opacity: 0.6, fontSize: 14 }}>
        ۵ ثانیه دیگر به صفحه اسکن باز می‌گردید…
      </p>
    </div>
  )
}
