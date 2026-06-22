import { InjectionToken } from '@angular/core';

export interface AuthConfig {
  authority: string;
  clientId: string;
  redirectUri: string;
  postLogoutRedirectUri?: string;
  logoutRedirectRoute?: string;
  scope: string;
  usePar?: boolean;
  renewBeforeExpirySec?: number;
  storagePrefix?: string;
  secureRoutes: string[];
  useDPoP?: boolean;
  indexedDbName?: string;
  oidcPathPrefix?: string;
  loginHandler?: (authUrl: string) => Promise<URLSearchParams>;
}

export interface StoredTokens {
  access_token: string;
  refresh_token?: string;
  id_token?: string;
  expires_at: number;
}

export interface UserProfile {
  sub: string;
  email?: string;
  name?: string;
  phone_number?: string;
  initials?: string;
  roles: string[];
  [key: string]: unknown;
}

export const AUTH_CONFIG = new InjectionToken<AuthConfig>('AUTH_CONFIG');
