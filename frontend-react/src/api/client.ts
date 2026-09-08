import createClient, { type Client } from 'openapi-fetch';
import type { paths } from './generated';

export type ContractClient = Client<paths>;

// OpenAPI paths already contain the /api prefix. VITE_API_BASE_URL historically
// points at that prefix, so the generated client must use its parent as baseUrl.
export function contractServerBaseUrl(
  apiBaseUrl: string,
  origin = typeof window === 'undefined' ? '' : window.location.origin,
): string {
  const normalized = apiBaseUrl.trim().replace(/\/+$/, '');
  const parent = normalized.replace(/\/api$/i, '');
  if (/^https?:\/\//i.test(parent)) return parent;
  if (!origin) return parent;
  return new URL(parent || '/', origin).toString().replace(/\/$/, '');
}

export function createContractClient(
  fetcher: (request: Request) => Promise<Response> = (request) => fetch(request),
  apiBaseUrl = '/api',
): ContractClient {
  return createClient<paths>({
    baseUrl: contractServerBaseUrl(apiBaseUrl),
    credentials: 'include',
    fetch: fetcher,
  });
}

/**
 * Small compatibility bridge for feature adapters while their consumer-facing
 * interfaces remain intentionally independent from generated OpenAPI names.
 *
 * It is deliberately the *only* place where an adapter may translate a
 * RequestInit-shaped call into openapi-fetch options.  All browser requests
 * therefore share the generated contract client, base URL normalisation and
 * credential policy.  Feature code must not call `fetch` directly.
 */
export type OpenApiMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
export type OpenApiResult<T> = { data: T | undefined; error?: unknown; response: Response };

const requestBody = (body: BodyInit | null | undefined, headers: HeadersInit | undefined): unknown => {
  if (typeof body !== 'string') return body;
  const contentType = new Headers(headers).get('content-type') ?? '';
  if (!contentType.toLowerCase().includes('application/json')) return body;
  return body === '' ? undefined : JSON.parse(body);
};

export function createOpenApiTransport(
  fetcher: (request: Request) => Promise<Response> = (request) => fetch(request),
  apiBaseUrl = '/api',
) {
  const client = createContractClient(fetcher, apiBaseUrl);

  return {
    async request<T>(method: OpenApiMethod, path: string, init: RequestInit = {}): Promise<OpenApiResult<T>> {
      // The generated client is generic over literal paths. Feature adapters
      // retain dynamic resource IDs, so this narrow boundary performs the
      // runtime call; `quality:contracts` independently verifies every
      // literal route used by adapters exists in generated.ts.
      const operation = client[method] as unknown as (
        route: string,
        options?: { body?: unknown; headers?: HeadersInit; parseAs?: 'json' | 'text' | 'blob' | 'arrayBuffer' | 'stream' },
      ) => Promise<OpenApiResult<T>>;
      const accept = new Headers(init.headers).get('accept') ?? '';
      const parseAs = /application\/(?:pdf|octet-stream)|text\/csv/i.test(accept) ? 'blob' : 'json';
      return operation(path, {
        headers: init.headers,
        body: requestBody(init.body, init.headers),
        parseAs,
      });
    },
  };
}
