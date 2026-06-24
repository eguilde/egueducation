import { CommonModule } from '@angular/common';
import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';
import { TableQuery } from '../../../core/api/api.types';
import { AuditTrailPanelComponent } from '../shared/wizard/audit-trail-panel.component';
import { EDUCATION_SECTIONS } from '../shared/education-config';
import type { EducationColumn } from '../shared/education.models';
import { DocumentPreviewPanelComponent } from '../shared/wizard/document-preview-panel.component';

interface StandardReportCard {
  key: string;
  title: string;
  summary: string;
  route: string;
  audience: string;
  resourceKey: string;
  filtersLabel: string[];
  queryParams?: Record<string, string>;
}

interface EducationExportPayload {
  title: string;
  filename: string;
  headers: string[];
  rows: string[][];
}

interface EducationRow {
  [key: string]: unknown;
}

interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

@Component({
  selector: 'app-education-standard-reports',
  standalone: true,
  imports: [CommonModule, RouterLink, ButtonModule, CardModule, TagModule, AuditTrailPanelComponent, DocumentPreviewPanelComponent],
  providers: [MessageService],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-chart-line"></i>
              Rapoarte operationale
            </div>
            <h1 class="m-0 text-3xl font-semibold">Rapoarte standard</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Intrari standard pentru registre, monitorizare si liste operative deja disponibile in platforma.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <p-button [routerLink]="['/education', 'dashboard', 'director']" label="Inapoi la cockpit" icon="pi pi-arrow-left" severity="secondary" [outlined]="true" />
          </div>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        @for (report of reports(); track report.key) {
          <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm">
            <div class="space-y-4">
              <div class="space-y-2">
                <div class="flex items-center justify-between gap-3">
                  <h2 class="m-0 text-lg font-semibold">{{ report.title }}</h2>
                  <p-tag [value]="report.audience" severity="secondary" />
                </div>
                <p class="m-0 text-sm text-muted-color">{{ report.summary }}</p>
                <div class="flex flex-wrap gap-2">
                  @for (filterLabel of report.filtersLabel; track filterLabel) {
                    <p-tag [value]="filterLabel" severity="contrast" />
                  }
                </div>
              </div>

              <div class="flex flex-wrap gap-2">
                <p-button [routerLink]="report.route" [queryParams]="report.queryParams" label="Deschide lista" icon="pi pi-arrow-right" size="small" />
                <p-button
                  label="CSV"
                  icon="pi pi-file-excel"
                  size="small"
                  severity="secondary"
                  [outlined]="true"
                  [loading]="exportingKey() === report.key + ':csv'"
                  (onClick)="exportReport(report, 'csv')"
                />
                <p-button
                  label="PDF"
                  icon="pi pi-file-pdf"
                  size="small"
                  severity="secondary"
                  [outlined]="true"
                  [loading]="exportingKey() === report.key + ':pdf'"
                  (onClick)="exportReport(report, 'pdf')"
                />
              </div>
            </div>
          </p-card>
        }
      </div>

      <div class="grid gap-4 xl:grid-cols-[1fr_1fr]">
        <app-document-preview-panel
          title="Pachet minim"
          description="Setul actual de rapoarte standard foloseste registrele deja implementate si le transforma in puncte de acces manageriale."
          [documents]="previewItems()"
        />

        <app-audit-trail-panel
          title="Evolutie recomandata"
          description="Ordinea buna pentru cresterea zonei de raportare in sprinturile urmatoare."
          [events]="nextSteps()"
        />
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationStandardReportsComponent {
  private readonly authz = inject(AuthzService);
  private readonly http = inject(HttpClient);
  private readonly messages = inject(MessageService);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly exportingKey = signal('');
  private readonly resourcesByKey = new Map(
    EDUCATION_SECTIONS.flatMap((section) => section.resources.map((resource) => [resource.key, resource] as const)),
  );

  protected readonly reports = computed<StandardReportCard[]>(() => [
    {
      key: 'ca',
      title: 'CA complet',
      summary: 'Monitorizare sedinte, minute, voturi si hotarari pentru Consiliul de Administratie.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'meetings',
      filtersLabel: ['organism: CA'],
      queryParams: { resource: 'meetings', filter_organism: 'ca', presetLabel: 'Sedinte CA' } as Record<string, string>,
    },
    {
      key: 'cp',
      title: 'CP complet',
      summary: 'Sedinte, documente oficiale si trasee procedurale pentru Consiliul Profesoral.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'cp_meetings',
      filtersLabel: ['organism: CP'],
      queryParams: { resource: 'cp_meetings', presetLabel: 'Sedinte CP' } as Record<string, string>,
    },
    {
      key: 'ca-docs',
      title: 'CA - documente si minute',
      summary: 'Acces la sedinte CA cu focus pe documentele oficiale, minute si hotarari publicabile.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'meetings',
      filtersLabel: ['CA', 'documente'],
      queryParams: { resource: 'meetings', filter_organism: 'ca', filter_status: 'held', presetLabel: 'Documente CA' } as Record<string, string>,
    },
    {
      key: 'ca-attendance',
      title: 'CA - participanti si semnaturi',
      summary: 'Acces la sedinte CA cu accent pe prezenta, semnaturi si cvorum procedural.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'meetings',
      filtersLabel: ['CA', 'semnaturi'],
      queryParams: { resource: 'meetings', filter_organism: 'ca', presetLabel: 'Participanti CA' } as Record<string, string>,
    },
    {
      key: 'cp-docs',
      title: 'CP - documente si minute',
      summary: 'Acces la sedinte CP cu focus pe documentele oficiale, minute si actele rezultate.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'cp_meetings',
      filtersLabel: ['CP', 'documente'],
      queryParams: { resource: 'cp_meetings', filter_status: 'held', presetLabel: 'Documente CP' } as Record<string, string>,
    },
    {
      key: 'managerial',
      title: 'Documente manageriale in lucru',
      summary: 'Acces rapid la dosare manageriale, publicari restante si documente aflate in aprobare.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'managerial',
      filtersLabel: ['status: in review'],
      queryParams: { resource: 'managerial', filter_status: 'in_review', presetLabel: 'Documente manageriale in lucru' } as Record<string, string>,
    },
    {
      key: 'director-portfolio',
      title: 'Portofoliu director',
      summary: 'Acces separat la verticala directorului, cu flux ghidat si exporturi dedicate.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'director_portfolio',
      filtersLabel: ['dossier: director'],
      queryParams: { resource: 'director_portfolio', presetLabel: 'Portofoliu director' } as Record<string, string>,
    },
    {
      key: 'adjunct-portfolio',
      title: 'Portofoliu director adjunct',
      summary: 'Acces separat la verticala directorului adjunct, cu flux ghidat si exporturi dedicate.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'adjunct_director_portfolio',
      filtersLabel: ['dossier: adjunct'],
      queryParams: { resource: 'adjunct_director_portfolio', presetLabel: 'Portofoliu director adjunct' } as Record<string, string>,
    },
    {
      key: 'portfolios',
      title: 'Situatie portofolii',
      summary: 'Validari, portofolii returnate, transferuri si urmarirea custodiei.',
      route: '/education/portfolio',
      audience: 'Director / Secretariat',
      resourceKey: 'portfolios',
      filtersLabel: ['status: returnat'],
      queryParams: { resource: 'portfolios', filter_status: 'returned', presetLabel: 'Portofolii returnate' } as Record<string, string>,
    },
    {
      key: 'personnel',
      title: 'Cadre didactice si conformitate',
      summary: 'Fise de personal, declaratii si stare operationala pe anul scolar curent.',
      route: '/education/personnel',
      audience: 'Director / Secretariat',
      resourceKey: 'personnel',
      filtersLabel: ['status: activ'],
      queryParams: { resource: 'personnel', filter_status: 'active', presetLabel: 'Cadre didactice active' } as Record<string, string>,
    },
    {
      key: 'evaluations',
      title: 'Stadiu evaluari si contestatii',
      summary: 'Urmarire evaluari anuale, contestatii si rezultate aflate in lucru sau finalizate.',
      route: '/education/personnel',
      audience: 'Director',
      resourceKey: 'evaluations',
      filtersLabel: ['status: contestat'],
      queryParams: { resource: 'evaluations', filter_status: 'contested', presetLabel: 'Evaluari contestate' } as Record<string, string>,
    },
    {
      key: 'mobility',
      title: 'Mobilitate si cazuri deschise',
      summary: 'Vizibilitate asupra transferurilor, pretransferurilor si altor cazuri de mobilitate.',
      route: '/education/personnel',
      audience: 'Director / Inspector',
      resourceKey: 'mobility',
      filtersLabel: ['status: deschis'],
      queryParams: { resource: 'mobility', filter_status: 'open', presetLabel: 'Cazuri de mobilitate deschise' } as Record<string, string>,
    },
    {
      key: 'committees',
      title: 'Comisii si evaluare',
      summary: 'Comisii, componenta, mandat, stare si comisia de evaluare cu vizibilitate procedurala.',
      route: '/education/governance',
      audience: 'Director / Secretariat',
      resourceKey: 'committees',
      filtersLabel: ['comisii active'],
      queryParams: { resource: 'committees', filter_status: 'active', presetLabel: 'Comisii active' } as Record<string, string>,
    },
    {
      key: 'regulations',
      title: 'ROF / ROI',
      summary: 'Regulamente, versiuni si trasee de avizare / aprobare pentru documentele institutionale.',
      route: '/education/governance',
      audience: 'Director',
      resourceKey: 'regulations',
      filtersLabel: ['regulament'],
      queryParams: { resource: 'regulations', presetLabel: 'Regulamente institutionale' } as Record<string, string>,
    },
  ]);

  protected readonly previewItems = computed(() => [
    {
      id: 'standard-reports',
      title: 'Rapoarte standard generatie 1',
      summary: 'Liste actionabile bazate pe registrele educationale deja disponibile in platforma.',
      status: 'active',
    },
    {
      id: 'filters',
      title: 'Filtre dedicate',
      summary: 'Urmatoarea iteratie va adauga presetari si filtre salvabile pe fiecare raport important.',
      status: 'pending',
    },
    {
      id: 'exports',
      title: 'Exporturi PDF / CSV',
      summary: 'Zona de raportare poate fi extinsa cu exporturi formale si distribuire controlata.',
      status: 'scheduled',
    },
  ]);

  protected readonly nextSteps = computed(() => [
    {
      id: 'step-1',
      title: 'Presetari pe filtre',
      summary: 'Fiecare raport standard trebuie sa deschida o lista deja filtrata pe scenariul managerial relevant.',
      actorName: 'Produs / Frontend',
    },
    {
      id: 'step-2',
      title: 'Exporturi standard',
      summary: 'Listele cele mai importante pot primi export CSV sau PDF pentru utilizare institutionala.',
      actorName: 'Backend / Frontend',
    },
    {
      id: 'step-3',
      title: 'Audit si acces sensibil',
      summary: 'Rapoartele cu date sensibile vor avea reguli de acces si urmarire mai stricte.',
      actorName: 'RBAC / Audit',
    },
  ]);

  protected exportReport(report: StandardReportCard, format: 'csv' | 'pdf'): void {
    const resource = this.resourcesByKey.get(report.resourceKey);
    if (!resource) {
      this.messages.add({
        severity: 'warn',
        summary: 'Raport indisponibil',
        detail: 'Configuratia resursei nu a fost gasita pentru acest raport.',
      });
      return;
    }

    const exportQuery: TableQuery = {
      page: 1,
      pageSize: 200,
      sort: resource.columns[0]?.field,
      direction: 'asc',
      filters: this.filtersFromQueryParams(report.queryParams),
    };

    this.exportingKey.set(`${report.key}:${format}`);
    this.http.get<PagedResponse<EducationRow>>(resource.endpoint, { params: this.toParams(exportQuery) }).subscribe({
      next: (response) => {
        const items = response.items ?? [];
        if (!items.length) {
          this.exportingKey.set('');
          this.messages.add({
            severity: 'warn',
            summary: 'Nu exista date pentru export',
            detail: 'Nu exista inregistrari pentru scenariul selectat.',
          });
          return;
        }

        const payload = this.buildExportPayload(report, resource.columns, items);
        this.http.post(`/api/education/exports/${format}`, payload, { responseType: 'blob' as const }).subscribe({
          next: (blob) => {
            this.downloadBlob(blob, `${payload.filename}.${format}`);
            this.exportingKey.set('');
          },
          error: () => {
            this.exportingKey.set('');
            this.messages.add({
              severity: 'error',
              summary: 'Exportul a esuat',
              detail: 'Fisierul nu a putut fi generat pentru raportul selectat.',
            });
          },
        });
      },
      error: () => {
        this.exportingKey.set('');
        this.messages.add({
          severity: 'error',
          summary: 'Exportul a esuat',
          detail: 'Datele raportului nu au putut fi incarcate.',
        });
      },
    });
  }

  private filtersFromQueryParams(queryParams?: Record<string, string>): Record<string, string> {
    if (!queryParams) {
      return {};
    }
    return Object.fromEntries(
      Object.entries(queryParams)
        .filter(([key, value]) => key.startsWith('filter_') && String(value).trim() !== '')
        .map(([key, value]) => [key.replace('filter_', ''), value]),
    );
  }

  private toParams(query: TableQuery): HttpParams {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      params = params.set(`filter.${key}`, String(value));
    }
    return params;
  }

  private buildExportPayload(report: StandardReportCard, columns: EducationColumn[], rows: EducationRow[]): EducationExportPayload {
    const presetLabel = report.queryParams?.['presetLabel'] ?? report.title;
    return {
      title: `${report.title} - ${presetLabel}`,
      filename: `${report.resourceKey}-${this.slugify(presetLabel)}`,
      headers: columns.map((column) => column.header),
      rows: rows.map((row) => columns.map((column) => this.formatConfiguredValue(column, row[column.field]))),
    };
  }

  private formatConfiguredValue(column: EducationColumn, value: unknown): string {
    if (column.options?.length) {
      const match = column.options.find((option: { label: string; value: string | boolean }) => option.value === value);
      if (match) {
        return match.label;
      }
    }
    if (column.type === 'boolean') {
      return value ? 'Da' : 'Nu';
    }
    if (value == null || value === '') {
      return '-';
    }
    return String(value);
  }

  private slugify(value: string): string {
    return value
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 80);
  }

  private downloadBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
    URL.revokeObjectURL(url);
  }
}
