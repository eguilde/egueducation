import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { CreateGdprPublicationReviewRequest, GdprPublicationReview } from '../../core/api/api.types';
import { GdprApiService } from '../../core/api/gdpr-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-gdpr-publication-page',
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
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './gdpr-publication-page.component.html',
  styleUrl: './gdpr-publication-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GdprPublicationPageComponent {
  private readonly api = inject(GdprApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);
  private readonly workflowLauncher = inject(WorkflowLauncherService);

  protected readonly tableState = signal<{
    page: number;
    pageSize: number;
    sort?: string;
    direction?: 'asc' | 'desc';
    filters: Record<string, string>;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'reviewed_on',
    direction: 'desc',
    filters: {},
  });
  protected readonly selectedRecordId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    source_module: this.fb.nonNullable.control('education.decisions', [Validators.required]),
    source_record_id: this.fb.nonNullable.control('', [Validators.required]),
    source_label: this.fb.nonNullable.control('', [Validators.required]),
    anonymization_status: this.fb.nonNullable.control('pending', [Validators.required]),
    publication_status: this.fb.nonNullable.control('blocked', [Validators.required]),
    reviewed_by: this.fb.nonNullable.control(''),
    reviewed_on: this.fb.control<Date | null>(null),
    legal_basis: this.fb.nonNullable.control('', [Validators.required]),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.publicationDashboard(), {
    initialValue: {
      stats: {
        total_reviews: 0,
        pending_anonymization: 0,
        ready_for_publication: 0,
        published_items: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.publicationReviewFilters(), {
    initialValue: {
      source_modules: [],
      anonymization_statuses: [],
      publication_statuses: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.publicationReviews({
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
  protected readonly selectedRecord = computed(
    () => this.rows().find((row) => row.id === this.selectedRecordId()) ?? null,
  );

  protected readonly columns = computed<ServerTableColumn<GdprPublicationReview>[]>(() => [
    {
      key: 'source_label',
      label: this.transloco.translate('gdprPublication.columns.sourceLabel'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'review_code',
      label: this.transloco.translate('gdprPublication.columns.reviewCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'source_module',
      label: this.transloco.translate('gdprPublication.columns.sourceModule'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().source_modules.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.publicationSourceModule.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.publicationSourceModule.${row.source_module}`),
    },
    {
      key: 'anonymization_status',
      label: this.transloco.translate('gdprPublication.columns.anonymizationStatus'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().anonymization_statuses.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.anonymizationStatus.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.anonymizationStatus.${row.anonymization_status}`),
    },
    {
      key: 'publication_status',
      label: this.transloco.translate('gdprPublication.columns.publicationStatus'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().publication_statuses.map((value) => ({
          value,
          label: this.transloco.translate(`gdpr.publicationStatus.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`gdpr.publicationStatus.${row.publication_status}`),
    },
    {
      key: 'reviewed_on',
      label: this.transloco.translate('gdprPublication.columns.reviewedOn'),
      sortable: true,
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) => row.reviewed_on || '—',
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
      direction: (sort.direction as 'asc' | 'desc' | '') || undefined,
    }));
  }

  protected onSelectRecord(record: GdprPublicationReview): void {
    this.selectedRecordId.set(record.id);
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateGdprPublicationReviewRequest = {
      source_module: raw.source_module,
      source_record_id: raw.source_record_id,
      source_label: raw.source_label,
      anonymization_status: raw.anonymization_status,
      publication_status: raw.publication_status,
      reviewed_by: raw.reviewed_by,
      reviewed_on: this.toIsoDate(raw.reviewed_on),
      legal_basis: raw.legal_basis,
      notes: raw.notes,
    };

    this.api.createPublicationReview(payload).subscribe({
      next: (item) => {
        this.selectedRecordId.set(item.id);
        this.snackBar.open(
          this.transloco.translate('gdprPublication.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('gdprPublication.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedRecord();
    if (!record) {
      return;
    }
    this.workflowLauncher.launchGdprPublicationWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('gdprPublication.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('gdprPublication.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  private toIsoDate(value: Date | string | null): string {
    if (!value) {
      return '';
    }
    if (typeof value === 'string') {
      return value;
    }
    return value.toISOString().slice(0, 10);
  }
}
