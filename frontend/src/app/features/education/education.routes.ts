import { Routes } from '@angular/router';

import { permissionGuard } from '../../core/authz/authz.guard';

export const educationRoutes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./education-shell.component').then((m) => m.EducationShellComponent),
    children: [
      {
        path: 'dashboard',
        canActivate: [permissionGuard],
        data: {
          permissions: [
            'education.governance.read',
            'education.personnel.read',
            'education.decisions.read',
            'education.regulations.read',
            'education.managerial.read',
            'education.evaluations.read',
            'education.declarations.read',
            'education.mobility.read',
            'education.gradatii.read',
            'education.portfolios.read',
          ],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./education-dashboard-page.component').then(
            (m) => m.EducationDashboardPageComponent,
          ),
      },
      {
        path: 'governance',
        canActivate: [permissionGuard],
        data: { permission: 'education.governance.read', module: 'education' },
        loadComponent: () =>
          import('./education-page.component').then((m) => m.EducationPageComponent),
      },
      {
        path: 'personnel',
        canActivate: [permissionGuard],
        data: { permission: 'education.personnel.read', module: 'education' },
        loadComponent: () =>
          import('./personnel-page.component').then((m) => m.PersonnelPageComponent),
      },
      {
        path: 'decisions',
        canActivate: [permissionGuard],
        data: { permission: 'education.decisions.read', module: 'education' },
        loadComponent: () =>
          import('./decisions-page.component').then((m) => m.DecisionsPageComponent),
      },
      {
        path: 'managerial',
        canActivate: [permissionGuard],
        data: { permission: 'education.managerial.read', module: 'education' },
        loadComponent: () =>
          import('./managerial-page.component').then((m) => m.ManagerialPageComponent),
      },
      {
        path: 'regulations',
        canActivate: [permissionGuard],
        data: { permission: 'education.regulations.read', module: 'education' },
        loadComponent: () =>
          import('./regulations-page.component').then((m) => m.RegulationsPageComponent),
      },
      {
        path: 'mobility',
        canActivate: [permissionGuard],
        data: { permission: 'education.mobility.read', module: 'education' },
        loadComponent: () =>
          import('./mobility-page.component').then((m) => m.MobilityPageComponent),
      },
      {
        path: 'gradatii',
        canActivate: [permissionGuard],
        data: { permission: 'education.gradatii.read', module: 'education' },
        loadComponent: () =>
          import('./gradatii-page.component').then((m) => m.GradatiiPageComponent),
      },
      {
        path: 'portfolios',
        canActivate: [permissionGuard],
        data: { permission: 'education.portfolios.read', module: 'education' },
        loadComponent: () =>
          import('./portfolio-page.component').then((m) => m.PortfolioPageComponent),
      },
      {
        path: 'evaluations',
        canActivate: [permissionGuard],
        data: { permission: 'education.evaluations.read', module: 'education' },
        loadComponent: () =>
          import('./evaluations-page.component').then((m) => m.EvaluationsPageComponent),
      },
      {
        path: 'declarations',
        canActivate: [permissionGuard],
        data: { permission: 'education.declarations.read', module: 'education' },
        loadComponent: () =>
          import('./declarations-page.component').then((m) => m.DeclarationsPageComponent),
      },
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
    ],
  },
];
