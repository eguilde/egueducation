import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
afterEach(() => cleanup());
Object.defineProperty(window, 'matchMedia', { writable: true, value: (query: string) => ({ matches: false, media: query, onchange: null, addEventListener: () => undefined, removeEventListener: () => undefined, addListener: () => undefined, removeListener: () => undefined, dispatchEvent: () => false }) });

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
