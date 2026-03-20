/**
 * Local model for a cached W3C Verifiable Credential.
 * Mirrors Android CredentialCard and iOS CredentialRecord.
 */
export interface CredentialRecord {
  id: string
  credentialType: string
  issuer: string
  issuedAt: string
  expiresAt: string
  vcJson: string
  status: string        // "active" | "revoked" | "expired"
}

export function isRevoked(r: CredentialRecord): boolean {
  return r.status === 'revoked'
}

export function isExpired(r: CredentialRecord): boolean {
  return new Date(r.expiresAt).getTime() < Date.now()
}

/**
 * Simple JSON-backed credential store using AppStorage preferences.
 * Provides upsert-by-id semantics.
 * PRD FR-006: credentials must be available offline for 72 hours.
 */
export class CredentialStore {

  private static readonly KEY = 'credential_list_json'
  private static prefs: Record<string, string> = {}

  static loadAll(): CredentialRecord[] {
    try {
      const raw = AppStorageProxy.get(CredentialStore.KEY)
      return raw ? JSON.parse(raw) as CredentialRecord[] : []
    } catch {
      return []
    }
  }

  static saveAll(records: CredentialRecord[]): void {
    AppStorageProxy.set(CredentialStore.KEY, JSON.stringify(records))
  }

  static upsert(record: CredentialRecord): void {
    const current = CredentialStore.loadAll().filter(r => r.id !== record.id)
    current.unshift(record)
    CredentialStore.saveAll(current)
  }

  static clear(): void {
    AppStorageProxy.set(CredentialStore.KEY, '[]')
  }
}

// Thin proxy that resolves at runtime to avoid circular imports.
const AppStorageProxy = {
  get: (key: string): string => {
    // Resolved at runtime by AppStorage.prefs
    return globalThis.__INDIS_PREFS__?.[key] ?? ''
  },
  set: (key: string, value: string): void => {
    if (!globalThis.__INDIS_PREFS__) globalThis.__INDIS_PREFS__ = {}
    globalThis.__INDIS_PREFS__[key] = value
  }
}
