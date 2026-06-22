import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideRouter, withComponentInputBinding, withInMemoryScrolling } from '@angular/router';
import { providePrimeNG } from 'primeng/config';
import Aura from '@primeuix/themes/aura';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { provideAppBranding } from './core/branding/provide-app-branding';
import { provideAppAuth } from './core/auth/provide-auth';
import { APP_ENVIRONMENT } from './core/config/app-environment';
import { provideAppI18n } from './core/i18n/transloco.providers';
import { environment } from '../environments/environment';

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
    }),
  ],
};
