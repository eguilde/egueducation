import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { from, throwError } from 'rxjs';
import { catchError, switchMap } from 'rxjs/operators';

import { AuthzService } from '../authz/authz.service';
import { APP_ENVIRONMENT } from '../config/app-environment';
import { ThemeService } from '../ui/theme.service';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const authz = inject(AuthzService);
  const environment = inject(APP_ENVIRONMENT);
  const router = inject(Router);
  const theme = inject(ThemeService);

  if (!req.url.startsWith('/api')) {
    return next(req);
  }

  const apiUrl = buildApiUrl(environment.apiBaseUrl, req.url);

  return from(auth.authHeaders(req.method, apiUrl)).pipe(
    switchMap((headers) =>
      next(
        req.clone({
          url: apiUrl,
          setHeaders: {
            ...headers,
            'Accept-Language': theme.language(),
          },
        }),
      ),
    ),
    catchError((error: unknown) => {
      if (error instanceof HttpErrorResponse) {
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
      }

      return throwError(() => error);
    }),
  );
};

function buildApiUrl(apiBaseUrl: string, requestUrl: string): string {
  const normalizedBase = apiBaseUrl.replace(/\/$/, '');
  return `${normalizedBase}${requestUrl.slice('/api'.length)}`;
}
