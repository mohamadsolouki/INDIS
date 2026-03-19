import { http } from './client';
import type {
  Identity, Credential, DigitalCard, GenerateCardResponse,
  EnrollmentStatus, InitiateEnrollmentRequest, CompleteEnrollmentResponse,
  PrivacyEvent, ConsentRule, ConsentRuleRequest, DataExportRequest,
  PaginatedResponse, AuditEvent,
} from '../types';

// ─── Health ────────────────────────────────────────────────────────────────────
export const health = {
  get: () => http.get<{ status: string }>('/health'),
};

// ─── Identity ─────────────────────────────────────────────────────────────────
export const identity = {
  register: (req: { public_key: string; locale?: string }) =>
    http.post<Identity>('/v1/identity/register', req),
  get: (did: string) =>
    http.get<Identity>(`/v1/identity/${encodeURIComponent(did)}`),
  deactivate: (did: string) =>
    http.post<void>(`/v1/identity/${encodeURIComponent(did)}/deactivate`),
};

// ─── Credential ───────────────────────────────────────────────────────────────
export const credential = {
  issue: (req: { subject_did: string; type: string; claims: Record<string, unknown>; expires_in_days?: number }) =>
    http.post<Credential>('/v1/credential/issue', req),
  get: (id: string) =>
    http.get<Credential>(`/v1/credential/${encodeURIComponent(id)}`),
  revoke: (id: string, req: { reason: string }) =>
    http.post<void>(`/v1/credential/${encodeURIComponent(id)}/revoke`, req),
};

// ─── Enrollment ───────────────────────────────────────────────────────────────
export const enrollment = {
  initiate: (req: InitiateEnrollmentRequest) =>
    http.post<EnrollmentStatus>('/v1/enrollment/initiate', req),
  get: (id: string) =>
    http.get<EnrollmentStatus>(`/v1/enrollment/${encodeURIComponent(id)}`),
  submitBiometrics: (id: string, req: { fingerprint_template?: string; face_descriptor?: string }) =>
    http.post<void>(`/v1/enrollment/${encodeURIComponent(id)}/biometrics`, req),
  submitAttestation: (id: string, req: { attestor_dids: string[]; attestation_statement: string }) =>
    http.post<void>(`/v1/enrollment/${encodeURIComponent(id)}/attestation`, req),
  complete: (id: string) =>
    http.post<CompleteEnrollmentResponse>(`/v1/enrollment/${encodeURIComponent(id)}/complete`),
};

// ─── Card ─────────────────────────────────────────────────────────────────────
export const card = {
  get: (did: string) =>
    http.get<DigitalCard>(`/v1/card/${encodeURIComponent(did)}`),
  generate: () =>
    http.post<GenerateCardResponse>('/v1/card/generate'),
};

// ─── Privacy ──────────────────────────────────────────────────────────────────
interface HistoryParams { from?: string; to?: string; page_token?: string; }

export const privacy = {
  getHistory: (params?: HistoryParams) => {
    const q = new URLSearchParams(params as Record<string, string> ?? {}).toString();
    return http.get<PaginatedResponse<PrivacyEvent>>(`/v1/privacy/history${q ? '?' + q : ''}`);
  },
  getSharing: (params?: HistoryParams) => {
    const q = new URLSearchParams(params as Record<string, string> ?? {}).toString();
    return http.get<PaginatedResponse<PrivacyEvent>>(`/v1/privacy/sharing${q ? '?' + q : ''}`);
  },
  listConsent: () =>
    http.get<{ rules: ConsentRule[] }>('/v1/privacy/consent'),
  createConsent: (req: ConsentRuleRequest) =>
    http.post<ConsentRule>('/v1/privacy/consent', req),
  deleteConsent: (id: string) =>
    http.delete<void>(`/v1/privacy/consent/${encodeURIComponent(id)}`),
  requestExport: () =>
    http.post<DataExportRequest>('/v1/privacy/data-export'),
  getExportStatus: (id: string) =>
    http.get<DataExportRequest>(`/v1/privacy/data-export/${encodeURIComponent(id)}`),
};

// ─── Audit ────────────────────────────────────────────────────────────────────
export const audit = {
  getEvents: (params?: { from?: string; to?: string; action?: string; page_token?: string }) => {
    const q = new URLSearchParams(params as Record<string, string> ?? {}).toString();
    return http.get<PaginatedResponse<AuditEvent>>(`/v1/audit/events${q ? '?' + q : ''}`);
  },
};

// ─── Notification ─────────────────────────────────────────────────────────────
export const notification = {
  send: (req: { recipient_did: string; channel: string; subject: string; body: string }) =>
    http.post<void>('/v1/notification/send', req),
};
