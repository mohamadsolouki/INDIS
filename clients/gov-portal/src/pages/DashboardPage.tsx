import { useEffect, useState } from 'react'
import './Page.css'

interface Stats {
  total_portal_users: number
  total_bulk_operations: number
  pending_bulk_operations: number
  total_enrollments?: number
  pending_enrollments?: number
  approved_enrollments?: number
  total_credentials_issued?: number
  active_citizens?: number
  revoked_credentials?: number
}

interface ActivityEvent {
  event_id: string
  action: string
  actor_did: string
  service_id: string
  timestamp: string
}

export default function DashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [activity, setActivity] = useState<ActivityEvent[]>([])
  const [error, setError] = useState('')
  const token = localStorage.getItem('gov_token')

  useEffect(() => {
    const headers = { Authorization: `Bearer ${token}` }

    Promise.all([
      fetch('/v1/portal/stats', { headers }).then(r => r.json()).catch(() => null),
      fetch('/v1/audit/events?limit=8', { headers }).then(r => r.json()).catch(() => null),
    ]).then(([statsData, auditData]) => {
      if (statsData) setStats(statsData as Stats)
      if (auditData) setActivity((auditData as { events: ActivityEvent[] }).events ?? [])
    }).catch(err => setError(String(err)))
  }, [token])

  const statCards = stats ? [
    { label: 'شهروندان فعال',       value: stats.active_citizens ?? stats.total_portal_users, icon: '🏛️', delta: null },
    { label: 'ثبت‌نام‌ها',           value: stats.total_enrollments ?? 0,                       icon: '📝', delta: `${stats.pending_enrollments ?? 0} در انتظار`, up: true },
    { label: 'ثبت‌نام تأیید شده',    value: stats.approved_enrollments ?? 0,                    icon: '✅', delta: null },
    { label: 'اعتبارنامه صادرشده',  value: stats.total_credentials_issued ?? 0,                icon: '🎫', delta: null },
    { label: 'اعتبارنامه ابطال‌شده', value: stats.revoked_credentials ?? 0,                     icon: '🚫', delta: null },
    { label: 'عملیات گروهی',        value: stats.total_bulk_operations,                        icon: '📦', delta: `${stats.pending_bulk_operations} در انتظار`, up: false },
    { label: 'کاربران پرتال',       value: stats.total_portal_users,                           icon: '👤', delta: null },
  ] : []

  return (
    <div className="page">
      <h1 className="page-title">داشبورد</h1>

      {error && <p style={{ color: '#c23030', marginBottom: 16 }}>{error}</p>}

      {stats ? (
        <div className="stats-grid">
          {statCards.map(card => (
            <div key={card.label} className="stat-card">
              <div className="stat-icon">{card.icon}</div>
              <div className="stat-value">{card.value.toLocaleString('fa-IR')}</div>
              <div className="stat-label">{card.label}</div>
              {card.delta !== null && (
                <div className={`stat-delta stat-delta--${card.up ? 'up' : 'down'}`}>
                  {card.delta}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : !error && (
        <p className="page-loading">در حال بارگذاری آمار…</p>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24, marginTop: 8 }}>
        {/* Quick actions */}
        <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
          <h2 style={{ fontSize: 16, marginBottom: 16 }}>دسترسی سریع</h2>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {[
              { href: '/enrollments', label: 'بررسی ثبت‌نام‌های در انتظار', icon: '📝' },
              { href: '/issuance', label: 'صدور اعتبارنامه برای تأییدشده‌ها', icon: '🎫' },
              { href: '/bulk-operations', label: 'عملیات گروهی در انتظار', icon: '📦' },
              { href: '/audit', label: 'گزارش حسابرسی', icon: '📋' },
            ].map(item => (
              <a
                key={item.href}
                href={item.href}
                style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  padding: '10px 12px', borderRadius: 8, textDecoration: 'none',
                  color: '#1e293b', fontSize: 14, background: '#f8fafc',
                  border: '1px solid #e2e8f0',
                }}
              >
                <span>{item.icon}</span>
                {item.label}
              </a>
            ))}
          </div>
        </div>

        {/* Recent activity */}
        <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
          <h2 style={{ fontSize: 16, marginBottom: 16 }}>فعالیت اخیر</h2>
          {activity.length === 0 ? (
            <p style={{ fontSize: 13, color: '#94a3b8' }}>رویدادی موجود نیست.</p>
          ) : (
            <div className="activity-feed">
              {activity.map(ev => (
                <div key={ev.event_id} className="activity-item">
                  <span className="activity-time">
                    {new Date(ev.timestamp).toLocaleTimeString('fa-IR', { hour: '2-digit', minute: '2-digit' })}
                  </span>
                  <div style={{ flex: 1 }}>
                    <span className="activity-action">{ev.action}</span>
                    <span style={{ marginRight: 8, color: '#64748b', fontSize: 12 }}>
                      {ev.service_id}
                    </span>
                  </div>
                  <span style={{ fontFamily: 'monospace', fontSize: 10, color: '#94a3b8' }}>
                    {ev.actor_did?.slice(-8)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
