import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import FeedbackState from '../components/FeedbackState'

const GATEWAY = import.meta.env.VITE_GATEWAY_URL ?? 'http://localhost:8080'

interface HistoryEntry {
  id: string
  verified_at: string
  predicate: string
  credential_type: string
  proof_system: string
  valid: boolean
}

/**
 * HistoryPage — displays the last 50 verifications performed by this terminal.
 *
 * Fetches from GET /v1/verifier/{id}/history (paginated, newest first).
 * Displays: timestamp, predicate, credential type, proof system, result.
 * No citizen PII is ever stored or shown — only boolean outcomes.
 */
export default function HistoryPage() {
  const navigate = useNavigate()
  const verifierId = localStorage.getItem('verifier_id') ?? 'dev-verifier'
  const token = localStorage.getItem('verifier_token') ?? ''
  const [entries, setEntries] = useState<HistoryEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch(`${GATEWAY}/v1/verifier/${verifierId}/history?limit=50`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
      .then(r => {
        if (!r.ok) throw new Error(`خطای ${r.status}`)
        return r.json()
      })
      .then(data => setEntries((data as { verifications: HistoryEntry[] }).verifications ?? []))
      .catch(err => setError(String(err)))
      .finally(() => setLoading(false))
  }, [verifierId])

  return (
    <div
      className="verifier-screen"
      dir="rtl"
    >
      {/* Header */}
      <div className="history-header">
        <button
          onClick={() => navigate('/')}
          className="verifier-btn history-back-btn"
          aria-label="بازگشت"
        >
          →
        </button>
        <div>
          <h1 className="history-title">تاریخچه تأییدیه‌ها</h1>
          <p className="history-subtitle">
            پایانه: <span dir="ltr">{verifierId}</span>
          </p>
        </div>
      </div>

      {loading && <FeedbackState kind="loading" title="در حال بارگذاری تاریخچه" message="سوابق پایانه در حال دریافت است." />}
      {error && <FeedbackState kind="error" title="بارگذاری تاریخچه ناموفق بود" message={error} />}

      {!loading && !error && entries.length === 0 && (
        <FeedbackState kind="empty" title="تاریخچه‌ای ثبت نشده است" message="هنوز هیچ رویداد تأییدی در این پایانه ذخیره نشده است." />
      )}

      {!loading && !error && entries.length > 0 && (
        <div className="history-list">
          {entries.map((entry) => (
            <div
              key={entry.id}
              className={`history-item ${entry.valid ? 'history-item--ok' : 'history-item--fail'}`}
            >
              {/* Result badge */}
              <div
                className={`history-badge ${entry.valid ? 'history-badge--ok' : 'history-badge--fail'}`}
              >
                {entry.valid ? '✅' : '❌'}
              </div>

              {/* Details */}
              <div className="history-content">
                <div className="history-row">
                  <span className={`history-status ${entry.valid ? 'history-status--ok' : 'history-status--fail'}`}>
                    {entry.valid ? 'تأیید شد' : 'رد شد'}
                  </span>
                  <span className="history-time">
                    {new Date(entry.verified_at).toLocaleString('fa-IR')}
                  </span>
                </div>
                <div className="history-meta">
                  <span>{entry.predicate || entry.credential_type}</span>
                  <span className="history-dot">•</span>
                  <span>{entry.proof_system}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
