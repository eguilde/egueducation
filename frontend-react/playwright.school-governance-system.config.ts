import { defineConfig } from '@playwright/test';
import base from './playwright.system.config';

// Keep the governance proof opt-in.  It uses the same real OIDC, Go and
// PostgreSQL topology as the broad system suite, but has an independent CI
// entry point and never falls back to mocked routes.
export default defineConfig({
  ...base,
  testMatch: 'school-governance-real-stack.spec.ts',
});
