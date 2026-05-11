import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreatePersonnelDeclarationRequest,
  CreatePersonnelEvaluationRequest,
  CreatePersonnelRecordRequest,
  PagedResponse,
  PersonnelDeclaration,
  PersonnelDeclarationDashboardResponse,
  PersonnelDeclarationFilters,
  PersonnelEvaluation,
  PersonnelEvaluationDashboardResponse,
  PersonnelEvaluationFilters,
  PersonnelDashboardResponse,
  PersonnelFilters,
  PersonnelRecord,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class EducationPersonnelApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<PersonnelDashboardResponse>('/api/education/personnel/dashboard');
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

    return this.http.get<PagedResponse<PersonnelRecord>>('/api/education/personnel/records', { params });
  }

  filters() {
    return this.http.get<PersonnelFilters>('/api/education/personnel/records/filters');
  }

  createRecord(payload: CreatePersonnelRecordRequest) {
    return this.http.post<PersonnelRecord>('/api/education/personnel/records', payload);
  }

  evaluationsDashboard() {
    return this.http.get<PersonnelEvaluationDashboardResponse>('/api/education/evaluations/dashboard');
  }

  evaluations(query: TableQuery) {
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

    return this.http.get<PagedResponse<PersonnelEvaluation>>('/api/education/evaluations/records', { params });
  }

  evaluationFilters() {
    return this.http.get<PersonnelEvaluationFilters>('/api/education/evaluations/records/filters');
  }

  createEvaluation(payload: CreatePersonnelEvaluationRequest) {
    return this.http.post<PersonnelEvaluation>('/api/education/evaluations/records', payload);
  }

  declarationsDashboard() {
    return this.http.get<PersonnelDeclarationDashboardResponse>('/api/education/declarations/dashboard');
  }

  declarations(query: TableQuery) {
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

    return this.http.get<PagedResponse<PersonnelDeclaration>>('/api/education/declarations/records', { params });
  }

  declarationFilters() {
    return this.http.get<PersonnelDeclarationFilters>('/api/education/declarations/records/filters');
  }

  createDeclaration(payload: CreatePersonnelDeclarationRequest) {
    return this.http.post<PersonnelDeclaration>('/api/education/declarations/records', payload);
  }
}
