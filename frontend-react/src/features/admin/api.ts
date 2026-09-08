import type { AdminApi, AdminResource, AdminResourcePath, AdminUser, Dashboard, ModuleSetting, Page, Role, UpsertUserInput, AdminWritableResourcePath } from "./types";
import { createOpenApiTransport } from "../../api/client";

type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

const asPage = <T,>(value: unknown): Page<T> => {
  if (Array.isArray(value)) return { items: value as T[], total: value.length, page: 1, pageSize: value.length };
  const page = value as Partial<Page<T>>;
  return { items: Array.isArray(page.items) ? page.items : [], total: Number(page.total ?? 0), page: Number(page.page ?? 1), pageSize: Number(page.pageSize ?? 50) };
};

export function createAdminApi(fetcher: Fetcher = fetch, apiBase = "/api"): AdminApi {
  const transport = createOpenApiTransport((request) => fetcher(request), apiBase);
  const request = async <T,>(path: string, init?: RequestInit): Promise<T> => {
    const result = await transport.request<T>((init?.method as "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | undefined) ?? "GET", `${apiBase}${path}`, {
      ...init,
      headers: { Accept: "application/json", ...(init?.headers ?? {}) },
    });
    if (!result.response.ok) {
      const detail = (result.error ?? result.data) as { code?: string } | undefined;
      throw new Error(detail?.code ?? `admin_request_${result.response.status}`);
    }
    return result.data as T;
  };
	const resourceURL = (path: AdminResourcePath) => path.startsWith("gdpr/") ? `/${path}` : `/admin/${path}`;
  return {
    dashboard: () => request<Dashboard>("/admin/dashboard"),
    users: (query = "") => {
      const search = query.trim();
      const filter = search ? `&filter.${search.includes("@") ? "email" : "name"}=${encodeURIComponent(search)}` : "";
      return request<unknown>(`/admin/users?page=1&pageSize=50${filter}`).then(asPage<AdminUser>);
    },
    saveUser: (input) => request<AdminUser>("/admin/users", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }),
    roles: () => request<unknown>("/admin/roles?page=1&pageSize=100").then(asPage<Role>),
    modules: () => request<unknown>("/admin/modules?page=1&pageSize=100").then(asPage<ModuleSetting>),
    saveModule: (input) => request<ModuleSetting>("/admin/modules", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }),
		resource: (path: AdminResourcePath) => request<unknown>(resourceURL(path)).then(asPage<AdminResource>),
		saveResource: (path: AdminWritableResourcePath, input: Record<string, unknown>) => request<AdminResource>(resourceURL(path), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }),
  };
}
