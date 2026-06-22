import { Routes } from '@angular/router';

import { permissionGuard } from '../../core/authz/authz.guard';
import { EDUCATION_ANY_READ_PERMISSIONS } from './shared/education-config';

export const educationRoutes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./education-shell.component').then((m) => m.EducationShellComponent),
    children: [
      {
        path: '',
        pathMatch: 'full',
        redirectTo: 'dashboard',
      },
      {
        path: 'dashboard',
        canActivate: [permissionGuard],
        data: {
          permissions: EDUCATION_ANY_READ_PERMISSIONS,
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./dashboard/education-dashboard.component').then((m) => m.EducationDashboardComponent),
      },
      {
        path: 'governance',
        canActivate: [permissionGuard],
        data: {
          permissions: [
            'education.governance.read',
            'education.decisions.read',
            'education.managerial.read',
            'education.regulations.read',
          ],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./governance/education-governance-page.component').then((m) => m.EducationGovernancePageComponent),
      },
      {
        path: 'personnel',
        canActivate: [permissionGuard],
        data: {
          permissions: [
            'education.personnel.read',
            'education.evaluations.read',
            'education.declarations.read',
            'education.mobility.read',
            'education.gradatii.read',
          ],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./personnel/education-personnel-page.component').then((m) => m.EducationPersonnelPageComponent),
      },
      {
        path: 'portfolio',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.portfolios.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./portfolio/education-portfolio-page.component').then((m) => m.EducationPortfolioPageComponent),
      },
      {
        path: 'compliance',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./compliance/education-compliance-page.component').then((m) => m.EducationCompliancePageComponent),
      },
    ],
  },
];
