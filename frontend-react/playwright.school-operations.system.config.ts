import { defineConfig } from '@playwright/test';
import systemConfig from './playwright.system.config';

// Kept separate while the broad system proof remains intentionally stable.
// CI can invoke this configuration independently, and the main system config
// can later include both specs when its time budget is adjusted.
export default defineConfig({
  ...systemConfig,
  testMatch: 'school-operations-real-stack.spec.ts',
});
