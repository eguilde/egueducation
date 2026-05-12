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

import {
  CreateRegulationRecordRequest,
  EducationTaxonomyItem,
  RegulationRecord,
} from '../../core/api/api.types';
import { EducationApiService } from '../../core/api/education-api.service';
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
  selector: 'app-regulations-page',
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
  templateUrl: './regulations-page.component.html',
  styleUrl: './regulations-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RegulationsPageComponent {
  private readonly api = inject(EducationApiService);
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
    sort: 'review_due_on',
    direction: 'asc',
    filters: {},
  });
  protected readonly selectedRecordId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    regulation_type: this.fb.nonNullable.control('roi', [Validators.required]),
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    approval_status: this.fb.nonNullable.control('working_group', [Validators.required]),
    owner_name: this.fb.nonNullable.control('', [Validators.required]),
    review_due_on: this.fb.control<Date | null>(new Date(), [Validators.required]),
    approved_on: this.fb.control<Date | null>(null),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.regulationsDashboard(), {
    initialValue: {
      stats: {
        total_regulations: 0,
        consultation_items: 0,
        approved_regulations: 0,
        published_regulations: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.regulationFilters(), {
    initialValue: {
      school_years: [],
      regulation_types: [],
      statuses: [],
      approval_statuses: [],
    },
  });

  protected readonly taxonomies = toSignal(
    this.api.taxonomies([
      'school_year',
      'education_regulation_type',
      'education_regulation_status',
      'education_regulation_approval_status',
    ]),
    { initialValue: { items: {} } },
  );

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.regulations({
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
  protected readonly rowActions = computed<ServerTableRowAction<RegulationRecord>[]>(() => [
    {
      key: 'open',
      icon: 'open_in_new',
      label: this.transloco.translate('common.open'),
    },
  ]);

  protected readonly columns = computed<ServerTableColumn<RegulationRecord>[]>(() => [
    {
      key: 'regulation_code',
      label: this.transloco.translate('educationRegulations.columns.regulationCode'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'title',
      label: this.transloco.translate('educationRegulations.columns.title'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'regulation_type',
      label: this.transloco.translate('educationRegulations.columns.regulationType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().regulation_types.map((value) => ({
          value,
          label: this.taxonomyLabel('education_regulation_type', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('education_regulation_type', row.regulation_type),
    },
    {
      key: 'status',
      label: this.transloco.translate('educationRegulations.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.taxonomyLabel('education_regulation_status', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('education_regulation_status', row.status),
    },
    {
      key: 'approval_status',
      label: this.transloco.translate('educationRegulations.columns.approvalStatus'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().approval_statuses.map((value) => ({
          value,
          label: this.taxonomyLabel('education_regulation_approval_status', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('education_regulation_approval_status', row.approval_status),
    },
    {
      key: 'review_due_on',
      label: this.transloco.translate('educationRegulations.columns.reviewDueOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(
          new Date(row.review_due_on),
        ),
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

  protected onSelectRecord(record: RegulationRecord): void {
    this.selectedRecordId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: RegulationRecord }): void {
    if (event.action === 'open') {
      this.onSelectRecord(event.row);
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
    const payload: CreateRegulationRecordRequest = {
      school_year: raw.school_year,
      regulation_type: raw.regulation_type,
      title: raw.title,
      status: raw.status,
      approval_status: raw.approval_status,
      owner_name: raw.owner_name,
      review_due_on: this.toIsoDate(raw.review_due_on),
      approved_on: this.toIsoDate(raw.approved_on),
      summary: raw.summary,
    };

    this.api.createRegulation(payload).subscribe({
      next: (item) => {
        this.selectedRecordId.set(item.id);
        this.tableState.update((state) => ({ ...state, page: 1 }));
        this.snackBar.open(
          this.transloco.translate('educationRegulations.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationRegulations.messages.createFailed'),
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

    this.workflowLauncher.launchRegulationWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationRegulations.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationRegulations.messages.workflowStartFailed'),
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
