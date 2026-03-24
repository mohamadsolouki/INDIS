import { useLocation, useNavigate } from 'react-router-dom'
import { useEffect } from 'react'
import {
  verificationStatusFromBoolean,
  verificationStatusPresentation,
} from '../lib/canonicalStatus'

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
  const status = verificationStatusFromBoolean(valid)
  const presentation = verificationStatusPresentation(status)

  useEffect(() => {
    const timer = setTimeout(() => navigate('/'), 5000)
    return () => clearTimeout(timer)
  }, [navigate])

  return (
    <div
      className={`result-screen ${presentation.tone === 'ok' ? 'result-screen--ok' : 'result-screen--fail'}`}
      onClick={() => navigate('/')}
    >
      <div className="result-icon">
        {presentation.tone === 'ok' ? '✅' : '❌'}
      </div>
      <h1 className="result-title">
        {presentation.labelFa}
      </h1>
      <p className="result-subtitle">
        {presentation.labelEn}
      </p>
      <p className="result-return-note">
        ۵ ثانیه دیگر به صفحه اسکن باز می‌گردید…
      </p>
    </div>
  )
}
