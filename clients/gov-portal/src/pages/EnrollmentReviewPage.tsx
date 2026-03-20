import { useEffect, useState } from 'react'
import { hasRole } from '../hooks/useGovAuth'
import type { GovRole } from '../hooks/useGovAuth'
import './Page.css'

interface EnrollmentApplication {
  enrollment_id: string
  national_id: string
  full_name: string
  pathway: 'standard' | 'enhanced' | 'social'
  status: 'pending' | 'under_review' | 'approved' | 'rejected' | 'requires_biometric'
  submitted_at: string
  ministry_reviewer?: string
  notes?: string
}

interface Props {
  role: GovRole
  token: string
}

function statusLabel(s: EnrollmentApplication['status']): string {
  const map: Record<string, string> = {
    pending: 'در انتظار',
    under_review: 'در حال بررسی',
    approved: 'تأیید شده',
    rejected: 'رد شده',
    requires_biometric: 'نیاز به بیومتریک',
  }
  return map[s] ?? s
}

function statusBadgeClass(s: EnrollmentApplication['status']): string {
  const map: Record<string, string> = {
    pending: 'status-badge--warning',
    under_review: 'status-badge--info',
    approved: 'status-badge--success',
    rejected: 'status-badge--error',
    requires_biometric: 'status-badge--info',
  }
  return map[s] ?? 'status-badge--default'
}

function pathwayLabel(p: EnrollmentApplication['pathway']): string {
  return { standard: 'استاندارد', enhanced: 'پیشرفته', social: 'اجتماعی' }[p] ?? p
}

