/**
 * Encrypted credential wallet backed by IndexedDB via the `idb` library.
 *
 * Credentials are stored as JSON strings.  In a production implementation
 * the values would be encrypted with a key derived from the citizen's PIN
 * via PBKDF2 / Web Crypto API.  That encryption layer is marked TODO and
 * will be added before the production launch.
 *
 * PRD FR-006: offline credential presentation (72 h).
 */
import { openDB, IDBPDatabase } from 'idb'

const DB_NAME = 'indis-wallet'
const DB_VERSION = 1
const STORE = 'credentials'

export interface StoredCredential {
  id: string
  type: string
  issuer: string
  issuedAt: string
  expiresAt: string
  /** Full W3C VC JSON string. */
  vcJson: string
  /** Proof (ZK or Ed25519) bytes as base64. */
  proofB64?: string
}

async function db(): Promise<IDBPDatabase> {
  return openDB(DB_NAME, DB_VERSION, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE, { keyPath: 'id' })
      }
    },
  })
}

export const wallet = {
  async list(): Promise<StoredCredential[]> {
    return (await db()).getAll(STORE)
  },

  async get(id: string): Promise<StoredCredential | undefined> {
    return (await db()).get(STORE, id)
  },

  async put(cred: StoredCredential): Promise<void> {
    await (await db()).put(STORE, cred)
  },

  async delete(id: string): Promise<void> {
    await (await db()).delete(STORE, id)
  },
}
