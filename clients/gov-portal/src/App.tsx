import { Routes, Route, Navigate } from 'react-router-dom'
import { useGovAuth } from './hooks/useGovAuth'
import './App.css'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import BulkOperationsPage from './pages/BulkOperationsPage'
import UsersPage from './pages/UsersPage'
import AuditPage from './pages/AuditPage'
import ElectoralAuthorityPage from './pages/ElectoralAuthorityPage'
import TransitionalJusticePage from './pages/TransitionalJusticePage'
import EnrollmentReviewPage from './pages/EnrollmentReviewPage'
import CredentialIssuancePage from './pages/CredentialIssuancePage'
import Sidebar from './components/Sidebar'

export default function App() {
  const auth = useGovAuth()

  if (!auth.isAuthenticated) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <div className="app-shell">
      <Sidebar role={auth.role} ministry={auth.ministry} />
      <main className="app-main">
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/electoral" element={<ElectoralAuthorityPage />} />
          <Route path="/bulk-operations" element={<BulkOperationsPage role={auth.role} token={auth.token} />} />
          <Route path="/users" element={<UsersPage role={auth.role} token={auth.token} />} />
          <Route path="/justice" element={<TransitionalJusticePage />} />
          <Route path="/audit" element={<AuditPage />} />
          <Route path="/enrollments" element={<EnrollmentReviewPage role={auth.role} token={auth.token} />} />
          <Route path="/issuance" element={<CredentialIssuancePage role={auth.role} token={auth.token} />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  )
}
