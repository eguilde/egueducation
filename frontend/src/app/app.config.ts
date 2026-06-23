import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideRouter, withComponentInputBinding, withInMemoryScrolling } from '@angular/router';
import { providePrimeNG } from 'primeng/config';
import { updatePrimaryPalette, updateSurfacePalette } from '@primeuix/themes';
import Aura from '@primeuix/themes/aura';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { provideAppBranding } from './core/branding/provide-app-branding';
import { provideAppAuth } from './core/auth/provide-auth';
import { APP_ENVIRONMENT } from './core/config/app-environment';
import { provideAppI18n } from './core/i18n/transloco.providers';
import { environment } from '../environments/environment';

updatePrimaryPalette({
  50: '#fff1f2',
  100: '#ffe4e6',
  200: '#fecdd3',
  300: '#fda4af',
  400: '#fb7185',
  500: '#f43f5e',
  600: '#e11d48',
  700: '#be123c',
  800: '#9f1239',
  900: '#881337',
  950: '#4c0519',
});

updateSurfacePalette({
  0: '#ffffff',
  50: '#f8fafc',
  100: '#f1f5f9',
  200: '#e2e8f0',
  300: '#cbd5e1',
  400: '#94a3b8',
  500: '#64748b',
  600: '#475569',
  700: '#334155',
  800: '#1e293b',
  900: '#0f172a',
  950: '#020617',
});

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    { provide: APP_ENVIRONMENT, useValue: environment },
    provideAnimationsAsync(),
    providePrimeNG({
      ripple: true,
      inputStyle: 'filled',
      theme: {
        preset: Aura,
        options: {
          prefix: 'p',
          darkModeSelector: '.app-dark',
          cssLayer: {
            name: 'primeng',
            order: 'theme, base, primeng',
          },
        },
      },
    }),
    provideHttpClient(withInterceptors([authInterceptor])),
    provideRouter(
      routes,
      withComponentInputBinding(),
      withInMemoryScrolling({ scrollPositionRestoration: 'enabled' }),
    ),
    provideAppI18n(),
    provideAppBranding(),
    provideAppAuth({
      authority: environment.oidcAuthority,
      clientId: environment.oidcClientId,
      redirectUri: `${window.location.origin}/auth/callback`,
      postLogoutRedirectUri: `${window.location.origin}/auth/logged-out`,
      scope: environment.authScope,
      secureRoutes: ['/api'],
      useDPoP: true,
      indexedDbName: 'egueducation_oidc_dpop',
      oidcPathPrefix: '/api/oidc/',
      renewBeforeExpirySec: 60,
      storagePrefix: 'egueducation_auth',
    }),
  ],
};
