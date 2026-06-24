export type EducationWizardStepStatus = 'pending' | 'active' | 'completed' | 'blocked';

export interface EducationWizardStepState {
  key: string;
  label: string;
  description?: string;
  status: EducationWizardStepStatus;
  valid?: boolean;
  dirty?: boolean;
}

export interface EducationWizardAuditEvent {
  id: string;
  title: string;
  summary: string;
  happenedOn?: string;
  actorName?: string;
}

export interface EducationWizardDocumentPreview {
  id: string;
  title: string;
  summary?: string;
  status?: string;
}

export interface EducationWizardDeadlineItem {
  id: string;
  label: string;
  dueOn?: string;
  status?: string;
}
