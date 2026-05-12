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
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTabsModule } from '@angular/material/tabs';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminPosition, UpsertAdminPositionRequest } from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import { ServerTableColumn, ServerTableComponent, ServerTableFilterState, ServerTableRowAction, ServerTableSortState } from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-positions-page',
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
    MatSnackBarModule,
    MatTabsModule,
    HasPermissionDirective,
    ServerTableComponent,
  ],
  templateUrl: './admin-positions-page.component.html',
  styleUrl: './admin-positions-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminPositionsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly tableState = signal<{ page: number; pageSize: number; sort?: string; direction?: 'asc' | 'desc'; filters: Record<string, string>; refreshToken: number }>({
    page: 1,
    pageSize: 10,
    sort: 'name',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedCode = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    code: this.fb.nonNullable.control('', [Validators.required]),
    name: this.fb.nonNullable.control('', [Validators.required]),
    scope_module: this.fb.nonNullable.control('', [Validators.required]),
    active: this.fb.nonNullable.control(true),
    sort_order: this.fb.nonNullable.control(10, [Validators.required]),
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.positions(state)),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly rows = computed(() => this.response().items);
  protected readonly selectedPosition = computed(
    () => this.rows().find((row) => row.code === this.selectedCode()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<AdminPosition>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);
  protected readonly columns = computed<ServerTableColumn<AdminPosition>[]>(() => [
    { key: 'name', label: this.transloco.translate('admin.positions.columns.name'), sortable: true, sticky: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'code', label: this.transloco.translate('admin.positions.columns.code') },
    { key: 'scope_module', label: this.transloco.translate('admin.positions.columns.scopeModule'), sortable: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'sort_order', label: this.transloco.translate('admin.positions.columns.sortOrder'), sortable: true },
    {
      key: 'active',
      label: this.transloco.translate('admin.positions.columns.active'),
      formatter: (row) => this.transloco.translate(`admin.positions.boolean.${row.active ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.positions.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.positions.boolean.no') },
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
    this.tableState.update((state) => ({ ...state, page: 1, sort: sort.active || undefined, direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined }));
  }

  protected onSelectPosition(record: AdminPosition): void {
    this.selectedCode.set(record.code);
    this.activePanel.set('details');
    this.form.reset({
      code: record.code,
      name: record.name,
      scope_module: record.scope_module,
      active: record.active,
      sort_order: record.sort_order,
    });
  }

  protected onActionClick(event: { action: string; row: AdminPosition }): void {
    if (event.action === 'open') {
      this.onSelectPosition(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected resetForm(): void {
    this.selectedCode.set(null);
    this.activePanel.set('create');
    this.form.reset({
      code: '',
      name: '',
      scope_module: '',
      active: true,
      sort_order: 10,
    });
  }

  protected savePosition(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpsertAdminPositionRequest = {
      code: raw.code,
      name: raw.name,
      scope_module: raw.scope_module,
      active: raw.active,
      sort_order: raw.sort_order,
    };

    this.adminApi.savePosition(payload).subscribe({
      next: () => {
        this.selectedCode.set(payload.code);
        this.activePanel.set('details');
        this.snackBar.open(
          this.transloco.translate('admin.positions.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.positions.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
