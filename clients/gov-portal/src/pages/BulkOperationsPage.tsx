import { useEffect, useState } from 'react'
import { hasRole } from '../hooks/useGovAuth'
import type { GovRole } from '../hooks/useGovAuth'
import './Page.css'

interface BulkOp {
  id: string
  operation_type: string
  ministry: string
  status: string
  requested_by: string
  created_at: string
  result_summary?: string
}

interface Props {
  role: GovRole
  token: string
}

/** Maps API status values to CSS modifier classes defined in Page.css. */
function statusClass(status: string): string {
  const map: Record<string, string> = {
    pending:    'status-badge--warning',
    executing:  'status-badge--info',
    processing: 'status-badge--info',
    completed:  'status-badge--success',
    approved:   'status-badge--success',
    failed:     'status-badge--error',
    rejected:   'status-badge--error',
  }
  return map[status] ?? 'status-badge--default'
}

export default function BulkOperationsPage({ role, token }: Props) {
  const [ops, setOps]       = useState<BulkOp[]>([])
  const [loading, setLoading] = useState(true)
  const canApprove            = hasRole(role, 'operator')

  useEffect(() => {
    fetch('/v1/portal/bulk-ops', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(data => setOps((data as { bulk_operations: BulkOp[] }).bulk_operations ?? []))
      .finally(() => setLoading(false))
  }, [token])

  async function approve(id: string) {
    const resp = await fetch(`/v1/portal/bulk-ops/${id}/approve`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!resp.ok) return
    setOps(prev => prev.map(o => o.id === id ? { ...o, status: 'approved' } : o))
  }

  return (
    <div className="page">
      <h1 className="page-title">عملیات گروهی</h1>

      {loading ? (
        <p className="page-loading">در حال بارگذاری…</p>
      ) : ops.length === 0 ? (
        <p className="page-empty">هیچ عملیاتی یافت نشد.</p>
      ) : (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                {['نوع عملیات', 'وزارتخانه', 'درخواست‌کننده', 'وضعیت', 'نتیجه', 'تاریخ', 'اقدام'].map(h => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {ops.map(op => (
                <tr key={op.id}>
                  <td>{op.operation_type}</td>
                  <td>{op.ministry}</td>
                  <td className="text-muted">{op.requested_by}</td>
                  <td>
                    <span className={`status-badge ${statusClass(op.status)}`}>
                      {op.status}
                    </span>
                  </td>
                  <td className="text-muted">
                    {op.result_summary ?? '—'}
                  </td>
                  <td className="text-muted">
                    {new Date(op.created_at).toLocaleDateString('fa-IR')}
                  </td>
                  <td>
                    {op.status === 'pending' && canApprove && (
                      <button
                        type="button"
                        className="btn btn-success btn-sm"
                        onClick={() => approve(op.id)}
                      >
                        تأیید
                      </button>
                    )}
                    {op.status === 'pending' && !canApprove && (
                      <span className="text-muted" title="نیاز به نقش اپراتور">—</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
