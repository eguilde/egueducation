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

const isFormDataBody = (value: unknown): value is FormData =>
  typeof value === 'object' &&
  value !== null &&
  Object.prototype.toString.call(value) === '[object FormData]' &&
  typeof (value as FormData).entries === 'function';

const multipartToken = () => {
  const random = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID().replaceAll('-', '')
    : Math.random().toString(36).slice(2);
  return `----egueducation-${random}`;
};

const multipartQuoted = (value: string) => value.replace(/[\r\n]/g, '').replace(/(["\\])/g, '\\$1');

/**
 * Encode multipart explicitly at the OpenAPI transport boundary. A Blob keeps
 * binary File parts intact without converting them to base64, while the exact
 * boundary is carried in Content-Type. This also works when an optimized
 * dependency executes in a different WebIDL realm and `instanceof FormData`
 * would otherwise make openapi-fetch serialize the body as JSON.
 */
const encodeMultipart = (form: FormData): { body: Blob; contentType: string } => {
  const boundary = multipartToken();
  const parts: BlobPart[] = [];
  for (const [name, value] of form.entries()) {
    parts.push(`--${boundary}\r\nContent-Disposition: form-data; name="${multipartQuoted(name)}"`);
    if (typeof value === 'string') {
      parts.push(`\r\n\r\n${value}\r\n`);
      continue;
    }
    parts.push(`; filename="${multipartQuoted(value.name)}"\r\nContent-Type: ${value.type || 'application/octet-stream'}\r\n\r\n`);
    parts.push(value, '\r\n');
  }
  parts.push(`--${boundary}--\r\n`);
  return { body: new Blob(parts), contentType: `multipart/form-data; boundary=${boundary}` };
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
        options?: {
          body?: unknown;
          headers?: HeadersInit | Record<string, string | null>;
          parseAs?: 'json' | 'text' | 'blob' | 'arrayBuffer' | 'stream';
          bodySerializer?: (body: unknown) => BodyInit;
        },
      ) => Promise<OpenApiResult<T>>;
      const accept = new Headers(init.headers).get('accept') ?? '';
      const parseAs = /application\/(?:pdf|octet-stream)|text\/csv/i.test(accept) ? 'blob' : 'json';
      const rawBody = requestBody(init.body, init.headers);
      const multipart = isFormDataBody(rawBody) ? encodeMultipart(rawBody) : undefined;
      const body = multipart?.body ?? rawBody;
      const headers = multipart
        ? { ...Object.fromEntries(new Headers(init.headers).entries()), 'Content-Type': multipart.contentType }
        : init.headers;
      return operation(path, {
        headers,
        body,
        parseAs,
        // The multipart Blob is already encoded with the boundary declared in
        // headers, so it must pass through without JSON serialization.
        bodySerializer: multipart ? (value) => value as BodyInit : undefined,
      });
    },
  };
}
