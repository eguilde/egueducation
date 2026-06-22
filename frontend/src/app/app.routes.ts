import { Routes } from '@angular/router';

import { permissionGuard } from './core/authz/authz.guard';
import { FEATURE_ACCESS_RULES } from './core/authz/role-catalog';

const featureAccess = (feature: string) => FEATURE_ACCESS_RULES.find((rule) => rule.feature === feature);

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () =>
      import('./features/home/home-page.component').then((m) => m.HomePageComponent),
  },
  {
    path: 'login',
    redirectTo: '',
    pathMatch: 'full',
  },
  {
    path: 'auth/login',
    loadComponent: () =>
      import('./features/prime-auth/login-page.component').then((m) => m.LoginPageComponent),
  },
  {
    path: 'auth/register',
    loadComponent: () =>
      import('./features/prime-auth/register-page.component').then((m) => m.RegisterPageComponent),
  },
  {
    path: 'register',
    redirectTo: 'auth/register',
    pathMatch: 'full',
  },
  {
    path: 'auth/start',
    loadComponent: () =>
      import('./features/auth/auth-start-page.component').then((m) => m.AuthStartPageComponent),
  },
  {
    path: 'callback',
    redirectTo: 'auth/callback',
    pathMatch: 'full',
  },
  {
    path: 'auth/callback',
    loadComponent: () =>
      import('./features/auth/callback-page.component').then((m) => m.CallbackPageComponent),
  },
  {
    path: 'auth/consent',
    loadComponent: () =>
      import('./features/prime-auth/consent-page.component').then((m) => m.ConsentPageComponent),
  },
  {
    path: 'auth/logout',
    loadComponent: () =>
      import('./features/prime-auth/logout-page.component').then((m) => m.LogoutPageComponent),
  },
  {
    path: 'auth/access-denied',
    loadComponent: () =>
      import('./features/prime-auth/auth-status-page.component').then((m) => m.AuthStatusPageComponent),
  },
  {
    path: 'auth/logged-out',
    loadComponent: () =>
      import('./features/prime-auth/auth-status-page.component').then((m) => m.AuthStatusPageComponent),
  },
  {
    path: '',
    loadComponent: () =>
      import('./layout/app-shell.component').then((m) => m.AppShellComponent),
    children: [
      {
        path: 'dashboard',
        redirectTo: 'documente',
        pathMatch: 'full',
      },
      {
        path: 'documente',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('documente')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('documente')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['registratura', 'workflow', 'earchiva'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/documente-workspace.component').then((m) => m.DocumenteWorkspaceComponent),
      },
      {
        path: 'documente/new',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('documente_create_edit')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('documente_create_edit')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['registratura'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/document-form-workspace.component').then((m) => m.DocumentFormWorkspaceComponent),
      },
      {
        path: 'documente/:id/edit',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('documente_create_edit')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('documente_create_edit')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['registratura'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/document-form-workspace.component').then((m) => m.DocumentFormWorkspaceComponent),
      },
      {
        path: 'documente/:id',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('documente')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('documente')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['registratura'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/document-detail-workspace.component').then((m) => m.DocumentDetailWorkspaceComponent),
      },
      {
        path: 'registre',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('registre')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('registre')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['registratura'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/registre-workspace.component').then((m) => m.RegistreWorkspaceComponent),
      },
      {
        path: 'workflow',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('workflow')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('workflow')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['workflow'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/workflow-workspace.component').then((m) => m.WorkflowWorkspaceComponent),
      },
      {
        path: 'earchiva',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('earchiva')?.roles ?? [],
          rolesMode: 'any',
          permissions: featureAccess('earchiva')?.permissions ?? [],
          permissionsMode: 'any',
          modules: ['earchiva'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/earchiva-workspace.component').then((m) => m.EarchivaWorkspaceComponent),
      },
      {
        path: 'profile',
        loadComponent: () =>
          import('./features/profile/profile-page.component').then((m) => m.ProfilePageComponent),
      },
      {
        path: 'education',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('education')?.roles ?? [],
          rolesMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/education-workspace.component').then((m) => m.EducationWorkspaceComponent),
      },
      {
        path: 'admin',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('admin')?.roles ?? [],
          rolesMode: 'any',
          permission: featureAccess('admin')?.permissions?.[0] ?? 'admin.read',
          module: 'admin',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/admin-workspace.component').then((m) => m.AdminWorkspaceComponent),
      },
      {
        path: 'gdpr',
        canActivate: [permissionGuard],
        data: {
          roles: featureAccess('gdpr')?.roles ?? [],
          rolesMode: 'any',
          module: 'gdpr',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/gdpr-workspace.component').then((m) => m.GdprWorkspaceComponent),
      },
      { path: 'registratura', redirectTo: 'documente', pathMatch: 'full' },
      { path: '', redirectTo: 'documente', pathMatch: 'full' },
    ],
  },
  { path: '**', redirectTo: 'dashboard' },
];
