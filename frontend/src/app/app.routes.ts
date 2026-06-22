import { Routes } from '@angular/router';

import { permissionGuard } from './core/authz/authz.guard';

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
          roles: ['admin', 'super_admin', 'director', 'secretar', 'registrator'],
          rolesMode: 'any',
          permissions: ['registratura.read', 'workflow.read', 'earchiva.read'],
          permissionsMode: 'any',
          modules: ['registratura', 'workflow', 'earchiva'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/documente-workspace.component').then((m) => m.DocumenteWorkspaceComponent),
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
          roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
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
          roles: ['admin', 'super_admin', 'director'],
          rolesMode: 'any',
          permission: 'admin.read',
          module: 'admin',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/admin-workspace.component').then((m) => m.AdminWorkspaceComponent),
      },
      {
        path: 'gdpr',
        canActivate: [permissionGuard],
        data: {
          roles: ['admin', 'super_admin', 'director', 'gdpr_officer'],
          rolesMode: 'any',
          module: 'gdpr',
        },
        loadComponent: () =>
          import('./features/prime-workspaces/gdpr-workspace.component').then((m) => m.GdprWorkspaceComponent),
      },
      { path: 'registratura', redirectTo: 'documente', pathMatch: 'full' },
      { path: 'workflow', redirectTo: 'documente', pathMatch: 'full' },
      { path: 'earchiva', redirectTo: 'documente', pathMatch: 'full' },
      { path: '', redirectTo: 'documente', pathMatch: 'full' },
    ],
  },
  { path: '**', redirectTo: 'dashboard' },
];
