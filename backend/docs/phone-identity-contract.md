# Phone identity and verification contract

Phone numbers are global login identities. A normalized E.164 number can
belong to exactly one `app_users` account, even when that account is a member
of several tenants. Tenant memberships, roles, permissions and modules remain
tenant-scoped; ownership of the login credential does not grant access to any
tenant.

## Assignment and verification

1. Administrative user creation or profile editing may assign a valid E.164
   phone, but cannot mark it verified.
2. The OIDC login resolver may locate the assigned, unverified phone so the
   first SMS challenge can be delivered.
3. A successful one-time SMS code is consumed in the same database transaction
   that verifies the primary `phone` identity and its `app_users` projection.
4. The transaction requires an active membership in the tenant selected from
   the request host. Verification does not add memberships or RBAC grants.
5. The successful enrollment/login outcome is appended to `app_audit_log`.
   Submitted codes, OTP hashes, phone values and provider payloads are never
   included in that audit record.

The database constraint installed by migration 0086 rejects a verified profile
phone unless an equally normalized, primary and verified identity exists.
Migration 0083 installs the global `(identity_type, normalized_value)` unique
constraint which prevents a number from being owned by two accounts.

## Changes and revocation

Changing or removing a phone clears `phone_number_verified`, replaces/removes
the primary identity, invalidates every pending login OTP for that user and
records a revocation/assignment event. Consequently, a code sent to the old
number cannot verify the replacement number. The replacement must complete a
new SMS challenge.

An administrator may explicitly revoke an unchanged verification, but an
administrative request containing `phone_verified: true` never promotes trust.

## OTP storage

Only a versioned HMAC-SHA-256 verifier is stored. `OTP_HMAC_KEY` is mandatory
outside isolated `APP_ENV=test`, must contain at least 32 bytes and is supplied
as a deployment secret. Plain SHA-256 and malformed/legacy verifier rows are
rejected and removed; the user must request a new ten-minute code. Comparisons
use constant-time MAC comparison. Operational SMS queue rows redact six-digit
codes.

## Automated fixture

The loopback-only fixture starts with an unverified synthetic phone and uses
the same OIDC, OTP storage, consumption, verification and audit path as an
ordinary account. CI asserts the database state before and after the browser
login. Production canary configuration is separately gated and must use the
dedicated `test@eguilde.cloud` identity; it must never reuse the loopback
identifier or code.
