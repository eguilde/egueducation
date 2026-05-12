import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { from, throwError } from 'rxjs';
import { catchError, switchMap } from 'rxjs/operators';

import { AuthzService } from '../authz/authz.service';
import { ThemeService } from '../ui/theme.service';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const authz = inject(AuthzService);
  const router = inject(Router);
  const theme = inject(ThemeService);

  if (!req.url.startsWith('/api')) {
    return next(req);
  }

  return from(auth.authHeaders(req.method, req.url)).pipe(
    switchMap((headers) =>
      next(
        req.clone({
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
        const isAuthScreen = currentUrl.startsWith('/login') || currentUrl.startsWith('/auth/');
        const isBootstrapRequest = req.url === '/api/me';

        if (error.status === 401 && !isAuthScreen && !isBootstrapRequest) {
          auth.clearLocalSession();
          authz.clearSession();
          auth.storeReturnUrl(currentUrl);
          void router.navigate(['/login']);
        }

        if (error.status === 403 && !currentUrl.startsWith('/auth/access-denied')) {
          void router.navigate(['/auth/access-denied'], { queryParams: { from: currentUrl } });
        }
      }

      return throwError(() => error);
    }),
  );
};
