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
        path: 'dashboard/director',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.read', 'education.governance.read', 'education.managerial.read', 'education.portfolios.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./dashboard/education-director-dashboard.component').then((m) => m.EducationDirectorDashboardComponent),
      },
      {
        path: 'dashboard/director/reports',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.read', 'education.governance.read', 'education.managerial.read', 'education.portfolios.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./dashboard/education-standard-reports.component').then((m) => m.EducationStandardReportsComponent),
      },
      {
        path: 'dashboard/secretariat',
        canActivate: [permissionGuard],
        data: {
          permissions: EDUCATION_ANY_READ_PERMISSIONS,
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./dashboard/education-secretariat-dashboard.component').then((m) => m.EducationSecretariatDashboardComponent),
      },
      {
        path: 'dashboard/compliance',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.read', 'education.managerial.read', 'education.regulations.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./dashboard/education-compliance-dashboard.component').then((m) => m.EducationComplianceDashboardComponent),
      },
      {
        path: 'dashboard/teacher',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.portfolios.read', 'education.evaluations.read', 'education.declarations.read', 'education.mobility.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./dashboard/education-teacher-dashboard.component').then((m) => m.EducationTeacherDashboardComponent),
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
        path: 'governance/ca-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.governance.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./governance/ca-meeting-wizard.component').then((m) => m.CaMeetingWizardComponent),
      },
      {
        path: 'governance/minutes-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.governance.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./governance/meeting-minute-wizard.component').then((m) => m.MeetingMinuteWizardComponent),
      },
      {
        path: 'governance/votes-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.governance.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./governance/meeting-vote-wizard.component').then((m) => m.MeetingVoteWizardComponent),
      },
      {
        path: 'governance/resolutions-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.governance.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./governance/meeting-resolution-wizard.component').then((m) => m.MeetingResolutionWizardComponent),
      },
      {
        path: 'governance/managerial-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.managerial.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./governance/managerial-dossier-wizard.component').then((m) => m.ManagerialDossierWizardComponent),
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
        path: 'personnel/wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.personnel.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./personnel/personnel-record-wizard.component').then((m) => m.PersonnelRecordWizardComponent),
      },
      {
        path: 'personnel/evaluations-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.evaluations.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./personnel/evaluation-record-wizard.component').then((m) => m.EvaluationRecordWizardComponent),
      },
      {
        path: 'personnel/declarations-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.declarations.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./personnel/declaration-record-wizard.component').then((m) => m.DeclarationRecordWizardComponent),
      },
      {
        path: 'personnel/mobility-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.mobility.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./personnel/mobility-record-wizard.component').then((m) => m.MobilityRecordWizardComponent),
      },
      {
        path: 'personnel/merit-wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.gradatii.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./personnel/merit-record-wizard.component').then((m) => m.MeritRecordWizardComponent),
      },
      {
        path: 'portfolio/me',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.portfolios.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./portfolio/education-portfolio-self-service.component').then((m) => m.EducationPortfolioSelfServiceComponent),
      },
      {
        path: 'portfolio/workflow',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.portfolios.read'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./portfolio/education-portfolio-workflow-wizard.component').then((m) => m.EducationPortfolioWorkflowWizardComponent),
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
        path: 'portfolio/wizard',
        canActivate: [permissionGuard],
        data: {
          permissions: ['education.portfolios.manage'],
          permissionsMode: 'any',
          module: 'education',
        },
        loadComponent: () =>
          import('./portfolio/portfolio-record-wizard.component').then((m) => m.PortfolioRecordWizardComponent),
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
