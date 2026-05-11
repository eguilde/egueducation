import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreateGovernanceDecisionRequest,
  CreateGovernanceMeetingRequest,
  CreateManagerialDossierRequest,
  CreateRegulationRecordRequest,
  EducationDashboardResponse,
  GovernanceFilters,
  GovernanceDecision,
  GovernanceDecisionDashboardResponse,
  GovernanceDecisionFilters,
  GovernanceMeeting,
  EducationTaxonomyCatalogResponse,
  ManagerialDossier,
  ManagerialDossierDashboardResponse,
  ManagerialDossierFilters,
  PagedResponse,
  RegulationDashboardResponse,
  RegulationFilters,
  RegulationRecord,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class EducationApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<EducationDashboardResponse>('/api/education/dashboard');
  }

  taxonomies(domains: string[]) {
    const params = new HttpParams().set('domains', domains.join(','));
    return this.http.get<EducationTaxonomyCatalogResponse>('/api/education/taxonomies', { params });
  }

  meetings(query: TableQuery) {
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

    return this.http.get<PagedResponse<GovernanceMeeting>>('/api/education/governance/meetings', { params });
  }

  filters() {
    return this.http.get<GovernanceFilters>('/api/education/governance/meetings/filters');
  }

  createMeeting(payload: CreateGovernanceMeetingRequest) {
    return this.http.post<GovernanceMeeting>('/api/education/governance/meetings', payload);
  }

  decisionsDashboard() {
    return this.http.get<GovernanceDecisionDashboardResponse>('/api/education/decisions/dashboard');
  }

  decisions(query: TableQuery) {
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

    return this.http.get<PagedResponse<GovernanceDecision>>('/api/education/decisions/records', { params });
  }

  decisionFilters() {
    return this.http.get<GovernanceDecisionFilters>('/api/education/decisions/records/filters');
  }

  createDecision(payload: CreateGovernanceDecisionRequest) {
    return this.http.post<GovernanceDecision>('/api/education/decisions/records', payload);
  }

  managerialDashboard() {
    return this.http.get<ManagerialDossierDashboardResponse>('/api/education/managerial/dashboard');
  }

  managerialDossiers(query: TableQuery) {
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

    return this.http.get<PagedResponse<ManagerialDossier>>('/api/education/managerial/records', { params });
  }

  managerialFilters() {
    return this.http.get<ManagerialDossierFilters>('/api/education/managerial/records/filters');
  }

  createManagerialDossier(payload: CreateManagerialDossierRequest) {
    return this.http.post<ManagerialDossier>('/api/education/managerial/records', payload);
  }

  regulationsDashboard() {
    return this.http.get<RegulationDashboardResponse>('/api/education/regulations/dashboard');
  }

  regulations(query: TableQuery) {
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

    return this.http.get<PagedResponse<RegulationRecord>>('/api/education/regulations/records', { params });
  }

  regulationFilters() {
    return this.http.get<RegulationFilters>('/api/education/regulations/records/filters');
  }

  createRegulation(payload: CreateRegulationRecordRequest) {
    return this.http.post<RegulationRecord>('/api/education/regulations/records', payload);
  }
}
