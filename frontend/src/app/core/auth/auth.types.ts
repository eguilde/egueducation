import { InjectionToken } from '@angular/core';

export interface AuthConfig {
  authority: string;
  clientId: string;
  redirectUri: string;
  postLogoutRedirectUri: string;
  scope: string;
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
  roles: string[];
}

export const AUTH_CONFIG = new InjectionToken<AuthConfig>('AUTH_CONFIG');
