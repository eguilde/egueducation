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
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { PageEvent } from '@angular/material/paginator';
import { MatTabsModule } from '@angular/material/tabs';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminPermission, AdminPermissionAssignment, UpsertAdminPermissionAssignmentRequest } from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import { ServerTableColumn, ServerTableComponent, ServerTableFilterState, ServerTableRowAction, ServerTableSortState } from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-permissions-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatIconModule,
    MatSelectModule,
    MatSnackBarModule,
    MatTabsModule,
    HasPermissionDirective,
    ServerTableComponent,
  ],
  templateUrl: './admin-permissions-page.component.html',
  styleUrl: './admin-permissions-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminPermissionsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly catalogState = signal<{ page: number; pageSize: number; sort?: string; direction?: 'asc' | 'desc'; filters: Record<string, string> }>({
    page: 1,
    pageSize: 10,
    sort: 'code',
    direction: 'asc',
    filters: {},
  });

  protected readonly assignmentState = signal<{ page: number; pageSize: number; sort?: string; direction?: 'asc' | 'desc'; filters: Record<string, string>; refreshToken: number }>({
    page: 1,
    pageSize: 10,
    sort: 'permission_code',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });

  protected readonly selectedPermissionCode = signal<string | null>(null);
  protected readonly selectedAssignmentId = signal<string | null>(null);
  protected readonly activePanel = signal<'assign' | 'details'>('assign');

  protected readonly form = this.fb.group({
    permission_code: this.fb.nonNullable.control('', [Validators.required]),
    position_code: this.fb.nonNullable.control('', [Validators.required]),
    assigned: this.fb.nonNullable.control(true),
  });

  protected readonly assignmentFilters = toSignal(this.adminApi.permissionAssignmentFilters(), {
    initialValue: { permissions: [], positions: [] },
  });

  protected readonly catalogResponse = toSignal(
    combineLatest([toObservable(this.catalogState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.permissions(state)),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly assignmentsResponse = toSignal(
    combineLatest([toObservable(this.assignmentState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.permissionAssignments(state)),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly permissionRows = computed(() => this.catalogResponse().items);
  protected readonly assignmentRows = computed(() => this.assignmentsResponse().items);

  protected readonly permissionColumns = computed<ServerTableColumn<AdminPermission>[]>(() => [
    { key: 'code', label: this.transloco.translate('admin.permissions.columns.code'), sortable: true, sticky: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'label', label: this.transloco.translate('admin.permissions.columns.label'), sortable: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
    { key: 'user_count', label: this.transloco.translate('admin.permissions.columns.userCount'), sortable: true },
    { key: 'role_count', label: this.transloco.translate('admin.permissions.columns.roleCount'), sortable: true },
  ]);
  protected readonly selectedPermission = computed(
    () => this.permissionRows().find((row) => row.code === this.selectedPermissionCode()) ?? null,
  );
  protected readonly selectedAssignment = computed(
    () => this.assignmentRows().find((row) => row.id === this.selectedAssignmentId()) ?? null,
  );
  protected readonly permissionRowActions = computed<ServerTableRowAction<AdminPermission>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);
  protected readonly assignmentRowActions = computed<ServerTableRowAction<AdminPermissionAssignment>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly assignmentColumns = computed<ServerTableColumn<AdminPermissionAssignment>[]>(() => [
    {
      key: 'permission_code',
      label: this.transloco.translate('admin.permissions.assignments.columns.permissionCode'),
      sortable: true,
      sticky: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.assignmentFilters().permissions.map((permission) => ({ value: permission.code, label: permission.code })),
      },
    },
    { key: 'permission_label', label: this.transloco.translate('admin.permissions.assignments.columns.permissionLabel') },
    {
      key: 'position_code',
      label: this.transloco.translate('admin.permissions.assignments.columns.positionCode'),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.assignmentFilters().positions.map((position) => ({ value: position.code, label: position.code })),
      },
    },
    { key: 'position_name', label: this.transloco.translate('admin.permissions.assignments.columns.positionName'), sortable: true },
    { key: 'scope_module', label: this.transloco.translate('admin.permissions.assignments.columns.scopeModule'), sortable: true, filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') } },
  ]);

  protected onCatalogPageChange(event: PageEvent): void {
    this.catalogState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onCatalogFilterChange(filters: ServerTableFilterState): void {
    this.catalogState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onCatalogSortChange(sort: ServerTableSortState): void {
    this.catalogState.update((state) => ({ ...state, page: 1, sort: sort.active || undefined, direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined }));
  }

  protected onAssignmentPageChange(event: PageEvent): void {
    this.assignmentState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onAssignmentFilterChange(filters: ServerTableFilterState): void {
    this.assignmentState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onAssignmentSortChange(sort: ServerTableSortState): void {
    this.assignmentState.update((state) => ({ ...state, page: 1, sort: sort.active || undefined, direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined }));
  }

  protected onSelectPermission(permission: AdminPermission): void {
    this.selectedPermissionCode.set(permission.code);
    this.activePanel.set('details');
    this.form.patchValue({ permission_code: permission.code });
    this.assignmentState.update((state) => ({
      ...state,
      page: 1,
      filters: { ...state.filters, permission_code: permission.code },
    }));
  }

  protected onPermissionActionClick(event: { action: string; row: AdminPermission }): void {
    if (event.action === 'open') {
      this.onSelectPermission(event.row);
    }
  }

  protected openAssignPanel(): void {
    this.activePanel.set('assign');
  }

  protected onSelectAssignment(assignment: AdminPermissionAssignment): void {
    this.selectedAssignmentId.set(assignment.id);
    this.activePanel.set('details');
    this.form.patchValue({
      permission_code: assignment.permission_code,
      position_code: assignment.position_code,
      assigned: true,
    });
  }

  protected onAssignmentActionClick(event: { action: string; row: AdminPermissionAssignment }): void {
    if (event.action === 'open') {
      this.onSelectAssignment(event.row);
    }
  }

  protected resetForm(): void {
    this.selectedAssignmentId.set(null);
    this.activePanel.set('assign');
    this.form.reset({
      permission_code: this.selectedPermissionCode() || '',
      position_code: '',
      assigned: true,
    });
  }

  protected saveAssignment(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpsertAdminPermissionAssignmentRequest = {
      permission_code: raw.permission_code,
      position_code: raw.position_code,
      assigned: raw.assigned,
    };

    this.adminApi.savePermissionAssignment(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate(`admin.permissions.messages.${payload.assigned ? 'saved' : 'removed'}`),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.assignmentState.update((state) => ({
          ...state,
          page: 1,
          filters: { ...state.filters, permission_code: payload.permission_code },
          refreshToken: state.refreshToken + 1,
        }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.permissions.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
