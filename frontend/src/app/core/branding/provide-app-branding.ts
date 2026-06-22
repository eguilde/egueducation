import { APP_INITIALIZER, EnvironmentProviders, inject, makeEnvironmentProviders } from '@angular/core';

import { AppBrandingService } from './app-branding.service';

export function provideAppBranding(): EnvironmentProviders {
  return makeEnvironmentProviders([
    {
      provide: APP_INITIALIZER,
      multi: true,
      useFactory: () => {
        const branding = inject(AppBrandingService);
        return async () => {
          await branding.init();
        };
      },
    },
  ]);
}
