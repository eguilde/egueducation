import { EnvironmentProviders, LOCALE_ID, makeEnvironmentProviders } from '@angular/core';
import {
  TRANSLOCO_CONFIG,
  TRANSLOCO_LOADER,
  TranslocoService,
  provideTransloco,
  translocoConfig,
} from '@jsverse/transloco';

import { AppTranslocoLoader } from './transloco-loader';

export function provideAppI18n(): EnvironmentProviders {
  return makeEnvironmentProviders([
    provideTransloco({
      config: translocoConfig({
        availableLangs: ['ro', 'en'],
        defaultLang: 'ro',
        fallbackLang: 'ro',
        reRenderOnLangChange: true,
        prodMode: false,
      }),
      loader: AppTranslocoLoader,
    }),
    {
      provide: LOCALE_ID,
      deps: [TranslocoService],
      useFactory: (transloco: TranslocoService) => transloco.getActiveLang(),
    },
  ]);
}
