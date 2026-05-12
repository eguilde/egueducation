import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminGdprSetting, UpdateAdminGdprSettingRequest } from '../../core/api/api.types';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-gdpr-settings-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatButtonToggleModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatInputModule,
    MatPaginatorModule,
    MatSelectModule,
    MatSnackBarModule,
    ServerTableComponent,
  ],
  templateUrl: './admin-gdpr-settings-page.component.html',
  styleUrl: './admin-gdpr-settings-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminGdprSettingsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly fb = inject(FormBuilder);
  private readonly transloco = inject(TranslocoService);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly refreshToken = signal(0);
  protected readonly panelMode = signal<'details' | 'edit'>('details');
  protected readonly registerState = signal<{
    page: number;
    pageSize: number;
    sort?: string;
    direction?: 'asc' | 'desc';
    filters: Record<string, string>;
  }>({
    page: 1,
    pageSize: 25,
    sort: 'code',
    direction: 'asc',
    filters: {},
  });

  protected readonly response = toSignal(toObservable(this.refreshToken).pipe(switchMap(() => this.adminApi.gdprSettings())), {
    initialValue: { items: [], total: 0, page: 1, pageSize: 0 },
  });

  protected readonly items = computed(() => this.response().items);
  protected readonly filteredItems = computed(() => {
    const state = this.registerState();
    let items = [...this.items()];
    const codeFilter = (state.filters['code'] ?? '').trim().toLowerCase();
    const typeFilter = state.filters['value_type'] ?? '';
    if (codeFilter) {
      items = items.filter((item) =>
        this.transloco.translate(`admin.gdprSettings.codes.${item.code}`).toLowerCase().includes(codeFilter) ||
        item.code.toLowerCase().includes(codeFilter),
      );
    }
    if (typeFilter) {
      items = items.filter((item) => item.value_type === typeFilter);
    }
    if (state.sort) {
      items.sort((left, right) => {
        const direction = state.direction === 'desc' ? -1 : 1;
        const leftValue = this.sortValue(left, state.sort!);
        const rightValue = this.sortValue(right, state.sort!);
        return leftValue.localeCompare(rightValue, undefined, { numeric: true, sensitivity: 'base' }) * direction;
      });
    }
    return items;
  });
  protected readonly pagedItems = computed(() => {
    const state = this.registerState();
    const start = (state.page - 1) * state.pageSize;
    return this.filteredItems().slice(start, start + state.pageSize);
  });
  protected readonly selected = signal<AdminGdprSetting | null>(null);
  protected readonly codeOptions = [
    'publication_anonymization_required',
    'subject_export_requires_approval',
    'default_response_sla_days',
    'retention_review_notice_days',
    'portfolio_consent_required',
    'portfolio_authenticity_required',
  ];

  protected readonly form = this.fb.group({
    code: this.fb.nonNullable.control('publication_anonymization_required', [Validators.required]),
    value_type: this.fb.nonNullable.control<'text' | 'bool' | 'int'>('bool', [Validators.required]),
    value_text: this.fb.nonNullable.control(''),
    value_bool: this.fb.nonNullable.control(true),
    value_int: this.fb.nonNullable.control(30, [Validators.required, Validators.min(0)]),
  });

  protected readonly currentType = computed(() => this.form.controls.value_type.getRawValue());
  protected readonly columns = computed<ServerTableColumn<AdminGdprSetting>[]>(() => [
    {
      key: 'code',
      label: this.transloco.translate('admin.gdprSettings.form.code'),
      sticky: true,
      sortable: true,
      filter: {
        type: 'text',
        placeholder: this.transloco.translate('table.filters.contains'),
      },
    },
    {
      key: 'value_type',
      label: this.transloco.translate('admin.gdprSettings.form.valueType'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'bool', label: this.transloco.translate('admin.gdprSettings.valueType.bool') },
          { value: 'int', label: this.transloco.translate('admin.gdprSettings.valueType.int') },
          { value: 'text', label: this.transloco.translate('admin.gdprSettings.valueType.text') },
        ],
      },
    },
    {
      key: 'value_text',
      label: this.transloco.translate('admin.gdprSettings.list.title'),
      formatter: (row) => this.displayValue(row),
    },
  ]);
  protected readonly rowActions = computed<ServerTableRowAction<AdminGdprSetting>[]>(() => [
    {
      key: 'open',
      icon: 'open_in_new',
      label: this.transloco.translate('admin.gdprSettings.list.title'),
    },
  ]);

  protected selectSetting(item: AdminGdprSetting): void {
    this.selected.set(item);
    this.panelMode.set('details');
    this.form.reset({
      code: item.code,
      value_type: item.value_type,
      value_text: item.value_text,
      value_bool: item.value_bool,
      value_int: item.value_int,
    });
  }

  protected onActionClick(event: { action: string; row: AdminGdprSetting }): void {
    if (event.action === 'open') {
      this.selectSetting(event.row);
    }
  }

  protected onPageChange(event: PageEvent): void {
    this.registerState.update((state) => ({
      ...state,
      page: event.pageIndex + 1,
      pageSize: event.pageSize,
    }));
  }

  protected onFilterChange(filters: ServerTableFilterState): void {
    this.registerState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onSortChange(sort: ServerTableSortState): void {
    this.registerState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: sort.direction ? (sort.direction as 'asc' | 'desc') : undefined,
    }));
  }

  protected beginCreate(): void {
    this.selected.set(null);
    this.panelMode.set('edit');
    this.form.reset({
      code: this.codeOptions[0],
      value_type: 'bool',
      value_text: '',
      value_bool: true,
      value_int: 30,
    });
  }

  protected editSelected(): void {
    if (!this.selected()) {
      return;
    }
    this.panelMode.set('edit');
  }

  protected showDetails(): void {
    this.panelMode.set('details');
  }

  protected save(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpdateAdminGdprSettingRequest = {
      code: raw.code,
      value_type: raw.value_type,
      value_text: raw.value_text,
      value_bool: raw.value_bool,
      value_int: raw.value_int,
    };

    this.adminApi.saveGdprSetting(payload).subscribe({
      next: () => {
        this.refreshToken.update((value) => value + 1);
        this.selected.set({
          code: payload.code,
          value_type: payload.value_type,
          value_text: payload.value_text,
          value_bool: payload.value_bool,
          value_int: payload.value_int,
        });
        this.panelMode.set('details');
        this.snackBar.open(
          this.transloco.translate('admin.gdprSettings.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.gdprSettings.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected displayValue(item: AdminGdprSetting): string {
    switch (item.value_type) {
      case 'bool':
        return this.transloco.translate(`admin.gdprSettings.boolean.${item.value_bool ? 'yes' : 'no'}`);
      case 'int':
        return String(item.value_int ?? '');
      default:
        return item.value_text || '—';
    }
  }

  private sortValue(item: AdminGdprSetting, sortKey: string): string {
    switch (sortKey) {
      case 'value_type':
        return item.value_type;
      case 'value_text':
        return this.displayValue(item);
      case 'code':
      default:
        return this.transloco.translate(`admin.gdprSettings.codes.${item.code}`);
    }
  }
}
