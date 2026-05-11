import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  ArchiveDashboardResponse,
  ArchiveFilters,
  ArchiveRecord,
  CreateArchiveRecordRequest,
  PagedResponse,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class EarchivaApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<ArchiveDashboardResponse>('/api/earchiva/dashboard');
  }

  records(query: TableQuery) {
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

    return this.http.get<PagedResponse<ArchiveRecord>>('/api/earchiva/records', { params });
  }

  filters() {
    return this.http.get<ArchiveFilters>('/api/earchiva/records/filters');
  }

  createRecord(payload: CreateArchiveRecordRequest) {
    return this.http.post<ArchiveRecord>('/api/earchiva/records', payload);
  }
}
