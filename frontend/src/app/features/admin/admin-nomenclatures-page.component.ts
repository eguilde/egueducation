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
import { MatTabsModule } from '@angular/material/tabs';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminNomenclature, CreateAdminNomenclatureRequest } from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-nomenclatures-page',
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
    MatTabsModule,
    HasPermissionDirective,
    ServerTableComponent,
  ],
  templateUrl: './admin-nomenclatures-page.component.html',
  styleUrl: './admin-nomenclatures-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminNomenclaturesPageComponent {
  private readonly adminApi = inject(AdminApiService);
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
    sort: 'domain',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedNomenclatureId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    domain: this.fb.nonNullable.control('registratura_document_type', [Validators.required]),
    code: this.fb.nonNullable.control('', [Validators.required]),
    label_ro: this.fb.nonNullable.control('', [Validators.required]),
    label_en: this.fb.nonNullable.control('', [Validators.required]),
    active: this.fb.nonNullable.control(true),
    sort_order: this.fb.nonNullable.control(10, [Validators.required]),
  });

  protected readonly filters = toSignal(this.adminApi.nomenclatureFilters(), {
    initialValue: { domains: [] },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.nomenclatures({
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
  protected readonly selectedNomenclature = computed(
    () => this.rows().find((row) => row.id === this.selectedNomenclatureId()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<AdminNomenclature>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly columns = computed<ServerTableColumn<AdminNomenclature>[]>(() => [
    {
      key: 'domain',
      label: this.transloco.translate('admin.nomenclatures.columns.domain'),
      sortable: true,
      sticky: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().domains.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'code',
      label: this.transloco.translate('admin.nomenclatures.columns.code'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    { key: 'label_ro', label: this.transloco.translate('admin.nomenclatures.columns.labelRo') },
    { key: 'label_en', label: this.transloco.translate('admin.nomenclatures.columns.labelEn') },
    { key: 'sort_order', label: this.transloco.translate('admin.nomenclatures.columns.sortOrder'), sortable: true },
    {
      key: 'active',
      label: this.transloco.translate('admin.nomenclatures.columns.active'),
      formatter: (row) => this.transloco.translate(`admin.nomenclatures.boolean.${row.active ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.nomenclatures.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.nomenclatures.boolean.no') },
        ],
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
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected onSelectNomenclature(record: AdminNomenclature): void {
    this.selectedNomenclatureId.set(record.id);
    this.activePanel.set('details');
    this.form.reset({
      domain: record.domain,
      code: record.code,
      label_ro: record.label_ro,
      label_en: record.label_en,
      active: record.active,
      sort_order: record.sort_order,
    });
  }

  protected onActionClick(event: { action: string; row: AdminNomenclature }): void {
    if (event.action === 'open') {
      this.onSelectNomenclature(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected resetForm(): void {
    this.selectedNomenclatureId.set(null);
    this.activePanel.set('create');
    this.form.reset({
      domain: 'registratura_document_type',
      code: '',
      label_ro: '',
      label_en: '',
      active: true,
      sort_order: 10,
    });
  }

  protected saveNomenclature(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateAdminNomenclatureRequest = {
      domain: raw.domain,
      code: raw.code,
      label_ro: raw.label_ro,
      label_en: raw.label_en,
      active: raw.active,
      sort_order: raw.sort_order,
    };

    this.adminApi.saveNomenclature(payload).subscribe({
      next: (saved) => {
        this.selectedNomenclatureId.set(saved.id);
        this.activePanel.set('details');
        this.snackBar.open(
          this.transloco.translate('admin.nomenclatures.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.nomenclatures.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
