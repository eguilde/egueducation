import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { AppMeta, AuthMethodsResponse, AuthUiConfig, SessionContext } from './api.types';

@Injectable({ providedIn: 'root' })
export class AppApiService {
  private readonly http = inject(HttpClient);

  meta() {
    return this.http.get<AppMeta>('/api/meta/app');
  }

  authMethods() {
    return this.http.get<AuthMethodsResponse>('/api/auth/methods');
  }

  authUiConfig() {
    return this.http.get<AuthUiConfig>('/api/auth/ui-config');
  }

  session() {
    return this.http.get<SessionContext>('/api/me');
  }
}
