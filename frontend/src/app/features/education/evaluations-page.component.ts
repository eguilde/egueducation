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

import {
  CreatePersonnelEvaluationRequest,
  EducationTaxonomyItem,
  PersonnelEvaluation,
} from '../../core/api/api.types';
import { EducationApiService } from '../../core/api/education-api.service';
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
  selector: 'app-evaluations-page',
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
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './evaluations-page.component.html',
  styleUrl: './evaluations-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EvaluationsPageComponent {
  private readonly api = inject(EducationPersonnelApiService);
  private readonly educationApi = inject(EducationApiService);
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
  }>({
    page: 1,
    pageSize: 10,
    sort: 'full_name',
    direction: 'asc',
    filters: {},
  });
  protected readonly selectedRecordId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    employee_code: this.fb.nonNullable.control('', [Validators.required]),
    full_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    role_title: this.fb.nonNullable.control('', [Validators.required]),
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    score: this.fb.nonNullable.control(95, [Validators.required, Validators.min(0), Validators.max(100)]),
    evaluator_name: this.fb.nonNullable.control('', [Validators.required]),
    finalized_on: this.fb.control<Date | null>(null),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.evaluationsDashboard(), {
    initialValue: {
      stats: {
        total_evaluations: 0,
        submitted_evaluations: 0,
        approved_evaluations: 0,
        contested_evaluations: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.evaluationFilters(), {
    initialValue: { school_years: [], statuses: [] },
  });

  protected readonly taxonomies = toSignal(
    this.educationApi.taxonomies(['school_year', 'education_evaluation_status']),
    { initialValue: { items: {} } },
  );

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.evaluations({
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
  protected readonly selectedRecord = computed(
    () => this.rows().find((row) => row.id === this.selectedRecordId()) ?? null,
  );

  protected readonly columns = computed<ServerTableColumn<PersonnelEvaluation>[]>(() => [
    {
      key: 'full_name',
      label: this.transloco.translate('educationEvaluations.columns.fullName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'evaluation_code',
      label: this.transloco.translate('educationEvaluations.columns.evaluationCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'employee_code',
      label: this.transloco.translate('educationEvaluations.columns.employeeCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'status',
      label: this.transloco.translate('educationEvaluations.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.taxonomyLabel('education_evaluation_status', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('education_evaluation_status', row.status),
    },
    {
      key: 'score',
      label: this.transloco.translate('educationEvaluations.columns.score'),
      sortable: true,
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationEvaluations.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
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
      direction: (sort.direction as 'asc' | 'desc' | '') || undefined,
    }));
  }

  protected onSelectRecord(record: PersonnelEvaluation): void {
    this.selectedRecordId.set(record.id);
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreatePersonnelEvaluationRequest = {
      employee_code: raw.employee_code,
      full_name: raw.full_name,
      role_title: raw.role_title,
      school_year: raw.school_year,
      status: raw.status,
      score: raw.score,
      evaluator_name: raw.evaluator_name,
      finalized_on: this.toIsoDate(raw.finalized_on),
      summary: raw.summary,
    };

    this.api.createEvaluation(payload).subscribe({
      next: (item) => {
        this.selectedRecordId.set(item.id);
        this.tableState.update((state) => ({ ...state, page: 1 }));
        this.snackBar.open(
          this.transloco.translate('educationEvaluations.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationEvaluations.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedRecord();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchEvaluationWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationEvaluations.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationEvaluations.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected taxonomyOptions(domain: string, fallback: string[]) {
    const catalog = this.taxonomies().items as Record<string, EducationTaxonomyItem[]>;
    const items = catalog[domain] ?? [];
    if (items.length > 0) {
      return items.map((item) => ({ value: item.code, label: this.taxonomyDisplay(item) }));
    }
    return fallback.map((value) => ({ value, label: value }));
  }

  protected taxonomyLabel(domain: string, code: string): string {
    const catalog = this.taxonomies().items as Record<string, EducationTaxonomyItem[]>;
    const item = catalog[domain]?.find((entry: EducationTaxonomyItem) => entry.code === code);
    return item ? this.taxonomyDisplay(item) : code;
  }

  private taxonomyDisplay(item: EducationTaxonomyItem): string {
    return this.transloco.getActiveLang() === 'en' ? item.label_en : item.label_ro;
  }

  private toIsoDate(value: Date | string | null): string {
    if (!value) {
      return '';
    }
    if (typeof value === 'string') {
      return value;
    }
    return value.toISOString().slice(0, 10);
  }
}
