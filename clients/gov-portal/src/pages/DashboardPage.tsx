import { useEffect, useState } from 'react'

interface Stats {
  total_portal_users: number
  total_bulk_operations: number
  pending_bulk_operations: number
}

export default function DashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    const token = localStorage.getItem('gov_token')
    fetch('/v1/portal/stats', {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => setStats(data as Stats))
      .catch(err => setError(String(err)))
  }, [])

  return (
    <div>
      <h1 style={{ fontSize: 24, marginBottom: 24 }}>داشبورد</h1>

      {error && <p style={{ color: '#c23030' }}>{error}</p>}

      {stats && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16, marginBottom: 32 }}>
          <StatCard label="کاربران پرتال" value={stats.total_portal_users} icon="👤" />
          <StatCard label="عملیات گروهی" value={stats.total_bulk_operations} icon="📦" />
          <StatCard label="عملیات در انتظار" value={stats.pending_bulk_operations} icon="⏳" />
        </div>
      )}

      <div style={{ background: '#fff', borderRadius: 12, padding: 20, boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
        <h2 style={{ fontSize: 16, marginBottom: 12 }}>راهنمای سریع</h2>
        <ul style={{ paddingRight: 20, color: '#555', fontSize: 14, lineHeight: 2 }}>
          <li>عملیات گروهی — صدور یا ابطال انبوه اعتبارنامه</li>
          <li>کاربران پرتال — مدیریت دسترسی‌های وزارتخانه</li>
          <li>گزارش حسابرسی — بررسی تاریخچه عملیات (فقط‌خواندنی)</li>
        </ul>
      </div>
    </div>
  )
}

function StatCard({ label, value, icon }: { label: string; value: number; icon: string }) {
  return (
    <div
      style={{
        background: '#fff',
        borderRadius: 12,
        padding: 20,
        boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
      }}
    >
      <div style={{ fontSize: 28, marginBottom: 8 }}>{icon}</div>
      <div style={{ fontSize: 28, fontWeight: 700 }}>{value.toLocaleString('fa-IR')}</div>
      <div style={{ fontSize: 13, color: '#666', marginTop: 4 }}>{label}</div>
    </div>
  )
}
