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
  CreatePersonnelDeclarationRequest,
  EducationTaxonomyItem,
  PersonnelDeclaration,
} from '../../core/api/api.types';
import { EducationApiService } from '../../core/api/education-api.service';
import { EducationPersonnelApiService } from '../../core/api/education-personnel-api.service';
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
  selector: 'app-declarations-page',
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
  templateUrl: './declarations-page.component.html',
  styleUrl: './declarations-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DeclarationsPageComponent {
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
    sort: 'submitted_on',
    direction: 'desc',
    filters: {},
  });
  protected readonly selectedRecordId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    employee_code: this.fb.nonNullable.control('', [Validators.required]),
    full_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    declaration_type: this.fb.nonNullable.control('authenticity', [Validators.required]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    submitted_on: this.fb.control<Date | null>(new Date(), [Validators.required]),
    valid_until: this.fb.control<Date | null>(null),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.declarationsDashboard(), {
    initialValue: {
      stats: {
        total_declarations: 0,
        submitted_declarations: 0,
        validated_declarations: 0,
        expired_declarations: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.declarationFilters(), {
    initialValue: { school_years: [], declaration_types: [], statuses: [] },
  });

  protected readonly taxonomies = toSignal(
    this.educationApi.taxonomies([
      'school_year',
      'education_declaration_type',
      'education_declaration_status',
    ]),
    { initialValue: { items: {} } },
  );

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.declarations({
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
  protected readonly rowActions = computed<ServerTableRowAction<PersonnelDeclaration>[]>(() => [
    {
      key: 'open',
      icon: 'open_in_new',
      label: this.transloco.translate('common.open'),
    },
  ]);

  protected readonly columns = computed<ServerTableColumn<PersonnelDeclaration>[]>(() => [
    {
      key: 'full_name',
      label: this.transloco.translate('educationDeclarations.columns.fullName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'declaration_code',
      label: this.transloco.translate('educationDeclarations.columns.declarationCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'declaration_type',
      label: this.transloco.translate('educationDeclarations.columns.declarationType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().declaration_types.map((value) => ({
          value,
          label: this.taxonomyLabel('education_declaration_type', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('education_declaration_type', row.declaration_type),
    },
    {
      key: 'status',
      label: this.transloco.translate('educationDeclarations.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.taxonomyLabel('education_declaration_status', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('education_declaration_status', row.status),
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationDeclarations.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'submitted_on',
      label: this.transloco.translate('educationDeclarations.columns.submittedOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(
          new Date(row.submitted_on),
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

  protected onSelectRecord(record: PersonnelDeclaration): void {
    this.selectedRecordId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: PersonnelDeclaration }): void {
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
    const payload: CreatePersonnelDeclarationRequest = {
      employee_code: raw.employee_code,
      full_name: raw.full_name,
      declaration_type: raw.declaration_type,
      status: raw.status,
      school_year: raw.school_year,
      submitted_on: this.toIsoDate(raw.submitted_on),
      valid_until: this.toIsoDate(raw.valid_until),
      summary: raw.summary,
    };

    this.api.createDeclaration(payload).subscribe({
      next: (item) => {
        this.selectedRecordId.set(item.id);
        this.tableState.update((state) => ({ ...state, page: 1 }));
        this.snackBar.open(
          this.transloco.translate('educationDeclarations.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationDeclarations.messages.createFailed'),
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

    this.workflowLauncher.launchDeclarationWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationDeclarations.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationDeclarations.messages.workflowStartFailed'),
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
