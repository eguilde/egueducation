import type { PasskeyCredential, PasskeyRegistrationOptions, PasskeyRegistrationResult, ProfileApi, ProfileUser } from "./types";
import { createOpenApiTransport } from "../../api/client";

type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
const requestError = (body: unknown, response: Response) => {
  const detail = body as { code?: string } | undefined;
  throw new Error(detail?.code ?? `profile_request_${response.status}`);
};
export function createProfileApi(fetcher: Fetcher = fetch, apiBase = "/api"): ProfileApi {
  const transport = createOpenApiTransport((request) => fetcher(request), apiBase);
  const request = async <T,>(path: string, init?: RequestInit): Promise<T> => {
    const result = await transport.request<T>((init?.method as "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | undefined) ?? "GET", `${apiBase}${path}`, { ...init, headers: { Accept: "application/json", ...(init?.headers ?? {}) } });
    if (!result.response.ok) requestError(result.data, result.response);
    return result.data as T;
  };
  return {
    update: (input) => request<ProfileUser | { user: ProfileUser }>("/profile", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) })
      .then((value) => "user" in value ? value.user : value),
    passkeys: () => request<PasskeyCredential[] | { items?: PasskeyCredential[] }>("/passkeys").then((value) => Array.isArray(value) ? value : value.items ?? []),
    registrationOptions: () => request<PasskeyRegistrationOptions>("/passkeys/register-options", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" }),
    finishRegistration: (input) => request<PasskeyCredential>("/passkeys/register-finish", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) }),
		activateEUDIWallet: () => request<{ status: string }>("/eudi-wallet/activate", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" }),
  };
}
