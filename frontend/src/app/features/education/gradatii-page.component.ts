import { DecimalPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { CreateMeritGrantRequest, MeritGrant } from '../../core/api/api.types';
import { EducationGradatiiApiService } from '../../core/api/education-gradatii-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-gradatii-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    DecimalPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './gradatii-page.component.html',
  styleUrl: './gradatii-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GradatiiPageComponent {
  private readonly api = inject(EducationGradatiiApiService);
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
    sort: 'decision_date',
    direction: 'desc' as 'asc' | 'desc',
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly selectedGrantId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    full_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    role_title: this.fb.nonNullable.control('', [Validators.required]),
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    category: this.fb.nonNullable.control('predare', [Validators.required]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    score: this.fb.nonNullable.control(0, [Validators.required, Validators.min(0), Validators.max(100)]),
    committee_name: this.fb.nonNullable.control(''),
    decision_date: this.fb.control<Date | null>(new Date(), [Validators.required]),
    funded: this.fb.nonNullable.control(false),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        total_records: 0,
        approved_records: 0,
        funded_records: 0,
        average_score: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.filters(), {
    initialValue: {
      school_years: [],
      categories: [],
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
  protected readonly selectedGrant = computed(
    () => this.rows().find((row) => row.id === this.selectedGrantId()) ?? null,
  );

  protected readonly columns = computed<ServerTableColumn<MeritGrant>[]>(() => [
    {
      key: 'full_name',
      label: this.transloco.translate('educationGradatii.columns.fullName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'grant_code',
      label: this.transloco.translate('educationGradatii.columns.grantCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'category',
      label: this.transloco.translate('educationGradatii.columns.category'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().categories.map((value) => ({
          value,
          label: this.transloco.translate(`educationGradatii.category.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`educationGradatii.category.${row.category}`),
    },
    {
      key: 'status',
      label: this.transloco.translate('educationGradatii.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`educationGradatii.status.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`educationGradatii.status.${row.status}`),
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationGradatii.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    { key: 'score', label: this.transloco.translate('educationGradatii.columns.score'), sortable: true },
    {
      key: 'decision_date',
      label: this.transloco.translate('educationGradatii.columns.decisionDate'),
      sortable: true,
      filterKey: 'decision_on',
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.decision_date)),
    },
    {
      key: 'funded',
      label: this.transloco.translate('educationGradatii.columns.funded'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('educationGradatii.boolean.yes') },
          { value: 'false', label: this.transloco.translate('educationGradatii.boolean.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`educationGradatii.boolean.${row.funded ? 'yes' : 'no'}`),
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

  protected onSelectGrant(record: MeritGrant): void {
    this.selectedGrantId.set(record.id);
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateMeritGrantRequest = {
      full_name: raw.full_name,
      role_title: raw.role_title,
      school_year: raw.school_year,
      category: raw.category,
      status: raw.status,
      score: raw.score,
      committee_name: raw.committee_name,
      decision_date: raw.decision_date ? this.formatDate(raw.decision_date) : '',
      funded: raw.funded,
      notes: raw.notes,
    };

    this.api.createRecord(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationGradatii.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.form.reset({
          full_name: '',
          role_title: '',
          school_year: '2025-2026',
          category: 'predare',
          status: 'draft',
          score: 0,
          committee_name: '',
          decision_date: new Date(),
          funded: false,
          notes: '',
        });
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationGradatii.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedGrant();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchMeritGrantWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationGradatii.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationGradatii.messages.workflowStartFailed'),
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
