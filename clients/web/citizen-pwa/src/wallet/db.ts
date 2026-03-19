import { openDB, type IDBPDatabase } from 'idb';
import type { WalletCredential } from '../types';

const DB_NAME    = 'indis_wallet';
const DB_VERSION = 1;
const STORE_NAME = 'credentials';

export type WalletDB = IDBPDatabase<{
  credentials: {
    key: string;
    value: WalletCredential;
    indexes: {
      by_type: string;
      by_synced_at: string;
    };
  };
}>;

let _db: WalletDB | null = null;

export async function getWalletDB(): Promise<WalletDB> {
  if (_db) return _db;
  _db = await openDB<{
    credentials: {
      key: string;
      value: WalletCredential;
      indexes: { by_type: string; by_synced_at: string };
    };
  }>(DB_NAME, DB_VERSION, {
    upgrade(db) {
      const store = db.createObjectStore(STORE_NAME, { keyPath: 'id' });
      store.createIndex('by_type', 'type', { unique: false });
      store.createIndex('by_synced_at', 'syncedAt', { unique: false });
    },
  });
  return _db;
}
