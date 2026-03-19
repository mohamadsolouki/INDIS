import { Suspense, lazy, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { applyLocale } from './i18n';
import Layout from './components/Layout/Layout';

// Lazy-loaded pages
const Home       = lazy(() => import('./pages/Home'));
const Wallet     = lazy(() => import('./pages/Wallet'));
const Privacy    = lazy(() => import('./pages/Privacy'));
const Verify     = lazy(() => import('./pages/Verify'));
const Settings   = lazy(() => import('./pages/Settings'));
const Enrollment = lazy(() => import('./pages/Enrollment'));

function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50">
      <div className="flex flex-col items-center gap-4">
        <div className="w-12 h-12 rounded-full border-4 border-indis-primary border-t-transparent animate-spin" />
        <span className="text-gray-500 text-sm">در حال بارگذاری...</span>
      </div>
    </div>
  );
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
          <Route path="/" element={<Layout />}>
            <Route index element={<Home />} />
            <Route path="wallet" element={<Wallet />} />
            <Route path="privacy/*" element={<Privacy />} />
            <Route path="verify" element={<Verify />} />
            <Route path="settings" element={<Settings />} />
          </Route>
          <Route path="/enrollment/*" element={<Enrollment />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
