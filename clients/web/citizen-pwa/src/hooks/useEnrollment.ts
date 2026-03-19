import { useState, useCallback } from 'react';
import { enrollment as enrollmentApi } from '../api/gateway';
import type { EnrollmentStatus, EnrollmentPathway } from '../types';

/**
 * useEnrollment — React hook encapsulating the multi-step enrollment workflow.
 *
 * Manages enrollment state transitions:
 *   initiate()        → POST /enrollment/initiate  → returns EnrollmentStatus
 *   submitBiometrics() → POST /enrollment/:id/biometrics
 *   complete()         → POST /enrollment/:id/complete → returns { did }
 *
 * All mutations update `status` in place, keeping the wizard UI in sync.
 */
export function useEnrollment() {
  const [status, setStatus] = useState<EnrollmentStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  /** Start a new enrollment session for the given pathway. */
  const initiate = useCallback(async (pathway: EnrollmentPathway) => {
    setLoading(true);
    setError(null);
    try {
      const s = await enrollmentApi.initiate({ pathway });
      setStatus(s);
      return s;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Enrollment failed');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  /** Submit raw biometric data (fingerprint template or face descriptor) for the active enrollment. */
  const submitBiometrics = useCallback(
    async (data: { fingerprint_template?: string; face_descriptor?: string }) => {
      if (!status) return;
      setLoading(true);
      setError(null);
      try {
        await enrollmentApi.submitBiometrics(status.enrollmentId, data);
        const updated = await enrollmentApi.get(status.enrollmentId);
        setStatus(updated);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Biometric submission failed');
      } finally {
        setLoading(false);
      }
    },
    [status],
  );

  /** Finalize the enrollment and obtain the newly issued DID. */
  const complete = useCallback(async () => {
    if (!status) return null;
    setLoading(true);
    setError(null);
    try {
      const result = await enrollmentApi.complete(status.enrollmentId);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Enrollment completion failed');
      return null;
    } finally {
      setLoading(false);
    }
  }, [status]);

  return { status, loading, error, initiate, submitBiometrics, complete };
}
