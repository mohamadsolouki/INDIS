import { useEffect, useState } from 'react'

interface BulkOp {
  id: string
  operation_type: string
  ministry: string
  status: string
  requested_by: string
  created_at: string
}

export default function BulkOperationsPage() {
  const [ops, setOps] = useState<BulkOp[]>([])
  const [loading, setLoading] = useState(true)
  const token = localStorage.getItem('gov_token')

  useEffect(() => {
    fetch('/v1/portal/bulk-ops', {
      headers: { Authorization: `Bearer ${token}` },
    })
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
    const updated = (await resp.json()) as BulkOp
    setOps(prev => prev.map(o => (o.id === id ? updated : o)))
  }

  const statusColor: Record<string, string> = {
    pending: '#b45309',
    executing: '#1a56db',
    completed: '#0f9960',
    failed: '#c23030',
    approved: '#0f9960', // backward compat (older UI states)
    rejected: '#c23030', // backward compat (older UI states)
    processing: '#1a56db', // backward compat (older UI states)
  }

  return (
    <div>
      <h1 style={{ fontSize: 24, marginBottom: 24 }}>عملیات گروهی</h1>

      {loading ? (
        <p style={{ color: '#666' }}>در حال بارگذاری…</p>
      ) : ops.length === 0 ? (
        <p style={{ color: '#666' }}>هیچ عملیاتی یافت نشد.</p>
      ) : (
        <div style={{ background: '#fff', borderRadius: 12, overflow: 'hidden', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
            <thead>
              <tr style={{ background: '#f8fafc', borderBottom: '1px solid #e2e8f0' }}>
                {['نوع عملیات', 'وزارتخانه', 'درخواست‌کننده', 'وضعیت', 'تاریخ', 'اقدام'].map(h => (
                  <th key={h} style={{ padding: '12px 16px', textAlign: 'right', fontWeight: 600, color: '#555' }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {ops.map(op => (
                <tr key={op.id} style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <td style={{ padding: '12px 16px' }}>{op.operation_type}</td>
                  <td style={{ padding: '12px 16px' }}>{op.ministry}</td>
                  <td style={{ padding: '12px 16px', fontSize: 12, color: '#666' }}>{op.requested_by}</td>
                  <td style={{ padding: '12px 16px' }}>
                    <span
                      style={{
                        padding: '2px 10px',
                        borderRadius: 20,
                        fontSize: 12,
                        background: (statusColor[op.status] ?? '#666') + '20',
                        color: statusColor[op.status] ?? '#666',
                      }}
                    >
                      {op.status}
                    </span>
                  </td>
                  <td style={{ padding: '12px 16px', fontSize: 12, color: '#666' }}>
                    {new Date(op.created_at).toLocaleDateString('fa-IR')}
                  </td>
                  <td style={{ padding: '12px 16px' }}>
                    {op.status === 'pending' && (
                      <button
                        onClick={() => approve(op.id)}
                        style={{ background: '#0f9960', color: '#fff', border: 'none', borderRadius: 6, padding: '6px 14px', fontSize: 12, cursor: 'pointer' }}
                      >
                        تأیید
                      </button>
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
