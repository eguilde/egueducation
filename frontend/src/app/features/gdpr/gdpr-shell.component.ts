import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';

import { AuthzService } from '../../core/authz/authz.service';

interface WorkspaceTab {
  icon: string;
  labelKey: string;
  route: string;
  permission: string | string[];
  permissionMode?: 'all' | 'any';
  module: string;
}

@Component({
  selector: 'app-gdpr-shell',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive, TranslocoPipe, MatIconModule, MatTabsModule],
  template: `
    <section class="space-y-4">
      <header class="rounded-3xl border border-[rgb(148_163_184_/_0.18)] bg-[rgb(255_255_255_/_0.7)] px-4 py-4 shadow-sm shadow-rose-100/40 backdrop-blur sm:px-6 dark:border-slate-800 dark:bg-slate-950/60">
        <p class="m-0 mb-2 text-[0.78rem] font-bold uppercase tracking-[0.16em] text-rose-600">
          {{ 'nav.gdpr' | transloco }}
        </p>
        <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
          <div class="space-y-2">
            <h1 class="m-0 text-2xl font-semibold tracking-tight text-slate-950 dark:text-slate-50 sm:text-3xl">
              {{ 'gdpr.title' | transloco }}
            </h1>
            <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
              {{ 'gdpr.subtitle' | transloco }}
            </p>
          </div>
          @if (activeTab(); as tab) {
            <div class="hidden items-center gap-2 rounded-full bg-rose-50 px-3 py-2 text-sm font-medium text-rose-700 dark:bg-rose-950/40 dark:text-rose-200 lg:flex">
              <mat-icon class="!text-[1.1rem]">{{ tab.icon }}</mat-icon>
              <span>{{ tab.labelKey | transloco }}</span>
            </div>
          }
        </div>
      </header>

      <nav mat-tab-nav-bar [tabPanel]="panel" class="overflow-x-auto rounded-[1rem] border border-[rgb(148_163_184_/_0.18)] bg-[rgb(255_255_255_/_0.7)] px-2 dark:border-slate-800 dark:bg-slate-950/60">
        @for (tab of visibleTabs(); track tab.route) {
          <a
            mat-tab-link
            [routerLink]="tab.route"
            routerLinkActive
            #rla="routerLinkActive"
            [active]="rla.isActive"
          >
            <mat-icon>{{ tab.icon }}</mat-icon>
            <span>{{ tab.labelKey | transloco }}</span>
          </a>
        }
      </nav>

      <mat-tab-nav-panel #panel>
        <section class="min-w-0">
          <router-outlet />
        </section>
      </mat-tab-nav-panel>
    </section>
  `,
  styles: [`
    :host ::ng-deep .mdc-tab__text-label {
      display: inline-flex;
      align-items: center;
      gap: 0.45rem;
      font-weight: 600;
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GdprShellComponent {
  private readonly authz = inject(AuthzService);
  private readonly router = inject(Router);

  private readonly tabs: WorkspaceTab[] = [
    {
      icon: 'dashboard',
      labelKey: 'nav.dashboard',
      route: '/gdpr/dashboard',
      permission: ['gdpr.read', 'gdpr.policies.read', 'gdpr.requests.read'],
      permissionMode: 'all',
      module: 'gdpr',
    },
    {
      icon: 'outbox',
      labelKey: 'nav.gdprExports',
      route: '/gdpr/exports',
      permission: 'gdpr.exports.read',
      module: 'gdpr',
    },
    {
      icon: 'visibility_lock',
      labelKey: 'nav.gdprPublication',
      route: '/gdpr/publication-reviews',
      permission: 'gdpr.publication.read',
      module: 'gdpr',
    },
  ];

  protected readonly visibleTabs = computed(() =>
    this.tabs.filter((tab) => {
      const permissionOk = Array.isArray(tab.permission)
        ? (tab.permissionMode === 'all'
            ? tab.permission.every((value) => this.authz.hasPermission(value))
            : this.authz.hasAnyPermission(tab.permission))
        : this.authz.hasPermission(tab.permission);
      return permissionOk && this.authz.hasModule(tab.module);
    }),
  );

  protected readonly activeTab = computed(() => {
    const currentUrl = this.router.url;
    return (
      this.visibleTabs()
        .slice()
        .sort((left, right) => right.route.length - left.route.length)
        .find((tab) => currentUrl.startsWith(tab.route)) ?? this.visibleTabs()[0] ?? null
    );
  });
}
