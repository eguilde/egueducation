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
import { MatTabsModule } from '@angular/material/tabs';

import {
  CreateManagerialDossierRequest,
  EducationTaxonomyItem,
  ManagerialDossier,
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
  selector: 'app-managerial-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
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
  templateUrl: './managerial-page.component.html',
  styleUrl: './managerial-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ManagerialPageComponent {
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
    sort: 'due_on',
    direction: 'asc',
    filters: {},
  });
  protected readonly selectedDossierId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    dossier_type: this.fb.nonNullable.control('annual_plan', [Validators.required]),
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    owner_name: this.fb.nonNullable.control('', [Validators.required]),
    due_on: this.fb.control<Date | null>(new Date(), [Validators.required]),
    publication_required: this.fb.nonNullable.control(true),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.managerialDashboard(), {
    initialValue: {
      stats: {
        total_dossiers: 0,
        review_dossiers: 0,
        published_dossiers: 0,
        overdue_dossiers: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.managerialFilters(), {
    initialValue: {
      school_years: [],
      dossier_types: [],
      statuses: [],
    },
  });

  protected readonly taxonomies = toSignal(
    this.api.taxonomies(['school_year', 'managerial_dossier_type', 'managerial_dossier_status']),
    { initialValue: { items: {} } },
  );

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.managerialDossiers({
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
  protected readonly selectedDossier = computed(
    () => this.rows().find((row) => row.id === this.selectedDossierId()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<ManagerialDossier>[]>(() => [
    {
      key: 'open',
      icon: 'open_in_new',
      label: this.transloco.translate('common.open'),
    },
  ]);

  protected readonly columns = computed<ServerTableColumn<ManagerialDossier>[]>(() => [
    {
      key: 'dossier_code',
      label: this.transloco.translate('educationManagerial.columns.dossierCode'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'title',
      label: this.transloco.translate('educationManagerial.columns.title'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'dossier_type',
      label: this.transloco.translate('educationManagerial.columns.dossierType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().dossier_types.map((value) => ({ value, label: this.taxonomyLabel('managerial_dossier_type', value) })),
      },
      formatter: (row) => this.taxonomyLabel('managerial_dossier_type', row.dossier_type),
    },
    {
      key: 'status',
      label: this.transloco.translate('educationManagerial.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({ value, label: this.taxonomyLabel('managerial_dossier_status', value) })),
      },
      formatter: (row) => this.taxonomyLabel('managerial_dossier_status', row.status),
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationManagerial.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'due_on',
      label: this.transloco.translate('educationManagerial.columns.dueOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) => new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.due_on)),
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

  protected onSelectDossier(record: ManagerialDossier): void {
    this.selectedDossierId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: ManagerialDossier }): void {
    if (event.action === 'open') {
      this.onSelectDossier(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected createDossier(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateManagerialDossierRequest = {
      school_year: raw.school_year,
      dossier_type: raw.dossier_type,
      title: raw.title,
      status: raw.status,
      owner_name: raw.owner_name,
      due_on: this.toIsoDate(raw.due_on),
      publication_required: raw.publication_required,
      summary: raw.summary,
    };

    this.api.createManagerialDossier(payload).subscribe({
      next: (item) => {
        this.selectedDossierId.set(item.id);
        this.tableState.update((state) => ({ ...state, page: 1 }));
        this.snackBar.open(
          this.transloco.translate('educationManagerial.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationManagerial.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedDossier();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchManagerialWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationManagerial.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationManagerial.messages.workflowStartFailed'),
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
