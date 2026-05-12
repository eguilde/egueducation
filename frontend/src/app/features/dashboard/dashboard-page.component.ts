import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';

import { AppApiService } from '../../core/api/app-api.service';
import { AuthzService } from '../../core/authz/authz.service';

interface DashboardCard {
  icon: string;
  titleKey: string;
  bodyKey: string;
  route: string;
  ctaKey: string;
  permissions?: string[];
  permissionsMode?: 'all' | 'any';
  modules?: string[];
  modulesMode?: 'all' | 'any';
}

@Component({
  selector: 'app-dashboard-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, MatButtonModule, MatCardModule, MatChipsModule, MatIconModule],
  templateUrl: './dashboard-page.component.html',
  styleUrl: './dashboard-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DashboardPageComponent {
  private readonly api = inject(AppApiService);
  protected readonly authz = inject(AuthzService);

  protected readonly meta = toSignal(this.api.meta(), {
    initialValue: {
      name: 'EguEducation',
      default_locale: 'ro',
      available_locales: ['ro', 'en'],
      theme: {
        family: 'material3-expressive',
        brand: 'red-rose',
      },
    },
  });

  protected readonly cards: DashboardCard[] = [
    {
      icon: 'folder_open',
      titleKey: 'dashboard.cards.documents.title',
      bodyKey: 'dashboard.cards.documents.body',
      route: '/documente/dashboard',
      ctaKey: 'dashboard.cards.documents.cta',
      permissions: ['registratura.read', 'workflow.read', 'earchiva.read'],
      permissionsMode: 'any' as const,
      modules: ['registratura', 'workflow', 'earchiva'],
      modulesMode: 'any' as const,
    },
    {
      icon: 'school',
      titleKey: 'dashboard.cards.education.title',
      bodyKey: 'dashboard.cards.education.body',
      route: '/education/dashboard',
      ctaKey: 'dashboard.cards.education.cta',
      permissions: [
        'education.governance.read',
        'education.personnel.read',
        'education.decisions.read',
        'education.regulations.read',
        'education.managerial.read',
        'education.evaluations.read',
        'education.declarations.read',
        'education.mobility.read',
        'education.gradatii.read',
        'education.portfolios.read',
      ],
      permissionsMode: 'any' as const,
      modules: ['education'],
      modulesMode: 'any' as const,
    },
    {
      icon: 'policy',
      titleKey: 'dashboard.cards.gdpr.title',
      bodyKey: 'dashboard.cards.gdpr.body',
      route: '/gdpr/dashboard',
      ctaKey: 'dashboard.cards.gdpr.cta',
      permissions: ['gdpr.read', 'gdpr.policies.read', 'gdpr.requests.read'],
      permissionsMode: 'all' as const,
      modules: ['gdpr'],
      modulesMode: 'any' as const,
    },
    {
      icon: 'admin_panel_settings',
      titleKey: 'dashboard.cards.admin.title',
      bodyKey: 'dashboard.cards.admin.body',
      route: '/admin/dashboard',
      ctaKey: 'dashboard.cards.admin.cta',
      permissions: ['admin.read'],
      permissionsMode: 'all' as const,
      modules: ['admin'],
      modulesMode: 'any' as const,
    },
  ];
  protected readonly visibleCards = computed(() =>
    this.cards.filter((card) => {
      const permissionOk =
        !card.permissions ||
        (card.permissionsMode === 'all'
          ? card.permissions.every((permission) => this.authz.hasPermission(permission))
          : card.permissions.some((permission) => this.authz.hasPermission(permission)));
      const moduleOk =
        !card.modules ||
        (card.modulesMode === 'all'
          ? card.modules.every((moduleCode) => this.authz.hasModule(moduleCode))
          : card.modules.some((moduleCode) => this.authz.hasModule(moduleCode)));
      return permissionOk && moduleOk;
    }),
  );

  protected readonly themeBadge = computed(
    () => `${this.meta().theme.family} / ${this.meta().theme.brand}`,
  );
}
