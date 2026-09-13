import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import {
  validateGetApiInstitutionCapabilitiesResponse,
  validateGetApiInstitutionEducationOfferingsResponse,
  validateGetApiInstitutionLocationsResponse,
  validateGetApiInstitutionOfferingAuthorizationsResponse,
  validateGetApiInstitutionPolicyCutoverPreflightResponse,
  validateGetApiInstitutionRegulatoryProfileResponse,
  validatePatchApiInstitutionEducationOfferingsOfferingidResponse,
  validatePatchApiInstitutionLocationsLocationidResponse,
  validatePostApiInstitutionEducationOfferingsResponse,
  validatePostApiInstitutionLocationsResponse,
  validatePostApiInstitutionOfferingAuthorizationsResponse,
} from "../../api/runtime-validators";

export type RegulatoryProfile = components["schemas"]["get_api_institution_regulatory_profile_response"];
export type InstitutionCapabilities = components["schemas"]["get_api_institution_capabilities_response"];
export type PutRegulatoryProfileInput = components["schemas"]["PutRegulatoryProfileRequest"];
export type PolicyCutoverPreflight = components["schemas"]["get_api_institution_policy_cutover_preflight_response"];
export type SchoolLocation = components["schemas"]["get_api_institution_locations_item"];
export type EducationOffering = components["schemas"]["get_api_institution_education_offerings_item"];
export type OfferingAuthorization = components["schemas"]["get_api_institution_offering_authorizations_item"];
export type SchoolLocationPage = components["schemas"]["get_api_institution_locations_response"];
export type EducationOfferingPage = components["schemas"]["get_api_institution_education_offerings_response"];
export type OfferingAuthorizationPage = components["schemas"]["get_api_institution_offering_authorizations_response"];
export type CreateSchoolLocationInput = components["schemas"]["CreateSchoolLocationRequest"];
export type UpdateSchoolLocationInput = components["schemas"]["UpdateSchoolLocationRequest"];
export type CreateEducationOfferingInput = components["schemas"]["CreateEducationOfferingRequest"];
export type UpdateEducationOfferingInput = components["schemas"]["UpdateEducationOfferingRequest"];
export type CreateOfferingAuthorizationInput = components["schemas"]["CreateOfferingAuthorizationRequest"];
export type InstitutionCatalogQuery = { page: number; pageSize: number; sort?: string; direction?: "asc" | "desc"; filters?: Record<string, string> };

export interface InstitutionPolicyApi {
  profile(): Promise<RegulatoryProfile>;
  capabilities(): Promise<InstitutionCapabilities>;
  cutoverPreflight?(): Promise<PolicyCutoverPreflight>;
  locations?(query: InstitutionCatalogQuery): Promise<SchoolLocationPage>;
  createLocation?(input: CreateSchoolLocationInput): Promise<SchoolLocation>;
  updateLocation?(id: string, input: UpdateSchoolLocationInput): Promise<SchoolLocation>;
  offerings?(query: InstitutionCatalogQuery): Promise<EducationOfferingPage>;
  createOffering?(input: CreateEducationOfferingInput): Promise<EducationOffering>;
  updateOffering?(id: string, input: UpdateEducationOfferingInput): Promise<EducationOffering>;
  authorizations?(query: InstitutionCatalogQuery): Promise<OfferingAuthorizationPage>;
  createAuthorization?(input: CreateOfferingAuthorizationInput): Promise<OfferingAuthorization>;
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
  const clean = (value?: string) => value?.trim() || undefined;
  return {
    profile: () => unwrap(client.GET("/api/institution/regulatory-profile"), validateGetApiInstitutionRegulatoryProfileResponse),
    capabilities: () => unwrap(client.GET("/api/institution/capabilities"), validateGetApiInstitutionCapabilitiesResponse),
    cutoverPreflight: () => unwrap(client.GET("/api/institution/policy-cutover-preflight"), validateGetApiInstitutionPolicyCutoverPreflightResponse),
    locations: (query) => unwrap(client.GET("/api/institution/locations", { params: { query: { page: query.page, pageSize: query.pageSize, sort: query.sort, direction: query.direction, "filter.code": clean(query.filters?.code), "filter.name": clean(query.filters?.name), "filter.active": clean(query.filters?.active), "filter.effective_from": clean(query.filters?.effective_from) } } }), validateGetApiInstitutionLocationsResponse),
    createLocation: (body) => unwrap(client.POST("/api/institution/locations", { body }), validatePostApiInstitutionLocationsResponse),
    updateLocation: (locationID, body) => unwrap(client.PATCH("/api/institution/locations/{locationID}", { params: { path: { locationID } }, body }), validatePatchApiInstitutionLocationsLocationidResponse),
    offerings: (query) => unwrap(client.GET("/api/institution/education-offerings", { params: { query: { page: query.page, pageSize: query.pageSize, sort: query.sort, direction: query.direction, "filter.code": clean(query.filters?.code), "filter.education_level": clean(query.filters?.education_level), "filter.title": clean(query.filters?.title), "filter.language_code": clean(query.filters?.language_code), "filter.active": clean(query.filters?.active), "filter.effective_from": clean(query.filters?.effective_from) } } }), validateGetApiInstitutionEducationOfferingsResponse),
    createOffering: (body) => unwrap(client.POST("/api/institution/education-offerings", { body }), validatePostApiInstitutionEducationOfferingsResponse),
    updateOffering: (offeringID, body) => unwrap(client.PATCH("/api/institution/education-offerings/{offeringID}", { params: { path: { offeringID } }, body }), validatePatchApiInstitutionEducationOfferingsOfferingidResponse),
    authorizations: (query) => unwrap(client.GET("/api/institution/offering-authorizations", { params: { query: { page: query.page, pageSize: query.pageSize, sort: query.sort, direction: query.direction, "filter.offering_code": clean(query.filters?.offering_code), "filter.location_code": clean(query.filters?.location_code), "filter.status": clean(query.filters?.status), "filter.decision_reference": clean(query.filters?.decision_reference), "filter.effective_from": clean(query.filters?.effective_from) } } }), validateGetApiInstitutionOfferingAuthorizationsResponse),
    createAuthorization: (body) => unwrap(client.POST("/api/institution/offering-authorizations", { body }), validatePostApiInstitutionOfferingAuthorizationsResponse),
    saveProfile: (body) => unwrap(client.PUT("/api/institution/regulatory-profile", { body }), validateGetApiInstitutionRegulatoryProfileResponse),
  };
}
