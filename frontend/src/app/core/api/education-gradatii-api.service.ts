import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreateMeritGrantRequest,
  MeritGrant,
  MeritGrantDashboardResponse,
  MeritGrantFilters,
  PagedResponse,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class EducationGradatiiApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<MeritGrantDashboardResponse>('/api/education/gradatii/dashboard');
  }

  records(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
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

    return this.http.get<PagedResponse<MeritGrant>>('/api/education/gradatii/records', { params });
  }

  filters() {
    return this.http.get<MeritGrantFilters>('/api/education/gradatii/records/filters');
  }

  createRecord(payload: CreateMeritGrantRequest) {
    return this.http.post<MeritGrant>('/api/education/gradatii/records', payload);
  }
}
