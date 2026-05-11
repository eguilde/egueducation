import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { CreatePersonnelRecordRequest, PersonnelRecord } from '../../core/api/api.types';
import { EducationPersonnelApiService } from '../../core/api/education-personnel-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-personnel-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './personnel-page.component.html',
  styleUrl: './personnel-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PersonnelPageComponent {
  private readonly api = inject(EducationPersonnelApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);
  private readonly workflowLauncher = inject(WorkflowLauncherService);

  protected readonly tableState = signal<{
    page: number;
    pageSize: number;
    sort?: string;
    direction?: 'asc' | 'desc';
    filters: Record<string, string>;
    refreshToken: number;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'full_name',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedRecordId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    full_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    role_title: this.fb.nonNullable.control('', [Validators.required]),
    employment_type: this.fb.nonNullable.control('titular', [Validators.required]),
    status: this.fb.nonNullable.control('active', [Validators.required]),
    evaluation_status: this.fb.nonNullable.control('draft', [Validators.required]),
    mobility_stage: this.fb.nonNullable.control('none', [Validators.required]),
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    assigned_unit: this.fb.nonNullable.control(''),
    phone: this.fb.nonNullable.control(''),
    email: this.fb.nonNullable.control(''),
    has_portfolio: this.fb.nonNullable.control(false),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        total_records: 0,
        active_records: 0,
        portfolios_enabled: 0,
        mobility_cases: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.filters(), {
    initialValue: {
      school_years: [],
      employment_types: [],
      statuses: [],
      evaluation_statuses: [],
      mobility_stages: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.records({
          page: state.page,
          pageSize: state.pageSize,
          sort: state.sort,
          direction: state.direction,
          filters: state.filters,
        }),
      ),
    ),
    {
      initialValue: {
        items: [],
        total: 0,
        page: 1,
        pageSize: 10,
      },
    },
  );

  protected readonly rows = computed(() => this.response().items);
  protected readonly selectedRecord = computed(
    () => this.rows().find((row) => row.id === this.selectedRecordId()) ?? null,
  );

  protected readonly columns = computed<ServerTableColumn<PersonnelRecord>[]>(() => [
    {
      key: 'full_name',
      label: this.transloco.translate('educationPersonnel.columns.fullName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'employee_code',
      label: this.transloco.translate('educationPersonnel.columns.employeeCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'employment_type',
      label: this.transloco.translate('educationPersonnel.columns.employmentType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().employment_types.map((value) => ({
          value,
          label: this.transloco.translate(`educationPersonnel.employmentType.${value}`),
        })),
      },
    },
    {
      key: 'status',
      label: this.transloco.translate('educationPersonnel.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`educationPersonnel.status.${value}`),
        })),
      },
    },
    {
      key: 'evaluation_status',
      label: this.transloco.translate('educationPersonnel.columns.evaluationStatus'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().evaluation_statuses.map((value) => ({
          value,
          label: this.transloco.translate(`educationPersonnel.evaluationStatus.${value}`),
        })),
      },
    },
    {
      key: 'mobility_stage',
      label: this.transloco.translate('educationPersonnel.columns.mobilityStage'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().mobility_stages.map((value) => ({
          value,
          label: this.transloco.translate(`educationPersonnel.mobilityStage.${value}`),
        })),
      },
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationPersonnel.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'has_portfolio',
      label: this.transloco.translate('educationPersonnel.columns.portfolio'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('educationPersonnel.portfolio.yes') },
          { value: 'false', label: this.transloco.translate('educationPersonnel.portfolio.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`educationPersonnel.portfolio.${row.has_portfolio ? 'yes' : 'no'}`),
    },
  ]);

  protected onPageChange(event: PageEvent): void {
    this.tableState.update((state) => ({
      ...state,
      page: event.pageIndex + 1,
      pageSize: event.pageSize,
    }));
  }

  protected onFilterChange(filters: ServerTableFilterState): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      filters,
    }));
  }

  protected onSortChange(sort: ServerTableSortState): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: (sort.direction as 'asc' | 'desc' | '') || undefined,
    }));
  }

  protected onSelectRecord(record: PersonnelRecord): void {
    this.selectedRecordId.set(record.id);
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreatePersonnelRecordRequest = {
      full_name: raw.full_name,
      role_title: raw.role_title,
      employment_type: raw.employment_type,
      status: raw.status,
      evaluation_status: raw.evaluation_status,
      mobility_stage: raw.mobility_stage,
      school_year: raw.school_year,
      assigned_unit: raw.assigned_unit,
      phone: raw.phone,
      email: raw.email,
      has_portfolio: raw.has_portfolio,
      notes: raw.notes,
    };

    this.api.createRecord(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationPersonnel.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.form.reset({
          full_name: '',
          role_title: '',
          employment_type: 'titular',
          status: 'active',
          evaluation_status: 'draft',
          mobility_stage: 'none',
          school_year: '2025-2026',
          assigned_unit: '',
          phone: '',
          email: '',
          has_portfolio: false,
          notes: '',
        });
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationPersonnel.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startEvaluationWorkflow(): void {
    const record = this.selectedRecord();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchPersonnelEvaluationWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationPersonnel.messages.evaluationWorkflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationPersonnel.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startMobilityWorkflow(): void {
    const record = this.selectedRecord();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchPersonnelMobilityWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationPersonnel.messages.mobilityWorkflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationPersonnel.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  private refreshData(): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      refreshToken: state.refreshToken + 1,
    }));
  }
}
