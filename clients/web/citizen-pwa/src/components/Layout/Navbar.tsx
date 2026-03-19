import { useTranslation } from 'react-i18next';

export default function Navbar() {
  const { t } = useTranslation();
  return (
    <header className="sticky top-0 z-40 bg-indis-primary shadow-md">
      <div className="flex items-center justify-between px-4 h-14 max-w-lg mx-auto">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center">
            <span className="text-white font-bold text-sm">ه</span>
          </div>
          <span className="text-white font-bold text-base">{t('app.name')}</span>
        </div>
        <div className="flex items-center gap-2">
          {/* Offline indicator */}
          <OnlineStatus />
        </div>
      </div>
    </header>
  );
}

function OnlineStatus() {
  const { t } = useTranslation();
  // Check online status
  const isOnline = typeof navigator !== 'undefined' ? navigator.onLine : true;
  if (isOnline) return null;
  return (
    <span className="text-xs bg-yellow-500 text-white px-2 py-0.5 rounded-full">
      {t('common.offline')}
    </span>
  );
}
