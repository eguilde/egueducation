import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import { validateGetApiInstitutionCapabilitiesResponse, validateGetApiInstitutionRegulatoryProfileResponse } from "../../api/runtime-validators";

export type RegulatoryProfile = components["schemas"]["get_api_institution_regulatory_profile_response"];
export type InstitutionCapabilities = components["schemas"]["get_api_institution_capabilities_response"];
export type PutRegulatoryProfileInput = components["schemas"]["PutRegulatoryProfileRequest"];

export interface InstitutionPolicyApi {
  profile(): Promise<RegulatoryProfile>;
  capabilities(): Promise<InstitutionCapabilities>;
  saveProfile(input: PutRegulatoryProfileInput): Promise<RegulatoryProfile>;
}

async function unwrap<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>, validate: (value: unknown) => boolean): Promise<T> {
  const result = await request;
  if (!result.response.ok || result.error || result.data === undefined) {
    const error = new Error(`institution_policy_${result.response.status}`) as Error & { status?: number };
    error.status = result.response.status;
    throw error;
  }
  if (!validate(result.data)) throw new Error("institution_policy_contract_mismatch");
  return result.data;
}

export function createInstitutionPolicyApi(client: ContractClient): InstitutionPolicyApi {
  return {
    profile: () => unwrap(client.GET("/api/institution/regulatory-profile"), validateGetApiInstitutionRegulatoryProfileResponse),
    capabilities: () => unwrap(client.GET("/api/institution/capabilities"), validateGetApiInstitutionCapabilitiesResponse),
    saveProfile: (body) => unwrap(client.PUT("/api/institution/regulatory-profile", { body }), validateGetApiInstitutionRegulatoryProfileResponse),
  };
}
