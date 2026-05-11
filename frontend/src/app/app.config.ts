import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideNativeDateAdapter } from '@angular/material/core';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideRouter, withComponentInputBinding, withInMemoryScrolling } from '@angular/router';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { provideAppAuth } from './core/auth/provide-auth';
import { provideAppI18n } from './core/i18n/transloco.providers';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideAnimationsAsync(),
    provideNativeDateAdapter(),
    provideHttpClient(withInterceptors([authInterceptor])),
    provideRouter(
      routes,
      withComponentInputBinding(),
      withInMemoryScrolling({ scrollPositionRestoration: 'enabled' }),
    ),
    provideAppI18n(),
    provideAppAuth({
      authority: '/api/oidc',
      clientId: 'egueducation-spa',
      redirectUri: `${window.location.origin}/auth/callback`,
      postLogoutRedirectUri: `${window.location.origin}/auth/logged-out`,
      scope: 'openid profile email phone offline_access',
    }),
  ],
};
