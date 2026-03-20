import { useState, useEffect } from 'react'
import { wallet, StoredCredential } from '../lib/wallet'
import { api } from '../lib/api'
import { useAuth } from './useAuth'

/**
 * Loads credentials from the local wallet (IndexedDB) and optionally
 * syncs fresh credentials from the gateway when online.
 */
export function useCredentials() {
  const { token, did } = useAuth()
  const [credentials, setCredentials] = useState<StoredCredential[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    async function load() {
      setLoading(true)
      try {
        // Load from local wallet first (works offline).
        const local = await wallet.list()
        if (!cancelled) setCredentials(local)

        // If online, fetch from gateway and merge.
        if (navigator.onLine && token && did) {
          const remote = await api.get<{ credentials: StoredCredential[] }>(
            `/credentials?did=${encodeURIComponent(did)}`,
            token,
          )
          for (const cred of remote.credentials) {
            await wallet.put(cred)
          }
          const merged = await wallet.list()
          if (!cancelled) setCredentials(merged)
        }
      } catch (e) {
        if (!cancelled) setError(String(e))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    load()
    return () => { cancelled = true }
  }, [token, did])

  return { credentials, loading, error }
}
