import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreateWorkflowTaskRequest,
  PagedResponse,
  TableQuery,
  TransitionWorkflowTaskRequest,
  WorkflowDashboardResponse,
  WorkflowDefinition,
  WorkflowFilters,
  WorkflowTask,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class WorkflowApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<WorkflowDashboardResponse>('/api/workflow/dashboard');
  }

  definitions() {
    return this.http.get<WorkflowDefinition[]>('/api/workflow/definitions');
  }

  tasks(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<WorkflowTask>>('/api/workflow/tasks', { params });
  }

  taskFilters() {
    return this.http.get<WorkflowFilters>('/api/workflow/tasks/filters');
  }

  createTask(payload: CreateWorkflowTaskRequest) {
    return this.http.post<WorkflowTask>('/api/workflow/tasks', payload);
  }

  transitionTask(taskId: string, payload: TransitionWorkflowTaskRequest) {
    return this.http.post<WorkflowTask>(`/api/workflow/tasks/${taskId}/transition`, payload);
  }
}
