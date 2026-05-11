import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AuthzService } from '../../core/authz/authz.service';
import {
  CreateWorkflowTaskRequest,
  WorkflowDefinition,
  WorkflowTask,
} from '../../core/api/api.types';
import { WorkflowApiService } from '../../core/api/workflow-api.service';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-workflow-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
  ],
  templateUrl: './workflow-page.component.html',
  styleUrl: './workflow-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkflowPageComponent {
  private readonly api = inject(WorkflowApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);
  protected readonly authz = inject(AuthzService);

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
    sort: 'started_at',
    direction: 'desc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedTaskId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    definition_code: this.fb.nonNullable.control('', [Validators.required]),
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    document_number: this.fb.nonNullable.control(''),
    priority: this.fb.nonNullable.control('medium', [Validators.required]),
    assigned_to: this.fb.nonNullable.control(''),
    due_date: this.fb.control<Date | null>(null),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        active_tasks: 0,
        overdue_tasks: 0,
        waiting_approval: 0,
        active_definitions: 0,
        ready_dossiers: 0,
        blocked_dossiers: 0,
      },
    },
  });

  protected readonly definitions = toSignal(this.api.definitions(), { initialValue: [] as WorkflowDefinition[] });
  protected readonly filters = toSignal(this.api.taskFilters(), {
    initialValue: {
      statuses: [],
      priorities: [],
      assignees: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.tasks({
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

  protected readonly rows = computed(() => this.response().items);
  protected readonly selectedTask = computed(
    () => this.rows().find((row) => row.id === this.selectedTaskId()) ?? null,
  );
  protected readonly canManage = computed(
    () => this.authz.hasPermission('workflow.manage') || this.authz.hasPermission('workflow.transition'),
  );

  protected readonly columns = computed<ServerTableColumn<WorkflowTask>[]>(() => [
    {
      key: 'title',
      label: this.transloco.translate('workflow.columns.title'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'definition_name',
      label: this.transloco.translate('workflow.columns.definition'),
      sortable: true,
      sortKey: 'definition_name',
      filterKey: 'definition_code',
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.definitions().map((definition) => ({
          value: definition.code,
          label: definition.name,
        })),
      },
    },
    {
      key: 'source_module',
      label: this.transloco.translate('workflow.columns.sourceModule'),
      sortable: false,
      formatter: (row) =>
        this.transloco.translate(`workflow.sourceModule.${row.source_module}`, {}, row.source_module),
    },
    {
      key: 'dossier_ready',
      label: this.transloco.translate('workflow.columns.dossierReady'),
      sortable: false,
      formatter: (row) =>
        this.transloco.translate(`workflow.dossierReady.${row.dossier_ready ? 'yes' : 'no'}`),
    },
    {
      key: 'status',
      label: this.transloco.translate('workflow.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`workflow.status.${value}`),
        })),
      },
    },
    {
      key: 'priority',
      label: this.transloco.translate('workflow.columns.priority'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().priorities.map((value) => ({
          value,
          label: this.transloco.translate(`workflow.priority.${value}`),
        })),
      },
    },
    {
      key: 'assigned_to',
      label: this.transloco.translate('workflow.columns.assignedTo'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().assignees.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'current_step',
      label: this.transloco.translate('workflow.columns.currentStep'),
      sortable: true,
    },
    {
      key: 'due_at',
      label: this.transloco.translate('workflow.columns.dueAt'),
      sortable: true,
      filterKey: 'due_on',
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        row.due_at
          ? new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.due_at))
          : '—',
    },
    {
      key: 'started_at',
      label: this.transloco.translate('workflow.columns.startedAt'),
      sortable: true,
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), {
          dateStyle: 'medium',
          timeStyle: 'short',
        }).format(new Date(row.started_at)),
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
      direction: (sort.direction as 'asc' | 'desc' | '') || undefined,
    }));
  }

  protected onSelectTask(task: WorkflowTask): void {
    this.selectedTaskId.set(task.id);
  }

  protected createTask(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateWorkflowTaskRequest = {
      definition_code: raw.definition_code,
      title: raw.title,
      document_number: raw.document_number,
      priority: raw.priority,
      assigned_to: raw.assigned_to,
      due_date: raw.due_date ? this.formatDate(raw.due_date) : null,
      summary: raw.summary,
    };

    this.api.createTask(payload).subscribe({
      next: (task) => {
        this.snackBar.open(
          this.transloco.translate('workflow.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.selectedTaskId.set(task.id);
        this.form.reset({
          definition_code: '',
          title: '',
          document_number: '',
          priority: 'medium',
          assigned_to: '',
          due_date: null,
          summary: '',
        });
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('workflow.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected transitionTask(action: string): void {
    const task = this.selectedTask();
    if (!task || this.isActionBlocked(action, task)) {
      return;
    }

    this.api.transitionTask(task.id, { action }).subscribe({
      next: (updated) => {
        this.snackBar.open(
          this.transloco.translate('workflow.messages.transitioned'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.selectedTaskId.set(updated.id);
        this.refreshData();
      },
      error: () => {
        const messageKey =
          action === 'submit' || action === 'approve'
            ? 'workflow.messages.dossierIncomplete'
            : 'workflow.messages.transitionFailed';
        this.snackBar.open(
          this.transloco.translate(messageKey),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected isActionBlocked(action: string, task: WorkflowTask): boolean {
    return !task.dossier_ready && (action === 'submit' || action === 'approve');
  }

  private refreshData(): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      refreshToken: state.refreshToken + 1,
    }));
  }

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }
}
