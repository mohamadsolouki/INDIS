import { Routes, Route, Navigate } from 'react-router-dom'
import { useState } from 'react'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import BulkOperationsPage from './pages/BulkOperationsPage'
import UsersPage from './pages/UsersPage'
import AuditPage from './pages/AuditPage'
import Sidebar from './components/Sidebar'

function useGovAuth() {
  const token = localStorage.getItem('gov_token')
  return { isAuthenticated: !!token }
}

export default function App() {
  const { isAuthenticated } = useGovAuth()

  if (!isAuthenticated) {
    return (
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <div style={{ display: 'flex', minHeight: '100dvh' }}>
      <Sidebar />
      <main style={{ flex: 1, padding: 24 }}>
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/bulk-operations" element={<BulkOperationsPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/audit" element={<AuditPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>
    </div>
  )
}
