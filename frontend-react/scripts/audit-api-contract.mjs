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
const [authProvider, runtimeValidators, packageJson, generated, schoolWizardApi, schoolWizards, ...adapters] = await Promise.all([
  load('../src/auth/AuthProvider.tsx'),
  load('../src/api/runtime-validators.ts'),
  load('../package.json'),
  load('../src/api/generated.ts'),
  load('../src/features/education/school-wizard-api.ts'),
  load('../src/features/education/wizards/index.tsx'),
  ...adapterPaths.map(load),
]);

const violations = [];
const retentionAdapter = await load('../src/features/earchiva/archive-retention-api.ts');
const sourcesAdapter = await load('../src/features/regulatory-sources/api.ts');
const administrationWorkspace = await load('../src/features/admin/AdministrationWorkspace.tsx');
const requirePattern = (source, pattern, message) => {
  if (!pattern.test(source)) violations.push(message);
};
const forbidPattern = (source, pattern, message) => {
  if (pattern.test(source)) violations.push(message);
};

requirePattern(retentionAdapter, /ContractClient/, 'Archive retention must use the generated client contract.');
requirePattern(sourcesAdapter, /ContractClient/, 'Regulatory sources must use the generated API client.');
forbidPattern(administrationWorkspace, /school\.regulatory_sources\.(verify|activate)/, 'Source command UI must use registered manage/approve permissions, not invented action names.');
requirePattern(sourcesAdapter, /components\["schemas"\]\["get_api_regulatory_sources_item"\]/, 'Regulatory source DTO must derive from OpenAPI.');
forbidPattern(sourcesAdapter, /\bfetch\(/, 'Regulatory sources must not bypass the generated client.');
forbidPattern(sourcesAdapter, /import\("\.\.\/\.\.\/api\/runtime-validators"\)/, 'Regulatory validators must use static, typechecked imports.');
for (const validator of ['validateGetApiRegulatorySourcesResponse', 'validatePostApiRegulatorySourcesResponse', 'validatePostApiRegulatorySourcesSourceidVerifyResponse', 'validatePostApiRegulatorySourcesSourceidActivateResponse']) {
  requirePattern(sourcesAdapter, new RegExp(validator), `Regulatory sources must use ${validator}.`);
  requirePattern(runtimeValidators, new RegExp(`export const ${validator}\\b`), `Missing source validator ${validator}.`);
}
forbidPattern(retentionAdapter, /\bfetch\(/, 'Archive retention must not call browser fetch directly.');
for (const validator of ['validateGetApiEarchivaRetentionRulesResponse', 'validatePostApiEarchivaRetentionRulesResponse', 'validatePostApiEarchivaRetentionRulesRuleidApproveResponse', 'validatePostApiEarchivaRetentionRulesRuleidRetireResponse']) {
  requirePattern(retentionAdapter, new RegExp(validator), `Archive retention must validate its response using ${validator}.`);
  requirePattern(runtimeValidators, new RegExp(`export const ${validator}\\b`), `Generated validator missing: ${validator}.`);
}

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

const portfolioCreate = generated.match(/CreatePortfolioRecordRequest:\s*\{([\s\S]*?)\n\s*\};/u)?.[1] ?? '';
requirePattern(portfolioCreate, /owner_personnel_id:\s*string;/u, 'Portfolio create must require the tenant-scoped owner personnel pairing.');
forbidPattern(portfolioCreate, /retention_until/u, 'Portfolio create must not expose server-controlled retention_until.');
requirePattern(generated, /UpdatePortfolioRecordRequest:\s*\{/u, 'Portfolio PATCH must use a separate generated update DTO.');
forbidPattern(schoolWizardApi, /retention_until/u, 'School portfolio wizard adapter must not submit server-controlled retention_until.');
forbidPattern(schoolWizards, /retention_until/u, 'School portfolio wizard must not request server-controlled retention_until.');

const educationAdapter = adapters[1];
forbidPattern(educationAdapter, /\bAuxiliaryClient\b|\bauxiliary(?:Get|Post|Patch|Remove)\b|\bauxiliaryClient\b/u, 'Education adapter must not retain an auxiliary transport bridge.');
forbidPattern(educationAdapter, /\bconst\s+(?:get|post|patch|remove)\s*=\s*auxiliary/u, 'Education adapter must not retain auxiliary method aliases.');
// Every mutable related-resource lot must use literal generated operations.
// The remaining AuxiliaryClient compatibility boundary is prohibited from all
// School related routes, including the mobility and merit dossier flows.
for (const route of [
  '/api/education/governance/memberships',
  '/api/education/governance/meetings/{meetingID}/documents/{documentID}/pdf',
  '/api/education/decisions/records/{decisionID}/issuances',
  '/api/education/regulations/records/{recordID}/versions',
  '/api/education/committees/records/{recordID}/members',
  '/api/education/managerial/records/{recordID}/documents/{documentID}/pdf',
  '/api/education/personnel/records/{recordID}/assignments',
  '/api/education/personnel/records/{recordID}/file-documents/{documentID}',
  '/api/education/personnel/records/{recordID}/disciplinary-cases/{itemID}',
  '/api/education/personnel/records/{recordID}/access-events/{eventID}',
  '/api/education/evaluations/records/{recordID}/self-reviews/{itemID}',
  '/api/education/evaluations/records/{recordID}/criteria/{itemID}',
  '/api/education/evaluations/records/{recordID}/appeals/{appealID}/pdf',
  '/api/education/evaluations/records/{recordID}/result-issues/{itemID}/pdf',
  '/api/education/mobility/records/{recordID}/documents/{itemID}',
  '/api/education/mobility/records/{recordID}/scores/{itemID}',
  '/api/education/mobility/records/{recordID}/appeals/{itemID}/pdf',
  '/api/education/mobility/records/{recordID}/final-decisions/{itemID}/pdf',
  '/api/education/mobility/records/{recordID}/result-issues/{itemID}/pdf',
  '/api/education/gradatii/records/{recordID}/documents/{itemID}',
  '/api/education/gradatii/records/{recordID}/scores/{itemID}',
  '/api/education/gradatii/records/{recordID}/appeals/{itemID}/pdf',
  '/api/education/gradatii/records/{recordID}/final-decisions/{itemID}/pdf',
  '/api/education/gradatii/records/{recordID}/result-issues/{itemID}/pdf',
]) {
  requirePattern(educationAdapter, new RegExp(`client\\.(?:GET|POST|PATCH|DELETE)\\("${route.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`), `Education related contract lot must use literal generated transport for ${route}.`);
}
forbidPattern(educationAdapter, /auxiliary(?:Get|Post|Patch|Remove)\(client,\s*"\/api\/education\/(?:governance\/(?:memberships|bodies|meetings\/.+\/(?:participants|documents|votes|minutes|resolutions))|decisions\/records\/.+\/(?:issuances|publication-steps)|regulations\/records\/.+\/(?:versions|workflow)|committees\/records\/.+\/members|managerial\/records\/.+\/(?:documents|workflow))/u, 'Governance/managerial related routes must not use AuxiliaryClient.');
forbidPattern(educationAdapter, /auxiliary(?:Get|Post|Patch|Remove)\(client,\s*"\/api\/education\/(?:personnel\/records\/.+\/(?:assignments|file-documents|disciplinary-cases|access-events)|evaluations\/records\/.+\/(?:self-reviews|criteria|appeals|result-issues))/u, 'Personnel/evaluation related routes must not use AuxiliaryClient.');
forbidPattern(educationAdapter, /auxiliary(?:Get|Post|Patch|Remove)\(client,\s*"\/api\/education\/(?:mobility|gradatii)\/records\/.+\/(?:documents|scores|appeals|final-decisions|result-issues)/u, 'Mobility/merit related routes must not use AuxiliaryClient.');
// Dashboard, cockpit, meeting CRUD and metadata are all closed generated
// contracts. They may not drift back through the residual AuxiliaryClient.
for (const route of [
  '/api/education/dashboard', '/api/education/director/cockpit', '/api/education/governance/eligible-users', '/api/education/governance/meetings',
  '/api/education/governance/meetings/filters', '/api/education/governance/meetings/{meetingID}/finalization-summary', '/api/education/governance/bodies/{bodyID}/completeness-summary',
  '/api/education/decisions/dashboard', '/api/education/decisions/records/filters', '/api/education/managerial/dashboard', '/api/education/managerial/records/filters',
  '/api/education/regulations/dashboard', '/api/education/regulations/records/filters', '/api/education/personnel/dashboard', '/api/education/personnel/records/filters',
  '/api/education/evaluations/dashboard', '/api/education/evaluations/records/filters', '/api/education/declarations/dashboard', '/api/education/declarations/records/filters',
  '/api/education/mobility/dashboard', '/api/education/mobility/records/filters', '/api/education/gradatii/dashboard', '/api/education/gradatii/records/filters',
  '/api/education/portfolios/dashboard', '/api/education/portfolios/records/filters', '/api/education/committees/records/{recordID}/completeness-summary',
  '/api/education/managerial/records/{recordID}/portfolio-summary', '/api/education/personnel/records/{recordID}/portfolio-dossier-summary',
  '/api/education/portfolios/records/{recordID}/transfer-summary', '/api/education/regulations/records/{recordID}/procedural-summary',
]) {
  forbidPattern(educationAdapter, new RegExp(`auxiliary(?:Get|Post|Patch|Remove)\\(client,\\s*"${route.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`, 'u'), `Education metadata/governance route must not use AuxiliaryClient: ${route}.`);
}
for (const resource of [
  'portfolio-documents', 'portfolio-checklist', 'portfolio-opis',
  'portfolio-custody', 'portfolio-reviews', 'portfolio-transfers',
  'portfolio-valorifications', 'portfolio-sections', 'taxonomies', 'requirements',
]) {
  forbidPattern(educationAdapter, new RegExp(`related(?:Records|Detail|Pdf|\\(|:)\\S*[\\s\\S]{0,200}${resource}`, 'u'), `Portfolio/catalog resource ${resource} must not use the generic related-resource adapter.`);
}
requirePattern(educationAdapter, /portfolioDocuments\(recordID/u, 'Portfolio documents must use their named generated transport.');
requirePattern(educationAdapter, /portfolioTransferHistory\(recordID/u, 'Portfolio transfer history must use its named read-only generated transport.');
requirePattern(educationAdapter, /taxonomyCatalog\(input/u, 'Taxonomy groups must use the named generated transport.');

const manifest = JSON.parse(packageJson);
if (!manifest.dependencies?.['openapi-fetch']) violations.push('openapi-fetch runtime dependency is missing.');
if (!manifest.scripts?.['api:generate']?.includes('generate-runtime-validators.mjs')) {
  violations.push('api:generate must regenerate runtime validators.');
}

if (violations.length) {
  console.error(violations.join('\n'));
  process.exitCode = 1;
} else {
  console.log(`API contract policy passed for ${adapterPaths.length + 2} production adapters, /api/me and generated runtime validators.`);
}
