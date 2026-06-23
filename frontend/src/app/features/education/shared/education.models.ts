import { ServerTableColumn } from '../../../shared/server-table/server-table.component';

export type EducationFieldType = 'text' | 'textarea' | 'number' | 'date' | 'select' | 'boolean';
export type EducationColumnType = 'text' | 'tag' | 'date' | 'boolean' | 'number';

export interface EducationOption {
  label: string;
  value: string | boolean;
}

export interface EducationFieldOptionsContext {
  resource: EducationResourceConfig;
  child?: EducationDetailChildResourceConfig;
  parentRow?: EducationRow;
  childRows: (child: EducationDetailChildResourceConfig, parentRow: EducationRow) => EducationRow[];
}

export interface EducationColumn {
  field: string;
  header: string;
  type?: EducationColumnType;
  sortable?: boolean;
  filter?: 'text' | 'select';
  options?: EducationOption[];
  width?: string;
}

export interface EducationFormField {
  field: string;
  label: string;
  type: EducationFieldType;
  required?: boolean;
  options?: EducationOption[] | ((context: EducationFieldOptionsContext) => EducationOption[]);
  defaultValue?: string | number | boolean;
  min?: number;
  max?: number;
  step?: number;
  wide?: boolean;
}

export interface EducationDetailChildResourceConfig {
  key: string;
  label: string;
  icon: string;
  description: string;
  listEndpoint: (parentRow: EducationRow) => string;
  detailEndpoint?: (parentRow: EducationRow, childRow: EducationRow) => string;
  createEndpoint: (parentRow: EducationRow) => string;
  allowCreate?: boolean;
  readPermission: string;
  managePermission?: string;
  columns: EducationColumn[];
  createFields: EducationFormField[];
  emptyText: string;
  pdfEndpoint?: (parentRow: EducationRow, childRow: EducationRow) => string;
  pdfFilename?: (parentRow: EducationRow, childRow: EducationRow) => string;
  pdfActionLabel?: string;
}

export interface EducationResourceConfig {
  key: string;
  label: string;
  icon: string;
  description: string;
  endpoint: string;
  createEndpoint: string;
  allowCreate?: boolean;
  readPermission: string;
  managePermission?: string;
  columns: EducationColumn[];
  createFields: EducationFormField[];
  emptyText: string;
  detailChildren?: EducationDetailChildResourceConfig[];
  pdfEndpoint?: (row: EducationRow) => string;
  pdfFilename?: (row: EducationRow) => string;
  pdfActionLabel?: string;
}

export interface EducationSectionConfig {
  key: string;
  label: string;
  icon: string;
  description: string;
  resources: EducationResourceConfig[];
}

export interface EducationRouteTab {
  path: string;
  label: string;
  icon: string;
  permissions: string[];
}

export interface EducationDashboardCardConfig {
  key: string;
  label: string;
  icon: string;
  endpoint: string;
  statKey: string;
  permission: string;
}

export type EducationCreateForm = Record<string, string | number | boolean | Date | null>;
export type EducationRow = Record<string, unknown>;
export type EducationServerTableColumn = ServerTableColumn<EducationRow>;
