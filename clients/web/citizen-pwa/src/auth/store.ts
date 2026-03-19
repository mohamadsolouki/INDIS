import { create } from 'zustand';
import { currentSession, saveSession, clearSession } from './session';
import type { AuthState } from '../types';

export const useAuthStore = create<AuthState>((set) => {
  // Hydrate from localStorage on store creation
  const session = currentSession();
  const initial = session
    ? { did: session.claims.sub, token: session.token, isAuthenticated: true }
    : { did: null, token: null, isAuthenticated: false };

  return {
    ...initial,
    login(token: string) {
      saveSession(token);
      const s = currentSession();
      if (s) set({ did: s.claims.sub, token, isAuthenticated: true });
    },
    logout() {
      clearSession();
      set({ did: null, token: null, isAuthenticated: false });
    },
  };
});