export default function EnrollmentReviewPage({ role, token }: Props) {
  const [apps, setApps] = useState<EnrollmentApplication[]>([])
  const [loading, setLoading] = useState(true)
  const [selected, setSelected] = useState<EnrollmentApplication | null>(null)
  const [filter, setFilter] = useState<string>('pending')
  const [searchNid, setSearchNid] = useState('')

  const canReview = hasRole(role, 'operator')
  const canOverride = hasRole(role, 'admin')

  useEffect(() => {
    setLoading(true)
    const params = new URLSearchParams({ limit: '100' })
    if (filter !== 'all') params.set('status', filter)
    fetch(`/v1/portal/enrollments?${params.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => setApps((data as { enrollments: EnrollmentApplication[] }).enrollments ?? []))
      .catch(() => setApps([]))
      .finally(() => setLoading(false))
  }, [token, filter])

  const visible = searchNid.trim()
    ? apps.filter(a => a.national_id.includes(searchNid.trim()) || a.full_name.includes(searchNid.trim()))
    : apps

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">بررسی ثبت‌نام‌ها</h1>
        <span style={{ fontSize: 12, color: '#666' }}>فقط خواندنی برای نقش بازدیدکننده</span>
      </div>

      <div style={{ display: 'flex', gap: 12, marginBottom: 16, alignItems: 'center', flexWrap: 'wrap' }}>
        <select
          value={filter}
          onChange={e => setFilter(e.target.value)}
          className="role-select"
          title="فیلتر وضعیت"
          style={{ minWidth: 160 }}
        >
          {['all', 'pending', 'under_review', 'approved', 'rejected', 'requires_biometric'].map(s => (
            <option key={s} value={s}>{s === 'all' ? 'همه' : statusLabel(s as EnrollmentApplication['status'])}</option>
          ))}
        </select>
        <input
          className="search-input"
          placeholder="جستجو با کد ملی یا نام…"
          value={searchNid}
          onChange={e => setSearchNid(e.target.value)}
          style={{ flex: 1, maxWidth: 280 }}
        />
        <span style={{ fontSize: 13, color: '#666' }}>{visible.length} مورد</span>
      </div>

      {loading ? (
        <p className="page-loading">در حال بارگذاری…</p>
      ) : visible.length === 0 ? (
        <p className="page-empty">هیچ ثبت‌نامی با این فیلتر یافت نشد.</p>
      ) : (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                {['کد ملی', 'نام کامل', 'مسیر', 'وضعیت', 'تاریخ ارسال', 'اقدام'].map(h => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {visible.map(app => (
                <tr key={app.enrollment_id}>
                  <td style={{ fontFamily: 'monospace', direction: 'ltr', fontSize: 13 }}>
                    {app.national_id}
                  </td>
                  <td>{app.full_name}</td>
                  <td>
                    <span className="pathway-badge">{pathwayLabel(app.pathway)}</span>
                  </td>
                  <td>
                    <span className={`status-badge ${statusBadgeClass(app.status)}`}>
                      {statusLabel(app.status)}
                    </span>
                  </td>
                  <td className="text-muted">
                    {new Date(app.submitted_at).toLocaleDateString('fa-IR')}
                  </td>
                  <td>
                    <button
                      type="button"
                      className="btn btn-sm"
                      style={{ background: '#f1f5f9', color: '#1e293b', border: '1px solid #e2e8f0' }}
                      onClick={() => setSelected(app)}
                    >
                      مشاهده
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selected && (
        <ReviewModal
          app={selected}
          token={token}
          canReview={canReview}
          canOverride={canOverride}
          onClose={() => setSelected(null)}
          onUpdated={(updated) => {
            setApps(prev => prev.map(a => a.enrollment_id === updated.enrollment_id ? updated : a))
            setSelected(null)
          }}
        />
      )}
    </div>
  )
}

// ── Review Modal ──────────────────────────────────────────────────────────────

interface ReviewModalProps {
  app: EnrollmentApplication
  token: string
  canReview: boolean
  canOverride: boolean
  onClose: () => void
  onUpdated: (app: EnrollmentApplication) => void
}

function ReviewModal({ app, token, canReview, canOverride, onClose, onUpdated }: ReviewModalProps) {
  const [notes, setNotes] = useState(app.notes ?? '')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function decide(decision: 'approve' | 'reject') {
    setLoading(true)
    setError('')
    try {
      const resp = await fetch(`/v1/portal/enrollments/${app.enrollment_id}/${decision}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ notes: notes.trim() }),
      })
      if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`)
      onUpdated({
        ...app,
        status: decision === 'approve' ? 'approved' : 'rejected',
        notes: notes.trim() || undefined,
      })
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  async function requestBiometric() {
    setLoading(true)
    setError('')
    try {
      const resp = await fetch(`/v1/portal/enrollments/${app.enrollment_id}/request-biometric`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`)
      onUpdated({ ...app, status: 'requires_biometric' })
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="review-modal-title"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="modal" style={{ maxWidth: 520 }}>
        <div className="modal-header">
          <h2 id="review-modal-title" className="modal-title">بررسی ثبت‌نام</h2>
          <button type="button" onClick={onClose} className="modal-close" aria-label="بستن">✕</button>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: '0 0 16px' }}>
          <Row label="شناسه ثبت‌نام" value={app.enrollment_id} mono />
          <Row label="کد ملی" value={app.national_id} mono />
          <Row label="نام کامل" value={app.full_name} />
          <Row label="مسیر" value={{ standard: 'استاندارد', enhanced: 'پیشرفته', social: 'اجتماعی' }[app.pathway]} />
          <Row label="وضعیت فعلی" value={app.status} />
          <Row label="تاریخ ارسال" value={new Date(app.submitted_at).toLocaleString('fa-IR')} />
          {app.ministry_reviewer && <Row label="بررسی‌کننده" value={app.ministry_reviewer} />}

          {canReview && (
            <div>
              <label className="form-label" htmlFor="review-notes">
                یادداشت بررسی
              </label>
              <textarea
                id="review-notes"
                value={notes}
                onChange={e => setNotes(e.target.value)}
                rows={3}
                className="form-input"
                style={{ resize: 'vertical' }}
                placeholder="دلیل تأیید یا رد (اختیاری)"
              />
            </div>
          )}

          {error && <p className="form-error" role="alert">{error}</p>}
        </div>

        <div className="modal-actions">
          <button type="button" onClick={onClose} className="btn btn-secondary">بستن</button>
          {canReview && app.status === 'pending' && (
            <>
              <button
                type="button"
                disabled={loading}
                className="btn"
                style={{ background: '#dc2626', color: '#fff' }}
                onClick={() => void decide('reject')}
              >
                رد
              </button>
              <button
                type="button"
                disabled={loading}
                className="btn"
                style={{ background: '#0f9960', color: '#fff' }}
                onClick={() => void requestBiometric()}
              >
                درخواست بیومتریک
              </button>
              <button
                type="button"
                disabled={loading}
                className="btn btn-primary"
                onClick={() => void decide('approve')}
              >
                {loading ? '…' : 'تأیید'}
              </button>
            </>
          )}
          {canOverride && app.status === 'rejected' && (
            <button
              type="button"
              disabled={loading}
              className="btn btn-primary"
              onClick={() => void decide('approve')}
            >
              بازبینی و تأیید (مدیر)
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function Row({ label, value, mono }: { label: string; value: string | undefined; mono?: boolean }) {
  return (
    <div style={{ display: 'flex', gap: 12, fontSize: 14, alignItems: 'baseline' }}>
      <span style={{ color: '#64748b', minWidth: 120, flexShrink: 0 }}>{label}:</span>
      <span style={{ fontFamily: mono ? 'monospace' : 'inherit', wordBreak: 'break-all' }}>{value ?? '—'}</span>
    </div>
  )
}
