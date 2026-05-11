import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, Input, output, signal } from '@angular/core';

import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldAppearance, MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatSelectModule } from '@angular/material/select';
import { MatSortModule, Sort, SortDirection } from '@angular/material/sort';
import { MatTableModule } from '@angular/material/table';

export interface ServerTableFilterOption {
  value: string;
  label: string;
}

export interface ServerTableFilterConfig {
  type: 'text' | 'select';
  placeholder?: string;
  options?: ServerTableFilterOption[];
}

export interface ServerTableColumn<T> {
  key: keyof T & string;
  label: string;
  sortable?: boolean;
  sticky?: boolean;
  formatter?: (row: T) => string;
  filter?: ServerTableFilterConfig;
}

export type ServerTableFilterState = Record<string, string>;
export type ServerTableSortState = Sort;

@Component({
  selector: 'app-server-table',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatPaginatorModule,
    MatSelectModule,
    MatSortModule,
    MatTableModule,
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

  readonly pageChange = output<PageEvent>();
  readonly filterChange = output<ServerTableFilterState>();
  readonly sortChange = output<ServerTableSortState>();

  protected readonly filters = signal<ServerTableFilterState>({});
  protected readonly filterFieldAppearance: MatFormFieldAppearance = 'outline';

  protected trackColumns(): string[] {
    return this.columns.map((column) => column.key);
  }

  protected trackFilterColumns(): string[] {
    return this.columns.map((column) => `${column.key}-filter`);
  }

  protected onFilterChange(key: string, value: string): void {
    this.filters.update((current) => ({ ...current, [key]: value }));
    this.filterChange.emit(this.filters());
  }

  protected onSortChange(sort: Sort): void {
    this.sortChange.emit(sort);
  }

  protected cellValue(row: any, column: ServerTableColumn<any>): string {
    if (column.formatter) {
      return column.formatter(row);
    }
    return String(row[column.key] ?? '');
  }
}
