import { Suspense, lazy, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { applyLocale } from './i18n';
import Layout from './components/Layout/Layout';
import { useAuthStore } from './auth/store';

// Lazy-loaded pages
const Home       = lazy(() => import('./pages/Home'));
const Wallet     = lazy(() => import('./pages/Wallet'));
const Privacy    = lazy(() => import('./pages/Privacy'));
const Verify     = lazy(() => import('./pages/Verify'));
const Settings   = lazy(() => import('./pages/Settings'));
const Enrollment = lazy(() => import('./pages/Enrollment'));
const Login      = lazy(() => import('./pages/Login'));

function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center min-h-dvh bg-gray-50">
      <div className="flex flex-col items-center gap-4">
        <div className="w-12 h-12 rounded-full border-4 border-indis-primary border-t-transparent animate-spin" />
        <span className="text-gray-500 text-sm">در حال بارگذاری...</span>
      </div>
    </div>
  );
}

/** Wraps protected routes — redirects to /login when not authenticated. */
function RequireAuth({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
}

export default function App() {
  const { i18n } = useTranslation();

  useEffect(() => {
    applyLocale(i18n.language);
  }, [i18n.language]);

  return (
    <BrowserRouter>
      <Suspense fallback={<LoadingSpinner />}>
        <Routes>
          {/* Public routes */}
          <Route path="/login" element={<Login />} />
          <Route path="/enrollment/*" element={<Enrollment />} />

          {/* Protected routes */}
          <Route
            path="/"
            element={
              <RequireAuth>
                <Layout />
              </RequireAuth>
            }
          >
            <Route index element={<Home />} />
            <Route path="wallet" element={<Wallet />} />
            <Route path="privacy/*" element={<Privacy />} />
            <Route path="verify" element={<Verify />} />
            <Route path="settings" element={<Settings />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
