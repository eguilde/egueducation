import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  AppMeta,
  AuthConsentDecisionRequest,
  AuthConsentDecisionResponse,
  AuthConsentRequestResponse,
  AuthMethodsResponse,
  AuthUiConfig,
  RequestSMSOTPRequest,
  SessionContext,
  SMSOTPRequestResponse,
  SMSOTPVerifyResponse,
  VerifySMSOTPRequest,
} from './api.types';

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

  requestSmsOtp(payload: RequestSMSOTPRequest) {
    return this.http.post<SMSOTPRequestResponse>('/api/auth/request-sms', payload);
  }

  verifySmsOtp(payload: VerifySMSOTPRequest) {
    return this.http.post<SMSOTPVerifyResponse>('/api/auth/verify-sms', payload);
  }

  consentRequest(requestId: string) {
    return this.http.get<AuthConsentRequestResponse>(`/api/oidc/consent/request?request=${encodeURIComponent(requestId)}`);
  }

  decideConsent(payload: AuthConsentDecisionRequest) {
    return this.http.post<AuthConsentDecisionResponse>('/api/oidc/consent/decision', payload);
  }

  logout() {
    return this.http.post<{ status: string }>('/api/auth/logout', {});
  }
}
