import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { SessionContext } from './authz.types';

@Injectable({ providedIn: 'root' })
export class AuthzService {
  private readonly http = inject(HttpClient);
  private readonly sessionSignal = signal<SessionContext | null>(null);

  readonly session = this.sessionSignal.asReadonly();
  readonly permissions = computed(() => this.sessionSignal()?.permissions ?? []);
  readonly modules = computed(() => this.sessionSignal()?.modules ?? []);
  readonly user = computed(() => this.sessionSignal()?.user ?? null);
  readonly institutionName = computed(() => this.sessionSignal()?.institution_name ?? '');

  async init(): Promise<void> {
    await this.reload();
  }

  async reload(): Promise<void> {
    try {
      const session = await firstValueFrom(this.http.get<SessionContext>('/api/me'));
      this.sessionSignal.set(session ?? null);
    } catch {
      this.sessionSignal.set(null);
    }
  }

  clearSession(): void {
    this.sessionSignal.set(null);
  }

  hasPermission(permission: string): boolean {
    return this.permissions().includes(permission);
  }

  hasAnyPermission(permissions: string[]): boolean {
    return permissions.some((permission) => this.hasPermission(permission));
  }

  hasModule(moduleCode: string): boolean {
    return this.modules().some((module) => module.code === moduleCode && module.active);
  }
}
