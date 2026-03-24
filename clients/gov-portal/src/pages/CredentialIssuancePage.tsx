import { useEffect, useState } from 'react'
import { hasRole } from '../hooks/useGovAuth'
import type { GovRole } from '../hooks/useGovAuth'
import FeedbackState from '../components/FeedbackState'
import {
  issuanceStatusBadgeClass,
  issuanceStatusLabel,
  type IssuanceStatus,
} from '../lib/canonicalStatus'
import './Page.css'

interface IssuanceJob {
  job_id: string
  enrollment_id: string
  national_id: string
  full_name: string
  credential_type: string
  status: IssuanceStatus
  issued_at?: string
  credential_id?: string
  error_message?: string
}

interface Props {
  role: GovRole
  token: string
}

const CREDENTIAL_TYPES = [
  { value: 'CitizenshipCredential', label: 'اعتبارنامه شهروندی' },
  { value: 'VoterEligibilityCredential', label: 'اعتبارنامه رأی‌گیری' },
  { value: 'HealthInsuranceCredential', label: 'اعتبارنامه بیمه درمانی' },
  { value: 'AgeRangeCredential', label: 'اعتبارنامه رده سنی' },
  { value: 'ResidencyCredential', label: 'اعتبارنامه اقامت' },
]

export default function CredentialIssuancePage({ role, token }: Props) {
  const [jobs, setJobs] = useState<IssuanceJob[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [pollingId, setPollingId] = useState<number | null>(null)

  const canIssue = hasRole(role, 'operator')

  useEffect(() => {
    loadJobs()
    // Poll every 5 seconds for in-progress jobs
    const id = window.setInterval(() => {
      setJobs(prev => {
        const hasActive = prev.some(j => j.status === 'queued' || j.status === 'issuing')
        if (hasActive) loadJobs()
        return prev
      })
    }, 5000)
    setPollingId(id)
    return () => window.clearInterval(id)
  }, [token]) // eslint-disable-line react-hooks/exhaustive-deps

  // Clear polling on unmount
  useEffect(() => () => { if (pollingId !== null) window.clearInterval(pollingId) }, [pollingId])

  function loadJobs() {
    setLoadError('')
    fetch('/v1/portal/credential-issuance?limit=50', {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(async r => {
        if (!r.ok) {
          throw new Error(`خطا در دریافت وضعیت صدور: ${r.status}`)
        }
        return r.json()
      })
      .then(data => setJobs((data as { jobs: IssuanceJob[] }).jobs ?? []))
      .catch(err => {
        setJobs([])
        setLoadError(String(err))
      })
      .finally(() => setLoading(false))
  }

  function onJobCreated(job: IssuanceJob) {
    setJobs(prev => [job, ...prev])
    setShowForm(false)
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">صدور اعتبارنامه</h1>
        {canIssue && (
          <button type="button" className="btn btn-primary" onClick={() => setShowForm(true)}>
            + صدور جدید
          </button>
        )}
      </div>

      {!canIssue && (
        <p className="role-notice">برای صدور اعتبارنامه به نقش «اپراتور» یا بالاتر نیاز دارید.</p>
      )}

      {loading ? (
        <FeedbackState kind="loading" title="در حال بارگذاری فرایندهای صدور" message="فهرست وضعیت صدور در حال تازه‌سازی است." />
      ) : loadError ? (
        <FeedbackState kind="error" title="دریافت وضعیت صدور ناموفق بود" message={loadError} />
      ) : jobs.length === 0 ? (
        <FeedbackState kind="empty" title="درخواستی برای صدور وجود ندارد" message="هنوز کاری برای صدور اعتبارنامه ثبت نشده است." />
      ) : (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                {['شناسه ثبت‌نام', 'کد ملی', 'نام کامل', 'نوع اعتبارنامه', 'وضعیت', 'تاریخ صدور', 'شناسه اعتبارنامه'].map(h => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {jobs.map(job => (
                <tr key={job.job_id}>
                  <td className="cell-mono-sm">{job.enrollment_id.slice(-8)}</td>
                  <td className="cell-mono">{job.national_id}</td>
                  <td>{job.full_name}</td>
                  <td>
                    <span className="credential-badge">
                      {CREDENTIAL_TYPES.find(c => c.value === job.credential_type)?.label ?? job.credential_type}
                    </span>
                  </td>
                  <td>
                    <span className={`status-badge ${issuanceStatusBadgeClass(job.status)}`}>
                      {issuanceStatusLabel(job.status)}
                    </span>
                    {job.error_message && (
                      <span title={job.error_message} className="warning-icon">⚠️</span>
                    )}
                  </td>
                  <td className="text-muted">
                    {job.issued_at ? new Date(job.issued_at).toLocaleDateString('fa-IR') : '—'}
                  </td>
                  <td className="cell-mono-xs-muted">
                    {job.credential_id ? job.credential_id.slice(-16) : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showForm && (
        <IssueCredentialModal
          token={token}
          onClose={() => setShowForm(false)}
          onCreated={onJobCreated}
        />
      )}
    </div>
  )
}

// ── Issue Credential Modal ────────────────────────────────────────────────────

interface IssueModalProps {
  token: string
  onClose: () => void
  onCreated: (job: IssuanceJob) => void
}

function IssueCredentialModal({ token, onClose, onCreated }: IssueModalProps) {
  const [enrollmentId, setEnrollmentId] = useState('')
  const [credentialType, setCredentialType] = useState(CREDENTIAL_TYPES[0].value)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!enrollmentId.trim()) { setError('شناسه ثبت‌نام الزامی است'); return }
    setLoading(true)
    setError('')
    try {
      const resp = await fetch('/v1/portal/credential-issuance', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          enrollment_id: enrollmentId.trim(),
          credential_type: credentialType,
        }),
      })
      if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`)
      onCreated(await resp.json() as IssuanceJob)
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
      aria-labelledby="issue-modal-title"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="modal">
        <div className="modal-header">
          <h2 id="issue-modal-title" className="modal-title">صدور اعتبارنامه جدید</h2>
          <button type="button" onClick={onClose} className="modal-close" aria-label="بستن">✕</button>
        </div>

        <form onSubmit={e => void handleSubmit(e)} className="modal-form" noValidate>
          <label htmlFor="issue-enrollment-id" className="form-label">
            شناسه ثبت‌نام (enrollment_id)
            <input
              id="issue-enrollment-id"
              type="text"
              value={enrollmentId}
              onChange={e => setEnrollmentId(e.target.value)}
              className="form-input"
              dir="ltr"
              placeholder="ENR-XXXXXXXXXX"
            />
          </label>

          <label htmlFor="issue-credential-type" className="form-label">
            نوع اعتبارنامه
            <select
              id="issue-credential-type"
              value={credentialType}
              onChange={e => setCredentialType(e.target.value)}
              className="form-input"
              title="نوع اعتبارنامه"
            >
              {CREDENTIAL_TYPES.map(ct => (
                <option key={ct.value} value={ct.value}>{ct.label}</option>
              ))}
            </select>
          </label>

          {error && <p className="form-error" role="alert">{error}</p>}

          <div className="modal-actions">
            <button type="button" onClick={onClose} className="btn btn-secondary">انصراف</button>
            <button type="submit" disabled={loading} className="btn btn-primary">
              {loading ? 'در حال صدور…' : 'صدور اعتبارنامه'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
