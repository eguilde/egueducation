import { HttpClient } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import { AppApiService } from '../api/app-api.service';
import { SessionContext } from './authz.types';
import { RoleCatalogItem, RolePositionItem } from '../api/api.types';

@Injectable({ providedIn: 'root' })
export class AuthzService {
  private readonly http = inject(HttpClient);
  private readonly api = inject(AppApiService);
  private readonly sessionSignal = signal<SessionContext | null>(null);
  private readonly roleCatalogSignal = signal<RoleCatalogItem[]>([]);
  private readonly rolePositionsSignal = signal<RolePositionItem[]>([]);
  private readonly initializedSignal = signal(false);

  readonly session = this.sessionSignal.asReadonly();
  readonly roleCatalog = this.roleCatalogSignal.asReadonly();
  readonly rolePositions = this.rolePositionsSignal.asReadonly();
  readonly permissions = computed(() => this.sessionSignal()?.permissions ?? []);
  readonly modules = computed(() => this.sessionSignal()?.modules ?? []);
  readonly user = computed(() => this.sessionSignal()?.user ?? null);
  readonly roles = computed(() => this.sessionSignal()?.user?.roles ?? []);
  readonly institutionName = computed(() => this.sessionSignal()?.institution_name ?? '');
  readonly initialized = this.initializedSignal.asReadonly();

  async init(): Promise<void> {
    this.sessionSignal.set(null);
  }

  async reload(): Promise<void> {
    try {
      const session = await firstValueFrom(this.http.get<SessionContext>('/api/me'));
      this.sessionSignal.set(session ?? null);
      this.initializedSignal.set(true);
    } catch {
      this.sessionSignal.set(null);
      this.initializedSignal.set(true);
    }
  }

  clearSession(): void {
    this.sessionSignal.set(null);
  }

  async bootstrapAuthenticated(): Promise<void> {
    await this.reload();
    await this.loadRoleCatalog();
    await this.loadRolePositions();
  }

  roleLabel(roleCode: string): string {
    const normalizedRoleCode = this.normalizeRoleCode(roleCode);
    return this.roleCatalog().find((role) => this.normalizeRoleCode(role.code) === normalizedRoleCode)?.label ?? roleCode;
  }

  rolePermissions(roleCode: string): string[] {
    return this.roleCatalog().find((role) => role.code === roleCode)?.permissions ?? [];
  }

  hasPermission(permission: string): boolean {
    return this.permissions().includes(permission);
  }

  hasRole(role: string): boolean {
    const normalizedRole = this.normalizeRoleCode(role);
    return this.roles().some((currentRole) => this.normalizeRoleCode(currentRole) === normalizedRole);
  }

  hasAnyRole(roles: string[]): boolean {
    return roles.some((role) => this.hasRole(role));
  }

  hasAnyPermission(permissions: string[]): boolean {
    return permissions.some((permission) => this.hasPermission(permission));
  }

  hasModule(moduleCode: string): boolean {
    return this.modules().some((module) => module.code === moduleCode && module.active);
  }

  hasAnyModule(moduleCodes: string[]): boolean {
    return moduleCodes.some((moduleCode) => this.hasModule(moduleCode));
  }

  hasFullAccess(): boolean {
    return this.hasRole('super_admin') || this.hasRole('admin');
  }

  private async loadRoleCatalog(): Promise<void> {
    try {
      const response = await firstValueFrom(this.api.roleCatalog());
      this.roleCatalogSignal.set(response?.roles ?? []);
    } catch {
      this.roleCatalogSignal.set([]);
    }
  }

  private async loadRolePositions(): Promise<void> {
    try {
      const response = await firstValueFrom(this.api.rolePositions());
      this.rolePositionsSignal.set(response?.items ?? []);
    } catch {
      this.rolePositionsSignal.set([]);
    }
  }

  private normalizeRoleCode(role: string): string {
    return role.toLowerCase().replace(/[^a-z0-9]+/g, '');
  }
}
