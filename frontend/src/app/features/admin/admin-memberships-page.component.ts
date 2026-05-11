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
import { AdminMembership, AdminOrgUnit, AdminPosition, AdminUser, UpsertAdminMembershipRequest } from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import { ServerTableColumn, ServerTableComponent, ServerTableFilterState, ServerTableSortState } from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-memberships-page',
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
  templateUrl: './admin-memberships-page.component.html',
  styleUrl: './admin-memberships-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminMembershipsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly tableState = signal<{ page: number; pageSize: number; sort?: string; direction?: 'asc' | 'desc'; filters: Record<string, string>; refreshToken: number }>({
    page: 1,
    pageSize: 10,
    sort: 'user_name',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedMembershipId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    id: this.fb.control<string | null>(null),
    user_id: this.fb.nonNullable.control('', [Validators.required]),
    position_code: this.fb.nonNullable.control('', [Validators.required]),
    org_unit_code: this.fb.nonNullable.control('', [Validators.required]),
    is_primary: this.fb.nonNullable.control(true),
    active: this.fb.nonNullable.control(true),
    start_date: this.fb.nonNullable.control('', [Validators.required]),
    end_date: this.fb.nonNullable.control(''),
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.memberships(state)),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );
  protected readonly orgUnitsResponse = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(() =>
        this.adminApi.orgUnits({
          page: 1,
          pageSize: 200,
          sort: 'sort_order',
          direction: 'asc',
          filters: {},
        }),
      ),
    ),
    { initialValue: { items: [] as AdminOrgUnit[], total: 0, page: 1, pageSize: 200 } },
  );
  protected readonly usersResponse = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(() =>
        this.adminApi.users({
          page: 1,
          pageSize: 200,
          sort: 'name',
          direction: 'asc',
          filters: {},
        }),
      ),
    ),
    { initialValue: { items: [] as AdminUser[], total: 0, page: 1, pageSize: 200 } },
  );
  protected readonly positionsResponse = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(() =>
        this.adminApi.positions({
          page: 1,
          pageSize: 200,
          sort: 'name',
          direction: 'asc',
          filters: {},
        }),
      ),
    ),
    { initialValue: { items: [] as AdminPosition[], total: 0, page: 1, pageSize: 200 } },
  );

  protected readonly rows = computed(() => this.response().items);
  protected readonly userOptions = computed(() =>
    this.usersResponse().items.map((item) => ({
      value: item.id,
      label: `${item.name} (${item.email})`,
    })),
  );
  protected readonly positionOptions = computed(() =>
    this.positionsResponse().items.map((item) => ({
      value: item.code,
      label: `${item.name} (${item.code})`,
    })),
  );
  protected readonly orgUnitOptions = computed(() =>
    this.orgUnitsResponse().items.map((item) => ({
      value: item.code,
      label: item.name,
    })),
  );
  protected readonly columns = computed<ServerTableColumn<AdminMembership>[]>(() => [
    { key: 'user_name', label: this.transloco.translate('admin.memberships.columns.userName'), sortable: true, sticky: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'user_email', label: this.transloco.translate('admin.memberships.columns.userEmail') },
    { key: 'position_name', label: this.transloco.translate('admin.memberships.columns.positionName'), sortable: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'organization_name', label: this.transloco.translate('admin.memberships.columns.organizationName'), sortable: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'start_date', label: this.transloco.translate('admin.memberships.columns.startDate'), sortable: true, filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') } },
    {
      key: 'active',
      label: this.transloco.translate('admin.memberships.columns.active'),
      formatter: (row) => this.transloco.translate(`admin.memberships.boolean.${row.active ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.memberships.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.memberships.boolean.no') },
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

  protected onSelectMembership(record: AdminMembership): void {
    this.selectedMembershipId.set(record.id);
    this.form.reset({
      id: record.id,
      user_id: record.user_id,
      position_code: record.position_code,
      org_unit_code: record.org_unit_code,
      is_primary: record.is_primary,
      active: record.active,
      start_date: record.start_date,
      end_date: record.end_date,
    });
  }

  protected resetForm(): void {
    this.selectedMembershipId.set(null);
    this.form.reset({
      id: null,
      user_id: '',
      position_code: '',
      org_unit_code: '',
      is_primary: true,
      active: true,
      start_date: '',
      end_date: '',
    });
  }

  protected saveMembership(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpsertAdminMembershipRequest = {
      id: raw.id || undefined,
      user_id: raw.user_id,
      position_code: raw.position_code,
      org_unit_code: raw.org_unit_code,
      organization_name: '',
      is_primary: raw.is_primary,
      active: raw.active,
      start_date: raw.start_date,
      end_date: raw.end_date || '',
    };

    this.adminApi.saveMembership(payload).subscribe({
      next: (saved) => {
        this.selectedMembershipId.set(saved.id);
        this.form.patchValue({ id: saved.id, user_id: saved.user_id });
        this.snackBar.open(
          this.transloco.translate('admin.memberships.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.memberships.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
