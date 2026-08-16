import { DOCUMENT } from '@angular/common';
import { DestroyRef, Injectable, computed, effect, inject, signal } from '@angular/core';
import { TranslocoService } from '@jsverse/transloco';

import { AppBrandingService } from '../branding/app-branding.service';

export type ColorScheme = 'system' | 'light' | 'dark';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private readonly document = inject(DOCUMENT);
  private readonly transloco = inject(TranslocoService);
  private readonly branding = inject(AppBrandingService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

  readonly colorScheme = signal(this.storedColorScheme());
  readonly systemPrefersDark = signal(this.mediaQuery.matches);
  readonly selectedLanguage = signal(this.storedLanguage());
  readonly isDarkMode = computed(() => {
    const scheme = this.colorScheme();
    return scheme === 'system' ? this.systemPrefersDark() : scheme === 'dark';
  });

  readonly mode = this.isDarkMode;
  readonly language = this.selectedLanguage.asReadonly();

  constructor() {
    const handler = (event: MediaQueryListEvent) => this.systemPrefersDark.set(event.matches);
    this.mediaQuery.addEventListener('change', handler);
    this.destroyRef.onDestroy(() => this.mediaQuery.removeEventListener('change', handler));

    effect(() => {
      localStorage.setItem(this.storageKey('color-scheme'), this.colorScheme());
      const isDark = this.isDarkMode();
      this.document.documentElement.classList.toggle('app-dark', isDark);
      this.document.documentElement.dataset['pTheme'] = isDark ? 'dark' : 'light';
    });

    effect(() => {
      const lang = this.selectedLanguage();
      localStorage.setItem(this.storageKey('language'), lang);
      this.transloco.setActiveLang(lang);
      this.document.documentElement.lang = lang;
    });
  }

  setColorScheme(mode: ColorScheme): void {
    this.colorScheme.set(mode);
  }

  setLanguage(lang: 'ro' | 'en'): void {
    this.selectedLanguage.set(lang);
  }

  private storedColorScheme(): ColorScheme {
    const value = localStorage.getItem(this.storageKey('color-scheme'));
    return value === 'light' || value === 'dark' || value === 'system' ? value : 'system';
  }

  private storedLanguage(): 'ro' | 'en' {
    return localStorage.getItem(this.storageKey('language')) === 'en' ? 'en' : 'ro';
  }

  private storageKey(name: string): string {
    const tenant = this.branding.institutionId().trim() || window.location.hostname.toLowerCase();
    return `egueducation.${tenant.replace(/[^a-z0-9._-]/g, '_')}.${name}`;
  }
}
