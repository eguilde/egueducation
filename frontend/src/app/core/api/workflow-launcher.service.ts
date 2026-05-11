import { Injectable, inject } from '@angular/core';
import { TranslocoService } from '@jsverse/transloco';

import { WorkflowApiService } from './workflow-api.service';
import {
  GdprSubjectRequest,
  GdprSubjectExport,
  GdprPublicationReview,
  GovernanceDecision,
  GovernanceMeeting,
  ManagerialDossier,
  MeritGrant,
  MobilityCase,
  PersonnelDeclaration,
  PersonnelEvaluation,
  PersonnelRecord,
  PortfolioRecord,
  RegulationRecord,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class WorkflowLauncherService {
  private readonly workflowApi = inject(WorkflowApiService);
  private readonly transloco = inject(TranslocoService);

  launchGovernanceWorkflow(record: GovernanceMeeting) {
    return this.workflowApi.createTask({
      definition_code: 'governance-meeting-review',
      title: this.transloco.translate('education.workflow.defaultTitle', {
        title: record.title,
      }),
      document_number: record.id,
      source_module: 'education.governance',
      source_record_id: record.id,
      priority: record.status === 'draft' ? 'medium' : 'high',
      assigned_to: record.secretary_name || record.chairperson,
      summary: [record.location, record.summary].filter(Boolean).join(' | '),
    });
  }

  launchDecisionWorkflow(record: GovernanceDecision) {
    return this.workflowApi.createTask({
      definition_code: 'governance-decision-review',
      title: this.transloco.translate('educationDecisions.workflow.defaultTitle', {
        title: record.title,
      }),
      document_number: record.decision_code,
      source_module: 'education.decisions',
      source_record_id: record.id,
      priority: record.publication_status === 'pending_anonymization' ? 'high' : 'medium',
      assigned_to: record.signed_by,
      summary: [record.legal_basis, record.summary].filter(Boolean).join(' | '),
    });
  }

  launchManagerialWorkflow(record: ManagerialDossier) {
    return this.workflowApi.createTask({
      definition_code: 'managerial-dossier-review',
      title: this.transloco.translate('educationManagerial.workflow.defaultTitle', {
        title: record.title,
      }),
      document_number: record.dossier_code,
      source_module: 'education.managerial',
      source_record_id: record.id,
      priority: record.publication_required ? 'high' : 'medium',
      assigned_to: record.owner_name,
      summary: [record.dossier_type, record.summary].filter(Boolean).join(' | '),
    });
  }

  launchRegulationWorkflow(record: RegulationRecord) {
    return this.workflowApi.createTask({
      definition_code: 'education-regulation-review',
      title: this.transloco.translate('educationRegulations.workflow.defaultTitle', {
        title: record.title,
      }),
      document_number: record.regulation_code,
      source_module: 'education.regulations',
      source_record_id: record.id,
      priority: record.status === 'consultation' ? 'high' : 'medium',
      assigned_to: record.owner_name,
      summary: [record.regulation_type, record.approval_status, record.summary].filter(Boolean).join(' | '),
    });
  }

  launchPersonnelEvaluationWorkflow(record: PersonnelRecord) {
    return this.workflowApi.createTask({
      definition_code: 'personnel-evaluation',
      title: this.transloco.translate('educationPersonnel.workflow.evaluationTitle', {
        name: record.full_name,
      }),
      document_number: record.employee_code,
      source_module: 'education.personnel',
      source_record_id: record.id,
      priority: record.evaluation_status === 'draft' ? 'medium' : 'high',
      assigned_to: record.assigned_unit,
      summary: [record.role_title, record.notes].filter(Boolean).join(' | '),
    });
  }

  launchPersonnelMobilityWorkflow(record: PersonnelRecord) {
    return this.workflowApi.createTask({
      definition_code: 'personnel-mobility',
      title: this.transloco.translate('educationPersonnel.workflow.mobilityTitle', {
        name: record.full_name,
      }),
      document_number: record.employee_code,
      source_module: 'education.personnel',
      source_record_id: record.id,
      priority: record.mobility_stage === 'none' ? 'medium' : 'urgent',
      assigned_to: record.assigned_unit,
      summary: [record.mobility_stage, record.notes].filter(Boolean).join(' | '),
    });
  }

  launchEvaluationWorkflow(record: PersonnelEvaluation) {
    return this.workflowApi.createTask({
      definition_code: 'personnel-evaluation-review',
      title: this.transloco.translate('educationEvaluations.workflow.defaultTitle', {
        name: record.full_name,
      }),
      document_number: record.evaluation_code,
      source_module: 'education.evaluations',
      source_record_id: record.id,
      priority: record.status === 'submitted' ? 'high' : 'medium',
      assigned_to: record.evaluator_name,
      summary: [record.role_title, String(record.score), record.summary].filter(Boolean).join(' | '),
    });
  }

  launchDeclarationWorkflow(record: PersonnelDeclaration) {
    return this.workflowApi.createTask({
      definition_code: 'personnel-declaration-review',
      title: this.transloco.translate('educationDeclarations.workflow.defaultTitle', {
        name: record.full_name,
      }),
      document_number: record.declaration_code,
      source_module: 'education.declarations',
      source_record_id: record.id,
      priority: record.status === 'submitted' ? 'high' : 'medium',
      assigned_to: record.full_name,
      summary: [record.declaration_type, record.summary].filter(Boolean).join(' | '),
    });
  }

  launchMobilityCaseWorkflow(record: MobilityCase) {
    return this.workflowApi.createTask({
      definition_code: 'mobility-case-review',
      title: this.transloco.translate('educationMobility.workflow.defaultTitle', {
        name: record.full_name,
      }),
      document_number: record.case_code,
      source_module: 'education.mobility',
      source_record_id: record.id,
      priority: record.status === 'approved' ? 'medium' : 'high',
      assigned_to: record.reviewed_by,
      summary: [record.request_type, record.destination_school, record.notes].filter(Boolean).join(' | '),
    });
  }

  launchMeritGrantWorkflow(record: MeritGrant) {
    return this.workflowApi.createTask({
      definition_code: 'merit-grant-review',
      title: this.transloco.translate('educationGradatii.workflow.defaultTitle', {
        name: record.full_name,
      }),
      document_number: record.grant_code,
      source_module: 'education.gradatii',
      source_record_id: record.id,
      priority: record.status === 'funded' ? 'medium' : 'high',
      assigned_to: record.committee_name,
      summary: [record.role_title, String(record.score), record.notes].filter(Boolean).join(' | '),
    });
  }

  launchPortfolioWorkflow(record: PortfolioRecord) {
    return this.workflowApi.createTask({
      definition_code: 'portfolio-review',
      title: this.transloco.translate('educationPortfolios.workflow.defaultTitle', {
        owner: record.owner_name,
      }),
      document_number: record.portfolio_code,
      source_module: 'education.portfolios',
      source_record_id: record.id,
      priority: record.transfer_status === 'none' ? 'medium' : 'high',
      assigned_to: record.custodian,
      summary: [record.owner_role, record.notes].filter(Boolean).join(' | '),
    });
  }

  launchGdprRequestWorkflow(record: GdprSubjectRequest) {
    return this.workflowApi.createTask({
      definition_code: 'gdpr-request',
      title: this.transloco.translate('gdpr.workflow.defaultTitle', {
        name: record.subject_name,
      }),
      document_number: record.request_code,
      source_module: 'gdpr.subject_requests',
      source_record_id: record.id,
      priority: record.anonymization_required ? 'high' : 'medium',
      assigned_to: record.handled_by,
      summary: [record.request_type, record.notes].filter(Boolean).join(' | '),
    });
  }

  launchGdprExportWorkflow(record: GdprSubjectExport) {
    return this.workflowApi.createTask({
      definition_code: 'gdpr-subject-export',
      title: this.transloco.translate('gdprExports.workflow.defaultTitle', {
        name: record.subject_name,
      }),
      document_number: record.export_code,
      source_module: 'gdpr.exports',
      source_record_id: record.id,
      priority: record.status === 'pending_approval' ? 'high' : 'medium',
      assigned_to: record.approved_by,
      summary: [record.export_format, record.package_summary, record.notes].filter(Boolean).join(' | '),
    });
  }

  launchGdprPublicationWorkflow(record: GdprPublicationReview) {
    return this.workflowApi.createTask({
      definition_code: 'gdpr-publication-review',
      title: this.transloco.translate('gdprPublication.workflow.defaultTitle', {
        label: record.source_label,
      }),
      document_number: record.review_code,
      source_module: 'gdpr.publication',
      source_record_id: record.id,
      priority: record.anonymization_status === 'pending' ? 'high' : 'medium',
      assigned_to: record.reviewed_by,
      summary: [record.source_module, record.legal_basis, record.notes].filter(Boolean).join(' | '),
    });
  }
}
