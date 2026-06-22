import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  AppMeta,
  AuthMethodsResponse,
  AuthUiConfig,
  RoleCatalogResponse,
  RolePositionResponse,
  SessionContext,
} from './api.types';
import { AppBootstrapConfig } from '../branding/app-branding.types';

@Injectable({ providedIn: 'root' })
export class AppApiService {
  private readonly http = inject(HttpClient);

  meta() {
    return this.http.get<AppMeta>('/api/meta/app');
  }

  config() {
    return this.http.get<AppBootstrapConfig>('/api/config');
  }

  authMethods() {
    return this.http.get<AuthMethodsResponse>('/api/auth/methods');
  }

  authUiConfig() {
    return this.http.get<AuthUiConfig>('/api/auth/ui-config');
  }

  roleCatalog() {
    return this.http.get<RoleCatalogResponse>('/api/auth/role-catalog');
  }

  rolePositions() {
    return this.http.get<RolePositionResponse>('/api/auth/role-positions');
  }

  session() {
    return this.http.get<SessionContext>('/api/me');
  }

  logout() {
    return this.http.post<{ status: string }>('/api/auth/logout', {});
  }
}
