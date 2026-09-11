import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import Ajv2020 from 'ajv/dist/2020.js';
import standaloneCode from 'ajv/dist/standalone/index.js';
import addFormats from 'ajv-formats';

const openapiPath = fileURLToPath(new URL('../../openapi/openapi.json', import.meta.url));
const outputPath = fileURLToPath(new URL('../src/api/runtime-validators.ts', import.meta.url));
const document = JSON.parse(readFileSync(openapiPath, 'utf8'));
const componentSchemas = document.components?.schemas ?? {};
const selectedSchemas = [
  'SessionContext',
  'get_api_registratura_documents_response',
  'post_api_registratura_documents_response',
  'get_api_institution_regulatory_profile_response',
  'get_api_institution_capabilities_response',
];

const referencedSchemaNames = new Set(selectedSchemas);
const collectReferences = (value) => {
  if (Array.isArray(value)) {
    value.forEach(collectReferences);
    return;
  }
  if (!value || typeof value !== 'object') return;
  for (const [key, child] of Object.entries(value)) {
    if (key === '$ref' && typeof child === 'string' && child.startsWith('#/components/schemas/')) {
      const name = child.slice('#/components/schemas/'.length);
      if (!referencedSchemaNames.has(name)) {
        referencedSchemaNames.add(name);
        collectReferences(componentSchemas[name]);
      }
    } else {
      collectReferences(child);
    }
  }
};
selectedSchemas.forEach((name) => collectReferences(componentSchemas[name]));
const runtimeDefinitions = Object.fromEntries(
  [...referencedSchemaNames].map((name) => [name, componentSchemas[name]]),
);

const rewriteReferences = (value) => {
  if (Array.isArray(value)) return value.map(rewriteReferences);
  if (!value || typeof value !== 'object') {
    return typeof value === 'string'
      ? value.replaceAll('#/components/schemas/', '#/$defs/')
      : value;
  }
  return Object.fromEntries(Object.entries(value).map(([key, child]) => [key, rewriteReferences(child)]));
};

const ajv = new Ajv2020({
  allErrors: true,
  strict: false,
  code: { esm: true, source: true },
});
addFormats(ajv);

const exports = {};
for (const schemaName of selectedSchemas) {
  const schema = componentSchemas[schemaName];
  if (!schema) throw new Error(`OpenAPI component schema ${schemaName} is missing`);
  const schemaId = `https://contracts.eguilde.cloud/openapi/${schemaName}`;
  ajv.addSchema({
    ...rewriteReferences(schema),
    $id: schemaId,
    $defs: rewriteReferences(runtimeDefinitions),
  });
  const exportSuffix = schemaName.split('_').map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join('');
  exports[`validate${exportSuffix}`] = schemaId;
}

const banner = `/* This file is generated from ../openapi/openapi.json. Do not edit manually. */\n// @ts-nocheck\nimport ucs2lengthModule from 'ajv/dist/runtime/ucs2length.js';\nimport { fullFormats } from 'ajv-formats/dist/formats.js';\nconst runtimeUcs2Length = typeof ucs2lengthModule === 'function' ? ucs2lengthModule : ucs2lengthModule.default;\n`;
const browserSafeCode = standaloneCode(ajv, exports)
  .replace(/require\("ajv\/dist\/runtime\/ucs2length"\)\.default/g, 'runtimeUcs2Length')
  .replace(/require\("ajv-formats\/dist\/formats"\)\.fullFormats/g, 'fullFormats');
if (browserSafeCode.includes('require(')) {
  throw new Error('Generated runtime validators contain a CommonJS require that is unsafe in the browser bundle');
}
writeFileSync(outputPath, `${banner}${browserSafeCode}\n`, 'utf8');
