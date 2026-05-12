import { Routes } from '@angular/router';

import { permissionGuard } from '../../core/authz/authz.guard';

export const gdprRoutes: Routes = [
  {
    path: '',
    loadComponent: () => import('./gdpr-shell.component').then((m) => m.GdprShellComponent),
    children: [
      {
        path: 'dashboard',
        canActivate: [permissionGuard],
        loadComponent: () => import('./gdpr-page.component').then((m) => m.GdprPageComponent),
        data: {
          permissions: ['gdpr.read', 'gdpr.policies.read', 'gdpr.requests.read'],
          permissionsMode: 'all',
          module: 'gdpr',
        },
      },
      {
        path: 'exports',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./gdpr-exports-page.component').then((m) => m.GdprExportsPageComponent),
        data: { permission: 'gdpr.exports.read', module: 'gdpr' },
      },
      {
        path: 'publication-reviews',
        canActivate: [permissionGuard],
        loadComponent: () =>
          import('./gdpr-publication-page.component').then((m) => m.GdprPublicationPageComponent),
        data: { permission: 'gdpr.publication.read', module: 'gdpr' },
      },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
];
