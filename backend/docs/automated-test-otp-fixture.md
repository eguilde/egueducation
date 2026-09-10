# Automated-test OTP fixture

The OIDC provider supports a deterministic OTP for isolated tests and for the
explicitly approved, tenant-bound production E2E user. It is disabled by
default.

Set all of the following in the test process only:

```text
APP_ENV=test
ENABLE_TEST_OTP_FIXTURE=true
TEST_OTP_FIXTURE_CODE=173829
TEST_OTP_FIXTURE_IDENTIFIER=oidc.browser.fixture@example.test
TEST_OTP_FIXTURE_SUBJECT=oidc-browser-fixture-subject
TEST_OTP_FIXTURE_TENANT_CODE=tenant-egueducation
FRONTEND_ORIGIN=http://127.0.0.1:<port>
BACKEND_URL=http://127.0.0.1:<port>
OIDC_ISSUER=http://127.0.0.1:<port>/api/oidc
```

The configured URLs must all be loopback `http` URLs. The fixture is accepted
only when the selected user has the exact configured identifier and subject and
belongs to the exact configured tenant. Startup rejects partial, malformed,
production, or public-host fixture configuration. The fixture user is created
only by disposable integration-test setup and is removed during cleanup; do not
add it to migrations or deployment data. The OTP is never rendered in HTML,
logs, API responses, or test output.

For the production E2E identity, set `APP_ENV=production` and
`ENABLE_PRODUCTION_FIXED_OTP_TEST_USER=true`, and provide the six-digit
`PRODUCTION_TEST_USER_OTP` only through the deployment secret. The
`PRODUCTION_TEST_USER_IDENTIFIER` must be `test@eguilde.cloud`;
`PRODUCTION_TEST_USER_SUBJECT` and `PRODUCTION_TEST_USER_TENANT` are
mandatory and the frontend, backend and issuer must be HTTPS URLs on the same
host. The dedicated UUID is permanently reserved, belongs to exactly one
tenant/institution, receives no platform role, and is provisioned with the
tenant-scoped `e2e_canary` role. There is no activation endpoint or activation
key: the browser follows the same provider-owned SMS OTP interaction as an
ordinary user and receives the same UI messages.
