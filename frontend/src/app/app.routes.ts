import { Routes } from '@angular/router';
import { permissionGuard } from './core/authz/authz.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('./features/auth/login-page.component').then((m) => m.LoginPageComponent),
  },
  {
    path: 'auth/callback',
    loadComponent: () =>
      import('./features/auth/callback-page.component').then((m) => m.CallbackPageComponent),
  },
  {
    path: 'auth/consent',
    loadComponent: () =>
      import('./features/auth/consent-page.component').then((m) => m.ConsentPageComponent),
  },
  {
    path: 'auth/logout',
    loadComponent: () =>
      import('./features/auth/logout-page.component').then((m) => m.LogoutPageComponent),
  },
  {
    path: 'auth/access-denied',
    loadComponent: () =>
      import('./features/auth/access-denied-page.component').then(
        (m) => m.AccessDeniedPageComponent,
      ),
  },
  {
    path: 'auth/logged-out',
    loadComponent: () =>
      import('./features/auth/logged-out-page.component').then((m) => m.LoggedOutPageComponent),
  },
  {
    path: '',
    loadComponent: () =>
      import('./layout/app-shell.component').then((m) => m.AppShellComponent),
    children: [
      {
        path: 'dashboard',
        canActivate: [permissionGuard],
        data: { permission: 'dashboard.read', module: 'dashboard' },
        loadComponent: () =>
          import('./features/dashboard/dashboard-page.component').then((m) => m.DashboardPageComponent),
      },
      {
        path: 'admin',
        canActivate: [permissionGuard],
        data: { permission: 'admin.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-dashboard-page.component').then(
            (m) => m.AdminDashboardPageComponent,
          ),
      },
      {
        path: 'admin/users',
        canActivate: [permissionGuard],
        data: { permission: 'admin.users.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-users-page.component').then((m) => m.AdminUsersPageComponent),
      },
      {
        path: 'admin/roles',
        canActivate: [permissionGuard],
        data: { permission: 'admin.roles.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-roles-page.component').then((m) => m.AdminRolesPageComponent),
      },
      {
        path: 'admin/org-units',
        canActivate: [permissionGuard],
        data: { permission: 'admin.org_units.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-org-units-page.component').then(
            (m) => m.AdminOrgUnitsPageComponent,
          ),
      },
      {
        path: 'admin/memberships',
        canActivate: [permissionGuard],
        data: { permission: 'admin.memberships.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-memberships-page.component').then(
            (m) => m.AdminMembershipsPageComponent,
          ),
      },
      {
        path: 'admin/positions',
        canActivate: [permissionGuard],
        data: { permission: 'admin.positions.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-positions-page.component').then(
            (m) => m.AdminPositionsPageComponent,
          ),
      },
      {
        path: 'admin/permissions',
        canActivate: [permissionGuard],
        data: { permission: 'admin.permissions.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-permissions-page.component').then(
            (m) => m.AdminPermissionsPageComponent,
          ),
      },
      {
        path: 'admin/dossier-requirements',
        canActivate: [permissionGuard],
        data: { permission: 'admin.dossier_requirements.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-dossier-requirements-page.component').then(
            (m) => m.AdminDossierRequirementsPageComponent,
          ),
      },
      {
        path: 'admin/education-taxonomies',
        canActivate: [permissionGuard],
        data: { permission: 'admin.education_taxonomies.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-education-taxonomies-page.component').then(
            (m) => m.AdminEducationTaxonomiesPageComponent,
          ),
      },
      {
        path: 'admin/workflow-definitions',
        canActivate: [permissionGuard],
        data: { permission: 'admin.workflow_definitions.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-workflow-definitions-page.component').then(
            (m) => m.AdminWorkflowDefinitionsPageComponent,
          ),
      },
      {
        path: 'admin/nomenclatures',
        canActivate: [permissionGuard],
        data: { permission: 'admin.nomenclatures.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-nomenclatures-page.component').then(
            (m) => m.AdminNomenclaturesPageComponent,
          ),
      },
      {
        path: 'admin/platform-settings',
        canActivate: [permissionGuard],
        data: { permission: 'admin.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-platform-settings-page.component').then(
            (m) => m.AdminPlatformSettingsPageComponent,
          ),
      },
      {
        path: 'admin/identity',
        canActivate: [permissionGuard],
        data: { permission: 'admin.identity.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-identity-page.component').then(
            (m) => m.AdminIdentityPageComponent,
          ),
      },
      {
        path: 'admin/audit',
        canActivate: [permissionGuard],
        data: { permission: 'admin.audit.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-audit-page.component').then(
            (m) => m.AdminAuditPageComponent,
          ),
      },
      {
        path: 'admin/gdpr-settings',
        canActivate: [permissionGuard],
        data: { permission: 'admin.gdpr_settings.read', module: 'admin' },
        loadComponent: () =>
          import('./features/admin/admin-gdpr-settings-page.component').then(
            (m) => m.AdminGdprSettingsPageComponent,
          ),
      },
      {
        path: 'registratura',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/registratura/registratura-page.component').then((m) => m.RegistraturaPageComponent),
        data: { permission: 'registratura.read', module: 'registratura' },
      },
      {
        path: 'workflow',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/workflow/workflow-page.component').then((m) => m.WorkflowPageComponent),
        data: { permission: 'workflow.read', module: 'workflow' },
      },
      {
        path: 'earchiva',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/earchiva/earchiva-page.component').then((m) => m.EarchivaPageComponent),
        data: { permission: 'earchiva.read', module: 'earchiva' },
      },
      {
        path: 'education',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/education-page.component').then((m) => m.EducationPageComponent),
        data: { permission: 'education.read', module: 'education' },
      },
      {
        path: 'education/personnel',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/personnel-page.component').then((m) => m.PersonnelPageComponent),
        data: { permission: 'education.personnel.read', module: 'education' },
      },
      {
        path: 'education/decisions',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/decisions-page.component').then((m) => m.DecisionsPageComponent),
        data: { permission: 'education.decisions.read', module: 'education' },
      },
      {
        path: 'education/managerial',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/managerial-page.component').then((m) => m.ManagerialPageComponent),
        data: { permission: 'education.managerial.read', module: 'education' },
      },
      {
        path: 'education/regulations',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/regulations-page.component').then((m) => m.RegulationsPageComponent),
        data: { permission: 'education.regulations.read', module: 'education' },
      },
      {
        path: 'education/mobility',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/mobility-page.component').then((m) => m.MobilityPageComponent),
        data: { permission: 'education.mobility.read', module: 'education' },
      },
      {
        path: 'education/gradatii',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/gradatii-page.component').then((m) => m.GradatiiPageComponent),
        data: { permission: 'education.gradatii.read', module: 'education' },
      },
      {
        path: 'education/portfolios',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/portfolio-page.component').then((m) => m.PortfolioPageComponent),
        data: { permission: 'education.portfolios.read', module: 'education' },
      },
      {
        path: 'education/evaluations',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/evaluations-page.component').then((m) => m.EvaluationsPageComponent),
        data: { permission: 'education.evaluations.read', module: 'education' },
      },
      {
        path: 'education/declarations',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/education/declarations-page.component').then((m) => m.DeclarationsPageComponent),
        data: { permission: 'education.declarations.read', module: 'education' },
      },
      {
        path: 'gdpr',
        canActivate: [permissionGuard],
        loadComponent: () => import('./features/gdpr/gdpr-page.component').then((m) => m.GdprPageComponent),
        data: { permission: 'gdpr.read', module: 'gdpr' },
      },
      {
        path: 'gdpr/exports',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/gdpr/gdpr-exports-page.component').then((m) => m.GdprExportsPageComponent),
        data: { permission: 'gdpr.exports.read', module: 'gdpr' },
      },
      {
        path: 'gdpr/publication-reviews',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/gdpr/gdpr-publication-page.component').then((m) => m.GdprPublicationPageComponent),
        data: { permission: 'gdpr.publication.read', module: 'gdpr' },
      },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
  { path: '**', redirectTo: 'dashboard' },
];
