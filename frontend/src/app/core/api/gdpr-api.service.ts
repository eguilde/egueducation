import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreateGdprPublicationReviewRequest,
  CreateGdprRetentionPolicyRequest,
  CreateGdprSubjectExportRequest,
  CreateGdprSubjectRequestRequest,
  GdprConfigResponse,
  GdprDashboardResponse,
  GdprExportDashboardResponse,
  GdprPublicationDashboardResponse,
  GdprPublicationReview,
  GdprPublicationReviewFilters,
  GdprRetentionFilters,
  GdprRetentionPolicy,
  GdprSubjectExport,
  GdprSubjectExportFilters,
  GdprSubjectRequest,
  GdprSubjectRequestFilters,
  PagedResponse,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class GdprApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<GdprDashboardResponse>('/api/gdpr/dashboard');
  }

  config() {
    return this.http.get<GdprConfigResponse>('/api/gdpr/config');
  }

  retentionPolicies(query: TableQuery) {
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

    return this.http.get<PagedResponse<GdprRetentionPolicy>>('/api/gdpr/retention-policies', { params });
  }

  retentionFilters() {
    return this.http.get<GdprRetentionFilters>('/api/gdpr/retention-policies/filters');
  }

  createRetentionPolicy(payload: CreateGdprRetentionPolicyRequest) {
    return this.http.post<GdprRetentionPolicy>('/api/gdpr/retention-policies', payload);
  }

  subjectRequests(query: TableQuery) {
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

    return this.http.get<PagedResponse<GdprSubjectRequest>>('/api/gdpr/subject-requests', { params });
  }

  subjectRequestFilters() {
    return this.http.get<GdprSubjectRequestFilters>('/api/gdpr/subject-requests/filters');
  }

  createSubjectRequest(payload: CreateGdprSubjectRequestRequest) {
    return this.http.post<GdprSubjectRequest>('/api/gdpr/subject-requests', payload);
  }

  exportsDashboard() {
    return this.http.get<GdprExportDashboardResponse>('/api/gdpr/exports/dashboard');
  }

  exports(query: TableQuery) {
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

    return this.http.get<PagedResponse<GdprSubjectExport>>('/api/gdpr/exports', { params });
  }

  exportFilters() {
    return this.http.get<GdprSubjectExportFilters>('/api/gdpr/exports/filters');
  }

  createExport(payload: CreateGdprSubjectExportRequest) {
    return this.http.post<GdprSubjectExport>('/api/gdpr/exports', payload);
  }

  publicationDashboard() {
    return this.http.get<GdprPublicationDashboardResponse>('/api/gdpr/publication-reviews/dashboard');
  }

  publicationReviews(query: TableQuery) {
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

    return this.http.get<PagedResponse<GdprPublicationReview>>('/api/gdpr/publication-reviews', { params });
  }

  publicationReviewFilters() {
    return this.http.get<GdprPublicationReviewFilters>('/api/gdpr/publication-reviews/filters');
  }

  createPublicationReview(payload: CreateGdprPublicationReviewRequest) {
    return this.http.post<GdprPublicationReview>('/api/gdpr/publication-reviews', payload);
  }
}
