import { Link } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useCredentials } from '../hooks/useCredentials'

export default function HomePage() {
  const { did } = useAuth()
  const { credentials } = useCredentials()

  const shortDid = did ? did.slice(-12) : ''

  return (
    <div style={{ padding: 20 }}>
      <header style={{ marginBottom: 24 }}>
        <h2 style={{ fontSize: 22 }}>خوش آمدید</h2>
        <p className="text-muted" dir="ltr" style={{ fontSize: 12, marginTop: 4 }}>
          …{shortDid}
        </p>
      </header>

      {/* Quick-action cards */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 12,
          marginBottom: 24,
        }}
      >
        <QuickCard
          to="/wallet"
          icon="🪪"
          label="اعتبارنامه‌ها"
          value={String(credentials.length)}
          subtitle="موجود در کیف‌پول"
        />
        <QuickCard
          to="/verify"
          icon="✅"
          label="تأیید هویت"
          value="ZK Proof"
          subtitle="بدون افشای اطلاعات"
        />
        <QuickCard
          to="/enrollment"
          icon="📝"
          label="ثبت‌نام"
          value="شروع"
          subtitle="افزودن هویت جدید"
        />
        <QuickCard
          to="/settings"
          icon="⚙️"
          label="تنظیمات"
          value="حریم خصوصی"
          subtitle="مدیریت داده‌ها"
        />
      </div>

      <div className="card">
        <h3 style={{ fontSize: 15, marginBottom: 8 }}>وضعیت اتصال</h3>
        <p className="text-muted" style={{ fontSize: 13 }}>
          {navigator.onLine
            ? '🟢 متصل — داده‌ها به‌روزرسانی می‌شوند'
            : '🟡 آفلاین — اعتبارنامه‌های محلی در دسترس هستند (تا ۷۲ ساعت)'}
        </p>
      </div>
    </div>
  )
}

function QuickCard({
  to,
  icon,
  label,
  value,
  subtitle,
}: {
  to: string
  icon: string
  label: string
  value: string
  subtitle: string
}) {
  return (
    <Link to={to}>
      <div
        className="card"
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 6,
          cursor: 'pointer',
          transition: 'box-shadow 0.15s',
        }}
      >
        <span style={{ fontSize: 28 }}>{icon}</span>
        <span style={{ fontSize: 13, color: 'var(--color-text-muted)' }}>{label}</span>
        <span style={{ fontSize: 18, fontWeight: 700 }}>{value}</span>
        <span style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>{subtitle}</span>
      </div>
    </Link>
  )
}
