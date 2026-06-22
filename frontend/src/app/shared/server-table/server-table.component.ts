import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, Input, TemplateRef, contentChild, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';

import { ButtonModule } from 'primeng/button';
import { DatePickerModule } from 'primeng/datepicker';
import { InputTextModule } from 'primeng/inputtext';
import { MenuModule } from 'primeng/menu';
import { SelectModule } from 'primeng/select';
import { TagModule } from 'primeng/tag';
import { TableLazyLoadEvent, TableModule } from 'primeng/table';
import { TooltipModule } from 'primeng/tooltip';

export interface ServerTableFilterOption {
  value: string;
  label: string;
}

export interface ServerTableFilterConfig {
  type: 'text' | 'select' | 'date';
  placeholder?: string;
  options?: ServerTableFilterOption[];
}

export interface ServerTableColumn<T> {
  key: keyof T & string;
  label: string;
  kind?: 'text' | 'tag' | 'boolean' | 'number';
  sortable?: boolean;
  sticky?: boolean;
  mobileHidden?: boolean;
  mobilePriority?: number;
  formatter?: (row: T) => string;
  tagSeverity?: (row: T) => 'success' | 'info' | 'warn' | 'danger' | 'secondary' | 'contrast';
  sortKey?: string;
  filterKey?: string;
  filter?: ServerTableFilterConfig;
  width?: string;
}

export interface ServerTableRowAction<T> {
  key: string;
  icon: string;
  label: string;
  disabled?: (row: T) => boolean;
  hidden?: (row: T) => boolean;
}

export interface PageEvent {
  pageIndex: number;
  pageSize: number;
  length: number;
}

export type SortDirection = 'asc' | 'desc' | '';

export interface Sort {
  active: string;
  direction: SortDirection;
}

export type ServerTableFilterState = Record<string, string>;
export type ServerTableSortState = Sort;

@Component({
  selector: 'app-server-table',
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    DatePickerModule,
    InputTextModule,
    MenuModule,
    SelectModule,
    TagModule,
    TableModule,
    TooltipModule,
  ],
  templateUrl: './server-table.component.html',
  styleUrl: './server-table.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ServerTableComponent {
  @Input({ required: true }) columns: ServerTableColumn<any>[] = [];
  @Input({ required: true }) rows: any[] = [];
  @Input() total = 0;
  @Input() pageIndex = 0;
  @Input() pageSize = 25;
  @Input() pageSizeOptions = [10, 25, 50, 100];
  @Input() loading = false;
  @Input() emptyMessage = 'No data';
  @Input() sortActive = '';
  @Input() sortDirection: SortDirection = '';
  @Input() rowClickable = false;
  @Input() selectedRowId?: string | null = null;
  @Input() rowIdKey = 'id';
  @Input() rowActions: ServerTableRowAction<any>[] = [];
  @Input() actionColumnLabel = 'Actions';
  @Input() filterToggleLabel = 'Filters';

  readonly pageChange = output<PageEvent>();
  readonly filterChange = output<ServerTableFilterState>();
  readonly sortChange = output<ServerTableSortState>();
  readonly rowClick = output<any>();
  readonly actionClick = output<{ action: string; row: any }>();

  protected readonly filters = signal<ServerTableFilterState>({});
  protected readonly filtersExpanded = signal(true);
  protected readonly toolbarTemplate = contentChild<TemplateRef<unknown>>('serverTableToolbar');
  protected readonly actionHeaderTemplate = contentChild<TemplateRef<unknown>>('serverTableActionHeader');

  protected hasFilterableColumns(): boolean {
    return this.columns.some((column) => !!column.filter);
  }

  protected visibleColumns(): ServerTableColumn<any>[] {
    return this.columns;
  }

  protected first(): number {
    return this.pageIndex * this.pageSize;
  }

  protected onLazyLoad(event: TableLazyLoadEvent): void {
    const nextPageSize = event.rows ?? this.pageSize;
    const nextFirst = event.first ?? this.first();
    const nextPageIndex = Math.floor(nextFirst / nextPageSize);

    this.pageChange.emit({
      pageIndex: nextPageIndex,
      pageSize: nextPageSize,
      length: this.total,
    });

    const sortField = Array.isArray(event.sortField) ? event.sortField[0] : event.sortField;
    const direction: SortDirection = event.sortOrder === 1 ? 'asc' : event.sortOrder === -1 ? 'desc' : '';
    if (sortField || direction) {
      this.sortChange.emit({ active: sortField ?? '', direction });
    }
  }

  protected onFilterChange(key: string, value: string | Date | null): void {
    const formatted = value instanceof Date ? this.formatDate(value) : String(value ?? '');
    this.filters.update((current) => ({ ...current, [key]: formatted }));
    this.filterChange.emit(this.filters());
  }

  protected cellValue(row: any, column: ServerTableColumn<any>): string {
    if (column.formatter) {
      return column.formatter(row);
    }
    return String(row[column.key] ?? '');
  }

  protected onRowClick(row: any): void {
    if (this.rowClickable) {
      this.rowClick.emit(row);
    }
  }

  protected rowIdentifier(row: any): string | null {
    const value = row?.[this.rowIdKey];
    return value == null ? null : String(value);
  }

  protected columnFilterKey(column: ServerTableColumn<any>): string {
    return column.filterKey || column.key;
  }

  protected columnSortKey(column: ServerTableColumn<any>): string {
    return column.sortKey || column.key;
  }

  protected mobileColumns(): ServerTableColumn<any>[] {
    return this.columns
      .filter((column) => !column.mobileHidden)
      .slice()
      .sort((left, right) => (left.mobilePriority ?? 999) - (right.mobilePriority ?? 999));
  }

  protected primaryMobileColumn(): ServerTableColumn<any> | null {
    return this.mobileColumns()[0] ?? null;
  }

  protected secondaryMobileColumns(): ServerTableColumn<any>[] {
    return this.mobileColumns().slice(1, 5);
  }

  protected visibleRowActions(row: any): ServerTableRowAction<any>[] {
    return this.rowActions.filter((action) => !action.hidden?.(row));
  }

  protected onActionClick(action: ServerTableRowAction<any>, row: any, event: Event): void {
    event.stopPropagation();
    if (!action.disabled?.(row)) {
      this.actionClick.emit({ action: action.key, row });
    }
  }

  protected actionIcon(icon: string): string {
    if (icon.startsWith('pi ')) {
      return icon;
    }
    const iconMap: Record<string, string> = {
      add: 'pi pi-plus',
      archive: 'pi pi-inbox',
      assignment: 'pi pi-file',
      block: 'pi pi-ban',
      check: 'pi pi-check',
      delete: 'pi pi-trash',
      edit: 'pi pi-pencil',
      fingerprint: 'pi pi-fingerprint',
      history: 'pi pi-history',
      launch: 'pi pi-external-link',
      more_vert: 'pi pi-ellipsis-v',
      open_in_new: 'pi pi-external-link',
      person_search: 'pi pi-user',
      policy: 'pi pi-shield',
      print: 'pi pi-print',
      publish: 'pi pi-send',
      schedule: 'pi pi-clock',
      settings: 'pi pi-cog',
      visibility: 'pi pi-eye',
    };
    return iconMap[icon] ?? 'pi pi-circle';
  }

  protected toggleFilters(): void {
    this.filtersExpanded.update((value) => !value);
  }

  protected columnSeverity(row: any, column: ServerTableColumn<any>): 'success' | 'info' | 'warn' | 'danger' | 'secondary' | 'contrast' {
    return column.tagSeverity?.(row) ?? 'secondary';
  }

  private formatDate(value: Date): string {
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }
}
