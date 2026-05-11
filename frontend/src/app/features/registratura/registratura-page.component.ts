import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest, of } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { PageEvent } from '@angular/material/paginator';

import {
  CreateRegistraturaDocumentAttachmentRequest,
  CreateRegistraturaDocumentRequest,
  CreateRegistraturaDocumentVersionRequest,
  RegistraturaDocument,
} from '../../core/api/api.types';
import { RegistraturaApiService } from '../../core/api/registratura-api.service';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-registratura-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
  ],
  templateUrl: './registratura-page.component.html',
  styleUrl: './registratura-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RegistraturaPageComponent {
  private readonly api = inject(RegistraturaApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

  private readonly detailRefresh = signal(0);

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
    sort: 'registered_at',
    direction: 'desc',
    filters: {},
    refreshToken: 0,
  });

  protected readonly selectedDocumentId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    subject: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    document_type: this.fb.nonNullable.control('', [Validators.required]),
    direction: this.fb.nonNullable.control('intrare', [Validators.required]),
    status: this.fb.nonNullable.control('registered', [Validators.required]),
    correspondent: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(3)]),
    assigned_to: this.fb.nonNullable.control(''),
    confidentiality: this.fb.nonNullable.control('normal', [Validators.required]),
    due_date: this.fb.control<Date | null>(null),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly versionForm = this.fb.group({
    subject: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    status: this.fb.nonNullable.control('registered', [Validators.required]),
    assigned_to: this.fb.nonNullable.control(''),
    confidentiality: this.fb.nonNullable.control('normal', [Validators.required]),
    due_date: this.fb.control<Date | null>(null),
    summary: this.fb.nonNullable.control(''),
    change_notes: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
  });

  protected readonly attachmentForm = this.fb.group({
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(3)]),
    file_name: this.fb.nonNullable.control('', [Validators.required]),
    mime_type: this.fb.nonNullable.control('application/pdf', [Validators.required]),
    storage_key: this.fb.nonNullable.control('', [Validators.required]),
    size_bytes: this.fb.nonNullable.control(0, [Validators.required, Validators.min(0)]),
    category: this.fb.nonNullable.control('supporting', [Validators.required]),
    status: this.fb.nonNullable.control('active', [Validators.required]),
    uploaded_by: this.fb.nonNullable.control(''),
  });

  protected readonly attachmentCategories = [
    'incoming_scan',
    'supporting',
    'signed_decision',
    'archive_copy',
    'export_package',
  ];

  protected readonly attachmentStatuses = ['active', 'superseded', 'archived'];

  protected readonly filters = toSignal(this.api.documentFilters(), {
    initialValue: {
      document_types: [],
      directions: [],
      statuses: [],
      confidentialities: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.documents({
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

  protected readonly selectedDocument = toSignal(
    combineLatest([toObservable(this.selectedDocumentId), toObservable(this.detailRefresh)]).pipe(
      switchMap(([documentId]) => (documentId ? this.api.document(documentId) : of(null))),
    ),
    { initialValue: null },
  );

  protected readonly versions = toSignal(
    combineLatest([toObservable(this.selectedDocumentId), toObservable(this.detailRefresh)]).pipe(
      switchMap(([documentId]) => (documentId ? this.api.documentVersions(documentId) : of([]))),
    ),
    { initialValue: [] },
  );

  protected readonly attachments = toSignal(
    combineLatest([toObservable(this.selectedDocumentId), toObservable(this.detailRefresh)]).pipe(
      switchMap(([documentId]) => (documentId ? this.api.documentAttachments(documentId) : of([]))),
    ),
    { initialValue: [] },
  );

  protected readonly columns = computed<ServerTableColumn<RegistraturaDocument>[]>(() => [
    {
      key: 'registry_number',
      label: this.transloco.translate('registratura.columns.registryNumber'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'subject',
      label: this.transloco.translate('registratura.columns.subject'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'document_type',
      label: this.transloco.translate('registratura.columns.documentType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().document_types.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'direction',
      label: this.transloco.translate('registratura.columns.direction'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().directions.map((value) => ({
          value,
          label: this.transloco.translate(`registratura.direction.${value}`),
        })),
      },
    },
    {
      key: 'status',
      label: this.transloco.translate('registratura.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`registratura.status.${value}`),
        })),
      },
    },
    {
      key: 'correspondent',
      label: this.transloco.translate('registratura.columns.correspondent'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'registered_at',
      label: this.transloco.translate('registratura.columns.registeredAt'),
      sortable: true,
      filterKey: 'registered_on',
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), {
          dateStyle: 'medium',
          timeStyle: 'short',
        }).format(new Date(row.registered_at)),
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
    },
    {
      key: 'due_date',
      label: this.transloco.translate('registratura.columns.dueDate'),
      sortable: true,
      formatter: (row) =>
        row.due_date
          ? new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(
              new Date(row.due_date),
            )
          : '—',
    },
  ]);

  protected readonly rows = computed(() => this.response().items);

  constructor() {
    effect(() => {
      const document = this.selectedDocument();
      if (!document) {
        this.versionForm.reset({
          subject: '',
          status: 'registered',
          assigned_to: '',
          confidentiality: 'normal',
          due_date: null,
          summary: '',
          change_notes: '',
        });
        return;
      }

      this.versionForm.reset({
        subject: document.subject,
        status: document.status,
        assigned_to: document.assigned_to,
        confidentiality: document.confidentiality,
        due_date: document.due_date ? new Date(document.due_date) : null,
        summary: document.summary,
        change_notes: '',
      });
    });
  }

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
      direction: sort.direction || undefined,
    }));
  }

  protected onSelectDocument(document: RegistraturaDocument): void {
    this.selectedDocumentId.set(document.id);
    this.detailRefresh.update((value) => value + 1);
  }

  protected createDocument(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateRegistraturaDocumentRequest = {
      subject: raw.subject,
      document_type: raw.document_type,
      direction: raw.direction,
      status: raw.status,
      correspondent: raw.correspondent,
      assigned_to: raw.assigned_to,
      confidentiality: raw.confidentiality,
      summary: raw.summary,
      due_date: raw.due_date ? this.formatDate(raw.due_date) : null,
    };

    this.api.createDocument(payload).subscribe({
      next: (document) => {
        this.snackBar.open(
          this.transloco.translate('registratura.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.form.reset({
          subject: '',
          document_type: '',
          direction: 'intrare',
          status: 'registered',
          correspondent: '',
          assigned_to: '',
          confidentiality: 'normal',
          due_date: null,
          summary: '',
        });
        this.selectedDocumentId.set(document.id);
        this.bumpRegistraturaRefresh();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('registratura.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected createVersion(): void {
    const documentId = this.selectedDocumentId();
    if (!documentId) {
      return;
    }
    if (this.versionForm.invalid) {
      this.versionForm.markAllAsTouched();
      return;
    }

    const raw = this.versionForm.getRawValue();
    const payload: CreateRegistraturaDocumentVersionRequest = {
      subject: raw.subject,
      status: raw.status,
      assigned_to: raw.assigned_to,
      confidentiality: raw.confidentiality,
      summary: raw.summary,
      due_date: raw.due_date ? this.formatDate(raw.due_date) : null,
      change_notes: raw.change_notes,
    };

    this.api.createDocumentVersion(documentId, payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('registratura.messages.versionCreated'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.versionForm.patchValue({ change_notes: '' });
        this.bumpRegistraturaRefresh();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('registratura.messages.versionCreateFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected createAttachment(): void {
    const documentId = this.selectedDocumentId();
    if (!documentId) {
      return;
    }
    if (this.attachmentForm.invalid) {
      this.attachmentForm.markAllAsTouched();
      return;
    }

    const raw = this.attachmentForm.getRawValue();
    const payload: CreateRegistraturaDocumentAttachmentRequest = {
      title: raw.title,
      file_name: raw.file_name,
      mime_type: raw.mime_type,
      storage_key: raw.storage_key,
      size_bytes: raw.size_bytes,
      category: raw.category,
      status: raw.status,
      uploaded_by: raw.uploaded_by,
    };

    this.api.createDocumentAttachment(documentId, payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('registratura.messages.attachmentCreated'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.attachmentForm.reset({
          title: '',
          file_name: '',
          mime_type: 'application/pdf',
          storage_key: '',
          size_bytes: 0,
          category: 'supporting',
          status: 'active',
          uploaded_by: '',
        });
        this.detailRefresh.update((value) => value + 1);
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('registratura.messages.attachmentCreateFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected formatBytes(value: number): string {
    if (value < 1024) {
      return `${value} B`;
    }
    if (value < 1024 * 1024) {
      return `${(value / 1024).toFixed(1)} KB`;
    }
    return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  }

  private bumpRegistraturaRefresh(): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      refreshToken: state.refreshToken + 1,
    }));
    this.detailRefresh.update((value) => value + 1);
  }

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }
}
