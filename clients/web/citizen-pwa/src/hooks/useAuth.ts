import { useAuthStore } from '../auth/store';

export function useAuth() {
  const { did, token, isAuthenticated, login, logout } = useAuthStore();
  return { did, token, isAuthenticated, login, logout };
}
