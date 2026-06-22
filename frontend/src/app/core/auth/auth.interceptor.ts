import { HttpErrorResponse, HttpEvent, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, from, throwError } from 'rxjs';
import { catchError, switchMap, tap } from 'rxjs/operators';

import { AuthzService } from '../authz/authz.service';
import { APP_ENVIRONMENT } from '../config/app-environment';
import { ThemeService } from '../ui/theme.service';
import { AuthService } from './auth.service';
import { AUTH_CONFIG } from './auth.types';
import { DPoPProofService } from './dpop-proof.service';

export const authInterceptor: HttpInterceptorFn = (req, next): Observable<HttpEvent<unknown>> => {
  const auth = inject(AuthService);
  const authz = inject(AuthzService);
  const config = inject(AUTH_CONFIG);
  const dpop = inject(DPoPProofService);
  const environment = inject(APP_ENVIRONMENT);
  const router = inject(Router);
  const theme = inject(ThemeService);

  if (!req.url.startsWith('/api')) {
    return next(req);
  }

  const oidcPrefix = config.oidcPathPrefix ?? '/api/oidc/';
  if (req.url.startsWith(oidcPrefix) || req.url.includes(oidcPrefix)) {
    return next(req.clone({ setHeaders: { 'Accept-Language': theme.language() } }));
  }

  const secureRoutes = config.secureRoutes ?? ['/api'];
  const isSecure = secureRoutes.some((route) => req.url.startsWith(route));
  const apiUrl = buildApiUrl(environment.apiBaseUrl, req.url);
  const baseHeaders = { 'Accept-Language': theme.language() };

  const unauthorizedHandler = (error: HttpErrorResponse): void => {
    const currentUrl = router.url || '/dashboard';
    const isAuthScreen = currentUrl.startsWith('/callback') || currentUrl.startsWith('/auth/');
    const isBootstrapRequest = req.url === '/api/me';

    if (error.status === 401 && !isAuthScreen && !isBootstrapRequest) {
      auth.clearLocalSession();
      authz.clearSession();
      auth.storeReturnUrl(currentUrl);
      void router.navigate(['/']);
    }

    if (error.status === 403 && !currentUrl.startsWith('/auth/access-denied')) {
      void router.navigate(['/auth/access-denied'], { queryParams: { from: currentUrl } });
    }
  };

  if (!isSecure) {
    return next(req.clone({ url: apiUrl, setHeaders: baseHeaders })).pipe(
      catchError((error: unknown) => {
        if (error instanceof HttpErrorResponse) {
          unauthorizedHandler(error);
        }
        return throwError(() => error);
      }),
    );
  }

  const token = auth.accessToken();
  if (!token) {
    return next(req.clone({ url: apiUrl, setHeaders: baseHeaders })).pipe(
      catchError((error: unknown) => {
        if (error instanceof HttpErrorResponse) {
          unauthorizedHandler(error);
        }
        return throwError(() => error);
      }),
    );
  }

  const captureNonce = (error: HttpErrorResponse): void => {
    const nonce = error.headers.get('DPoP-Nonce');
    if (nonce) {
      dpop.updateNonce(nonce);
    }
  };

  return from(auth.authHeaders(req.method, apiUrl)).pipe(
    switchMap((headers): Observable<HttpEvent<unknown>> =>
      next(req.clone({ url: apiUrl, setHeaders: { ...baseHeaders, ...headers } })).pipe(
        tap({
          next: (event: HttpEvent<unknown>) => {
            const nonce = (event as { headers?: { get(name: string): string | null } })?.headers?.get?.('DPoP-Nonce');
            if (nonce) {
              dpop.updateNonce(nonce);
            }
          },
        }),
        catchError((error: unknown): Observable<HttpEvent<unknown>> => {
          if (!(error instanceof HttpErrorResponse)) {
            return throwError(() => error);
          }

          captureNonce(error);

          if (
            config.useDPoP &&
            error.status === 401 &&
            error.headers.get('DPoP-Nonce') &&
            (error.headers.get('WWW-Authenticate') ?? '').includes('use_dpop_nonce')
          ) {
            return from(auth.authHeaders(req.method, apiUrl)).pipe(
              switchMap((retryHeaders): Observable<HttpEvent<unknown>> =>
                next(req.clone({ url: apiUrl, setHeaders: { ...baseHeaders, ...retryHeaders } })),
              ),
            );
          }

          if (error.status === 401) {
            return from(auth.tryRefresh()).pipe(
              switchMap((success): Observable<HttpEvent<unknown>> => {
                if (!success) {
                  unauthorizedHandler(error);
                  return throwError(() => error);
                }
                return from(auth.authHeaders(req.method, apiUrl)).pipe(
                  switchMap((retryHeaders): Observable<HttpEvent<unknown>> =>
                    next(req.clone({ url: apiUrl, setHeaders: { ...baseHeaders, ...retryHeaders } })),
                  ),
                );
              }),
            );
          }

          unauthorizedHandler(error);
          return throwError(() => error);
        }),
      ),
    ),
  );
};

function buildApiUrl(apiBaseUrl: string, requestUrl: string): string {
  const normalizedBase = apiBaseUrl.replace(/\/$/, '');
  return `${normalizedBase}${requestUrl.slice('/api'.length)}`;
}
