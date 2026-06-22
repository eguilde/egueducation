import * as oauth from 'oauth4webapi';

const DEFAULT_DB_NAME = 'egueducation_oidc_dpop';
const STORE_NAME = 'keys';
const KEY_ID = 'dpop';

let cachedKeys: CryptoKeyPair | null = null;

export async function loadOrCreateKeyPair(dbName: string = DEFAULT_DB_NAME): Promise<CryptoKeyPair> {
  if (cachedKeys) {
    return cachedKeys;
  }

  try {
    const stored = await loadFromIndexedDb(dbName);
    if (stored) {
      cachedKeys = stored;
      return cachedKeys;
    }
  } catch {
    // IndexedDB may be unavailable in some browser contexts.
  }

  cachedKeys = await oauth.generateKeyPair('ES256', { extractable: false });
  await saveToIndexedDb(dbName, cachedKeys).catch(() => null);
  return cachedKeys;
}

export async function clearKeyPair(dbName: string = DEFAULT_DB_NAME): Promise<void> {
  cachedKeys = null;
  try {
    await deleteFromIndexedDb(dbName);
  } catch {
    // Best effort cleanup.
  }
}

function openDb(dbName: string): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(dbName, 1);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

async function loadFromIndexedDb(dbName: string): Promise<CryptoKeyPair | null> {
  const db = await openDb(dbName);
  return new Promise((resolve, reject) => {
    const transaction = db.transaction(STORE_NAME, 'readonly');
    const request = transaction.objectStore(STORE_NAME).get(KEY_ID);
    request.onsuccess = () => resolve(request.result ?? null);
    request.onerror = () => reject(request.error);
    transaction.oncomplete = () => db.close();
  });
}

async function saveToIndexedDb(dbName: string, keys: CryptoKeyPair): Promise<void> {
  const db = await openDb(dbName);
  return new Promise((resolve, reject) => {
    const transaction = db.transaction(STORE_NAME, 'readwrite');
    transaction.objectStore(STORE_NAME).put(keys, KEY_ID);
    transaction.oncomplete = () => {
      db.close();
      resolve();
    };
    transaction.onerror = () => {
      db.close();
      reject(transaction.error);
    };
  });
}

async function deleteFromIndexedDb(dbName: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.deleteDatabase(dbName);
    request.onsuccess = () => resolve();
    request.onerror = () => reject(request.error);
  });
}
