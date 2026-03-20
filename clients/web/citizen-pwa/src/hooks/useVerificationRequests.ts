import { useState, useEffect, useCallback, useRef } from 'react';
import { useAuthStore } from '../auth/store';
import { getToken } from '../api/client';

export interface VerificationRequest {
  id: string;
  verifierName: string;
  verifierDid: string;
  requestedCredentials: string[];
  purpose: string;
  timestamp: string;
  expiresAt: string;
}

type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

/**
 * useVerificationRequests — subscribes to real-time verification requests
 * via Server-Sent Events (SSE) from the gateway notification endpoint.
 *
 * Falls back gracefully when offline or in dev without the full stack.
 *
 * PRD FR-013: citizen must explicitly approve or deny every ZK proof request.
 * No credential data is sent until the citizen taps "Approve".
 */
export function useVerificationRequests() {
  const did = useAuthStore((s) => s.did);
  const [requests, setRequests] = useState<VerificationRequest[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const esRef = useRef<EventSource | null>(null);

  const connect = useCallback(() => {
    if (!did || !navigator.onLine) return;

    const token = getToken();
    if (!token) return;

    const base = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? 'http://localhost:8080';
    const url = `${base}/v1/notification/stream?did=${encodeURIComponent(did)}&token=${encodeURIComponent(token)}`;

    const es = new EventSource(url);
    esRef.current = es;
    setStatus('connecting');

    es.onopen = () => setStatus('connected');

    es.addEventListener('verification_request', (e) => {
      try {
        const req = JSON.parse((e as MessageEvent).data) as VerificationRequest;
        setRequests((prev) => {
          // Deduplicate by id; most recent first.
          const without = prev.filter((r) => r.id !== req.id);
          return [req, ...without];
        });
      } catch {
        // Malformed event — ignore.
      }
    });

    es.onerror = () => {
      setStatus('error');
      es.close();
      esRef.current = null;
      // Reconnect after 5 s if still online.
      setTimeout(() => { if (navigator.onLine) connect(); }, 5000);
    };
  }, [did]);

  useEffect(() => {
    connect();
    return () => {
      esRef.current?.close();
      esRef.current = null;
    };
  }, [connect]);

  // Reconnect when going back online.
  useEffect(() => {
    function onOnline() { if (!esRef.current) connect(); }
    function onOffline() { setStatus('disconnected'); }
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);
    return () => {
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  }, [connect]);

  const dismiss = useCallback((id: string) => {
    setRequests((prev) => prev.filter((r) => r.id !== id));
  }, []);

  return { requests, status, dismiss };
}
