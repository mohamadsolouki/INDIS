import { useState, useEffect, useCallback } from 'react';
import { card as cardApi } from '../api/gateway';
import { useAuthStore } from '../auth/store';
import type { DigitalCard } from '../types';

/**
 * useCard — React hook for fetching and managing the citizen's digital identity card.
 *
 * Automatically fetches the card when the authenticated DID is available.
 * Exposes `generate()` to request a new card when none exists.
 *
 * @returns cardData  — the resolved DigitalCard or null if not yet fetched
 * @returns loading   — true while any async operation is in flight
 * @returns error     — error message string or null
 * @returns refresh   — re-fetch the card from the API
 * @returns generate  — call the card generation endpoint, then refresh
 */
export function useCard() {
  const did = useAuthStore((s) => s.did);
  const [cardData, setCardData] = useState<DigitalCard | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    if (!did) return;
    setLoading(true);
    setError(null);
    try {
      const data = await cardApi.get(did);
      setCardData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load card');
    } finally {
      setLoading(false);
    }
  }, [did]);

  const generate = useCallback(async () => {
    if (!did) return;
    setLoading(true);
    setError(null);
    try {
      await cardApi.generate();
      await fetch();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate card');
    } finally {
      setLoading(false);
    }
  }, [did, fetch]);

  useEffect(() => {
    void fetch();
  }, [fetch]);

  return { cardData, loading, error, refresh: fetch, generate };
}
