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
        path: 'registratura',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/placeholder/feature-placeholder-page.component').then(
            (m) => m.FeaturePlaceholderPageComponent,
          ),
        data: {
          titleKey: 'nav.registratura',
          descriptionKey: 'placeholder.registratura',
          permission: 'registratura.read',
          module: 'registratura',
        },
      },
      {
        path: 'workflow',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/placeholder/feature-placeholder-page.component').then(
            (m) => m.FeaturePlaceholderPageComponent,
          ),
        data: {
          titleKey: 'nav.workflow',
          descriptionKey: 'placeholder.workflow',
          permission: 'workflow.read',
          module: 'workflow',
        },
      },
      {
        path: 'earchiva',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/placeholder/feature-placeholder-page.component').then(
            (m) => m.FeaturePlaceholderPageComponent,
          ),
        data: {
          titleKey: 'nav.earchiva',
          descriptionKey: 'placeholder.earchiva',
          permission: 'earchiva.read',
          module: 'earchiva',
        },
      },
      {
        path: 'education',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./features/placeholder/feature-placeholder-page.component').then(
            (m) => m.FeaturePlaceholderPageComponent,
          ),
        data: {
          titleKey: 'nav.education',
          descriptionKey: 'placeholder.education',
          permission: 'education.read',
          module: 'education',
        },
      },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
  { path: '**', redirectTo: 'dashboard' },
];
