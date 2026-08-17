import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
afterEach(() => cleanup());
Object.defineProperty(window, 'matchMedia', { writable: true, value: (query: string) => ({ matches: false, media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false }) });

if (!window.localStorage) {
  const values = new Map<string, string>();
  const storage: Storage = {
    get length() { return values.size; },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => [...values.keys()][index] ?? null,
    removeItem: (key) => { values.delete(key); },
    setItem: (key, value) => { values.set(key, String(value)); },
  };
  Object.defineProperty(window, 'localStorage', { configurable: true, value: storage });
}

if (!globalThis.ResizeObserver) {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

// oauth4webapi uses the current typed-array Base64 API. JSDOM's realm can lag
// the Node/browser runtime, so provide the standards-compatible test polyfill.
if (!(Uint8Array.prototype as Uint8Array & { toBase64?: unknown }).toBase64) {
  Object.defineProperty(Uint8Array.prototype, 'toBase64', {
    configurable: true,
    value(this: Uint8Array, options?: { alphabet?: 'base64' | 'base64url'; omitPadding?: boolean }) {
      let binary = '';
      for (const byte of this) binary += String.fromCharCode(byte);
      let encoded = btoa(binary);
      if (options?.alphabet === 'base64url') encoded = encoded.replace(/\+/g, '-').replace(/\//g, '_');
      return options?.omitPadding ? encoded.replace(/=+$/g, '') : encoded;
    }
  });
}
