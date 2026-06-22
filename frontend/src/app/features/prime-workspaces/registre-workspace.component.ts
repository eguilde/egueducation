import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToolbarModule } from 'primeng/toolbar';

import {
  RegistraturaRegistry,
  CreateRegistraturaRegistryRequest,
  UpdateRegistraturaRegistryRequest,
} from '../../core/api/api.types';
import { RegistraturaApiService } from '../../core/api/registratura-api.service';

const emptyRegistry = (): CreateRegistraturaRegistryRequest => ({
  nume: '',
  prefix_nr: '',
  nr_inceput: 1,
  nr_curent: '',
  nr_urmator: '',
  data_resetare: null,
  tip_registru: 'GENERAL',
  isDefault: false,
});

@Component({
  selector: 'app-registre-workspace',
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
            <h1 class="m-0 text-2xl font-black tracking-[-0.03em]">Registre</h1>
            <p class="m-0 text-sm text-muted-color">Administrare registre, prefixe și registrul implicit.</p>
          </div>
        </ng-template>
        <ng-template pTemplate="end">
          <p-button icon="pi pi-plus" label="Registru nou" (onClick)="openCreateDialog()" />
        </ng-template>
      </p-toolbar>

      <p-card class="min-h-0 flex-1 overflow-hidden">
        <p-table [value]="registries()" [loading]="loading()" [scrollable]="true" scrollHeight="flex" [paginator]="true" [rows]="20" class="h-full">
          <ng-template pTemplate="header">
            <tr>
              <th>Nume</th>
              <th>Prefix</th>
              <th>Tip</th>
              <th>Nr. început</th>
              <th>Curent</th>
              <th>Următor</th>
              <th>Implicit</th>
              <th style="width: 12rem">Acțiuni</th>
            </tr>
          </ng-template>
          <ng-template pTemplate="body" let-registry>
            <tr>
              <td>{{ registry.nume }}</td>
              <td>{{ registry.prefix_nr }}</td>
              <td>{{ registry.tip_registru }}</td>
              <td>{{ registry.nr_inceput }}</td>
              <td>{{ registry.nr_curent }}</td>
              <td>{{ registry.nr_urmator }}</td>
              <td><p-tag [value]="registry.isDefault ? 'Da' : 'Nu'" [severity]="registry.isDefault ? 'success' : 'secondary'" /></td>
              <td>
                <div class="flex flex-wrap gap-1">
                  <p-button icon="pi pi-pencil" [text]="true" severity="secondary" size="small" (onClick)="openEditDialog(registry)" />
                  <p-button icon="pi pi-star" [text]="true" severity="help" size="small" (onClick)="setDefault(registry)" />
                  <p-button icon="pi pi-trash" [text]="true" severity="danger" size="small" (onClick)="deleteRegistry(registry)" />
                </div>
              </td>
            </tr>
          </ng-template>
          <ng-template pTemplate="emptymessage">
            <tr><td colspan="8" class="p-8 text-center text-muted-color">Nu există registre.</td></tr>
          </ng-template>
        </p-table>
      </p-card>
    </section>

    <p-dialog [visible]="dialogOpen()" (visibleChange)="dialogOpen.set($event)" [modal]="true" [draggable]="false" header="Registru" [style]="{ width: 'min(48rem, 94vw)' }">
      <div class="grid gap-3">
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Nume</span>
          <input pInputText [(ngModel)]="form.nume" />
        </label>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Prefix</span>
            <input pInputText [(ngModel)]="form.prefix_nr" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Tip registru</span>
            <p-select appendTo="body" [options]="tipOptions" [(ngModel)]="form.tip_registru" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-3">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Nr. început</span>
            <input pInputText type="number" [(ngModel)]="form.nr_inceput" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Nr. curent</span>
            <input pInputText [(ngModel)]="form.nr_curent" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Nr. următor</span>
            <input pInputText [(ngModel)]="form.nr_urmator" />
          </label>
        </div>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="dialogOpen.set(false)" />
          <p-button label="Salvează" icon="pi pi-check" (onClick)="saveRegistry()" />
        </div>
      </ng-template>
    </p-dialog>
  `,
})
export class RegistreWorkspaceComponent {
  private readonly api = inject(RegistraturaApiService);

  protected readonly registries = signal<RegistraturaRegistry[]>([]);
  protected readonly loading = signal(false);
  protected readonly dialogOpen = signal(false);
  protected readonly editingId = signal<number | null>(null);
  protected form = emptyRegistry();

  protected readonly tipOptions = [
    { label: 'General', value: 'GENERAL' },
    { label: 'Intrări', value: 'INTRARI' },
    { label: 'Ieșiri', value: 'IESIRI' },
    { label: 'Intern', value: 'INTERN' },
    { label: 'Petiții', value: 'PETITII' },
    { label: 'Contracte', value: 'CONTRACTE' },
    { label: 'Decizii', value: 'DECIZII' },
    { label: 'Hotărâri', value: 'HOTARARI' },
    { label: 'Dispoziții', value: 'DISPOZITII' },
  ];

  ngOnInit(): void {
    this.load();
  }

  protected load(): void {
    this.loading.set(true);
    this.api.registries().subscribe({
      next: (registries) => {
        this.registries.set(registries);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  protected openCreateDialog(): void {
    this.editingId.set(null);
    this.form = emptyRegistry();
    this.dialogOpen.set(true);
  }

  protected openEditDialog(registry: RegistraturaRegistry): void {
    this.editingId.set(registry.id);
    this.form = {
      nume: registry.nume,
      prefix_nr: registry.prefix_nr,
      nr_inceput: registry.nr_inceput,
      nr_curent: registry.nr_curent,
      nr_urmator: registry.nr_urmator,
      data_resetare: registry.data_resetare ?? null,
      tip_registru: registry.tip_registru,
      isDefault: registry.isDefault,
    };
    this.dialogOpen.set(true);
  }

  protected saveRegistry(): void {
    if (!this.form.nume.trim() || !this.form.prefix_nr.trim()) {
      return;
    }
    const payload: CreateRegistraturaRegistryRequest | UpdateRegistraturaRegistryRequest = {
      ...this.form,
      nume: this.form.nume.trim(),
      prefix_nr: this.form.prefix_nr.trim(),
      nr_curent: this.form.nr_curent?.trim() || undefined,
      nr_urmator: this.form.nr_urmator?.trim() || undefined,
    };

    const request$ = this.editingId() === null
      ? this.api.createRegistry(payload as CreateRegistraturaRegistryRequest)
      : this.api.updateRegistry(this.editingId()!, payload as UpdateRegistraturaRegistryRequest);

    request$.subscribe({
      next: () => {
        this.dialogOpen.set(false);
        this.load();
      },
      error: () => this.dialogOpen.set(false),
    });
  }

  protected deleteRegistry(registry: RegistraturaRegistry): void {
    if (!confirm(`Ștergi registrul ${registry.nume}?`)) {
      return;
    }
    this.api.deleteRegistry(registry.id).subscribe({ next: () => this.load() });
  }

  protected setDefault(registry: RegistraturaRegistry): void {
    this.api.setDefaultRegistry(registry.id).subscribe({ next: () => this.load() });
  }
}
