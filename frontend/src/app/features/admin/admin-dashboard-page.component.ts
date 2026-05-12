import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';

import { AdminApiService } from '../../core/api/admin-api.service';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';

interface AdminActionLink {
  labelKey: string;
  route: string;
  permission: string | string[];
}

interface AdminControlGroup {
  icon: string;
  titleKey: string;
  bodyKey: string;
  actions: AdminActionLink[];
}

@Component({
  selector: 'app-admin-dashboard-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, MatButtonModule, MatCardModule, MatChipsModule, MatIconModule, HasPermissionDirective],
  templateUrl: './admin-dashboard-page.component.html',
  styleUrl: './admin-dashboard-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminDashboardPageComponent {
  private readonly adminApi = inject(AdminApiService);

  protected readonly dashboard = toSignal(this.adminApi.dashboard(), {
    initialValue: {
      stats: {
        users: 0,
        memberships: 0,
        positions: 0,
        permissions: 0,
        workflows: 0,
        archives: 0,
        ready_dossiers: 0,
        blocked_dossiers: 0,
      },
      modules: [],
      admin_sections: [],
      warnings: [],
    },
  });

  protected readonly statCards = computed(() => [
    { icon: 'group', value: String(this.dashboard().stats.users), titleKey: 'admin.stats.users' },
    { icon: 'badge', value: String(this.dashboard().stats.positions), titleKey: 'admin.stats.positions' },
    { icon: 'account_tree', value: String(this.dashboard().stats.workflows), titleKey: 'admin.stats.workflows' },
    { icon: 'verified_user', value: String(this.dashboard().stats.permissions), titleKey: 'admin.stats.permissions' },
    { icon: 'task_alt', value: String(this.dashboard().stats.ready_dossiers), titleKey: 'admin.stats.readyDossiers' },
    { icon: 'warning', value: String(this.dashboard().stats.blocked_dossiers), titleKey: 'admin.stats.blockedDossiers' },
  ]);

  protected readonly controlGroups = computed<AdminControlGroup[]>(() => [
    {
      icon: 'group',
      titleKey: 'admin.sections.users',
      bodyKey: 'admin.quickLinks.title',
      actions: [
        { labelKey: 'admin.quickLinks.users', route: '/admin/users', permission: 'admin.users.read' },
        { labelKey: 'admin.quickLinks.roles', route: '/admin/roles', permission: 'admin.roles.read' },
        {
          labelKey: 'admin.quickLinks.memberships',
          route: '/admin/memberships',
          permission: 'admin.memberships.read',
        },
      ],
    },
    {
      icon: 'apartment',
      titleKey: 'admin.sections.org_units',
      bodyKey: 'admin.subtitle',
      actions: [
        {
          labelKey: 'admin.quickLinks.orgUnits',
          route: '/admin/org-units',
          permission: 'admin.org_units.read',
        },
        {
          labelKey: 'admin.quickLinks.positions',
          route: '/admin/positions',
          permission: 'admin.positions.read',
        },
        {
          labelKey: 'admin.quickLinks.permissions',
          route: '/admin/permissions',
          permission: 'admin.permissions.read',
        },
      ],
    },
    {
      icon: 'settings_suggest',
      titleKey: 'admin.sections.workflow_definitions',
      bodyKey: 'admin.comingSoon',
      actions: [
        {
          labelKey: 'admin.quickLinks.workflowDefinitions',
          route: '/admin/workflow-definitions',
          permission: 'admin.workflow_definitions.read',
        },
        {
          labelKey: 'admin.quickLinks.dossierRequirements',
          route: '/admin/dossier-requirements',
          permission: 'admin.dossier_requirements.read',
        },
        {
          labelKey: 'admin.quickLinks.educationTaxonomies',
          route: '/admin/education-taxonomies',
          permission: 'admin.education_taxonomies.read',
        },
        {
          labelKey: 'admin.quickLinks.nomenclatures',
          route: '/admin/nomenclatures',
          permission: 'admin.nomenclatures.read',
        },
      ],
    },
    {
      icon: 'fingerprint',
      titleKey: 'admin.sections.auth_methods',
      bodyKey: 'admin.subtitle',
      actions: [
        {
          labelKey: 'admin.quickLinks.platformSettings',
          route: '/admin/platform-settings',
          permission: ['admin.auth_methods.read', 'admin.modules.read'],
        },
        {
          labelKey: 'admin.quickLinks.identity',
          route: '/admin/identity',
          permission: 'admin.identity.read',
        },
      ],
    },
    {
      icon: 'policy',
      titleKey: 'admin.sections.gdpr',
      bodyKey: 'admin.subtitle',
      actions: [
        {
          labelKey: 'admin.quickLinks.gdprSettings',
          route: '/admin/gdpr-settings',
          permission: 'admin.gdpr_settings.read',
        },
        {
          labelKey: 'admin.quickLinks.audit',
          route: '/admin/audit',
          permission: 'admin.audit.read',
        },
      ],
    },
  ]);

  protected readonly sectionKeys = computed(() =>
    this.dashboard().admin_sections.map((section) => `admin.sections.${section}`),
  );

  protected readonly activeModules = computed(() =>
    this.dashboard().modules.filter((module) => module.active),
  );
}
