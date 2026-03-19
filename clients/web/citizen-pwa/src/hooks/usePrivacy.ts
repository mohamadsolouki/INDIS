import { useState, useEffect, useCallback } from 'react';
import { privacy as privacyApi } from '../api/gateway';
import type { PrivacyEvent, ConsentRule, ConsentRuleRequest, DataExportRequest, PaginatedResponse } from '../types';

export function usePrivacyHistory() {
  const [events, setEvents] = useState<PrivacyEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [nextPageToken, setNextPageToken] = useState<string | undefined>();

  const load = useCallback(async (pageToken?: string) => {
    setLoading(true); setError(null);
    try {
      const res: PaginatedResponse<PrivacyEvent> = await privacyApi.getHistory({ page_token: pageToken });
      setEvents(pageToken ? (prev) => [...prev, ...res.items] : res.items);
      setNextPageToken(res.nextPageToken);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load history');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  return { events, loading, error, hasMore: !!nextPageToken, loadMore: () => load(nextPageToken), reload: () => load() };
}

export function useConsentRules() {
  const [rules, setRules] = useState<ConsentRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await privacyApi.listConsent();
      setRules(res.rules ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load consent rules');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const addRule = useCallback(async (req: ConsentRuleRequest) => {
    const rule = await privacyApi.createConsent(req);
    setRules((prev) => [...prev, rule]);
    return rule;
  }, []);

  const removeRule = useCallback(async (id: string) => {
    await privacyApi.deleteConsent(id);
    setRules((prev) => prev.filter((r) => r.id !== id));
  }, []);

  return { rules, loading, error, reload: load, addRule, removeRule };
}

export function useDataExport() {
  const [request, setRequest] = useState<DataExportRequest | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const requestExport = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const req = await privacyApi.requestExport();
      setRequest(req);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export request failed');
    } finally {
      setLoading(false);
    }
  }, []);

  const checkStatus = useCallback(async (id: string) => {
    setLoading(true);
    try {
      const req = await privacyApi.getExportStatus(id);
      setRequest(req);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to check export status');
    } finally {
      setLoading(false);
    }
  }, []);

  return { request, loading, error, requestExport, checkStatus };
}
