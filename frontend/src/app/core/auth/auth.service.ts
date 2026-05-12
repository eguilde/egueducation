import { DOCUMENT } from '@angular/common';
import { Inject, Injectable, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import * as oauth from 'oauth4webapi';
import { firstValueFrom } from 'rxjs';

import { ThemeService } from '../ui/theme.service';
import { AUTH_CONFIG, AuthConfig, StoredTokens, UserProfile } from './auth.types';
import { DPoPProofService } from './dpop-proof.service';
import {
  buildAuthorizationUrl,
  createClient,
  discover,
  exchangeCode,
  parseIdTokenClaims,
  refreshTokens,
  revokeToken,
  validateAuthResponse,
} from './oauth2-client';

const TOKENS_KEY = 'egueducation.tokens';
const CODE_VERIFIER_KEY = 'egueducation.pkce.verifier';
const STATE_KEY = 'egueducation.pkce.state';
const RETURN_URL_KEY = 'egueducation.auth.return_url';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly config = inject(AUTH_CONFIG);
  private readonly dpop = inject(DPoPProofService);
  private readonly theme = inject(ThemeService);
  private readonly document = inject(DOCUMENT);
  private readonly http = inject(HttpClient);

  private readonly profileSignal = signal<UserProfile | null>(null);
  private readonly tokensSignal = signal<StoredTokens | null>(null);
  private server: oauth.AuthorizationServer | null = null;
  private dpopHandle: oauth.DPoPHandle | null = null;

  readonly profile = this.profileSignal.asReadonly();
  readonly tokens = this.tokensSignal.asReadonly();
  readonly isAuthenticated = computed(() => !!this.profileSignal());

  async init(): Promise<void> {
    try {
      await this.ensureAuthRuntime();
    } catch {
      this.server = null;
      this.dpopHandle = null;
      this.clearLocalSession();
      return;
    }
    const stored = localStorage.getItem(TOKENS_KEY);
    if (!stored) {
      return;
    }
    const tokens = JSON.parse(stored) as StoredTokens;
    if (tokens.expires_at <= Math.floor(Date.now() / 1000) + 30 && tokens.refresh_token) {
      await this.refresh();
      return;
    }
    this.setSession(tokens);
  }

  async login(returnUrl?: string): Promise<void> {
    await this.ensureAuthRuntime();
    const verifier = oauth.generateRandomCodeVerifier();
    const { url, state } = await buildAuthorizationUrl(this.server!, this.resolvedConfig(), verifier);
    this.storeRedirectLogin(state, verifier, returnUrl);
    this.document.location.href = url.toString();
  }

  async handleCallback(query: URLSearchParams): Promise<void> {
    await this.ensureAuthRuntime();
    const verifier = sessionStorage.getItem(CODE_VERIFIER_KEY);
    const state = sessionStorage.getItem(STATE_KEY);
    if (!verifier || !state) {
      throw new Error('Missing PKCE verifier or state');
    }

    const validated = validateAuthResponse(this.server!, this.resolvedConfig(), query, state);
    const tokens = await exchangeCode(this.server!, this.resolvedConfig(), validated, verifier, this.dpopHandle ?? undefined);
    sessionStorage.removeItem(CODE_VERIFIER_KEY);
    sessionStorage.removeItem(STATE_KEY);
    this.setSession(tokens);
    await firstValueFrom(this.http.post<{ status: string }>('/api/auth/session/exchange', {}));
  }

  async logout(): Promise<void> {
    if (this.server === null) {
      try {
        await this.ensureAuthRuntime();
      } catch {
        this.server = null;
        this.dpopHandle = null;
      }
    }
    const tokens = this.tokensSignal();
    this.clearLocalSession();

    if (tokens?.refresh_token && this.server) {
      try {
        await revokeToken(this.server!, this.resolvedConfig(), tokens.refresh_token, this.dpopHandle ?? undefined);
      } catch {
        // Best-effort token revocation must not block logout UX.
      }
    }
    if (tokens?.access_token && this.server) {
      try {
        await revokeToken(this.server!, this.resolvedConfig(), tokens.access_token, this.dpopHandle ?? undefined);
      } catch {
        // Best-effort token revocation must not block logout UX.
      }
    }

    try {
      await firstValueFrom(this.http.post<{ status: string }>('/api/auth/logout', {}));
    } catch {
      // The browser should still leave the authenticated area even if backend logout fails.
    }

    this.document.location.href = this.config.postLogoutRedirectUri;
  }

  clearLocalSession(): void {
    localStorage.removeItem(TOKENS_KEY);
    this.profileSignal.set(null);
    this.tokensSignal.set(null);
  }

  async refresh(): Promise<void> {
    await this.ensureAuthRuntime();
    const tokens = this.tokensSignal() ?? (JSON.parse(localStorage.getItem(TOKENS_KEY) ?? 'null') as StoredTokens | null);
    if (!tokens?.refresh_token) {
      return;
    }
    const refreshed = await refreshTokens(
      this.server!,
      this.resolvedConfig(),
      tokens.refresh_token,
      this.dpopHandle ?? undefined,
    );
    this.setSession(refreshed);
  }

  async authHeaders(method: string, url: string): Promise<Record<string, string>> {
    const tokens = this.tokensSignal();
    if (!tokens) {
      return {};
    }
    const proof = await this.dpop.generateProof(method, url, tokens.access_token);
    return {
      Authorization: `DPoP ${tokens.access_token}`,
      DPoP: proof,
      'Accept-Language': this.theme.language(),
    };
  }

  private setSession(tokens: StoredTokens): void {
    localStorage.setItem(TOKENS_KEY, JSON.stringify(tokens));
    this.tokensSignal.set(tokens);
    if (tokens.id_token) {
      this.profileSignal.set(parseIdTokenClaims(tokens.id_token));
    }
  }

  private absoluteAuthority(authority: string): string {
    return new URL(authority, window.location.origin).toString();
  }

  storeReturnUrl(returnUrl?: string): void {
    if (returnUrl && returnUrl.startsWith('/')) {
      sessionStorage.setItem(RETURN_URL_KEY, returnUrl);
      return;
    }
    const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`;
    if (!currentPath.startsWith('/login') && !currentPath.startsWith('/auth/')) {
      sessionStorage.setItem(RETURN_URL_KEY, currentPath || '/dashboard');
    }
  }

  consumeReturnUrl(): string {
    const returnUrl = sessionStorage.getItem(RETURN_URL_KEY) || '/dashboard';
    sessionStorage.removeItem(RETURN_URL_KEY);
    return returnUrl.startsWith('/') ? returnUrl : '/dashboard';
  }

  private async ensureAuthRuntime(): Promise<void> {
    if (this.server && this.dpopHandle) {
      return;
    }
    this.server = await discover(this.absoluteAuthority(this.config.authority));
    this.dpopHandle = oauth.DPoP(
      createClient(this.resolvedConfig()),
      await oauth.generateKeyPair('ES256'),
    );
  }

  private storeRedirectLogin(state: string, verifier: string, explicitReturnUrl?: string): void {
    this.storeReturnUrl(explicitReturnUrl);
    sessionStorage.setItem(CODE_VERIFIER_KEY, verifier);
    sessionStorage.setItem(STATE_KEY, state);
  }

  private resolvedConfig(): AuthConfig {
    return { ...this.config, authority: this.absoluteAuthority(this.config.authority) };
  }
}
