import { DOCUMENT } from '@angular/common';
import { Injectable, effect, inject, signal } from '@angular/core';
import { TranslocoService } from '@jsverse/transloco';

type ThemeMode = 'light' | 'dark';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly document = inject(DOCUMENT);
  private readonly transloco = inject(TranslocoService);

  private readonly modeSignal = signal<ThemeMode>((localStorage.getItem('egueducation.theme') as ThemeMode) || 'light');
  private readonly languageSignal = signal<'ro' | 'en'>(
    (localStorage.getItem('egueducation.lang') as 'ro' | 'en') || 'ro',
  );

  readonly mode = this.modeSignal.asReadonly();
  readonly language = this.languageSignal.asReadonly();

  constructor() {
    effect(() => {
      const mode = this.modeSignal();
      localStorage.setItem('egueducation.theme', mode);
      const root = this.document.documentElement;
      root.classList.toggle('dark-theme', mode === 'dark');
      root.dataset['theme'] = mode;
    });

    effect(() => {
      const lang = this.languageSignal();
      localStorage.setItem('egueducation.lang', lang);
      this.transloco.setActiveLang(lang);
      this.document.documentElement.lang = lang;
    });
  }

  toggleTheme(): void {
    this.modeSignal.update((mode) => (mode === 'light' ? 'dark' : 'light'));
  }

  setLanguage(lang: 'ro' | 'en'): void {
    this.languageSignal.set(lang);
  }
}
