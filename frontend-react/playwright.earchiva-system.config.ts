import { defineConfig } from '@playwright/test';
import path from 'node:path';

const origins = 'http://127.0.0.1:4176,http://127.0.0.1:4177';
const archiveEnvironment = (overrides: Record<string, string>) => ({
  ...process.env,
  FRONTEND_ORIGINS: origins,
  FORCE_SECURE_COOKIES: 'false',
  ENABLE_EUDI_WALLET: 'false',
  ARCHIVE_WORKER_ENABLED: 'true',
  ARCHIVE_WORKER_POLL_INTERVAL_SECONDS: '1',
  ARCHIVE_STORAGE_ENDPOINT: process.env.ARCHIVE_STORAGE_ENDPOINT ?? 'http://127.0.0.1:9000',
  ARCHIVE_STORAGE_REGION: 'us-east-1',
  ARCHIVE_STORAGE_BUCKET: 'earchiva-system-e2e',
  ARCHIVE_STORAGE_ACCESS_KEY: process.env.ARCHIVE_STORAGE_ACCESS_KEY ?? 'minioadmin',
  ARCHIVE_STORAGE_SECRET_KEY: process.env.ARCHIVE_STORAGE_SECRET_KEY ?? 'minioadmin123',
  ARCHIVE_STORAGE_USE_PATH_STYLE: 'true',
  ARCHIVE_STORAGE_CREATE_BUCKET: 'true',
  ARCHIVE_PDF_VALIDATOR_PATH: process.env.ARCHIVE_PDF_VALIDATOR_PATH ?? '',
  CLAMD_ADDRESS: process.env.CLAMD_ADDRESS ?? '127.0.0.1:3310',
  AZURE_DOCUMENT_INTELLIGENCE_ENDPOINT: process.env.AZURE_DOCUMENT_INTELLIGENCE_ENDPOINT ?? 'http://127.0.0.1:9090',
  AZURE_DOCUMENT_INTELLIGENCE_KEY: process.env.AZURE_DOCUMENT_INTELLIGENCE_KEY ?? 'system-e2e-azure-key',
  AZURE_DOCUMENT_INTELLIGENCE_MODEL: 'prebuilt-layout',
  AZURE_DOCUMENT_INTELLIGENCE_API_VERSION: '2024-11-30',
  ...overrides,
});

export default defineConfig({
  testDir: './e2e/system',
  testMatch: 'earchiva-real-pipeline.spec.ts',
  fullyParallel: false,
  workers: 1,
  timeout: 120_000,
  use: { baseURL: 'http://127.0.0.1:4176', trace: 'retain-on-failure' },
  webServer: [
    { command: 'node e2e/system/azure-ocr-emulator.mjs', cwd: import.meta.dirname, url: 'http://127.0.0.1:9090/healthz', reuseExistingServer: false, timeout: 30_000 },
    { command: 'go run ./cmd/server', cwd: path.resolve(import.meta.dirname, '../backend'), url: 'http://127.0.0.1:8083/health', reuseExistingServer: false, timeout: 120_000, env: archiveEnvironment({ PORT: '8083', FRONTEND_ORIGIN: 'http://127.0.0.1:4176', BACKEND_URL: 'http://127.0.0.1:8083', OIDC_ISSUER: 'http://127.0.0.1:4176/api/oidc', CUSTOMER_NAME: 'EguEducation Archive Fixture', TEST_OTP_FIXTURE_IDENTIFIER: 'archive.pipeline.fixture@example.test', TEST_OTP_FIXTURE_SUBJECT: 'archive-pipeline-fixture-subject', TEST_OTP_FIXTURE_TENANT_CODE: 'tenant-egueducation', TEST_OTP_FIXTURE_CODE: '864210' }) },
    { command: 'go run ./cmd/server', cwd: path.resolve(import.meta.dirname, '../backend'), url: 'http://127.0.0.1:8084/health', reuseExistingServer: false, timeout: 120_000, env: archiveEnvironment({ PORT: '8084', FRONTEND_ORIGIN: 'http://127.0.0.1:4177', BACKEND_URL: 'http://127.0.0.1:8084', OIDC_ISSUER: 'http://127.0.0.1:4177/api/oidc', CUSTOMER_NAME: 'Balotesti Archive Fixture', TEST_OTP_FIXTURE_IDENTIFIER: 'archive.pipeline.tenant-b@example.test', TEST_OTP_FIXTURE_SUBJECT: 'archive-pipeline-tenant-b-subject', TEST_OTP_FIXTURE_TENANT_CODE: 'tenant-balotesti', TEST_OTP_FIXTURE_CODE: '975310' }) },
    { command: 'npm run dev -- --host 127.0.0.1 --port 4176', cwd: import.meta.dirname, url: 'http://127.0.0.1:4176', reuseExistingServer: false, timeout: 120_000, env: { ...process.env, VITE_BACKEND_PROXY_TARGET: 'http://127.0.0.1:8083' } },
    { command: 'npm run dev -- --host 127.0.0.1 --port 4177', cwd: import.meta.dirname, url: 'http://127.0.0.1:4177', reuseExistingServer: false, timeout: 120_000, env: { ...process.env, VITE_BACKEND_PROXY_TARGET: 'http://127.0.0.1:8084' } },
  ],
});
