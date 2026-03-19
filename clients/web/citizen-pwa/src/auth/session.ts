import { setToken, clearToken, getToken } from '../api/client';
import type { JWTClaims } from '../types';

/** Decode a JWT without verifying signature (verification happens server-side). */
export function decodeJWT(token: string): JWTClaims | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload) as JWTClaims;
  } catch {
    return null;
  }
}

export function isTokenExpired(token: string): boolean {
  const claims = decodeJWT(token);
  if (!claims) return true;
  return claims.exp * 1000 < Date.now();
}

export function saveSession(token: string): void {
  setToken(token);
}

export function clearSession(): void {
  clearToken();
}

export function currentSession(): { token: string; claims: JWTClaims } | null {
  const token = getToken();
  if (!token) return null;
  if (isTokenExpired(token)) { clearSession(); return null; }
  const claims = decodeJWT(token);
  if (!claims) { clearSession(); return null; }
  return { token, claims };
}
