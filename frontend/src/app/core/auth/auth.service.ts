import { DOCUMENT } from '@angular/common';
import { Inject, Injectable, computed, inject, signal } from '@angular/core';
import * as oauth from 'oauth4webapi';

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

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly config = inject(AUTH_CONFIG);
  private readonly dpop = inject(DPoPProofService);
  private readonly theme = inject(ThemeService);
  private readonly document = inject(DOCUMENT);

  private readonly profileSignal = signal<UserProfile | null>(null);
  private readonly tokensSignal = signal<StoredTokens | null>(null);
  private server: oauth.AuthorizationServer | null = null;
  private dpopHandle: oauth.DPoPHandle | null = null;

  readonly profile = this.profileSignal.asReadonly();
  readonly tokens = this.tokensSignal.asReadonly();
  readonly isAuthenticated = computed(() => !!this.profileSignal());

  async init(): Promise<void> {
    this.server = await discover(this.absoluteAuthority(this.config.authority));
    this.dpopHandle = oauth.DPoP(
      createClient(this.resolvedConfig()),
      await oauth.generateKeyPair('ES256'),
    );
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

  async login(): Promise<void> {
    const verifier = oauth.generateRandomCodeVerifier();
    const { url, state } = await buildAuthorizationUrl(this.server!, this.resolvedConfig(), verifier);
    sessionStorage.setItem(CODE_VERIFIER_KEY, verifier);
    sessionStorage.setItem(STATE_KEY, state);
    this.document.location.href = url.toString();
  }

  async handleCallback(query: URLSearchParams): Promise<void> {
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
  }

  async logout(): Promise<void> {
    const tokens = this.tokensSignal();
    if (tokens?.refresh_token) {
      await revokeToken(this.server!, this.resolvedConfig(), tokens.refresh_token, this.dpopHandle ?? undefined);
    }
    if (tokens?.access_token) {
      await revokeToken(this.server!, this.resolvedConfig(), tokens.access_token, this.dpopHandle ?? undefined);
    }
    localStorage.removeItem(TOKENS_KEY);
    this.profileSignal.set(null);
    this.tokensSignal.set(null);
    this.document.location.href = this.config.postLogoutRedirectUri;
  }

  async refresh(): Promise<void> {
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

  private resolvedConfig(): AuthConfig {
    return { ...this.config, authority: this.absoluteAuthority(this.config.authority) };
  }
}
