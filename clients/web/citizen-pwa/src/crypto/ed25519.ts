// Ed25519 via WebCrypto API (available in modern browsers)
// PRD FR-001.4: citizen private keys never leave the device.

import { openDB } from 'idb';

const DB_NAME = 'indis_crypto';
const KEY_STORE_NAME = 'indis_ed25519_key';

/** Generate a new Ed25519 key pair. Stores the private key in IndexedDB (non-extractable). */
export async function generateEd25519KeyPair(): Promise<CryptoKeyPair> {
  const keyPair = await crypto.subtle.generateKey(
    { name: 'Ed25519' },
    false, // non-extractable — private key never leaves the device
    ['sign', 'verify'],
  );
  return keyPair;
}

/** Export the public key as raw bytes (32 bytes for Ed25519). */
export async function exportPublicKeyRaw(publicKey: CryptoKey): Promise<ArrayBuffer> {
  return crypto.subtle.exportKey('raw', publicKey);
}

/** Export the public key as multibase (base58btc) string for DID document. */
export async function exportPublicKeyMultibase(publicKey: CryptoKey): Promise<string> {
  const raw = await exportPublicKeyRaw(publicKey);
  return 'z' + arrayBufferToBase58(raw); // z prefix = base58btc
}

/** Sign a message with the private key. Returns base64url signature. */
export async function sign(privateKey: CryptoKey, message: BufferSource): Promise<string> {
  const signature = await crypto.subtle.sign('Ed25519', privateKey, message);
  return arrayBufferToBase64url(signature);
}

/** Verify a signature. */
export async function verify(publicKey: CryptoKey, message: BufferSource, signatureBase64url: string): Promise<boolean> {
  const signature = base64urlToArrayBuffer(signatureBase64url);
  return crypto.subtle.verify('Ed25519', publicKey, signature, message);
}

// ── Key persistence via IndexedDB (idb library) ───────────────────────────────

function openKeyStore() {
  return openDB(DB_NAME, 1, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(KEY_STORE_NAME)) {
        db.createObjectStore(KEY_STORE_NAME, { keyPath: 'id' });
      }
    },
  });
}

/** Store the key pair in IndexedDB. Returns the stored key ID. */
export async function storeKeyPair(keyPair: CryptoKeyPair): Promise<string> {
  const keyId = crypto.randomUUID();
  const db = await openKeyStore();
  await db.put(KEY_STORE_NAME, { id: keyId, privateKey: keyPair.privateKey, publicKey: keyPair.publicKey });
  db.close();
  localStorage.setItem('indis_key_id', keyId);
  return keyId;
}

/** Load a stored key pair from IndexedDB. */
export async function loadKeyPair(keyId: string): Promise<CryptoKeyPair | null> {
  const db = await openKeyStore();
  const record = await db.get(KEY_STORE_NAME, keyId) as { privateKey: CryptoKey; publicKey: CryptoKey } | undefined;
  db.close();
  if (!record) return null;
  return { privateKey: record.privateKey, publicKey: record.publicKey };
}

// ── Encoding helpers ──────────────────────────────────────────────────────────

function arrayBufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

function base64urlToArrayBuffer(s: string): ArrayBuffer {
  const base64 = s.replace(/-/g, '+').replace(/_/g, '/');
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer as ArrayBuffer;
}

const BASE58_ALPHABET = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';

function arrayBufferToBase58(buffer: ArrayBuffer): string {
  const bytes = Array.from(new Uint8Array(buffer));
  let num = BigInt('0x' + bytes.map((b) => b.toString(16).padStart(2, '0')).join(''));
  let result = '';
  const base = BigInt(58);
  while (num > 0n) {
    result = BASE58_ALPHABET[Number(num % base)] + result;
    num /= base;
  }
  // Leading zeros
  for (const b of bytes) {
    if (b !== 0) break;
    result = '1' + result;
  }
  return result;
}
