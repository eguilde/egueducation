import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import Ajv2020 from 'ajv/dist/2020.js';
import standaloneCode from 'ajv/dist/standalone/index.js';
import addFormats from 'ajv-formats';

const openapiPath = fileURLToPath(new URL('../../openapi/openapi.json', import.meta.url));
const outputPath = fileURLToPath(new URL('../src/api/runtime-validators.ts', import.meta.url));
const document = JSON.parse(readFileSync(openapiPath, 'utf8'));
const componentSchemas = document.components?.schemas ?? {};
// Some older education operations reference a shared page schema directly
// instead of materializing an operation-specific response component.
componentSchemas.get_api_education_classes_response ??= componentSchemas.EducationPageOfSchoolClass;
componentSchemas.get_api_education_portfolios_me_recordid_procedure_response = componentSchemas.OwnPortfolioAppliedProcedureResponse;
componentSchemas.get_api_admissions_dss_retention_policies_current_response ??= componentSchemas.AdmissionDSSRetentionPolicy;
componentSchemas.post_api_admissions_dss_retention_policies_response ??= componentSchemas.AdmissionDSSRetentionPolicy;
componentSchemas.post_api_admissions_retention_rule_versions_response ??= componentSchemas.AdmissionRetentionRuleVersion;
componentSchemas.post_api_admissions_retention_rule_versions_approve_response ??= componentSchemas.AdmissionRetentionRuleVersion;
const selectedSchemas = [
  'get_api_education_portfolios_me_recordid_procedure_response',
  'post_api_education_portfolios_me_recordid_archive_documents_response',
  'get_api_regulatory_sources_response',
  'post_api_regulatory_sources_response',
  'post_api_regulatory_sources_sourceid_verify_response',
  'post_api_regulatory_sources_sourceid_activate_response',
  'get_api_earchiva_taxonomy_response',
  'get_api_earchiva_retention_rules_response',
  'post_api_earchiva_retention_rules_response',
  'post_api_earchiva_retention_rules_ruleid_approve_response',
  'post_api_earchiva_retention_rules_ruleid_retire_response',
  'get_api_earchiva_admin_portfolio_custody_intents_response',
  'post_api_earchiva_admin_portfolio_custody_intents_intentid_reconcile_response',
  'get_api_earchiva_admin_portfolio_custody_intents_intentid_recovery_operations_operationid_response',
  'SessionContext',
  'get_api_registratura_documents_response',
  'post_api_registratura_documents_response',
  'get_api_institution_regulatory_profile_response',
  'get_api_institution_capabilities_response',
  'get_api_institution_policy_cutover_preflight_response',
  'get_api_institution_locations_response',
  'post_api_institution_locations_response',
  'patch_api_institution_locations_locationid_response',
  'get_api_institution_education_offerings_response',
  'post_api_institution_education_offerings_response',
  'patch_api_institution_education_offerings_offeringid_response',
  'get_api_institution_offering_authorizations_response',
  'post_api_institution_offering_authorizations_response',
  'get_api_education_classes_response',
  'get_api_admissions_campaigns_response',
  'post_api_admissions_applications_applicationid_decision_preparations_response',
  'post_api_admissions_appeals_appealid_resolution_preparations_response',
  'post_api_admissions_legal_preparations_finalize_response',
  'post_api_admissions_legal_preparations_preparationid_cancel_response',
  'get_api_admissions_legal_preparations_preparationid_artifacts_artifactslot_response',
  'post_api_admissions_legal_preparations_preparationid_artifacts_artifactslot_response',
  'get_api_admissions_signer_authorizations_response',
  'post_api_admissions_signer_authorizations_response',
  'post_api_admissions_signer_authorizations_approve_response',
  'post_api_admissions_signer_authorizations_authorizationid_revoke_response',
  'get_api_admissions_class_offering_contexts_response',
  'post_api_admissions_class_offering_contexts_response',
  'get_api_admissions_regulatory_sources_response',
  'get_api_admissions_candidate_parties_response',
  'get_api_admissions_students_response',
  'get_api_admissions_eligible_archive_versions_response',
  'get_api_admissions_dss_retention_policies_current_response',
  'post_api_admissions_dss_retention_policies_response',
  'post_api_admissions_retention_rule_versions_response',
  'post_api_admissions_retention_rule_versions_approve_response',
  'post_api_admissions_campaigns_response',
  'post_api_admissions_campaigns_campaignid_transitions_response',
  'get_api_admissions_campaigns_campaignid_criteria_response',
  'post_api_admissions_campaigns_campaignid_criteria_response',
  'get_api_admissions_campaigns_campaignid_document_requirements_response',
  'post_api_admissions_campaigns_campaignid_document_requirements_response',
  'get_api_admissions_applications_response',
  'post_api_admissions_applications_response',
  'get_api_admissions_applications_applicationid_response',
  'post_api_admissions_applications_applicationid_transitions_response',
  'post_api_admissions_applications_applicationid_assessments_response',
  'post_api_admissions_applications_applicationid_documents_documentid_response',
  'post_api_admissions_applications_applicationid_decisions_response',
  'post_api_admissions_applications_applicationid_appeals_response',
  'post_api_admissions_applications_applicationid_enrolment_response',
  'get_api_admissions_decisions_response',
  'get_api_admissions_appeals_response',
  'post_api_admissions_appeals_appealid_resolution_response',
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
