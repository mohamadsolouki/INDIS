import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

const LOCALES = [
  { code: 'fa', label: 'فارسی' },
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'Français' },
]

export default function Sidebar() {
  const { t, i18n } = useTranslation()

  function logout() {
    localStorage.removeItem('diaspora_token')
    window.location.href = '/'
  }

  return (
    <aside className="sidebar">
      <div className="sidebar-logo">
        <h1>INDIS</h1>
        <p>{t('tagline')}</p>
      </div>

      <nav className="sidebar-nav">
        <NavLink
          to="/enroll"
          className={({ isActive }) => `sidebar-link${isActive ? ' sidebar-link--active' : ''}`}
        >
          📋 {t('nav.enroll')}
        </NavLink>
        <NavLink
          to="/status"
          className={({ isActive }) => `sidebar-link${isActive ? ' sidebar-link--active' : ''}`}
        >
          🔍 {t('nav.status')}
        </NavLink>
        <button
          type="button"
          className="sidebar-link"
          style={{ background: 'none', border: 'none', cursor: 'pointer', width: '100%', textAlign: 'inherit' }}
          onClick={logout}
        >
          🚪 {t('nav.logout')}
        </button>
      </nav>

      <div className="sidebar-lang">
        <select
          value={i18n.language}
          onChange={e => i18n.changeLanguage(e.target.value)}
          title="Select language"
        >
          {LOCALES.map(l => (
            <option key={l.code} value={l.code}>{l.label}</option>
          ))}
        </select>
      </div>
    </aside>
  )
}
