import { DOCUMENT } from '@angular/common';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { firstValueFrom } from 'rxjs';

import { AppApiService } from '../api/app-api.service';
import { AppBootstrapConfig } from './app-branding.types';

@Injectable({ providedIn: 'root' })
export class AppBrandingService {
  private readonly api = inject(AppApiService);
  private readonly title = inject(Title);
  private readonly document = inject(DOCUMENT);
  private readonly projectTitleSignal = signal('EguEducation');

  readonly projectTitle = this.projectTitleSignal.asReadonly();
  readonly pageTitle = computed(() => this.projectTitleSignal());

  async init(): Promise<void> {
    this.applyTitle(this.fallbackTitleFromHostname());

    try {
      const config = await firstValueFrom(this.api.config());
      const bootstrap = config as AppBootstrapConfig;
      const configuredTitle = bootstrap.customer?.name || bootstrap.service?.title;
      if (configuredTitle) {
        this.applyTitle(configuredTitle);
      }
    } catch {
      // Keep the hostname-derived fallback when bootstrap config is unavailable.
    }
  }

  private applyTitle(value: string): void {
    const normalized = this.normalizeTitle(value);
    if (!normalized) {
      return;
    }

    this.projectTitleSignal.set(normalized);
    this.title.setTitle(normalized);
  }

  private normalizeTitle(value: string): string {
    return value
      .normalize('NFKD')
      .replace(/\p{Diacritic}/gu, '')
      .replace(/[_-]+/g, ' ')
      .replace(/\s+/g, ' ')
      .trim();
  }

  private fallbackTitleFromHostname(): string {
    const hostname = this.document.location.hostname.trim().toLowerCase();
    if (!hostname || hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '[::1]') {
      return 'EguEducation';
    }

    const labels = hostname.split('.').filter(Boolean);
    const tenant = labels.find((label) => !['www', 'app'].includes(label)) ?? labels[0] ?? '';
    if (!tenant) {
      return 'EguEducation';
    }

    return tenant
      .replace(/[-_]+/g, ' ')
      .replace(/\s+/g, ' ')
      .trim()
      .split(' ')
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ') || 'EguEducation';
  }
}
