import { NavLink } from 'react-router-dom'

const items = [
  { to: '/', label: 'خانه', icon: '🏠' },
  { to: '/wallet', label: 'کیف‌پول', icon: '🪪' },
  { to: '/enrollment', label: 'ثبت‌نام', icon: '📝' },
  { to: '/verify', label: 'تأیید', icon: '✅' },
  { to: '/settings', label: 'تنظیمات', icon: '⚙️' },
]

export default function NavBar() {
  return (
    <nav
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        right: 0,
        height: 'var(--nav-height)',
        background: 'var(--color-surface)',
        borderTop: '1px solid var(--color-border)',
        display: 'flex',
        alignItems: 'stretch',
        zIndex: 100,
      }}
    >
      {items.map(({ to, label, icon }) => (
        <NavLink
          key={to}
          to={to}
          end={to === '/'}
          style={({ isActive }) => ({
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 2,
            color: isActive ? 'var(--color-primary)' : 'var(--color-text-muted)',
            fontSize: 11,
            fontWeight: isActive ? 600 : 400,
          })}
        >
          <span style={{ fontSize: 20 }}>{icon}</span>
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
