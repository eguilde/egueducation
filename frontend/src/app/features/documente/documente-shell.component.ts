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
  module: string | string[];
}

@Component({
  selector: 'app-documente-shell',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive, TranslocoPipe, MatIconModule, MatTabsModule],
  template: `
    <section class="workspace-shell space-y-4">
      <header class="workspace-shell__header rounded-3xl border border-[rgb(148_163_184_/_0.18)] bg-[rgb(255_255_255_/_0.7)] px-4 py-4 shadow-sm shadow-rose-100/40 backdrop-blur sm:px-6 dark:border-slate-800 dark:bg-slate-950/60">
        <p class="workspace-shell__eyebrow">{{ 'nav.documente' | transloco }}</p>
        <div class="flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between">
          <div class="space-y-2">
            <h1 class="m-0 text-2xl font-semibold tracking-tight text-slate-950 dark:text-slate-50 sm:text-3xl">
              {{ 'documenteWorkspace.title' | transloco }}
            </h1>
            <p class="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
              {{ 'documenteWorkspace.subtitle' | transloco }}
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

      <nav mat-tab-nav-bar [tabPanel]="panel" class="workspace-shell__tabs">
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
        <section class="workspace-shell__content">
          <router-outlet />
        </section>
      </mat-tab-nav-panel>
    </section>
  `,
  styles: `
    .workspace-shell__eyebrow {
      margin: 0 0 0.5rem;
      font-size: 0.78rem;
      font-weight: 700;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--brand-600);
    }

    .workspace-shell__tabs {
      overflow-x: auto;
      border-radius: 1rem;
      border: 1px solid rgb(148 163 184 / 0.18);
      background: rgb(255 255 255 / 0.7);
      padding-inline: 0.5rem;
    }

    .workspace-shell__tabs a {
      gap: 0.5rem;
    }

    .workspace-shell__content {
      min-width: 0;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DocumenteShellComponent {
  private readonly authz = inject(AuthzService);
  private readonly router = inject(Router);

  private readonly tabs: WorkspaceTab[] = [
    {
      icon: 'dashboard',
      labelKey: 'nav.dashboard',
      route: '/documente/dashboard',
      permission: ['registratura.read', 'workflow.read', 'earchiva.read'],
      module: ['registratura', 'workflow', 'earchiva'],
    },
    {
      icon: 'inbox',
      labelKey: 'nav.registratura',
      route: '/documente/registratura',
      permission: 'registratura.read',
      module: 'registratura',
    },
    {
      icon: 'account_tree',
      labelKey: 'nav.documentFlow',
      route: '/documente/workflow',
      permission: 'workflow.read',
      module: 'workflow',
    },
    {
      icon: 'inventory_2',
      labelKey: 'nav.earchiva',
      route: '/documente/earchiva',
      permission: 'earchiva.read',
      module: 'earchiva',
    },
  ];

  protected readonly visibleTabs = computed(() =>
    this.tabs.filter((tab) => {
      const permissionOk = Array.isArray(tab.permission)
        ? this.authz.hasAnyPermission(tab.permission)
        : this.authz.hasPermission(tab.permission);
      const moduleOk = Array.isArray(tab.module)
        ? this.authz.hasAnyModule(tab.module)
        : this.authz.hasModule(tab.module);
      return permissionOk && moduleOk;
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
