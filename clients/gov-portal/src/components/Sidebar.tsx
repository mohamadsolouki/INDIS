import { NavLink } from 'react-router-dom'

const items = [
  { to: '/', label: 'داشبورد', icon: '📊', end: true },
  { to: '/electoral', label: 'ماژول انتخابات', icon: 'E', end: false },
  { to: '/bulk-operations', label: 'عملیات گروهی', icon: '📦', end: false },
  { to: '/users', label: 'کاربران پرتال', icon: '👤', end: false },
  { to: '/justice', label: 'عدالت انتقالی', icon: 'J', end: false },
  { to: '/audit', label: 'گزارش حسابرسی', icon: '📋', end: false },
]

export default function Sidebar() {
  function logout() {
    localStorage.removeItem('gov_token')
    window.location.reload()
  }

  return (
    <nav
      style={{
        width: 220,
        background: '#1c2437',
        color: '#fff',
        display: 'flex',
        flexDirection: 'column',
        padding: '24px 0',
        minHeight: '100dvh',
        position: 'sticky',
        top: 0,
      }}
    >
      <div style={{ padding: '0 20px 24px', borderBottom: '1px solid #2d3748' }}>
        <h2 style={{ fontSize: 18, fontWeight: 700 }}>پرتال دولتی</h2>
        <p style={{ fontSize: 12, color: '#a0aec0', marginTop: 4 }}>ایندیس</p>
      </div>

      <div style={{ flex: 1, padding: '16px 8px' }}>
        {items.map(({ to, label, icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            style={({ isActive }) => ({
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              padding: '10px 12px',
              borderRadius: 8,
              color: isActive ? '#fff' : '#a0aec0',
              background: isActive ? '#2d3748' : 'transparent',
              marginBottom: 4,
              fontSize: 14,
              transition: 'background 0.15s',
            })}
          >
            <span style={{ fontSize: 18 }}>{icon}</span>
            {label}
          </NavLink>
        ))}
      </div>

      <div style={{ padding: '16px 12px', borderTop: '1px solid #2d3748' }}>
        <button
          onClick={logout}
          style={{
            background: 'transparent',
            color: '#a0aec0',
            border: 'none',
            fontSize: 14,
            cursor: 'pointer',
            padding: '8px 12px',
          }}
        >
          🚪 خروج
        </button>
      </div>
    </nav>
  )
}
