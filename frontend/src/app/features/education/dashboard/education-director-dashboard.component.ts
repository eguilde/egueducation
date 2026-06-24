import { CommonModule } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';
import { AuditTrailPanelComponent } from '../shared/wizard/audit-trail-panel.component';
import { DeadlineBadgeComponent } from '../shared/wizard/deadline-badge.component';
import { DocumentPreviewPanelComponent } from '../shared/wizard/document-preview-panel.component';
import { WizardSummaryPanelComponent } from '../shared/wizard/wizard-summary-panel.component';
import { EDUCATION_SECTIONS } from '../shared/education-config';

interface DirectorCockpitResponse {
  school_year: string;
  institution_id: string;
  governance: {
    total_meetings: number;
    scheduled_meetings: number;
    meetings_without_minute: number;
    meetings_without_vote: number;
    published_resolutions: number;
  };
  portfolios: {
    total_records: number;
    draft_records: number;
    review_records: number;
    returned_records: number;
    validated_records: number;
    transfer_in_progress: number;
  };
  evaluations: {
    total_records: number;
    submitted_records: number;
    reviewed_records: number;
    contested_records: number;
    approved_records: number;
    communicated_documents: number;
  };
  managerial: {
    total_dossiers: number;
    draft_dossiers: number;
    review_dossiers: number;
    approved_dossiers: number;
    published_documents: number;
    workflow_open_steps: number;
  };
  personnel: {
    total_records: number;
    active_records: number;
    portfolio_enabled: number;
    evaluation_pending: number;
    mobility_cases: number;
  };
  compliance: {
    total_requirements: number;
    implemented_requirements: number;
    partial_requirements: number;
    pending_publications: number;
    anonymization_pending: number;
  };
  alerts: Array<{
    id: string;
    title: string;
    summary: string;
    status: string;
    route: string;
    priority: number;
  }>;
  recommended_links: Array<{
    key: string;
    label: string;
    route: string;
  }>;
}

interface DirectorMetricCard {
  key: string;
  label: string;
  icon: string;
  value: number | string;
  route: string;
  queryParams?: Record<string, string>;
}

interface DirectorAlertItem {
  id: string;
  title: string;
  summary: string;
  status: string;
  statusLabel: string;
  route: string;
  queryParams?: Record<string, string>;
}

interface DirectorReportItem {
  key: string;
  title: string;
  summary: string;
  route: string;
  queryParams?: Record<string, string>;
}

