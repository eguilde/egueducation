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

import { CreateArchiveRecordRequest, ArchiveRecord } from '../../core/api/api.types';
import { EarchivaApiService } from '../../core/api/earchiva-api.service';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-earchiva-page',
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
    ServerTableComponent,
  ],
  templateUrl: './earchiva-page.component.html',
  styleUrl: './earchiva-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EarchivaPageComponent {
  private readonly api = inject(EarchivaApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

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
    sort: 'archived_at',
    direction: 'desc',
    filters: {},
    refreshToken: 0,
  });

  protected readonly form = this.fb.group({
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    fond: this.fb.nonNullable.control('', [Validators.required]),
    series: this.fb.nonNullable.control('', [Validators.required]),
    source_module: this.fb.nonNullable.control('registratura', [Validators.required]),
    source_reference: this.fb.nonNullable.control(''),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    retention_years: this.fb.nonNullable.control(5, [Validators.required, Validators.min(1)]),
    assigned_archivist: this.fb.nonNullable.control(''),
    box_number: this.fb.nonNullable.control(''),
    location_code: this.fb.nonNullable.control(''),
    archived_at: this.fb.control<Date | null>(new Date(), [Validators.required]),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        total_records: 0,
        validated_records: 0,
        draft_records: 0,
        unique_fonds: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.filters(), {
    initialValue: {
      fonds: [],
      series: [],
      statuses: [],
      source_modules: [],
      archivists: [],
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

  protected readonly columns = computed<ServerTableColumn<ArchiveRecord>[]>(() => [
    {
      key: 'record_number',
      label: this.transloco.translate('earchiva.columns.recordNumber'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'title',
      label: this.transloco.translate('earchiva.columns.title'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'fond',
      label: this.transloco.translate('earchiva.columns.fond'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().fonds.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'series',
      label: this.transloco.translate('earchiva.columns.series'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().series.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'source_module',
      label: this.transloco.translate('earchiva.columns.sourceModule'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().source_modules.map((value) => ({
          value,
          label: this.transloco.translate(`earchiva.sourceModule.${value}`),
        })),
      },
    },
    {
      key: 'status',
      label: this.transloco.translate('earchiva.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`earchiva.status.${value}`),
        })),
      },
    },
    {
      key: 'retention_years',
      label: this.transloco.translate('earchiva.columns.retentionYears'),
      sortable: true,
    },
    {
      key: 'archived_at',
      label: this.transloco.translate('earchiva.columns.archivedAt'),
      sortable: true,
      filterKey: 'archived_on',
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.archived_at)),
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

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateArchiveRecordRequest = {
      title: raw.title,
      fond: raw.fond,
      series: raw.series,
      source_module: raw.source_module,
      source_reference: raw.source_reference,
      status: raw.status,
      retention_years: raw.retention_years,
      assigned_archivist: raw.assigned_archivist,
      box_number: raw.box_number,
      location_code: raw.location_code,
      archived_at: raw.archived_at ? this.formatDate(raw.archived_at) : '',
      notes: raw.notes,
    };

    this.api.createRecord(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('earchiva.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.form.reset({
          title: '',
          fond: '',
          series: '',
          source_module: 'registratura',
          source_reference: '',
          status: 'draft',
          retention_years: 5,
          assigned_archivist: '',
          box_number: '',
          location_code: '',
          archived_at: new Date(),
          notes: '',
        });
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('earchiva.messages.createFailed'),
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

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }
}
