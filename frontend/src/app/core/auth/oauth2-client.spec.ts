import * as oidcClient from './oauth2-client';

describe('OIDC authorization requests', () => {
  const config = {
    clientId: 'egueducation-spa',
    redirectUri: 'https://app.example.test/auth/callback',
    scope: 'openid profile',
    secureRoutes: ['/api'],
  };

  it('binds every authorization request to a nonce as well as PKCE and state', () => {
    const params = oidcClient.authorizationParameters(config, 'S256-challenge', 'state-123', 'nonce-456');

    expect(params.get('nonce')).toBe('nonce-456');
    expect(params.get('state')).toBe('state-123');
    expect(params.get('code_challenge_method')).toBe('S256');
    expect(params.get('code_challenge')).toBe('S256-challenge');
  });

  it('does not expose a browser JWT decoding helper for application identity', () => {
    expect('parseIdTokenClaims' in oidcClient).toBe(false);
  });
});
