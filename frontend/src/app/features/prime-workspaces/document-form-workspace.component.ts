import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DatePickerModule } from 'primeng/datepicker';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TextareaModule } from 'primeng/textarea';
import { ToolbarModule } from 'primeng/toolbar';
import { ToastModule } from 'primeng/toast';
import { MessageService } from 'primeng/api';

import {
  CreateRegistraturaDocumentRequest,
  RegistraturaDocument,
  RegistraturaDocumentFilters,
  RegistraturaRegistry,
  UpdateRegistraturaDocumentRequest,
} from '../../core/api/api.types';
import { RegistraturaApiService } from '../../core/api/registratura-api.service';

interface DocumentFormState {
  registru_id: number | null;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  confidentiality: string;
  summary: string;
  due_date: Date | string | null;
}

const emptyDocument = (): DocumentFormState => ({
  registru_id: null,
  subject: '',
  document_type: '',
  direction: 'intrare',
  status: 'draft',
  correspondent: '',
  assigned_to: '',
  confidentiality: 'normal',
  summary: '',
  due_date: null,
});

@Component({
  selector: 'app-document-form-workspace',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    CardModule,
    DatePickerModule,
    InputTextModule,
    SelectModule,
    TextareaModule,
    ToolbarModule,
    ToastModule,
  ],
  providers: [MessageService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p-toast />
    <section class="flex h-[calc(100dvh-6rem)] min-h-0 flex-col gap-4 overflow-hidden">
      <p-toolbar styleClass="rounded-2xl border border-surface-200 bg-surface-0">
        <ng-template pTemplate="start">
          <div class="flex items-center gap-2">
            <p-button icon="pi pi-arrow-left" [text]="true" severity="secondary" (onClick)="goBack()" />
            <div class="grid">
              <h1 class="m-0 text-2xl font-black tracking-[-0.03em]">{{ pageTitle() }}</h1>
              <p class="m-0 text-sm text-muted-color">Formular registratură, aliniat cu fluxul de lucru al modulului.</p>
            </div>
          </div>
        </ng-template>
        <ng-template pTemplate="end">
          <div class="flex items-center gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="goBack()" [disabled]="saving()" />
            <p-button icon="pi pi-save" label="Salvează" (onClick)="save()" [loading]="saving()" />
          </div>
        </ng-template>
      </p-toolbar>

      <p-card class="min-h-0 flex-1 overflow-auto">
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Registru</span>
            <p-select appendTo="body" [options]="registryOptions()" [(ngModel)]="form.registru_id" optionLabel="nume" optionValue="id" [disabled]="editing()" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Tip document</span>
            <p-select appendTo="body" [options]="documentTypeOptions()" [(ngModel)]="form.document_type" [disabled]="editing()" />
          </label>
          <label class="grid gap-1 lg:col-span-2">
            <span class="text-sm font-semibold">Subiect</span>
            <textarea pTextarea rows="5" [(ngModel)]="form.subject"></textarea>
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Direcție</span>
            <p-select appendTo="body" [options]="directionOptions()" [(ngModel)]="form.direction" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Status</span>
            <p-select appendTo="body" [options]="statusOptions()" [(ngModel)]="form.status" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Confidențialitate</span>
            <p-select appendTo="body" [options]="confidentialityOptions()" [(ngModel)]="form.confidentiality" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Termen răspuns</span>
            <p-datepicker [(ngModel)]="form.due_date" dateFormat="dd.mm.yy" [showButtonBar]="true" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Emitent</span>
            <input pInputText [(ngModel)]="form.correspondent" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Destinatar</span>
            <input pInputText [(ngModel)]="form.assigned_to" />
          </label>
          <label class="grid gap-1 lg:col-span-2">
            <span class="text-sm font-semibold">Rezumat</span>
            <textarea pTextarea rows="4" [(ngModel)]="form.summary"></textarea>
          </label>
        </div>
      </p-card>
    </section>
  `,
})
export class DocumentFormWorkspaceComponent implements OnInit {
  private readonly api = inject(RegistraturaApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly msg = inject(MessageService);

  protected readonly loading = signal(false);
  protected readonly saving = signal(false);
  protected readonly editing = signal(false);
  protected readonly documentId = signal<string | null>(null);
  protected readonly registries = signal<RegistraturaRegistry[]>([]);
  protected readonly filters = signal<RegistraturaDocumentFilters | null>(null);
  protected form = emptyDocument();

  protected readonly pageTitle = signal('Document nou');

  ngOnInit(): void {
    this.documentId.set(this.route.snapshot.paramMap.get('id'));
    this.editing.set(this.route.snapshot.url.some((segment) => segment.path === 'edit'));
    this.pageTitle.set(this.editing() ? 'Editare document' : 'Document nou');
    this.loadBootstrap();
  }

  protected registryOptions() {
    return this.registries();
  }

  protected documentTypeOptions() {
    return (this.filters()?.document_types ?? []).map((value) => ({ label: value, value }));
  }

  protected directionOptions() {
    return (this.filters()?.directions ?? []).map((value) => ({ label: value, value }));
  }

  protected statusOptions() {
    return (this.filters()?.statuses ?? []).map((value) => ({ label: value, value }));
  }

  protected confidentialityOptions() {
    return (this.filters()?.confidentialities ?? []).map((value) => ({ label: value, value }));
  }

  protected goBack(): void {
    if (this.documentId()) {
      void this.router.navigate(['/documente', this.documentId()]);
      return;
    }
    void this.router.navigate(['/documente']);
  }

  protected save(): void {
    this.saving.set(true);
    const payload: CreateRegistraturaDocumentRequest = {
      ...this.form,
      registru_id: this.form.registru_id ?? this.registries()[0]?.id ?? null,
      due_date: this.normalizeDateValue(this.form.due_date),
    };

    if (this.editing() && this.documentId()) {
      const updatePayload: UpdateRegistraturaDocumentRequest = {
        ...payload,
        change_notes: 'Actualizare din formular',
      };
      this.api.updateDocument(this.documentId()!, updatePayload).subscribe({
        next: () => {
          this.saving.set(false);
          void this.router.navigate(['/documente', this.documentId()]);
        },
        error: () => this.finishSaveFailure(),
      });
      return;
    }

    this.api.createDocument(payload).subscribe({
      next: (created) => {
        this.saving.set(false);
        void this.router.navigate(['/documente', created.id]);
      },
      error: () => this.finishSaveFailure(),
    });
  }

  private loadBootstrap(): void {
    this.loading.set(true);
    const bootstrap$ = this.editing() && this.documentId()
      ? this.api.document(this.documentId()!)
      : null;

    const registries$ = this.api.registries();
    const filters$ = this.api.documentFilters();

    if (bootstrap$) {
      bootstrap$.subscribe({
        next: (document) => {
          this.form = {
            registru_id: document.registru_id ?? null,
            subject: document.subject,
            document_type: document.document_type,
            direction: document.direction,
            status: document.status,
            correspondent: document.correspondent,
            assigned_to: document.assigned_to,
            confidentiality: document.confidentiality,
            summary: document.summary,
            due_date: document.due_date ?? null,
          };
          this.afterBootstrap(registries$, filters$);
        },
        error: () => this.afterBootstrap(registries$, filters$),
      });
      return;
    }

    this.afterBootstrap(registries$, filters$);
  }

  private afterBootstrap(
    registries$: ReturnType<RegistraturaApiService['registries']>,
    filters$: ReturnType<RegistraturaApiService['documentFilters']>,
  ): void {
    registries$.subscribe({
      next: (registries) => this.registries.set(registries),
    });
    filters$.subscribe({
      next: (filters) => {
        this.filters.set(filters);
        this.loading.set(false);
        if (!this.editing()) {
          this.form.document_type = filters.document_types[0] ?? '';
          this.form.status = filters.statuses[0] ?? 'draft';
          this.form.confidentiality = filters.confidentialities[0] ?? 'normal';
        }
      },
      error: () => this.loading.set(false),
    });
  }

  private finishSaveFailure(): void {
    this.saving.set(false);
    this.msg.add({
      severity: 'error',
      summary: 'Eroare',
      detail: 'Documentul nu a putut fi salvat.',
      life: 5000,
    });
  }

  private normalizeDateValue(value: Date | string | null): string | null {
    if (!value) {
      return null;
    }
    if (value instanceof Date) {
      return this.normalizeDate(value);
    }
    return String(value);
  }

  private normalizeDate(date: Date): string {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
  }
}
