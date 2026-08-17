import * as oauth from 'oauth4webapi';
import type { OidcConfig } from './config';
import { readThemePreferences, resolveDarkScheme } from '../theme/preferences';

const key = (name: string) => `egueducation.oidc.${name}`;
const authorizationTransactionKey = (state: string) => key(`authorization.${state}`);
const AUTHORIZATION_TRANSACTION_MAX_AGE_MS = 30 * 60 * 1000;
let server: oauth.AuthorizationServer | undefined;
let completionPromise: Promise<Tokens> | undefined;
let refreshPromise: Promise<Tokens | null> | undefined;
let authorizationStartPromise: Promise<void> | undefined;

interface AuthorizationTransaction {
  verifier: string;
  nonce: string;
  returnTo: string;
  createdAt: number;
}

function readAuthorizationTransaction(state: string): AuthorizationTransaction | undefined {
  const raw = sessionStorage.getItem(authorizationTransactionKey(state));
  if (!raw) return undefined;
  try {
    const value = JSON.parse(raw) as Partial<AuthorizationTransaction>;
    if (
      typeof value.verifier !== 'string' || !value.verifier ||
      typeof value.nonce !== 'string' || !value.nonce ||
      typeof value.returnTo !== 'string' || !value.returnTo.startsWith('/') ||
      typeof value.createdAt !== 'number' || Date.now() - value.createdAt > AUTHORIZATION_TRANSACTION_MAX_AGE_MS
    ) {
      sessionStorage.removeItem(authorizationTransactionKey(state));
      return undefined;
    }
    return value as AuthorizationTransaction;
  } catch {
    sessionStorage.removeItem(authorizationTransactionKey(state));
    return undefined;
  }
}

const insecureOptions = (authority: string) => {
  const url = new URL(authority, window.location.origin);
  if (url.protocol === 'http:' && !['localhost', '127.0.0.1'].includes(url.hostname)) throw new Error('OIDC authority must use HTTPS');
  return url.protocol === 'http:' ? { [oauth.allowInsecureRequests]: true as const } : {};
};
const client = (config: OidcConfig): oauth.Client => ({ client_id: config.clientId, [oauth.clockTolerance]: 30 });
const credentialsFetch: typeof fetch = (input, init) => fetch(input, { ...init, credentials: 'include' });

export async function discover(config: OidcConfig) {
  if (server) return server;
  const issuer = new URL(config.authority, window.location.origin);
  const response = await oauth.discoveryRequest(issuer, { algorithm: 'oidc', ...insecureOptions(issuer.toString()) });
  server = await oauth.processDiscoveryResponse(issuer, response);
  return server;
}

export async function prepareAuthorization(config: OidcConfig, returnTo = '/') {
  const authorizationServer = await discover(config);
  if (!authorizationServer.authorization_endpoint) throw new Error('OIDC provider has no authorization endpoint');
  const verifier = oauth.generateRandomCodeVerifier();
  const state = oauth.generateRandomState();
  const nonce = oauth.generateRandomNonce();
  const safeReturnTo = returnTo.startsWith('/') ? returnTo : '/';
  sessionStorage.setItem(authorizationTransactionKey(state), JSON.stringify({
    verifier,
    nonce,
    returnTo: safeReturnTo,
    createdAt: Date.now(),
  } satisfies AuthorizationTransaction));
  const theme = readThemePreferences();
  const parameters = new URLSearchParams({
    client_id: config.clientId,
    redirect_uri: config.redirectUri,
    response_type: 'code',
    scope: config.scope,
    state,
    nonce,
    code_challenge: await oauth.calculatePKCECodeChallenge(verifier),
    code_challenge_method: 'S256',
    // The OIDC UI may consume these optional hints. The theme cookie is also
    // available on this origin for an authorization UI that does not forward query parameters.
    ui_theme_scheme: theme.scheme,
    ui_theme_preset: theme.preset,
    ui_theme_primary: theme.primary,
    ui_theme_surface: theme.surface,
    ui_theme_dark: resolveDarkScheme(theme.scheme) ? '1' : '0',
  });
  return `${authorizationServer.authorization_endpoint}?${parameters.toString()}`;
}

