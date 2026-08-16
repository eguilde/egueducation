import type { PasskeyCredential, PasskeyRegistrationOptions, PasskeyRegistrationResult, ProfileApi, ProfileUser } from "./types";

type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
const requestError = async (response: Response) => {
  const body = await response.json().catch(() => undefined) as { code?: string } | undefined;
  throw new Error(body?.code ?? `profile_request_${response.status}`);
};
export function createProfileApi(fetcher: Fetcher = fetch, apiBase = "/api"): ProfileApi {
  const request = async <T,>(path: string, init?: RequestInit): Promise<T> => {
    const response = await fetcher(`${apiBase}${path}`, { credentials: "include", ...init, headers: { Accept: "application/json", ...(init?.headers ?? {}) } });
    if (!response.ok) await requestError(response);
    return response.json() as Promise<T>;
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
