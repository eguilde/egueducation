import { defineConfig } from '@playwright/test';
import path from 'node:path';

const sharedFrontendOrigins = 'http://127.0.0.1:4173,http://127.0.0.1:4174,http://127.0.0.1:4175';
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
  testMatch: 'real-stack.spec.ts',
  fullyParallel: false,
  workers: 1,
  // This proof drives three real OIDC browser sessions, PostgreSQL RLS,
  // Registratura uploads, a 20-row batch, workflow approval and MinIO outbox
  // delivery. Keep a bounded timeout that reflects the complete contract.
  timeout: 300_000,
  use: {
    baseURL: 'http://127.0.0.1:4173',
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
    {
      command: 'go run ./cmd/server',
      cwd: path.resolve(import.meta.dirname, '../backend'),
      url: 'http://127.0.0.1:8080/health',
      reuseExistingServer: false,
      timeout: 120_000,
      // The primary instance owns asynchronous outbox delivery. Secondary
      // OIDC issuers share the database but do not need duplicate workers.
      env: backendEnvironment({ ARCHIVE_WORKER_ENABLED: 'true' }),
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
        FRONTEND_ORIGIN: 'http://127.0.0.1:4174',
        BACKEND_URL: 'http://127.0.0.1:8081',
        OIDC_ISSUER: 'http://127.0.0.1:4174/api/oidc',
        CUSTOMER_NAME: 'EguEducation Approver Fixture',
        TEST_OTP_FIXTURE_IDENTIFIER: 'oidc.approver.fixture@example.test',
        TEST_OTP_FIXTURE_SUBJECT: 'oidc-browser-approver-subject',
        TEST_OTP_FIXTURE_TENANT_CODE: 'tenant-egueducation',
        TEST_OTP_FIXTURE_CODE: '428615',
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
        FRONTEND_ORIGIN: 'http://127.0.0.1:4175',
        BACKEND_URL: 'http://127.0.0.1:8082',
        OIDC_ISSUER: 'http://127.0.0.1:4175/api/oidc',
        CUSTOMER_NAME: 'Scoala Balotesti',
        TEST_OTP_FIXTURE_IDENTIFIER: 'oidc.balotesti.fixture@example.test',
        TEST_OTP_FIXTURE_SUBJECT: 'oidc-browser-balotesti-subject',
        TEST_OTP_FIXTURE_TENANT_CODE: 'tenant-balotesti',
        TEST_OTP_FIXTURE_CODE: '739204',
      }),
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 4173',
      cwd: import.meta.dirname,
      url: 'http://127.0.0.1:4173',
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 4174',
      cwd: import.meta.dirname,
      url: 'http://127.0.0.1:4174',
      reuseExistingServer: false,
      timeout: 120_000,
      env: { ...process.env, VITE_BACKEND_PROXY_TARGET: 'http://127.0.0.1:8081' },
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 4175',
      cwd: import.meta.dirname,
      url: 'http://127.0.0.1:4175',
      reuseExistingServer: false,
      timeout: 120_000,
      env: { ...process.env, VITE_BACKEND_PROXY_TARGET: 'http://127.0.0.1:8082' },
    },
  ],
});
