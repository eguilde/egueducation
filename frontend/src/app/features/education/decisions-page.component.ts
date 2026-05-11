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

import {
  CreateGovernanceDecisionRequest,
  EducationTaxonomyItem,
  GovernanceDecision,
} from '../../core/api/api.types';
import { EducationApiService } from '../../core/api/education-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-decisions-page',
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
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './decisions-page.component.html',
  styleUrl: './decisions-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DecisionsPageComponent {
  private readonly api = inject(EducationApiService);
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
    sort: 'decision_on',
    direction: 'desc',
    filters: {},
  });
  protected readonly selectedDecisionId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    organism: this.fb.nonNullable.control('ca', [Validators.required]),
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    publication_status: this.fb.nonNullable.control('internal', [Validators.required]),
    decision_date: this.fb.control<Date | null>(new Date(), [Validators.required]),
    legal_basis: this.fb.nonNullable.control(''),
    signed_by: this.fb.nonNullable.control(''),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.decisionsDashboard(), {
    initialValue: {
      stats: {
        total_decisions: 0,
        approved_decisions: 0,
        published_decisions: 0,
        pending_publication: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.decisionFilters(), {
    initialValue: {
      school_years: [],
      organisms: [],
      statuses: [],
      publication_statuses: [],
    },
  });

  protected readonly taxonomies = toSignal(
    this.api.taxonomies([
      'school_year',
      'governance_organism',
      'governance_decision_status',
      'governance_publication_status',
    ]),
    { initialValue: { items: {} } },
  );

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.decisions({
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
  protected readonly selectedDecision = computed(
    () => this.rows().find((row) => row.id === this.selectedDecisionId()) ?? null,
  );

  protected readonly columns = computed<ServerTableColumn<GovernanceDecision>[]>(() => [
    {
      key: 'decision_code',
      label: this.transloco.translate('educationDecisions.columns.decisionCode'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'title',
      label: this.transloco.translate('educationDecisions.columns.title'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'organism',
      label: this.transloco.translate('educationDecisions.columns.organism'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().organisms.map((value) => ({ value, label: this.taxonomyLabel('governance_organism', value) })),
      },
      formatter: (row) => this.taxonomyLabel('governance_organism', row.organism),
    },
    {
      key: 'status',
      label: this.transloco.translate('educationDecisions.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({ value, label: this.taxonomyLabel('governance_decision_status', value) })),
      },
      formatter: (row) => this.taxonomyLabel('governance_decision_status', row.status),
    },
    {
      key: 'publication_status',
      label: this.transloco.translate('educationDecisions.columns.publicationStatus'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().publication_statuses.map((value) => ({ value, label: this.taxonomyLabel('governance_publication_status', value) })),
      },
      formatter: (row) => this.taxonomyLabel('governance_publication_status', row.publication_status),
    },
    {
      key: 'decision_date',
      label: this.transloco.translate('educationDecisions.columns.decisionDate'),
      sortable: true,
      filterKey: 'decision_on',
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) => new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.decision_date)),
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

  protected onSelectDecision(record: GovernanceDecision): void {
    this.selectedDecisionId.set(record.id);
  }

  protected createDecision(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateGovernanceDecisionRequest = {
      school_year: raw.school_year,
      organism: raw.organism,
      title: raw.title,
      status: raw.status,
      publication_status: raw.publication_status,
      decision_date: this.toIsoDate(raw.decision_date),
      legal_basis: raw.legal_basis,
      signed_by: raw.signed_by,
      summary: raw.summary,
    };

    this.api.createDecision(payload).subscribe({
      next: (item) => {
        this.selectedDecisionId.set(item.id);
        this.tableState.update((state) => ({ ...state, page: 1 }));
        this.snackBar.open(
          this.transloco.translate('educationDecisions.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationDecisions.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedDecision();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchDecisionWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationDecisions.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationDecisions.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected taxonomyOptions(domain: string, fallback: string[]) {
    const catalog = this.taxonomies().items as Record<string, EducationTaxonomyItem[]>;
    const items = catalog[domain] ?? [];
    if (items.length > 0) {
      return items.map((item) => ({ value: item.code, label: this.taxonomyDisplay(item) }));
    }
    return fallback.map((value) => ({ value, label: value }));
  }

  protected taxonomyLabel(domain: string, code: string): string {
    const catalog = this.taxonomies().items as Record<string, EducationTaxonomyItem[]>;
    const item = catalog[domain]?.find((entry: EducationTaxonomyItem) => entry.code === code);
    return item ? this.taxonomyDisplay(item) : code;
  }

  private taxonomyDisplay(item: EducationTaxonomyItem): string {
    return this.transloco.getActiveLang() === 'en' ? item.label_en : item.label_ro;
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
