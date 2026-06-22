import { CommonModule } from '@angular/common';
import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DatePickerModule } from 'primeng/datepicker';
import { DialogModule } from 'primeng/dialog';
import { InputNumberModule } from 'primeng/inputnumber';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TableLazyLoadEvent, TableModule } from 'primeng/table';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToggleSwitchModule } from 'primeng/toggleswitch';
import { TooltipModule } from 'primeng/tooltip';

import { PagedResponse, TableQuery } from '../../core/api/api.types';

type FieldType = 'text' | 'textarea' | 'number' | 'date' | 'select' | 'boolean';
type ColumnType = 'text' | 'tag' | 'date' | 'boolean' | 'number';

interface EducationColumn {
  field: string;
  header: string;
  type?: ColumnType;
  sortable?: boolean;
  filter?: 'text' | 'select';
  options?: Array<{ label: string; value: string }>;
  width?: string;
}

interface EducationFormField {
  field: string;
  label: string;
  type: FieldType;
  required?: boolean;
  options?: Array<{ label: string; value: string | boolean }>;
  defaultValue?: string | number | boolean;
  wide?: boolean;
}

interface EducationResource {
  key: string;
  label: string;
  icon: string;
  description: string;
  endpoint: string;
  createEndpoint: string;
  allowCreate?: boolean;
  permissionHint: string;
  columns: EducationColumn[];
  createFields: EducationFormField[];
  emptyText: string;
}

interface EducationGroup {
  key: string;
  label: string;
  icon: string;
  resources: EducationResource[];
}

interface DashboardCard {
  label: string;
  value: number | string;
  icon: string;
  tone: 'primary' | 'green' | 'orange' | 'blue';
}

