import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { AdminDashboardResponse, AdminUser, AdminUserFilters, PagedResponse, TableQuery } from './api.types';

@Injectable({ providedIn: 'root' })
export class AdminApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<AdminDashboardResponse>('/api/admin/dashboard');
  }

  users(query: TableQuery) {
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

    return this.http.get<PagedResponse<AdminUser>>('/api/admin/users', { params });
  }

  userFilters() {
    return this.http.get<AdminUserFilters>('/api/admin/users/filters');
  }
}
