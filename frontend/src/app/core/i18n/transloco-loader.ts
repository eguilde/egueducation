import { Injectable } from '@angular/core';
import { TranslocoLoader } from '@jsverse/transloco';
import { Observable, from } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class AppTranslocoLoader implements TranslocoLoader {
  getTranslation(lang: string): Observable<Record<string, unknown>> {
    return from(fetch(`/i18n/${lang}.json`).then((response) => response.json()));
  }
}
