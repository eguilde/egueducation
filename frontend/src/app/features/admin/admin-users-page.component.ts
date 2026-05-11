import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { PageEvent } from '@angular/material/paginator';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminUser } from '../../core/api/api.types';
import { ServerTableColumn, ServerTableComponent, ServerTableFilterState, ServerTableSortState } from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-users-page',
  standalone: true,
  imports: [TranslocoPipe, MatButtonModule, MatCardModule, MatIconModule, ServerTableComponent],
  templateUrl: './admin-users-page.component.html',
  styleUrl: './admin-users-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminUsersPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);

  protected readonly tableState = signal<{
    page: number;
    pageSize: number;
    sort?: string;
    direction?: 'asc' | 'desc';
    filters: Record<string, string>;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'name',
    direction: 'asc',
    filters: {},
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
        options: this.filters().positions.map((value) => ({
          value,
          label: value,
        })),
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
    },
    {
      key: 'last_login_at',
      label: this.transloco.translate('admin.users.columns.lastLoginAt'),
      sortable: true,
      formatter: (row) => new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(row.last_login_at)),
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
}
