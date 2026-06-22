import { CommonModule } from '@angular/common';
import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputNumberModule } from 'primeng/inputnumber';
import { InputTextModule } from 'primeng/inputtext';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { SelectModule } from 'primeng/select';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToastModule } from 'primeng/toast';
import { ToggleSwitchModule } from 'primeng/toggleswitch';
import { DatePickerModule } from 'primeng/datepicker';

import { TableQuery } from '../../../core/api/api.types';
import { AuthzService } from '../../../core/authz/authz.service';
import {
  PageEvent,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../../shared/server-table/server-table.component';
import {
  EducationColumn,
  EducationCreateForm,
  EducationDetailChildResourceConfig,
  EducationResourceConfig,
  EducationRow,
  EducationSectionConfig,
  EducationServerTableColumn,
} from './education.models';

interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

interface EducationExportPayload {
  title: string;
  filename: string;
  headers: string[];
  rows: string[][];
}

type EditorTarget =
  | { kind: 'resource'; resource: EducationResourceConfig }
  | { kind: 'child'; resource: EducationResourceConfig; child: EducationDetailChildResourceConfig; parent: EducationRow };

type DeleteTarget = EditorTarget & { row: EducationRow };

@Component({
  selector: 'app-education-domain-workspace',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    DatePickerModule,
    DialogModule,
    InputNumberModule,
    InputTextModule,
    ProgressSpinnerModule,
    SelectModule,
    ServerTableComponent,
    TabsModule,
    TagModule,
    TextareaModule,
    ToastModule,
    ToggleSwitchModule,
  ],
  providers: [MessageService],
  template: `
    <section class="flex h-full min-h-0 flex-col overflow-hidden">
      <p-toast />

      <div class="shrink-0 px-3 pb-3 pt-3">
        <div class="rounded-3xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-1">
              <h1 class="m-0 text-2xl font-semibold">{{ section().label }}</h1>
              <p class="m-0 max-w-4xl text-sm text-muted-color">{{ section().description }}</p>
            </div>
            <p-tag [value]="visibleResources().length + ' registre'" severity="secondary" />
          </div>
        </div>
      </div>

      <p-tabs
        [value]="activeResourceKey()"
        (valueChange)="activateResource(coerceTabValue($event))"
        class="flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <p-tablist class="shrink-0">
          @for (resource of visibleResources(); track resource.key) {
            <p-tab [value]="resource.key">
              <span class="inline-flex items-center gap-2">
                <i [class]="resource.icon"></i>
                {{ resource.label }}
              </span>
            </p-tab>
          }
        </p-tablist>

        <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
          @for (resource of visibleResources(); track resource.key) {
            <p-tabpanel [value]="resource.key" class="flex min-h-0 flex-1 overflow-hidden p-0">
              <div class="min-h-0 flex-1 overflow-hidden px-3 pb-3">
                <app-server-table
                  class="h-full"
                  [columns]="tableColumns(resource.columns)"
                  [rows]="rows(resource)"
                  [total]="total(resource)"
                  [pageIndex]="query(resource).page - 1"
                  [pageSize]="query(resource).pageSize"
                  [loading]="loadingKey() === resource.key"
                  [emptyMessage]="resource.emptyText"
                  [sortActive]="query(resource).sort || ''"
                  [sortDirection]="query(resource).direction || ''"
                  [rowActions]="rowActions(resource)"
                  actionColumnLabel="Actiuni"
                  (pageChange)="onPageChange(resource, $event)"
                  (sortChange)="onSortChange(resource, $event)"
                  (filterChange)="onFilterChange(resource, $event)"
                  (actionClick)="onAction(resource, $event.action, $event.row)"
                >
                  <ng-template #serverTableToolbar>
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div class="space-y-1">
                        <h2 class="m-0 text-xl font-semibold">{{ resource.label }}</h2>
                        <p class="m-0 text-sm text-muted-color">{{ resource.description }}</p>
                      </div>
                      <div class="flex flex-wrap items-center justify-end gap-2">
                        <p-tag [value]="resource.readPermission" severity="secondary" />
                        @if (total(resource) > 0) {
                          <p-button
                            label="PDF"
                            icon="pi pi-file-pdf"
                            severity="secondary"
                            [outlined]="true"
                            [loading]="exportingKey() === resource.key + ':pdf'"
                            (onClick)="exportPdf(resource)"
                          />
                          <p-button
                            label="Excel (CSV)"
                            icon="pi pi-file-excel"
                            severity="secondary"
                            [outlined]="true"
                            [loading]="exportingKey() === resource.key + ':csv'"
                            (onClick)="exportCsv(resource)"
                          />
                        }
                      </div>
                    </div>
                  </ng-template>

                  <ng-template #serverTableActionHeader>
                    @if (canCreateResource(resource)) {
                      <p-button
                        icon="pi pi-plus"
                        size="small"
                        [text]="true"
                        [rounded]="true"
                        (onClick)="openCreateResource(resource)"
                      />
                    }
                  </ng-template>
                </app-server-table>
              </div>
            </p-tabpanel>
          }
        </p-tabpanels>
      </p-tabs>

      <p-dialog
        [visible]="editorDialogOpen()"
        (visibleChange)="onEditorDialogVisibilityChange($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editorDialogHeader()"
        [style]="{ width: 'min(58rem, 94vw)' }"
      >
        @if (editorFields().length) {
          <div class="grid gap-4 md:grid-cols-2">
            @for (field of editorFields(); track field.field) {
              <label class="education-field" [ngClass]="field.wide ? 'md:col-span-2' : null">
                <span>{{ field.label }} @if (field.required) { <strong>*</strong> }</span>
                @switch (field.type) {
                  @case ('textarea') {
                    <textarea pTextarea rows="4" [(ngModel)]="createForm[field.field]"></textarea>
                  }
                  @case ('number') {
                    <p-inputNumber [(ngModel)]="createForm[field.field]" [min]="0" />
                  }
                  @case ('date') {
                    <p-datepicker appendTo="body" dateFormat="yy-mm-dd" [(ngModel)]="createForm[field.field]" [showIcon]="true" />
                  }
                  @case ('select') {
                    <p-select
                      appendTo="body"
                      [options]="fieldOptions(field)"
                      optionLabel="label"
                      optionValue="value"
                      [(ngModel)]="createForm[field.field]"
                    />
                  }
                  @case ('boolean') {
                    <p-toggleSwitch [(ngModel)]="createForm[field.field]" />
                  }
                  @default {
                    <input pInputText [(ngModel)]="createForm[field.field]" />
                  }
                }
              </label>
            }
          </div>
        }
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunta" severity="secondary" [outlined]="true" (onClick)="closeEditorDialog()" />
            <p-button [label]="editorMode() === 'edit' ? 'Actualizeaza' : 'Salveaza'" icon="pi pi-check" [loading]="creating()" (onClick)="saveRecord()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="detailDialogOpen()"
        (visibleChange)="onDetailDialogVisibilityChange($event)"
        [modal]="true"
        [draggable]="false"
        header="Detalii inregistrare"
        [style]="{ width: 'min(82rem, 96vw)' }"
      >
        @if (selectedResource(); as resource) {
          @if (detailLoading()) {
            <div class="flex min-h-40 flex-col items-center justify-center gap-3">
              <p-progressSpinner strokeWidth="4" [style]="{ width: '3rem', height: '3rem' }" />
              <p-tag value="Se incarca detaliile complete..." severity="secondary" />
            </div>
          } @else if (selectedRow(); as row) {
            <p-tabs
              [value]="activeDetailTab()"
              (valueChange)="activateDetailTab(coerceTabValue($event, 'overview'))"
              class="flex min-h-0 flex-col overflow-hidden"
            >
              <p-tablist class="shrink-0">
                <p-tab value="overview">
                  <span class="inline-flex items-center gap-2">
                    <i class="pi pi-info-circle"></i>
                    Date generale
                  </span>
                </p-tab>
                @for (child of resource.detailChildren ?? []; track child.key) {
                  <p-tab [value]="child.key">
                    <span class="inline-flex items-center gap-2">
                      <i [class]="child.icon"></i>
                      {{ child.label }}
                    </span>
                  </p-tab>
                }
              </p-tablist>

              <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
                <p-tabpanel value="overview" class="p-0 pt-4">
                  <div class="grid gap-3 md:grid-cols-2">
                    @for (column of resource.columns; track column.field) {
                      <div class="rounded-xl border border-surface p-3">
                        <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">{{ column.header }}</div>
                        <div class="mt-1 font-medium">{{ displayConfiguredCell(resource.columns, row, column.field) }}</div>
                      </div>
                    }
                  </div>
                </p-tabpanel>

                @for (child of resource.detailChildren ?? []; track child.key) {
                  <p-tabpanel [value]="child.key" class="flex min-h-0 flex-1 overflow-hidden p-0 pt-4">
                    <div class="min-h-0 flex-1 overflow-hidden">
                      <app-server-table
                        class="h-full"
                        [columns]="tableColumns(child.columns)"
                        [rows]="childRows(child, row)"
                        [total]="childTotal(child, row)"
                        [pageIndex]="childQuery(child, row).page - 1"
                        [pageSize]="childQuery(child, row).pageSize"
                        [loading]="childLoadingKey() === childStoreKey(child, row)"
                        [emptyMessage]="child.emptyText"
                        [sortActive]="childQuery(child, row).sort || ''"
                        [sortDirection]="childQuery(child, row).direction || ''"
                        [rowActions]="childRowActions(child)"
                        actionColumnLabel="Actiuni"
                        (pageChange)="onChildPageChange(child, row, $event)"
                        (sortChange)="onChildSortChange(child, row, $event)"
                        (filterChange)="onChildFilterChange(child, row, $event)"
                        (actionClick)="onChildAction(resource, child, row, $event.action, $event.row)"
                      >
                        <ng-template #serverTableToolbar>
                          <div class="flex flex-wrap items-start justify-between gap-3">
                            <div class="space-y-1">
                              <h3 class="m-0 text-lg font-semibold">{{ child.label }}</h3>
                              <p class="m-0 text-sm text-muted-color">{{ child.description }}</p>
                            </div>
                            <div class="flex flex-wrap items-center gap-2">
                              <p-tag [value]="summarizeRow(row)" severity="secondary" />
                            </div>
                          </div>
                        </ng-template>

                        <ng-template #serverTableActionHeader>
                          @if (canCreateChild(child)) {
                            <p-button
                              icon="pi pi-plus"
                              size="small"
                              [text]="true"
                              [rounded]="true"
                              (onClick)="openCreateChild(resource, child, row)"
                            />
                          }
                        </ng-template>
                      </app-server-table>
                    </div>
                  </p-tabpanel>
                }
              </p-tabpanels>
            </p-tabs>
          }
        }
      </p-dialog>

      <p-dialog
        [visible]="deleteDialogOpen()"
        (visibleChange)="onDeleteDialogVisibilityChange($event)"
        [modal]="true"
        [draggable]="false"
        header="Confirmare stergere"
        [style]="{ width: 'min(32rem, 92vw)' }"
      >
        <div class="space-y-3">
          <p class="m-0 text-sm text-muted-color">
            @if (deleteTarget(); as target) {
              Confirmati stergerea inregistrarii din {{ deleteTargetLabel(target).toLowerCase() }}?
            } @else {
              Confirmati stergerea inregistrarii selectate?
            }
          </p>
          @if (deleteTarget(); as target) {
            <div class="rounded-2xl border border-surface p-3">
              <div class="font-medium">{{ summarizeRow(target.row) }}</div>
            </div>
          }
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunta" severity="secondary" [outlined]="true" (onClick)="closeDeleteDialog()" />
            <p-button label="Sterge" icon="pi pi-trash" severity="danger" [loading]="deleting()" (onClick)="deleteRecord()" />
          </div>
        </ng-template>
      </p-dialog>
    </section>
  `,
  styles: `
    :host {
      display: block;
      min-height: 0;
      height: 100%;
    }

    :host ::ng-deep .p-tabs,
    :host ::ng-deep .p-tabpanels,
    :host ::ng-deep .p-tabpanel {
      display: flex;
      flex: 1 1 auto;
      min-height: 0;
      flex-direction: column;
      overflow: hidden;
      background: transparent;
    }

    .education-field {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
      color: var(--p-text-muted-color);
      font-size: 0.875rem;
      font-weight: 600;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationDomainWorkspaceComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly messages = inject(MessageService);
  private readonly authz = inject(AuthzService);
  private detailRequestSequence = 0;

  readonly section = input.required<EducationSectionConfig>();
  protected readonly visibleResources = computed(() =>
    this.section().resources.filter((resource) => this.authz.hasPermission(resource.readPermission)),
  );

  protected readonly rowStore = signal<Record<string, EducationRow[]>>({});
  protected readonly totalStore = signal<Record<string, number>>({});
  protected readonly queryStore = signal<Record<string, TableQuery>>({});
  protected readonly childRowStore = signal<Record<string, EducationRow[]>>({});
  protected readonly childTotalStore = signal<Record<string, number>>({});
  protected readonly childQueryStore = signal<Record<string, TableQuery>>({});
  protected readonly loadingKey = signal('');
  protected readonly childLoadingKey = signal('');
  protected readonly activeResourceKey = signal('');
  protected readonly editorDialogOpen = signal(false);
  protected readonly detailDialogOpen = signal(false);
  protected readonly deleteDialogOpen = signal(false);
  protected readonly creating = signal(false);
  protected readonly deleting = signal(false);
  protected readonly detailLoading = signal(false);
  protected readonly exportingKey = signal('');
  protected readonly editorMode = signal<'create' | 'edit'>('create');
  protected readonly editingRecordID = signal('');
  protected readonly activeDetailTab = signal('overview');
  protected readonly selectedResource = signal<EducationResourceConfig | null>(null);
  protected readonly selectedRow = signal<EducationRow | null>(null);
  protected readonly editorTarget = signal<EditorTarget | null>(null);
  protected readonly deleteTarget = signal<DeleteTarget | null>(null);
  protected readonly createForm: EducationCreateForm = {};

  ngOnInit(): void {
    const firstResource = this.visibleResources()[0];
    if (!firstResource) {
      return;
    }

    this.activeResourceKey.set(firstResource.key);
    this.loadResource(firstResource, this.query(firstResource));
  }

  protected activateResource(key: string): void {
    const resource = this.visibleResources().find((candidate) => candidate.key === key);
    if (!resource) {
      return;
    }

    this.activeResourceKey.set(resource.key);
    if (!(resource.key in this.rowStore())) {
      this.loadResource(resource, this.query(resource));
    }
  }

  protected rows(resource: EducationResourceConfig): EducationRow[] {
    return this.rowStore()[resource.key] ?? [];
  }

  protected total(resource: EducationResourceConfig): number {
    return this.totalStore()[resource.key] ?? 0;
  }

  protected query(resource: EducationResourceConfig): TableQuery {
    return this.queryStore()[resource.key] ?? {
      page: 1,
      pageSize: 20,
      sort: resource.columns[0]?.field,
      direction: 'asc',
      filters: {},
    };
  }

  protected childRows(child: EducationDetailChildResourceConfig, parentRow: EducationRow): EducationRow[] {
    return this.childRowStore()[this.childStoreKey(child, parentRow)] ?? [];
  }

  protected childTotal(child: EducationDetailChildResourceConfig, parentRow: EducationRow): number {
    return this.childTotalStore()[this.childStoreKey(child, parentRow)] ?? 0;
  }

  protected childQuery(child: EducationDetailChildResourceConfig, parentRow: EducationRow): TableQuery {
    return this.childQueryStore()[this.childStoreKey(child, parentRow)] ?? {
      page: 1,
      pageSize: 10,
      sort: child.columns[0]?.field,
      direction: 'asc',
      filters: {},
    };
  }

  protected tableColumns(columns: EducationColumn[]): EducationServerTableColumn[] {
    return columns.map((column) => ({
      key: column.field,
      label: column.header,
      kind: column.type === 'tag' ? 'tag' : column.type === 'boolean' ? 'boolean' : column.type === 'number' ? 'number' : 'text',
      sortable: column.sortable,
      sortKey: column.field,
      filterKey: column.field,
      width: column.width,
      formatter: (row) => this.formatConfiguredValue(column, row[column.field]),
      tagSeverity: (row) => this.tagSeverity(row[column.field]),
      filter: column.filter === 'text'
        ? { type: 'text', placeholder: column.header }
        : column.filter === 'select'
          ? {
              type: 'select',
              placeholder: 'Toate',
              options: (column.options ?? []).map((option) => ({
                label: option.label,
                value: String(option.value),
              })),
            }
          : undefined,
    }));
  }

  protected rowActions(resource: EducationResourceConfig): ServerTableRowAction<EducationRow>[] {
    const actions: ServerTableRowAction<EducationRow>[] = [{ key: 'view', icon: 'visibility', label: 'Deschide' }];
    if (resource.pdfEndpoint) {
      actions.push({
        key: 'pdf',
        icon: 'pi pi-file-pdf',
        label: resource.pdfActionLabel ?? 'Descarca PDF',
      });
    }
    if (this.canManageResource(resource)) {
      actions.push(
        { key: 'edit', icon: 'edit', label: 'Editeaza' },
        { key: 'delete', icon: 'delete', label: 'Sterge' },
      );
    }
    return actions;
  }

  protected childRowActions(child: EducationDetailChildResourceConfig): ServerTableRowAction<EducationRow>[] {
    const actions: ServerTableRowAction<EducationRow>[] = [];
    if (child.pdfEndpoint) {
      actions.push({
        key: 'pdf',
        icon: 'pi pi-file-pdf',
        label: child.pdfActionLabel ?? 'Descarca PDF',
      });
    }
    if (this.canManageChild(child)) {
      actions.push(
        { key: 'edit', icon: 'edit', label: 'Editeaza' },
        { key: 'delete', icon: 'delete', label: 'Sterge' },
      );
    }
    return actions;
  }

  protected canCreateResource(resource: EducationResourceConfig): boolean {
    return resource.allowCreate !== false
      && !!resource.createEndpoint
      && (!!resource.managePermission ? this.authz.hasPermission(resource.managePermission) : true);
  }

  protected canManageResource(resource: EducationResourceConfig): boolean {
    return !!resource.createEndpoint
      && !!resource.managePermission
      && this.authz.hasPermission(resource.managePermission);
  }

  protected canCreateChild(child: EducationDetailChildResourceConfig): boolean {
    return child.allowCreate !== false
      && (!!child.managePermission ? this.authz.hasPermission(child.managePermission) : true);
  }

  protected canManageChild(child: EducationDetailChildResourceConfig): boolean {
    return !!child.managePermission && this.authz.hasPermission(child.managePermission);
  }

  protected onPageChange(resource: EducationResourceConfig, event: PageEvent): void {
    const nextQuery: TableQuery = {
      ...this.query(resource),
      page: event.pageIndex + 1,
      pageSize: event.pageSize,
    };
    this.loadResource(resource, nextQuery);
  }

  protected onSortChange(resource: EducationResourceConfig, sort: ServerTableSortState): void {
    const current = this.query(resource);
    this.loadResource(resource, {
      ...current,
      page: 1,
      sort: sort.active || current.sort,
      direction: sort.direction || current.direction,
    });
  }

  protected onFilterChange(resource: EducationResourceConfig, filters: ServerTableFilterState): void {
    this.loadResource(resource, {
      ...this.query(resource),
      page: 1,
      filters: this.normalizeFilters(filters),
    });
  }

  protected onChildPageChange(child: EducationDetailChildResourceConfig, parentRow: EducationRow, event: PageEvent): void {
    this.loadChildResource(child, parentRow, {
      ...this.childQuery(child, parentRow),
      page: event.pageIndex + 1,
      pageSize: event.pageSize,
    });
  }

  protected onChildSortChange(child: EducationDetailChildResourceConfig, parentRow: EducationRow, sort: ServerTableSortState): void {
    const current = this.childQuery(child, parentRow);
    this.loadChildResource(child, parentRow, {
      ...current,
      page: 1,
      sort: sort.active || current.sort,
      direction: sort.direction || current.direction,
    });
  }

  protected onChildFilterChange(child: EducationDetailChildResourceConfig, parentRow: EducationRow, filters: ServerTableFilterState): void {
    this.loadChildResource(child, parentRow, {
      ...this.childQuery(child, parentRow),
      page: 1,
      filters: this.normalizeFilters(filters),
    });
  }

  protected openCreateResource(resource: EducationResourceConfig): void {
    this.editorMode.set('create');
    this.editingRecordID.set('');
    this.editorTarget.set({ kind: 'resource', resource });
    this.seedCreateForm(resource.createFields);
    this.editorDialogOpen.set(true);
  }

  protected openCreateChild(resource: EducationResourceConfig, child: EducationDetailChildResourceConfig, parentRow: EducationRow): void {
    this.editorMode.set('create');
    this.editingRecordID.set('');
    this.editorTarget.set({ kind: 'child', resource, child, parent: parentRow });
    this.seedCreateForm(child.createFields);
    this.editorDialogOpen.set(true);
  }

  protected saveRecord(): void {
    const target = this.editorTarget();
    const editingRecordID = this.editingRecordID();
    if (!target) {
      return;
    }

    const payload = Object.fromEntries(
      Object.entries(this.createForm).map(([key, value]) => [key, this.normalizeValue(value)]),
    );

    const request = target.kind === 'resource'
      ? editingRecordID
        ? this.http.patch(`${target.resource.endpoint}/${editingRecordID}`, payload)
        : this.http.post(target.resource.createEndpoint, payload)
      : editingRecordID
        ? this.http.patch(this.childDetailEndpoint(target.child, target.parent, { id: editingRecordID }), payload)
        : this.http.post(target.child.createEndpoint(target.parent), payload);

    this.creating.set(true);
    request.subscribe({
      next: () => {
        this.creating.set(false);
        const successLabel = this.editorTargetLabel(target);
        this.closeEditorDialog();
        this.messages.add({
          severity: 'success',
          summary: editingRecordID ? 'Inregistrare actualizata' : 'Inregistrare adaugata',
          detail: `${successLabel} a fost salvat.`,
        });

        if (target.kind === 'resource') {
          this.loadResource(target.resource, { ...this.query(target.resource), page: 1 });
        } else {
          this.loadChildResource(target.child, target.parent, { ...this.childQuery(target.child, target.parent), page: 1 });
        }
      },
      error: () => {
        this.creating.set(false);
        this.messages.add({
          severity: 'error',
          summary: 'Salvarea a esuat',
          detail: 'Datele nu au putut fi salvate cu configuratia actuala a backendului.',
        });
      },
    });
  }

  protected onAction(resource: EducationResourceConfig, action: string, row: EducationRow): void {
    if (action === 'view') {
      this.openDetail(resource, row);
      return;
    }
    if (action === 'pdf') {
      this.downloadResourcePdf(resource, row);
      return;
    }
    if (action === 'edit') {
      this.editorMode.set('edit');
      this.editingRecordID.set(String(row['id'] ?? '').trim());
      this.editorTarget.set({ kind: 'resource', resource });
      this.seedCreateForm(resource.createFields, row);
      this.editorDialogOpen.set(true);
      return;
    }
    if (action === 'delete') {
      this.deleteTarget.set({ kind: 'resource', resource, row });
      this.deleteDialogOpen.set(true);
    }
  }

  protected onChildAction(resource: EducationResourceConfig, child: EducationDetailChildResourceConfig, parentRow: EducationRow, action: string, row: EducationRow): void {
    if (action === 'pdf') {
      this.downloadChildPdf(child, parentRow, row);
      return;
    }
    if (action === 'edit') {
      this.editorMode.set('edit');
      this.editingRecordID.set(String(row['id'] ?? '').trim());
      this.editorTarget.set({ kind: 'child', resource, child, parent: parentRow });
      this.seedCreateForm(child.createFields, row);
      this.editorDialogOpen.set(true);
      return;
    }
    if (action === 'delete') {
      this.deleteTarget.set({ kind: 'child', resource, child, parent: parentRow, row });
      this.deleteDialogOpen.set(true);
    }
  }

  protected displayConfiguredCell(columns: EducationColumn[], row: EducationRow, field: string): string {
    const column = columns.find((candidate) => candidate.field === field);
    if (!column) {
      return this.formatCell(row[field]);
    }
    return this.formatConfiguredValue(column, row[field]);
  }

  protected activateDetailTab(value: string): void {
    this.activeDetailTab.set(value);
    if (value === 'overview') {
      return;
    }

    const resource = this.selectedResource();
    const row = this.selectedRow();
    const child = resource?.detailChildren?.find((candidate) => candidate.key === value);
    if (!child || !row) {
      return;
    }
    this.ensureChildLoaded(child, row);
  }

  protected onDetailDialogVisibilityChange(visible: boolean): void {
    this.detailDialogOpen.set(visible);
    if (!visible) {
      this.detailRequestSequence += 1;
      this.detailLoading.set(false);
      this.selectedRow.set(null);
      this.selectedResource.set(null);
      this.activeDetailTab.set('overview');
    }
  }

  protected onEditorDialogVisibilityChange(visible: boolean): void {
    if (!visible) {
      this.closeEditorDialog();
      return;
    }
    this.editorDialogOpen.set(true);
  }

  protected editorDialogHeader(): string {
    const target = this.editorTarget();
    if (!target) {
      return this.editorMode() === 'edit' ? 'Editeaza' : 'Adauga';
    }
    const verb = this.editorMode() === 'edit' ? 'Editeaza' : 'Adauga';
    return `${verb} ${this.editorTargetLabel(target)}`;
  }

  protected editorFields() {
    const target = this.editorTarget();
    if (!target) {
      return [];
    }
    return target.kind === 'resource' ? target.resource.createFields : target.child.createFields;
  }

  protected fieldOptions(field: EducationColumn | EducationResourceConfig['createFields'][number]) {
    if (Array.isArray(field.options)) {
      return field.options;
    }
    if (typeof field.options !== 'function') {
      return [];
    }

    const target = this.editorTarget();
    if (!target) {
      return [];
    }

    return field.options({
      resource: target.resource,
      child: target.kind === 'child' ? target.child : undefined,
      parentRow: target.kind === 'child' ? target.parent : undefined,
      childRows: (child, parentRow) => this.childRows(child, parentRow),
    });
  }

  protected onDeleteDialogVisibilityChange(visible: boolean): void {
    if (!visible) {
      this.closeDeleteDialog();
      return;
    }
    this.deleteDialogOpen.set(true);
  }

  protected deleteTargetLabel(target: DeleteTarget): string {
    return target.kind === 'resource' ? target.resource.label : target.child.label;
  }

  protected deleteRecord(): void {
    const target = this.deleteTarget();
    const recordID = String(target?.row['id'] ?? '').trim();
    if (!target || !recordID) {
      this.closeDeleteDialog();
      return;
    }

    const request = target.kind === 'resource'
      ? this.http.delete(`${target.resource.endpoint}/${recordID}`)
      : this.http.delete(this.childDetailEndpoint(target.child, target.parent, target.row));

    this.deleting.set(true);
    request.subscribe({
      next: () => {
        this.deleting.set(false);
        this.closeDeleteDialog();
        this.messages.add({
          severity: 'success',
          summary: 'Inregistrare stearsa',
          detail: `${this.deleteTargetLabel(target)} a fost stearsa.`,
        });

        if (target.kind === 'resource') {
          this.loadResource(target.resource, this.query(target.resource));
        } else {
          this.loadChildResource(target.child, target.parent, this.childQuery(target.child, target.parent));
        }
      },
      error: () => {
        this.deleting.set(false);
        this.messages.add({
          severity: 'error',
          summary: 'Stergerea a esuat',
          detail: 'Inregistrarea nu a putut fi stearsa cu configuratia actuala a backendului.',
        });
      },
    });
  }

  protected summarizeRow(row: EducationRow): string {
    return String(
      row['title']
      ?? row['review_code']
      ?? row['criterion_code']
      ?? row['resolution_code']
      ?? row['transfer_code']
      ?? row['publication_code']
      ?? row['issuance_code']
      ?? row['assignment_code']
      ?? row['issue_code']
      ?? row['document_code']
      ?? row['version_label']
      ?? row['appeal_code']
      ?? row['criterion_label']
      ?? row['requirement_label']
      ?? row['entry_title']
      ?? row['topic_title']
      ?? row['entity_label']
      ?? row['publication_reference']
      ?? row['document_title']
      ?? row['responsible_name']
      ?? row['actor_name']
      ?? row['holder_name']
      ?? row['full_name']
      ?? row['owner_name']
      ?? row['panel_name']
      ?? row['decision_code']
      ?? row['dossier_code']
      ?? row['portfolio_code']
      ?? row['employee_code']
      ?? row['case_code']
      ?? row['grant_code']
      ?? row['evaluation_code']
      ?? row['declaration_code']
      ?? row['document_number']
      ?? row['id']
      ?? '-',
    );
  }

  protected exportPdf(resource: EducationResourceConfig): void {
    this.exportResource(resource, 'pdf');
  }

  protected exportCsv(resource: EducationResourceConfig): void {
    this.exportResource(resource, 'csv');
  }

  protected childStoreKey(child: EducationDetailChildResourceConfig, parentRow: EducationRow): string {
    return `${child.key}:${String(parentRow['id'] ?? '')}`;
  }

  private loadResource(resource: EducationResourceConfig, query: TableQuery): void {
    this.queryStore.set({ ...this.queryStore(), [resource.key]: query });
    this.loadingKey.set(resource.key);

    this.http.get<PagedResponse<EducationRow>>(resource.endpoint, { params: this.toParams(query) }).subscribe({
      next: (response) => {
        this.rowStore.set({ ...this.rowStore(), [resource.key]: response.items ?? [] });
        this.totalStore.set({ ...this.totalStore(), [resource.key]: response.total ?? 0 });
        this.loadingKey.set('');
      },
      error: () => {
        this.rowStore.set({ ...this.rowStore(), [resource.key]: [] });
        this.totalStore.set({ ...this.totalStore(), [resource.key]: 0 });
        this.loadingKey.set('');
      },
    });
  }

  private openDetail(resource: EducationResourceConfig, row: EducationRow): void {
    const recordID = String(row['id'] ?? '').trim();
    const requestSequence = ++this.detailRequestSequence;

    this.selectedResource.set(resource);
    this.selectedRow.set(row);
    this.detailLoading.set(false);
    this.detailDialogOpen.set(true);
    this.activeDetailTab.set('overview');

    if (!recordID) {
      this.loadAllChildren(resource, row);
      return;
    }

    this.detailLoading.set(true);
    this.http.get<EducationRow>(`${resource.endpoint}/${recordID}`).subscribe({
      next: (detail) => {
        if (requestSequence !== this.detailRequestSequence) {
          return;
        }
        this.selectedRow.set(detail);
        this.detailLoading.set(false);
        this.loadAllChildren(resource, detail);
      },
      error: () => {
        if (requestSequence !== this.detailRequestSequence) {
          return;
        }
        this.detailLoading.set(false);
        this.messages.add({
          severity: 'warn',
          summary: 'Detaliile complete nu sunt disponibile',
          detail: 'Afisam datele disponibile din tabel si subregistrele asociate.',
        });
        this.loadAllChildren(resource, row);
      },
    });
  }

  private loadAllChildren(resource: EducationResourceConfig, row: EducationRow): void {
    for (const child of resource.detailChildren ?? []) {
      this.ensureChildLoaded(child, row);
    }
  }

  private ensureChildLoaded(child: EducationDetailChildResourceConfig, parentRow: EducationRow): void {
    const key = this.childStoreKey(child, parentRow);
    if (!(key in this.childRowStore())) {
      this.loadChildResource(child, parentRow, this.childQuery(child, parentRow));
    }
  }

  private loadChildResource(child: EducationDetailChildResourceConfig, parentRow: EducationRow, query: TableQuery): void {
    const key = this.childStoreKey(child, parentRow);
    this.childQueryStore.set({ ...this.childQueryStore(), [key]: query });
    this.childLoadingKey.set(key);

    this.http.get<PagedResponse<EducationRow>>(child.listEndpoint(parentRow), { params: this.toParams(query) }).subscribe({
      next: (response) => {
        this.childRowStore.set({ ...this.childRowStore(), [key]: response.items ?? [] });
        this.childTotalStore.set({ ...this.childTotalStore(), [key]: response.total ?? 0 });
        if (this.childLoadingKey() === key) {
          this.childLoadingKey.set('');
        }
      },
      error: () => {
        this.childRowStore.set({ ...this.childRowStore(), [key]: [] });
        this.childTotalStore.set({ ...this.childTotalStore(), [key]: 0 });
        if (this.childLoadingKey() === key) {
          this.childLoadingKey.set('');
        }
      },
    });
  }

  private formatCell(value: unknown): string {
    if (typeof value === 'boolean') {
      return value ? 'Da' : 'Nu';
    }
    if (value == null || value === '') {
      return '-';
    }
    return String(value);
  }

  private formatConfiguredValue(column: EducationColumn, value: unknown): string {
    if (column.options?.length) {
      const match = column.options.find((option) => option.value === value);
      if (match) {
        return match.label;
      }
    }
    if (column.type === 'boolean') {
      return value ? 'Da' : 'Nu';
    }
    if (value == null || value === '') {
      return '-';
    }
    return String(value);
  }

  private tagSeverity(value: unknown): 'success' | 'info' | 'warn' | 'danger' | 'secondary' {
    if (typeof value === 'boolean') {
      return value ? 'success' : 'secondary';
    }

    const normalized = String(value ?? '').toLowerCase();
    if (['approved', 'published', 'validated', 'active', 'finalized', 'funded', 'completed', 'held', 'reviewed', 'true', 'implemented', 'received', 'prezent', 'verificat', 'publicat', 'adoptat', 'complet', 'activ', 'finalizata', 'nu_este_necesara', 'receptionat', 'inchis', 'acceptat', 'realizat', 'accepted', 'resolved', 'semnat', 'transmis', 'confirmat', 'admis', 'finantat', 'solutionat', 'foarte_bine', 'bine'].includes(normalized)) {
      return 'success';
    }
    if (['draft', 'scheduled', 'submitted', 'in_review', 'review', 'in_progress', 'partial', 'consultation', 'consultare', 'pending', 'pending_anonymization', 'internal', 'prepared', 'sent', 'endorsed', 'working_group', 'cp_endorsed', 'ca_approved', 'registered', 'none', 'invitat', 'declarat', 'intern', 'anonimizare_necesara', 'portofoliu', 'dosar_personal', 'dosar_director', 'dosar_director_adjunct', 'amanat', 'in_verificare', 'pregatit_publicare', 'necesara', 'pregatit', 'trimis', 'depunere', 'verificare_secretariat', 'validare_manageriala', 'reverificare', 'completari', 'de_stabilit', 'in_urmarire', 'preluare', 'consultare', 'transfer', 'arhivare', 'restituire', 'fizic', 'digital', 'mixt', 'elaborare', 'avizare_cp', 'aprobare_ca', 'publicare', 'inregistrare', 'redactare', 'consultare_publica', 'identificare', 'studii', 'cariera', 'evaluare', 'declaratie', 'medical', 'disciplina', 'management', 'intern', 'confidential', 'strict_confidential', 'actualizare', 'export', 'cerere', 'adeverinta', 'aviz', 'fisa_evaluare', 'decizie', 'anexa', 'emitere', 'vechime', 'performanta', 'social', 'administrativ', 'autoevaluare', 'impact', 'dezvoltare', 'incluziune', 'evaluare_comisie', 'validare_finala', 'extras', 'comunicare', 'dispozitie', 'analiza_juridica', 'anonimizare', 'aprobare_publicare', 'validare_dosar', 'repartizare', 'detasare', 'solutionare_contestatie', 'raport_final', 'evaluare_initiala', 'extras_punctaj', 'rezerva', 'emis', 'redistribuit', 'propus', 'deschis', 'in_cercetare', 'proiectare', 'predare', 'management_clasa', 'parteneriat', 'reviewed', 'satisfacator'].includes(normalized)) {
      return 'warn';
    }
    if (['blocked', 'expired', 'rejected', 'inactive', 'false', 'planned', 'absent_motivat', 'absent_nemotivat', 'respins', 'lipsa', 'suspendat', 'expirat', 'retras', 'returned', 'cancelled', 'returnat', 'contestat', 'contested', 'nesatisfacator'].includes(normalized)) {
      return 'danger';
    }
    if (['archived', 'waived'].includes(normalized)) {
      return 'secondary';
    }
    return 'secondary';
  }

  private toParams(query: TableQuery): HttpParams {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      params = params.set(`filter.${key}`, String(value));
    }

    return params;
  }

  private normalizeValue(value: string | number | boolean | Date | null): string | number | boolean | null {
    if (value instanceof Date) {
      return value.toISOString().slice(0, 10);
    }
    return value;
  }

  private normalizeFilters(filters: ServerTableFilterState): ServerTableFilterState {
    return Object.fromEntries(
      Object.entries(filters).filter(([, value]) => String(value ?? '').trim() !== ''),
    );
  }

  protected closeEditorDialog(): void {
    this.editorDialogOpen.set(false);
    this.editorTarget.set(null);
    this.editingRecordID.set('');
    this.editorMode.set('create');
    this.clearCreateForm();
  }

  protected closeDeleteDialog(): void {
    this.deleteDialogOpen.set(false);
    this.deleting.set(false);
    this.deleteTarget.set(null);
  }

  private seedCreateForm(fields: EducationResourceConfig['createFields'], row?: EducationRow): void {
    this.clearCreateForm();
    for (const field of fields) {
      if (!row) {
        this.createForm[field.field] = field.defaultValue ?? (field.type === 'boolean' ? false : '');
        continue;
      }

      const rawValue = row[field.field];
      if (field.type === 'date') {
        this.createForm[field.field] = typeof rawValue === 'string' && rawValue !== '' ? new Date(`${rawValue}T00:00:00`) : null;
        continue;
      }
      if (field.type === 'boolean') {
        this.createForm[field.field] = Boolean(rawValue);
        continue;
      }
      if (field.type === 'number') {
        const parsed = typeof rawValue === 'number' ? rawValue : Number(rawValue ?? field.defaultValue ?? 0);
        this.createForm[field.field] = Number.isFinite(parsed) ? parsed : (field.defaultValue as number | undefined) ?? 0;
        continue;
      }
      this.createForm[field.field] = rawValue == null ? (field.defaultValue ?? '') : String(rawValue);
    }
  }

  private clearCreateForm(): void {
    for (const key of Object.keys(this.createForm)) {
      delete this.createForm[key];
    }
  }

  private exportResource(resource: EducationResourceConfig, format: 'pdf' | 'csv'): void {
    const currentRows = this.rows(resource);
    const exportLimit = 5000;
    const exportCount = Math.min(Math.max(this.total(resource), currentRows.length, 1), exportLimit);
    const exportQuery: TableQuery = {
      ...this.query(resource),
      page: 1,
      pageSize: exportCount,
    };

    if (this.total(resource) > exportLimit) {
      this.messages.add({
        severity: 'warn',
        summary: 'Export limitat',
        detail: `Exportam primele ${exportLimit} inregistrari pentru ${resource.label}.`,
      });
    }

    this.exportingKey.set(`${resource.key}:${format}`);
    this.http.get<PagedResponse<EducationRow>>(resource.endpoint, { params: this.toParams(exportQuery) }).subscribe({
      next: (response) => {
        const items = response.items ?? [];
        if (!items.length) {
          this.exportingKey.set('');
          this.messages.add({
            severity: 'warn',
            summary: 'Nu exista date pentru export',
            detail: 'Nu exista inregistrari pentru filtrele curente.',
          });
          return;
        }

        const payload = this.buildExportPayload(resource, items);
        this.http.post(`/api/education/exports/${format}`, payload, { responseType: 'blob' as const }).subscribe({
          next: (blob) => {
            this.downloadBlob(blob, `${payload.filename}.${format === 'pdf' ? 'pdf' : 'csv'}`);
            this.exportingKey.set('');
          },
          error: () => {
            this.exportingKey.set('');
            this.messages.add({
              severity: 'error',
              summary: 'Exportul a esuat',
              detail: 'Fisierul nu a putut fi generat cu configuratia actuala a backendului.',
            });
          },
        });
      },
      error: () => {
        this.exportingKey.set('');
        this.messages.add({
          severity: 'error',
          summary: 'Exportul a esuat',
          detail: 'Nu am putut incarca datele necesare pentru export.',
        });
      },
    });
  }

  private buildExportPayload(resource: EducationResourceConfig, rows: EducationRow[]): EducationExportPayload {
    return {
      title: resource.label,
      filename: resource.key,
      headers: resource.columns.map((column) => column.header),
      rows: rows.map((row) =>
        resource.columns.map((column) => this.displayConfiguredCell(resource.columns, row, column.field)),
      ),
    };
  }

  private downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  private downloadResourcePdf(resource: EducationResourceConfig, row: EducationRow): void {
    if (!resource.pdfEndpoint) {
      return;
    }
    const fallbackFilename = resource.pdfFilename?.(row) ?? `${resource.key}-${String(row['id'] ?? 'document')}.pdf`;
    this.downloadPdf(resource.pdfEndpoint(row), fallbackFilename);
  }

  private downloadChildPdf(child: EducationDetailChildResourceConfig, parentRow: EducationRow, row: EducationRow): void {
    if (!child.pdfEndpoint) {
      return;
    }
    const fallbackFilename = child.pdfFilename?.(parentRow, row) ?? `${child.key}-${String(row['id'] ?? 'document')}.pdf`;
    this.downloadPdf(child.pdfEndpoint(parentRow, row), fallbackFilename);
  }

  private downloadPdf(endpoint: string, fallbackFilename: string): void {
    this.http.get(endpoint, { observe: 'response', responseType: 'blob' }).subscribe({
      next: (response) => {
        this.downloadBlob(response.body ?? new Blob(), this.extractFilename(response.headers.get('Content-Disposition')) || fallbackFilename);
      },
      error: () => {
        this.messages.add({
          severity: 'error',
          summary: 'Generarea PDF a esuat',
          detail: 'Documentul nu a putut fi generat cu configuratia actuala a backendului.',
        });
      },
    });
  }

  private extractFilename(contentDisposition: string | null): string {
    const match = /filename="([^"]+)"/i.exec(contentDisposition ?? '');
    return match?.[1]?.trim() ?? '';
  }

  protected coerceTabValue(value: unknown, fallback = ''): string {
    const normalized = String(value ?? '').trim();
    return normalized || fallback;
  }

  private editorTargetLabel(target: EditorTarget): string {
    return target.kind === 'resource' ? target.resource.label : target.child.label;
  }

  private childDetailEndpoint(child: EducationDetailChildResourceConfig, parentRow: EducationRow, row: EducationRow): string {
    if (child.detailEndpoint) {
      return child.detailEndpoint(parentRow, row);
    }
    return `${child.listEndpoint(parentRow)}/${row['id']}`;
  }
}
