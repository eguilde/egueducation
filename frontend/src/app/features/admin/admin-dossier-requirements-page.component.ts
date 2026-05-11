import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AdminApiService } from '../../core/api/admin-api.service';
import {
  AdminDossierRequirement,
  CreateAdminDossierRequirementRequest,
} from '../../core/api/api.types';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-dossier-requirements-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
  ],
  templateUrl: './admin-dossier-requirements-page.component.html',
  styleUrl: './admin-dossier-requirements-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminDossierRequirementsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly transloco = inject(TranslocoService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);

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
    sort: 'source_module',
    direction: 'asc',
    filters: {},
    refreshToken: 0,
  });
  protected readonly selectedRequirementId = signal<string | null>(null);

  protected readonly form = this.fb.group({
    source_module: this.fb.nonNullable.control('education.governance', [Validators.required]),
    relation_type: this.fb.nonNullable.control('decision', [Validators.required]),
    min_count: this.fb.nonNullable.control(1, [Validators.required, Validators.min(1)]),
    required_for_readiness: this.fb.nonNullable.control(true),
    required_for_submit: this.fb.nonNullable.control(true),
    required_for_approve: this.fb.nonNullable.control(true),
  });

  protected readonly filters = toSignal(this.adminApi.dossierRequirementFilters(), {
    initialValue: { source_modules: [], relation_types: [] },
  });

  protected readonly response = toSignal(
    combineLatest([toObservable(this.tableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.dossierRequirements({
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
  protected readonly selectedRequirement = computed(
    () => this.rows().find((row) => row.id === this.selectedRequirementId()) ?? null,
  );

  protected readonly columns = computed<ServerTableColumn<AdminDossierRequirement>[]>(() => [
    {
      key: 'source_module',
      label: this.transloco.translate('admin.dossierRequirements.columns.sourceModule'),
      sortable: true,
      sticky: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().source_modules.map((value) => ({
          value,
          label: this.transloco.translate(`workflow.sourceModule.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`workflow.sourceModule.${row.source_module}`),
    },
    {
      key: 'relation_type',
      label: this.transloco.translate('admin.dossierRequirements.columns.relationType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: this.filters().relation_types.map((value) => ({
          value,
          label: this.transloco.translate(`links.relationType.${value}`),
        })),
      },
      formatter: (row) => this.transloco.translate(`links.relationType.${row.relation_type}`),
    },
    {
      key: 'min_count',
      label: this.transloco.translate('admin.dossierRequirements.columns.minCount'),
      sortable: true,
    },
    {
      key: 'required_for_readiness',
      label: this.transloco.translate('admin.dossierRequirements.columns.readiness'),
      formatter: (row) =>
        this.transloco.translate(`admin.dossierRequirements.boolean.${row.required_for_readiness ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.dossierRequirements.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.dossierRequirements.boolean.no') },
        ],
      },
    },
    {
      key: 'required_for_submit',
      label: this.transloco.translate('admin.dossierRequirements.columns.submit'),
      formatter: (row) =>
        this.transloco.translate(`admin.dossierRequirements.boolean.${row.required_for_submit ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.dossierRequirements.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.dossierRequirements.boolean.no') },
        ],
      },
    },
    {
      key: 'required_for_approve',
      label: this.transloco.translate('admin.dossierRequirements.columns.approve'),
      formatter: (row) =>
        this.transloco.translate(`admin.dossierRequirements.boolean.${row.required_for_approve ? 'yes' : 'no'}`),
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.dossierRequirements.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.dossierRequirements.boolean.no') },
        ],
      },
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
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected onSelectRequirement(record: AdminDossierRequirement): void {
    this.selectedRequirementId.set(record.id);
    this.form.reset({
      source_module: record.source_module,
      relation_type: record.relation_type,
      min_count: record.min_count,
      required_for_readiness: record.required_for_readiness,
      required_for_submit: record.required_for_submit,
      required_for_approve: record.required_for_approve,
    });
  }

  protected saveRequirement(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: CreateAdminDossierRequirementRequest = {
      source_module: raw.source_module,
      relation_type: raw.relation_type,
      min_count: raw.min_count,
      required_for_readiness: raw.required_for_readiness,
      required_for_submit: raw.required_for_submit,
      required_for_approve: raw.required_for_approve,
    };

    this.adminApi.saveDossierRequirement(payload).subscribe({
      next: (saved) => {
        this.selectedRequirementId.set(saved.id);
        this.snackBar.open(
          this.transloco.translate('admin.dossierRequirements.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.tableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.dossierRequirements.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
