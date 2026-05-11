import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import {
  CreateGdprRetentionPolicyRequest,
  CreateGdprSubjectRequestRequest,
  GdprRetentionPolicy,
  GdprSubjectRequest,
} from '../../core/api/api.types';
import { GdprApiService } from '../../core/api/gdpr-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-gdpr-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatChipsModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './gdpr-page.component.html',
  styleUrl: './gdpr-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GdprPageComponent {
  private readonly api = inject(GdprApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);
  private readonly workflowLauncher = inject(WorkflowLauncherService);

  protected readonly requestState = signal<{
    page: number;
    pageSize: number;
    sort?: string;
    direction?: 'asc' | 'desc';
    filters: Record<string, string>;
    refreshToken: number;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'due_on',
    direction: 'asc' as 'asc' | 'desc',
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly retentionState = signal<{
    page: number;
    pageSize: number;
    sort?: string;
    direction?: 'asc' | 'desc';
    filters: Record<string, string>;
    refreshToken: number;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'review_due_on',
    direction: 'asc' as 'asc' | 'desc',
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly selectedRequestId = signal<string | null>(null);

  protected readonly subjectForm = this.fb.group({
    subject_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(3)]),
    request_type: this.fb.nonNullable.control('access', [Validators.required]),
    status: this.fb.nonNullable.control('received', [Validators.required]),
    submitted_on: this.fb.control<Date | null>(new Date(), [Validators.required]),
    due_on: this.fb.control<Date | null>(new Date(new Date().setDate(new Date().getDate() + 30)), [Validators.required]),
    handled_by: this.fb.nonNullable.control(''),
    source_module: this.fb.nonNullable.control('education', [Validators.required]),
    anonymization_required: this.fb.nonNullable.control(false),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly retentionForm = this.fb.group({
    domain_code: this.fb.nonNullable.control('education', [Validators.required]),
    record_category: this.fb.nonNullable.control('', [Validators.required]),
    retention_years: this.fb.nonNullable.control(3, [Validators.required, Validators.min(1)]),
    legal_basis: this.fb.nonNullable.control('', [Validators.required]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    review_due_on: this.fb.control<Date | null>(new Date(new Date().setDate(new Date().getDate() + 90)), [Validators.required]),
    owner_name: this.fb.nonNullable.control(''),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        active_policies: 0,
        pending_requests: 0,
        overdue_requests: 0,
        anonymization_cases: 0,
      },
    },
  });

  protected readonly config = toSignal(this.api.config(), {
    initialValue: {
      settings: {
        publication_anonymization_required: true,
        subject_export_requires_approval: true,
        default_response_sla_days: 30,
        retention_review_notice_days: 90,
        portfolio_consent_required: true,
        portfolio_authenticity_required: true,
      },
      catalogs: {
        domains: [],
        policy_status: [],
        request_types: [],
        request_status: [],
        source_modules: [],
      },
    },
  });

  protected readonly requestFilters = toSignal(this.api.subjectRequestFilters(), {
    initialValue: { request_types: [], statuses: [], source_modules: [] },
  });
  protected readonly retentionFilters = toSignal(this.api.retentionFilters(), {
    initialValue: { domains: [], statuses: [] },
  });

  protected readonly requestResponse = toSignal(
    combineLatest([toObservable(this.requestState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.subjectRequests({
          page: state.page,
          pageSize: state.pageSize,
          sort: state.sort,
          direction: state.direction,
          filters: state.filters,
        }),
      ),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly retentionResponse = toSignal(
    combineLatest([toObservable(this.retentionState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.retentionPolicies({
          page: state.page,
          pageSize: state.pageSize,
          sort: state.sort,
          direction: state.direction,
          filters: state.filters,
        }),
      ),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly requestRows = computed(() => this.requestResponse().items);
  protected readonly selectedRequest = computed(
    () => this.requestRows().find((row) => row.id === this.selectedRequestId()) ?? null,
  );

  protected readonly requestColumns = computed<ServerTableColumn<GdprSubjectRequest>[]>(() => [
    {
      key: 'subject_name',
      label: this.transloco.translate('gdpr.requests.columns.subjectName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'request_code',
      label: this.transloco.translate('gdpr.requests.columns.requestCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'request_type',
      label: this.transloco.translate('gdpr.requests.columns.requestType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.requestFilters().request_types.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.requestType.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.requestType.${row.request_type}`),
    },
    {
      key: 'status',
      label: this.transloco.translate('gdpr.requests.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.requestFilters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.requestStatus.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.requestStatus.${row.status}`),
    },
    {
      key: 'source_module',
      label: this.transloco.translate('gdpr.requests.columns.sourceModule'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.requestFilters().source_modules.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.sourceModule.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.sourceModule.${row.source_module}`),
    },
    {
      key: 'due_on',
      label: this.transloco.translate('gdpr.requests.columns.dueOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.due_on)),
    },
    {
      key: 'anonymization_required',
      label: this.transloco.translate('gdpr.requests.columns.anonymization'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('gdpr.boolean.yes') },
          { value: 'false', label: this.transloco.translate('gdpr.boolean.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`gdpr.boolean.${row.anonymization_required ? 'yes' : 'no'}`),
    },
  ]);

  protected readonly retentionColumns = computed<ServerTableColumn<GdprRetentionPolicy>[]>(() => [
    {
      key: 'policy_code',
      label: this.transloco.translate('gdpr.policies.columns.policyCode'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'domain_code',
      label: this.transloco.translate('gdpr.policies.columns.domain'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.retentionFilters().domains.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.domain.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.domain.${row.domain_code}`),
    },
    {
      key: 'record_category',
      label: this.transloco.translate('gdpr.policies.columns.recordCategory'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    { key: 'retention_years', label: this.transloco.translate('gdpr.policies.columns.retentionYears'), sortable: true },
    {
      key: 'status',
      label: this.transloco.translate('gdpr.policies.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.retentionFilters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.policyStatus.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.policyStatus.${row.status}`),
    },
    {
      key: 'review_due_on',
      label: this.transloco.translate('gdpr.policies.columns.reviewDueOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.review_due_on)),
    },
  ]);

  protected onRequestPageChange(event: PageEvent): void {
    this.requestState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onRetentionPageChange(event: PageEvent): void {
    this.retentionState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onRequestFilterChange(filters: ServerTableFilterState): void {
    this.requestState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onRetentionFilterChange(filters: ServerTableFilterState): void {
    this.retentionState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onRequestSortChange(sort: ServerTableSortState): void {
    this.requestState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected onRetentionSortChange(sort: ServerTableSortState): void {
    this.retentionState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected onSelectRequest(record: GdprSubjectRequest): void {
    this.selectedRequestId.set(record.id);
  }

  protected createSubjectRequest(): void {
    if (this.subjectForm.invalid) {
      this.subjectForm.markAllAsTouched();
      return;
    }
    const raw = this.subjectForm.getRawValue();
    const payload: CreateGdprSubjectRequestRequest = {
      subject_name: raw.subject_name,
      request_type: raw.request_type,
      status: raw.status,
      submitted_on: raw.submitted_on ? this.formatDate(raw.submitted_on) : '',
      due_on: raw.due_on ? this.formatDate(raw.due_on) : '',
      handled_by: raw.handled_by,
      source_module: raw.source_module,
      anonymization_required: raw.anonymization_required,
      notes: raw.notes,
    };
    this.api.createSubjectRequest(payload).subscribe({
      next: () => {
        this.snackBar.open(this.transloco.translate('gdpr.messages.requestCreated'), this.transloco.translate('common.close'), { duration: 3000 });
        const dueDate = this.defaultDueDate();
        this.subjectForm.reset({
          subject_name: '',
          request_type: 'access',
          status: 'received',
          submitted_on: new Date(),
          due_on: dueDate,
          handled_by: '',
          source_module: 'education',
          anonymization_required: false,
          notes: '',
        });
        this.requestState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(this.transloco.translate('gdpr.messages.requestCreateFailed'), this.transloco.translate('common.close'), { duration: 4000 });
      },
    });
  }

  protected createRetentionPolicy(): void {
    if (this.retentionForm.invalid) {
      this.retentionForm.markAllAsTouched();
      return;
    }
    const raw = this.retentionForm.getRawValue();
    const payload: CreateGdprRetentionPolicyRequest = {
      domain_code: raw.domain_code,
      record_category: raw.record_category,
      retention_years: raw.retention_years,
      legal_basis: raw.legal_basis,
      status: raw.status,
      review_due_on: raw.review_due_on ? this.formatDate(raw.review_due_on) : '',
      owner_name: raw.owner_name,
      notes: raw.notes,
    };
    this.api.createRetentionPolicy(payload).subscribe({
      next: () => {
        this.snackBar.open(this.transloco.translate('gdpr.messages.policyCreated'), this.transloco.translate('common.close'), { duration: 3000 });
        const reviewDate = this.defaultReviewDate();
        this.retentionForm.reset({
          domain_code: 'education',
          record_category: '',
          retention_years: 3,
          legal_basis: '',
          status: 'draft',
          review_due_on: reviewDate,
          owner_name: '',
          notes: '',
        });
        this.retentionState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(this.transloco.translate('gdpr.messages.policyCreateFailed'), this.transloco.translate('common.close'), { duration: 4000 });
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedRequest();
    if (!record) {
      return;
    }
    this.workflowLauncher.launchGdprRequestWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(this.transloco.translate('gdpr.messages.workflowStarted'), this.transloco.translate('common.close'), { duration: 3000 });
      },
      error: () => {
        this.snackBar.open(this.transloco.translate('gdpr.messages.workflowStartFailed'), this.transloco.translate('common.close'), { duration: 4000 });
      },
    });
  }

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }

  protected defaultDueDate(): Date {
    const dueDate = new Date();
    dueDate.setDate(dueDate.getDate() + this.config().settings.default_response_sla_days);
    return dueDate;
  }

  protected defaultReviewDate(): Date {
    const reviewDate = new Date();
    reviewDate.setDate(reviewDate.getDate() + this.config().settings.retention_review_notice_days);
    return reviewDate;
  }
}
