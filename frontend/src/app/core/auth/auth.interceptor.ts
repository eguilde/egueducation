import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { from } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { ThemeService } from '../ui/theme.service';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
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
  );
};
