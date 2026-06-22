import { InjectionToken } from '@angular/core';

export interface AppEnvironment {
  production: boolean;
  apiBaseUrl: string;
  oidcAuthority: string;
  oidcClientId: string;
  oidcDesktopClientId: string;
  authScope: string;
}

export const APP_ENVIRONMENT = new InjectionToken<AppEnvironment>('APP_ENVIRONMENT');
