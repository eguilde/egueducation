import { DOCUMENT } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { DestroyRef, Injectable, NgZone, computed, inject, signal } from '@angular/core';
import * as oauth from 'oauth4webapi';

import { APP_ENVIRONMENT } from '../config/app-environment';
import { ThemeService } from '../ui/theme.service';
import { AUTH_CONFIG, AuthConfig, StoredTokens, UserProfile } from './auth.types';
import { DPoPProofService } from './dpop-proof.service';
import {
  buildAuthorizationUrl,
  clearDiscoveryCache,
  createClient,
  discover,
  exchangeCode,
  parseIdTokenClaims,
  refreshTokens,
  revokeToken,
  validateAuthResponse,
} from './oauth2-client';
import { clearKeyPair, loadOrCreateKeyPair } from './dpop-key-store';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly config = inject(AUTH_CONFIG);
  private readonly environment = inject(APP_ENVIRONMENT);
  private readonly dpopProof = inject(DPoPProofService);
  private readonly theme = inject(ThemeService);
  private readonly document = inject(DOCUMENT);
  private readonly http = inject(HttpClient);
  private readonly destroyRef = inject(DestroyRef);
  private readonly ngZone = inject(NgZone);

  private readonly profileSignal = signal<UserProfile | null>(null);
  private readonly accessTokenSignal = signal<string | null>(null);
  private readonly expiresAtSignal = signal<number | null>(null);
  private readonly loginInProgressSignal = signal(false);
  private readonly storagePrefix = this.config.storagePrefix ?? 'egueducation_auth';
  private readonly redirectVerifierKey = this.storageKey('redirect_verifier');
  private readonly redirectStateKey = this.storageKey('redirect_state');
  private readonly returnUrlKey = this.storageKey('return_url');
  private server: oauth.AuthorizationServer | null = null;
  private dpopHandle: oauth.DPoPHandle | null = null;
  private renewTimer: ReturnType<typeof setTimeout> | null = null;
  private currentRefreshToken: string | null = null;
  private refreshPromise: Promise<boolean> | null = null;

  readonly profile = this.profileSignal.asReadonly();
  readonly accessToken = this.accessTokenSignal.asReadonly();
  readonly isAuthenticated = computed(() => {
    const token = this.accessTokenSignal();
    const expiresAt = this.expiresAtSignal();
    return !!token && !!expiresAt && Date.now() / 1000 < expiresAt;
  });
  readonly loginInProgress = this.loginInProgressSignal.asReadonly();

  constructor() {
    this.destroyRef.onDestroy(() => {
      if (this.renewTimer) {
        clearTimeout(this.renewTimer);
        this.renewTimer = null;
      }
    });
  }

  async init(): Promise<void> {
    this.cleanupLegacyStorage();
    this.setupVisibilityRefresh();

    if (this.config.useDPoP) {
      await this.ensureDpopHandle();
    }

    try {
      await this.ensureServer();
    } catch {
      this.server = null;
      this.dpopHandle = null;
      this.clearLocalSession();
      this.scheduleRetry(30);
      return;
    }

    const hasSession = this.readStorage('has_session');
    if (!hasSession) {
      return;
    }

    const storedIdToken = this.readStorage('id_token');
    if (storedIdToken) {
      try {
        this.profileSignal.set(parseIdTokenClaims(storedIdToken));
      } catch {
        this.removeStorage('id_token');
      }
    }

    const storedAccessToken = this.readStorage('access_token');
    const storedExpiresAt = this.readStorage('expires_at');
    let hasValidStoredAccessToken = false;
    if (storedAccessToken && storedExpiresAt) {
      const parsedExpiresAt = Number(storedExpiresAt);
      if (Number.isFinite(parsedExpiresAt) && parsedExpiresAt > Date.now() / 1000) {
        this.accessTokenSignal.set(storedAccessToken);
        this.expiresAtSignal.set(parsedExpiresAt);
        this.scheduleRenewal(parsedExpiresAt);
        hasValidStoredAccessToken = true;
      } else {
        this.removeStorage('access_token');
        this.removeStorage('expires_at');
      }
    }

    const storedRefreshToken = this.readStorage('refresh_token');
    if (storedRefreshToken) {
      this.currentRefreshToken = storedRefreshToken;
    }

    if (hasValidStoredAccessToken && !storedRefreshToken) {
      return;
    }

    const refreshed = await this.tryRefresh();
    if (!refreshed) {
      if (hasValidStoredAccessToken) {
        return;
      }
      this.clearLocalSession();
    }
  }

  async login(returnUrl?: string): Promise<void> {
    if (this.loginInProgressSignal()) {
      return;
    }
    this.loginInProgressSignal.set(true);
    try {
      await this.ensureServer();
      const dpop = await this.ensureDpopHandle();
      const verifier = oauth.generateRandomCodeVerifier();
      const themeScheme = this.theme.colorScheme();
      const themeDark = this.theme.isDarkMode();
      const { url, state } = await buildAuthorizationUrl(
        this.server!,
        this.resolvedConfig(),
        verifier,
        {
          ui_theme_scheme: themeScheme,
          ui_theme_dark: String(themeDark),
          ui_theme_primary: this.theme.selectedPrimaryColor(),
          ui_theme_surface: this.theme.selectedSurface(),
        },
        dpop ?? undefined,
      );
      this.storeRedirectLogin(state, verifier, returnUrl);
      this.document.location.href = url.toString();
    } finally {
      this.loginInProgressSignal.set(false);
    }
  }

  async handleCallback(query: URLSearchParams): Promise<void> {
    await this.ensureServer();
    const verifier = sessionStorage.getItem(this.redirectVerifierKey);
    const state = sessionStorage.getItem(this.redirectStateKey);
    if (!verifier || !state) {
      throw new Error('Missing PKCE verifier or state');
    }

    const validated = validateAuthResponse(this.server!, this.resolvedConfig(), query, state);
    try {
      const tokens = await exchangeCode(
        this.server!,
        this.resolvedConfig(),
        validated,
        verifier,
        this.dpopHandle ?? undefined,
      );
      this.setSession(tokens);
      this.scheduleRenewal(tokens.expires_at);
    } finally {
      sessionStorage.removeItem(this.redirectVerifierKey);
      sessionStorage.removeItem(this.redirectStateKey);
    }
  }

  async logout(): Promise<void> {
    const idToken = this.readStorage('id_token');
    const accessToken = this.accessTokenSignal();

    try {
      await this.ensureServer();
    } catch {
      this.server = null;
      this.dpopHandle = null;
    }

    const refreshToken = this.currentRefreshToken ?? this.readStorage('refresh_token');

    if (accessToken) {
      try {
        await fetch(this.resolveApiUrl('/api/auth/logout'), {
          method: 'POST',
          headers: { Authorization: `Bearer ${accessToken}` },
          credentials: 'include',
        });
      } catch {
        // Best-effort backend logout.
      }
    }

    if (refreshToken && this.server) {
      try {
        await revokeToken(this.server, this.resolvedConfig(), refreshToken, this.dpopHandle ?? undefined);
      } catch {
        // Best-effort token revocation must not block logout UX.
      }
    }
    if (accessToken && this.server) {
      try {
        await revokeToken(this.server, this.resolvedConfig(), accessToken, this.dpopHandle ?? undefined);
      } catch {
        // Best-effort token revocation must not block logout UX.
      }
    }

    if (this.config.useDPoP) {
      await clearKeyPair(this.config.indexedDbName);
      this.dpopHandle = null;
    }

    this.clearLocalSession();
    clearDiscoveryCache();

    try {
      await this.ensureServer();
      if (this.server?.end_session_endpoint && idToken) {
        const logoutUrl = new URL(this.server.end_session_endpoint);
        logoutUrl.searchParams.set('id_token_hint', idToken);
        logoutUrl.searchParams.set(
          'post_logout_redirect_uri',
          this.config.postLogoutRedirectUri ?? window.location.origin,
        );
        this.document.location.href = logoutUrl.toString();
        return;
      }
    } catch {
      // Fall back to provider alias.
    }

    const logoutUrl = new URL('/logout', this.absoluteAuthority(this.config.authority));
    logoutUrl.searchParams.set('returnTo', this.config.postLogoutRedirectUri ?? window.location.origin);
    this.document.location.href = logoutUrl.toString();
  }

  clearLocalSession(): void {
    this.profileSignal.set(null);
    this.accessTokenSignal.set(null);
    this.expiresAtSignal.set(null);
    this.currentRefreshToken = null;
    this.removeStorage('id_token');
    this.removeStorage('refresh_token');
    this.removeStorage('has_session');
    if (this.renewTimer) {
      clearTimeout(this.renewTimer);
      this.renewTimer = null;
    }
  }

  async refresh(): Promise<void> {
    await this.tryRefresh();
  }

  async authHeaders(method: string, url: string): Promise<Record<string, string>> {
    const token = this.accessTokenSignal();
    if (!token) {
      return {};
    }

    if (!this.config.useDPoP) {
      return {
        Authorization: `Bearer ${token}`,
        'Accept-Language': this.theme.language(),
      };
    }

    const proof = await this.dpopProof.generateProof(method, url, token);
    return {
      Authorization: `DPoP ${token}`,
      DPoP: proof,
      'Accept-Language': this.theme.language(),
    };
  }

  private setSession(tokens: StoredTokens): void {
    this.accessTokenSignal.set(tokens.access_token);
    this.expiresAtSignal.set(tokens.expires_at);
    this.currentRefreshToken = tokens.refresh_token ?? null;
    this.writeStorage('has_session', '1');
    this.writeStorage('access_token', tokens.access_token);
    this.writeStorage('expires_at', String(tokens.expires_at));
    if (tokens.refresh_token) {
      this.writeStorage('refresh_token', tokens.refresh_token);
    } else {
      this.removeStorage('refresh_token');
    }
    if (tokens.id_token) {
      this.writeStorage('id_token', tokens.id_token);
      this.profileSignal.set(parseIdTokenClaims(tokens.id_token));
    }
  }

  private absoluteAuthority(authority: string): string {
    return new URL(authority, window.location.origin).toString();
  }

  private resolveApiUrl(requestUrl: string): string {
    const apiBaseUrl = this.environment.apiBaseUrl.replace(/\/$/, '');
    if (/^https?:\/\//i.test(apiBaseUrl)) {
      return `${apiBaseUrl}${requestUrl.slice('/api'.length)}`;
    }
    return requestUrl;
  }

  storeReturnUrl(returnUrl?: string): void {
    if (returnUrl && returnUrl.startsWith('/')) {
      sessionStorage.setItem(this.returnUrlKey, returnUrl);
      return;
    }
    const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`;
    if (!currentPath.startsWith('/callback') && !currentPath.startsWith('/auth/')) {
      sessionStorage.setItem(this.returnUrlKey, currentPath || '/dashboard');
    }
  }

  consumeReturnUrl(): string {
    const returnUrl = sessionStorage.getItem(this.returnUrlKey) || '/dashboard';
    sessionStorage.removeItem(this.returnUrlKey);
    return returnUrl.startsWith('/') ? returnUrl : '/dashboard';
  }

  async tryRefresh(): Promise<boolean> {
    if (this.refreshPromise) {
      return this.refreshPromise;
    }
    this.refreshPromise = this.doRefresh();
    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async doRefresh(): Promise<boolean> {
    const refreshToken = this.currentRefreshToken;
    const hasSession = !!this.readStorage('has_session');
    if (!refreshToken && !hasSession) {
      return false;
    }

    try {
      await this.ensureServer();
      const refreshed = await refreshTokens(
        this.server!,
        this.resolvedConfig(),
        refreshToken ?? 'cookie',
        this.dpopHandle ?? undefined,
      );
      this.setSession(refreshed);
      this.scheduleRenewal(refreshed.expires_at);
      return true;
    } catch (error) {
      if (error instanceof Error && error.message.includes('invalid_grant')) {
        this.clearLocalSession();
        return false;
      }
      this.scheduleRetry(30);
      return false;
    }
  }

  private async ensureServer(): Promise<void> {
    if (this.server) {
      return;
    }
    this.server = await discover(this.absoluteAuthority(this.config.authority));
  }

  private async ensureDpopHandle(): Promise<oauth.DPoPHandle | null> {
    if (!this.config.useDPoP) {
      return null;
    }
    if (this.dpopHandle) {
      return this.dpopHandle;
    }
    const keyPair = await loadOrCreateKeyPair(this.config.indexedDbName);
    this.dpopHandle = oauth.DPoP(createClient(this.resolvedConfig()), keyPair);
    return this.dpopHandle;
  }

  private storeRedirectLogin(state: string, verifier: string, explicitReturnUrl?: string): void {
    this.storeReturnUrl(explicitReturnUrl);
    sessionStorage.setItem(this.redirectVerifierKey, verifier);
    sessionStorage.setItem(this.redirectStateKey, state);
  }

  private resolvedConfig(): AuthConfig {
    return { ...this.config, authority: this.absoluteAuthority(this.config.authority) };
  }

  private storageKey(field: string): string {
    return `${this.storagePrefix}_${field}`;
  }

  private readStorage(field: string): string | null {
    return localStorage.getItem(this.storageKey(field));
  }

  private writeStorage(field: string, value: string): void {
    localStorage.setItem(this.storageKey(field), value);
  }

  private removeStorage(field: string): void {
    localStorage.removeItem(this.storageKey(field));
  }

  private scheduleRenewal(expiresAt: number): void {
    if (this.renewTimer) {
      clearTimeout(this.renewTimer);
    }
    const renewBefore = this.config.renewBeforeExpirySec ?? 60;
    const msUntilRenewal = (expiresAt - renewBefore - Date.now() / 1000) * 1000;
    const delay = Math.max(0, msUntilRenewal);
    this.ngZone.runOutsideAngular(() => {
      this.renewTimer = setTimeout(() => {
        void this.ngZone.run(() => this.tryRefresh());
      }, delay);
    });
  }

  private scheduleRetry(delaySec: number): void {
    if (this.renewTimer) {
      clearTimeout(this.renewTimer);
    }
    this.ngZone.runOutsideAngular(() => {
      this.renewTimer = setTimeout(() => {
        void this.ngZone.run(() => this.tryRefresh());
      }, delaySec * 1000);
    });
  }

  private setupVisibilityRefresh(): void {
    let wasHidden = false;
    const handler = () => {
      if (document.visibilityState === 'hidden') {
        wasHidden = true;
        return;
      }
      if (!wasHidden) {
        return;
      }
      wasHidden = false;
      const expiresAt = this.expiresAtSignal();
      if (!expiresAt) {
        return;
      }
      const renewBefore = this.config.renewBeforeExpirySec ?? 60;
      if (Date.now() / 1000 >= expiresAt - renewBefore) {
        void this.tryRefresh();
      }
    };
    document.addEventListener('visibilitychange', handler);
    this.destroyRef.onDestroy(() => document.removeEventListener('visibilitychange', handler));
  }

  private cleanupLegacyStorage(): void {
    for (let index = 0; index < localStorage.length; index += 1) {
      const key = localStorage.key(index);
      if (key && /^\d+-\w+-spa$/.test(key)) {
        localStorage.removeItem(key);
      }
    }
    localStorage.removeItem('egueducation.tokens');
    localStorage.removeItem('egueducation-dpop-es256');
  }
}
