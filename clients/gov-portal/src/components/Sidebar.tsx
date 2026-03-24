import { NavLink } from 'react-router-dom'
import type { GovRole } from '../hooks/useGovAuth'
import './Sidebar.css'

interface SidebarProps {
  role?: GovRole
  ministry?: string
}

const sections = [
  {
    title: 'کارهای روزانه',
    items: [
      { to: '/', label: 'نمای کلی عملیات', icon: '📊', end: true },
      { to: '/enrollments', label: 'بررسی ثبت‌نام شهروندان', icon: '📝', end: false },
      { to: '/issuance', label: 'صدور اعتبارنامه', icon: '🎫', end: false },
    ],
  },
  {
    title: 'مدیریت فرایند',
    items: [
      { to: '/bulk-operations', label: 'عملیات گروهی پرونده‌ها', icon: '📦', end: false },
      { to: '/users', label: 'مدیریت کاربران پرتال', icon: '👤', end: false },
    ],
  },
  {
    title: 'حاکمیت و نظارت',
    items: [
      { to: '/electoral', label: 'نظارت فرایندهای انتخاباتی', icon: '🗳️', end: false },
      { to: '/justice', label: 'پرونده‌های عدالت انتقالی', icon: '⚖️', end: false },
      { to: '/audit', label: 'گزارش‌های حسابرسی', icon: '📋', end: false },
    ],
  },
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
    <nav className="sidebar" aria-label="ناوبری وظیفه‌محور پرتال دولتی">
      <div className="sidebar-header">
        <h2 className="sidebar-title">پرتال دولتی</h2>
        <p className="sidebar-subtitle">ایندیس</p>
        {ministry && <p className="sidebar-ministry">{ministry}</p>}
        <span className="sidebar-role-badge">{ROLE_LABELS[role]}</span>
      </div>

      <div className="sidebar-nav">
        {sections.map(section => (
          <section key={section.title} className="sidebar-section" aria-label={section.title}>
            <h3 className="sidebar-section-title">{section.title}</h3>
            {section.items.map(({ to, label, icon, end }) => (
              <NavLink
                key={to}
                to={to}
                end={end}
                className={({ isActive }) =>
                  ['sidebar-link', isActive ? 'sidebar-link--active' : ''].join(' ').trim()
                }
              >
                <span className="sidebar-link-icon" aria-hidden="true">{icon}</span>
                {label}
              </NavLink>
            ))}
          </section>
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
