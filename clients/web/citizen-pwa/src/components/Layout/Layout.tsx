import { Outlet } from 'react-router-dom';
import Navbar from './Navbar';
import BottomNav from './BottomNav';

export default function Layout() {
  return (
    <div className="flex flex-col min-h-dvh bg-gray-50">
      <Navbar />
      <main className="flex-1 pb-20 overflow-auto">
        <Outlet />
      </main>
      <BottomNav />
    </div>
  );
}
