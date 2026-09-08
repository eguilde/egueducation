import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const load = (relativePath) => readFile(fileURLToPath(new URL(relativePath, import.meta.url)), 'utf8');
const adapterPaths = [
  '../src/features/admin/api.ts',
  '../src/features/education/api.ts',
  '../src/features/earchiva/api.ts',
  '../src/features/profile/api.ts',
  '../src/features/registratura/api.ts',
  '../src/features/workflow/api.ts',
];
const [authProvider, runtimeValidators, packageJson, generated, ...adapters] = await Promise.all([
  load('../src/auth/AuthProvider.tsx'),
  load('../src/api/runtime-validators.ts'),
  load('../package.json'),
  load('../src/api/generated.ts'),
  ...adapterPaths.map(load),
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

for (const [index, source] of adapters.entries()) {
  const name = adapterPaths[index].split('/').at(-2);
  requirePattern(source, /create(?:OpenApiTransport|ContractClient)/, `${name} adapter must use the generated OpenAPI client.`);
  forbidPattern(source, /fetcher\(`\$\{apiBase\}/, `${name} adapter reverted to handwritten fetch transport.`);
  forbidPattern(source, /\bfetch\(/, `${name} adapter must not call browser fetch directly.`);
}

for (const route of [
  '/api/admin/users',
  '/api/education/personnel/records',
  '/api/earchiva/documents',
  '/api/profile',
  '/api/passkeys/register-options',
  '/api/registratura/documents',
  '/api/registratura/flux/queue',
]) {
  requirePattern(generated, new RegExp(`"${route.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}"`), `Generated OpenAPI client is missing ${route}.`);
}

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
  console.log(`API contract policy passed for ${adapterPaths.length} production adapters, /api/me and generated runtime validators.`);
}