@Component({
  selector: 'app-education-workspace',
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    CardModule,
    DatePickerModule,
    DialogModule,
    InputNumberModule,
    InputTextModule,
    SelectModule,
    TableModule,
    TabsModule,
    TagModule,
    TextareaModule,
    ToggleSwitchModule,
    TooltipModule,
  ],
  template: `
    <section class="education-workspace flex h-[calc(100dvh-6rem)] min-h-0 flex-col overflow-hidden">
      <p-tabs value="dashboard" class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <p-tablist class="shrink-0">
          <p-tab value="dashboard">
            <span class="inline-flex items-center gap-2"><i class="pi pi-chart-bar"></i> Dashboard</span>
          </p-tab>
          @for (group of groups; track group.key) {
            <p-tab [value]="group.key">
              <span class="inline-flex items-center gap-2"><i [class]="group.icon"></i> {{ group.label }}</span>
            </p-tab>
          }
        </p-tablist>

        <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
          <p-tabpanel value="dashboard" class="flex min-h-0 flex-1 overflow-auto p-4">
            <div class="grid w-full gap-4 xl:grid-cols-[1fr_0.85fr]">
              <section class="grid content-start gap-4 sm:grid-cols-2 xl:grid-cols-3">
                @for (card of dashboardCards(); track card.label) {
                  <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
                    <div class="flex items-center justify-between gap-3">
                      <div>
                        <p class="m-0 text-sm text-muted-color">{{ card.label }}</p>
                        <strong class="mt-1 block text-3xl tracking-[-0.04em]">{{ card.value }}</strong>
                      </div>
                      <span class="grid size-12 place-items-center rounded-2xl" [ngClass]="dashboardTone(card.tone)">
                        <i [class]="card.icon"></i>
                      </span>
                    </div>
                  </p-card>
                }
              </section>

              <section class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm dark:bg-surface-900">
                <h2 class="m-0 text-xl font-bold">Catalog funcțional educație</h2>
                <p class="mt-2 text-sm text-muted-color">
                  Modulul nu mai este o pagină goală: fiecare arie folosește tabele PrimeNG lazy cu paginare, sortare, filtre server-side și acțiuni pe rând.
                </p>
                <div class="mt-4 grid gap-3">
                  @for (group of groups; track group.key) {
                    <div class="rounded-2xl border border-surface p-3">
                      <div class="mb-2 flex items-center gap-2 font-semibold"><i [class]="group.icon"></i>{{ group.label }}</div>
                      <div class="flex flex-wrap gap-2">
                        @for (resource of group.resources; track resource.key) {
                          <p-tag [value]="resource.label" severity="secondary" />
                        }
                      </div>
                    </div>
                  }
                </div>
              </section>
            </div>
          </p-tabpanel>

          @for (group of groups; track group.key) {
            <p-tabpanel [value]="group.key" class="flex min-h-0 flex-1 overflow-hidden p-0">
              <p-tabs [value]="group.resources[0].key" class="flex min-h-0 flex-1 flex-col overflow-hidden" (valueChange)="activateResource($event ?? '')">
                <p-tablist class="shrink-0">
                  @for (resource of group.resources; track resource.key) {
                    <p-tab [value]="resource.key">{{ resource.label }}</p-tab>
                  }
                </p-tablist>
                <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
                  @for (resource of group.resources; track resource.key) {
                    <p-tabpanel [value]="resource.key" class="flex min-h-0 flex-1 overflow-hidden p-0">
                      <div class="flex min-h-0 flex-1 flex-col gap-3 p-3">
                        <div class="flex shrink-0 flex-wrap items-center justify-between gap-3">
                          <div>
                            <h2 class="m-0 text-xl font-semibold">{{ resource.label }}</h2>
                            <p class="m-0 mt-1 text-sm text-muted-color">{{ resource.description }}</p>
                          </div>
                          <div class="flex flex-wrap items-center gap-2">
                            <p-tag [value]="resource.permissionHint" severity="info" />
                            @if (resource.allowCreate !== false) {
                              <p-button label="Adaugă" icon="pi pi-plus" size="small" (onClick)="openCreate(resource)" />
                            }
                          </div>
                        </div>

                        <div class="min-h-0 flex-1 overflow-hidden rounded-2xl border border-surface bg-surface-0 dark:bg-surface-900">
                          <p-table
                            [value]="rows(resource)"
                            [lazy]="true"
                            [loading]="loadingKey() === resource.key"
                            [paginator]="true"
                            [rows]="query(resource).pageSize"
                            [first]="(query(resource).page - 1) * query(resource).pageSize"
                            [totalRecords]="total(resource)"
                            [rowsPerPageOptions]="[10,20,50,100]"
                            [showCurrentPageReport]="true"
                            currentPageReportTemplate="Afișare {first} - {last} din {totalRecords} înregistrări"
                            [scrollable]="true"
                            scrollHeight="flex"
                            styleClass="p-datatable-sm p-datatable-gridlines education-table"
                            (onLazyLoad)="loadResource(resource, $event)"
                          >
                            <ng-template pTemplate="header">
                              <tr>
                                @for (column of resource.columns; track column.field) {
                                  <th [pSortableColumn]="column.sortable ? column.field : undefined" [style.width]="column.width" [style.min-width]="column.width">
                                    {{ column.header }}
                                    @if (column.sortable) {
                                      <p-sortIcon [field]="column.field" />
                                    }
                                  </th>
                                }
                                <th class="text-center" style="width: 10rem">Acțiuni</th>
                              </tr>
                              <tr class="filter-row">
                                @for (column of resource.columns; track column.field) {
                                  <th>
                                    @if (column.filter === 'select') {
                                      <p-select appendTo="body" class="w-full" [options]="column.options ?? []" [showClear]="true" placeholder="Toate" [(ngModel)]="filtersFor(resource)[column.field]" (onChange)="reload(resource)" />
                                    } @else if (column.filter === 'text') {
                                      <input pInputText class="w-full filter-input" [placeholder]="column.header" [(ngModel)]="filtersFor(resource)[column.field]" (keyup.enter)="reload(resource)" />
                                    }
                                  </th>
                                }
                                <th></th>
                              </tr>
                            </ng-template>
                            <ng-template pTemplate="body" let-row>
                              <tr>
                                @for (column of resource.columns; track column.field) {
                                  <td>
                                    @switch (column.type) {
                                      @case ('tag') {
                                        <p-tag [value]="displayCell(row, column.field)" [severity]="tagSeverity(cell(row, column.field))" />
                                      }
                                      @case ('boolean') {
                                        <p-tag [value]="cell(row, column.field) ? 'Da' : 'Nu'" [severity]="cell(row, column.field) ? 'success' : 'secondary'" />
                                      }
                                      @case ('number') {
                                        <span class="font-semibold">{{ cell(row, column.field) }}</span>
                                      }
                                      @default {
                                        <span class="line-clamp-2">{{ cell(row, column.field) || '-' }}</span>
                                      }
                                    }
                                  </td>
                                }
                                <td class="text-center">
                                  <div class="inline-flex items-center gap-1">
                                    <p-button icon="pi pi-eye" [rounded]="true" [text]="true" pTooltip="Deschide" (onClick)="selectRow(resource, row)" />
                                    <p-button icon="pi pi-sitemap" [rounded]="true" [text]="true" pTooltip="Atașează flux" />
                                    <p-button icon="pi pi-file-plus" [rounded]="true" [text]="true" pTooltip="Documente" />
                                  </div>
                                </td>
                              </tr>
                            </ng-template>
                            <ng-template pTemplate="emptymessage">
                              <tr>
                                <td [attr.colspan]="resource.columns.length + 1" class="py-8 text-center text-muted-color">{{ resource.emptyText }}</td>
                              </tr>
                            </ng-template>
                          </p-table>
                        </div>
                      </div>
                    </p-tabpanel>
                  }
                </p-tabpanels>
              </p-tabs>
            </p-tabpanel>
          }
        </p-tabpanels>
      </p-tabs>

      <p-dialog
        [visible]="createDialogOpen()"
        (visibleChange)="createDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [header]="activeCreateResource() ? 'Adaugă ' + activeCreateResource()?.label : 'Adaugă'"
        [style]="{ width: 'min(58rem, 94vw)' }"
      >
        @if (activeCreateResource(); as resource) {
          <div class="grid gap-4 md:grid-cols-2">
            @for (field of resource.createFields; track field.field) {
              <label class="education-field" [class.md:col-span-2]="field.wide">
                <span>{{ field.label }} @if (field.required) { <strong class="text-primary">*</strong> }</span>
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
                    <p-select appendTo="body" [options]="field.options ?? []" [(ngModel)]="createForm[field.field]" />
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
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="createDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" [loading]="creating()" (onClick)="createRecord()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="detailDialogOpen()"
        (visibleChange)="detailDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        header="Detalii înregistrare"
        [style]="{ width: 'min(52rem, 94vw)' }"
      >
        @if (selectedResource(); as resource) {
          @if (selectedRow(); as row) {
            <div class="grid gap-3 md:grid-cols-2">
              @for (column of resource.columns; track column.field) {
                <div class="rounded-xl border border-surface p-3">
                  <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">{{ column.header }}</div>
                  <div class="mt-1 font-medium">{{ cell(row, column.field) || '-' }}</div>
                </div>
              }
            </div>
          }
        }
      </p-dialog>
    </section>
  `,
  styles: `
    :host {
      display: block;
      min-height: 0;
    }

    :host ::ng-deep .education-workspace .p-tabs,
    :host ::ng-deep .education-workspace .p-tabpanels,
    :host ::ng-deep .education-workspace .p-tabpanel {
      display: flex;
      flex: 1 1 auto;
      min-height: 0;
      flex-direction: column;
      overflow: hidden;
      background: transparent;
      color: var(--p-text-color);
    }

    :host ::ng-deep .education-table,
    :host ::ng-deep .education-table .p-datatable-table-container,
    :host ::ng-deep .education-table .p-datatable-table,
    :host ::ng-deep .education-table .p-datatable-tbody > tr,
    :host ::ng-deep .education-table .p-datatable-tbody > tr > td {
      background: var(--p-content-background);
      color: var(--p-text-color);
    }

    :host ::ng-deep .education-table .p-datatable-thead {
      position: sticky;
      top: 0;
      z-index: 3;
    }

    :host ::ng-deep .education-table .p-datatable-thead > tr > th,
    :host ::ng-deep .education-table .filter-row > th {
      background: var(--p-surface-50);
      color: var(--p-text-color);
    }

    :host-context(.app-dark) ::ng-deep .education-table,
    :host-context(.app-dark) ::ng-deep .education-table .p-datatable-table-container,
    :host-context(.app-dark) ::ng-deep .education-table .p-datatable-table,
    :host-context(.app-dark) ::ng-deep .education-table .p-datatable-tbody > tr,
    :host-context(.app-dark) ::ng-deep .education-table .p-datatable-tbody > tr > td {
      background: var(--p-surface-900);
    }

    :host-context(.app-dark) ::ng-deep .education-table .p-datatable-thead > tr > th,
    :host-context(.app-dark) ::ng-deep .education-table .filter-row > th {
      background: var(--p-surface-800);
    }

    .filter-input {
      height: 2rem;
      font-size: 0.8125rem;
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
export class EducationWorkspaceComponent {
  private readonly http = inject(HttpClient);

  protected readonly groups: EducationGroup[] = [
    {
      key: 'governance',
      label: 'Guvernanță',
      icon: 'pi pi-users',
      resources: [
        this.requirementsResource(),
        this.meetingsResource(),
        this.decisionsResource(),
        this.managerialResource(),
        this.regulationsResource(),
      ],
    },
    {
      key: 'personnel',
      label: 'Personal',
      icon: 'pi pi-id-card',
      resources: [
        this.personnelResource(),
        this.evaluationsResource(),
        this.declarationsResource(),
        this.mobilityResource(),
        this.meritResource(),
      ],
    },
    {
      key: 'portfolio',
      label: 'Portofoliu CD',
      icon: 'pi pi-folder-open',
      resources: [
        this.portfolioResource(),
        this.portfolioSectionsResource(),
      ],
    },
  ];

  protected readonly rowStore = signal<Record<string, Record<string, unknown>[]>>({});
  protected readonly totalStore = signal<Record<string, number>>({});
  protected readonly queryStore = signal<Record<string, TableQuery>>({});
  protected readonly filterStore = signal<Record<string, Record<string, string>>>({});
  protected readonly loadingKey = signal('');
  protected readonly createDialogOpen = signal(false);
  protected readonly detailDialogOpen = signal(false);
  protected readonly creating = signal(false);
  protected readonly activeCreateResource = signal<EducationResource | null>(null);
  protected readonly selectedResource = signal<EducationResource | null>(null);
  protected readonly selectedRow = signal<Record<string, unknown> | null>(null);
  protected readonly dashboardCards = signal<DashboardCard[]>([
    { label: 'Ședințe guvernanță', value: '-', icon: 'pi pi-calendar', tone: 'primary' },
    { label: 'Decizii CA/CP', value: '-', icon: 'pi pi-verified', tone: 'green' },
    { label: 'Dosare manageriale', value: '-', icon: 'pi pi-briefcase', tone: 'blue' },
    { label: 'Personal activ', value: '-', icon: 'pi pi-id-card', tone: 'orange' },
    { label: 'Portofolii validate', value: '-', icon: 'pi pi-folder-open', tone: 'green' },
  ]);
  protected readonly createForm: Record<string, string | number | boolean | Date | null> = {};

  constructor() {
    this.loadDashboard();
  }

  protected rows(resource: EducationResource): Record<string, unknown>[] {
    return this.rowStore()[resource.key] ?? [];
  }

  protected total(resource: EducationResource): number {
    return this.totalStore()[resource.key] ?? 0;
  }

  protected query(resource: EducationResource): TableQuery {
    return this.queryStore()[resource.key] ?? { page: 1, pageSize: 20, sort: resource.columns[0]?.field, direction: 'asc', filters: {} };
  }

  protected filtersFor(resource: EducationResource): Record<string, string> {
    const store = this.filterStore();
    if (!store[resource.key]) {
      store[resource.key] = {};
      this.filterStore.set({ ...store });
    }
    return store[resource.key];
  }

  protected activateResource(value: string | number): void {
    const resource = this.findResource(String(value));
    if (resource && !this.rowStore()[resource.key]) {
      this.loadResource(resource);
    }
  }

  protected loadResource(resource: EducationResource, event?: TableLazyLoadEvent): void {
    const pageSize = event?.rows ?? this.query(resource).pageSize;
    const page = Math.floor((event?.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? event?.sortField[0] : event?.sortField;
    const filters = Object.fromEntries(Object.entries(this.filtersFor(resource)).filter(([, value]) => String(value ?? '').trim() !== ''));
    const query: TableQuery = {
      page,
      pageSize,
      sort: sort || this.query(resource).sort,
      direction: event?.sortOrder === -1 ? 'desc' : 'asc',
      filters,
    };
    this.queryStore.set({ ...this.queryStore(), [resource.key]: query });
    this.loadingKey.set(resource.key);
    this.http.get<PagedResponse<Record<string, unknown>>>(resource.endpoint, { params: this.toParams(query) }).subscribe({
      next: (res) => {
        this.rowStore.set({ ...this.rowStore(), [resource.key]: res.items ?? [] });
        this.totalStore.set({ ...this.totalStore(), [resource.key]: res.total ?? 0 });
        this.loadingKey.set('');
      },
      error: () => {
        this.rowStore.set({ ...this.rowStore(), [resource.key]: [] });
        this.totalStore.set({ ...this.totalStore(), [resource.key]: 0 });
        this.loadingKey.set('');
      },
    });
  }

  protected reload(resource: EducationResource): void {
    this.loadResource(resource, { first: 0, rows: this.query(resource).pageSize });
  }

  protected openCreate(resource: EducationResource): void {
    this.activeCreateResource.set(resource);
    for (const field of resource.createFields) {
      this.createForm[field.field] = field.defaultValue ?? (field.type === 'boolean' ? false : '');
    }
    this.createDialogOpen.set(true);
  }

  protected createRecord(): void {
    const resource = this.activeCreateResource();
    if (!resource) {
      return;
    }
    if (resource.allowCreate === false) {
      this.createDialogOpen.set(false);
      return;
    }
    const payload = Object.fromEntries(Object.entries(this.createForm).map(([key, value]) => [key, this.normalizeValue(value)]));
    this.creating.set(true);
    this.http.post(resource.createEndpoint, payload).subscribe({
      next: () => {
        this.creating.set(false);
        this.createDialogOpen.set(false);
        this.reload(resource);
        this.loadDashboard();
      },
      error: () => {
        this.creating.set(false);
      },
    });
  }

  protected selectRow(resource: EducationResource, row: Record<string, unknown>): void {
    this.selectedResource.set(resource);
    this.selectedRow.set(row);
    this.detailDialogOpen.set(true);
  }

  protected cell(row: Record<string, unknown>, field: string): string | number | boolean {
    const value = row[field];
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      return value;
    }
    return '';
  }

  protected displayCell(row: Record<string, unknown>, field: string): string {
    const value = this.cell(row, field);
    return value === '' ? '-' : String(value);
  }

  protected tagSeverity(value: string | number | boolean): 'success' | 'info' | 'warn' | 'danger' | 'secondary' {
    const normalized = String(value).toLowerCase();
    if (['approved', 'published', 'validat', 'validata', 'active', 'activ', 'finalizat'].includes(normalized)) {
      return 'success';
    }
    if (['draft', 'programat', 'scheduled', 'in_review', 'consultare'].includes(normalized)) {
      return 'warn';
    }
    if (['blocked', 'expired', 'anulat', 'rejected'].includes(normalized)) {
      return 'danger';
    }
    return 'secondary';
  }

  protected dashboardTone(tone: DashboardCard['tone']): string {
    return {
      primary: 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-100',
      green: 'bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-100',
      orange: 'bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-100',
      blue: 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-100',
    }[tone];
  }

  private loadDashboard(): void {
    this.http.get<{ stats: Record<string, number> }>('/api/education/dashboard').subscribe((res) => {
      const governance = res.stats ?? {};
      this.dashboardCards.update((cards) => cards.map((card) => card.label === 'Ședințe guvernanță' ? { ...card, value: governance['total_meetings'] ?? 0 } : card));
    });
    this.http.get<{ stats: Record<string, number> }>('/api/education/decisions/dashboard').subscribe((res) => {
      const stats = res.stats ?? {};
      this.dashboardCards.update((cards) => cards.map((card) => card.label === 'Decizii CA/CP' ? { ...card, value: stats['total_decisions'] ?? 0 } : card));
    });
    this.http.get<{ stats: Record<string, number> }>('/api/education/managerial/dashboard').subscribe((res) => {
      const stats = res.stats ?? {};
      this.dashboardCards.update((cards) => cards.map((card) => card.label === 'Dosare manageriale' ? { ...card, value: stats['total_dossiers'] ?? 0 } : card));
    });
    this.http.get<{ stats: Record<string, number> }>('/api/education/personnel/dashboard').subscribe((res) => {
      const stats = res.stats ?? {};
      this.dashboardCards.update((cards) => cards.map((card) => card.label === 'Personal activ' ? { ...card, value: stats['active_records'] ?? 0 } : card));
    });
    this.http.get<{ stats: Record<string, number> }>('/api/education/portfolios/dashboard').subscribe((res) => {
      const stats = res.stats ?? {};
      this.dashboardCards.update((cards) => cards.map((card) => card.label === 'Portofolii validate' ? { ...card, value: stats['validated_portfolios'] ?? 0 } : card));
    });
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
      params = params.set(`filter.${key}`, value);
    }
    return params;
  }

  private normalizeValue(value: string | number | boolean | Date | null): string | number | boolean | null {
    if (value instanceof Date) {
      return value.toISOString().slice(0, 10);
    }
    return value;
  }

  private findResource(key: string): EducationResource | undefined {
    return this.groups.flatMap((group) => group.resources).find((resource) => resource.key === key);
  }

  private requirementsResource(): EducationResource {
    return {
      key: 'requirements',
      label: 'Cerințe legale',
      icon: 'pi pi-bookmark',
      endpoint: '/api/education/requirements',
      createEndpoint: '',
      allowCreate: false,
      permissionHint: 'education.read',
      description: 'Catalog trasabil din Legea 198/2023, ROFUIP, metodologii de portofoliu, mobilitate, evaluare și gradații.',
      columns: [
        { field: 'priority', header: 'Prioritate', type: 'number', sortable: true, width: '7rem' },
        { field: 'domain', header: 'Domeniu', sortable: true, filter: 'text', width: '10rem' },
        { field: 'title_ro', header: 'Cerință', sortable: true, filter: 'text', width: '28rem' },
        { field: 'source_ref', header: 'Sursă', sortable: true, filter: 'text', width: '18rem' },
        { field: 'requirement_type', header: 'Tip', type: 'tag', sortable: true, filter: 'text', width: '12rem' },
        { field: 'implementation_status', header: 'Implementare', type: 'tag', sortable: true, filter: 'select', options: [{ label: 'Implementat', value: 'implemented' }, { label: 'Parțial', value: 'partial' }, { label: 'Planificat', value: 'planned' }], width: '10rem' },
      ],
      createFields: [],
      emptyText: 'Catalogul de cerințe nu conține încă înregistrări.',
    };
  }

  private meetingsResource(): EducationResource {
    return {
      key: 'meetings',
      label: 'Ședințe CA/CP/CEAC',
      icon: 'pi pi-calendar',
      endpoint: '/api/education/governance/meetings',
      createEndpoint: '/api/education/governance/meetings',
      permissionHint: 'education.governance',
      description: 'Convocare, prezență, cvorum, agendă, minute, anexă, vot, semnături și custodie.',
      columns: [
        { field: 'school_year', header: 'An școlar', sortable: true, filter: 'text', width: '9rem' },
        { field: 'organism', header: 'Organism', sortable: true, filter: 'text', width: '10rem' },
        { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '20rem' },
        { field: 'meeting_type', header: 'Tip', sortable: true, filter: 'text', width: '10rem' },
        { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: 'meeting_date', header: 'Data', sortable: true, width: '9rem' },
        { field: 'participants_count', header: 'Participanți', type: 'number', sortable: true, width: '8rem' },
      ],
      createFields: [
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026', required: true },
        { field: 'organism', label: 'Organism', type: 'select', options: this.organismOptions(), defaultValue: 'CA', required: true },
        { field: 'title', label: 'Titlu', type: 'text', required: true, wide: true },
        { field: 'meeting_type', label: 'Tip ședință', type: 'select', options: [{ label: 'Ordinară', value: 'ordinara' }, { label: 'Extraordinară', value: 'extraordinara' }], defaultValue: 'ordinara' },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'draft' },
        { field: 'quorum_required', label: 'Cvorum necesar', type: 'number', defaultValue: 0 },
        { field: 'participants_count', label: 'Participanți', type: 'number', defaultValue: 0 },
        { field: 'meeting_date', label: 'Data ședinței', type: 'date' },
        { field: 'location', label: 'Locație', type: 'text' },
        { field: 'chairperson', label: 'Președinte', type: 'text' },
        { field: 'secretary_name', label: 'Secretar', type: 'text' },
        { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
      ],
      emptyText: 'Nu există ședințe pentru filtrele curente.',
    };
  }

  private decisionsResource(): EducationResource {
    return {
      key: 'decisions',
      label: 'Decizii CA/CP',
      icon: 'pi pi-verified',
      endpoint: '/api/education/decisions/records',
      createEndpoint: '/api/education/decisions/records',
      permissionHint: 'education.decisions',
      description: 'Decizii cu statut, bază legală, semnare, publicare și anonimizare.',
      columns: [
        { field: 'decision_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
        { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
        { field: 'organism', header: 'Organism', sortable: true, filter: 'text', width: '9rem' },
        { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
        { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: 'publication_status', header: 'Publicare', type: 'tag', sortable: true, width: '10rem' },
        { field: 'decision_date', header: 'Data', sortable: true, width: '9rem' },
      ],
      createFields: [
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026', required: true },
        { field: 'organism', label: 'Organism', type: 'select', options: this.organismOptions(), defaultValue: 'CA' },
        { field: 'title', label: 'Titlu decizie', type: 'text', wide: true, required: true },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'draft' },
        { field: 'publication_status', label: 'Status publicare', type: 'select', options: [{ label: 'Nepublicat', value: 'unpublished' }, { label: 'Publicat', value: 'published' }, { label: 'Anonimizare necesară', value: 'anonymization_required' }], defaultValue: 'unpublished' },
        { field: 'decision_date', label: 'Data deciziei', type: 'date' },
        { field: 'legal_basis', label: 'Bază legală', type: 'textarea', wide: true },
        { field: 'signed_by', label: 'Semnat de', type: 'text' },
        { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
      ],
      emptyText: 'Nu există decizii pentru filtrele curente.',
    };
  }

  private managerialResource(): EducationResource {
    return {
      key: 'managerial',
      label: 'Dosare manageriale',
      icon: 'pi pi-briefcase',
      endpoint: '/api/education/managerial/records',
      createEndpoint: '/api/education/managerial/records',
      permissionHint: 'education.managerial',
      description: 'PDI/PAS, plan anual, RAEI, rapoarte, organigramă, încadrare și orar.',
      columns: [
        { field: 'dossier_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
        { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
        { field: 'dossier_type', header: 'Tip', sortable: true, filter: 'text', width: '12rem' },
        { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
        { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: 'owner_name', header: 'Responsabil', sortable: true, filter: 'text', width: '12rem' },
        { field: 'due_on', header: 'Termen', sortable: true, width: '9rem' },
      ],
      createFields: [
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026', required: true },
        { field: 'dossier_type', label: 'Tip dosar', type: 'select', options: [{ label: 'PDI/PAS', value: 'pdi_pas' }, { label: 'Plan anual', value: 'plan_anual' }, { label: 'RAEI', value: 'raei' }, { label: 'Raport', value: 'raport' }], defaultValue: 'plan_anual' },
        { field: 'title', label: 'Titlu', type: 'text', wide: true, required: true },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'draft' },
        { field: 'owner_name', label: 'Responsabil', type: 'text' },
        { field: 'due_on', label: 'Termen', type: 'date' },
        { field: 'publication_required', label: 'Publicare necesară', type: 'boolean', defaultValue: false },
        { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
      ],
      emptyText: 'Nu există dosare manageriale pentru filtrele curente.',
    };
  }

  private regulationsResource(): EducationResource {
    return {
      key: 'regulations',
      label: 'ROF / ROI',
      icon: 'pi pi-book',
      endpoint: '/api/education/regulations/records',
      createEndpoint: '/api/education/regulations/records',
      permissionHint: 'education.regulations',
      description: 'Regulamente, consultare, aprobare, publicare și revizuire periodică.',
      columns: [
        { field: 'regulation_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
        { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
        { field: 'regulation_type', header: 'Tip', sortable: true, filter: 'text', width: '10rem' },
        { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
        { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: 'approval_status', header: 'Aprobare', type: 'tag', sortable: true, width: '10rem' },
        { field: 'review_due_on', header: 'Revizuire', sortable: true, width: '9rem' },
      ],
      createFields: [
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026' },
        { field: 'regulation_type', label: 'Tip regulament', type: 'select', options: [{ label: 'ROF', value: 'rof' }, { label: 'ROI', value: 'roi' }], defaultValue: 'rof' },
        { field: 'title', label: 'Titlu', type: 'text', wide: true, required: true },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'draft' },
        { field: 'approval_status', label: 'Status aprobare', type: 'select', options: [{ label: 'Consultare', value: 'consultare' }, { label: 'Aprobat', value: 'approved' }], defaultValue: 'consultare' },
        { field: 'owner_name', label: 'Responsabil', type: 'text' },
        { field: 'review_due_on', label: 'Revizuire până la', type: 'date' },
        { field: 'approved_on', label: 'Aprobat la', type: 'date' },
        { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
      ],
      emptyText: 'Nu există regulamente pentru filtrele curente.',
    };
  }

  private personnelResource(): EducationResource {
    return {
      key: 'personnel',
      label: 'Cadre didactice',
      icon: 'pi pi-id-card',
      endpoint: '/api/education/personnel/records',
      createEndpoint: '/api/education/personnel/records',
      permissionHint: 'education.personnel',
      description: 'Fișe personal, încadrare, statut, evaluare, mobilitate și portofoliu.',
      columns: [
        { field: 'employee_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
        { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
        { field: 'role_title', header: 'Funcție', sortable: true, filter: 'text', width: '14rem' },
        { field: 'employment_type', header: 'Încadrare', sortable: true, filter: 'text', width: '12rem' },
        { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: 'evaluation_status', header: 'Evaluare', type: 'tag', sortable: true, width: '10rem' },
        { field: 'has_portfolio', header: 'Portofoliu', type: 'boolean', sortable: true, width: '8rem' },
      ],
      createFields: [
        { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
        { field: 'role_title', label: 'Funcție', type: 'text' },
        { field: 'employment_type', label: 'Încadrare', type: 'select', options: [{ label: 'Titular', value: 'titular' }, { label: 'Suplinitor', value: 'suplinitor' }, { label: 'Auxiliar', value: 'auxiliar' }], defaultValue: 'titular' },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'active' },
        { field: 'evaluation_status', label: 'Evaluare', type: 'select', options: [{ label: 'Neîncepută', value: 'not_started' }, { label: 'În lucru', value: 'in_progress' }, { label: 'Finalizată', value: 'finalized' }], defaultValue: 'not_started' },
        { field: 'mobility_stage', label: 'Etapă mobilitate', type: 'text' },
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026' },
        { field: 'assigned_unit', label: 'Structură', type: 'text' },
        { field: 'phone', label: 'Telefon', type: 'text' },
        { field: 'email', label: 'Email', type: 'text' },
        { field: 'has_portfolio', label: 'Are portofoliu', type: 'boolean', defaultValue: false },
        { field: 'notes', label: 'Note', type: 'textarea', wide: true },
      ],
      emptyText: 'Nu există cadre pentru filtrele curente.',
    };
  }

  private evaluationsResource(): EducationResource {
    return this.simplePersonnelSubResource('evaluations', 'Evaluări anuale', '/api/education/evaluations/records', 'education.evaluations', 'Punctaj, evaluator, contestare și finalizare.');
  }

  private declarationsResource(): EducationResource {
    return this.simplePersonnelSubResource('declarations', 'Declarații', '/api/education/declarations/records', 'education.declarations', 'Declarații și adeverințe asociate cadrului didactic.');
  }

  private mobilityResource(): EducationResource {
    return this.simplePersonnelSubResource('mobility', 'Mobilitate', '/api/education/mobility/records', 'education.mobility', 'Cazuri mobilitate, transfer, etape, sursă și destinație.');
  }

  private meritResource(): EducationResource {
    return this.simplePersonnelSubResource('gradatii', 'Gradații', '/api/education/gradatii/records', 'education.gradatii', 'Gradații de merit, punctaj, comisie și finanțare.');
  }

  private portfolioResource(): EducationResource {
    return {
      key: 'portfolios',
      label: 'Portofolii CD',
      icon: 'pi pi-folder-open',
      endpoint: '/api/education/portfolios/records',
      createEndpoint: '/api/education/portfolios/records',
      permissionHint: 'education.portfolios',
      description: 'Opis, secțiuni, transfer digital, declarație autenticitate, consimțământ și retenție 3 ani.',
      columns: [
        { field: 'portfolio_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
        { field: 'owner_name', header: 'Titular', sortable: true, filter: 'text', width: '18rem' },
        { field: 'owner_role', header: 'Funcție', sortable: true, filter: 'text', width: '12rem' },
        { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
        { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: 'section_count', header: 'Secțiuni', type: 'number', sortable: true, width: '8rem' },
        { field: 'transfer_status', header: 'Transfer', type: 'tag', sortable: true, width: '10rem' },
        { field: 'retention_until', header: 'Retenție', sortable: true, width: '9rem' },
      ],
      createFields: [
        { field: 'owner_name', label: 'Titular', type: 'text', wide: true, required: true },
        { field: 'owner_role', label: 'Funcție', type: 'text' },
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026' },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'draft' },
        { field: 'section_count', label: 'Număr secțiuni', type: 'number', defaultValue: 0 },
        { field: 'last_updated_on', label: 'Ultima actualizare', type: 'date' },
        { field: 'retention_until', label: 'Retenție până la', type: 'date' },
        { field: 'transfer_status', label: 'Status transfer', type: 'select', options: [{ label: 'Netransferat', value: 'none' }, { label: 'În transfer', value: 'in_transfer' }, { label: 'Transferat', value: 'transferred' }], defaultValue: 'none' },
        { field: 'authenticity_declared', label: 'Declarație autenticitate', type: 'boolean', defaultValue: false },
        { field: 'consent_captured', label: 'Consimțământ capturat', type: 'boolean', defaultValue: false },
        { field: 'custodian', label: 'Custode', type: 'text' },
        { field: 'notes', label: 'Note', type: 'textarea', wide: true },
      ],
      emptyText: 'Nu există portofolii pentru filtrele curente.',
    };
  }

  private portfolioSectionsResource(): EducationResource {
    return {
      key: 'portfolio_sections',
      label: 'Structură portofoliu',
      icon: 'pi pi-list-check',
      endpoint: '/api/education/portfolios/sections',
      createEndpoint: '',
      allowCreate: false,
      permissionHint: 'education.portfolios',
      description: 'Structura-cadru din Anexa 1: secțiuni, componente, exemple de documente, date sensibile și regulă de retenție.',
      columns: [
        { field: 'sort_order', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
        { field: 'section_code', header: 'Secțiune', sortable: true, filter: 'text', width: '12rem' },
        { field: 'component_code', header: 'Componentă', sortable: true, filter: 'text', width: '14rem' },
        { field: 'label_ro', header: 'Denumire', sortable: true, filter: 'text', width: '22rem' },
        { field: 'required', header: 'Obligatoriu', type: 'boolean', sortable: true, width: '8rem' },
        { field: 'sensitive_data', header: 'Date sensibile', type: 'boolean', sortable: true, width: '9rem' },
        { field: 'retention_rule', header: 'Retenție', sortable: true, filter: 'text', width: '18rem' },
      ],
      createFields: [],
      emptyText: 'Structura portofoliului nu este încă seed-uită.',
    };
  }

  private simplePersonnelSubResource(key: string, label: string, endpoint: string, permissionHint: string, description: string): EducationResource {
    return {
      key,
      label,
      icon: 'pi pi-list',
      endpoint,
      createEndpoint: endpoint,
      permissionHint,
      description,
      columns: [
        { field: key === 'mobility' ? 'case_code' : key === 'gradatii' ? 'grant_code' : key === 'declarations' ? 'declaration_code' : 'evaluation_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
        { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
        { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
        { field: key === 'mobility' ? 'stage' : 'status', header: key === 'mobility' ? 'Etapă' : 'Status', type: 'tag', sortable: true, filter: 'select', options: this.statusOptions(), width: '10rem' },
        { field: key === 'gradatii' ? 'score' : key === 'mobility' ? 'request_type' : key === 'declarations' ? 'declaration_type' : 'role_title', header: key === 'gradatii' ? 'Punctaj' : 'Tip / Funcție', sortable: true, filter: 'text', width: '12rem' },
        { field: key === 'mobility' ? 'submitted_on' : key === 'declarations' ? 'submitted_on' : key === 'gradatii' ? 'decision_date' : 'finalized_on', header: 'Data', sortable: true, width: '9rem' },
      ],
      createFields: [
        { field: 'employee_code', label: 'Cod angajat', type: 'text' },
        { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
        { field: 'role_title', label: 'Funcție', type: 'text' },
        { field: 'school_year', label: 'An școlar', type: 'text', defaultValue: '2025-2026' },
        { field: 'status', label: 'Status', type: 'select', options: this.statusOptions(), defaultValue: 'draft' },
        { field: key === 'gradatii' ? 'score' : 'summary', label: key === 'gradatii' ? 'Punctaj' : 'Rezumat', type: key === 'gradatii' ? 'number' : 'textarea', wide: key !== 'gradatii', defaultValue: key === 'gradatii' ? 0 : '' },
      ],
      emptyText: `Nu există înregistrări pentru ${label.toLowerCase()}.`,
    };
  }

  private statusOptions(): Array<{ label: string; value: string }> {
    return [
      { label: 'Draft', value: 'draft' },
      { label: 'Activ', value: 'active' },
      { label: 'Programat', value: 'scheduled' },
      { label: 'În lucru', value: 'in_progress' },
      { label: 'Aprobat', value: 'approved' },
      { label: 'Publicat', value: 'published' },
      { label: 'Finalizat', value: 'finalized' },
    ];
  }

  private organismOptions(): Array<{ label: string; value: string }> {
    return [
      { label: 'CA', value: 'CA' },
      { label: 'CP', value: 'CP' },
      { label: 'CEAC', value: 'CEAC' },
      { label: 'CFDCD', value: 'CFDCD' },
    ];
  }
}
