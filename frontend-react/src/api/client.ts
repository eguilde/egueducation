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
