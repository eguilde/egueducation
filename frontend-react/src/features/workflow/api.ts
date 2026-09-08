import type { components } from '../../api/generated';
import { createContractClient } from '../../api/client';

export type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
export type FluxDocument = NonNullable<components['schemas']['get_api_registratura_flux_queue_item']>;
export type FluxPage = { items: FluxDocument[]; total: number; page: number; pageSize: number };
/** @deprecated Feature consumers should use FluxPage. Kept while archive contracts migrate. */
export type Page<T> = { items: T[]; total: number; page: number; pageSize: number };
export type FluxStats = Array<NonNullable<components['schemas']['get_api_registratura_flux_pipeline_stats_item']>>;
export type FluxQuery = {
  page: number; pageSize: number; sort?: string; direction?: 'asc' | 'desc';
  nr_doc?: string; continut?: string; emitent?: string; compartiment?: string; tip?: string;
  mapa_filter?: 'mine' | 'peers'; status?: string;
};
export type WorkflowTransition = components['schemas']['DocumentWorkflowActionRequest'];
export type WorkflowAssignees = { departments: Array<{ id?: string; name?: string }>; users: Array<{ id?: string; name?: string; department_ids?: string[] }> };

export interface WorkflowApi {
  queue(query: FluxQuery): Promise<FluxPage>;
  mapa(query: FluxQuery): Promise<FluxPage>;
  pipeline(query: FluxQuery): Promise<FluxPage>;
  pipelineStats(): Promise<FluxStats>;
  assignees(): Promise<WorkflowAssignees>;
  transition(id: string, input: WorkflowTransition): Promise<void>;
}

const asPage = (value: { items?: FluxDocument[]; total?: number; page?: number; pageSize?: number }): FluxPage => ({
  items: value.items ?? [], total: value.total ?? 0, page: value.page ?? 1, pageSize: value.pageSize ?? 20,
});

/** Contract-first adapter. Flux projections are server-filtered and server-paged;
 * no client-side narrowing is permitted here. */
export function createWorkflowApi(fetcher: Fetcher = fetch, apiBase = '/api'): WorkflowApi {
  const contractClient = createContractClient((request) => fetcher(request), apiBase);
  const getPage = async (path: '/api/registratura/flux/queue' | '/api/registratura/flux/mapa' | '/api/registratura/flux/pipeline', query: FluxQuery) => {
    const result = await contractClient.GET(path, { params: { query } });
    if (!result.response.ok || !result.data) throw new Error(`Flux documente: ${result.response.status}`);
    return asPage(result.data as { items?: FluxDocument[]; total?: number; page?: number; pageSize?: number });
  };
  return {
    queue: (query) => getPage('/api/registratura/flux/queue', query),
    mapa: (query) => getPage('/api/registratura/flux/mapa', query),
    pipeline: (query) => getPage('/api/registratura/flux/pipeline', query),
    async pipelineStats() {
      const result = await contractClient.GET('/api/registratura/flux/pipeline/stats');
      if (!result.response.ok || !result.data) throw new Error(`Flux documente: ${result.response.status}`);
      return result.data as FluxStats;
    },
    async assignees() {
      const result = await contractClient.GET('/api/registratura/workflow-assignees');
      if (!result.response.ok || !result.data) throw new Error(`Flux documente: ${result.response.status}`);
      return result.data as WorkflowAssignees;
    },
    async transition(id, input) {
      const result = await contractClient.POST('/api/registratura/documents/{documentID}/workflow-actions', { params: { path: { documentID: id } }, body: input });
      if (!result.response.ok) throw new Error(`Flux documente: ${result.response.status}`);
    },
  };
}
