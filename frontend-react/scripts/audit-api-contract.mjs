import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const load = (relativePath) => readFile(fileURLToPath(new URL(relativePath, import.meta.url)), 'utf8');
const [authProvider, registraturaApi, runtimeValidators, packageJson] = await Promise.all([
  load('../src/auth/AuthProvider.tsx'),
  load('../src/features/registratura/api.ts'),
  load('../src/api/runtime-validators.ts'),
  load('../package.json'),
]);

const violations = [];
const requirePattern = (source, pattern, message) => {
  if (!pattern.test(source)) violations.push(message);
};
const forbidPattern = (source, pattern, message) => {
  if (pattern.test(source)) violations.push(message);
};

requirePattern(authProvider, /createContractClient/, '/api/me must use the generated OpenAPI transport.');
requirePattern(authProvider, /validateSessionContext/, '/api/me must use its generated runtime validator.');
forbidPattern(authProvider, /fetch\(`\$\{config\.apiBaseUrl\}\/me`/, '/api/me reverted to handwritten fetch transport.');

requirePattern(registraturaApi, /contractClient\.GET\('\/api\/registratura\/documents'/, 'Registratura list must use the generated OpenAPI transport.');
requirePattern(registraturaApi, /contractClient\.POST\('\/api\/registratura\/documents'/, 'Registratura create must use the generated OpenAPI transport.');
requirePattern(registraturaApi, /validateGetApiRegistraturaDocumentsResponse/, 'Registratura list must validate its runtime response.');
requirePattern(registraturaApi, /validatePostApiRegistraturaDocumentsResponse/, 'Registratura create must validate its runtime response.');
forbidPattern(registraturaApi, /request(?:<[^>]+>)?\('\/registratura\/documents'/, 'Registratura list/create reverted to handwritten transport.');
forbidPattern(runtimeValidators, /require\(/, 'Generated runtime validators must be browser-safe ESM and cannot contain require(...).');

const manifest = JSON.parse(packageJson);
if (!manifest.dependencies?.['openapi-fetch']) violations.push('openapi-fetch runtime dependency is missing.');
if (!manifest.scripts?.['api:generate']?.includes('generate-runtime-validators.mjs')) {
  violations.push('api:generate must regenerate runtime validators.');
}

if (violations.length) {
  console.error(violations.join('\n'));
  process.exitCode = 1;
} else {
  console.log('API contract policy passed for /api/me and Registratura list/create.');
}
