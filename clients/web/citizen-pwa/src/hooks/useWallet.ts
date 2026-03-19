import { useState, useEffect, useCallback } from 'react';
import { listCredentials, upsertCredential, deleteCredential } from '../wallet/wallet';
import type { WalletCredential, CredentialType } from '../types';

export function useWallet(filterType?: CredentialType) {
  const [credentials, setCredentials] = useState<WalletCredential[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const creds = await listCredentials(filterType);
      setCredentials(creds);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load wallet');
    } finally {
      setLoading(false);
    }
  }, [filterType]);

  useEffect(() => { void load(); }, [load]);

  const save = useCallback(async (cred: WalletCredential) => {
    await upsertCredential(cred);
    await load();
  }, [load]);

  const remove = useCallback(async (id: string) => {
    await deleteCredential(id);
    await load();
  }, [load]);

  return { credentials, loading, error, reload: load, save, remove };
}
