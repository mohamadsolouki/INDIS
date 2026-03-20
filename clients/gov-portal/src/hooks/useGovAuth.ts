/**
 * useGovAuth — reads the stored JWT and extracts the operator role from its payload.
 *
 * The gateway issues HMAC-HS256 JWTs with claims: { sub, role, ministry, exp }.
 * Roles: "viewer" | "operator" | "senior" | "admin"
 *
 * Role hierarchy:
 *   viewer   — read-only: dashboards, audit log
 *   operator — viewer + approve bulk ops, view users
 *   senior   — operator + create users, change roles
 *   admin    — senior + all destructive actions
 */

export type GovRole = 'viewer' | 'operator' | 'senior' | 'admin'

interface GovAuthState {
  isAuthenticated: boolean
  token: string
  role: GovRole
  ministry: string
  sub: string
}

function parseJwtPayload(token: string): Record<string, unknown> {
  try {
    const parts = token.split('.')
    if (parts.length < 2) return {}
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(payload) as Record<string, unknown>
  } catch {
    return {}
  }
}

export function useGovAuth(): GovAuthState {
  const token = localStorage.getItem('gov_token') ?? ''
  if (!token) {
    return { isAuthenticated: false, token: '', role: 'viewer', ministry: '', sub: '' }
  }
  const payload = parseJwtPayload(token)
  const role = (payload['role'] as GovRole | undefined) ?? 'viewer'
  const ministry = (payload['ministry'] as string | undefined) ?? ''
  const sub = (payload['sub'] as string | undefined) ?? ''
  return { isAuthenticated: true, token, role, ministry, sub }
}

/** Returns true when the current role is at least as privileged as [required]. */
export function hasRole(current: GovRole, required: GovRole): boolean {
  const order: GovRole[] = ['viewer', 'operator', 'senior', 'admin']
  return order.indexOf(current) >= order.indexOf(required)
}
