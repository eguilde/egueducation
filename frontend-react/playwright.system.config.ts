import { defineConfig } from '@playwright/test';
import path from 'node:path';

// WebAuthn requires a domain RP ID. Chromium deliberately rejects the numeric
// loopback address as an RP ID, while localhost remains a trustworthy origin.
const primaryFrontendPort = Number(process.env.E2E_PRIMARY_FRONTEND_PORT ?? '4173');
const teacherFrontendPort = Number(process.env.E2E_TEACHER_FRONTEND_PORT ?? '4174');
const alternateFrontendPort = Number(process.env.E2E_ALTERNATE_FRONTEND_PORT ?? '4175');
const frontendPorts = [primaryFrontendPort, teacherFrontendPort, alternateFrontendPort];
if (frontendPorts.some(port => !Number.isInteger(port) || port < 1024 || port > 65535) || new Set(frontendPorts).size !== 3) throw new Error('Invalid or duplicate E2E frontend ports');
const primaryFrontendOrigin = `http://localhost:${primaryFrontendPort}`;
const teacherFrontendOrigin = `http://localhost:${teacherFrontendPort}`;
const alternateFrontendOrigin = `http://localhost:${alternateFrontendPort}`;
const sharedFrontendOrigins = `${primaryFrontendOrigin},${teacherFrontendOrigin},${alternateFrontendOrigin}`;
const backendEnvironment = (overrides: Record<string, string>) => ({
  ...process.env,
  FRONTEND_ORIGINS: sharedFrontendOrigins,
  ENABLE_EUDI_WALLET: 'false',
  FORCE_SECURE_COOKIES: 'false',
  ...overrides,
});

// This config deliberately starts the production HTTP handlers and the React
// application together.  The tests under e2e/system must not add route mocks:
// their purpose is to prove the browser -> OIDC -> API -> PostgreSQL path.
export default defineConfig({
  testDir: './e2e/system',
  // Keep every School system proof in the same process topology so CI starts
  // PostgreSQL/OIDC/React only once and cannot silently omit a newly added
  // real-stack suite.
  testMatch: /(?:^|\/)(?:real-stack|teacher-portfolio-real-stack|admission-real-stack|institution-policy-real-stack|school-governance-real-stack|school-operations-real-stack|school-coverage-real-stack|school-mobile-real-stack)\.spec\.ts$/,
  fullyParallel: false,
  workers: 1,
  // This proof drives repeated real OIDC browser sessions, the statutory
  // five-section teacher portfolio workflow, PostgreSQL RLS, Registratura
  // uploads, a 20-row batch, workflow approval and MinIO outbox delivery.
  // Keep the run bounded while allowing the full browser/API/database contract
  // to complete on a cold GitHub-hosted runner.
  timeout: 600_000,
  use: {
    baseURL: primaryFrontendOrigin,
    actionTimeout: 20_000,
    navigationTimeout: 30_000,
    trace: 'retain-on-failure',
  },
  webServer: [
    {
      command: 'node e2e/system/azure-ocr-emulator.mjs',
      cwd: import.meta.dirname,
      url: 'http://127.0.0.1:9090/healthz',
      reuseExistingServer: false,
      timeout: 30_000,
    },
    { command: 'node e2e/system/signature-verifier-emulator.mjs', cwd: import.meta.dirname, url: 'http://127.0.0.1:9091/healthz', reuseExistingServer: false, timeout: 30_000 },
    {
      command: 'go run ./cmd/server',
      cwd: path.resolve(import.meta.dirname, '../backend'),
      url: 'http://127.0.0.1:8080/health',
      reuseExistingServer: false,
      timeout: 120_000,
      // The primary instance owns asynchronous outbox delivery. Secondary
      // OIDC issuers share the database but do not need duplicate workers.
      env: backendEnvironment({ FRONTEND_ORIGIN: primaryFrontendOrigin, OIDC_ISSUER: `${primaryFrontendOrigin}/api/oidc`, ARCHIVE_WORKER_ENABLED: 'true', SIGNATURE_VERIFIER_URL: 'http://127.0.0.1:9091', SIGNATURE_VERIFIER_TOKEN: 'system-test-dss-token' }),
    },
    {
      command: 'go run ./cmd/server',
      cwd: path.resolve(import.meta.dirname, '../backend'),
      url: 'http://127.0.0.1:8081/health',
      reuseExistingServer: false,
      timeout: 120_000,
      env: backendEnvironment({
        PORT: '8081',
        ARCHIVE_WORKER_ENABLED: 'false',
        FRONTEND_ORIGIN: teacherFrontendOrigin,
        BACKEND_URL: 'http://127.0.0.1:8081',
        OIDC_ISSUER: `${teacherFrontendOrigin}/api/oidc`,
        CUSTOMER_NAME: 'EguEducation Approver Fixture',
        TEST_OTP_FIXTURE_IDENTIFIER: 'oidc.approver.fixture@example.test',
        TEST_OTP_FIXTURE_SUBJECT: 'oidc-browser-approver-subject',
        TEST_OTP_FIXTURE_TENANT_CODE: 'tenant-egueducation',
        TEST_OTP_FIXTURE_CODE: '428615',
        SIGNATURE_VERIFIER_URL: 'http://127.0.0.1:9091', SIGNATURE_VERIFIER_TOKEN: 'system-test-dss-token',
      }),
    },
    {
      command: 'go run ./cmd/server',
      cwd: path.resolve(import.meta.dirname, '../backend'),
      url: 'http://127.0.0.1:8082/health',
      reuseExistingServer: false,
      timeout: 120_000,
      env: backendEnvironment({
        PORT: '8082',
        ARCHIVE_WORKER_ENABLED: 'false',
        FRONTEND_ORIGIN: alternateFrontendOrigin,
        BACKEND_URL: 'http://127.0.0.1:8082',
        OIDC_ISSUER: `${alternateFrontendOrigin}/api/oidc`,
        CUSTOMER_NAME: 'Scoala Balotesti',
        TEST_OTP_FIXTURE_IDENTIFIER: 'oidc.balotesti.fixture@example.test',
        TEST_OTP_FIXTURE_SUBJECT: 'oidc-browser-balotesti-subject',
        TEST_OTP_FIXTURE_TENANT_CODE: 'tenant-balotesti',
        TEST_OTP_FIXTURE_CODE: '739204',
        SIGNATURE_VERIFIER_URL: 'http://127.0.0.1:9091', SIGNATURE_VERIFIER_TOKEN: 'system-test-dss-token',
      }),
    },
    {
      command: `npm run dev -- --host 127.0.0.1 --port ${primaryFrontendPort} --strictPort`,
      cwd: import.meta.dirname,
      url: `http://127.0.0.1:${primaryFrontendPort}`,
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: `npm run dev -- --host 127.0.0.1 --port ${teacherFrontendPort} --strictPort`,
      cwd: import.meta.dirname,
      url: `http://127.0.0.1:${teacherFrontendPort}`,
      reuseExistingServer: false,
      timeout: 120_000,
      env: { ...process.env, VITE_BACKEND_PROXY_TARGET: 'http://127.0.0.1:8081' },
    },
    {
      command: `npm run dev -- --host 127.0.0.1 --port ${alternateFrontendPort} --strictPort`,
      cwd: import.meta.dirname,
      url: `http://127.0.0.1:${alternateFrontendPort}`,
      reuseExistingServer: false,
      timeout: 120_000,
      env: { ...process.env, VITE_BACKEND_PROXY_TARGET: 'http://127.0.0.1:8082' },
    },
  ],
});
