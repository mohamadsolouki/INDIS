import { useEffect, useState } from 'react'

interface AuditEvent {
  id: string
  event_type: string
  actor_did: string
  target_did: string
  description: string
  occurred_at: string
}

export default function AuditPage() {
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)
  const token = localStorage.getItem('gov_token')

  useEffect(() => {
    fetch('/v1/audit/events?limit=50', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.json())
      .then(data => setEvents((data as { events: AuditEvent[] }).events ?? []))
      .finally(() => setLoading(false))
  }, [token])

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <h1 style={{ fontSize: 24 }}>گزارش حسابرسی</h1>
        <span style={{ fontSize: 12, color: '#666', background: '#f0f0f0', padding: '4px 12px', borderRadius: 20 }}>
          فقط‌خواندنی
        </span>
      </div>

      {loading ? (
        <p style={{ color: '#666' }}>در حال بارگذاری…</p>
      ) : events.length === 0 ? (
        <p style={{ color: '#666' }}>رویدادی یافت نشد.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {events.map(ev => (
            <div
              key={ev.id}
              style={{ background: '#fff', borderRadius: 8, padding: '12px 16px', boxShadow: '0 1px 4px rgba(0,0,0,0.05)', display: 'flex', gap: 16, alignItems: 'flex-start' }}
            >
              <div style={{ fontSize: 11, color: '#999', whiteSpace: 'nowrap', marginTop: 2 }}>
                {new Date(ev.occurred_at).toLocaleString('fa-IR')}
              </div>
              <div style={{ flex: 1 }}>
                <span
                  style={{
                    fontSize: 11,
                    padding: '2px 8px',
                    borderRadius: 4,
                    background: '#e8f0fe',
                    color: '#1a56db',
                    marginLeft: 8,
                  }}
                >
                  {ev.event_type}
                </span>
                <span style={{ fontSize: 13 }}>{ev.description}</span>
              </div>
              <div style={{ fontSize: 11, color: '#999', fontFamily: 'monospace', direction: 'ltr' }}>
                {ev.actor_did?.slice(-12)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
