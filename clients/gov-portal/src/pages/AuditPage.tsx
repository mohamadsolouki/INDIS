import { useEffect, useState, useCallback } from 'react'
import './Page.css'

interface AuditEvent {
  event_id: string
  category: number
  action: string
  actor_did: string
  subject_did: string
  resource_id: string
  service_id: string
  timestamp: string
}

const CATEGORIES: Record<number, string> = {
  0: 'نامشخص',
  1: 'هویت',
  2: 'اعتبارنامه',
  3: 'ثبت‌نام',
  4: 'تأیید',
  5: 'مدیریت',
  6: 'حسابرسی',
  7: 'بیومتریک',
  8: 'انتخابات',
  9: 'عدالت',
}

const PAGE_SIZE = 25

export default function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [catFilter, setCatFilter] = useState<string>('all')
  const [actionFilter, setActionFilter] = useState('')
  const token = localStorage.getItem('gov_token')

  const load = useCallback((reset: boolean) => {
    setLoading(true)
    const offset = reset ? 0 : page * PAGE_SIZE
    const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(offset) })
    if (catFilter !== 'all') params.set('category', catFilter)
    if (actionFilter.trim()) params.set('action', actionFilter.trim())

    fetch(`/v1/audit/events?${params.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => {
        const evs = (data as { events: AuditEvent[] }).events ?? []
        setEvents(prev => reset ? evs : [...prev, ...evs])
        setHasMore(evs.length === PAGE_SIZE)
      })
      .finally(() => setLoading(false))
  }, [token, page, catFilter, actionFilter]) // eslint-disable-line react-hooks/exhaustive-deps

  // Reset on filter change
  useEffect(() => {
    setPage(0)
    setEvents([])
    load(true)
  }, [catFilter, actionFilter]) // eslint-disable-line react-hooks/exhaustive-deps

  // Load more
  useEffect(() => {
    if (page > 0) load(false)
  }, [page]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">گزارش حسابرسی</h1>
        <span style={{ fontSize: 12, color: '#666', background: '#f0f0f0', padding: '4px 12px', borderRadius: 20 }}>
          فقط‌خواندنی
        </span>
      </div>

      <div style={{ display: 'flex', gap: 12, marginBottom: 16, alignItems: 'center', flexWrap: 'wrap' }}>
        <select
          value={catFilter}
          onChange={e => setCatFilter(e.target.value)}
          className="role-select"
          title="دسته‌بندی"
          style={{ minWidth: 140 }}
        >
          <option value="all">همه دسته‌ها</option>
          {Object.entries(CATEGORIES).map(([k, v]) => (
            <option key={k} value={k}>{v}</option>
          ))}
        </select>
        <input
          className="search-input"
          placeholder="فیلتر بر اساس action…"
          value={actionFilter}
          onChange={e => setActionFilter(e.target.value)}
          style={{ minWidth: 220 }}
          dir="ltr"
        />
      </div>

      {events.length === 0 && loading ? (
        <p className="page-loading">در حال بارگذاری…</p>
      ) : events.length === 0 ? (
        <p className="page-empty">رویدادی یافت نشد.</p>
      ) : (
        <>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {events.map(ev => (
              <div
                key={ev.event_id}
                style={{
                  background: '#fff',
                  borderRadius: 8,
                  padding: '10px 16px',
                  boxShadow: '0 1px 4px rgba(0,0,0,0.05)',
                  display: 'grid',
                  gridTemplateColumns: '90px 1fr 120px 80px',
                  gap: 12,
                  alignItems: 'start',
                  fontSize: 13,
                }}
              >
                <div style={{ color: '#999', fontSize: 11, paddingTop: 2 }}>
                  {new Date(ev.timestamp).toLocaleString('fa-IR', {
                    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
                  })}
                </div>
                <div>
                  <span className="activity-action">{ev.action}</span>
                  <span style={{ marginRight: 8, color: '#555', fontSize: 12 }}>
                    {ev.service_id}{ev.resource_id ? ` • ${ev.resource_id}` : ''}
                  </span>
                  {ev.subject_did && (
                    <span style={{ display: 'block', fontFamily: 'monospace', fontSize: 10, color: '#94a3b8', marginTop: 2 }}>
                      subject: {ev.subject_did.slice(-16)}
                    </span>
                  )}
                </div>
                <div style={{ fontFamily: 'monospace', fontSize: 10, color: '#94a3b8', direction: 'ltr', paddingTop: 2 }}>
                  {ev.actor_did?.slice(-12)}
                </div>
                <div>
                  <span style={{
                    fontSize: 10,
                    padding: '2px 6px',
                    borderRadius: 4,
                    background: '#f0f0f0',
                    color: '#555',
                  }}>
                    {CATEGORIES[ev.category] ?? `cat-${ev.category}`}
                  </span>
                </div>
              </div>
            ))}
          </div>

          {hasMore && (
            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <button
                type="button"
                className="btn btn-secondary"
                disabled={loading}
                onClick={() => setPage(p => p + 1)}
              >
                {loading ? 'در حال بارگذاری…' : 'بارگذاری بیشتر'}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
