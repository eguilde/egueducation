# OIDC Provider Data Model

This backend now uses the transplanted Go OIDC provider runtime instead of the previous hand-rolled authorization code and cookie session implementation.

## Authoritative runtime tables

`oidc_clients`
- Registered OIDC clients and their JSON metadata payload.
- Seeded from config for the SPA and desktop clients.

`oidc_authn_sessions`
- Short-lived provider interaction state for `/authorize`.
- Stores serialized `goidc.AuthnSession` records.

`oidc_grant_sessions`
- Issued grant state for authorization codes, access tokens, and refresh tokens.
- Stores serialized `goidc.GrantSession` records.

`oidc_jwks_keys`
- Active and rotated signing keys for JWT access tokens and ID tokens.
- Supports overlap windows during key rotation.

`oidc_otp_codes`
- OTP verification state for the provider login interaction.
- Replaces the legacy `sms_otp_codes` table for OIDC login.

`oidc_passkey_login_nonces`
- One-time bridge from the existing WebAuthn assertion endpoint into the OIDC interaction flow.

## Reused application tables

`app_users`
- Source of subject identity, email, phone, locale, and active status.

`app_session_context`
- Source of institution context and enabled auth methods for `/api/me` and grant enrichment.

`app_user_roles`
`app_role_permissions`
`app_memberships`
`app_position_roles`
`app_position_permissions`
- Source of role and permission expansion used by protected API requests.

`app_passkeys`
`app_passkey_challenges`
`app_passkey_login_challenges`
- Existing passkey registration and assertion storage, reused by the OIDC login pages.

`app_eudi_wallets`
- Still only tracks wallet activation status for a signed-in user profile.
- It is not yet enough on its own to provide the full wallet identity resolution flow used in `eguilde_platform`.

## Legacy tables now obsolete for authentication

These are historical tables from the removed implementation and should not be treated as the active source of truth for login anymore:

`oidc_authorization_codes`
`oidc_refresh_tokens`
`oidc_consents`
`oidc_consent_requests`
`oidc_signing_keys`
`app_login_sessions`
`sms_otp_codes`

They remain in historical migrations, but the new provider runtime does not write to them.

The legacy admin OIDC consent/session endpoints that depended on those tables have also been removed from `egueducation`, so there is no longer an active admin surface reading stale consent or refresh-token records.

## Reference parity notes

Compared with `E:\dev\eguilde_platform\backend`, the data model here now matches the core OIDC provider storage shape for:

- client registration
- authn session storage
- grant session storage
- JWT signing key rotation
- provider-managed OTP verification
- passkey handoff into OIDC

The remaining parity gap is wallet identity resolution. `eguilde_platform` also carries verifier-side transaction storage and PID identity linking logic that is not present in `egueducation` today, so wallet login cannot yet be considered a full transplant just from the provider tables above.
