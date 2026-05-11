import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreateMobilityCaseRequest,
  MobilityCase,
  MobilityDashboardResponse,
  MobilityFilters,
  PagedResponse,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class EducationMobilityApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<MobilityDashboardResponse>('/api/education/mobility/dashboard');
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

    return this.http.get<PagedResponse<MobilityCase>>('/api/education/mobility/records', { params });
  }

  filters() {
    return this.http.get<MobilityFilters>('/api/education/mobility/records/filters');
  }

  createRecord(payload: CreateMobilityCaseRequest) {
    return this.http.post<MobilityCase>('/api/education/mobility/records', payload);
  }
}
