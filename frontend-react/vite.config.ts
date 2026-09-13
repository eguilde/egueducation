import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: { port: 4200, proxy: { '/api': { target: process.env.VITE_BACKEND_PROXY_TARGET ?? 'http://localhost:8080', changeOrigin: true } } },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
    exclude: ['e2e/**', 'node_modules/**'],
    // PrimeReact's styled compound components are intentionally exercised in
    // JSDOM. Bounding parallelism avoids CPU contention turning quick UI
    // assertions into false five-second timeouts on CI runners.
    maxWorkers: 4,
    testTimeout: 15_000,
  }
});
