import { Routes } from '@angular/router';

import { permissionGuard } from '../../core/authz/authz.guard';

export const documenteRoutes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./documente-shell.component').then((m) => m.DocumenteShellComponent),
    children: [
      {
        path: 'dashboard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['registratura.read', 'workflow.read', 'earchiva.read'],
          permissionsMode: 'any',
          modules: ['registratura', 'workflow', 'earchiva'],
          modulesMode: 'any',
        },
        loadComponent: () =>
          import('./documente-dashboard-page.component').then(
            (m) => m.DocumenteDashboardPageComponent,
          ),
      },
      {
        path: 'registratura',
        canActivate: [permissionGuard],
        data: { permission: 'registratura.read', module: 'registratura' },
        loadComponent: () =>
          import('../registratura/registratura-page.component').then(
            (m) => m.RegistraturaPageComponent,
          ),
      },
      {
        path: 'workflow',
        canActivate: [permissionGuard],
        data: { permission: 'workflow.read', module: 'workflow' },
        loadComponent: () =>
          import('../workflow/workflow-page.component').then((m) => m.WorkflowPageComponent),
      },
      {
        path: 'earchiva',
        canActivate: [permissionGuard],
        data: { permission: 'earchiva.read', module: 'earchiva' },
        loadComponent: () =>
          import('../earchiva/earchiva-page.component').then((m) => m.EarchivaPageComponent),
      },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
];
