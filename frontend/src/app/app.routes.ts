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
          import('./features/dashboard/dashboard-page.component').then(
            (m) => m.DashboardPageComponent,
          ),
      },
      {
        path: 'documente',
        loadChildren: () =>
          import('./features/documente/documente.routes').then((m) => m.documenteRoutes),
      },
      {
        path: 'education',
        loadChildren: () =>
          import('./features/education/education.routes').then((m) => m.educationRoutes),
      },
      {
        path: 'admin',
        loadChildren: () =>
          import('./features/admin/admin.routes').then((m) => m.adminRoutes),
      },
      {
        path: 'gdpr',
        loadChildren: () =>
          import('./features/gdpr/gdpr.routes').then((m) => m.gdprRoutes),
      },
      { path: 'registratura', redirectTo: 'documente/registratura', pathMatch: 'full' },
      { path: 'workflow', redirectTo: 'documente/workflow', pathMatch: 'full' },
      { path: 'earchiva', redirectTo: 'documente/earchiva', pathMatch: 'full' },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
  { path: '**', redirectTo: 'dashboard' },
];
