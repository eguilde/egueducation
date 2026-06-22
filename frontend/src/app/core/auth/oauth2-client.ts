import * as oauth from 'oauth4webapi';

import { AuthConfig, StoredTokens, UserProfile } from './auth.types';

let cachedServer: oauth.AuthorizationServer | null = null;
let cachedAuthority = '';

function requestOptions(authority: string): { [oauth.allowInsecureRequests]?: true } {
  const local =
    authority.startsWith('http://localhost') ||
    authority.startsWith('http://127.0.0.1') ||
    authority.startsWith('http://[::1]');
  return local ? { [oauth.allowInsecureRequests]: true } : {};
}

function credentialFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  return fetch(input, { ...init, credentials: 'include' });
}

function tokenRequestOptions(authority: string): Record<string | symbol, unknown> {
  return {
    ...requestOptions(authority),
    [oauth.customFetch]: credentialFetch,
  };
}

function client(config: AuthConfig): oauth.Client {
  return { client_id: config.clientId, [oauth.clockTolerance]: 30 };
}

export function createClient(config: AuthConfig): oauth.Client {
  return client(config);
}

export async function discover(authority: string): Promise<oauth.AuthorizationServer> {
  if (cachedServer && cachedAuthority === authority) {
    return cachedServer;
  }
  const issuer = new URL(authority, window.location.origin);
  const metadataUrl = new URL(`${issuer.toString().replace(/\/$/, '')}/.well-known/openid-configuration`);
  const response = await credentialFetch(metadataUrl);
  cachedServer = await oauth.processDiscoveryResponse(issuer, response);
  cachedAuthority = authority;
  return cachedServer;
}

export async function buildAuthorizationUrl(
  server: oauth.AuthorizationServer,
  config: AuthConfig,
  codeVerifier: string,
): Promise<{ url: URL; state: string }> {
  const codeChallenge = await oauth.calculatePKCECodeChallenge(codeVerifier);
  const state = oauth.generateRandomState();
  const params = new URLSearchParams({
    client_id: config.clientId,
    redirect_uri: config.redirectUri,
    response_type: 'code',
    scope: config.scope,
    code_challenge: codeChallenge,
    code_challenge_method: 'S256',
    state,
  });

  const url = new URL(server.authorization_endpoint!);
  params.forEach((value, key) => url.searchParams.set(key, value));
  return { url, state };
}

export function validateAuthResponse(
  server: oauth.AuthorizationServer,
  config: AuthConfig,
  params: URLSearchParams,
  expectedState: string,
): URLSearchParams {
  return oauth.validateAuthResponse(server, client(config), params, expectedState);
}

export async function exchangeCode(
  server: oauth.AuthorizationServer,
  config: AuthConfig,
  params: URLSearchParams,
  codeVerifier: string,
  dpop?: oauth.DPoPHandle,
): Promise<StoredTokens> {
const response = await oauth.authorizationCodeGrantRequest(
    server,
    client(config),
    oauth.None(),
    params,
    config.redirectUri,
    codeVerifier,
    { ...tokenRequestOptions(config.authority), ...(dpop ? { DPoP: dpop } : {}) },
  );
  const result = await oauth.processAuthorizationCodeResponse(server, client(config), response);
  const now = Math.floor(Date.now() / 1000);
  const refreshToken = result.refresh_token ?? 'cookie';
  return {
    access_token: result.access_token,
    refresh_token: refreshToken,
    id_token: result.id_token ?? undefined,
    expires_at: now + (result.expires_in ?? 900),
  };
}

export async function refreshTokens(
  server: oauth.AuthorizationServer,
  config: AuthConfig,
  refreshToken: string,
  dpop?: oauth.DPoPHandle,
): Promise<StoredTokens> {
  const response = await oauth.refreshTokenGrantRequest(
    server,
    client(config),
    oauth.None(),
    refreshToken,
    { ...tokenRequestOptions(config.authority), ...(dpop ? { DPoP: dpop } : {}) },
  );
  const result = await oauth.processRefreshTokenResponse(server, client(config), response);
  const now = Math.floor(Date.now() / 1000);
  return {
    access_token: result.access_token,
    refresh_token: result.refresh_token ?? refreshToken,
    id_token: result.id_token ?? undefined,
    expires_at: now + (result.expires_in ?? 900),
  };
}

export async function revokeToken(
  server: oauth.AuthorizationServer,
  config: AuthConfig,
  token: string,
  dpop?: oauth.DPoPHandle,
): Promise<void> {
  const response = await oauth.revocationRequest(
    server,
    client(config),
    oauth.None(),
    token,
    { ...tokenRequestOptions(config.authority), ...(dpop ? { DPoP: dpop } : {}) } as oauth.RevocationRequestOptions,
  );
  await oauth.processRevocationResponse(response);
}

export function parseIdTokenClaims(idToken: string): UserProfile {
  const [, payload] = idToken.split('.');
  const claims = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/'))) as Record<string, unknown>;
  return {
    sub: String(claims['sub']),
    email: typeof claims['email'] === 'string' ? claims['email'] : undefined,
    name:
      typeof claims['name'] === 'string'
        ? claims['name']
        : [claims['given_name'], claims['family_name']].filter(Boolean).join(' ') || undefined,
    phone_number: typeof claims['phone_number'] === 'string' ? claims['phone_number'] : undefined,
    roles: Array.isArray(claims['roles']) ? claims['roles'].filter((value): value is string => typeof value === 'string') : [],
  };
}
