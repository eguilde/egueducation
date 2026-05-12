import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTabsModule } from '@angular/material/tabs';

import { CreatePortfolioRecordRequest, PortfolioRecord } from '../../core/api/api.types';
import { EducationPortfolioApiService } from '../../core/api/education-portfolio-api.service';
import { WorkflowLauncherService } from '../../core/api/workflow-launcher.service';
import { LinkedDocumentsCardComponent } from '../../shared/linked-documents-card/linked-documents-card.component';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-portfolio-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    MatTabsModule,
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './portfolio-page.component.html',
  styleUrl: './portfolio-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PortfolioPageComponent {
  private readonly api = inject(EducationPortfolioApiService);
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
    refreshToken: number;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'last_updated_on',
    direction: 'desc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedPortfolioId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    owner_name: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    owner_role: this.fb.nonNullable.control('', [Validators.required]),
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    section_count: this.fb.nonNullable.control(0, [Validators.required, Validators.min(0)]),
    last_updated_on: this.fb.control<Date | null>(new Date(), [Validators.required]),
    retention_until: this.fb.control<Date | null>(new Date(new Date().setFullYear(new Date().getFullYear() + 3)), [Validators.required]),
    transfer_status: this.fb.nonNullable.control('none', [Validators.required]),
    authenticity_declared: this.fb.nonNullable.control(false),
    consent_captured: this.fb.nonNullable.control(false),
    custodian: this.fb.nonNullable.control(''),
    notes: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        total_portfolios: 0,
        validated_portfolios: 0,
        transfer_portfolios: 0,
        declared_portfolios: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.filters(), {
    initialValue: {
      school_years: [],
      statuses: [],
      transfer_statuses: [],
    },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.records({
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

  protected readonly rows = computed(() => this.response().items);
  protected readonly selectedPortfolio = computed(
    () => this.rows().find((row) => row.id === this.selectedPortfolioId()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<PortfolioRecord>[]>(() => [
    {
      key: 'open',
      icon: 'open_in_new',
      label: this.transloco.translate('common.open'),
    },
  ]);

  protected readonly columns = computed<ServerTableColumn<PortfolioRecord>[]>(() => [
    {
      key: 'owner_name',
      label: this.transloco.translate('educationPortfolios.columns.ownerName'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'portfolio_code',
      label: this.transloco.translate('educationPortfolios.columns.portfolioCode'),
      sortable: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'school_year',
      label: this.transloco.translate('educationPortfolios.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'status',
      label: this.transloco.translate('educationPortfolios.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.transloco.translate(`educationPortfolios.status.${value}`),
        })),
      },
    },
    {
      key: 'section_count',
      label: this.transloco.translate('educationPortfolios.columns.sectionCount'),
      sortable: true,
    },
    {
      key: 'transfer_status',
      label: this.transloco.translate('educationPortfolios.columns.transferStatus'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().transfer_statuses.map((value) => ({
          value,
          label: this.transloco.translate(`educationPortfolios.transferStatus.${value}`),
        })),
      },
    },
    {
      key: 'retention_until',
      label: this.transloco.translate('educationPortfolios.columns.retentionUntil'),
      sortable: true,
      filterKey: 'retention_on',
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.retention_until)),
    },
    {
      key: 'authenticity_declared',
      label: this.transloco.translate('educationPortfolios.columns.authenticity'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('educationPortfolios.boolean.yes') },
          { value: 'false', label: this.transloco.translate('educationPortfolios.boolean.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`educationPortfolios.boolean.${row.authenticity_declared ? 'yes' : 'no'}`),
    },
  ]);

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
      direction: (sort.direction as 'asc' | 'desc' | '') || undefined,
    }));
  }

  protected onSelectPortfolio(record: PortfolioRecord): void {
    this.selectedPortfolioId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: PortfolioRecord }): void {
    if (event.action === 'open') {
      this.onSelectPortfolio(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected createRecord(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreatePortfolioRecordRequest = {
      owner_name: raw.owner_name,
      owner_role: raw.owner_role,
      school_year: raw.school_year,
      status: raw.status,
      section_count: raw.section_count,
      last_updated_on: raw.last_updated_on ? this.formatDate(raw.last_updated_on) : '',
      retention_until: raw.retention_until ? this.formatDate(raw.retention_until) : '',
      transfer_status: raw.transfer_status,
      authenticity_declared: raw.authenticity_declared,
      consent_captured: raw.consent_captured,
      custodian: raw.custodian,
      notes: raw.notes,
    };

    this.api.createRecord(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationPortfolios.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        const retentionDate = new Date();
        retentionDate.setFullYear(retentionDate.getFullYear() + 3);
        this.form.reset({
          owner_name: '',
          owner_role: '',
          school_year: '2025-2026',
          status: 'draft',
          section_count: 0,
          last_updated_on: new Date(),
          retention_until: retentionDate,
          transfer_status: 'none',
          authenticity_declared: false,
          consent_captured: false,
          custodian: '',
          notes: '',
        });
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationPortfolios.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedPortfolio();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchPortfolioWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('educationPortfolios.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('educationPortfolios.messages.workflowStartFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  private refreshData(): void {
    this.tableState.update((state) => ({
      ...state,
      page: 1,
      refreshToken: state.refreshToken + 1,
    }));
  }

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }
}
