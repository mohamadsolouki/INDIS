import { Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import LoginPage from './pages/LoginPage'
import EnrollmentPage from './pages/EnrollmentPage'
import StatusPage from './pages/StatusPage'
import Sidebar from './components/Sidebar'

function useAuth() {
  return { isAuthenticated: !!localStorage.getItem('diaspora_token') }
}

export default function App() {
  const { isAuthenticated } = useAuth()
  const { i18n } = useTranslation()
  const dir = i18n.language === 'en' || i18n.language === 'fr' ? 'ltr' : 'rtl'

  if (!isAuthenticated) {
    return <LoginPage />
  }

  return (
    <div className="app-shell" dir={dir}>
      <Sidebar />
      <main className="app-main">
        <Routes>
          <Route path="/"         element={<Navigate to="/enroll" replace />} />
          <Route path="/enroll"   element={<EnrollmentPage />} />
          <Route path="/status"   element={<StatusPage />} />
          <Route path="*"         element={<Navigate to="/enroll" replace />} />
        </Routes>
      </main>
    </div>
  )
}
