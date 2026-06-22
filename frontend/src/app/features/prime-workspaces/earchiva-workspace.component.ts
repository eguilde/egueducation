import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
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
  ArchiveDashboardResponse,
  ArchiveRecord,
  CreateArchiveRecordRequest,
  TableQuery,
} from '../../core/api/api.types';
import { EarchivaApiService } from '../../core/api/earchiva-api.service';

const emptyArchiveRecord = (): CreateArchiveRecordRequest => ({
  title: '',
  fond: '',
  series: '',
  source_module: 'registratura',
  source_reference: '',
  status: 'archived',
  retention_years: 5,
  assigned_archivist: '',
  box_number: '',
  location_code: '',
  archived_at: new Date().toISOString().slice(0, 10),
  notes: '',
});

@Component({
  selector: 'app-earchiva-workspace',
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
            <h1 class="m-0 text-2xl font-black tracking-[-0.03em]">eArhivă</h1>
            <p class="m-0 text-sm text-muted-color">Pachete și înregistrări arhivistice.</p>
          </div>
        </ng-template>
        <ng-template pTemplate="end">
          <p-button icon="pi pi-plus" label="Înregistrare nouă" (onClick)="openCreateDialog()" />
        </ng-template>
      </p-toolbar>

      <div class="grid gap-4 md:grid-cols-4">
        <p-card>
          <div class="text-sm text-muted-color">Total</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.total_records ?? 0 }}</div>
        </p-card>
        <p-card>
          <div class="text-sm text-muted-color">Validate</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.validated_records ?? 0 }}</div>
        </p-card>
        <p-card>
          <div class="text-sm text-muted-color">Draft</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.draft_records ?? 0 }}</div>
        </p-card>
        <p-card>
          <div class="text-sm text-muted-color">Fonduri</div>
          <div class="text-2xl font-semibold">{{ dashboard()?.stats?.unique_fonds ?? 0 }}</div>
        </p-card>
      </div>

      @if (selectedRecord()) {
        <p-card>
          <div class="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
            <div><span class="font-semibold">Titlu:</span> {{ selectedRecord()!.title }}</div>
            <div><span class="font-semibold">Număr:</span> {{ selectedRecord()!.record_number }}</div>
            <div><span class="font-semibold">Fond:</span> {{ selectedRecord()!.fond }}</div>
            <div><span class="font-semibold">Serie:</span> {{ selectedRecord()!.series }}</div>
            <div><span class="font-semibold">Status:</span> {{ selectedRecord()!.status }}</div>
            <div><span class="font-semibold">Arhivat la:</span> {{ selectedRecord()!.archived_at }}</div>
            <div><span class="font-semibold">Cutie:</span> {{ selectedRecord()!.box_number }}</div>
            <div><span class="font-semibold">Locație:</span> {{ selectedRecord()!.location_code }}</div>
          </div>
          <div class="mt-3">
            <div class="font-semibold">Note</div>
            <p class="whitespace-pre-wrap">{{ selectedRecord()!.notes || '-' }}</p>
          </div>
        </p-card>
      }

      <p-card class="min-h-0 flex-1 overflow-hidden">
        <p-table [value]="records()" [loading]="loading()" [scrollable]="true" scrollHeight="flex" [paginator]="true" [rows]="20" (onLazyLoad)="onLazyLoad($event)" [lazy]="true">
          <ng-template pTemplate="header">
            <tr>
              <th>Număr</th>
              <th>Titlu</th>
              <th>Fond</th>
              <th>Serie</th>
              <th>Status</th>
              <th>Arhivat la</th>
              <th style="width: 12rem">Acțiuni</th>
            </tr>
          </ng-template>
          <ng-template pTemplate="body" let-record>
            <tr>
              <td>{{ record.record_number }}</td>
              <td>{{ record.title }}</td>
              <td>{{ record.fond }}</td>
              <td>{{ record.series }}</td>
              <td><p-tag [value]="record.status" severity="info" /></td>
              <td>{{ record.archived_at }}</td>
              <td>
                <div class="flex gap-1">
                  <p-button icon="pi pi-eye" [text]="true" size="small" severity="secondary" (onClick)="selectedRecord.set(record)" />
                  <p-button icon="pi pi-folder-open" [text]="true" size="small" severity="secondary" />
                </div>
              </td>
            </tr>
          </ng-template>
        </p-table>
      </p-card>
    </section>

    <p-dialog [visible]="dialogOpen()" (visibleChange)="dialogOpen.set($event)" [modal]="true" [draggable]="false" header="Înregistrare arhivă" [style]="{ width: 'min(56rem, 94vw)' }">
      <div class="grid gap-3">
        <div class="grid gap-3 md:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Titlu</span>
            <input pInputText [ngModel]="form.title" (ngModelChange)="form.title = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Număr sursă</span>
            <input pInputText [ngModel]="form.source_reference" (ngModelChange)="form.source_reference = $event" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-3">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Fond</span>
            <input pInputText [ngModel]="form.fond" (ngModelChange)="form.fond = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Serie</span>
            <input pInputText [ngModel]="form.series" (ngModelChange)="form.series = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Status</span>
            <p-select appendTo="body" [options]="statusOptions" optionLabel="label" optionValue="value" [ngModel]="form.status" (ngModelChange)="form.status = $event" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-3">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Archivist</span>
            <input pInputText [ngModel]="form.assigned_archivist" (ngModelChange)="form.assigned_archivist = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Cutie</span>
            <input pInputText [ngModel]="form.box_number" (ngModelChange)="form.box_number = $event" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Locație</span>
            <input pInputText [ngModel]="form.location_code" (ngModelChange)="form.location_code = $event" />
          </label>
        </div>
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Note</span>
          <textarea pTextarea rows="4" [ngModel]="form.notes" (ngModelChange)="form.notes = $event"></textarea>
        </label>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="dialogOpen.set(false)" />
          <p-button label="Salvează" icon="pi pi-check" (onClick)="saveRecord()" />
        </div>
      </ng-template>
    </p-dialog>
  `,
})
export class EarchivaWorkspaceComponent {
  private readonly api = inject(EarchivaApiService);

  protected readonly dashboard = signal<ArchiveDashboardResponse | null>(null);
  protected readonly records = signal<ArchiveRecord[]>([]);
  protected readonly loading = signal(false);
  protected readonly dialogOpen = signal(false);
  protected readonly selectedRecord = signal<ArchiveRecord | null>(null);
  protected readonly query = signal<TableQuery>({ page: 1, pageSize: 20, sort: 'archived_at', direction: 'desc', filters: {} });
  protected form = emptyArchiveRecord();

  protected readonly statusOptions = [
    { label: 'Arhivat', value: 'archived' },
    { label: 'Transferat', value: 'transferred' },
    { label: 'Eliminat', value: 'disposed' },
    { label: 'Înlocuit', value: 'superseded' },
  ];

  ngOnInit(): void {
    this.reload();
  }

  protected reload(): void {
    this.loading.set(true);
    this.api.dashboard().subscribe({ next: (value) => this.dashboard.set(value) });
    this.api.records(this.query()).subscribe({
      next: (response) => {
        this.records.set(response.items);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  protected onLazyLoad(event: TableLazyLoadEvent): void {
    const pageSize = event.rows ?? this.query().pageSize;
    const page = Math.floor((event.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event.sortField) ? String(event.sortField[0] ?? 'archived_at') : String(event.sortField ?? 'archived_at');
    const direction = event.sortOrder === 1 ? 'asc' : 'desc';
    this.query.set({ page, pageSize, sort, direction, filters: {} });
    this.reload();
  }

  protected openCreateDialog(): void {
    this.form = emptyArchiveRecord();
    this.dialogOpen.set(true);
  }

  protected saveRecord(): void {
    this.api.createRecord(this.form).subscribe({
      next: () => {
        this.dialogOpen.set(false);
        this.reload();
      },
      error: () => this.dialogOpen.set(false),
    });
  }
}
