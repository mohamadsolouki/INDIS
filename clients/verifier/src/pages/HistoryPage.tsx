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
      style={{ direction: 'rtl' }}
    >
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 24 }}>
        <button
          onClick={() => navigate('/')}
          className="verifier-btn"
          style={{ border: 'none', fontSize: 24, lineHeight: 1 }}
          aria-label="بازگشت"
        >
          →
        </button>
        <div>
          <h1 style={{ fontSize: 20, fontWeight: 700, margin: 0 }}>تاریخچه تأییدیه‌ها</h1>
          <p style={{ fontSize: 12, color: '#666', margin: '4px 0 0' }}>
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
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {entries.map((entry) => (
            <div
              key={entry.id}
              style={{
                background: 'var(--verifier-panel)',
                borderRadius: 12,
                padding: '14px 16px',
                display: 'flex',
                alignItems: 'center',
                gap: 14,
                border: '1px solid var(--verifier-line)',
                borderRight: `4px solid ${entry.valid ? '#0f9960' : '#c23030'}`,
              }}
            >
              {/* Result badge */}
              <div
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: '50%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 18,
                  background: entry.valid ? '#0f996020' : '#c2303020',
                  flexShrink: 0,
                }}
              >
                {entry.valid ? '✅' : '❌'}
              </div>

              {/* Details */}
              <div style={{ flex: 1 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <span style={{ fontSize: 14, fontWeight: 600, color: entry.valid ? '#4ade80' : '#f87171' }}>
                    {entry.valid ? 'تأیید شد' : 'رد شد'}
                  </span>
                  <span style={{ fontSize: 11, color: '#666', direction: 'ltr' }}>
                    {new Date(entry.verified_at).toLocaleString('fa-IR')}
                  </span>
                </div>
                <div style={{ fontSize: 12, color: '#888', marginTop: 4, display: 'flex', gap: 12 }}>
                  <span>{entry.predicate || entry.credential_type}</span>
                  <span style={{ color: '#555' }}>•</span>
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
