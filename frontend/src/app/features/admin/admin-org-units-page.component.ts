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
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminOrgUnit, UpsertAdminOrgUnitRequest } from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-org-units-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    HasPermissionDirective,
    ServerTableComponent,
  ],
  templateUrl: './admin-org-units-page.component.html',
  styleUrl: './admin-org-units-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminOrgUnitsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly tableState = signal({
    page: 1,
    pageSize: 10,
    sort: 'sort_order' as string | undefined,
    direction: 'asc' as 'asc' | 'desc' | undefined,
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly selectedCode = signal<string | null>(null);

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.orgUnits({
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
  protected readonly orgUnitOptions = computed(() =>
    this.rows()
      .filter((row) => row.code !== this.selectedCode())
      .map((row) => ({ value: row.code, label: row.name })),
  );

  protected readonly form = this.fb.group({
    code: this.fb.nonNullable.control('', [Validators.required]),
    name: this.fb.nonNullable.control('', [Validators.required]),
    parent_code: this.fb.nonNullable.control(''),
    active: this.fb.nonNullable.control(true),
    sort_order: this.fb.nonNullable.control(100, [Validators.required]),
  });

  protected readonly columns = computed<ServerTableColumn<AdminOrgUnit>[]>(() => [
    {
      key: 'name',
      label: this.transloco.translate('admin.orgUnits.columns.name'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'code',
      label: this.transloco.translate('admin.orgUnits.columns.code'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'parent_name',
      label: this.transloco.translate('admin.orgUnits.columns.parentName'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
      formatter: (row) => row.parent_name || '—',
    },
    {
      key: 'sort_order',
      label: this.transloco.translate('admin.orgUnits.columns.sortOrder'),
      sortable: true,
    },
    {
      key: 'active',
      label: this.transloco.translate('admin.orgUnits.columns.active'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.orgUnits.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.orgUnits.boolean.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`admin.orgUnits.boolean.${row.active ? 'yes' : 'no'}`),
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
    this.tableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onSortChange(sort: ServerTableSortState): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: (sort.direction as 'asc' | 'desc') || undefined,
    }));
  }

  protected selectOrgUnit(item: AdminOrgUnit): void {
    this.selectedCode.set(item.code);
    this.form.reset({
      code: item.code,
      name: item.name,
      parent_code: item.parent_code,
      active: item.active,
      sort_order: item.sort_order,
    });
  }

  protected resetForm(): void {
    this.selectedCode.set(null);
    this.form.reset({
      code: '',
      name: '',
      parent_code: '',
      active: true,
      sort_order: 100,
    });
  }

  protected saveOrgUnit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpsertAdminOrgUnitRequest = {
      code: raw.code.trim(),
      name: raw.name.trim(),
      parent_code: raw.parent_code.trim(),
      active: raw.active,
      sort_order: raw.sort_order,
    };

    this.adminApi.saveOrgUnit(payload).subscribe({
      next: (saved) => {
        this.selectedCode.set(saved.code);
        this.snackBar.open(
          this.transloco.translate('admin.orgUnits.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.orgUnits.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
