// ─── Auth ─────────────────────────────────────────────────────────────────────
export interface JWTClaims {
  sub: string;        // citizen DID e.g. "did:indis:abc123"
  ministry?: string;
  role?: string;
  exp: number;
  iat: number;
}

export interface AuthState {
  did: string | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (token: string) => void;
  logout: () => void;
}

// ─── Identity ─────────────────────────────────────────────────────────────────
export interface VerificationMethod {
  id: string;
  type: string;
  controller: string;
  publicKeyMultibase?: string;
}

export interface DIDDocument {
  id: string;
  verificationMethod: VerificationMethod[];
  authentication: string[];
  created: string;
  updated: string;
}

export interface Identity {
  did: string;
  publicKey: string;
  document: DIDDocument;
  createdAt: string;
  status: 'active' | 'inactive' | 'suspended';
}

// ─── Credentials ──────────────────────────────────────────────────────────────
export type CredentialType =
  | 'CitizenshipCredential'
  | 'AgeRangeCredential'
  | 'VoterEligibilityCredential'
  | 'ResidencyCredential'
  | 'MilitaryServiceCredential'
  | 'EmploymentCredential'
  | 'HealthInsuranceCredential'
  | 'DisabilityCredential'
  | 'TemporaryEnrollmentReceipt'
  | 'GuardianCredential'
  | 'SocialAttestationCredential';

export interface Credential {
  id: string;
  type: CredentialType;
  issuer: string;
  subject: string;
  issuedAt: string;
  expiresAt: string;
  status: 'active' | 'revoked' | 'expired' | 'suspended';
  claims: Record<string, unknown>;
}

// ─── Digital Card ─────────────────────────────────────────────────────────────
export interface DigitalCard {
  cardId: string;
  did: string;
  qrCode: string;      // base64-encoded PNG
  expiresAt: string;
  fullName: string;    // Persian name
  fullNameEn: string;  // English transliteration
  nationalId: string;  // always masked e.g. "●●●●●●●●●●"
  photo?: string;      // base64 image (optional, privacy-gated)
}

export interface GenerateCardResponse {
  card_id: string;
  qr_code: string;
  expires_at: string;
}

// ─── Enrollment ───────────────────────────────────────────────────────────────
export type EnrollmentPathway = 'standard' | 'enhanced' | 'social';

export interface EnrollmentStatus {
  enrollmentId: string;
  pathway: EnrollmentPathway;
  step: string;
  status: 'pending' | 'biometrics_complete' | 'completed' | 'failed';
  createdAt: string;
}

export interface InitiateEnrollmentRequest {
  pathway: EnrollmentPathway;
  locale?: string;
}

export interface CompleteEnrollmentResponse {
  did: string;
  credentialIds: string[];
}

// ─── Privacy ──────────────────────────────────────────────────────────────────
export interface PrivacyEvent {
  eventId: string;
  verifierDid: string;
  verifierName: string;
  credentialType: string;
  timestamp: string;
  result: 'approved' | 'denied' | 'auto_approved';
  actionType: string;
}

export interface ConsentRule {
  id: string;
  verifierCategory: string;
  credentialType: string;
  rule: 'always' | 'ask' | 'never';
  createdAt: string;
}

export interface ConsentRuleRequest {
  verifier_category: string;
  credential_type: string;
  rule: 'always' | 'ask' | 'never';
}

export interface DataExportRequest {
  requestId: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  requestedAt: string;
  downloadUrl?: string;
}

// ─── i18n ─────────────────────────────────────────────────────────────────────
export type SupportedLocale = 'fa' | 'en' | 'ckb' | 'kmr' | 'ar' | 'az';
export type TextDirection = 'rtl' | 'ltr';

export interface LocaleConfig {
  code: SupportedLocale;
  name: string;        // English name
  nativeName: string;  // Native script name
  dir: TextDirection;
}

// ─── Wallet (IndexedDB) ───────────────────────────────────────────────────────
export interface WalletCredential {
  id: string;
  type: CredentialType;
  raw: string;  // JSON-LD VC string
  syncedAt: string;
  isOfflineAvailable: boolean;
}

// ─── Settings ─────────────────────────────────────────────────────────────────
export interface AppSettings {
  locale: SupportedLocale;
  usePersianNumerals: boolean;
  useSolarHijri: boolean;
  fontSize: 'normal' | 'large' | 'xlarge';
  theme: 'light' | 'dark' | 'system';
}

// ─── API Utilities ────────────────────────────────────────────────────────────
export interface ApiError {
  code: string;
  message: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  nextPageToken?: string;
  total: number;
}

export interface AuditEvent {
  eventId: string;
  actionType: string;
  actorDid: string;
  subjectDid: string;
  resourceId: string;
  timestamp: string;
  ipRange: string;
  prevHash: string;
  hash: string;
}
