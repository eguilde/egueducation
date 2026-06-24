import { CommonModule, DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToolbarModule } from 'primeng/toolbar';
import { ToastModule } from 'primeng/toast';
import { MessageService } from 'primeng/api';
import { forkJoin, catchError, of } from 'rxjs';

import {
  CancelRegistraturaDocumentRequest,
  CreateArchiveRecordRequest,
  CreateWorkflowTaskRequest,
  LinkedDocument,
  WorkflowDefinition,
  RegistraturaDocument,
  RegistraturaDocumentAttachment,
  RegistraturaDocumentVersion,
} from '../../core/api/api.types';
import { DocumentLinksApiService } from '../../core/api/document-links-api.service';
import { EarchivaApiService } from '../../core/api/earchiva-api.service';
import { RegistraturaApiService } from '../../core/api/registratura-api.service';
import { WorkflowApiService } from '../../core/api/workflow-api.service';

@Component({
  selector: 'app-document-detail-workspace',
  standalone: true,
  imports: [
    CommonModule,
    DatePipe,
    FormsModule,
    ButtonModule,
    CardModule,
    DialogModule,
    InputTextModule,
    ProgressSpinnerModule,
    SelectModule,
    TagModule,
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
              <h1 class="m-0 text-2xl font-black tracking-[-0.03em]">
                @if (documentRecord()) {
                  {{ documentRecord()!.registry_number }} - {{ documentRecord()!.subject }}
                } @else {
                  Document
                }
              </h1>
              <p class="m-0 text-sm text-muted-color">Detalii, versiuni, atașamente și legături.</p>
            </div>
          </div>
        </ng-template>
        <ng-template pTemplate="end">
          @if (documentRecord()) {
            <div class="flex items-center gap-2">
              <p-button label="Task workflow" icon="pi pi-sitemap" severity="secondary" (onClick)="openWorkflowDialog()" />
              @if (documentRecord()!.status !== 'archived') {
                <p-button label="Arhivează" icon="pi pi-briefcase" severity="success" (onClick)="openArchiveDialog()" />
              }
              <p-button label="Editează" icon="pi pi-pencil" severity="secondary" (onClick)="openEdit()" />
              <p-button label="Anulează" icon="pi pi-ban" severity="danger" (onClick)="openCancelDialog()" />
            </div>
          }
        </ng-template>
      </p-toolbar>

      @if (loading()) {
        <div class="flex flex-1 items-center justify-center">
          <p-progressSpinner />
        </div>
      } @else if (documentRecord()) {
        <div class="grid min-h-0 flex-1 gap-4 overflow-auto lg:grid-cols-[1.4fr_0.9fr]">
          <div class="grid gap-4">
            <p-card>
              <div class="grid gap-3 sm:grid-cols-2">
                <div><span class="font-semibold">Nr. document:</span> {{ documentRecord()!.registry_number }}</div>
                <div><span class="font-semibold">Registru:</span> {{ documentRecord()!.registru_id ?? '-' }}</div>
                <div><span class="font-semibold">Tip:</span> {{ documentRecord()!.document_type }}</div>
                <div><span class="font-semibold">Direcție:</span> {{ documentRecord()!.direction }}</div>
                <div><span class="font-semibold">Status:</span> <p-tag [value]="documentRecord()!.status" [severity]="statusSeverity(documentRecord()!.status)" /></div>
                <div><span class="font-semibold">Confidențialitate:</span> {{ documentRecord()!.confidentiality }}</div>
                <div><span class="font-semibold">Emitent:</span> {{ documentRecord()!.correspondent }}</div>
                <div><span class="font-semibold">Destinatar:</span> {{ documentRecord()!.assigned_to || '-' }}</div>
                <div><span class="font-semibold">Data:</span> {{ documentRecord()!.registered_at | date:'dd.MM.yyyy HH:mm' }}</div>
                <div><span class="font-semibold">Termen:</span> {{ documentRecord()!.due_date ? (documentRecord()!.due_date | date:'dd.MM.yyyy') : '-' }}</div>
              </div>
              <div class="mt-4">
                <div class="font-semibold">Rezumat</div>
                <p class="whitespace-pre-wrap">{{ documentRecord()!.summary || '-' }}</p>
              </div>
            </p-card>

            <p-card>
              <h3 class="m-0 mb-2 text-base font-semibold">Versiuni</h3>
              <div class="grid gap-2">
                @for (version of versionItems(); track version.id) {
                  <div class="rounded-lg border border-surface-200 p-3">
                    <div class="flex items-center justify-between gap-2">
                      <strong>Versiunea {{ version.version_no }}</strong>
                      <span class="text-xs text-muted-color">{{ version.created_at | date:'dd.MM.yyyy HH:mm' }}</span>
                    </div>
                    <div class="text-sm">{{ version.change_notes }}</div>
                    <div class="text-xs text-muted-color">{{ version.created_by }}</div>
                  </div>
                } @empty {
                  <div class="text-sm text-muted-color">Nu există versiuni înregistrate.</div>
                }
              </div>
            </p-card>

            <p-card>
              <h3 class="m-0 mb-2 text-base font-semibold">Atașamente</h3>
              <div class="grid gap-2">
                @for (file of attachmentItems(); track file.id) {
                  <div class="rounded-lg border border-surface-200 p-3">
                    <div class="font-semibold">{{ file.title }}</div>
                    <div class="text-xs text-muted-color">{{ file.file_name }} • {{ file.mime_type }}</div>
                  </div>
                } @empty {
                  <div class="text-sm text-muted-color">Nu există atașamente.</div>
                }
              </div>
            </p-card>
          </div>

          <div class="grid gap-4">
            <p-card>
              <h3 class="m-0 mb-2 text-base font-semibold">Legături</h3>
              <div class="grid gap-2">
                @for (link of linkedDocuments(); track link.link_id) {
                  <div class="rounded-lg border border-surface-200 p-3">
                    <div class="font-semibold">{{ link.registry_number }}</div>
                    <div class="text-sm">{{ link.subject }}</div>
                    <div class="text-xs text-muted-color">{{ link.document_type }} • {{ link.relation_type }} • {{ link.status }}</div>
                  </div>
                } @empty {
                  <div class="text-sm text-muted-color">Nu există legături.</div>
                }
              </div>
            </p-card>
          </div>
        </div>
      }
    </section>

    <p-dialog [visible]="cancelDialogOpen" (visibleChange)="cancelDialogOpen = $event" [modal]="true" [draggable]="false" header="Anulează document" [style]="{ width: 'min(40rem, 94vw)' }">
      <div class="grid gap-3">
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Motiv</span>
          <textarea pTextarea rows="4" [(ngModel)]="cancelReason"></textarea>
        </label>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="cancelDialogOpen = false" />
          <p-button label="Anulează" severity="danger" (onClick)="cancelDocument()" />
        </div>
      </ng-template>
    </p-dialog>

    <p-dialog [visible]="workflowDialogOpen" (visibleChange)="workflowDialogOpen = $event" [modal]="true" [draggable]="false" header="Task workflow" [style]="{ width: 'min(42rem, 94vw)' }">
      <div class="grid gap-3">
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Definiție</span>
          <p-select appendTo="body" [options]="workflowDefinitionOptions" optionLabel="label" optionValue="value" [(ngModel)]="workflowDefinitionCode"></p-select>
        </label>
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Prioritate</span>
          <p-select appendTo="body" [options]="priorityOptions" optionLabel="label" optionValue="value" [(ngModel)]="workflowPriority"></p-select>
        </label>
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Sumar</span>
          <textarea pTextarea rows="4" [(ngModel)]="workflowSummary"></textarea>
        </label>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="workflowDialogOpen = false" />
          <p-button label="Creează task" icon="pi pi-check" (onClick)="createWorkflowTask()" />
        </div>
      </ng-template>
    </p-dialog>

    <p-dialog [visible]="archiveDialogOpen" (visibleChange)="archiveDialogOpen = $event" [modal]="true" [draggable]="false" header="Arhivează document" [style]="{ width: 'min(48rem, 94vw)' }">
      <div class="grid gap-3">
        <div class="grid gap-3 md:grid-cols-2">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Titlu</span>
            <input pInputText [(ngModel)]="archiveForm.title" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Referință</span>
            <input pInputText [(ngModel)]="archiveForm.source_reference" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-3">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Fond</span>
            <input pInputText [(ngModel)]="archiveForm.fond" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Serie</span>
            <input pInputText [(ngModel)]="archiveForm.series" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Status</span>
            <input pInputText [(ngModel)]="archiveForm.status" />
          </label>
        </div>
        <div class="grid gap-3 md:grid-cols-3">
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Archivist</span>
            <input pInputText [(ngModel)]="archiveForm.assigned_archivist" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Cutie</span>
            <input pInputText [(ngModel)]="archiveForm.box_number" />
          </label>
          <label class="grid gap-1">
            <span class="text-sm font-semibold">Locație</span>
            <input pInputText [(ngModel)]="archiveForm.location_code" />
          </label>
        </div>
        <label class="grid gap-1">
          <span class="text-sm font-semibold">Note</span>
          <textarea pTextarea rows="4" [(ngModel)]="archiveForm.notes"></textarea>
        </label>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="archiveDialogOpen = false" />
          <p-button label="Arhivează" icon="pi pi-briefcase" severity="success" (onClick)="archiveDocument()" />
        </div>
      </ng-template>
    </p-dialog>
  `,
})
export class DocumentDetailWorkspaceComponent implements OnInit {
  private readonly api = inject(RegistraturaApiService);
  private readonly linksApi = inject(DocumentLinksApiService);
  private readonly workflowApi = inject(WorkflowApiService);
  private readonly archiveApi = inject(EarchivaApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  protected readonly loading = signal(false);
  protected readonly documentRecord = signal<RegistraturaDocument | null>(null);
  protected readonly versionItems = signal<RegistraturaDocumentVersion[]>([]);
  protected readonly attachmentItems = signal<RegistraturaDocumentAttachment[]>([]);
  protected readonly linkedDocuments = signal<LinkedDocument[]>([]);
  protected workflowDialogOpen = false;
  protected archiveDialogOpen = false;
  protected workflowDefinitions: WorkflowDefinition[] = [];
  protected workflowDefinitionCode = '';
  protected workflowPriority = 'medium';
  protected workflowSummary = '';
  protected archiveForm: CreateArchiveRecordRequest = {
    title: '',
    fond: 'registratura',
    series: 'documente',
    source_module: 'registratura',
    source_reference: '',
    status: 'archived',
    retention_years: 5,
    assigned_archivist: '',
    box_number: '',
    location_code: '',
    archived_at: new Date().toISOString().slice(0, 10),
    notes: '',
  };
  protected cancelDialogOpen = false;
  protected cancelReason = '';
  protected workflowDefinitionOptions: Array<{ label: string; value: string }> = [];
  protected readonly priorityOptions = [
    { label: 'Mică', value: 'low' },
    { label: 'Medie', value: 'medium' },
    { label: 'Mare', value: 'high' },
    { label: 'Urgentă', value: 'urgent' },
  ];

  ngOnInit(): void {
    this.load();
  }

  protected load(): void {
    const documentId = this.route.snapshot.paramMap.get('id');
    if (!documentId) {
      void this.router.navigate(['/documente']);
      return;
    }

    this.loading.set(true);
    forkJoin({
      document: this.api.document(documentId),
      versions: this.api.documentVersions(documentId),
      attachments: this.api.documentAttachments(documentId),
      links: this.linksApi.listLinks('registratura', documentId).pipe(catchError(() => of([]))),
      definitions: this.workflowApi.definitions().pipe(catchError(() => of([]))),
    }).subscribe({
      next: ({ document, versions, attachments, links, definitions }) => {
        this.documentRecord.set(document);
        this.versionItems.set(versions);
        this.attachmentItems.set(attachments);
        this.linkedDocuments.set(links);
        this.workflowDefinitions = definitions;
        this.workflowDefinitionOptions = definitions.map((item) => ({ label: item.name, value: item.code }));
        this.workflowDefinitionCode = definitions[0]?.code ?? '';
        this.workflowSummary = document.summary || document.subject;
        this.archiveForm = {
          ...this.archiveForm,
          title: document.subject,
          source_reference: document.registry_number,
          notes: document.summary,
        };
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
      },
    });
  }

  protected goBack(): void {
    void this.router.navigate(['/documente']);
  }

  protected openEdit(): void {
    const documentId = this.documentRecord()?.id;
    if (!documentId) return;
    void this.router.navigate(['/documente', documentId, 'edit']);
  }

  protected openCancelDialog(): void {
    this.cancelReason = '';
    this.cancelDialogOpen = true;
  }

  protected openWorkflowDialog(): void {
    this.workflowDialogOpen = true;
  }

  protected openArchiveDialog(): void {
    this.archiveDialogOpen = true;
  }

  protected createWorkflowTask(): void {
    const document = this.documentRecord();
    if (!document || !this.workflowDefinitionCode) {
      return;
    }
    const payload: CreateWorkflowTaskRequest = {
      definition_code: this.workflowDefinitionCode,
      title: document.subject,
      document_number: document.registry_number,
      source_module: 'registratura',
      source_record_id: document.id,
      priority: this.workflowPriority,
      assigned_to: '',
      summary: this.workflowSummary || document.summary || document.subject,
      due_date: null,
    };
    this.workflowApi.createTask(payload).subscribe({
      next: () => this.workflowDialogOpen = false,
      error: () => this.workflowDialogOpen = false,
    });
  }

  protected archiveDocument(): void {
    const document = this.documentRecord();
    if (!document) {
      return;
    }
    const payload: CreateArchiveRecordRequest = {
      ...this.archiveForm,
      title: document.subject,
      source_reference: document.registry_number,
      notes: this.archiveForm.notes || document.summary || '',
    };
    this.archiveApi.createRecord(payload).subscribe({
      next: () => this.archiveDialogOpen = false,
      error: () => this.archiveDialogOpen = false,
    });
  }

  protected cancelDocument(): void {
    const documentId = this.documentRecord()?.id;
    if (!documentId) return;

    const payload: CancelRegistraturaDocumentRequest = {
      reason: this.cancelReason.trim() || 'Anulare document',
    };
    this.api.cancelDocument(documentId, payload).subscribe({
      next: (updated) => {
        this.documentRecord.set(updated);
        this.cancelDialogOpen = false;
        this.load();
      },
      error: () => {
        this.cancelDialogOpen = false;
      },
    });
  }

  protected statusSeverity(status: string): 'info' | 'success' | 'warn' | 'danger' | 'secondary' {
    switch (status) {
      case 'registered':
      case 'finalized':
      case 'archived':
        return 'success';
      case 'in_workflow':
        return 'warn';
      case 'draft':
        return 'secondary';
      default:
        return 'info';
    }
  }
}
