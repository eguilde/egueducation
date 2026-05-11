import { Injectable } from '@angular/core';

import { loadOrCreateKeyPair } from './dpop-key-store';

@Injectable({ providedIn: 'root' })
export class DPoPProofService {
  private keyPair: CryptoKeyPair | null = null;
  private publicJwk: JsonWebKey | null = null;
  private nonce: string | null = null;

  private async ensureReady(): Promise<void> {
    if (this.keyPair && this.publicJwk) {
      return;
    }
    this.keyPair = await loadOrCreateKeyPair();
    this.publicJwk = await crypto.subtle.exportKey('jwk', this.keyPair.publicKey);
  }

  updateNonce(nonce: string): void {
    this.nonce = nonce;
  }

  async generateProof(method: string, url: string, accessToken?: string): Promise<string> {
    await this.ensureReady();

    const parsed = new URL(url, window.location.origin);
    const header = {
      typ: 'dpop+jwt',
      alg: 'ES256',
      jwk: this.publicJwk,
    };
    const payload: Record<string, unknown> = {
      jti: crypto.randomUUID(),
      htm: method.toUpperCase(),
      htu: `${parsed.origin}${parsed.pathname}`,
      iat: Math.floor(Date.now() / 1000),
    };

    if (accessToken) {
      const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(accessToken));
      payload['ath'] = this.base64Url(new Uint8Array(digest));
    }
    if (this.nonce) {
      payload['nonce'] = this.nonce;
    }

    const encodedHeader = this.base64Url(new TextEncoder().encode(JSON.stringify(header)));
    const encodedPayload = this.base64Url(new TextEncoder().encode(JSON.stringify(payload)));
    const signingInput = `${encodedHeader}.${encodedPayload}`;
    const signature = await crypto.subtle.sign(
      { name: 'ECDSA', hash: 'SHA-256' },
      this.keyPair!.privateKey,
      new TextEncoder().encode(signingInput),
    );

    return `${encodedHeader}.${encodedPayload}.${this.base64Url(new Uint8Array(signature))}`;
  }

  private base64Url(bytes: Uint8Array): string {
    let value = '';
    for (const byte of bytes) {
      value += String.fromCharCode(byte);
    }
    return btoa(value).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }
}
