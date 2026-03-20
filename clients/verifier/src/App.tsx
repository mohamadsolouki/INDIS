import { Routes, Route, Navigate } from 'react-router-dom'
import ScanPage from './pages/ScanPage'
import ResultPage from './pages/ResultPage'
import LoginPage from './pages/LoginPage'
import HistoryPage from './pages/HistoryPage'

function useVerifierAuth() {
  return { isAuthenticated: !!localStorage.getItem('verifier_id') }
}

export default function App() {
  const { isAuthenticated } = useVerifierAuth()

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      {isAuthenticated ? (
        <>
          <Route path="/" element={<ScanPage />} />
          <Route path="/result" element={<ResultPage />} />
          <Route path="/history" element={<HistoryPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </>
      ) : (
        <Route path="*" element={<Navigate to="/login" replace />} />
      )}
    </Routes>
  )
}
