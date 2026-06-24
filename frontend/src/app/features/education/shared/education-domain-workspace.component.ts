import { CommonModule } from '@angular/common';
import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputNumberModule } from 'primeng/inputnumber';
import { InputTextModule } from 'primeng/inputtext';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { SelectModule } from 'primeng/select';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToastModule } from 'primeng/toast';
import { ToggleSwitchModule } from 'primeng/toggleswitch';
import { DatePickerModule } from 'primeng/datepicker';

import { TableQuery } from '../../../core/api/api.types';
import { AuthzService } from '../../../core/authz/authz.service';
import {
  PageEvent,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../../shared/server-table/server-table.component';
import {
  EducationColumn,
  EducationCreateForm,
  EducationDetailChildResourceConfig,
  EducationResourceConfig,
  EducationRow,
  EducationSectionConfig,
  EducationServerTableColumn,
} from './education.models';
import { educationStatusSeverity } from './education-status.helpers';

interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

interface EducationExportPayload {
  title: string;
  filename: string;
  headers: string[];
  rows: string[][];
}

interface GovernanceMeetingFinalizationSummary {
  participants: {
    recorded_participants: number;
    present_participants: number;
    signed_participants: number;
    voting_participants: number;
  };
  votes: {
    total: number;
    adopted: number;
    requires_follow_up: number;
    missing_resolutions: number;
  };
  minutes: {
    total: number;
    requires_publication: number;
    open_follow_up_items: number;
  };
  resolutions: {
    total: number;
    ready_for_publication: number;
    published: number;
    pending_publication: number;
    pending_anonymization: number;
  };
  documents: {
    total: number;
    process_verbal_documents: number;
    published_process_verbals: number;
  };
  readiness: {
    ready_to_close: boolean;
    ready_to_publish: boolean;
    blockers: string[];
  };
}

interface GovernanceSummaryAction {
  id: string;
  label: string;
  icon: string;
  targetTab: string;
  kind: 'tab' | 'create';
}

interface PersonnelPortfolioDossierSummary {
  personnel: {
    id: string;
    full_name: string;
    role_title: string;
    school_year: string;
    has_portfolio: boolean;
  };
  dossier: {
    total_documents: number;
    personal_file_documents: number;
    director_file_documents: number;
    adjunct_director_file_documents: number;
    sensitive_documents: number;
    documents_marked_for_portfolio: number;
    evaluation_documents: number;
    administrative_career_documents: number;
  };
  portfolio: {
    matched_records: number;
    validated_records: number;
    total_documents: number;
    portfolio_scope_documents: number;
    personnel_scope_documents: number;
    verified_documents: number;
    last_updated_on: string;
  };
  relation: {
    mirrored_file_references: number;
    evaluation_results_enter_personnel_file: boolean;
    administrative_docs_enter_personnel_file: boolean;
    institution_may_duplicate_or_separate: boolean;
    duplication_mode: string;
    rules: string[];
  };
  readiness: {
    clear_delimitation: boolean;
    blockers: string[];
  };
}

interface PersonnelSummaryAction {
  id: string;
  label: string;
  icon: string;
  targetTab: string;
}

interface PortfolioTransferSummary {
  portfolio: {
    id: string;
    portfolio_code: string;
    owner_name: string;
    owner_role: string;
    school_year: string;
    transfer_status: string;
  };
  completeness: {
    total_documents: number;
    portfolio_documents: number;
    personnel_documents: number;
    sensitive_documents: number;
    total_checklist_items: number;
    mandatory_checklist_items: number;
    completed_checklist_items: number;
    partial_checklist_items: number;
    missing_checklist_items: number;
    reviewing_checklist_items: number;
    opis_entries: number;
    custody_events: number;
    review_events: number;
    valorification_events: number;
    ready_for_review: boolean;
    ready_for_transfer: boolean;
    blockers: string[];
  };
  transfer: {
    total_events: number;
    prepared_events: number;
    sent_events: number;
    received_events: number;
    closed_events: number;
    current_direction: string;
    last_transfer: null | {
      id: string;
      transfer_code: string;
      transfer_type: string;
      source_institution: string;
      destination_institution: string;
      status: string;
      handover_on: string;
      received_on: string;
      handover_by: string;
      received_by: string;
    };
  };
  mobility: {
    matched_cases: number;
    active_cases: number;
    transfer_cases: number;
    detachment_cases: number;
    restriction_cases: number;
    current_unit_mentions: number;
    destination_mentions: number;
  };
  valorification: {
    total_events: number;
    open_events: number;
    completed_events: number;
    linked_evaluations: number;
    linked_mobility: number;
    linked_merit: number;
    last_event: null | {
      id: string;
      valorification_code: string;
      scope: string;
      status: string;
      requested_by: string;
      target_institution: string;
      target_reference: string;
      started_on: string;
      completed_on: string;
    };
    scopes: Array<{
      scope: string;
      total: number;
      open: number;
      completed: number;
    }>;
  };
  readiness: {
    ready_to_request: boolean;
    ready_to_send: boolean;
    ready_to_confirm: boolean;
    ready_to_close: boolean;
    blockers: string[];
  };
}

interface PortfolioTransferSummaryAction {
  id: string;
  label: string;
  icon: string;
  kind: 'tab' | 'create' | 'advance' | 'utility';
  targetTab?: string;
  advanceAction?: 'mark_sent' | 'confirm_received' | 'close_transfer';
  utilityAction?: 'regenerate_opis';
}

interface PortfolioValorificationSummaryAction {
  id: string;
  label: string;
  icon: string;
  kind: 'tab' | 'create';
  targetTab: string;
}

interface RegulationProceduralSummary {
  regulation: {
    id: string;
    regulation_code: string;
    regulation_type: string;
    title: string;
    school_year: string;
    status: string;
    approval_status: string;
    review_due_on: string;
    approved_on: string;
  };
  versions: {
    total_versions: number;
    consultation_versions: number;
    endorsed_versions: number;
    approved_versions: number;
    published_versions: number;
    latest_version: null | {
      id: string;
      version_label: string;
      version_status: string;
      approved_on: string;
      effective_from: string;
      published_on: string;
      prepared_by: string;
      file_reference: string;
    };
  };
  workflow: {
    total_phases: number;
    completed_phases: number;
    open_phases: number;
    returned_phases: number;
    cancelled_phases: number;
    feedback_count: number;
    current_phase: null | {
      id: string;
      phase_order: number;
      phase_type: string;
      status: string;
      audience: string;
      started_on: string;
      due_on: string;
      completed_on: string;
      feedback_count: number;
      decision_reference: string;
    };
  };
  readiness: {
    ready_for_cp_endorsement: boolean;
    ready_for_ca_approval: boolean;
    ready_for_publication: boolean;
    ready_for_review: boolean;
    blockers: string[];
  };
}

interface RegulationSummaryAction {
  id: string;
  label: string;
  icon: string;
  kind: 'tab' | 'create';
  targetTab: string;
}

interface GovernanceBodyCompletenessSummary {
  body: {
    id: string;
    school_year: string;
    organism: string;
    active_members: number;
    voting_members: number;
    chairperson_covered: boolean;
    secretary_covered: boolean;
    expired_mandates: number;
    total_meetings: number;
    scheduled_meetings: number;
    held_meetings: number;
    published_meetings: number;
    latest_meeting_on: string;
    readiness_status: string;
  };
  membership: {
    active_members: number;
    voting_members: number;
    chairperson_covered: boolean;
    secretary_covered: boolean;
    expired_mandates: number;
    member_names: string[];
  };
  meetings: {
    total_meetings: number;
    scheduled_meetings: number;
    held_meetings: number;
    published_meetings: number;
    last_meeting_title: string;
    last_meeting_on: string;
  };
  readiness: {
    ready_for_operation: boolean;
    ready_for_meetings: boolean;
    blockers: string[];
  };
}

interface GovernanceBodySummaryAction {
  id: string;
  label: string;
  icon: string;
  targetTab: string;
}

interface CommitteeCompletenessSummary {
  committee: {
    id: string;
    committee_code: string;
    school_year: string;
    committee_type: string;
    title: string;
    status: string;
    decision_reference: string;
    starts_on: string;
    ends_on: string;
    evaluation_scope: boolean;
  };
  membership: {
    active_members: number;
    voting_members: number;
    chairperson_covered: boolean;
    secretary_covered: boolean;
    member_names: string[];
  };
  readiness: {
    ready_for_operation: boolean;
    blockers: string[];
  };
}

interface CommitteeSummaryAction {
  id: string;
  label: string;
  icon: string;
  targetTab: string;
}

interface ManagerialPortfolioSummary {
  dossier: {
    id: string;
    dossier_code: string;
    dossier_type: string;
    title: string;
    school_year: string;
    status: string;
    owner_name: string;
    publication_required: boolean;
  };
  portfolio: {
    matched_personnel: number;
    managerial_documents: number;
    mandatory_documents: number;
    approved_documents: number;
    published_documents: number;
    publication_required_documents: number;
    missing_mandatory_categories: string[];
  };
  workflow: {
    total_steps: number;
    completed_steps: number;
    open_steps: number;
    signature_steps: number;
    completed_signature_steps: number;
  };
  personnel_file: {
    matched_documents: number;
    management_documents: number;
    sensitive_documents: number;
    mirrored_references: number;
  };
  readiness: {
    ready_for_review: boolean;
    ready_for_publication: boolean;
    blockers: string[];
  };
}

interface ManagerialPortfolioSummaryAction {
  id: string;
  label: string;
  icon: string;
  targetTab: string;
}

type EditorTarget =
  | { kind: 'resource'; resource: EducationResourceConfig }
  | { kind: 'child'; resource: EducationResourceConfig; child: EducationDetailChildResourceConfig; parent: EducationRow };

type DeleteTarget = EditorTarget & { row: EducationRow };

@Component({
  selector: 'app-education-domain-workspace',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    DatePickerModule,
    DialogModule,
    InputNumberModule,
    InputTextModule,
    ProgressSpinnerModule,
    SelectModule,
    ServerTableComponent,
    TabsModule,
    TagModule,
    TextareaModule,
    ToastModule,
    ToggleSwitchModule,
  ],
  providers: [MessageService],
  template: `
    <section class="flex h-full min-h-0 flex-col overflow-hidden">
      <p-toast />

      <div class="shrink-0 px-3 pb-3 pt-3">
        <div class="rounded-3xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-1">
              <h1 class="m-0 text-2xl font-semibold">{{ section().label }}</h1>
              <p class="m-0 max-w-4xl text-sm text-muted-color">{{ section().description }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              @if (activePresetLabel()) {
                <p-tag [value]="activePresetLabel()" severity="contrast" />
                <p-button
                  label="Reseteaza presetarea"
                  icon="pi pi-times"
                  size="small"
                  severity="secondary"
                  [outlined]="true"
                  (onClick)="clearPreset()"
                />
              }
              <p-tag [value]="visibleResources().length + ' registre'" severity="secondary" />
            </div>
          </div>
        </div>
      </div>

      <p-tabs
        [value]="activeResourceKey()"
        (valueChange)="activateResource(coerceTabValue($event))"
        class="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <p-tablist class="shrink-0">
          @for (resource of visibleResources(); track resource.key) {
            <p-tab [value]="resource.key">
              <span class="inline-flex items-center gap-2">
                <i [class]="resource.icon"></i>
                {{ resource.label }}
              </span>
            </p-tab>
          }
        </p-tablist>

        <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
          @for (resource of visibleResources(); track resource.key) {
            <p-tabpanel [value]="resource.key" class="flex min-h-0 flex-1 overflow-hidden p-0">
              <div class="min-h-0 flex-1 overflow-hidden px-3 pb-3">
                <app-server-table
                  class="h-full"
                  [columns]="tableColumns(resource.columns)"
                  [rows]="rows(resource)"
                  [total]="total(resource)"
                  [pageIndex]="query(resource).page - 1"
                  [pageSize]="query(resource).pageSize"
                  [loading]="loadingKey() === resource.key"
                  [emptyMessage]="resource.emptyText"
                  [sortActive]="query(resource).sort || ''"
                  [sortDirection]="query(resource).direction || ''"
                  [rowActions]="rowActions(resource)"
                  actionColumnLabel="Actiuni"
                  (pageChange)="onPageChange(resource, $event)"
                  (sortChange)="onSortChange(resource, $event)"
                  (filterChange)="onFilterChange(resource, $event)"
                  (actionClick)="onAction(resource, $event.action, $event.row)"
                >
                  <ng-template #serverTableToolbar>
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div class="space-y-1">
                        <h2 class="m-0 text-xl font-semibold">{{ resource.label }}</h2>
                        <p class="m-0 text-sm text-muted-color">{{ resource.description }}</p>
                      </div>
                      <div class="flex flex-wrap items-center justify-end gap-2">
                        <p-tag [value]="resource.readPermission" severity="secondary" />
                        @if (total(resource) > 0) {
                          <p-button
                            label="PDF"
                            icon="pi pi-file-pdf"
                            severity="secondary"
                            [outlined]="true"
                            [loading]="exportingKey() === resource.key + ':pdf'"
                            (onClick)="exportPdf(resource)"
                          />
                          <p-button
                            label="Excel (CSV)"
                            icon="pi pi-file-excel"
                            severity="secondary"
                            [outlined]="true"
                            [loading]="exportingKey() === resource.key + ':csv'"
                            (onClick)="exportCsv(resource)"
                          />
                        }
                      </div>
                    </div>
                  </ng-template>

                  <ng-template #serverTableActionHeader>
                    @if (canCreateResource(resource)) {
                      <p-button
                        icon="pi pi-plus"
                        size="small"
                        [text]="true"
                        [rounded]="true"
                        [attr.aria-label]="'Adauga ' + resource.label"
                        (onClick)="openCreateResource(resource)"
                      />
                    }
                  </ng-template>
                </app-server-table>
              </div>
            </p-tabpanel>
          }
        </p-tabpanels>
      </p-tabs>

      <p-dialog
        [visible]="editorDialogOpen()"
        (visibleChange)="onEditorDialogVisibilityChange($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editorDialogHeader()"
        [style]="{ width: 'min(58rem, 94vw)' }"
      >
        @if (editorFields().length) {
          <div class="grid gap-4 md:grid-cols-2">
            @for (field of editorFields(); track field.field) {
              <label class="education-field" [ngClass]="field.wide ? 'md:col-span-2' : null">
                <span>{{ field.label }} @if (field.required) { <strong>*</strong> }</span>
                @switch (field.type) {
                  @case ('textarea') {
                    <textarea pTextarea rows="4" [(ngModel)]="createForm[field.field]"></textarea>
                  }
                  @case ('number') {
                    <p-inputNumber
                      [(ngModel)]="createForm[field.field]"
                      [min]="field.min ?? 0"
                      [max]="field.max ?? null"
                      [minFractionDigits]="field.step && field.step < 1 ? 2 : 0"
                      [maxFractionDigits]="field.step && field.step < 1 ? 2 : 0"
                      [step]="field.step ?? 1"
                    />
                  }
                  @case ('date') {
                    <p-datepicker appendTo="body" dateFormat="yy-mm-dd" [(ngModel)]="createForm[field.field]" [showIcon]="true" />
                  }
                  @case ('select') {
                    <p-select
                      appendTo="body"
                      [options]="fieldOptions(field)"
                      optionLabel="label"
                      optionValue="value"
                      [(ngModel)]="createForm[field.field]"
                    />
                  }
                  @case ('boolean') {
                    <p-toggleSwitch [(ngModel)]="createForm[field.field]" />
                  }
                  @default {
                    <input pInputText [(ngModel)]="createForm[field.field]" />
                  }
                }
              </label>
            }
          </div>
        }
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunta" severity="secondary" [outlined]="true" (onClick)="closeEditorDialog()" />
            <p-button [label]="editorMode() === 'edit' ? 'Actualizeaza' : 'Salveaza'" icon="pi pi-check" [loading]="creating()" (onClick)="saveRecord()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="detailDialogOpen()"
        (visibleChange)="onDetailDialogVisibilityChange($event)"
        [modal]="true"
        [draggable]="false"
        header="Detalii inregistrare"
        [style]="{ width: 'min(82rem, 96vw)' }"
      >
        @if (selectedResource(); as resource) {
          @if (detailLoading()) {
            <div class="flex min-h-40 flex-col items-center justify-center gap-3">
              <p-progressSpinner strokeWidth="4" [style]="{ width: '3rem', height: '3rem' }" />
              <p-tag value="Se incarca detaliile complete..." severity="secondary" />
            </div>
          } @else if (selectedRow(); as row) {
            <p-tabs
              [value]="activeDetailTab()"
              (valueChange)="activateDetailTab(coerceTabValue($event, 'overview'))"
              class="flex min-h-0 flex-col overflow-hidden"
            >
              <p-tablist class="shrink-0">
                <p-tab value="overview">
                  <span class="inline-flex items-center gap-2">
                    <i class="pi pi-info-circle"></i>
                    Date generale
                  </span>
                </p-tab>
                @for (child of resource.detailChildren ?? []; track child.key) {
                  <p-tab [value]="child.key">
                    <span class="inline-flex items-center gap-2">
                      <i [class]="child.icon"></i>
                      {{ child.label }}
                    </span>
                  </p-tab>
                }
              </p-tablist>

              <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
                <p-tabpanel value="overview" class="p-0 pt-4">
                  @if (detailSummaryLoading()) {
                    <div class="mb-4 flex items-center gap-2 rounded-2xl border border-surface p-3">
                      <p-progressSpinner strokeWidth="4" [style]="{ width: '1.5rem', height: '1.5rem' }" />
                      <span class="text-sm text-muted-color">Se incarca sumarul procedural...</span>
                    </div>
                  } @else if (governanceFinalizationSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Stare de finalizare</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Rezumat procedural pentru inchiderea si publicarea sedintei selectate.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.ready_to_close ? 'Pregatita pentru inchidere' : 'Inchidere blocata'"
                            [severity]="summary.readiness.ready_to_close ? 'success' : 'warn'"
                          />
                          <p-tag
                            [value]="summary.readiness.ready_to_publish ? 'Pregatita pentru publicare' : 'Publicare blocata'"
                            [severity]="summary.readiness.ready_to_publish ? 'success' : 'danger'"
                          />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Participanti</div>
                          <div class="mt-2 text-sm">
                            <div>Cvorum necesar: <strong>{{ row['quorum_required'] ?? 0 }}</strong></div>
                            <div>Inregistrati: <strong>{{ summary.participants.recorded_participants }}</strong></div>
                            <div>Prezenti: <strong>{{ summary.participants.present_participants }}</strong></div>
                            <div>Semnati: <strong>{{ summary.participants.signed_participants }}</strong></div>
                            <div>Cu vot: <strong>{{ summary.participants.voting_participants }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Voturi si hotarari</div>
                          <div class="mt-2 text-sm">
                            <div>Voturi: <strong>{{ summary.votes.total }}</strong></div>
                            <div>Adoptate: <strong>{{ summary.votes.adopted }}</strong></div>
                            <div>Necesita follow-up: <strong>{{ summary.votes.requires_follow_up }}</strong></div>
                            <div>Hotarari lipsa: <strong>{{ summary.votes.missing_resolutions }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Minute si documente</div>
                          <div class="mt-2 text-sm">
                            <div>Minute: <strong>{{ summary.minutes.total }}</strong></div>
                            <div>Minute de publicat: <strong>{{ summary.minutes.requires_publication }}</strong></div>
                            <div>Actiuni deschise: <strong>{{ summary.minutes.open_follow_up_items }}</strong></div>
                            <div>Procese-verbale publicate: <strong>{{ summary.documents.published_process_verbals }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje procedurale" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="governanceBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of governanceSummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                (onClick)="executeGovernanceSummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Tranzitii procedurale</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            <p-button
                              label="Marcheaza ca tinuta"
                              icon="pi pi-check-circle"
                              size="small"
                              [disabled]="!canTransitionGovernanceMeeting(summary, 'held')"
                              [loading]="governanceTransitionLoading() === 'held'"
                              (onClick)="transitionGovernanceMeeting('held')"
                            />
                            <p-button
                              label="Marcheaza ca publicata"
                              icon="pi pi-send"
                              size="small"
                              severity="secondary"
                              [outlined]="true"
                              [disabled]="!canTransitionGovernanceMeeting(summary, 'published')"
                              [loading]="governanceTransitionLoading() === 'published'"
                              (onClick)="transitionGovernanceMeeting('published')"
                            />
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Export procedural</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            <p-button
                              label="PDF sumar procedural"
                              icon="pi pi-file-pdf"
                              size="small"
                              severity="secondary"
                              [outlined]="true"
                              [loading]="governanceSummaryExporting()"
                              (onClick)="exportGovernanceSummaryPdf(summary)"
                            />
                          </div>
                        </div>
                      </div>
                    </section>
                  } @else if (personnelPortfolioDossierSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Relatia portofoliu - dosar personal</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Delimitare operationala intre documentele din dosarul personal si cele din portofoliul profesional.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.clear_delimitation ? 'Delimitare clara' : 'Necesita clarificari'"
                            [severity]="summary.readiness.clear_delimitation ? 'success' : 'warn'"
                          />
                          <p-tag
                            [value]="summary.relation.duplication_mode === 'selectiva' ? 'Dublare selectiva' : summary.relation.duplication_mode"
                            severity="secondary"
                          />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Dosar personal</div>
                          <div class="mt-2 text-sm">
                            <div>Total documente: <strong>{{ summary.dossier.total_documents }}</strong></div>
                            <div>Documente evaluare: <strong>{{ summary.dossier.evaluation_documents }}</strong></div>
                            <div>Studii / cariera / management: <strong>{{ summary.dossier.administrative_career_documents }}</strong></div>
                            <div>Referibile in portofoliu: <strong>{{ summary.dossier.documents_marked_for_portfolio }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Portofoliu profesional</div>
                          <div class="mt-2 text-sm">
                            <div>Registre potrivite: <strong>{{ summary.portfolio.matched_records }}</strong></div>
                            <div>Portofolii validate: <strong>{{ summary.portfolio.validated_records }}</strong></div>
                            <div>Documente totale: <strong>{{ summary.portfolio.total_documents }}</strong></div>
                            <div>Cu sursa dosar personal: <strong>{{ summary.portfolio.personnel_scope_documents }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Trasabilitate</div>
                          <div class="mt-2 text-sm">
                            <div>Referinte comune: <strong>{{ summary.relation.mirrored_file_references }}</strong></div>
                            <div>Documente sensibile in dosar: <strong>{{ summary.dossier.sensitive_documents }}</strong></div>
                            <div>Documente verificate in portofoliu: <strong>{{ summary.portfolio.verified_documents }}</strong></div>
                            <div>Ultima actualizare portofoliu: <strong>{{ summary.portfolio.last_updated_on || '-' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Reguli institutionale vizibile</div>
                          <div class="mt-3 grid gap-2">
                            @for (rule of summary.relation.rules; track rule) {
                              <div class="rounded-lg border border-surface px-3 py-2 text-sm">{{ rule }}</div>
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje de delimitare" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="personnelPortfolioBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of personnelSummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                (onClick)="executePersonnelSummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                      </div>
                    </section>
                  } @else if (portfolioTransferSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Transfer digital si mobilitate</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Starea circuitului de solicitare, transmitere si confirmare pentru portofoliul profesional.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.blockers.length ? 'Necesita actiuni' : 'Circuit clar'"
                            [severity]="summary.readiness.blockers.length ? 'warn' : 'success'"
                          />
                          <p-tag [value]="displayTransferStatus(summary.portfolio.transfer_status)" severity="secondary" />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="flex flex-wrap items-start justify-between gap-3">
                            <div>
                              <h4 class="m-0 text-sm font-semibold">Completitudine portofoliu</h4>
                              <p class="mt-1 text-sm text-muted-color">
                                Urmarim prezenta documentelor, opisul, checklistul si traseele institutionale care arata ca portofoliul este pregatit pentru analiza.
                              </p>
                            </div>
                            <div class="flex flex-wrap items-center gap-2">
                              <p-tag
                                [value]="summary.completeness.ready_for_review ? 'Pregatit pentru analiza' : 'Necesita completari'"
                                [severity]="summary.completeness.ready_for_review ? 'success' : 'warn'"
                              />
                              <p-tag
                                [value]="summary.completeness.ready_for_transfer ? 'Pregatit pentru transfer' : 'Transfer nefinalizat'"
                                [severity]="summary.completeness.ready_for_transfer ? 'info' : 'secondary'"
                              />
                            </div>
                          </div>

                          <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Documente</div>
                              <div class="mt-2 text-sm">
                                <div>Total: <strong>{{ summary.completeness.total_documents }}</strong></div>
                                <div>Portofoliu: <strong>{{ summary.completeness.portfolio_documents }}</strong></div>
                                <div>Dosar personal: <strong>{{ summary.completeness.personnel_documents }}</strong></div>
                                <div>Sensibile: <strong>{{ summary.completeness.sensitive_documents }}</strong></div>
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Checklist</div>
                              <div class="mt-2 text-sm">
                                <div>Total: <strong>{{ summary.completeness.total_checklist_items }}</strong></div>
                                <div>Obligatorii: <strong>{{ summary.completeness.mandatory_checklist_items }}</strong></div>
                                <div>Completate: <strong>{{ summary.completeness.completed_checklist_items }}</strong></div>
                                <div>Lipsa / partial: <strong>{{ summary.completeness.missing_checklist_items + summary.completeness.partial_checklist_items }}</strong></div>
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Opis si verificari</div>
                              <div class="mt-2 text-sm">
                                <div>Opis: <strong>{{ summary.completeness.opis_entries }}</strong></div>
                                <div>Verificari: <strong>{{ summary.completeness.review_events }}</strong></div>
                                <div>Custodie: <strong>{{ summary.completeness.custody_events }}</strong></div>
                                <div>Valorificari: <strong>{{ summary.completeness.valorification_events }}</strong></div>
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Stare institutionala</div>
                              <div class="mt-2 text-sm">
                                <div>Pregatit pentru analiza: <strong>{{ summary.completeness.ready_for_review ? 'Da' : 'Nu' }}</strong></div>
                                <div>Pregatit pentru transfer: <strong>{{ summary.completeness.ready_for_transfer ? 'Da' : 'Nu' }}</strong></div>
                                <div>Documente verificate: <strong>{{ summary.completeness.reviewing_checklist_items }}</strong></div>
                                <div>Elemente pendinte: <strong>{{ summary.completeness.blockers.length }}</strong></div>
                              </div>
                            </div>
                          </div>

                          <div class="mt-4 rounded-xl border border-surface p-3">
                            <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                            <div class="mt-3 flex flex-wrap gap-2">
                              @if (!summary.completeness.blockers.length) {
                                <p-tag value="Nu exista blocaje procedurale" severity="success" />
                              } @else {
                                @for (blocker of summary.completeness.blockers; track blocker) {
                                  <p-tag [value]="portfolioCompletenessBlockerLabel(blocker)" severity="warn" />
                                }
                              }
                            </div>
                          </div>
                        </div>

                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Circuit transfer</div>
                          <div class="mt-2 text-sm">
                            <div>Evenimente totale: <strong>{{ summary.transfer.total_events }}</strong></div>
                            <div>Pregatite: <strong>{{ summary.transfer.prepared_events }}</strong></div>
                            <div>Trimise: <strong>{{ summary.transfer.sent_events }}</strong></div>
                            <div>Receptionate / inchise: <strong>{{ summary.transfer.received_events + summary.transfer.closed_events }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Mobilitate asociata</div>
                          <div class="mt-2 text-sm">
                            <div>Cazuri potrivite: <strong>{{ summary.mobility.matched_cases }}</strong></div>
                            <div>Cazuri active: <strong>{{ summary.mobility.active_cases }}</strong></div>
                            <div>Transferuri: <strong>{{ summary.mobility.transfer_cases }}</strong></div>
                            <div>Detasari: <strong>{{ summary.mobility.detachment_cases }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Unitate sursa / curenta</div>
                          <div class="mt-2 text-sm">
                            <div>Mentiuni unitate baza: <strong>{{ summary.mobility.current_unit_mentions }}</strong></div>
                            <div>Mentiuni destinatie: <strong>{{ summary.mobility.destination_mentions }}</strong></div>
                            <div>Directie curenta: <strong>{{ summary.transfer.current_direction || '-' }}</strong></div>
                            <div>Ultim cod transfer: <strong>{{ summary.transfer.last_transfer?.transfer_code || '-' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Ultimul eveniment</div>
                          <div class="mt-2 text-sm">
                            @if (summary.transfer.last_transfer; as lastTransfer) {
                              <div>{{ lastTransfer.source_institution }} -> {{ lastTransfer.destination_institution }}</div>
                              <div class="text-muted-color">
                                {{ displayPortfolioTransferStatus(lastTransfer.status) }} | predat {{ lastTransfer.handover_on || '-' }} | receptionat {{ lastTransfer.received_on || '-' }}
                              </div>
                            } @else {
                              <div>Nu exista inca eveniment de transfer inregistrat.</div>
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje de transfer" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="portfolioTransferBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="flex flex-wrap items-start justify-between gap-3">
                            <div>
                              <h3 class="m-0 text-lg font-semibold">Fluxuri de valorificare</h3>
                              <p class="mt-1 text-sm text-muted-color">
                                Situatiile procedurale in care portofoliul sustine evaluarea, mobilitatea, gradele didactice si alte circuite institutionale.
                              </p>
                            </div>
                            <div class="flex flex-wrap items-center gap-2">
                              <p-tag [value]="summary.valorification.total_events ? 'Fluxuri documentate' : 'Fara fluxuri documentate'" [severity]="summary.valorification.total_events ? 'info' : 'warn'" />
                            </div>
                          </div>
                          <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Acoperire procedurala</div>
                              <div class="mt-2 text-sm">
                                <div>Fluxuri totale: <strong>{{ summary.valorification.total_events }}</strong></div>
                                <div>In lucru: <strong>{{ summary.valorification.open_events }}</strong></div>
                                <div>Finalizate: <strong>{{ summary.valorification.completed_events }}</strong></div>
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Corelari existente</div>
                              <div class="mt-2 text-sm">
                                <div>Evaluari corelate: <strong>{{ summary.valorification.linked_evaluations }}</strong></div>
                                <div>Mobilitati corelate: <strong>{{ summary.valorification.linked_mobility }}</strong></div>
                                <div>Gradatii / distinctii: <strong>{{ summary.valorification.linked_merit }}</strong></div>
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Ultimul flux</div>
                              <div class="mt-2 text-sm">
                                @if (summary.valorification.last_event; as event) {
                                  <div><strong>{{ displayPortfolioValorificationScope(event.scope) }}</strong></div>
                                  <div>{{ displayPortfolioValorificationStatus(event.status) }} | pornit {{ event.started_on || '-' }} | finalizat {{ event.completed_on || '-' }}</div>
                                  <div>{{ event.target_institution || '-' }} · {{ event.target_reference || '-' }}</div>
                                } @else {
                                  <div>Nu exista inca flux de valorificare inregistrat.</div>
                                }
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Scopuri acoperite</div>
                              <div class="mt-3 flex flex-wrap gap-2">
                                @if (!summary.valorification.scopes.length) {
                                  <p-tag value="Niciun scop documentat" severity="warn" />
                                } @else {
                                  @for (scope of summary.valorification.scopes; track scope.scope) {
                                    <p-tag [value]="displayPortfolioValorificationScope(scope.scope) + ': ' + scope.total" severity="secondary" />
                                  }
                                }
                              </div>
                            </div>
                            <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                              <div class="mt-3 flex flex-wrap gap-2">
                                @for (action of portfolioValorificationSummaryActions(summary); track action.id) {
                                  <p-button
                                    [label]="action.label"
                                    [icon]="action.icon"
                                    size="small"
                                    severity="secondary"
                                    [outlined]="true"
                                    (onClick)="executePortfolioValorificationSummaryAction(action)"
                                  />
                                }
                              </div>
                            </div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of portfolioTransferSummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                [loading]="portfolioTransferActionLoading() === action.id"
                                (onClick)="executePortfolioTransferSummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                      </div>
                    </section>
                  } @else if (regulationProceduralSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Circuit procedural ROF / ROI</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Urmarirea versiunilor, a consultarii, a avizarii in CP, a aprobarii in CA si a publicarii regulamentului.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.blockers.length ? 'Necesita completari' : 'Flux acoperit'"
                            [severity]="summary.readiness.blockers.length ? 'warn' : 'success'"
                          />
                          <p-tag [value]="displayRegulationStatus(summary.regulation.status)" severity="secondary" />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Versiuni</div>
                          <div class="mt-2 text-sm">
                            <div>Total versiuni: <strong>{{ summary.versions.total_versions }}</strong></div>
                            <div>In consultare: <strong>{{ summary.versions.consultation_versions }}</strong></div>
                            <div>Avizate CP: <strong>{{ summary.versions.endorsed_versions }}</strong></div>
                            <div>Aprobate / publicate: <strong>{{ summary.versions.approved_versions + summary.versions.published_versions }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Workflow</div>
                          <div class="mt-2 text-sm">
                            <div>Faze totale: <strong>{{ summary.workflow.total_phases }}</strong></div>
                            <div>Finalizate: <strong>{{ summary.workflow.completed_phases }}</strong></div>
                            <div>Deschise: <strong>{{ summary.workflow.open_phases }}</strong></div>
                            <div>Feedback colectat: <strong>{{ summary.workflow.feedback_count }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Readiness</div>
                          <div class="mt-2 text-sm">
                            <div>Pregatit pentru CP: <strong>{{ summary.readiness.ready_for_cp_endorsement ? 'Da' : 'Nu' }}</strong></div>
                            <div>Pregatit pentru CA: <strong>{{ summary.readiness.ready_for_ca_approval ? 'Da' : 'Nu' }}</strong></div>
                            <div>Pregatit pentru publicare: <strong>{{ summary.readiness.ready_for_publication ? 'Da' : 'Nu' }}</strong></div>
                            <div>Revizuire programata: <strong>{{ summary.regulation.review_due_on || '-' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Ultima versiune</div>
                          <div class="mt-2 text-sm">
                            @if (summary.versions.latest_version; as version) {
                              <div><strong>{{ version.version_label }}</strong> · {{ displayRegulationVersionStatus(version.version_status) }}</div>
                              <div class="text-muted-color">
                                Efecte de la {{ version.effective_from || '-' }} | aprobat {{ version.approved_on || '-' }} | publicat {{ version.published_on || '-' }}
                              </div>
                              <div>{{ version.prepared_by || '-' }} · {{ version.file_reference || '-' }}</div>
                            } @else {
                              <div>Nu exista inca versiune inregistrata pentru acest regulament.</div>
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Faza curenta</div>
                          <div class="mt-2 text-sm">
                            @if (summary.workflow.current_phase; as phase) {
                              <div><strong>{{ displayRegulationPhaseType(phase.phase_type) }}</strong> · {{ displayRegulationWorkflowStatus(phase.status) }}</div>
                              <div class="text-muted-color">
                                Termen {{ phase.due_on || '-' }} | inceput {{ phase.started_on || '-' }} | finalizat {{ phase.completed_on || '-' }}
                              </div>
                              <div>{{ phase.audience || '-' }} · feedback {{ phase.feedback_count }}</div>
                            } @else {
                              <div>Nu exista inca faze procedurale inregistrate.</div>
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje procedurale" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="regulationBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of regulationSummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                (onClick)="executeRegulationSummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                      </div>
                    </section>
                  } @else if (governanceBodyCompletenessSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Constituire si operare organism</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Verificare rapida pentru completitudinea CA, CP sau a altui organism de guvernanta.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.ready_for_operation ? 'Organism complet' : 'Necesita completari'"
                            [severity]="summary.readiness.ready_for_operation ? 'success' : 'warn'"
                          />
                          <p-tag [value]="displayGovernanceOrganism(summary.body.organism)" severity="secondary" />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Componenta</div>
                          <div class="mt-2 text-sm">
                            <div>Membri activi: <strong>{{ summary.membership.active_members }}</strong></div>
                            <div>Membri cu vot: <strong>{{ summary.membership.voting_members }}</strong></div>
                            <div>Presedinte acoperit: <strong>{{ summary.membership.chairperson_covered ? 'Da' : 'Nu' }}</strong></div>
                            <div>Secretar acoperit: <strong>{{ summary.membership.secretary_covered ? 'Da' : 'Nu' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Mandate</div>
                          <div class="mt-2 text-sm">
                            <div>Mandate expirate: <strong>{{ summary.membership.expired_mandates }}</strong></div>
                            <div>Pregatit operational: <strong>{{ summary.readiness.ready_for_operation ? 'Da' : 'Nu' }}</strong></div>
                            <div>Pregatit pentru sedinte: <strong>{{ summary.readiness.ready_for_meetings ? 'Da' : 'Nu' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Sedinte</div>
                          <div class="mt-2 text-sm">
                            <div>Total: <strong>{{ summary.meetings.total_meetings }}</strong></div>
                            <div>Programate: <strong>{{ summary.meetings.scheduled_meetings }}</strong></div>
                            <div>Tinute: <strong>{{ summary.meetings.held_meetings }}</strong></div>
                            <div>Publicate: <strong>{{ summary.meetings.published_meetings }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Membri activi</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.membership.member_names.length) {
                              <p-tag value="Nu exista membri activi" severity="warn" />
                            } @else {
                              @for (member of summary.membership.member_names; track member) {
                                <p-tag [value]="member" severity="secondary" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Ultima sedinta</div>
                          <div class="mt-2 text-sm">
                            <div><strong>{{ summary.meetings.last_meeting_title || 'Nu exista inca sedinta inregistrata' }}</strong></div>
                            <div class="text-muted-color">{{ summary.meetings.last_meeting_on || '-' }}</div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje de constituire" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="governanceBodyBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of governanceBodySummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                (onClick)="executeGovernanceBodySummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                      </div>
                    </section>
                  } @else if (committeeCompletenessSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Comisie institutionala</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Operabilitate, componenta si acoperire pentru comisii si pentru comisia de evaluare.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.ready_for_operation ? 'Comisie completa' : 'Necesita completari'"
                            [severity]="summary.readiness.ready_for_operation ? 'success' : 'warn'"
                          />
                          <p-tag [value]="displayCommitteeType(summary.committee.committee_type)" severity="secondary" />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Componenta</div>
                          <div class="mt-2 text-sm">
                            <div>Membri activi: <strong>{{ summary.membership.active_members }}</strong></div>
                            <div>Membri cu vot: <strong>{{ summary.membership.voting_members }}</strong></div>
                            <div>Presedinte acoperit: <strong>{{ summary.membership.chairperson_covered ? 'Da' : 'Nu' }}</strong></div>
                            <div>Secretar acoperit: <strong>{{ summary.membership.secretary_covered ? 'Da' : 'Nu' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Act administrativ</div>
                          <div class="mt-2 text-sm">
                            <div>Decizie: <strong>{{ summary.committee.decision_reference || '-' }}</strong></div>
                            <div>Inceput: <strong>{{ summary.committee.starts_on || '-' }}</strong></div>
                            <div>Final: <strong>{{ summary.committee.ends_on || '-' }}</strong></div>
                            <div>Evaluare: <strong>{{ summary.committee.evaluation_scope ? 'Da' : 'Nu' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Stare</div>
                          <div class="mt-2 text-sm">
                            <div>Status: <strong>{{ summary.committee.status }}</strong></div>
                            <div>Tip: <strong>{{ displayCommitteeType(summary.committee.committee_type) }}</strong></div>
                            <div>An scolar: <strong>{{ summary.committee.school_year }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Membri activi</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.membership.member_names.length) {
                              <p-tag value="Nu exista membri activi" severity="warn" />
                            } @else {
                              @for (member of summary.membership.member_names; track member) {
                                <p-tag [value]="member" severity="secondary" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje de constituire" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="committeeBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of committeeSummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                (onClick)="executeCommitteeSummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                      </div>
                    </section>
                  } @else if (managerialPortfolioSummary(); as summary) {
                    <section class="mb-4 rounded-2xl border border-surface bg-surface-0 p-4">
                      <div class="flex flex-wrap items-start justify-between gap-3">
                        <div class="space-y-1">
                          <h3 class="m-0 text-lg font-semibold">Portofoliu managerial</h3>
                          <p class="m-0 text-sm text-muted-color">
                            Vedere procedurala pentru portofoliul directorului si al directorului adjunct, corelata cu documentele manageriale si dosarul personal.
                          </p>
                        </div>
                        <div class="flex flex-wrap items-center gap-2">
                          <p-tag
                            [value]="summary.readiness.blockers.length ? 'Necesita completari' : 'Portofoliu coerent'"
                            [severity]="summary.readiness.blockers.length ? 'warn' : 'success'"
                          />
                          <p-tag [value]="displayManagerialDossierType(summary.dossier.dossier_type)" severity="secondary" />
                        </div>
                      </div>

                      <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Dosar managerial</div>
                          <div class="mt-2 text-sm">
                            <div>Cod: <strong>{{ summary.dossier.dossier_code }}</strong></div>
                            <div>Status: <strong>{{ summary.dossier.status }}</strong></div>
                            <div>Responsabil: <strong>{{ summary.dossier.owner_name || '-' }}</strong></div>
                            <div>Publicare necesara: <strong>{{ summary.dossier.publication_required ? 'Da' : 'Nu' }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Documente-model</div>
                          <div class="mt-2 text-sm">
                            <div>Documente totale: <strong>{{ summary.portfolio.managerial_documents }}</strong></div>
                            <div>Obligatorii: <strong>{{ summary.portfolio.mandatory_documents }}</strong></div>
                            <div>Aprobate: <strong>{{ summary.portfolio.approved_documents }}</strong></div>
                            <div>Publicate: <strong>{{ summary.portfolio.published_documents }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Legatura cu dosarul personal</div>
                          <div class="mt-2 text-sm">
                            <div>Persoane potrivite: <strong>{{ summary.portfolio.matched_personnel }}</strong></div>
                            <div>Documente potrivite: <strong>{{ summary.personnel_file.matched_documents }}</strong></div>
                            <div>Documente management: <strong>{{ summary.personnel_file.management_documents }}</strong></div>
                            <div>Referinte comune: <strong>{{ summary.personnel_file.mirrored_references }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Flux procedural</div>
                          <div class="mt-2 text-sm">
                            <div>Pasi totali: <strong>{{ summary.workflow.total_steps }}</strong></div>
                            <div>Finalizati: <strong>{{ summary.workflow.completed_steps }}</strong></div>
                            <div>Deschisi: <strong>{{ summary.workflow.open_steps }}</strong></div>
                            <div>Cu semnatura: <strong>{{ summary.workflow.signature_steps }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Readiness</div>
                          <div class="mt-2 text-sm">
                            <div>Pentru avizare: <strong>{{ summary.readiness.ready_for_review ? 'Da' : 'Nu' }}</strong></div>
                            <div>Pentru publicare: <strong>{{ summary.readiness.ready_for_publication ? 'Da' : 'Nu' }}</strong></div>
                            <div>Documente cu publicare: <strong>{{ summary.portfolio.publication_required_documents }}</strong></div>
                            <div>Semnaturi finalizate: <strong>{{ summary.workflow.completed_signature_steps }}</strong></div>
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Categorii lipsa</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.portfolio.missing_mandatory_categories.length) {
                              <p-tag value="Documentele de baza sunt prezente" severity="success" />
                            } @else {
                              @for (category of summary.portfolio.missing_mandatory_categories; track category) {
                                <p-tag [value]="managerialMissingCategoryLabel(category)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Blocaje curente</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @if (!summary.readiness.blockers.length) {
                              <p-tag value="Nu exista blocaje manageriale" severity="success" />
                            } @else {
                              @for (blocker of summary.readiness.blockers; track blocker) {
                                <p-tag [value]="managerialPortfolioBlockerLabel(blocker)" severity="warn" />
                              }
                            }
                          </div>
                        </div>
                        <div class="rounded-xl border border-surface p-3 md:col-span-2 xl:col-span-3">
                          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Actiuni recomandate</div>
                          <div class="mt-3 flex flex-wrap gap-2">
                            @for (action of managerialPortfolioSummaryActions(summary); track action.id) {
                              <p-button
                                [label]="action.label"
                                [icon]="action.icon"
                                size="small"
                                severity="secondary"
                                [outlined]="true"
                                (onClick)="executeManagerialPortfolioSummaryAction(action)"
                              />
                            }
                          </div>
                        </div>
                      </div>
                    </section>
                  }

                  <div class="grid gap-3 md:grid-cols-2">
                    @for (column of resource.columns; track column.field) {
                      <div class="rounded-xl border border-surface p-3">
                        <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">{{ column.header }}</div>
                        <div class="mt-1 font-medium">{{ displayConfiguredCell(resource.columns, row, column.field) }}</div>
                      </div>
                    }
                  </div>
                </p-tabpanel>

                @for (child of resource.detailChildren ?? []; track child.key) {
                  <p-tabpanel [value]="child.key" class="flex min-h-0 flex-1 overflow-hidden p-0 pt-4">
                    <div class="min-h-0 flex-1 overflow-hidden">
                      <app-server-table
                        class="h-full"
                        [columns]="tableColumns(child.columns)"
                        [rows]="childRows(child, row)"
                        [total]="childTotal(child, row)"
                        [pageIndex]="childQuery(child, row).page - 1"
                        [pageSize]="childQuery(child, row).pageSize"
                        [loading]="childLoadingKey() === childStoreKey(child, row)"
                        [emptyMessage]="child.emptyText"
                        [sortActive]="childQuery(child, row).sort || ''"
                        [sortDirection]="childQuery(child, row).direction || ''"
                        [rowActions]="childRowActions(child)"
                        actionColumnLabel="Actiuni"
                        (pageChange)="onChildPageChange(child, row, $event)"
                        (sortChange)="onChildSortChange(child, row, $event)"
                        (filterChange)="onChildFilterChange(child, row, $event)"
                        (actionClick)="onChildAction(resource, child, row, $event.action, $event.row)"
                      >
                        <ng-template #serverTableToolbar>
                          <div class="flex flex-wrap items-start justify-between gap-3">
                            <div class="space-y-1">
                              <h3 class="m-0 text-lg font-semibold">{{ child.label }}</h3>
                              <p class="m-0 text-sm text-muted-color">{{ child.description }}</p>
                            </div>
                            <div class="flex flex-wrap items-center gap-2">
                              @if (canRegeneratePortfolioOpis(resource, child)) {
                                <p-button
                                  label="Regenereaza opis"
                                  icon="pi pi-refresh"
                                  size="small"
                                  severity="secondary"
                                  [outlined]="true"
                                  [loading]="portfolioOpisSyncingKey() === childStoreKey(child, row)"
                                  (onClick)="regeneratePortfolioOpis(row, child)"
                                />
                              }
                              <p-tag [value]="summarizeRow(row)" severity="secondary" />
                            </div>
                          </div>
                        </ng-template>

                        <ng-template #serverTableActionHeader>
                          @if (canCreateChild(child)) {
                            <p-button
                              icon="pi pi-plus"
                              size="small"
                              [text]="true"
                              [rounded]="true"
                              [attr.aria-label]="'Adauga ' + child.label"
                              (onClick)="openCreateChild(resource, child, row)"
                            />
                          }
                        </ng-template>
                      </app-server-table>
                    </div>
                  </p-tabpanel>
                }
              </p-tabpanels>
            </p-tabs>
          }
        }
      </p-dialog>

      <p-dialog
        [visible]="deleteDialogOpen()"
        (visibleChange)="onDeleteDialogVisibilityChange($event)"
        [modal]="true"
        [draggable]="false"
        header="Confirmare stergere"
        [style]="{ width: 'min(32rem, 92vw)' }"
      >
        <div class="space-y-3">
          <p class="m-0 text-sm text-muted-color">
            @if (deleteTarget(); as target) {
              Confirmati stergerea inregistrarii din {{ deleteTargetLabel(target).toLowerCase() }}?
            } @else {
              Confirmati stergerea inregistrarii selectate?
            }
          </p>
          @if (deleteTarget(); as target) {
            <div class="rounded-2xl border border-surface p-3">
              <div class="font-medium">{{ summarizeRow(target.row) }}</div>
            </div>
          }
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunta" severity="secondary" [outlined]="true" (onClick)="closeDeleteDialog()" />
            <p-button label="Sterge" icon="pi pi-trash" severity="danger" [loading]="deleting()" (onClick)="deleteRecord()" />
          </div>
        </ng-template>
      </p-dialog>
    </section>
  `,
  styles: `
    :host {
      display: block;
      min-height: 0;
      height: 100%;
    }

    :host ::ng-deep .p-tabs,
    :host ::ng-deep .p-tabpanels,
    :host ::ng-deep .p-tabpanel {
      display: flex;
      flex: 1 1 auto;
      min-height: 0;
      flex-direction: column;
      overflow: hidden;
      background: transparent;
    }

    .education-field {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
      color: var(--p-text-muted-color);
      font-size: 0.875rem;
      font-weight: 600;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationDomainWorkspaceComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly messages = inject(MessageService);
  private readonly authz = inject(AuthzService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private detailRequestSequence = 0;

  readonly section = input.required<EducationSectionConfig>();
  protected readonly visibleResources = computed(() =>
    this.section().resources.filter((resource) => this.authz.hasPermission(resource.readPermission)),
  );

  protected readonly rowStore = signal<Record<string, EducationRow[]>>({});
  protected readonly totalStore = signal<Record<string, number>>({});
  protected readonly queryStore = signal<Record<string, TableQuery>>({});
  protected readonly childRowStore = signal<Record<string, EducationRow[]>>({});
  protected readonly childTotalStore = signal<Record<string, number>>({});
  protected readonly childQueryStore = signal<Record<string, TableQuery>>({});
  protected readonly loadingKey = signal('');
  protected readonly childLoadingKey = signal('');
  protected readonly activeResourceKey = signal('');
  protected readonly editorDialogOpen = signal(false);
  protected readonly detailDialogOpen = signal(false);
  protected readonly deleteDialogOpen = signal(false);
  protected readonly creating = signal(false);
  protected readonly deleting = signal(false);
  protected readonly detailLoading = signal(false);
  protected readonly detailSummaryLoading = signal(false);
  protected readonly exportingKey = signal('');
  protected readonly portfolioOpisSyncingKey = signal('');
  protected readonly portfolioTransferActionLoading = signal('');
  protected readonly governanceTransitionLoading = signal<'held' | 'published' | ''>('');
  protected readonly governanceSummaryExporting = signal(false);
  protected readonly editorMode = signal<'create' | 'edit'>('create');
  protected readonly editingRecordID = signal('');
  protected readonly activeDetailTab = signal('overview');
  protected readonly activePresetLabel = signal('');
  protected readonly selectedResource = signal<EducationResourceConfig | null>(null);
  protected readonly selectedRow = signal<EducationRow | null>(null);
  protected readonly governanceFinalizationSummary = signal<GovernanceMeetingFinalizationSummary | null>(null);
  protected readonly personnelPortfolioDossierSummary = signal<PersonnelPortfolioDossierSummary | null>(null);
  protected readonly portfolioTransferSummary = signal<PortfolioTransferSummary | null>(null);
  protected readonly regulationProceduralSummary = signal<RegulationProceduralSummary | null>(null);
  protected readonly governanceBodyCompletenessSummary = signal<GovernanceBodyCompletenessSummary | null>(null);
  readonly committeeCompletenessSummary = signal<CommitteeCompletenessSummary | null>(null);
  readonly managerialPortfolioSummary = signal<ManagerialPortfolioSummary | null>(null);
  protected readonly editorTarget = signal<EditorTarget | null>(null);
  protected readonly deleteTarget = signal<DeleteTarget | null>(null);
  protected readonly createForm: EducationCreateForm = {};

  ngOnInit(): void {
    const firstResource = this.visibleResources()[0];
    if (!firstResource) {
      return;
    }

    const requestedResourceKey = this.route.snapshot.queryParamMap.get('resource');
    const initialResource = this.visibleResources().find((resource) => resource.key === requestedResourceKey) ?? firstResource;
    const initialQuery = this.queryFromRoute(initialResource);
    this.activePresetLabel.set(this.route.snapshot.queryParamMap.get('presetLabel') ?? '');

    this.activeResourceKey.set(initialResource.key);
    this.loadResource(initialResource, initialQuery);
  }

  protected activateResource(key: string): void {
    const resource = this.visibleResources().find((candidate) => candidate.key === key);
    if (!resource) {
      return;
    }

    this.activeResourceKey.set(resource.key);
    if (this.activePresetLabel()) {
      this.activePresetLabel.set('');
      void this.router.navigate([], {
        relativeTo: this.route,
        queryParams: {
          resource: resource.key,
          filter_status: null,
          filter_stage: null,
          filter_transfer_status: null,
          filter_evaluation_status: null,
          presetLabel: null,
          page: null,
          pageSize: null,
          sort: null,
          direction: null,
        },
        queryParamsHandling: 'merge',
      });
    }
    if (!(resource.key in this.rowStore())) {
      this.loadResource(resource, this.query(resource));
    }
  }

  protected rows(resource: EducationResourceConfig): EducationRow[] {
    return this.rowStore()[resource.key] ?? [];
  }

  protected total(resource: EducationResourceConfig): number {
    return this.totalStore()[resource.key] ?? 0;
  }

  protected query(resource: EducationResourceConfig): TableQuery {
    return this.queryStore()[resource.key] ?? {
      page: 1,
      pageSize: 20,
      sort: resource.columns[0]?.field,
      direction: 'asc',
      filters: {},
    };
  }

  protected childRows(child: EducationDetailChildResourceConfig, parentRow: EducationRow): EducationRow[] {
    return this.childRowStore()[this.childStoreKey(child, parentRow)] ?? [];
  }

  protected childTotal(child: EducationDetailChildResourceConfig, parentRow: EducationRow): number {
    return this.childTotalStore()[this.childStoreKey(child, parentRow)] ?? 0;
  }

  protected childQuery(child: EducationDetailChildResourceConfig, parentRow: EducationRow): TableQuery {
    return this.childQueryStore()[this.childStoreKey(child, parentRow)] ?? {
      page: 1,
      pageSize: 10,
      sort: child.columns[0]?.field,
      direction: 'asc',
      filters: {},
    };
  }

  protected tableColumns(columns: EducationColumn[]): EducationServerTableColumn[] {
    return columns.map((column) => ({
      key: column.field,
      label: column.header,
      kind: column.type === 'tag' ? 'tag' : column.type === 'boolean' ? 'boolean' : column.type === 'number' ? 'number' : 'text',
      sortable: column.sortable,
      sortKey: column.field,
      filterKey: column.field,
      width: column.width,
      formatter: (row) => this.formatConfiguredValue(column, row[column.field]),
      tagSeverity: (row) => this.tagSeverity(row[column.field]),
      filter: column.filter === 'text'
        ? { type: 'text', placeholder: column.header }
        : column.filter === 'select'
          ? {
              type: 'select',
              placeholder: 'Toate',
              options: (column.options ?? []).map((option) => ({
                label: option.label,
                value: String(option.value),
              })),
            }
          : undefined,
    }));
  }

  protected rowActions(resource: EducationResourceConfig): ServerTableRowAction<EducationRow>[] {
    const actions: ServerTableRowAction<EducationRow>[] = [{ key: 'view', icon: 'visibility', label: 'Deschide' }];
    if (resource.pdfEndpoint) {
      actions.push({
        key: 'pdf',
        icon: 'pi pi-file-pdf',
        label: resource.pdfActionLabel ?? 'Descarca PDF',
      });
    }
    if (this.canManageResource(resource)) {
      actions.push(
        { key: 'edit', icon: 'edit', label: 'Editeaza' },
        { key: 'delete', icon: 'delete', label: 'Sterge' },
      );
    }
    return actions;
  }

  protected childRowActions(child: EducationDetailChildResourceConfig): ServerTableRowAction<EducationRow>[] {
    const actions: ServerTableRowAction<EducationRow>[] = [];
    if (child.pdfEndpoint) {
      actions.push({
        key: 'pdf',
        icon: 'pi pi-file-pdf',
        label: child.pdfActionLabel ?? 'Descarca PDF',
      });
    }
    if (this.canManageChild(child)) {
      actions.push(
        { key: 'edit', icon: 'edit', label: 'Editeaza' },
        { key: 'delete', icon: 'delete', label: 'Sterge' },
      );
    }
    return actions;
  }

  protected canCreateResource(resource: EducationResourceConfig): boolean {
    return resource.allowCreate !== false
      && !!resource.createEndpoint
      && (!!resource.managePermission ? this.authz.hasPermission(resource.managePermission) : true);
  }

  protected canManageResource(resource: EducationResourceConfig): boolean {
    return !!resource.createEndpoint
      && !!resource.managePermission
      && this.authz.hasPermission(resource.managePermission);
  }

  protected canCreateChild(child: EducationDetailChildResourceConfig): boolean {
    return child.allowCreate !== false
      && (!!child.managePermission ? this.authz.hasPermission(child.managePermission) : true);
  }

  protected canManageChild(child: EducationDetailChildResourceConfig): boolean {
    return !!child.managePermission && this.authz.hasPermission(child.managePermission);
  }

  protected canRegeneratePortfolioOpis(resource: EducationResourceConfig, child: EducationDetailChildResourceConfig): boolean {
    return resource.key === 'portfolios'
      && child.key === 'portfolio_opis'
      && this.canManageChild(child);
  }

  protected onPageChange(resource: EducationResourceConfig, event: PageEvent): void {
    const nextQuery: TableQuery = {
      ...this.query(resource),
      page: event.pageIndex + 1,
      pageSize: event.pageSize,
    };
    this.loadResource(resource, nextQuery);
  }

  protected onSortChange(resource: EducationResourceConfig, sort: ServerTableSortState): void {
    const current = this.query(resource);
    this.loadResource(resource, {
      ...current,
      page: 1,
      sort: sort.active || current.sort,
      direction: sort.direction || current.direction,
    });
  }

  protected onFilterChange(resource: EducationResourceConfig, filters: ServerTableFilterState): void {
    this.loadResource(resource, {
      ...this.query(resource),
      page: 1,
      filters: this.normalizeFilters(filters),
    });
  }

  protected onChildPageChange(child: EducationDetailChildResourceConfig, parentRow: EducationRow, event: PageEvent): void {
    this.loadChildResource(child, parentRow, {
      ...this.childQuery(child, parentRow),
      page: event.pageIndex + 1,
      pageSize: event.pageSize,
    });
  }

  protected onChildSortChange(child: EducationDetailChildResourceConfig, parentRow: EducationRow, sort: ServerTableSortState): void {
    const current = this.childQuery(child, parentRow);
    this.loadChildResource(child, parentRow, {
      ...current,
      page: 1,
      sort: sort.active || current.sort,
      direction: sort.direction || current.direction,
    });
  }

  protected onChildFilterChange(child: EducationDetailChildResourceConfig, parentRow: EducationRow, filters: ServerTableFilterState): void {
    this.loadChildResource(child, parentRow, {
      ...this.childQuery(child, parentRow),
      page: 1,
      filters: this.normalizeFilters(filters),
    });
  }

  protected openCreateResource(resource: EducationResourceConfig): void {
    if (resource.createWizardRoute) {
      void this.router.navigateByUrl(resource.createWizardRoute);
      return;
    }
    this.editorMode.set('create');
    this.editingRecordID.set('');
    this.editorTarget.set({ kind: 'resource', resource });
    this.seedCreateForm(resource.createFields);
    this.editorDialogOpen.set(true);
  }

  protected openCreateChild(resource: EducationResourceConfig, child: EducationDetailChildResourceConfig, parentRow: EducationRow): void {
    if (child.createWizardRoute) {
      void this.router.navigateByUrl(child.createWizardRoute(parentRow));
      return;
    }
    this.editorMode.set('create');
    this.editingRecordID.set('');
    this.editorTarget.set({ kind: 'child', resource, child, parent: parentRow });
    this.seedCreateForm(child.createFields);
    this.editorDialogOpen.set(true);
  }

  protected saveRecord(): void {
    const target = this.editorTarget();
    const editingRecordID = this.editingRecordID();
    if (!target) {
      return;
    }

    const payload = Object.fromEntries(
      Object.entries(this.createForm).map(([key, value]) => [key, this.normalizeValue(value)]),
    );

    const request = target.kind === 'resource'
      ? editingRecordID
        ? this.http.patch(`${target.resource.endpoint}/${editingRecordID}`, payload)
        : this.http.post(target.resource.createEndpoint, payload)
      : editingRecordID
        ? this.http.patch(this.childDetailEndpoint(target.child, target.parent, { id: editingRecordID }), payload)
        : this.http.post(target.child.createEndpoint(target.parent), payload);

    this.creating.set(true);
    request.subscribe({
      next: () => {
        this.creating.set(false);
        const successLabel = this.editorTargetLabel(target);
        this.closeEditorDialog();
        this.messages.add({
          severity: 'success',
          summary: editingRecordID ? 'Inregistrare actualizata' : 'Inregistrare adaugata',
          detail: `${successLabel} a fost salvat.`,
        });

        if (target.kind === 'resource') {
          this.loadResource(target.resource, { ...this.query(target.resource), page: 1 });
        } else {
          this.loadChildResource(target.child, target.parent, { ...this.childQuery(target.child, target.parent), page: 1 });
        }
      },
      error: () => {
        this.creating.set(false);
        this.messages.add({
          severity: 'error',
          summary: 'Salvarea a esuat',
          detail: 'Datele nu au putut fi salvate cu configuratia actuala a backendului.',
        });
      },
    });
  }

  protected onAction(resource: EducationResourceConfig, action: string, row: EducationRow): void {
    if (action === 'view') {
      this.openDetail(resource, row);
      return;
    }
    if (action === 'pdf') {
      this.downloadResourcePdf(resource, row);
      return;
    }
    if (action === 'edit') {
      this.editorMode.set('edit');
      this.editingRecordID.set(String(row['id'] ?? '').trim());
      this.editorTarget.set({ kind: 'resource', resource });
      this.seedCreateForm(resource.createFields, row);
      this.editorDialogOpen.set(true);
      return;
    }
    if (action === 'delete') {
      this.deleteTarget.set({ kind: 'resource', resource, row });
      this.deleteDialogOpen.set(true);
    }
  }

  protected onChildAction(resource: EducationResourceConfig, child: EducationDetailChildResourceConfig, parentRow: EducationRow, action: string, row: EducationRow): void {
    if (action === 'pdf') {
      this.downloadChildPdf(child, parentRow, row);
      return;
    }
    if (action === 'edit') {
      this.editorMode.set('edit');
      this.editingRecordID.set(String(row['id'] ?? '').trim());
      this.editorTarget.set({ kind: 'child', resource, child, parent: parentRow });
      this.seedCreateForm(child.createFields, row);
      this.editorDialogOpen.set(true);
      return;
    }
    if (action === 'delete') {
      this.deleteTarget.set({ kind: 'child', resource, child, parent: parentRow, row });
      this.deleteDialogOpen.set(true);
    }
  }

  protected regeneratePortfolioOpis(parentRow: EducationRow, child: EducationDetailChildResourceConfig): void {
    const recordID = String(parentRow['id'] ?? '').trim();
    if (!recordID) {
      return;
    }

    const syncKey = this.childStoreKey(child, parentRow);
    this.portfolioOpisSyncingKey.set(syncKey);
    this.http.post(`/api/education/portfolios/records/${recordID}/opis/regenerate`, {}).subscribe({
      next: () => {
        this.portfolioOpisSyncingKey.set('');
        this.loadChildResource(child, parentRow, { ...this.childQuery(child, parentRow), page: 1 });
        this.messages.add({
          severity: 'success',
          summary: 'Opis regenerat',
          detail: 'Opisul a fost reconstruit automat din documentele portofoliului.',
        });
      },
      error: () => {
        this.portfolioOpisSyncingKey.set('');
        this.messages.add({
          severity: 'error',
          summary: 'Regenerarea a esuat',
          detail: 'Opisul nu a putut fi regenerat cu configuratia actuala a backendului.',
        });
      },
    });
  }

  protected displayConfiguredCell(columns: EducationColumn[], row: EducationRow, field: string): string {
    const column = columns.find((candidate) => candidate.field === field);
    if (!column) {
      return this.formatCell(row[field]);
    }
    return this.formatConfiguredValue(column, row[field]);
  }

  protected activateDetailTab(value: string): void {
    this.activeDetailTab.set(value);
    if (value === 'overview') {
      return;
    }

    const resource = this.selectedResource();
    const row = this.selectedRow();
    const child = resource?.detailChildren?.find((candidate) => candidate.key === value);
    if (!child || !row) {
      return;
    }
    this.ensureChildLoaded(child, row);
  }

  protected onDetailDialogVisibilityChange(visible: boolean): void {
    this.detailDialogOpen.set(visible);
    if (!visible) {
      this.detailRequestSequence += 1;
      this.detailLoading.set(false);
      this.detailSummaryLoading.set(false);
      this.governanceFinalizationSummary.set(null);
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      this.portfolioTransferActionLoading.set('');
      this.selectedRow.set(null);
      this.selectedResource.set(null);
      this.activeDetailTab.set('overview');
    }
  }

  protected onEditorDialogVisibilityChange(visible: boolean): void {
    if (!visible) {
      this.closeEditorDialog();
      return;
    }
    this.editorDialogOpen.set(true);
  }

  protected editorDialogHeader(): string {
    const target = this.editorTarget();
    if (!target) {
      return this.editorMode() === 'edit' ? 'Editeaza' : 'Adauga';
    }
    const verb = this.editorMode() === 'edit' ? 'Editeaza' : 'Adauga';
    return `${verb} ${this.editorTargetLabel(target)}`;
  }

  protected editorFields() {
    const target = this.editorTarget();
    if (!target) {
      return [];
    }
    return target.kind === 'resource' ? target.resource.createFields : target.child.createFields;
  }

  protected fieldOptions(field: EducationColumn | EducationResourceConfig['createFields'][number]) {
    if (Array.isArray(field.options)) {
      return field.options;
    }
    if (typeof field.options !== 'function') {
      return [];
    }

    const target = this.editorTarget();
    if (!target) {
      return [];
    }

    return field.options({
      resource: target.resource,
      child: target.kind === 'child' ? target.child : undefined,
      parentRow: target.kind === 'child' ? target.parent : undefined,
      childRows: (child, parentRow) => this.childRows(child, parentRow),
    });
  }

  protected onDeleteDialogVisibilityChange(visible: boolean): void {
    if (!visible) {
      this.closeDeleteDialog();
      return;
    }
    this.deleteDialogOpen.set(true);
  }

  protected deleteTargetLabel(target: DeleteTarget): string {
    return target.kind === 'resource' ? target.resource.label : target.child.label;
  }

  protected deleteRecord(): void {
    const target = this.deleteTarget();
    const recordID = String(target?.row['id'] ?? '').trim();
    if (!target || !recordID) {
      this.closeDeleteDialog();
      return;
    }

    const request = target.kind === 'resource'
      ? this.http.delete(`${target.resource.endpoint}/${recordID}`)
      : this.http.delete(this.childDetailEndpoint(target.child, target.parent, target.row));

    this.deleting.set(true);
    request.subscribe({
      next: () => {
        this.deleting.set(false);
        this.closeDeleteDialog();
        this.messages.add({
          severity: 'success',
          summary: 'Inregistrare stearsa',
          detail: `${this.deleteTargetLabel(target)} a fost stearsa.`,
        });

        if (target.kind === 'resource') {
          this.loadResource(target.resource, this.query(target.resource));
        } else {
          this.loadChildResource(target.child, target.parent, this.childQuery(target.child, target.parent));
        }
      },
      error: () => {
        this.deleting.set(false);
        this.messages.add({
          severity: 'error',
          summary: 'Stergerea a esuat',
          detail: 'Inregistrarea nu a putut fi stearsa cu configuratia actuala a backendului.',
        });
      },
    });
  }

  protected summarizeRow(row: EducationRow): string {
    return String(
      row['title']
      ?? row['review_code']
      ?? row['criterion_code']
      ?? row['resolution_code']
      ?? row['transfer_code']
      ?? row['publication_code']
      ?? row['issuance_code']
      ?? row['assignment_code']
      ?? row['issue_code']
      ?? row['document_code']
      ?? row['version_label']
      ?? row['appeal_code']
      ?? row['criterion_label']
      ?? row['requirement_label']
      ?? row['entry_title']
      ?? row['topic_title']
      ?? row['entity_label']
      ?? row['publication_reference']
      ?? row['document_title']
      ?? row['responsible_name']
      ?? row['actor_name']
      ?? row['holder_name']
      ?? row['full_name']
      ?? row['owner_name']
      ?? row['panel_name']
      ?? row['decision_code']
      ?? row['dossier_code']
      ?? row['portfolio_code']
      ?? row['employee_code']
      ?? row['case_code']
      ?? row['grant_code']
      ?? row['evaluation_code']
      ?? row['declaration_code']
      ?? row['document_number']
      ?? row['id']
      ?? '-',
    );
  }

  protected governanceBlockerLabel(blocker: string): string {
    switch (blocker) {
      case 'quorum_incomplete':
        return 'Cvorum incomplet';
      case 'signatures_missing':
        return 'Semnaturi lipsa';
      case 'votes_missing':
        return 'Lipsesc voturile';
      case 'minutes_missing':
        return 'Lipsesc minutele';
      case 'resolutions_missing_for_adopted_votes':
        return 'Lipsesc hotarari pentru voturile adoptate';
      case 'resolution_anonymization_pending':
        return 'Anonimizarea hotararilor este in asteptare';
      case 'published_minute_document_missing':
        return 'Lipseste procesul-verbal publicat';
      default:
        return blocker;
    }
  }

  protected personnelPortfolioBlockerLabel(blocker: string): string {
    switch (blocker) {
      case 'dosar_personal_fara_documente':
        return 'Dosarul personal nu are documente';
      case 'portofoliu_lipsa_pentru_cadrul_marcat_cu_portofoliu':
        return 'Cadrul este marcat cu portofoliu, dar nu exista registru asociat';
      case 'documente_din_dosar_marcate_pentru_portofoliu_dar_neregasite_in_portofoliu':
        return 'Documentele marcate pentru portofoliu nu apar in registrul portofoliului';
      case 'documente_cu_sursa_dosar_personal_fara_referinta_comuna':
        return 'Exista documente cu sursa dosar personal fara referinta comuna';
      default:
        return blocker;
    }
  }

  protected portfolioTransferBlockerLabel(blocker: string): string {
    switch (blocker) {
      case 'mobilitate_activa_fara_transfer_portofoliu':
        return 'Exista mobilitate activa fara transfer de portofoliu';
      case 'status_portofoliu_fara_eveniment_transfer':
        return 'Statusul de transfer din portofoliu nu are eveniment suport';
      case 'institutie_sursa_si_destinatie_identice':
        return 'Institutia sursa si destinatie sunt identice';
      case 'detasare_fara_circuit_digital_portofoliu':
        return 'Exista detasare fara circuit digital al portofoliului';
      case 'evaluare_fara_flux_valorificare_portofoliu':
        return 'Exista evaluari corelate fara flux de valorificare documentat';
      case 'mobilitate_fara_flux_valorificare_portofoliu':
        return 'Exista mobilitate corelata fara flux de valorificare documentat';
      case 'gradatie_sau_distinctie_fara_flux_valorificare_portofoliu':
        return 'Exista gradatie sau distinctie corelata fara flux de valorificare documentat';
      case 'transfer_fara_fluxuri_de_valorificare_documentate':
        return 'Exista transferuri, dar nu sunt documentate fluxuri de valorificare';
      default:
        return blocker;
    }
  }

  protected regulationBlockerLabel(blocker: string): string {
    switch (blocker) {
      case 'regulation_version_missing':
        return 'Lipseste o versiune de regulament';
      case 'public_consultation_missing':
        return 'Lipseste consultarea publica';
      case 'cp_endorsement_missing':
        return 'Lipseste avizarea in CP';
      case 'ca_approval_missing':
        return 'Lipseste aprobarea in CA';
      case 'registration_missing':
        return 'Lipseste inregistrarea formei aprobate';
      case 'publication_missing':
        return 'Lipseste publicarea regulamentului';
      case 'review_overdue':
        return 'Revizuirea regulamentului este depasita';
      default:
        return blocker;
    }
  }

  protected governanceBodyBlockerLabel(blocker: string): string {
    switch (blocker) {
      case 'no_active_members':
        return 'Nu exista membri activi';
      case 'no_voting_members':
        return 'Nu exista membri cu drept de vot';
      case 'chairperson_missing':
        return 'Lipseste presedintele organismului';
      case 'secretary_missing':
        return 'Lipseste secretarul organismului';
      case 'expired_mandates':
        return 'Exista mandate expirate';
      case 'no_meeting_recorded':
        return 'Nu exista sedinte inregistrate';
      case 'no_held_meeting':
        return 'Nu exista sedinte tinute';
      default:
        return blocker;
    }
  }

  committeeBlockerLabel(blocker: string): string {
    switch (blocker) {
      case 'no_active_members':
        return 'Nu exista membri activi';
      case 'chairperson_missing':
        return 'Lipseste presedintele comisiei';
      case 'secretary_missing':
        return 'Lipseste secretarul comisiei';
      case 'decision_reference_missing':
        return 'Lipseste decizia de constituire';
      case 'evaluation_scope_missing':
        return 'Nu este marcat scopul de evaluare';
      default:
        return blocker;
    }
  }

  readonly managerialPortfolioBlockerLabel = (blocker: string): string => {
    switch (blocker) {
      case 'personnel_link_missing':
        return 'Nu exista legatura cu persoana din dosarul de personal';
      case 'managerial_documents_missing':
        return 'Nu exista documente manageriale in portofoliu';
      case 'mandatory_categories_missing':
        return 'Lipsesc categorii obligatorii de documente-model';
      case 'mandatory_documents_pending':
        return 'Documentele obligatorii nu sunt toate aprobate';
      case 'workflow_missing':
        return 'Lipseste circuitul procedural';
      case 'personnel_file_documents_missing':
        return 'Nu exista documente in dosarul personal corelat';
      default:
        return blocker;
    }
  };

  readonly managerialMissingCategoryLabel = (value: string): string => {
    switch (value) {
      case 'opis_evidenta':
        return 'Lipseste opisul / evidenta portofoliului';
      case 'documente_baza_manageriale':
        return 'Lipsesc documentele manageriale de baza';
      default:
        return value;
    }
  };

  protected displayTransferStatus(value: string): string {
    switch (value) {
      case 'none':
        return 'Fara transfer';
      case 'prepared':
        return 'Pregatit';
      case 'sent':
        return 'Trimis';
      case 'received':
        return 'Receptionat';
      default:
        return value || '-';
    }
  }

  protected displayPortfolioTransferStatus(value: string): string {
    switch (value) {
      case 'pregatit':
        return 'Pregatit';
      case 'trimis':
        return 'Trimis';
      case 'receptionat':
        return 'Receptionat';
      case 'inchis':
        return 'Inchis';
      default:
        return value || '-';
    }
  }

  protected displayPortfolioValorificationScope(value: string): string {
    switch (value) {
      case 'licentiere':
        return 'Licentiere';
      case 'debut':
        return 'Debut';
      case 'definitivat':
        return 'Definitivat';
      case 'grad_ii':
        return 'Grad didactic II';
      case 'grad_i':
        return 'Grad didactic I';
      case 'evaluare_profesionala':
        return 'Evaluare profesionala anuala';
      case 'mobilitate':
        return 'Mobilitate';
      case 'dezvoltare_profesionala':
        return 'Monitorizarea dezvoltarii profesionale';
      case 'inspectie_scolara':
        return 'Inspectie scolara';
      case 'evaluare_externa_calitate':
        return 'Evaluare externa a calitatii';
      case 'gradatie_merit':
        return 'Gradatie de merit';
      case 'distinctie_premiu':
        return 'Distinctie / premiu';
      default:
        return value || '-';
    }
  }

  protected displayPortfolioValorificationStatus(value: string): string {
    switch (value) {
      case 'planificat':
        return 'Planificat';
      case 'in_pregatire':
        return 'In pregatire';
      case 'transmis':
        return 'Transmis';
      case 'validat':
        return 'Validat';
      case 'finalizat':
        return 'Finalizat';
      default:
        return value || '-';
    }
  }

  protected displayRegulationStatus(value: string): string {
    switch (value) {
      case 'draft':
        return 'Draft';
      case 'consultation':
        return 'In consultare';
      case 'endorsed':
        return 'Avizat';
      case 'approved':
        return 'Aprobat';
      case 'published':
        return 'Publicat';
      default:
        return value || '-';
    }
  }

  protected displayRegulationVersionStatus(value: string): string {
    switch (value) {
      case 'draft':
        return 'Draft';
      case 'consultation':
        return 'In consultare';
      case 'endorsed':
        return 'Avizata in CP';
      case 'approved':
        return 'Aprobata';
      case 'published':
        return 'Publicata';
      case 'retired':
        return 'Retrasa';
      default:
        return value || '-';
    }
  }

  protected displayRegulationPhaseType(value: string): string {
    switch (value) {
      case 'redactare':
        return 'Redactare';
      case 'consultare_publica':
        return 'Consultare publica';
      case 'avizare_cp':
        return 'Avizare in CP';
      case 'aprobare_ca':
        return 'Aprobare in CA';
      case 'inregistrare':
        return 'Inregistrare';
      case 'publicare':
        return 'Publicare';
      default:
        return value || '-';
    }
  }

  protected displayRegulationWorkflowStatus(value: string): string {
    switch (value) {
      case 'pending':
        return 'In asteptare';
      case 'active':
        return 'Activa';
      case 'completed':
        return 'Finalizata';
      case 'returned':
        return 'Returnata';
      case 'cancelled':
        return 'Anulata';
      default:
        return value || '-';
    }
  }

  protected displayGovernanceOrganism(value: string): string {
    switch (value) {
      case 'ca':
        return 'Consiliul de Administratie';
      case 'cp':
        return 'Consiliul Profesoral';
      case 'ceac':
        return 'CEAC';
      case 'cfdcd':
        return 'CFDCD';
      default:
        return value || '-';
    }
  }

  displayCommitteeType(value: string): string {
    switch (value) {
      case 'evaluare_personal_didactic':
        return 'Evaluare personal didactic';
      case 'permanenta':
        return 'Permanenta';
      case 'temporara':
        return 'Temporara';
      case 'curriculum':
        return 'Curriculum';
      case 'mentorat':
        return 'Mentorat';
      case 'securitate':
        return 'Securitate';
      case 'burse':
        return 'Burse';
      case 'alta':
        return 'Alta';
      default:
        return value || '-';
    }
  }

  readonly displayManagerialDossierType = (value: string): string => {
    switch (value) {
      case 'director_portfolio':
        return 'Portofoliu director';
      case 'adjunct_director_portfolio':
        return 'Portofoliu director adjunct';
      case 'pdi_pas':
        return 'PDI / PAS';
      case 'annual_plan':
        return 'Plan managerial anual';
      case 'raei':
        return 'RAEI';
      case 'organigram':
        return 'Organigrama';
      case 'staffing_plan':
        return 'Plan de incadrare';
      case 'timetable':
        return 'Orar';
      case 'commission_report':
        return 'Raport de comisie';
      default:
        return value || '-';
    }
  };

  protected governanceSummaryActions(summary: GovernanceMeetingFinalizationSummary): GovernanceSummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'governance-finalization') {
      return [];
    }

    const actions: GovernanceSummaryAction[] = [];
    const pushUnique = (action: GovernanceSummaryAction) => {
      if (!actions.some((candidate) => candidate.id === action.id)) {
        actions.push(action);
      }
    };

    if (summary.participants.present_participants < Number(row['quorum_required'] ?? 0) || summary.participants.signed_participants < summary.participants.present_participants) {
      pushUnique({ id: 'participants-tab', label: 'Verifica participantii', icon: 'pi pi-users', targetTab: 'meeting_participants', kind: 'tab' });
    }
    if (summary.votes.total === 0) {
      pushUnique({ id: 'votes-create', label: 'Adauga vot', icon: 'pi pi-plus', targetTab: 'meeting_votes', kind: 'create' });
    } else {
      pushUnique({ id: 'votes-tab', label: 'Deschide voturile', icon: 'pi pi-check-square', targetTab: 'meeting_votes', kind: 'tab' });
    }
    if (summary.minutes.total === 0) {
      pushUnique({ id: 'minutes-create', label: 'Adauga minuta', icon: 'pi pi-plus', targetTab: 'meeting_minutes', kind: 'create' });
    } else {
      pushUnique({ id: 'minutes-tab', label: 'Deschide minutele', icon: 'pi pi-file-edit', targetTab: 'meeting_minutes', kind: 'tab' });
    }
    if (summary.votes.missing_resolutions > 0) {
      pushUnique({ id: 'resolutions-create', label: 'Adauga hotarare', icon: 'pi pi-plus', targetTab: 'meeting_resolutions', kind: 'create' });
    } else if (summary.resolutions.total > 0) {
      pushUnique({ id: 'resolutions-tab', label: 'Deschide hotararile', icon: 'pi pi-verified', targetTab: 'meeting_resolutions', kind: 'tab' });
    }
    if (summary.minutes.requires_publication > 0 || summary.documents.process_verbal_documents === 0) {
      pushUnique({ id: 'documents-tab', label: 'Verifica documentele', icon: 'pi pi-file', targetTab: 'meeting_documents', kind: 'tab' });
    }

    return actions;
  }

  protected personnelSummaryActions(summary: PersonnelPortfolioDossierSummary): PersonnelSummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'personnel-portfolio-dossier') {
      return [];
    }

    const actions: PersonnelSummaryAction[] = [];
    const pushUnique = (action: PersonnelSummaryAction) => {
      if (!actions.some((candidate) => candidate.id === action.id)) {
        actions.push(action);
      }
    };

    pushUnique({ id: 'file-documents', label: 'Deschide dosarul personal', icon: 'pi pi-folder-open', targetTab: 'personnel_file_documents' });
    if (summary.dossier.documents_marked_for_portfolio > 0 || summary.portfolio.personnel_scope_documents > 0) {
      pushUnique({ id: 'access-events', label: 'Verifica trasabilitatea', icon: 'pi pi-lock', targetTab: 'personnel_access_events' });
    }
    if (summary.portfolio.matched_records === 0 && Boolean(row['has_portfolio'])) {
      pushUnique({ id: 'assignments', label: 'Verifica datele cadrului', icon: 'pi pi-id-card', targetTab: 'personnel_assignments' });
    }

    return actions;
  }

  protected portfolioTransferSummaryActions(summary: PortfolioTransferSummary): PortfolioTransferSummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'portfolio-transfer-summary') {
      return [];
    }

    const actions: PortfolioTransferSummaryAction[] = [];
    const pushUnique = (action: PortfolioTransferSummaryAction) => {
      if (!actions.some((candidate) => candidate.id === action.id)) {
        actions.push(action);
      }
    };

    pushUnique({ id: 'transfers-tab', label: 'Deschide transferurile', icon: 'pi pi-send', kind: 'tab', targetTab: 'portfolio_transfers' });
    if (summary.completeness.total_documents === 0) {
      pushUnique({ id: 'documents-tab', label: 'Deschide documentele', icon: 'pi pi-file', kind: 'tab', targetTab: 'portfolio_documents' });
    }
    if (summary.completeness.opis_entries === 0 || summary.completeness.missing_checklist_items > 0) {
      pushUnique({ id: 'checklist-tab', label: 'Deschide checklistul', icon: 'pi pi-check-square', kind: 'tab', targetTab: 'portfolio_checklist' });
    }
    if (summary.completeness.opis_entries === 0) {
      pushUnique({ id: 'opis-tab', label: 'Deschide opisul', icon: 'pi pi-list', kind: 'tab', targetTab: 'portfolio_opis' });
    } else {
      pushUnique({
        id: 'opis-regenerate',
        label: 'Regenerare opis',
        icon: 'pi pi-refresh',
        kind: 'utility',
        targetTab: 'portfolio_opis',
        utilityAction: 'regenerate_opis',
      });
    }

    if (!summary.transfer.last_transfer) {
      pushUnique({ id: 'transfer-create', label: 'Initiaza transfer', icon: 'pi pi-plus', kind: 'create', targetTab: 'portfolio_transfers' });
      return actions;
    }

    const status = summary.transfer.last_transfer.status;
    if (status === 'pregatit') {
      pushUnique({ id: 'transfer-send', label: 'Marcheaza trimis', icon: 'pi pi-arrow-right', kind: 'advance', advanceAction: 'mark_sent' });
    }
    if (status === 'trimis') {
      pushUnique({ id: 'transfer-confirm', label: 'Confirma receptionarea', icon: 'pi pi-check-circle', kind: 'advance', advanceAction: 'confirm_received' });
    }
    if (status === 'receptionat') {
      pushUnique({ id: 'transfer-close', label: 'Inchide transferul', icon: 'pi pi-verified', kind: 'advance', advanceAction: 'close_transfer' });
    }

    return actions;
  }

  protected portfolioCompletenessBlockerLabel(blocker: string): string {
    const labels: Record<string, string> = {
      portfolio_documents_missing: 'Nu exista documente in portofoliu',
      opis_missing: 'Lipseste opisul portofoliului',
      mandatory_checklist_pending: 'Checklistul obligatoriu nu este complet',
      no_review_history: 'Nu exista inca o verificare institutionala',
    };
    return labels[blocker] ?? blocker;
  }

  protected portfolioValorificationSummaryActions(summary: PortfolioTransferSummary): PortfolioValorificationSummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'portfolio-transfer-summary') {
      return [];
    }

    const actions: PortfolioValorificationSummaryAction[] = [];
    const pushUnique = (action: PortfolioValorificationSummaryAction) => {
      if (!actions.some((candidate) => candidate.id === action.id)) {
        actions.push(action);
      }
    };

    pushUnique({ id: 'valorifications-tab', label: 'Deschide fluxurile', icon: 'pi pi-sitemap', kind: 'tab', targetTab: 'portfolio_valorifications' });
    if (!summary.valorification.total_events) {
      pushUnique({ id: 'valorifications-create', label: 'Initiaza flux', icon: 'pi pi-plus', kind: 'create', targetTab: 'portfolio_valorifications' });
    }
    if (summary.valorification.linked_evaluations > 0 && !summary.valorification.scopes.some((item) => item.scope === 'evaluare_profesionala')) {
      pushUnique({ id: 'valorifications-evaluation', label: 'Documenteaza evaluarea', icon: 'pi pi-briefcase', kind: 'create', targetTab: 'portfolio_valorifications' });
    }
    if (summary.valorification.linked_mobility > 0 && !summary.valorification.scopes.some((item) => item.scope === 'mobilitate')) {
      pushUnique({ id: 'valorifications-mobility', label: 'Documenteaza mobilitatea', icon: 'pi pi-send', kind: 'create', targetTab: 'portfolio_valorifications' });
    }
    if (summary.valorification.linked_merit > 0 && !summary.valorification.scopes.some((item) => item.scope === 'gradatie_merit' || item.scope === 'distinctie_premiu')) {
      pushUnique({ id: 'valorifications-merit', label: 'Documenteaza gradatia', icon: 'pi pi-star', kind: 'create', targetTab: 'portfolio_valorifications' });
    }

    return actions;
  }

  protected regulationSummaryActions(summary: RegulationProceduralSummary): RegulationSummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'regulation-procedural') {
      return [];
    }

    const actions: RegulationSummaryAction[] = [];
    const pushUnique = (action: RegulationSummaryAction) => {
      if (!actions.some((candidate) => candidate.id === action.id)) {
        actions.push(action);
      }
    };

    if (!summary.versions.total_versions) {
      pushUnique({ id: 'regulation-create-version', label: 'Adauga versiune', icon: 'pi pi-plus', kind: 'create', targetTab: 'regulation_versions' });
    } else {
      pushUnique({ id: 'regulation-open-versions', label: 'Deschide versiunile', icon: 'pi pi-copy', kind: 'tab', targetTab: 'regulation_versions' });
    }

    if (!summary.workflow.total_phases) {
      pushUnique({ id: 'regulation-create-workflow', label: 'Adauga faza', icon: 'pi pi-plus', kind: 'create', targetTab: 'regulation_workflow' });
    } else {
      pushUnique({ id: 'regulation-open-workflow', label: 'Deschide workflow-ul', icon: 'pi pi-sitemap', kind: 'tab', targetTab: 'regulation_workflow' });
    }

    if (summary.readiness.blockers.includes('public_consultation_missing')) {
      pushUnique({ id: 'regulation-consultation', label: 'Completeaza consultarea', icon: 'pi pi-comments', kind: 'tab', targetTab: 'regulation_workflow' });
    }
    if (summary.readiness.blockers.includes('cp_endorsement_missing')) {
      pushUnique({ id: 'regulation-cp', label: 'Documenteaza avizarea CP', icon: 'pi pi-users', kind: 'tab', targetTab: 'regulation_workflow' });
    }
    if (summary.readiness.blockers.includes('ca_approval_missing')) {
      pushUnique({ id: 'regulation-ca', label: 'Documenteaza aprobarea CA', icon: 'pi pi-verified', kind: 'tab', targetTab: 'regulation_workflow' });
    }

    return actions;
  }

  protected governanceBodySummaryActions(summary: GovernanceBodyCompletenessSummary): GovernanceBodySummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'governance-body-completeness') {
      return [];
    }

    const actions: GovernanceBodySummaryAction[] = [
      { id: 'body-members', label: 'Deschide membrii', icon: 'pi pi-users', targetTab: 'body_memberships' },
      { id: 'body-meetings', label: 'Deschide sedintele', icon: 'pi pi-calendar', targetTab: 'body_meetings' },
    ];
    if (summary.readiness.blockers.includes('chairperson_missing') || summary.readiness.blockers.includes('secretary_missing')) {
      actions.unshift({ id: 'body-members-priority', label: 'Verifica conducerea', icon: 'pi pi-sitemap', targetTab: 'body_memberships' });
    }
    return actions;
  }

  committeeSummaryActions(summary: CommitteeCompletenessSummary): CommitteeSummaryAction[] {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'committee-completeness') {
      return [];
    }

    const actions: CommitteeSummaryAction[] = [
      { id: 'committee-members', label: 'Deschide membrii', icon: 'pi pi-users', targetTab: 'committee_members' },
    ];
    if (summary.readiness.blockers.includes('chairperson_missing') || summary.readiness.blockers.includes('secretary_missing')) {
      actions.unshift({ id: 'committee-roles', label: 'Completeaza conducerea', icon: 'pi pi-sitemap', targetTab: 'committee_members' });
    }
    return actions;
  }

  readonly managerialPortfolioSummaryActions = (summary: ManagerialPortfolioSummary): ManagerialPortfolioSummaryAction[] => {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.detailSummaryKind !== 'managerial-portfolio') {
      return [];
    }

    const actions: ManagerialPortfolioSummaryAction[] = [
      { id: 'managerial-documents', label: 'Deschide documentele', icon: 'pi pi-file', targetTab: 'managerial_documents' },
      { id: 'managerial-workflow', label: 'Deschide workflow-ul', icon: 'pi pi-sitemap', targetTab: 'managerial_workflow' },
    ];
    if (summary.readiness.blockers.includes('mandatory_categories_missing')) {
      actions.unshift({ id: 'managerial-priority-docs', label: 'Completeaza documentele-model', icon: 'pi pi-plus', targetTab: 'managerial_documents' });
    }
    return actions;
  };

  protected canTransitionGovernanceMeeting(
    summary: GovernanceMeetingFinalizationSummary,
    nextStatus: 'held' | 'published',
  ): boolean {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row || resource.key !== 'meetings') {
      return false;
    }
    if (!this.authz.hasAnyPermission(['education.governance.manage', 'education.governance.meeting.close'])) {
      return false;
    }

    const currentStatus = String(row['status'] ?? '').trim();
    if (nextStatus === 'held') {
      return (currentStatus === 'draft' || currentStatus === 'scheduled') && summary.readiness.ready_to_close;
    }

    return currentStatus === 'held' && summary.readiness.ready_to_publish;
  }

  protected executeGovernanceSummaryAction(action: GovernanceSummaryAction): void {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row) {
      return;
    }

    const child = resource.detailChildren?.find((candidate) => candidate.key === action.targetTab);
    if (!child) {
      return;
    }

    if (action.kind === 'create' && this.canCreateChild(child)) {
      this.openCreateChild(resource, child, row);
      return;
    }

    this.activateDetailTab(action.targetTab);
  }

  protected executePersonnelSummaryAction(action: PersonnelSummaryAction): void {
    this.activateDetailTab(action.targetTab);
  }

  protected executePortfolioTransferSummaryAction(action: PortfolioTransferSummaryAction): void {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row) {
      return;
    }

    const child = resource.detailChildren?.find((candidate) => candidate.key === 'portfolio_transfers');
    if (!child) {
      return;
    }

    if (action.kind === 'create' && this.canCreateChild(child)) {
      this.openCreateChild(resource, child, row);
      return;
    }
    if (action.kind === 'tab' && action.targetTab) {
      this.activateDetailTab(action.targetTab);
      return;
    }
    if (action.kind === 'utility' && action.utilityAction === 'regenerate_opis') {
      const recordID = String(row['id'] ?? '').trim();
      const child = resource.detailChildren?.find((candidate) => candidate.key === 'portfolio_opis');
      if (!recordID || !child) {
        return;
      }

      this.portfolioTransferActionLoading.set(action.id);
      this.http.post(`/api/education/portfolios/records/${recordID}/opis/regenerate`, {}).subscribe({
        next: () => {
          this.portfolioTransferActionLoading.set('');
          this.messages.add({
            severity: 'success',
            summary: 'Opis regenerat',
            detail: 'Opisul a fost reconstruit din documentele existente ale portofoliului.',
          });
          this.loadDetailSummary(resource, row);
          this.loadChildResource(child, row, this.childQuery(child, row));
          this.loadResource(resource, this.query(resource));
        },
        error: () => {
          this.portfolioTransferActionLoading.set('');
          this.messages.add({
            severity: 'error',
            summary: 'Regenerarea a esuat',
            detail: 'Opisul nu a putut fi regenerat procedural.',
          });
        },
      });
      return;
    }
    if (action.kind !== 'advance' || !action.advanceAction) {
      return;
    }

    const summary = this.portfolioTransferSummary();
    const transferID = String(summary?.transfer.last_transfer?.id ?? '').trim();
    const recordID = String(row['id'] ?? '').trim();
    if (!summary || !transferID || !recordID) {
      return;
    }

    this.portfolioTransferActionLoading.set(action.id);
    this.http.post(`/api/education/portfolios/records/${recordID}/transfers/${transferID}/advance`, { action: action.advanceAction }).subscribe({
      next: () => {
        this.portfolioTransferActionLoading.set('');
        this.messages.add({
          severity: 'success',
          summary: 'Transfer actualizat',
          detail: 'Fluxul procedural de transfer a fost actualizat.',
        });
        this.loadDetailSummary(resource, row);
        this.loadChildResource(child, row, this.childQuery(child, row));
        this.loadResource(resource, this.query(resource));
      },
      error: () => {
        this.portfolioTransferActionLoading.set('');
        this.messages.add({
          severity: 'error',
          summary: 'Actualizarea a esuat',
          detail: 'Transferul nu a putut fi avansat procedural.',
        });
      },
    });
  }

  protected executePortfolioValorificationSummaryAction(action: PortfolioValorificationSummaryAction): void {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row) {
      return;
    }

    const child = resource.detailChildren?.find((candidate) => candidate.key === action.targetTab);
    if (!child) {
      return;
    }

    if (action.kind === 'create' && this.canCreateChild(child)) {
      this.openCreateChild(resource, child, row);
      return;
    }

    this.activateDetailTab(action.targetTab);
  }

  protected executeRegulationSummaryAction(action: RegulationSummaryAction): void {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    if (!resource || !row) {
      return;
    }

    const child = resource.detailChildren?.find((candidate) => candidate.key === action.targetTab);
    if (!child) {
      return;
    }

    if (action.kind === 'create' && this.canCreateChild(child)) {
      this.openCreateChild(resource, child, row);
      return;
    }

    this.activateDetailTab(action.targetTab);
  }

  protected executeGovernanceBodySummaryAction(action: GovernanceBodySummaryAction): void {
    this.activateDetailTab(action.targetTab);
  }

  executeCommitteeSummaryAction(action: CommitteeSummaryAction): void {
    this.activateDetailTab(action.targetTab);
  }

  readonly executeManagerialPortfolioSummaryAction = (action: ManagerialPortfolioSummaryAction): void => {
    this.activateDetailTab(action.targetTab);
  };

  protected transitionGovernanceMeeting(nextStatus: 'held' | 'published'): void {
    const resource = this.selectedResource();
    const row = this.selectedRow();
    const summary = this.governanceFinalizationSummary();
    const recordID = String(row?.['id'] ?? '').trim();
    if (!resource || !row || !summary || resource.key !== 'meetings' || !recordID || !this.canTransitionGovernanceMeeting(summary, nextStatus)) {
      return;
    }

    const payload = this.buildGovernanceMeetingPayload(row, nextStatus);
    this.governanceTransitionLoading.set(nextStatus);
    this.http.patch<EducationRow>(`${resource.endpoint}/${recordID}`, payload).subscribe({
      next: (updated) => {
        const mergedRow = {
          ...row,
          ...updated,
          status: String(updated?.['status'] ?? nextStatus),
        };
        this.selectedRow.set(mergedRow);
        this.governanceTransitionLoading.set('');
        this.messages.add({
          severity: 'success',
          summary: 'Status actualizat',
          detail: nextStatus === 'held'
            ? 'Sedinta a fost marcata ca tinuta.'
            : 'Sedinta a fost marcata ca publicata.',
        });
        this.loadDetailSummary(resource, mergedRow);
        this.loadResource(resource, this.query(resource));
      },
      error: () => {
        this.governanceTransitionLoading.set('');
        this.messages.add({
          severity: 'error',
          summary: 'Actualizarea a esuat',
          detail: 'Statusul procedural al sedintei nu a putut fi actualizat.',
        });
      },
    });
  }

  protected exportGovernanceSummaryPdf(summary: GovernanceMeetingFinalizationSummary): void {
    const row = this.selectedRow();
    if (!row) {
      return;
    }

    const payload = this.buildGovernanceSummaryExportPayload(row, summary);
    this.governanceSummaryExporting.set(true);
    this.http.post('/api/education/exports/pdf', payload, { responseType: 'blob' as const }).subscribe({
      next: (blob) => {
        this.downloadBlob(blob, `${payload.filename}.pdf`);
        this.governanceSummaryExporting.set(false);
      },
      error: () => {
        this.governanceSummaryExporting.set(false);
        this.messages.add({
          severity: 'error',
          summary: 'Exportul a esuat',
          detail: 'Sumarul procedural nu a putut fi generat in format PDF.',
        });
      },
    });
  }

  protected exportPdf(resource: EducationResourceConfig): void {
    this.exportResource(resource, 'pdf');
  }

  protected exportCsv(resource: EducationResourceConfig): void {
    this.exportResource(resource, 'csv');
  }

  protected childStoreKey(child: EducationDetailChildResourceConfig, parentRow: EducationRow): string {
    return `${child.key}:${String(parentRow['id'] ?? '')}`;
  }

  private loadResource(resource: EducationResourceConfig, query: TableQuery): void {
    this.queryStore.set({ ...this.queryStore(), [resource.key]: query });
    this.loadingKey.set(resource.key);

    this.http.get<PagedResponse<EducationRow>>(resource.endpoint, { params: this.toParams(query) }).subscribe({
      next: (response) => {
        this.rowStore.set({ ...this.rowStore(), [resource.key]: response.items ?? [] });
        this.totalStore.set({ ...this.totalStore(), [resource.key]: response.total ?? 0 });
        this.loadingKey.set('');
      },
      error: () => {
        this.rowStore.set({ ...this.rowStore(), [resource.key]: [] });
        this.totalStore.set({ ...this.totalStore(), [resource.key]: 0 });
        this.loadingKey.set('');
      },
    });
  }

  private openDetail(resource: EducationResourceConfig, row: EducationRow): void {
    const recordID = String(row['id'] ?? '').trim();
    const requestSequence = ++this.detailRequestSequence;

    this.selectedResource.set(resource);
    this.selectedRow.set(row);
    this.detailLoading.set(false);
    this.detailSummaryLoading.set(false);
    this.governanceFinalizationSummary.set(null);
    this.personnelPortfolioDossierSummary.set(null);
    this.portfolioTransferSummary.set(null);
    this.regulationProceduralSummary.set(null);
    this.governanceBodyCompletenessSummary.set(null);
    this.committeeCompletenessSummary.set(null);
    this.managerialPortfolioSummary.set(null);
    this.portfolioTransferActionLoading.set('');
    this.detailDialogOpen.set(true);
    this.activeDetailTab.set('overview');

    if (!recordID) {
      this.regulationProceduralSummary.set(null);
      this.loadAllChildren(resource, row);
      return;
    }

    this.detailLoading.set(true);
    this.http.get<EducationRow>(`${resource.endpoint}/${recordID}`).subscribe({
      next: (detail) => {
        if (requestSequence !== this.detailRequestSequence) {
          return;
        }
        this.selectedRow.set(detail);
        this.loadDetailSummary(resource, detail);
        this.detailLoading.set(false);
        this.loadAllChildren(resource, detail);
      },
      error: () => {
        if (requestSequence !== this.detailRequestSequence) {
          return;
        }
        this.detailLoading.set(false);
        this.messages.add({
          severity: 'warn',
          summary: 'Detaliile complete nu sunt disponibile',
          detail: 'Afisam datele disponibile din tabel si subregistrele asociate.',
        });
        this.loadDetailSummary(resource, row);
        this.loadAllChildren(resource, row);
      },
    });
  }

  private loadDetailSummary(resource: EducationResourceConfig, row: EducationRow): void {
    const endpoint = resource.detailSummaryEndpoint?.(row)?.trim();
    if (!endpoint || !resource.detailSummaryKind) {
      this.detailSummaryLoading.set(false);
      this.governanceFinalizationSummary.set(null);
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      return;
    }

    this.detailSummaryLoading.set(true);
    if (resource.detailSummaryKind === 'governance-finalization') {
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      this.http.get<GovernanceMeetingFinalizationSummary>(endpoint).subscribe({
        next: (summary) => {
          this.governanceFinalizationSummary.set(summary);
          this.detailSummaryLoading.set(false);
        },
        error: () => {
          this.governanceFinalizationSummary.set(null);
          this.detailSummaryLoading.set(false);
        },
      });
      return;
    }

    if (resource.detailSummaryKind === 'personnel-portfolio-dossier') {
      this.governanceFinalizationSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      this.http.get<PersonnelPortfolioDossierSummary>(endpoint).subscribe({
        next: (summary) => {
          this.personnelPortfolioDossierSummary.set(summary);
          this.detailSummaryLoading.set(false);
        },
        error: () => {
          this.personnelPortfolioDossierSummary.set(null);
          this.detailSummaryLoading.set(false);
        },
      });
      return;
    }

    if (resource.detailSummaryKind === 'regulation-procedural') {
      this.governanceFinalizationSummary.set(null);
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      this.http.get<RegulationProceduralSummary>(endpoint).subscribe({
        next: (summary) => {
          this.regulationProceduralSummary.set(summary);
          this.detailSummaryLoading.set(false);
        },
        error: () => {
          this.regulationProceduralSummary.set(null);
          this.detailSummaryLoading.set(false);
        },
      });
      return;
    }

    if (resource.detailSummaryKind === 'governance-body-completeness') {
      this.governanceFinalizationSummary.set(null);
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      this.http.get<GovernanceBodyCompletenessSummary>(endpoint).subscribe({
        next: (summary) => {
          this.governanceBodyCompletenessSummary.set(summary);
          this.detailSummaryLoading.set(false);
        },
        error: () => {
          this.governanceBodyCompletenessSummary.set(null);
          this.detailSummaryLoading.set(false);
        },
      });
      return;
    }

    if (resource.detailSummaryKind === 'committee-completeness') {
      this.governanceFinalizationSummary.set(null);
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.managerialPortfolioSummary.set(null);
      this.http.get<CommitteeCompletenessSummary>(endpoint).subscribe({
        next: (summary) => {
          this.committeeCompletenessSummary.set(summary);
          this.detailSummaryLoading.set(false);
        },
        error: () => {
          this.committeeCompletenessSummary.set(null);
          this.detailSummaryLoading.set(false);
        },
      });
      return;
    }

    if (resource.detailSummaryKind === 'managerial-portfolio') {
      this.governanceFinalizationSummary.set(null);
      this.personnelPortfolioDossierSummary.set(null);
      this.portfolioTransferSummary.set(null);
      this.regulationProceduralSummary.set(null);
      this.governanceBodyCompletenessSummary.set(null);
      this.committeeCompletenessSummary.set(null);
      this.http.get<ManagerialPortfolioSummary>(endpoint).subscribe({
        next: (summary) => {
          this.managerialPortfolioSummary.set(summary);
          this.detailSummaryLoading.set(false);
        },
        error: () => {
          this.managerialPortfolioSummary.set(null);
          this.detailSummaryLoading.set(false);
        },
      });
      return;
    }

    this.governanceFinalizationSummary.set(null);
    this.personnelPortfolioDossierSummary.set(null);
    this.regulationProceduralSummary.set(null);
    this.governanceBodyCompletenessSummary.set(null);
    this.committeeCompletenessSummary.set(null);
    this.managerialPortfolioSummary.set(null);
    this.http.get<PortfolioTransferSummary>(endpoint).subscribe({
      next: (summary) => {
        this.portfolioTransferSummary.set(summary);
        this.detailSummaryLoading.set(false);
      },
      error: () => {
        this.portfolioTransferSummary.set(null);
        this.detailSummaryLoading.set(false);
      },
    });
  }

  private buildGovernanceMeetingPayload(row: EducationRow, nextStatus: 'held' | 'published'): EducationCreateForm {
    return {
      school_year: String(row['school_year'] ?? ''),
      organism: String(row['organism'] ?? ''),
      title: String(row['title'] ?? ''),
      meeting_type: String(row['meeting_type'] ?? ''),
      status: nextStatus,
      quorum_required: Number(row['quorum_required'] ?? 1),
      participants_count: Number(row['participants_count'] ?? 0),
      meeting_date: String(row['meeting_date'] ?? ''),
      location: String(row['location'] ?? ''),
      chairperson: String(row['chairperson'] ?? ''),
      secretary_name: String(row['secretary_name'] ?? ''),
      summary: String(row['summary'] ?? ''),
    };
  }

  private buildGovernanceSummaryExportPayload(
    row: EducationRow,
    summary: GovernanceMeetingFinalizationSummary,
  ): EducationExportPayload {
    const title = `Sumar procedural sedinta - ${String(row['title'] ?? 'sedinta')}`;
    const filename = `sumar-procedural-${this.slugify(String(row['title'] ?? row['id'] ?? 'sedinta'))}`;
    const blockers = summary.readiness.blockers.length
      ? summary.readiness.blockers.map((blocker) => this.governanceBlockerLabel(blocker)).join(', ')
      : 'Nu exista blocaje procedurale';

    return {
      title,
      filename,
      headers: ['Categorie', 'Indicator', 'Valoare'],
      rows: [
        ['Sedinta', 'Organism', String(row['organism'] ?? '-')],
        ['Sedinta', 'Status curent', String(row['status'] ?? '-')],
        ['Sedinta', 'Data', String(row['meeting_date'] ?? '-')],
        ['Readiness', 'Pregatita pentru inchidere', summary.readiness.ready_to_close ? 'Da' : 'Nu'],
        ['Readiness', 'Pregatita pentru publicare', summary.readiness.ready_to_publish ? 'Da' : 'Nu'],
        ['Readiness', 'Blocaje', blockers],
        ['Participanti', 'Cvorum necesar', String(row['quorum_required'] ?? 0)],
        ['Participanti', 'Inregistrati', String(summary.participants.recorded_participants)],
        ['Participanti', 'Prezenti', String(summary.participants.present_participants)],
        ['Participanti', 'Semnati', String(summary.participants.signed_participants)],
        ['Participanti', 'Cu vot', String(summary.participants.voting_participants)],
        ['Voturi', 'Total', String(summary.votes.total)],
        ['Voturi', 'Adoptate', String(summary.votes.adopted)],
        ['Voturi', 'Necesita follow-up', String(summary.votes.requires_follow_up)],
        ['Voturi', 'Hotarari lipsa', String(summary.votes.missing_resolutions)],
        ['Minute', 'Total', String(summary.minutes.total)],
        ['Minute', 'Necesita publicare', String(summary.minutes.requires_publication)],
        ['Minute', 'Actiuni deschise', String(summary.minutes.open_follow_up_items)],
        ['Hotarari', 'Total', String(summary.resolutions.total)],
        ['Hotarari', 'Publicate', String(summary.resolutions.published)],
        ['Hotarari', 'Publicare in asteptare', String(summary.resolutions.pending_publication)],
        ['Hotarari', 'Anonimizare in asteptare', String(summary.resolutions.pending_anonymization)],
        ['Documente', 'Total', String(summary.documents.total)],
        ['Documente', 'Procese-verbale', String(summary.documents.process_verbal_documents)],
        ['Documente', 'Procese-verbale publicate', String(summary.documents.published_process_verbals)],
      ],
    };
  }

  private loadAllChildren(resource: EducationResourceConfig, row: EducationRow): void {
    for (const child of resource.detailChildren ?? []) {
      this.ensureChildLoaded(child, row);
    }
  }

  private ensureChildLoaded(child: EducationDetailChildResourceConfig, parentRow: EducationRow): void {
    const key = this.childStoreKey(child, parentRow);
    if (!(key in this.childRowStore())) {
      this.loadChildResource(child, parentRow, this.childQuery(child, parentRow));
    }
  }

  private loadChildResource(child: EducationDetailChildResourceConfig, parentRow: EducationRow, query: TableQuery): void {
    const key = this.childStoreKey(child, parentRow);
    this.childQueryStore.set({ ...this.childQueryStore(), [key]: query });
    this.childLoadingKey.set(key);

    this.http.get<PagedResponse<EducationRow>>(child.listEndpoint(parentRow), { params: this.toParams(query) }).subscribe({
      next: (response) => {
        this.childRowStore.set({ ...this.childRowStore(), [key]: response.items ?? [] });
        this.childTotalStore.set({ ...this.childTotalStore(), [key]: response.total ?? 0 });
        if (this.childLoadingKey() === key) {
          this.childLoadingKey.set('');
        }
      },
      error: () => {
        this.childRowStore.set({ ...this.childRowStore(), [key]: [] });
        this.childTotalStore.set({ ...this.childTotalStore(), [key]: 0 });
        if (this.childLoadingKey() === key) {
          this.childLoadingKey.set('');
        }
      },
    });
  }

  private formatCell(value: unknown): string {
    if (typeof value === 'boolean') {
      return value ? 'Da' : 'Nu';
    }
    if (value == null || value === '') {
      return '-';
    }
    return String(value);
  }

  private formatConfiguredValue(column: EducationColumn, value: unknown): string {
    if (column.options?.length) {
      const match = column.options.find((option) => option.value === value);
      if (match) {
        return match.label;
      }
    }
    if (column.type === 'boolean') {
      return value ? 'Da' : 'Nu';
    }
    if (value == null || value === '') {
      return '-';
    }
    return String(value);
  }

  private tagSeverity(value: unknown): 'success' | 'info' | 'warn' | 'danger' | 'secondary' {
    return educationStatusSeverity(value);
  }

  private toParams(query: TableQuery): HttpParams {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      params = params.set(`filter.${key}`, String(value));
    }

    return params;
  }

  private normalizeValue(value: string | number | boolean | Date | null): string | number | boolean | null {
    if (value instanceof Date) {
      return value.toISOString().slice(0, 10);
    }
    return value;
  }

  private normalizeFilters(filters: ServerTableFilterState): ServerTableFilterState {
    return Object.fromEntries(
      Object.entries(filters).filter(([, value]) => String(value ?? '').trim() !== ''),
    );
  }

  protected clearPreset(): void {
    const activeResource = this.visibleResources().find((resource) => resource.key === this.activeResourceKey()) ?? this.visibleResources()[0];
    if (!activeResource) {
      return;
    }

    this.activePresetLabel.set('');
    const resetQuery: TableQuery = {
      page: 1,
      pageSize: 20,
      sort: activeResource.columns[0]?.field,
      direction: 'asc',
      filters: {},
    };
    this.loadResource(activeResource, resetQuery);

    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        resource: activeResource.key,
        presetLabel: null,
        filter_status: null,
        filter_stage: null,
        filter_transfer_status: null,
        filter_evaluation_status: null,
        page: null,
        pageSize: null,
        sort: null,
        direction: null,
      },
      queryParamsHandling: 'merge',
    });
  }

  private queryFromRoute(resource: EducationResourceConfig): TableQuery {
    const snapshot = this.route.snapshot.queryParamMap;
    const filters: ServerTableFilterState = {};

    snapshot.keys
      .filter((key) => key.startsWith('filter_'))
      .forEach((key) => {
        const value = snapshot.get(key);
        if (value && value.trim() !== '') {
          filters[key.replace('filter_', '')] = value;
        }
      });

    const page = Number(snapshot.get('page') ?? '');
    const pageSize = Number(snapshot.get('pageSize') ?? '');
    const sort = snapshot.get('sort') ?? resource.columns[0]?.field;
    const direction = snapshot.get('direction') === 'desc' ? 'desc' : 'asc';

    return {
      page: Number.isFinite(page) && page > 0 ? page : 1,
      pageSize: Number.isFinite(pageSize) && pageSize > 0 ? pageSize : 20,
      sort,
      direction,
      filters,
    };
  }

  protected closeEditorDialog(): void {
    this.editorDialogOpen.set(false);
    this.editorTarget.set(null);
    this.editingRecordID.set('');
    this.editorMode.set('create');
    this.clearCreateForm();
  }

  protected closeDeleteDialog(): void {
    this.deleteDialogOpen.set(false);
    this.deleting.set(false);
    this.deleteTarget.set(null);
  }

  private seedCreateForm(fields: EducationResourceConfig['createFields'], row?: EducationRow): void {
    this.clearCreateForm();
    for (const field of fields) {
      if (!row) {
        this.createForm[field.field] = field.defaultValue ?? (field.type === 'boolean' ? false : '');
        continue;
      }

      const rawValue = row[field.field];
      if (field.type === 'date') {
        this.createForm[field.field] = typeof rawValue === 'string' && rawValue !== '' ? new Date(`${rawValue}T00:00:00`) : null;
        continue;
      }
      if (field.type === 'boolean') {
        this.createForm[field.field] = Boolean(rawValue);
        continue;
      }
      if (field.type === 'number') {
        const parsed = typeof rawValue === 'number' ? rawValue : Number(rawValue ?? field.defaultValue ?? 0);
        this.createForm[field.field] = Number.isFinite(parsed) ? parsed : (field.defaultValue as number | undefined) ?? 0;
        continue;
      }
      this.createForm[field.field] = rawValue == null ? (field.defaultValue ?? '') : String(rawValue);
    }
  }

  private clearCreateForm(): void {
    for (const key of Object.keys(this.createForm)) {
      delete this.createForm[key];
    }
  }

  private exportResource(resource: EducationResourceConfig, format: 'pdf' | 'csv'): void {
    const currentRows = this.rows(resource);
    const exportLimit = 5000;
    const exportCount = Math.min(Math.max(this.total(resource), currentRows.length, 1), exportLimit);
    const exportQuery: TableQuery = {
      ...this.query(resource),
      page: 1,
      pageSize: exportCount,
    };

    if (this.total(resource) > exportLimit) {
      this.messages.add({
        severity: 'warn',
        summary: 'Export limitat',
        detail: `Exportam primele ${exportLimit} inregistrari pentru ${resource.label}.`,
      });
    }

    this.exportingKey.set(`${resource.key}:${format}`);
    this.http.get<PagedResponse<EducationRow>>(resource.endpoint, { params: this.toParams(exportQuery) }).subscribe({
      next: (response) => {
        const items = response.items ?? [];
        if (!items.length) {
          this.exportingKey.set('');
          this.messages.add({
            severity: 'warn',
            summary: 'Nu exista date pentru export',
            detail: 'Nu exista inregistrari pentru filtrele curente.',
          });
          return;
        }

        const payload = this.buildExportPayload(resource, items);
        this.http.post(`/api/education/exports/${format}`, payload, { responseType: 'blob' as const }).subscribe({
          next: (blob) => {
            this.downloadBlob(blob, `${payload.filename}.${format === 'pdf' ? 'pdf' : 'csv'}`);
            this.exportingKey.set('');
          },
          error: () => {
            this.exportingKey.set('');
            this.messages.add({
              severity: 'error',
              summary: 'Exportul a esuat',
              detail: 'Fisierul nu a putut fi generat cu configuratia actuala a backendului.',
            });
          },
        });
      },
      error: () => {
        this.exportingKey.set('');
        this.messages.add({
          severity: 'error',
          summary: 'Exportul a esuat',
          detail: 'Nu am putut incarca datele necesare pentru export.',
        });
      },
    });
  }

  private buildExportPayload(resource: EducationResourceConfig, rows: EducationRow[]): EducationExportPayload {
    const presetLabel = this.activePresetLabel().trim();
    const baseTitle = presetLabel ? `${resource.label} - ${presetLabel}` : resource.label;
    const baseFilename = presetLabel ? `${resource.key}-${this.slugify(presetLabel)}` : resource.key;

    return {
      title: baseTitle,
      filename: baseFilename,
      headers: resource.columns.map((column) => column.header),
      rows: rows.map((row) =>
        resource.columns.map((column) => this.displayConfiguredCell(resource.columns, row, column.field)),
      ),
    };
  }

  private slugify(value: string): string {
    return value
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 80);
  }

  private downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  private downloadResourcePdf(resource: EducationResourceConfig, row: EducationRow): void {
    if (!resource.pdfEndpoint) {
      return;
    }
    const fallbackFilename = resource.pdfFilename?.(row) ?? `${resource.key}-${String(row['id'] ?? 'document')}.pdf`;
    this.downloadPdf(resource.pdfEndpoint(row), fallbackFilename);
  }

  private downloadChildPdf(child: EducationDetailChildResourceConfig, parentRow: EducationRow, row: EducationRow): void {
    if (!child.pdfEndpoint) {
      return;
    }
    const fallbackFilename = child.pdfFilename?.(parentRow, row) ?? `${child.key}-${String(row['id'] ?? 'document')}.pdf`;
    this.downloadPdf(child.pdfEndpoint(parentRow, row), fallbackFilename);
  }

  private downloadPdf(endpoint: string, fallbackFilename: string): void {
    this.http.get(endpoint, { observe: 'response', responseType: 'blob' }).subscribe({
      next: (response) => {
        this.downloadBlob(response.body ?? new Blob(), this.extractFilename(response.headers.get('Content-Disposition')) || fallbackFilename);
      },
      error: () => {
        this.messages.add({
          severity: 'error',
          summary: 'Generarea PDF a esuat',
          detail: 'Documentul nu a putut fi generat cu configuratia actuala a backendului.',
        });
      },
    });
  }

  private extractFilename(contentDisposition: string | null): string {
    const match = /filename="([^"]+)"/i.exec(contentDisposition ?? '');
    return match?.[1]?.trim() ?? '';
  }

  protected coerceTabValue(value: unknown, fallback = ''): string {
    const normalized = String(value ?? '').trim();
    return normalized || fallback;
  }

  private editorTargetLabel(target: EditorTarget): string {
    return target.kind === 'resource' ? target.resource.label : target.child.label;
  }

  private childDetailEndpoint(child: EducationDetailChildResourceConfig, parentRow: EducationRow, row: EducationRow): string {
    if (child.detailEndpoint) {
      return child.detailEndpoint(parentRow, row);
    }
    return `${child.listEndpoint(parentRow)}/${row['id']}`;
  }
}
