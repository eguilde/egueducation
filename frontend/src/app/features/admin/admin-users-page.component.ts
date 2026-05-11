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

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminUser, UpsertAdminUserRequest } from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-users-page',
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
    HasPermissionDirective,
    ServerTableComponent,
  ],
  templateUrl: './admin-users-page.component.html',
  styleUrl: './admin-users-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminUsersPageComponent {
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
    sort: 'name',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedUserId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    id: this.fb.control<string | null>(null),
    name: this.fb.nonNullable.control('', [Validators.required]),
    email: this.fb.nonNullable.control('', [Validators.required, Validators.email]),
    phone: this.fb.nonNullable.control(''),
    locale: this.fb.nonNullable.control<'ro' | 'en'>('ro', [Validators.required]),
    status: this.fb.nonNullable.control('active', [Validators.required]),
    email_verified: this.fb.nonNullable.control(true),
    phone_verified: this.fb.nonNullable.control(false),
    preferred_otp_channel: this.fb.nonNullable.control('sms', [Validators.required]),
  });

  protected readonly filters = toSignal(this.adminApi.userFilters(), {
    initialValue: {
      positions: [],
      statuses: [],
      locales: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.users({
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

  protected readonly columns = computed<ServerTableColumn<AdminUser>[]>(() => [
    {
      key: 'name',
      label: this.transloco.translate('admin.users.columns.name'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'email',
      label: this.transloco.translate('admin.users.columns.email'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'position',
      label: this.transloco.translate('admin.users.columns.position'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().positions.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'status',
      label: this.transloco.translate('admin.users.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`admin.userStatus.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`admin.userStatus.${row.status}`),
    },
    {
      key: 'locale',
      label: this.transloco.translate('admin.users.columns.locale'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().locales.map((value) => ({
          value,
          label: value.toUpperCase(),
        })),
      },
      formatter: (row) => row.locale.toUpperCase(),
    },
    {
      key: 'last_login_at',
      label: this.transloco.translate('admin.users.columns.lastLoginAt'),
      sortable: true,
      formatter: (row) =>
        row.last_login_at
          ? new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium', timeStyle: 'short' }).format(
              new Date(row.last_login_at),
            )
          : '—',
    },
  ]);

  protected readonly rows = computed(() => this.response().items);

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
      direction: sort.direction || undefined,
    }));
  }

  protected onSelectUser(user: AdminUser): void {
    this.selectedUserId.set(user.id);
    this.form.reset({
      id: user.id,
      name: user.name,
      email: user.email,
      phone: user.phone,
      locale: user.locale as 'ro' | 'en',
      status: user.status,
      email_verified: user.email_verified,
      phone_verified: user.phone_verified,
      preferred_otp_channel: user.preferred_otp_channel || 'sms',
    });
  }

  protected resetForm(): void {
    this.selectedUserId.set(null);
    this.form.reset({
      id: null,
      name: '',
      email: '',
      phone: '',
      locale: 'ro',
      status: 'active',
      email_verified: true,
      phone_verified: false,
      preferred_otp_channel: 'sms',
    });
  }

  protected saveUser(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpsertAdminUserRequest = {
      id: raw.id || undefined,
      name: raw.name,
      email: raw.email,
      phone: raw.phone,
      locale: raw.locale,
      status: raw.status,
      email_verified: raw.email_verified,
      phone_verified: raw.phone_verified,
      preferred_otp_channel: raw.preferred_otp_channel,
    };

    this.adminApi.saveUser(payload).subscribe({
      next: (saved) => {
        this.selectedUserId.set(saved.id);
        this.form.patchValue({ id: saved.id });
        this.snackBar.open(
          this.transloco.translate('admin.users.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.users.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
