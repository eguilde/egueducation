export interface OidcConfig { authority: string; clientId: string; redirectUri: string; logoutRedirectUri: string; scope: string; apiBaseUrl: string }
export const oidcConfig = (): OidcConfig => ({
  authority: import.meta.env.VITE_OIDC_AUTHORITY ?? '/api/oidc',
  clientId: import.meta.env.VITE_OIDC_CLIENT_ID ?? 'egueducation-spa',
  redirectUri: `${window.location.origin}/auth/callback`,
  logoutRedirectUri: `${window.location.origin}/auth/logout`,
  scope: import.meta.env.VITE_OIDC_SCOPE ?? 'openid profile email phone offline_access',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '/api'
});
