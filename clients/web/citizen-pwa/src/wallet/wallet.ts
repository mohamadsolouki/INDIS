import { getWalletDB } from './db';
import type { WalletCredential, CredentialType } from '../types';

/** Store or update a credential in the local wallet. */
export async function upsertCredential(cred: WalletCredential): Promise<void> {
  const db = await getWalletDB();
  await db.put('credentials', cred);
}

/** Get a single credential by ID. */
export async function getCredential(id: string): Promise<WalletCredential | undefined> {
  const db = await getWalletDB();
  return db.get('credentials', id);
}

/** Get all credentials, optionally filtered by type. */
export async function listCredentials(type?: CredentialType): Promise<WalletCredential[]> {
  const db = await getWalletDB();
  if (type) {
    return db.getAllFromIndex('credentials', 'by_type', type);
  }
  return db.getAll('credentials');
}

/** Delete a credential from the wallet. */
export async function deleteCredential(id: string): Promise<void> {
  const db = await getWalletDB();
  await db.delete('credentials', id);
}

/** Clear all credentials (used on logout). */
export async function clearWallet(): Promise<void> {
  const db = await getWalletDB();
  await db.clear('credentials');
}

/** Count all stored credentials. */
export async function countCredentials(): Promise<number> {
  const db = await getWalletDB();
  return db.count('credentials');
}
