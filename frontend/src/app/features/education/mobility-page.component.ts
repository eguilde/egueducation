import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTabsModule } from '@angular/material/tabs';

import { CreateMobilityCaseRequest, MobilityCase } from '../../core/api/api.types';
import { EducationMobilityApiService } from '../../core/api/education-mobility-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-mobility-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    MatTabsModule,
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './mobility-page.component.html',
  styleUrl: './mobility-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MobilityPageComponent {
  private readonly api = inject(EducationMobilityApiService);
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
    sort: 'submitted_on',
    direction: 'desc' as 'asc' | 'desc',
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly selectedCaseId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    employee_code: this.fb.nonNullable.control('', [Validators.required]),
    full_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    request_type: this.fb.nonNullable.control('transfer', [Validators.required]),
    stage: this.fb.nonNullable.control('draft', [Validators.required]),
    status: this.fb.nonNullable.control('open', [Validators.required]),
    source_school: this.fb.nonNullable.control(''),
    destination_school: this.fb.nonNullable.control(''),
    submitted_on: this.fb.control<Date | null>(new Date(), [Validators.required]),
    reviewed_by: this.fb.nonNullable.control(''),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        total_cases: 0,
        open_cases: 0,
        approved_cases: 0,
        transfer_cases: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.filters(), {
    initialValue: {
      school_years: [],
      request_types: [],
      stages: [],
      statuses: [],
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
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly rows = computed(() => this.response().items);
  protected readonly selectedCase = computed(
    () => this.rows().find((row) => row.id === this.selectedCaseId()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<MobilityCase>[]>(() => [
    {
      key: 'open',
      icon: 'open_in_new',
      label: this.transloco.translate('common.open'),
    },
  ]);

  protected readonly columns = computed<ServerTableColumn<MobilityCase>[]>(() => [
    {
      key: 'full_name',
      label: this.transloco.translate('educationMobility.columns.fullName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'case_code',
      label: this.transloco.translate('educationMobility.columns.caseCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'request_type',
      label: this.transloco.translate('educationMobility.columns.requestType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().request_types.map((value) => ({
          value,
          label: this.transloco.translate(`educationMobility.requestType.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`educationMobility.requestType.${row.request_type}`),
    },
    {
      key: 'stage',
      label: this.transloco.translate('educationMobility.columns.stage'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().stages.map((value) => ({
          value,
          label: this.transloco.translate(`educationMobility.stage.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`educationMobility.stage.${row.stage}`),
    },
    {
      key: 'status',
      label: this.transloco.translate('educationMobility.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`educationMobility.status.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`educationMobility.status.${row.status}`),
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationMobility.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'submitted_on',
      label: this.transloco.translate('educationMobility.columns.submittedOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.submitted_on)),
    },
  ]);

  protected onPageChange(event: PageEvent): void {
    this.tableState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onFilterChange(filters: ServerTableFilterState): void {
    this.tableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onSortChange(sort: ServerTableSortState): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected onSelectCase(record: MobilityCase): void {
    this.selectedCaseId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: MobilityCase }): void {
    if (event.action === 'open') {
      this.onSelectCase(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateMobilityCaseRequest = {
      employee_code: raw.employee_code,
      full_name: raw.full_name,
      school_year: raw.school_year,
      request_type: raw.request_type,
      stage: raw.stage,
      status: raw.status,
      source_school: raw.source_school,
      destination_school: raw.destination_school,
      submitted_on: raw.submitted_on ? this.formatDate(raw.submitted_on) : '',
      reviewed_by: raw.reviewed_by,
      notes: raw.notes,
    };

    this.api.createRecord(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationMobility.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.form.reset({
          employee_code: '',
          full_name: '',
          school_year: '2025-2026',
          request_type: 'transfer',
          stage: 'draft',
          status: 'open',
          source_school: '',
          destination_school: '',
          submitted_on: new Date(),
          reviewed_by: '',
          notes: '',
        });
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationMobility.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedCase();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchMobilityCaseWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationMobility.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationMobility.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  private refreshData(): void {
    this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
  }

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }
}
