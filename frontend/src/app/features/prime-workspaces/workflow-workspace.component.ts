import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TableLazyLoadEvent, TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToolbarModule } from 'primeng/toolbar';

import {
  CreateWorkflowTaskRequest,
  TransitionWorkflowTaskRequest,
  WorkflowDashboardResponse,
  WorkflowDefinition,
  WorkflowTask,
  TableQuery,
} from '../../core/api/api.types';
import { WorkflowApiService } from '../../core/api/workflow-api.service';

const emptyTask = (): CreateWorkflowTaskRequest => ({
  definition_code: '',
  title: '',
  document_number: '',
  source_module: 'registratura',
  source_record_id: '',
  priority: 'medium',
  assigned_to: '',
  due_date: null,
  summary: '',
});

@Component({
  selector: 'app-workflow-workspace',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    CardModule,
    DialogModule,
    InputTextModule,
    SelectModule,
    TableModule,
    TagModule,
    TextareaModule,
    ToolbarModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="flex h-[calc(100dvh-6rem)] min-h-0 flex-col gap-4 overflow-hidden">
      <p-toolbar styleClass="rounded-2xl border border-surface-200 bg-surface-0">
        <ng-template pTemplate="start">
          <div class="grid">
            <h1 class="m-0 text-2xl font-black tracking-[-0.03em]">Flux documente</h1>
            <p class="m-0 text-sm text-muted-color">Taskuri, definiții și tranziții workflow.</p>
          </div>
        </ng-template>
        <ng-template pTemplate="end">
          <p-button icon="pi pi-plus" label="Task nou" (onClick)="openCreateDialog()" />
        </ng-template>
      </p-toolbar>

      <div class="grid gap-4 md:grid-cols-4">
        <p-card>
          <div class="text-sm text-muted-color">Taskuri active</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.active_tasks ?? 0 }}</div>
        </p-card>
        <p-card>
          <div class="text-sm text-muted-color">Întârziate</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.overdue_tasks ?? 0 }}</div>
        </p-card>
        <p-card>
          <div class="text-sm text-muted-color">Aprobări</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.waiting_approval ?? 0 }}</div>
        </p-card>
        <p-card>
          <div class="text-sm text-muted-color">Definiții active</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.active_definitions ?? 0 }}</div>
        </p-card>
      </div>

      <div class="grid min-h-0 flex-1 gap-4 overflow-hidden lg:grid-cols-[19rem_1fr]">
        <p-card class="min-h-0 overflow-hidden">
          <div class="mb-3 flex items-center justify-between gap-2">
            <div class="grid">
              <div class="font-semibold">Definiții active</div>
              <div class="text-sm text-muted-color">Selectează un flux pentru detalii rapide.</div>
            </div>
            <p-tag [value]="definitions().length.toString()" severity="secondary" />
          </div>
          <div class="grid gap-2 overflow-auto pr-1">
            @for (definition of definitions(); track definition.code) {
              <button type="button" class="rounded-xl border border-surface-200 p-3 text-left transition hover:border-primary" [class.border-primary]="selectedDefinition()?.code === definition.code" (click)="selectedDefinition.set(definition)">
                <div class="font-semibold">{{ definition.name }}</div>
                <div class="text-xs text-muted-color">{{ definition.code }}</div>
                <div class="mt-1 flex items-center justify-between text-xs">
                  <span>{{ definition.category }}</span>
                  <p-tag [value]="definition.active ? 'Activ' : 'Inactiv'" [severity]="definition.active ? 'success' : 'secondary'" />
                </div>
              </button>
            } @empty {
              <div class="rounded-xl border border-dashed border-surface-200 p-4 text-sm text-muted-color">Nu există definiții.</div>
            }
          </div>
        </p-card>

        <p-card class="min-h-0 overflow-hidden">
          <div class="mb-3 flex items-center justify-between gap-2">
            <div class="grid">
              <div class="font-semibold">Taskuri workflow</div>
              <div class="text-sm text-muted-color">
                @if (selectedDefinition()) {
                  Filtrat pentru {{ selectedDefinition()!.name }}
                } @else {
                  Toate taskurile active
                }
              </div>
            </div>
            <p-button icon="pi pi-plus" label="Task nou" size="small" (onClick)="openCreateDialog()" />
          </div>
          @if (selectedTask()) {
            <div class="mb-3 rounded-2xl border border-surface-200 bg-surface-50 p-3">
              <div class="flex items-center justify-between gap-3">
                <div class="grid">
                  <div class="font-semibold">{{ selectedTask()!.title }}</div>
                  <div class="text-xs text-muted-color">{{ selectedTask()!.document_number }} • {{ selectedTask()!.definition_name }}</div>
                </div>
                <p-tag [value]="selectedTask()!.status" severity="info" />
              </div>
              <div class="mt-2 grid gap-2 text-sm md:grid-cols-2">
                <div><span class="font-semibold">Prioritate:</span> {{ selectedTask()!.priority }}</div>
                <div><span class="font-semibold">Asignat:</span> {{ selectedTask()!.assigned_to || '-' }}</div>
                <div><span class="font-semibold">Pas curent:</span> {{ selectedTask()!.current_step }}</div>
                <div><span class="font-semibold">Acțiuni:</span> {{ selectedTask()!.available_actions.join(', ') || '-' }}</div>
              </div>
              <div class="mt-3 flex flex-wrap gap-2">
                <p-button label="Tranziție" icon="pi pi-send" size="small" (onClick)="openTransitionDialog(selectedTask()!)" />
              </div>
            </div>
          }
          <p-table [value]="filteredTasks()" [loading]="loading()" [scrollable]="true" scrollHeight="flex" [paginator]="true" [rows]="20" [lazy]="true" (onLazyLoad)="onLazyLoad($event)">
            <ng-template pTemplate="header">
              <tr>
                <th>Referință</th>
                <th>Titlu</th>
                <th>Definiție</th>
                <th>Status</th>
                <th>Prioritate</th>
                <th>Asignat</th>
                <th style="width: 12rem">Acțiuni</th>
              </tr>
            </ng-template>
            <ng-template pTemplate="body" let-task>
              <tr>
                <td>{{ task.document_number }}</td>
                <td>{{ task.title }}</td>
                <td>{{ task.definition_name }}</td>
                <td><p-tag [value]="task.status" severity="info" /></td>
                <td><p-tag [value]="task.priority" severity="secondary" /></td>
                <td>{{ task.assigned_to || '-' }}</td>
              <td>
                <div class="flex gap-1">
                  <p-button icon="pi pi-eye" [text]="true" size="small" severity="secondary" (onClick)="selectedTask.set(task)" />
                  <p-button icon="pi pi-send" [text]="true" size="small" severity="secondary" (onClick)="openTransitionDialog(task)" />
                </div>
              </td>
              </tr>
            </ng-template>
          </p-table>
        </p-card>
      </div>
    </section>

    <p-dialog [visible]="createDialogOpen()" (visibleChange)="createDialogOpen.set($event)" [modal]="true" [draggable]="false" header="Task workflow" [style]="{ width: 'min(56rem, 94vw)' }">
      <div class="grid gap-3">
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Definiție</span>
          <p-select appendTo="body" [options]="definitionOptions()" optionLabel="label" optionValue="value" [ngModel]="taskForm.definition_code" (ngModelChange)="taskForm.definition_code = $event" />
        </label>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Titlu</span>
            <input pInputText [ngModel]="taskForm.title" (ngModelChange)="taskForm.title = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Număr document</span>
            <input pInputText [ngModel]="taskForm.document_number" (ngModelChange)="taskForm.document_number = $event" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Sursă</span>
            <input pInputText [ngModel]="taskForm.source_module" (ngModelChange)="taskForm.source_module = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">ID sursă</span>
            <input pInputText [ngModel]="taskForm.source_record_id" (ngModelChange)="taskForm.source_record_id = $event" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Prioritate</span>
            <p-select appendTo="body" [options]="priorityOptions" optionLabel="label" optionValue="value" [ngModel]="taskForm.priority" (ngModelChange)="taskForm.priority = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Asignat către</span>
            <input pInputText [ngModel]="taskForm.assigned_to" (ngModelChange)="taskForm.assigned_to = $event" />
          </label>
        </div>
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Sumar</span>
          <textarea pTextarea rows="4" [ngModel]="taskForm.summary" (ngModelChange)="taskForm.summary = $event"></textarea>
        </label>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="createDialogOpen.set(false)" />
          <p-button label="Creează" icon="pi pi-check" (onClick)="createTask()" />
        </div>
      </ng-template>
    </p-dialog>

    <p-dialog [visible]="transitionDialogOpen()" (visibleChange)="transitionDialogOpen.set($event)" [modal]="true" [draggable]="false" header="Tranziție task" [style]="{ width: 'min(32rem, 94vw)' }">
      <div class="grid gap-3">
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Acțiune</span>
          <p-select appendTo="body" [options]="actionOptions" optionLabel="label" optionValue="value" [ngModel]="transitionAction()" (ngModelChange)="transitionAction.set($event)" />
        </label>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="transitionDialogOpen.set(false)" />
          <p-button label="Aplică" icon="pi pi-check" (onClick)="transitionTask()" />
        </div>
      </ng-template>
    </p-dialog>
  `,
})
export class WorkflowWorkspaceComponent {
  private readonly api = inject(WorkflowApiService);

  protected readonly loading = signal(false);
  protected readonly dashboard = signal<WorkflowDashboardResponse | null>(null);
  protected readonly definitions = signal<WorkflowDefinition[]>([]);
  protected readonly tasks = signal<WorkflowTask[]>([]);
  protected readonly total = signal(0);
  protected readonly query = signal<TableQuery>({ page: 1, pageSize: 20, sort: 'started_at', direction: 'desc', filters: {} });
  protected readonly createDialogOpen = signal(false);
  protected readonly transitionDialogOpen = signal(false);
  protected readonly selectedTask = signal<WorkflowTask | null>(null);
  protected readonly selectedDefinition = signal<WorkflowDefinition | null>(null);
  protected readonly transitionAction = signal('approve');
  protected taskForm = emptyTask();

  protected readonly priorityOptions = [
    { label: 'Mică', value: 'low' },
    { label: 'Medie', value: 'medium' },
    { label: 'Mare', value: 'high' },
    { label: 'Urgentă', value: 'urgent' },
  ];

  protected readonly actionOptions = [
    { label: 'Aprobare', value: 'approve' },
    { label: 'Arhivare', value: 'archive' },
    { label: 'Trimitere', value: 'submit' },
    { label: 'Returnare', value: 'return' },
  ];

  protected readonly definitionOptions = computed(() =>
    this.definitions().map((definition) => ({ label: `${definition.name} (${definition.code})`, value: definition.code })),
  );
  protected readonly filteredTasks = computed(() =>
    this.selectedDefinition()
      ? this.tasks().filter((task) => task.definition_code === this.selectedDefinition()!.code)
      : this.tasks(),
  );

  ngOnInit(): void {
    this.reload();
  }

  protected reload(): void {
    this.loading.set(true);
    this.api.dashboard().subscribe({ next: (value) => this.dashboard.set(value) });
    this.api.definitions().subscribe({
      next: (definitions) => {
        this.definitions.set(definitions);
        if (!this.selectedDefinition() && definitions.length > 0) {
          this.selectedDefinition.set(definitions[0]);
        }
      },
    });
    this.api.tasks(this.query()).subscribe({
      next: (response) => {
        this.tasks.set(response.items);
        this.total.set(response.total);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  protected onLazyLoad(event: TableLazyLoadEvent): void {
    const pageSize = event.rows ?? this.query().pageSize;
    const page = Math.floor((event.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event.sortField) ? String(event.sortField[0] ?? 'started_at') : String(event.sortField ?? 'started_at');
    const direction = event.sortOrder === 1 ? 'asc' : 'desc';
    this.query.set({ page, pageSize, sort, direction, filters: {} });
    this.reload();
  }

  protected openCreateDialog(): void {
    this.taskForm = emptyTask();
    this.taskForm.definition_code = this.selectedDefinition()?.code ?? this.definitionOptions()[0]?.value ?? '';
    this.createDialogOpen.set(true);
  }

  protected createTask(): void {
    const payload: CreateWorkflowTaskRequest = {
      ...this.taskForm,
      source_record_id: this.taskForm.source_record_id || undefined,
      due_date: this.taskForm.due_date || null,
    };
    this.api.createTask(payload).subscribe({
      next: () => {
        this.createDialogOpen.set(false);
        this.reload();
      },
      error: () => this.createDialogOpen.set(false),
    });
  }

  protected openTransitionDialog(task: WorkflowTask): void {
    this.selectedTask.set(task);
    this.transitionAction.set(task.available_actions[0] ?? 'approve');
    this.transitionDialogOpen.set(true);
  }

  protected transitionTask(): void {
    const task = this.selectedTask();
    if (!task) return;
    const payload: TransitionWorkflowTaskRequest = { action: this.transitionAction() };
    this.api.transitionTask(task.id, payload).subscribe({
      next: () => {
        this.transitionDialogOpen.set(false);
        this.reload();
      },
      error: () => this.transitionDialogOpen.set(false),
    });
  }
}
