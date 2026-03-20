import { NavLink } from 'react-router-dom'
import type { GovRole } from '../hooks/useGovAuth'
import './Sidebar.css'

interface SidebarProps {
  role?: GovRole
  ministry?: string
}

const items = [
  { to: '/', label: 'داشبورد', icon: '📊', end: true },
  { to: '/enrollments', label: 'بررسی ثبت‌نام‌ها', icon: '📝', end: false },
  { to: '/issuance', label: 'صدور اعتبارنامه', icon: '🎫', end: false },
  { to: '/bulk-operations', label: 'عملیات گروهی', icon: '📦', end: false },
  { to: '/users', label: 'کاربران پرتال', icon: '👤', end: false },
  { to: '/electoral', label: 'ماژول انتخابات', icon: '🗳️', end: false },
  { to: '/justice', label: 'عدالت انتقالی', icon: '⚖️', end: false },
  { to: '/audit', label: 'گزارش حسابرسی', icon: '📋', end: false },
]

const ROLE_LABELS: Record<GovRole, string> = {
  viewer: 'بازدیدکننده',
  operator: 'اپراتور',
  senior: 'ارشد',
  admin: 'مدیر',
}

export default function Sidebar({ role = 'viewer', ministry = '' }: SidebarProps) {
  function logout() {
    localStorage.removeItem('gov_token')
    window.location.reload()
  }

  return (
    <nav className="sidebar">
      <div className="sidebar-header">
        <h2 className="sidebar-title">پرتال دولتی</h2>
        <p className="sidebar-subtitle">ایندیس</p>
        {ministry && <p className="sidebar-ministry">{ministry}</p>}
        <span className="sidebar-role-badge">{ROLE_LABELS[role]}</span>
      </div>

      <div className="sidebar-nav">
        {items.map(({ to, label, icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              ['sidebar-link', isActive ? 'sidebar-link--active' : ''].join(' ').trim()
            }
          >
            <span className="sidebar-link-icon">{icon}</span>
            {label}
          </NavLink>
        ))}
      </div>

      <div className="sidebar-footer">
        <button onClick={logout} className="sidebar-logout-btn">
          🚪 خروج
        </button>
      </div>
    </nav>
  )
}
