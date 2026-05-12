import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { filter, map, startWith } from 'rxjs/operators';

import { MatIconModule } from '@angular/material/icon';

import { AuthzService } from '../../core/authz/authz.service';

interface WorkspaceTab {
  icon: string;
  labelKey: string;
  route: string;
  permission: string | string[];
  group: WorkspaceGroupKey;
}

type WorkspaceGroupKey = 'overview' | 'people' | 'structure' | 'configuration' | 'platform';

interface WorkspaceTabGroup {
  key: WorkspaceGroupKey;
  icon: string;
  labelKey: string;
  descriptionKey: string;
  tabs: WorkspaceTab[];
}

@Component({
  selector: 'app-admin-shell',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive, TranslocoPipe, MatIconModule],
  templateUrl: './admin-shell.component.html',
  styleUrl: './admin-shell.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminShellComponent {
  private readonly authz = inject(AuthzService);
  private readonly router = inject(Router);
  private readonly currentUrl = toSignal(
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
      startWith(null),
      map(() => this.router.url),
    ),
    { initialValue: this.router.url },
  );

  private readonly tabs: WorkspaceTab[] = [
    {
      icon: 'dashboard',
      labelKey: 'nav.dashboard',
      route: '/admin/dashboard',
      permission: 'admin.read',
      group: 'overview',
    },
    {
      icon: 'group',
      labelKey: 'adminWorkspace.tabs.users',
      route: '/admin/users',
      permission: 'admin.users.read',
      group: 'people',
    },
    {
      icon: 'security',
      labelKey: 'adminWorkspace.tabs.roles',
      route: '/admin/roles',
      permission: 'admin.roles.read',
      group: 'people',
    },
    {
      icon: 'badge',
      labelKey: 'adminWorkspace.tabs.memberships',
      route: '/admin/memberships',
      permission: 'admin.memberships.read',
      group: 'people',
    },
    {
      icon: 'apartment',
      labelKey: 'adminWorkspace.tabs.structure',
      route: '/admin/org-units',
      permission: 'admin.org_units.read',
      group: 'structure',
    },
    {
      icon: 'work',
      labelKey: 'adminWorkspace.tabs.positions',
      route: '/admin/positions',
      permission: 'admin.positions.read',
      group: 'structure',
    },
    {
      icon: 'lock',
      labelKey: 'adminWorkspace.tabs.permissions',
      route: '/admin/permissions',
      permission: 'admin.permissions.read',
      group: 'structure',
    },
    {
      icon: 'assignment',
      labelKey: 'adminWorkspace.tabs.workflows',
      route: '/admin/workflow-definitions',
      permission: 'admin.workflow_definitions.read',
      group: 'configuration',
    },
    {
      icon: 'task_alt',
      labelKey: 'admin.quickLinks.dossierRequirements',
      route: '/admin/dossier-requirements',
      permission: 'admin.dossier_requirements.read',
      group: 'configuration',
    },
    {
      icon: 'schema',
      labelKey: 'adminWorkspace.tabs.taxonomies',
      route: '/admin/education-taxonomies',
      permission: 'admin.education_taxonomies.read',
      group: 'configuration',
    },
    {
      icon: 'list_alt',
      labelKey: 'adminWorkspace.tabs.nomenclatures',
      route: '/admin/nomenclatures',
      permission: 'admin.nomenclatures.read',
      group: 'configuration',
    },
    {
      icon: 'settings_applications',
      labelKey: 'adminWorkspace.tabs.platform',
      route: '/admin/platform-settings',
      permission: ['admin.auth_methods.read', 'admin.modules.read'],
      group: 'platform',
    },
    {
      icon: 'fingerprint',
      labelKey: 'adminWorkspace.tabs.identity',
      route: '/admin/identity',
      permission: 'admin.identity.read',
      group: 'platform',
    },
    {
      icon: 'policy',
      labelKey: 'adminWorkspace.tabs.gdpr',
      route: '/admin/gdpr-settings',
      permission: 'admin.gdpr_settings.read',
      group: 'platform',
    },
    {
      icon: 'history',
      labelKey: 'adminWorkspace.tabs.audit',
      route: '/admin/audit',
      permission: 'admin.audit.read',
      group: 'platform',
    },
  ];

  protected readonly visibleTabs = computed(() =>
    this.tabs.filter((tab) => {
      if (!this.authz.hasModule('admin')) {
        return false;
      }
      return Array.isArray(tab.permission)
        ? tab.permission.every((permission) => this.authz.hasPermission(permission))
        : this.authz.hasPermission(tab.permission);
    }),
  );

  protected readonly groupedTabs = computed<WorkspaceTabGroup[]>(() => {
    const visibleTabs = this.visibleTabs();
    const createGroup = (
      key: WorkspaceGroupKey,
      icon: string,
      labelKey: string,
      descriptionKey: string,
    ): WorkspaceTabGroup | null => {
      const tabs = visibleTabs.filter((tab) => tab.group === key);
      return tabs.length ? { key, icon, labelKey, descriptionKey, tabs } : null;
    };

    return [
      createGroup('overview', 'dashboard', 'nav.dashboard', 'admin.subtitle'),
      createGroup('people', 'group', 'admin.sections.users', 'admin.subtitle'),
      createGroup('structure', 'apartment', 'admin.sections.org_units', 'admin.subtitle'),
      createGroup(
        'configuration',
        'settings_suggest',
        'admin.sections.workflow_definitions',
        'admin.comingSoon',
      ),
      createGroup('platform', 'settings_applications', 'admin.sections.auth_methods', 'admin.subtitle'),
    ].filter((group): group is WorkspaceTabGroup => group !== null);
  });

  protected readonly activeTab = computed(() => {
    const currentUrl = this.currentUrl();
    return (
      this.visibleTabs()
        .slice()
        .sort((left, right) => right.route.length - left.route.length)
        .find((tab) => currentUrl.startsWith(tab.route)) ?? this.visibleTabs()[0]
    );
  });

  protected readonly activeGroup = computed(() => {
    const activeTab = this.activeTab();
    if (!activeTab) {
      return (
        this.groupedTabs()[0] ?? {
          key: 'overview' as WorkspaceGroupKey,
          icon: 'dashboard',
          labelKey: 'adminWorkspace.title',
          descriptionKey: 'adminWorkspace.subtitle',
          tabs: [],
        }
      );
    }
    return (
      this.groupedTabs().find((group) => group.key === activeTab.group) ?? {
        key: 'overview' as WorkspaceGroupKey,
        icon: 'dashboard',
        labelKey: 'adminWorkspace.title',
        descriptionKey: 'adminWorkspace.subtitle',
        tabs: [],
      }
    );
  });

  protected readonly groupNavItems = computed(() =>
    this.groupedTabs().map((group) => ({
      ...group,
      route: group.tabs[0]?.route ?? '/admin/dashboard',
      active: this.activeGroup().key === group.key,
    })),
  );

  protected readonly secondaryTabs = computed(() => this.activeGroup().tabs);
}