@Component({
  selector: 'app-education-director-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ButtonModule,
    CardModule,
    TagModule,
    AuditTrailPanelComponent,
    DeadlineBadgeComponent,
    DocumentPreviewPanelComponent,
    WizardSummaryPanelComponent,
  ],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-briefcase"></i>
              Management educational
            </div>
            <h1 class="m-0 text-3xl font-semibold">Cockpit director</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Vedere operationala pentru guvernanta, portofolii, evaluari, documente manageriale si conformitate.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <p-tag value="Director" severity="contrast" />
            <p-button
              [routerLink]="['/education', 'dashboard', 'director', 'reports']"
              label="Rapoarte standard"
              icon="pi pi-chart-line"
              severity="secondary"
              [outlined]="true"
            />
            <p-button
              [routerLink]="['/education', 'dashboard']"
              label="Dashboard general"
              icon="pi pi-arrow-left"
              severity="secondary"
              [outlined]="true"
            />
          </div>
        </div>
      </div>

      <app-wizard-summary-panel
        title="Rezumat institutional"
        description="Sinteza rapida a ariilor educationale deja instrumentate in platforma."
        [items]="summaryItems()"
      />

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-8">
        @for (card of cards(); track card.key) {
          <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm">
            <div class="space-y-4">
              <div class="flex items-center justify-between gap-3">
                <div class="space-y-1">
                  <p class="m-0 text-sm text-muted-color">{{ card.label }}</p>
                  <strong class="block text-3xl tracking-tight">{{ card.value }}</strong>
                </div>
                <span class="grid size-12 place-items-center rounded-2xl border border-surface-200 bg-surface-50">
                  <i [class]="card.icon"></i>
                </span>
              </div>
              <p-button
                [routerLink]="card.route"
                [queryParams]="card.queryParams"
                label="Deschide"
                icon="pi pi-arrow-right"
                size="small"
                severity="secondary"
                [outlined]="true"
              />
            </div>
          </p-card>
        }
      </div>

      <div class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div class="space-y-1">
              <h2 class="m-0 text-xl font-semibold">Alerte operationale</h2>
              <p class="m-0 text-sm text-muted-color">Semnale rapide pe baza indicatorilor existenti in sistem.</p>
            </div>
            <p-tag [value]="alerts().length + ' alerte'" severity="secondary" />
          </div>

          <div class="mt-4 grid gap-3">
            @for (alert of alerts(); track alert.id) {
              <article class="rounded-xl border border-surface p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="space-y-2">
                    <div class="font-medium">{{ alert.title }}</div>
                    <p class="m-0 text-sm text-muted-color">{{ alert.summary }}</p>
                  </div>
                  <app-deadline-badge [label]="alert.statusLabel" [status]="alert.status" />
                </div>
                <div class="mt-3">
                  <p-button
                    [routerLink]="alert.route"
                    [queryParams]="alert.queryParams"
                    label="Vezi lista"
                    icon="pi pi-arrow-right"
                    size="small"
                  />
                </div>
              </article>
            } @empty {
              <article class="rounded-xl border border-dashed border-surface p-4 text-sm text-muted-color">
                Nu exista alerte prioritare pe baza indicatorilor disponibili in acest moment.
              </article>
            }
          </div>
        </section>

        <div class="grid gap-4">
          <app-document-preview-panel
            title="Arii active"
            description="Zonele educationale deja pregatite pentru operare si rafinare procedurala."
            [documents]="sectionPreviewItems()"
          />

          <app-audit-trail-panel
            title="Trasee recomandate"
            description="Secventa recomandata de lucru pentru director in etapa actuala."
            [events]="recommendedFlow()"
          />
        </div>
      </div>

      <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
        <div class="space-y-1">
          <h2 class="m-0 text-xl font-semibold">Rapoarte standard</h2>
          <p class="m-0 text-sm text-muted-color">Setul initial de rapoarte operative pe care directorul il poate deschide rapid din cockpit.</p>
        </div>

        <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          @for (report of reports(); track report.key) {
            <article class="rounded-xl border border-surface p-4">
              <div class="space-y-3">
                <div>
                  <div class="font-medium">{{ report.title }}</div>
                  <p class="m-0 mt-1 text-sm text-muted-color">{{ report.summary }}</p>
                </div>
                <p-button
                  [routerLink]="report.route"
                  [queryParams]="report.queryParams"
                  label="Deschide raport"
                  icon="pi pi-arrow-right"
                  size="small"
                  severity="secondary"
                  [outlined]="true"
                />
              </div>
            </article>
          }
        </div>
      </section>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationDirectorDashboardComponent {
  private readonly authz = inject(AuthzService);
  private readonly cockpitResource = httpResource<DirectorCockpitResponse>(() => ({
    url: '/api/education/director/cockpit',
    method: 'GET',
  }));

  protected readonly institutionName = this.authz.institutionName;
  protected readonly cards = computed<DirectorMetricCard[]>(() =>
    this.cockpitValue()
      ? [
          {
            key: 'meetings',
            label: 'Sedinte guvernanta',
            icon: 'pi pi-calendar',
            value: this.cockpitValue()!.governance.total_meetings,
            route: '/education/governance',
            queryParams: { resource: 'governance_meetings', filter_status: 'scheduled', presetLabel: 'Sedinte planificate' },
          },
          {
            key: 'portfolios',
            label: 'Portofolii validate',
            icon: 'pi pi-folder-open',
            value: this.cockpitValue()!.portfolios.validated_records,
            route: '/education/portfolio',
            queryParams: { resource: 'portfolios', filter_status: 'validated', presetLabel: 'Portofolii validate' },
          },
          {
            key: 'evaluations',
            label: 'Evaluari contestate',
            icon: 'pi pi-megaphone',
            value: this.cockpitValue()!.evaluations.contested_records,
            route: '/education/personnel',
            queryParams: { resource: 'evaluations', filter_status: 'contested', presetLabel: 'Evaluari contestate' },
          },
          {
            key: 'managerial',
            label: 'Dosare manageriale',
            icon: 'pi pi-briefcase',
            value: this.cockpitValue()!.managerial.total_dossiers,
            route: '/education/governance',
            queryParams: { resource: 'managerial', filter_status: 'review', presetLabel: 'Documente manageriale in lucru' },
          },
          {
            key: 'personnel',
            label: 'Personal activ',
            icon: 'pi pi-id-card',
            value: this.cockpitValue()!.personnel.active_records,
            route: '/education/personnel',
            queryParams: { resource: 'personnel', filter_status: 'active', presetLabel: 'Cadre didactice active' },
          },
          {
            key: 'compliance',
            label: 'Publicari restante',
            icon: 'pi pi-shield',
            value: this.cockpitValue()!.compliance.pending_publications,
            route: '/education/compliance',
          },
        ]
      : [],
  );
  protected readonly schoolYear = computed(() => this.cockpitValue()?.school_year || '-');
  protected readonly cockpitValue = computed(() => this.cockpitResource.value() ?? null);

  protected readonly summaryItems = computed(() => {
    const availableSections = EDUCATION_SECTIONS.filter((section) =>
      section.resources.some((resource) => this.authz.hasPermission(resource.readPermission)),
    );
    const cockpit = this.cockpitValue();

    return [
      {
        key: 'sections',
        label: 'Domenii active',
        value: String(availableSections.length),
      },
      {
        key: 'resources',
        label: 'Registre disponibile',
        value: String(availableSections.reduce((total, section) => total + section.resources.length, 0)),
      },
      {
        key: 'school-year',
        label: 'An scolar activ',
        value: cockpit?.school_year || '-',
      },
      {
        key: 'alerts',
        label: 'Alerte active',
        value: String(cockpit?.alerts.length ?? 0),
      },
      {
        key: 'focus',
        label: 'Prioritate curenta',
        value: this.primaryFocus(cockpit),
      },
    ];
  });

  protected readonly alerts = computed<DirectorAlertItem[]>(() =>
    (this.cockpitValue()?.alerts ?? [])
      .slice()
      .sort((left, right) => left.priority - right.priority)
      .map((alert) => ({
        id: alert.id,
        title: alert.title,
        summary: alert.summary,
        status: alert.status,
        statusLabel: this.statusLabel(alert.status),
        route: alert.route,
        queryParams: this.alertQueryParams(alert.id),
      })),
  );

  protected readonly sectionPreviewItems = computed(() =>
    (this.cockpitValue()?.recommended_links ?? []).map((link) => ({
      id: link.key,
      title: link.label,
      summary: `Traseu recomandat pentru director prin ${link.label.toLowerCase()}.`,
      status: 'active',
    })),
  );

  protected readonly recommendedFlow = computed(() => [
    {
      id: 'flow-1',
      title: 'Revizuire guvernanta',
      summary: 'Verifica sedintele, hotararile si componenta organismelor inainte de a deschide fluxurile asistate CA/CP.',
      actorName: 'Director',
    },
    {
      id: 'flow-2',
      title: 'Verificare portofolii si evaluari',
      summary: 'Coreleaza portofoliile validate, evaluarile si documentele care trebuie sa ajunga in dosarul personal.',
      actorName: 'Director',
    },
    {
      id: 'flow-3',
      title: 'Pregatire documente manageriale',
      summary: 'Foloseste dosarele manageriale si regulamentele existente drept baza pentru viitoarele wizard-uri PDI/PAS si raport de calitate.',
      actorName: 'Director',
    },
  ]);

  protected readonly reports = computed<DirectorReportItem[]>(() => [
    {
      key: 'governance-register',
      title: 'Registru sedinte si hotarari',
      summary: 'Acces rapid la sedinte, minute, voturi si hotarari pentru urmarirea guvernantei.',
      route: '/education/governance',
      queryParams: { resource: 'governance_meetings', filter_status: 'scheduled', presetLabel: 'Registru sedinte planificate' },
    },
    {
      key: 'portfolio-status',
      title: 'Situatie portofolii',
      summary: 'Lista operationala pentru portofolii validate, returnate si aflate in transfer.',
      route: '/education/portfolio',
      queryParams: { resource: 'portfolios', filter_status: 'returned', presetLabel: 'Portofolii returnate' },
    },
    {
      key: 'evaluation-status',
      title: 'Stadiu evaluari',
      summary: 'Monitorizare evaluari, contestatii si rezultate comunicate sau restante.',
      route: '/education/personnel',
      queryParams: { resource: 'evaluations', filter_status: 'contested', presetLabel: 'Evaluari contestate' },
    },
    {
      key: 'managerial-publications',
      title: 'Documente manageriale',
      summary: 'Acces la dosare manageriale si la documentele institutionale care cer publicare sau aprobare.',
      route: '/education/governance',
      queryParams: { resource: 'managerial', filter_status: 'review', presetLabel: 'Documente manageriale in review' },
    },
  ]);

  private primaryFocus(cockpit: DirectorCockpitResponse | null): string {
    if (!cockpit) {
      return 'Se incarca';
    }
    if (cockpit.evaluations.contested_records > 0) {
      return 'Evaluari contestate';
    }
    if (cockpit.portfolios.returned_records > 0) {
      return 'Portofolii returnate';
    }
    if (cockpit.compliance.pending_publications > 0) {
      return 'Publicari restante';
    }
    if (cockpit.governance.meetings_without_minute > 0) {
      return 'Sedinte fara minute';
    }
    return 'Guvernanta si portofolii';
  }

  private statusLabel(status: string): string {
    switch (status) {
      case 'contested':
        return 'Contestat';
      case 'pending':
        return 'Restant';
      case 'in_progress':
        return 'In lucru';
      case 'validated':
        return 'Validat';
      case 'scheduled':
        return 'Planificat';
      default:
        return 'Monitorizare';
    }
  }

  private alertQueryParams(alertID: string): Record<string, string> | undefined {
    switch (alertID) {
      case 'evaluations-contested':
        return { resource: 'evaluations', filter_status: 'contested', presetLabel: 'Evaluari contestate' };
      case 'publications-pending':
        return { resource: 'managerial', filter_status: 'review', presetLabel: 'Publicari restante' };
      default:
        return undefined;
    }
  }
}
