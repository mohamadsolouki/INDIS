import { useState, useEffect, useCallback } from 'react'

const TOKEN_KEY = 'indis_token'
const DID_KEY = 'indis_did'

interface AuthState {
  token: string | null
  did: string | null
  isAuthenticated: boolean
}

/**
 * Simple auth state hook backed by localStorage.
 *
 * In production the token would be validated on mount against the
 * revocation list cached by the service worker.
 */
export function useAuth() {
  const [state, setState] = useState<AuthState>(() => {
    const token = localStorage.getItem(TOKEN_KEY)
    const did = localStorage.getItem(DID_KEY)
    return { token, did, isAuthenticated: !!token }
  })

  const login = useCallback((token: string, did: string) => {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(DID_KEY, did)
    setState({ token, did, isAuthenticated: true })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(DID_KEY)
    setState({ token: null, did: null, isAuthenticated: false })
  }, [])

  return { ...state, login, logout }
}
