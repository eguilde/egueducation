import { defineConfig } from '@playwright/test';

function productionOrigin(): string {
  const value = process.env.PRODUCTION_ORIGIN?.trim();
  if (!value) throw new Error('PRODUCTION_ORIGIN is required for the production OIDC canary.');

  const origin = new URL(value);
  if (origin.protocol !== 'https:' || origin.pathname !== '/' || origin.search || origin.hash) {
    throw new Error('PRODUCTION_ORIGIN must be an HTTPS origin without a path, query, or fragment.');
  }
  return origin.origin;
}

export default defineConfig({
  testDir: './e2e',
  testMatch: 'production-oidc-canary.spec.ts',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 90_000,
  reporter: [['line']],
  outputDir: 'test-results/production-oidc-canary',
  use: {
    baseURL: productionOrigin(),
    // The interaction includes production-only credentials. Do not retain any
    // browser artifact that could contain entered values or authorization data.
    trace: 'off',
    screenshot: 'off',
    video: 'off',
  },
});
