import { useState } from 'react'
import { useTranslation } from 'react-i18next'

type StatusCode = 'pending' | 'approved' | 'rejected'

interface StatusResult {
  status: StatusCode
  enrollment_id: string
  updated_at?: string
  notes?: string
}

export default function StatusPage() {
  const { t } = useTranslation()
  const [enrollmentId, setEnrollmentId] = useState('')
  const [result, setResult] = useState<StatusResult | null>(null)
  const [checking, setChecking] = useState(false)
  const [error, setError] = useState('')

  async function handleCheck(e: React.FormEvent) {
    e.preventDefault()
    if (!enrollmentId.trim()) return
    setError('')
    setResult(null)
    setChecking(true)
    try {
      const token = localStorage.getItem('diaspora_token')
      const res = await fetch(`/v1/diaspora/enrollment/${enrollmentId}/status`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) throw new Error('not_found')
      const data: StatusResult = await res.json()
      setResult(data)
    } catch {
      // dev fallback
      if (enrollmentId.startsWith('DEV-')) {
        setResult({ status: 'pending', enrollment_id: enrollmentId })
      } else {
        setError(t('errors.network'))
      }
    } finally {
      setChecking(false)
    }
  }

  function statusBadgeClass(status: StatusCode) {
    if (status === 'approved') return 'status-badge status--approved'
    if (status === 'rejected') return 'status-badge status--rejected'
    return 'status-badge status--pending'
  }

  function statusLabel(status: StatusCode) {
    if (status === 'approved') return t('status.status_approved')
    if (status === 'rejected') return t('status.status_rejected')
    return t('status.status_pending')
  }

  return (
    <div>
      <h1 className="page-title">{t('status.title')}</h1>

      <div className="card">
        <form onSubmit={handleCheck} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div className="form-group">
            <label className="form-label">{t('status.enrollment_id')}</label>
            <input
              className="form-input"
              value={enrollmentId}
              onChange={e => setEnrollmentId(e.target.value)}
              placeholder="ENR-XXXXXXXXXX"
            />
          </div>

          {error && <div className="alert alert-error">{error}</div>}

          <button type="submit" className="btn btn-primary" disabled={checking || !enrollmentId.trim()}>
            {checking ? t('status.checking') : t('status.check_btn')}
          </button>
        </form>

        {result && (
          <div style={{ marginTop: '24px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span style={{ fontSize: '14px', color: '#475569' }}>{t('status.enrollment_id')}:</span>
              <strong style={{ fontSize: '14px' }}>{result.enrollment_id}</strong>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span style={{ fontSize: '14px', color: '#475569' }}>وضعیت / Status:</span>
              <span className={statusBadgeClass(result.status)}>
                {statusLabel(result.status)}
              </span>
            </div>
            {result.updated_at && (
              <div style={{ fontSize: '13px', color: '#64748b' }}>
                {result.updated_at}
              </div>
            )}
            {result.notes && (
              <div className="alert alert-error" style={{ marginTop: '8px' }}>
                {result.notes}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
