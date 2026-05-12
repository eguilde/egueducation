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
import {
  AdminWorkflowDefinition,
  CreateAdminWorkflowDefinitionRequest,
} from '../../core/api/api.types';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-workflow-definitions-page',
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
    ServerTableComponent,
  ],
  templateUrl: './admin-workflow-definitions-page.component.html',
  styleUrl: './admin-workflow-definitions-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminWorkflowDefinitionsPageComponent {
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

  protected readonly form = this.fb.group({
    code: this.fb.nonNullable.control('', [Validators.required]),
    name: this.fb.nonNullable.control('', [Validators.required]),
    category: this.fb.nonNullable.control('education', [Validators.required]),
    initial_step: this.fb.nonNullable.control('', [Validators.required]),
    sla_hours: this.fb.nonNullable.control(48, [Validators.required, Validators.min(1)]),
    active: this.fb.nonNullable.control(true),
  });
  protected readonly selectedCode = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly filters = toSignal(this.adminApi.workflowDefinitionFilters(), {
    initialValue: { categories: [] },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.workflowDefinitions({
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
  protected readonly selectedDefinition = computed(
    () => this.rows().find((row) => row.code === this.selectedCode()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<AdminWorkflowDefinition>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly columns = computed<ServerTableColumn<AdminWorkflowDefinition>[]>(() => [
    {
      key: 'name',
      label: this.transloco.translate('admin.workflowDefinitions.columns.name'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'code',
      label: this.transloco.translate('admin.workflowDefinitions.columns.code'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'category',
      label: this.transloco.translate('admin.workflowDefinitions.columns.category'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().categories.map((value) => ({ value, label: value })),
      },
    },
    { key: 'initial_step', label: this.transloco.translate('admin.workflowDefinitions.columns.initialStep') },
    { key: 'sla_hours', label: this.transloco.translate('admin.workflowDefinitions.columns.slaHours'), sortable: true },
    {
      key: 'active',
      label: this.transloco.translate('admin.workflowDefinitions.columns.active'),
      formatter: (row) => this.transloco.translate(`admin.workflowDefinitions.boolean.${row.active ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.workflowDefinitions.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.workflowDefinitions.boolean.no') },
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

  protected onSelectWorkflowDefinition(record: AdminWorkflowDefinition): void {
    this.selectedCode.set(record.code);
    this.activePanel.set('details');
    this.form.reset({
      code: record.code,
      name: record.name,
      category: record.category,
      initial_step: record.initial_step,
      sla_hours: record.sla_hours,
      active: record.active,
    });
  }

  protected onActionClick(event: { action: string; row: AdminWorkflowDefinition }): void {
    if (event.action === 'open') {
      this.onSelectWorkflowDefinition(event.row);
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
      category: 'education',
      initial_step: '',
      sla_hours: 48,
      active: true,
    });
  }

  protected saveWorkflowDefinition(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateAdminWorkflowDefinitionRequest = {
      code: raw.code,
      name: raw.name,
      category: raw.category,
      initial_step: raw.initial_step,
      sla_hours: raw.sla_hours,
      active: raw.active,
    };

    this.adminApi.saveWorkflowDefinition(payload).subscribe({
      next: () => {
        this.selectedCode.set(payload.code);
        this.activePanel.set('details');
        this.snackBar.open(
          this.transloco.translate('admin.workflowDefinitions.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.workflowDefinitions.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
