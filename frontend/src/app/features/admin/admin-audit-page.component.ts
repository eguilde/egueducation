import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { PageEvent } from '@angular/material/paginator';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminAuditEvent } from '../../core/api/api.types';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-audit-page',
  standalone: true,
  imports: [TranslocoPipe, MatCardModule, MatIconModule, ServerTableComponent],
  templateUrl: './admin-audit-page.component.html',
  styleUrl: './admin-audit-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminAuditPageComponent {
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
    pageSize: 15,
    sort: 'created_at',
    direction: 'desc',
    filters: {},
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.auditEvents(state)),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 15 } },
  );
  protected readonly filtersResponse = toSignal(
    this.transloco.langChanges$.pipe(switchMap(() => this.adminApi.auditFilters())),
    { initialValue: { domains: [], target_types: [], statuses: [] } },
  );

  protected readonly rows = computed(() => this.response().items);
  protected readonly activeFilterCount = computed(
    () => Object.values(this.tableState().filters).filter((value) => !!value).length,
  );
  protected readonly summaryCards = computed(() => [
    {
      labelKey: 'admin.audit.columns.domain',
      value: this.filtersResponse().domains.length,
    },
    {
      labelKey: 'admin.audit.columns.targetType',
      value: this.filtersResponse().target_types.length,
    },
    {
      labelKey: 'admin.audit.columns.status',
      value: this.filtersResponse().statuses.length,
    },
  ]);
  protected readonly columns = computed<ServerTableColumn<AdminAuditEvent>[]>(() => [
    {
      key: 'created_at',
      label: this.transloco.translate('admin.audit.columns.createdAt'),
      sortable: true,
      sticky: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
    },
    {
      key: 'actor_subject',
      label: this.transloco.translate('admin.audit.columns.actorSubject'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'domain',
      label: this.transloco.translate('admin.audit.columns.domain'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filtersResponse().domains.map((domain) => ({
          value: domain,
          label: this.transloco.translate(`admin.audit.domains.${domain}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`admin.audit.domains.${row.domain}`),
    },
    {
      key: 'action',
      label: this.transloco.translate('admin.audit.columns.action'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'target_type',
      label: this.transloco.translate('admin.audit.columns.targetType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filtersResponse().target_types.map((targetType) => ({
          value: targetType,
          label: targetType,
        })),
      },
    },
    {
      key: 'target_id',
      label: this.transloco.translate('admin.audit.columns.targetId'),
      sortable: false,
    },
    {
      key: 'status',
      label: this.transloco.translate('admin.audit.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filtersResponse().statuses.map((status) => ({
          value: status,
          label: this.transloco.translate(`admin.audit.status.${status}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`admin.audit.status.${row.status}`),
    },
    {
      key: 'summary',
      label: this.transloco.translate('admin.audit.columns.summary'),
      sortable: false,
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
}
