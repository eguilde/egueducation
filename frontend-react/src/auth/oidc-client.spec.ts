import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { completeAuthorization, completeLogout, prepareAuthorization, prepareLogout } from './oidc-client';

const config = { authority: 'https://issuer.example', clientId: 'spa', redirectUri: 'https://app.example/auth/callback', logoutRedirectUri: 'https://app.example/auth/logout', scope: 'openid profile offline_access', apiBaseUrl: '/api' };
const discovery = {
  issuer: config.authority,
  authorization_endpoint: `${config.authority}/authorize`,
  token_endpoint: `${config.authority}/token`,
  end_session_endpoint: `${config.authority}/session/end`,
  jwks_uri: `${config.authority}/jwks`,
  response_types_supported: ['code'],
  subject_types_supported: ['public'],
  id_token_signing_alg_values_supported: ['RS256'],
  grant_types_supported: ['authorization_code', 'refresh_token'],
  code_challenge_methods_supported: ['S256']
};
const runtimeCrypto = globalThis.crypto;

describe('OIDC authorization', () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.localStorage.clear();
    vi.restoreAllMocks();
  });
  afterEach(() => vi.unstubAllGlobals());

  const useDeterministicDigest = () => vi.stubGlobal('crypto', {
    getRandomValues: runtimeCrypto.getRandomValues.bind(runtimeCrypto),
    subtle: { digest: vi.fn(async () => new Uint8Array(32).buffer) }
  });

  it('fails closed when a callback has no PKCE transaction', async () => {
    await expect(completeAuthorization(config)).rejects.toThrow('Sesiunea de autentificare a expirat');
  });

  it('builds authorization code + PKCE S256 with state and nonce', async () => {
    useDeterministicDigest();
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(discovery), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));

    const request = new URL(await prepareAuthorization(config, '/registratura?status=INCOMING'));
    expect(request.origin + request.pathname).toBe(discovery.authorization_endpoint);
    expect(request.searchParams.get('response_type')).toBe('code');
    expect(request.searchParams.get('code_challenge_method')).toBe('S256');
    expect(request.searchParams.get('code_challenge')).toMatch(/^[A-Za-z0-9_-]{43}$/);
    expect(request.searchParams.get('state')).toBeTruthy();
    expect(request.searchParams.get('nonce')).toBeTruthy();
    expect(request.searchParams.get('scope')).toContain('offline_access');
    expect(request.searchParams.get('ui_theme_scheme')).toBe('system');
    expect(request.searchParams.get('ui_theme_dark')).toBe('0');
    expect(request.searchParams.get('ui_theme_primary')).toBe('rose');
    const state = request.searchParams.get('state');
    const transaction = JSON.parse(sessionStorage.getItem(`egueducation.oidc.authorization.${state}`) ?? '{}') as { verifier?: string; returnTo?: string };
    expect(transaction.verifier).toBeTruthy();
    expect(transaction.returnTo).toBe('/registratura?status=INCOMING');
  });

  it('rejects a callback with an unknown state without destroying the valid transaction', async () => {
    useDeterministicDigest();
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(discovery), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));
    const request = new URL(await prepareAuthorization(config));
    const state = request.searchParams.get('state');

    await expect(completeAuthorization(config, new URLSearchParams({ code: 'authorization-code', state: 'attacker-state' }))).rejects.toThrow();
    expect(sessionStorage.getItem(`egueducation.oidc.authorization.${state}`)).toBeTruthy();
  });

  it('keeps overlapping login attempts isolated by their returned state', async () => {
    useDeterministicDigest();
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(discovery), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));

    const first = new URL(await prepareAuthorization(config, '/registratura'));
    const second = new URL(await prepareAuthorization(config, '/earchiva'));
    const firstState = first.searchParams.get('state');
    const secondState = second.searchParams.get('state');

    expect(firstState).toBeTruthy();
    expect(secondState).toBeTruthy();
    expect(firstState).not.toBe(secondState);
    expect(JSON.parse(sessionStorage.getItem(`egueducation.oidc.authorization.${firstState}`) ?? '{}')).toMatchObject({ returnTo: '/registratura' });
    expect(JSON.parse(sessionStorage.getItem(`egueducation.oidc.authorization.${secondState}`) ?? '{}')).toMatchObject({ returnTo: '/earchiva' });
  });

  it('carries the selected PrimeReact color scheme into the provider interaction', async () => {
    useDeterministicDigest();
    window.localStorage.setItem('egueducation.scheme', 'dark');
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(discovery), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));

    const request = new URL(await prepareAuthorization(config));
    expect(request.searchParams.get('ui_theme_scheme')).toBe('dark');
    expect(request.searchParams.get('ui_theme_dark')).toBe('1');
  });

  it('builds and validates an RP-initiated logout transaction', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(discovery), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    }));
    const request = new URL(await prepareLogout(config, 'signed-id-token'));
    expect(request.origin + request.pathname).toBe(discovery.end_session_endpoint);
    expect(request.searchParams.get('id_token_hint')).toBe('signed-id-token');
    expect(request.searchParams.get('post_logout_redirect_uri')).toBe(config.logoutRedirectUri);
    expect(request.searchParams.get('client_id')).toBe(config.clientId);
    const state = request.searchParams.get('state');
    expect(state).toBeTruthy();
    expect(() => completeLogout(new URLSearchParams({ state: state! }))).not.toThrow();
    expect(() => completeLogout(new URLSearchParams({ state: 'replayed-state' }))).toThrow('nu corespunde');
  });
});
