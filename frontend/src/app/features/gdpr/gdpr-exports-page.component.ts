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

import { CreateGdprSubjectExportRequest, GdprSubjectExport } from '../../core/api/api.types';
import { GdprApiService } from '../../core/api/gdpr-api.service';
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
  selector: 'app-gdpr-exports-page',
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
  templateUrl: './gdpr-exports-page.component.html',
  styleUrl: './gdpr-exports-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GdprExportsPageComponent {
  private readonly api = inject(GdprApiService);
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
    sort: 'generated_on',
    direction: 'desc',
    filters: {},
  });
  protected readonly selectedRecordId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    request_id: this.fb.nonNullable.control(''),
    subject_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(3)]),
    source_module: this.fb.nonNullable.control('education', [Validators.required]),
    status: this.fb.nonNullable.control('pending_approval', [Validators.required]),
    export_format: this.fb.nonNullable.control('pdf_bundle', [Validators.required]),
    approved_by: this.fb.nonNullable.control(''),
    approved_on: this.fb.control<Date | null>(null),
    generated_on: this.fb.control<Date | null>(null),
    package_summary: this.fb.nonNullable.control('', [Validators.required]),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.exportsDashboard(), {
    initialValue: {
      stats: {
        total_exports: 0,
        pending_approval: 0,
        generated_exports: 0,
        delivered_exports: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.exportFilters(), {
    initialValue: {
      statuses: [],
      export_formats: [],
      source_modules: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.exports({
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
  protected readonly rowActions = computed<ServerTableRowAction<GdprSubjectExport>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly columns = computed<ServerTableColumn<GdprSubjectExport>[]>(() => [
    {
      key: 'subject_name',
      label: this.transloco.translate('gdprExports.columns.subjectName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'export_code',
      label: this.transloco.translate('gdprExports.columns.exportCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'source_module',
      label: this.transloco.translate('gdprExports.columns.sourceModule'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().source_modules.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.sourceModule.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.sourceModule.${row.source_module}`),
    },
    {
      key: 'status',
      label: this.transloco.translate('gdprExports.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.exportStatus.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.exportStatus.${row.status}`),
    },
    {
      key: 'export_format',
      label: this.transloco.translate('gdprExports.columns.exportFormat'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().export_formats.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.exportFormat.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.exportFormat.${row.export_format}`),
    },
    {
      key: 'generated_on',
      label: this.transloco.translate('gdprExports.columns.generatedOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) => row.generated_on || '—',
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

  protected onSelectRecord(record: GdprSubjectExport): void {
    this.selectedRecordId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: GdprSubjectExport }): void {
    if (event.action === 'open') {
      this.onSelectRecord(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected resetForm(): void {
    this.selectedRecordId.set(null);
    this.activePanel.set('create');
    this.form.reset({
      request_id: '',
      subject_name: '',
      source_module: 'education',
      status: 'pending_approval',
      export_format: 'pdf_bundle',
      approved_by: '',
      approved_on: null,
      generated_on: null,
      package_summary: '',
      notes: '',
    });
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateGdprSubjectExportRequest = {
      request_id: raw.request_id,
      subject_name: raw.subject_name,
      source_module: raw.source_module,
      status: raw.status,
      export_format: raw.export_format,
      approved_by: raw.approved_by,
      approved_on: this.toIsoDate(raw.approved_on),
      generated_on: this.toIsoDate(raw.generated_on),
      package_summary: raw.package_summary,
      notes: raw.notes,
    };

    this.api.createExport(payload).subscribe({
      next: (item) => {
        this.selectedRecordId.set(item.id);
        this.activePanel.set('details');
        this.snackBar.open(
          this.transloco.translate('gdprExports.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('gdprExports.messages.createFailed'),
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
    this.workflowLauncher.launchGdprExportWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('gdprExports.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('gdprExports.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
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
