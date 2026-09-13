import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";

export type Contract = components["schemas"]["get_api_school_operations_contracts_item"];
export type ContractDetail = components["schemas"]["get_api_school_operations_contracts_contractid_response"];
export type Obligation = components["schemas"]["get_api_school_operations_contracts_contractid_obligations_item"];
export type Supplier = components["schemas"]["get_api_school_operations_suppliers_item"];
export type ContractPage = { items: Contract[]; total: number; page: number; pageSize: number };
export type ObligationPage = { items: Obligation[]; total: number; page: number; pageSize: number };
export type ContractQuery = { page: number; pageSize: number; sort?: "contract_number" | "supplier_name" | "title" | "category" | "lifecycle_status" | "starts_on" | "ends_on" | "total_value" | "archive_status"; direction?: "asc" | "desc"; filters: Record<string, string> };
export type ObligationSort = "title" | "status" | "due_on";
export type ContractInput = components["schemas"]["Request_post_api_school_operations_contracts"];
export type ContractAmendment = components["schemas"]["Request_patch_api_school_operations_contracts_contractid"];
export type ObligationInput = components["schemas"]["Request_post_api_school_operations_contracts_contractid_obligations"];

const requireData = async <T,>(request: Promise<{ data?: T; error?: unknown }>): Promise<T> => {
  const response = await request;
  if (response.error || !response.data) throw new Error("school_operations_contract_request_failed");
  return response.data;
};

const contractQuery = (value: ContractQuery) => ({
  page: value.page,
  pageSize: value.pageSize,
  sort: value.sort,
  direction: value.direction,
  "filter.contract_number": value.filters.contract_number?.trim() || undefined,
  "filter.title": value.filters.title?.trim() || undefined,
  "filter.lifecycle_status": value.filters.lifecycle_status?.trim() || undefined,
  "filter.category": value.filters.category?.trim() || undefined,
  "filter.supplier_name": value.filters.supplier_name?.trim() || undefined,
});

/** Operational contracts use generated OpenAPI literal paths exclusively. */
export function createSchoolOperationsApi(client: ContractClient) {
  return {
    list: (query: ContractQuery): Promise<ContractPage> => requireData(client.GET("/api/school-operations/contracts", { params: { query: contractQuery(query) } })),
    suppliers: (query: { page: number; pageSize: number; q?: string; sort?: "display_name" | "code" | "tax_id"; direction?: "asc" | "desc" }): Promise<{ items: Supplier[]; total: number; page: number; pageSize: number }> => requireData(client.GET("/api/school-operations/suppliers", { params: { query: { page: query.page, pageSize: query.pageSize, sort: query.sort ?? "display_name", direction: query.direction ?? "asc", "filter.display_name": query.q?.trim() || undefined } } })),
    detail: (contractID: string): Promise<ContractDetail> => requireData(client.GET("/api/school-operations/contracts/{contractID}", { params: { path: { contractID } } })),
    create: (body: ContractInput) => requireData(client.POST("/api/school-operations/contracts", { body })),
    amend: (contractID: string, body: ContractAmendment) => requireData(client.PATCH("/api/school-operations/contracts/{contractID}", { params: { path: { contractID } }, body })),
    obligations: (contractID: string, query: { page: number; pageSize: number; sort?: ObligationSort; direction?: "asc" | "desc"; filters?: Record<string, string> }): Promise<ObligationPage> => requireData(client.GET("/api/school-operations/contracts/{contractID}/obligations", { params: { path: { contractID }, query: { page: query.page, pageSize: query.pageSize, sort: query.sort, direction: query.direction, "filter.title": query.filters?.title?.trim() || undefined, "filter.status": query.filters?.status?.trim() || undefined } } })),
    addObligation: (contractID: string, body: ObligationInput) => requireData(client.POST("/api/school-operations/contracts/{contractID}/obligations", { params: { path: { contractID } }, body })),
    transition: (contractID: string, body: components["schemas"]["Request_post_api_school_operations_contracts_contractid_transition"]) => requireData(client.POST("/api/school-operations/contracts/{contractID}/transition", { params: { path: { contractID } }, body })),
  };
}

export type SchoolOperationsApi = ReturnType<typeof createSchoolOperationsApi>;
