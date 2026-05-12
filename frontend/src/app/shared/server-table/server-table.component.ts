import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, Input, contentChild, output, signal, TemplateRef } from '@angular/core';

import { MatButtonModule } from '@angular/material/button';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatFormFieldAppearance, MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatMenuModule } from '@angular/material/menu';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSortModule, Sort, SortDirection } from '@angular/material/sort';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';

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
  sortable?: boolean;
  sticky?: boolean;
  mobileHidden?: boolean;
  mobilePriority?: number;
  formatter?: (row: T) => string;
  sortKey?: string;
  filterKey?: string;
  filter?: ServerTableFilterConfig;
}

export interface ServerTableRowAction<T> {
  key: string;
  icon: string;
  label: string;
  disabled?: (row: T) => boolean;
  hidden?: (row: T) => boolean;
}

export type ServerTableFilterState = Record<string, string>;
export type ServerTableSortState = Sort;

@Component({
  selector: 'app-server-table',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatDatepickerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatMenuModule,
    MatPaginatorModule,
    MatSelectModule,
    MatSortModule,
    MatTableModule,
    MatTooltipModule,
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
  protected readonly filterFieldAppearance: MatFormFieldAppearance = 'outline';
  protected readonly toolbarTemplate = contentChild<TemplateRef<unknown>>('serverTableToolbar');

  protected hasFilterableColumns(): boolean {
    return this.columns.some((column) => !!column.filter);
  }

  protected trackColumns(): string[] {
    const columns = this.columns.map((column) => column.key);
    if (this.rowActions.length > 0) {
      columns.push('__actions');
    }
    return columns;
  }

  protected trackFilterColumns(): string[] {
    const columns = this.columns.map((column) => `${column.key}-filter`);
    if (this.rowActions.length > 0) {
      columns.push('__actions-filter');
    }
    return columns;
  }

  protected onFilterChange(key: string, value: string): void {
    this.filters.update((current) => ({ ...current, [key]: value }));
    this.filterChange.emit(this.filters());
  }

  protected onSortChange(sort: Sort): void {
    this.sortChange.emit(sort);
  }

  protected onDateFilterChange(key: string, value: Date | null): void {
    const formatted =
      value == null
        ? ''
        : `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
    this.onFilterChange(key, formatted);
  }

  protected cellValue(row: any, column: ServerTableColumn<any>): string {
    if (column.formatter) {
      return column.formatter(row);
    }
    return String(row[column.key] ?? '');
  }

  protected onRowClick(row: any): void {
    if (!this.rowClickable) {
      return;
    }
    this.rowClick.emit(row);
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
    if (action.disabled?.(row)) {
      return;
    }
    this.actionClick.emit({ action: action.key, row });
  }

  protected toggleFilters(): void {
    this.filtersExpanded.update((value) => !value);
  }
}
