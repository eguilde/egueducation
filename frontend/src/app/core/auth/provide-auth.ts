import { APP_INITIALIZER, EnvironmentProviders, inject, makeEnvironmentProviders } from '@angular/core';

import { AuthzService } from '../authz/authz.service';
import { AuthService } from './auth.service';
import { AUTH_CONFIG, AuthConfig } from './auth.types';

export function provideAppAuth(config: AuthConfig): EnvironmentProviders {
  return makeEnvironmentProviders([
    { provide: AUTH_CONFIG, useValue: config },
    {
      provide: APP_INITIALIZER,
      multi: true,
      useFactory: () => {
        const auth = inject(AuthService);
        const authz = inject(AuthzService);
        return async () => {
          await auth.init();
          await authz.init();
          if (auth.isAuthenticated()) {
            await authz.bootstrapAuthenticated();
          }
        };
      },
    },
  ]);
}
