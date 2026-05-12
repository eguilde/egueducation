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
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatInputModule } from '@angular/material/input';
import { MatTabsModule } from '@angular/material/tabs';

import { AdminApiService } from '../../core/api/admin-api.service';
import {
  AdminRole,
  AdminUser,
  AdminUserRoleAssignment,
  UpsertAdminRoleRequest,
  UpsertAdminUserRoleAssignmentRequest,
} from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-roles-page',
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
  templateUrl: './admin-roles-page.component.html',
  styleUrl: './admin-roles-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminRolesPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly roleTableState = signal({
    page: 1,
    pageSize: 10,
    sort: 'code' as string | undefined,
    direction: 'asc' as 'asc' | 'desc' | undefined,
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly assignmentTableState = signal({
    page: 1,
    pageSize: 10,
    sort: 'user_name' as string | undefined,
    direction: 'asc' as 'asc' | 'desc' | undefined,
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });

  protected readonly selectedRoleCode = signal<string | null>(null);
  protected readonly selectedAssignmentId = signal<string | null>(null);
  protected readonly activePanel = signal<'role-create' | 'role-details' | 'assignment-create' | 'assignment-details'>(
    'role-create',
  );

  protected readonly roleForm = this.fb.group({
    code: this.fb.nonNullable.control('', [Validators.required]),
    label: this.fb.nonNullable.control('', [Validators.required]),
  });

  protected readonly assignmentForm = this.fb.group({
    user_id: this.fb.nonNullable.control('', [Validators.required]),
    role_code: this.fb.nonNullable.control('', [Validators.required]),
    assigned: this.fb.nonNullable.control(true),
  });

  protected readonly rolesResponse = toSignal(
    combineLatest([toObservable(this.roleTableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.roles(state)),
    ),
    { initialValue: { items: [] as AdminRole[], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly assignmentsResponse = toSignal(
    combineLatest([toObservable(this.assignmentTableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) => this.adminApi.roleAssignments(state)),
    ),
    { initialValue: { items: [] as AdminUserRoleAssignment[], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly usersResponse = toSignal(
    combineLatest([toObservable(this.assignmentTableState), this.transloco.langChanges$]).pipe(
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

  protected readonly roleRows = computed(() => this.rolesResponse().items);
  protected readonly assignmentRows = computed(() => this.assignmentsResponse().items);
  protected readonly selectedRole = computed(
    () => this.roleRows().find((role) => role.code === this.selectedRoleCode()) ?? null,
  );
  protected readonly selectedAssignment = computed(
    () => this.assignmentRows().find((assignment) => assignment.id === this.selectedAssignmentId()) ?? null,
  );
  protected readonly roleOptions = computed(() =>
    this.rolesResponse().items.map((role) => ({
      value: role.code,
      label: `${role.code} - ${role.label}`,
    })),
  );
  protected readonly userOptions = computed(() =>
    this.usersResponse().items.map((user) => ({
      value: user.id,
      label: `${user.name} (${user.email})`,
    })),
  );
  protected readonly roleRowActions = computed<ServerTableRowAction<AdminRole>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);
  protected readonly assignmentRowActions = computed<ServerTableRowAction<AdminUserRoleAssignment>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly roleColumns = computed<ServerTableColumn<AdminRole>[]>(() => [
    {
      key: 'code',
      label: this.transloco.translate('admin.roles.columns.code'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'label',
      label: this.transloco.translate('admin.roles.columns.label'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
  ]);

  protected readonly assignmentColumns = computed<ServerTableColumn<AdminUserRoleAssignment>[]>(() => [
    {
      key: 'user_name',
      label: this.transloco.translate('admin.roles.assignments.columns.userName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'user_email',
      label: this.transloco.translate('admin.roles.assignments.columns.userEmail'),
    },
    {
      key: 'role_code',
      label: this.transloco.translate('admin.roles.assignments.columns.roleCode'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.roleOptions(),
      },
    },
    {
      key: 'role_label',
      label: this.transloco.translate('admin.roles.assignments.columns.roleLabel'),
    },
  ]);

  protected onRolePageChange(event: PageEvent): void {
    this.roleTableState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onRoleFilterChange(filters: ServerTableFilterState): void {
    this.roleTableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onRoleSortChange(sort: ServerTableSortState): void {
    this.roleTableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected onAssignmentPageChange(event: PageEvent): void {
    this.assignmentTableState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onAssignmentFilterChange(filters: ServerTableFilterState): void {
    this.assignmentTableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onAssignmentSortChange(sort: ServerTableSortState): void {
    this.assignmentTableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected selectRole(role: AdminRole): void {
    this.selectedRoleCode.set(role.code);
    this.activePanel.set('role-details');
    this.roleForm.reset({ code: role.code, label: role.label });
    this.assignmentForm.patchValue({ role_code: role.code });
    this.assignmentTableState.update((state) => ({
      ...state,
      page: 1,
      filters: { ...state.filters, role_code: role.code },
    }));
  }

  protected selectAssignment(assignment: AdminUserRoleAssignment): void {
    this.selectedAssignmentId.set(assignment.id);
    this.activePanel.set('assignment-details');
    this.assignmentForm.reset({
      user_id: assignment.user_id,
      role_code: assignment.role_code,
      assigned: true,
    });
  }

  protected resetRoleForm(): void {
    this.selectedRoleCode.set(null);
    this.activePanel.set('role-create');
    this.roleForm.reset({ code: '', label: '' });
  }

  protected resetAssignmentForm(): void {
    this.selectedAssignmentId.set(null);
    this.activePanel.set('assignment-create');
    this.assignmentForm.reset({
      user_id: '',
      role_code: this.selectedRoleCode() ?? '',
      assigned: true,
    });
  }

  protected openRoleCreatePanel(): void {
    this.activePanel.set('role-create');
  }

  protected openAssignmentCreatePanel(): void {
    this.activePanel.set('assignment-create');
    const selectedRoleCode = this.selectedRoleCode();
    if (selectedRoleCode) {
      this.assignmentForm.patchValue({ role_code: selectedRoleCode });
    }
  }

  protected onRoleActionClick(event: { action: string; row: AdminRole }): void {
    if (event.action === 'open') {
      this.selectRole(event.row);
    }
  }

  protected onAssignmentActionClick(event: { action: string; row: AdminUserRoleAssignment }): void {
    if (event.action === 'open') {
      this.selectAssignment(event.row);
    }
  }

  protected saveRole(): void {
    if (this.roleForm.invalid) {
      this.roleForm.markAllAsTouched();
      return;
    }
    const raw = this.roleForm.getRawValue();
    const payload: UpsertAdminRoleRequest = {
      code: raw.code.trim(),
      label: raw.label.trim(),
    };
    this.adminApi.saveRole(payload).subscribe({
      next: (saved) => {
        this.selectedRoleCode.set(saved.code);
        this.activePanel.set('role-details');
        this.snackBar.open(this.transloco.translate('admin.roles.messages.saved'), this.transloco.translate('common.close'), { duration: 3000 });
        this.roleTableState.update((state) => ({ ...state, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(this.transloco.translate('admin.roles.messages.saveFailed'), this.transloco.translate('common.close'), { duration: 4000 });
      },
    });
  }

  protected saveAssignment(): void {
    if (this.assignmentForm.invalid) {
      this.assignmentForm.markAllAsTouched();
      return;
    }
    const raw = this.assignmentForm.getRawValue();
    const payload: UpsertAdminUserRoleAssignmentRequest = {
      user_id: raw.user_id,
      role_code: raw.role_code,
      assigned: raw.assigned,
    };
    this.adminApi.saveRoleAssignment(payload).subscribe({
      next: (saved) => {
        this.selectedAssignmentId.set(saved.id);
        this.activePanel.set('assignment-details');
        this.snackBar.open(
          this.transloco.translate(`admin.roles.messages.${payload.assigned ? 'assigned' : 'removed'}`),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.assignmentTableState.update((state) => ({ ...state, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(this.transloco.translate('admin.roles.messages.assignmentFailed'), this.transloco.translate('common.close'), { duration: 4000 });
      },
    });
  }
}
