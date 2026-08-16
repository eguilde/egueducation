# Automated-test OTP fixture

The OIDC provider supports a deterministic OTP only for an isolated browser or
integration test server. It is disabled by default and is never a deployment
login mechanism.

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