export async function beginAuthorization(config: OidcConfig, returnTo = '/') {
  authorizationStartPromise ??= prepareAuthorization(config, returnTo)
    .then((authorizationUrl) => window.location.assign(authorizationUrl))
    .catch((error) => {
      authorizationStartPromise = undefined;
      throw error;
    });
  return authorizationStartPromise;
}

// RP-Initiated Logout uses a separate one-time transaction. The redirect URI
// is an exact registered client metadata value; state is validated locally so
// a forged provider redirect cannot be mistaken for a completed logout.
export async function prepareLogout(config: OidcConfig, idToken: string) {
  const authorizationServer = await discover(config);
  if (!authorizationServer.end_session_endpoint) throw new Error('OIDC provider has no RP-initiated logout endpoint');
  if (!idToken) throw new Error('Lipsește ID token-ul necesar pentru logout OIDC.');
  const state = oauth.generateRandomState();
  sessionStorage.setItem(key('logoutState'), state);
  const parameters = new URLSearchParams({
    id_token_hint: idToken,
    post_logout_redirect_uri: config.logoutRedirectUri,
    state,
    client_id: config.clientId
  });
  return `${authorizationServer.end_session_endpoint}?${parameters.toString()}`;
}

export async function beginLogout(config: OidcConfig, idToken: string) {
  window.location.assign(await prepareLogout(config, idToken));
}

export function completeLogout(query = new URLSearchParams(window.location.search)) {
  const expected = sessionStorage.getItem(key('logoutState'));
  const received = query.get('state');
  sessionStorage.removeItem(key('logoutState'));
  if (!expected || !received || received !== expected) throw new Error('Răspunsul de logout OIDC nu corespunde sesiunii inițiate.');
}

export interface Tokens { accessToken: string; expiresAt: number; idToken?: string; returnTo?: string }
async function runAuthorizationCompletion(config: OidcConfig, query: URLSearchParams): Promise<Tokens> {
  const state = query.get('state');
  const transaction = state ? readAuthorizationTransaction(state) : undefined;
  if (!state || !transaction) throw new Error('Sesiunea de autentificare a expirat. Reîncepeți autentificarea.');
  const authorizationServer = await discover(config);
  try {
    const validated = oauth.validateAuthResponse(authorizationServer, client(config), query, state);
    const response = await oauth.authorizationCodeGrantRequest(authorizationServer, client(config), oauth.None(), validated, config.redirectUri, transaction.verifier, { ...insecureOptions(config.authority), [oauth.customFetch]: credentialsFetch });
    const token = await oauth.processAuthorizationCodeResponse(authorizationServer, client(config), response, { expectedNonce: transaction.nonce, requireIdToken: true });
    sessionStorage.removeItem(authorizationTransactionKey(state));
    return { accessToken: token.access_token, idToken: token.id_token, expiresAt: Math.floor(Date.now() / 1000) + (token.expires_in ?? 900), returnTo: transaction.returnTo };
  } catch (error) {
    // Preserve the one-time PKCE transaction only for a transient browser/network
    // failure so the callback can be retried without asking for another OTP.
    if (!(error instanceof TypeError)) {
      sessionStorage.removeItem(authorizationTransactionKey(state));
    }
    throw error;
  }
}

export function completeAuthorization(config: OidcConfig, query = new URLSearchParams(window.location.search)): Promise<Tokens> {
  completionPromise ??= runAuthorizationCompletion(config, query).finally(() => { completionPromise = undefined; });
  return completionPromise;
}

async function runCookieRefresh(config: OidcConfig): Promise<Tokens | null> {
  try {
    const authorizationServer = await discover(config);
    const response = await oauth.refreshTokenGrantRequest(authorizationServer, client(config), oauth.None(), 'cookie', { ...insecureOptions(config.authority), [oauth.customFetch]: credentialsFetch });
    const token = await oauth.processRefreshTokenResponse(authorizationServer, client(config), response);
    return { accessToken: token.access_token, idToken: token.id_token, expiresAt: Math.floor(Date.now() / 1000) + (token.expires_in ?? 900) };
  } catch { return null; }
}

export function refreshWithCookie(config: OidcConfig): Promise<Tokens | null> {
  refreshPromise ??= runCookieRefresh(config).finally(() => { refreshPromise = undefined; });
  return refreshPromise;
}
