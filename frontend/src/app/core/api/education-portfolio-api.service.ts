import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreatePortfolioRecordRequest,
  PagedResponse,
  PortfolioDashboardResponse,
  PortfolioFilters,
  PortfolioRecord,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class EducationPortfolioApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<PortfolioDashboardResponse>('/api/education/portfolios/dashboard');
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

    return this.http.get<PagedResponse<PortfolioRecord>>('/api/education/portfolios/records', { params });
  }

  filters() {
    return this.http.get<PortfolioFilters>('/api/education/portfolios/records/filters');
  }

  createRecord(payload: CreatePortfolioRecordRequest) {
    return this.http.post<PortfolioRecord>('/api/education/portfolios/records', payload);
  }
}
