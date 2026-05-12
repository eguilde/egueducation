import { Routes } from '@angular/router';

import { permissionGuard } from '../../core/authz/authz.guard';

export const adminRoutes: Routes = [
  {
    path: '',
    loadComponent: () => import('./admin-shell.component').then((m) => m.AdminShellComponent),
    children: [
      {
        path: 'dashboard',
        canActivate: [permissionGuard],
        data: { permission: 'admin.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-dashboard-page.component').then((m) => m.AdminDashboardPageComponent),
      },
      {
        path: 'users',
        canActivate: [permissionGuard],
        data: { permission: 'admin.users.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-users-page.component').then((m) => m.AdminUsersPageComponent),
      },
      {
        path: 'roles',
        canActivate: [permissionGuard],
        data: { permission: 'admin.roles.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-roles-page.component').then((m) => m.AdminRolesPageComponent),
      },
      {
        path: 'org-units',
        canActivate: [permissionGuard],
        data: { permission: 'admin.org_units.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-org-units-page.component').then((m) => m.AdminOrgUnitsPageComponent),
      },
      {
        path: 'memberships',
        canActivate: [permissionGuard],
        data: { permission: 'admin.memberships.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-memberships-page.component').then(
            (m) => m.AdminMembershipsPageComponent,
          ),
      },
      {
        path: 'positions',
        canActivate: [permissionGuard],
        data: { permission: 'admin.positions.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-positions-page.component').then((m) => m.AdminPositionsPageComponent),
      },
      {
        path: 'permissions',
        canActivate: [permissionGuard],
        data: { permission: 'admin.permissions.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-permissions-page.component').then(
            (m) => m.AdminPermissionsPageComponent,
          ),
      },
      {
        path: 'dossier-requirements',
        canActivate: [permissionGuard],
        data: { permission: 'admin.dossier_requirements.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-dossier-requirements-page.component').then(
            (m) => m.AdminDossierRequirementsPageComponent,
          ),
      },
      {
        path: 'education-taxonomies',
        canActivate: [permissionGuard],
        data: { permission: 'admin.education_taxonomies.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-education-taxonomies-page.component').then(
            (m) => m.AdminEducationTaxonomiesPageComponent,
          ),
      },
      {
        path: 'workflow-definitions',
        canActivate: [permissionGuard],
        data: { permission: 'admin.workflow_definitions.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-workflow-definitions-page.component').then(
            (m) => m.AdminWorkflowDefinitionsPageComponent,
          ),
      },
      {
        path: 'nomenclatures',
        canActivate: [permissionGuard],
        data: { permission: 'admin.nomenclatures.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-nomenclatures-page.component').then(
            (m) => m.AdminNomenclaturesPageComponent,
          ),
      },
      {
        path: 'platform-settings',
        canActivate: [permissionGuard],
        data: {
          permissions: ['admin.auth_methods.read', 'admin.modules.read'],
          module: 'admin',
        },
        loadComponent: () =>
          import('./admin-platform-settings-page.component').then(
            (m) => m.AdminPlatformSettingsPageComponent,
          ),
      },
      {
        path: 'identity',
        canActivate: [permissionGuard],
        data: { permission: 'admin.identity.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-identity-page.component').then((m) => m.AdminIdentityPageComponent),
      },
      {
        path: 'audit',
        canActivate: [permissionGuard],
        data: { permission: 'admin.audit.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-audit-page.component').then((m) => m.AdminAuditPageComponent),
      },
      {
        path: 'gdpr-settings',
        canActivate: [permissionGuard],
        data: { permission: 'admin.gdpr_settings.read', module: 'admin' },
        loadComponent: () =>
          import('./admin-gdpr-settings-page.component').then(
            (m) => m.AdminGdprSettingsPageComponent,
          ),
      },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
];
