import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { HomeIcon, WalletIcon, ShieldCheckIcon, QrCodeIcon, Cog6ToothIcon } from '@heroicons/react/24/outline';
import { HomeIcon as HomeIconSolid, WalletIcon as WalletIconSolid, ShieldCheckIcon as ShieldIconSolid, QrCodeIcon as QrIconSolid, Cog6ToothIcon as CogIconSolid } from '@heroicons/react/24/solid';
import { cn } from '../../lib/cn';

const NAV_ITEMS = [
  { to: '/',        labelKey: 'nav.home',     Icon: HomeIcon,       ActiveIcon: HomeIconSolid },
  { to: '/wallet',  labelKey: 'nav.wallet',   Icon: WalletIcon,     ActiveIcon: WalletIconSolid },
  { to: '/privacy', labelKey: 'nav.privacy',  Icon: ShieldCheckIcon, ActiveIcon: ShieldIconSolid },
  { to: '/verify',  labelKey: 'nav.verify',   Icon: QrCodeIcon,     ActiveIcon: QrIconSolid },
  { to: '/settings',labelKey: 'nav.settings', Icon: Cog6ToothIcon,  ActiveIcon: CogIconSolid },
] as const;

export default function BottomNav() {
  const { t } = useTranslation();
  return (
    <nav
      className="fixed bottom-0 inset-x-0 z-40 bg-white border-t border-gray-200 safe-area-inset-bottom"
      aria-label={t('nav.home')}
    >
      <div className="flex items-stretch justify-around max-w-lg mx-auto h-16">
        {NAV_ITEMS.map(({ to, labelKey, Icon, ActiveIcon }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex flex-col items-center justify-center flex-1 gap-1 text-xs transition-colors',
                isActive ? 'text-indis-primary' : 'text-gray-500 hover:text-gray-700',
              )
            }
          >
            {({ isActive }) => (
              <>
                {isActive
                  ? <ActiveIcon className="w-6 h-6" />
                  : <Icon className="w-6 h-6" />
                }
                <span className="text-[10px] font-medium">{t(labelKey)}</span>
              </>
            )}
          </NavLink>
        ))}
      </div>
    </nav>
  );
}
