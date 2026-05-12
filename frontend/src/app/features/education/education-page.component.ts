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
import { MatTabsModule } from '@angular/material/tabs';

import {
  CreateGovernanceMeetingRequest,
  EducationTaxonomyItem,
  GovernanceMeeting,
} from '../../core/api/api.types';
import { EducationApiService } from '../../core/api/education-api.service';
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
  selector: 'app-education-page',
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
    MatTabsModule,
    ServerTableComponent,
    LinkedDocumentsCardComponent,
  ],
  templateUrl: './education-page.component.html',
  styleUrl: './education-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationPageComponent {
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
    refreshToken: number;
  }>({
    page: 1,
    pageSize: 10,
    sort: 'meeting_date',
    direction: 'desc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedMeetingId = signal<string | null>(null);
  protected readonly activePanel = signal<'create' | 'details'>('create');

  protected readonly form = this.fb.group({
    school_year: this.fb.nonNullable.control('2025-2026', [Validators.required]),
    organism: this.fb.nonNullable.control('ca', [Validators.required]),
    title: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(5)]),
    meeting_type: this.fb.nonNullable.control('ordinary', [Validators.required]),
    status: this.fb.nonNullable.control('draft', [Validators.required]),
    quorum_required: this.fb.nonNullable.control(7, [Validators.required, Validators.min(1)]),
    participants_count: this.fb.nonNullable.control(0, [Validators.required, Validators.min(0)]),
    meeting_date: this.fb.control<Date | null>(new Date(), [Validators.required]),
    location: this.fb.nonNullable.control(''),
    chairperson: this.fb.nonNullable.control(''),
    secretary_name: this.fb.nonNullable.control(''),
    summary: this.fb.nonNullable.control(''),
  });

  protected readonly dashboard = toSignal(this.api.dashboard(), {
    initialValue: {
      stats: {
        total_meetings: 0,
        scheduled_meetings: 0,
        held_meetings: 0,
        published_meetings: 0,
      },
    },
  });

  protected readonly filters = toSignal(this.api.filters(), {
    initialValue: {
      school_years: [],
      organisms: [],
      meeting_types: [],
      statuses: [],
    },
  });
  protected readonly taxonomies = toSignal(
    this.api.taxonomies([
      'school_year',
      'governance_organism',
      'governance_meeting_type',
      'governance_status',
    ]),
    { initialValue: { items: {} } },
  );

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.api.meetings({
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
  protected readonly selectedMeeting = computed(
    () => this.rows().find((row) => row.id === this.selectedMeetingId()) ?? null,
  );
  protected readonly rowActions = computed<ServerTableRowAction<GovernanceMeeting>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly columns = computed<ServerTableColumn<GovernanceMeeting>[]>(() => [
    {
      key: 'title',
      label: this.transloco.translate('education.columns.title'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'school_year',
      label: this.transloco.translate('education.columns.schoolYear'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().school_years.map((value) => ({ value, label: value })),
      },
    },
    {
      key: 'organism',
      label: this.transloco.translate('education.columns.organism'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().organisms.map((value) => ({
          value,
          label: this.taxonomyLabel('governance_organism', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('governance_organism', row.organism),
    },
    {
      key: 'meeting_type',
      label: this.transloco.translate('education.columns.meetingType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().meeting_types.map((value) => ({
          value,
          label: this.taxonomyLabel('governance_meeting_type', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('governance_meeting_type', row.meeting_type),
    },
    {
      key: 'status',
      label: this.transloco.translate('education.columns.status'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().statuses.map((value) => ({
          value,
          label: this.taxonomyLabel('governance_status', value),
        })),
      },
      formatter: (row) => this.taxonomyLabel('governance_status', row.status),
    },
    {
      key: 'participants_count',
      label: this.transloco.translate('education.columns.participants'),
      sortable: false,
      formatter: (row) => `${row.participants_count} / ${row.quorum_required}`,
    },
    {
      key: 'meeting_date',
      label: this.transloco.translate('education.columns.meetingDate'),
      sortable: true,
      filterKey: 'meeting_on',
      filter: { type: 'date', placeholder: this.transloco.translate('table.filters.onDate') },
      formatter: (row) =>
        new Intl.DateTimeFormat(this.transloco.getActiveLang(), { dateStyle: 'medium' }).format(new Date(row.meeting_date)),
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

  protected onSelectMeeting(record: GovernanceMeeting): void {
    this.selectedMeetingId.set(record.id);
    this.activePanel.set('details');
  }

  protected onActionClick(event: { action: string; row: GovernanceMeeting }): void {
    if (event.action === 'open') {
      this.onSelectMeeting(event.row);
    }
  }

  protected openCreatePanel(): void {
    this.activePanel.set('create');
  }

  protected resetForm(): void {
    this.selectedMeetingId.set(null);
    this.activePanel.set('create');
    this.form.reset({
      school_year: '2025-2026',
      organism: 'ca',
      title: '',
      meeting_type: 'ordinary',
      status: 'draft',
      quorum_required: 7,
      participants_count: 0,
      meeting_date: new Date(),
      location: '',
      chairperson: '',
      secretary_name: '',
      summary: '',
    });
  }

  protected createMeeting(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateGovernanceMeetingRequest = {
      school_year: raw.school_year,
      organism: raw.organism,
      title: raw.title,
      meeting_type: raw.meeting_type,
      status: raw.status,
      quorum_required: raw.quorum_required,
      participants_count: raw.participants_count,
      meeting_date: raw.meeting_date ? this.formatDate(raw.meeting_date) : '',
      location: raw.location,
      chairperson: raw.chairperson,
      secretary_name: raw.secretary_name,
      summary: raw.summary,
    };

    this.api.createMeeting(payload).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('education.messages.created'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.resetForm();
        this.refreshData();
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('education.messages.createFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected startWorkflow(): void {
    const record = this.selectedMeeting();
    if (!record) {
      return;
    }

    this.workflowLauncher.launchGovernanceWorkflow(record).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('education.messages.workflowStarted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('education.messages.workflowStartFailed'),
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

  protected taxonomyOptions(domain: string, fallback: string[]): Array<{ value: string; label: string }> {
    const items = (this.taxonomies().items as Record<string, EducationTaxonomyItem[]>)[domain] ?? [];
    if (items.length > 0) {
      return items.map((item) => ({ value: item.code, label: this.taxonomyLabel(domain, item.code) }));
    }
    return fallback.map((value) => ({ value, label: this.taxonomyLabel(domain, value) }));
  }

  protected taxonomyLabel(domain: string, code: string): string {
    const item = ((this.taxonomies().items as Record<string, EducationTaxonomyItem[]>)[domain] ?? []).find(
      (entry) => entry.code === code,
    );
    if (!item) {
      return domain === 'school_year' ? code : this.transloco.translate(this.legacyTaxonomyKey(domain, code));
    }
    return this.transloco.getActiveLang() === 'en' ? item.label_en : item.label_ro;
  }

  private legacyTaxonomyKey(domain: string, code: string): string {
    switch (domain) {
      case 'governance_organism':
        return `education.organism.${code}`;
      case 'governance_meeting_type':
        return `education.meetingType.${code}`;
      case 'governance_status':
        return `education.status.${code}`;
      default:
        return code;
    }
  }
}
