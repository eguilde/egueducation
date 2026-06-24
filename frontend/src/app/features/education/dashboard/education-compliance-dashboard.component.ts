import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';
import { EDUCATION_COMPLIANCE_SECTION, EDUCATION_GOVERNANCE_SECTION } from '../shared/education-config';

interface ComplianceMetricCard {
  key: string;
  label: string;
  icon: string;
  value: number | string;
}

interface ComplianceQuickLink {
  key: string;
  title: string;
  summary: string;
  route: string;
  queryParams?: Record<string, string>;
}

interface CompliancePlaybookItem {
  id: string;
  title: string;
  summary: string;
  actorName: string;
}

@Component({
  selector: 'app-education-compliance-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ButtonModule,
    CardModule,
    TagModule,
  ],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-shield"></i>
              Conformitate educationala
            </div>
            <h1 class="m-0 text-3xl font-semibold">Cockpit conformitate</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Spatiu de lucru pentru publicari, cerinte, dovezi, anonimizare si urmarirea documentelor institutionale
              care trebuie validate sau arhivate.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            @for (role of roleTags(); track role) {
              <p-tag [value]="role" severity="contrast" />
            }
            <p-tag [value]="institutionName()" severity="secondary" />
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

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        @for (card of metrics(); track card.key) {
          <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm">
            <div class="flex items-center justify-between gap-3">
              <div class="space-y-1">
                <p class="m-0 text-sm text-muted-color">{{ card.label }}</p>
                <strong class="block text-3xl tracking-tight">{{ card.value }}</strong>
              </div>
              <span class="grid size-12 place-items-center rounded-2xl border border-surface-200 bg-surface-50">
                <i [class]="card.icon"></i>
              </span>
            </div>
          </p-card>
        }
      </div>

      <div class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div class="space-y-1">
              <h2 class="m-0 text-xl font-semibold">Actiuni prioritare</h2>
              <p class="m-0 text-sm text-muted-color">
                Trasee rapide pentru zonele in care registratura, conformitatea sau conducerea trebuie sa intervina.
              </p>
            </div>
            <p-tag [value]="quickLinks().length + ' trasee'" severity="secondary" />
          </div>

          <div class="mt-4 grid gap-3">
            @for (item of quickLinks(); track item.key) {
              <article class="rounded-xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="font-medium">{{ item.title }}</div>
                  <p class="m-0 text-sm text-muted-color">{{ item.summary }}</p>
                </div>
                <div class="mt-3">
                  <p-button
                    [routerLink]="item.route"
                    [queryParams]="item.queryParams"
                    label="Deschide lista"
                    icon="pi pi-arrow-right"
                    size="small"
                  />
                </div>
              </article>
            }
          </div>
        </section>

        <div class="grid gap-4">
          <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
            <div class="space-y-1">
              <h2 class="m-0 text-xl font-semibold">Traseu operational</h2>
              <p class="m-0 text-sm text-muted-color">
                Secventa recomandata pentru verificare, publicare si inchidere documentara.
              </p>
            </div>

            <div class="mt-4 grid gap-3">
              @for (item of playbook(); track item.id) {
                <article class="rounded-xl border border-surface p-4">
                  <div class="space-y-2">
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div class="font-medium">{{ item.title }}</div>
                      <p-tag [value]="item.actorName" severity="secondary" />
                    </div>
                    <p class="m-0 text-sm text-muted-color">{{ item.summary }}</p>
                  </div>
                </article>
              }
            </div>
          </section>

          <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
            <div class="space-y-1">
              <h2 class="m-0 text-xl font-semibold">Zone urmarite</h2>
              <p class="m-0 text-sm text-muted-color">
                Ariile institutionale conectate direct la publicare, dovada si trasabilitate.
              </p>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              <p-tag [value]="complianceSectionLabel" severity="contrast" />
              <p-tag [value]="governanceSectionLabel" severity="contrast" />
              <p-tag value="Publicari" severity="secondary" />
              <p-tag value="Regulamente" severity="secondary" />
              <p-tag value="Dovezi" severity="secondary" />
              <p-tag value="Anonimizare" severity="secondary" />
            </div>
          </section>
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationComplianceDashboardComponent {
  protected readonly authz = inject(AuthzService);
  protected readonly institutionName = this.authz.institutionName;
  protected readonly complianceSectionLabel = EDUCATION_COMPLIANCE_SECTION.label;
  protected readonly governanceSectionLabel = EDUCATION_GOVERNANCE_SECTION.label;
  protected readonly roleTags = computed(() =>
    this.authz.roles().length
      ? this.authz.roles().map((role) => this.authz.roleLabel(role))
      : ['Conformitate'],
  );

  protected readonly metrics = computed<ComplianceMetricCard[]>(() => {
    const canReadCompliance = this.authz.hasPermission('education.read');
    const canReadGovernance = this.authz.hasPermission('education.governance.read');
    const canReadManagerial = this.authz.hasPermission('education.managerial.read');
    const canReadRegulations = this.authz.hasPermission('education.regulations.read');

    return [
      {
        key: 'requirements',
        label: 'Registre conformitate',
        icon: 'pi pi-check-square',
        value: canReadCompliance ? EDUCATION_COMPLIANCE_SECTION.resources.length : 0,
      },
      {
        key: 'governance',
        label: 'Fluxuri conectate',
        icon: 'pi pi-sitemap',
        value: [canReadGovernance, canReadManagerial, canReadRegulations].filter(Boolean).length,
      },
      {
        key: 'actions',
        label: 'Actiuni rapide',
        icon: 'pi pi-bolt',
        value: this.quickLinks().length,
      },
      {
        key: 'identity',
        label: 'Rol curent',
        icon: 'pi pi-user',
        value: this.roleTags().join(' / '),
      },
    ];
  });

  protected readonly quickLinks = computed<ComplianceQuickLink[]>(() => {
    const items: ComplianceQuickLink[] = [];

    if (this.authz.hasPermission('education.read')) {
      items.push(
        {
          key: 'requirements',
          title: 'Cerinte institutionale',
          summary: 'Verifica cerintele active, stadiul implementarii si zonele care au ramas partial acoperite.',
          route: '/education/compliance',
          queryParams: { resource: 'requirements', presetLabel: 'Cerinte institutionale' },
        },
        {
          key: 'publications',
          title: 'Publicari si dovezi',
          summary: 'Deschide registrul de publicari pentru a urmari restanta, anonimizarea si starea de publicare.',
          route: '/education/compliance',
          queryParams: { resource: 'publications', presetLabel: 'Publicari si dovezi' },
        },
      );
    }

    if (this.authz.hasPermission('education.managerial.read')) {
      items.push({
        key: 'managerial',
        title: 'Documente manageriale in lucru',
        summary: 'Acces rapid la documentele manageriale care pot necesita validare, publicare sau corelare cu cerinte.',
        route: '/education/governance',
        queryParams: { resource: 'managerial', filter_status: 'review', presetLabel: 'Documente manageriale in review' },
      });
    }

    if (this.authz.hasPermission('education.regulations.read')) {
      items.push({
        key: 'regulations',
        title: 'Regulamente si versiuni',
        summary: 'Urmareste regulamentele aflate in consultare, aprobare sau publicare pentru trasabilitate institutionala.',
        route: '/education/governance',
        queryParams: { resource: 'regulations', filter_status: 'consultation', presetLabel: 'Regulamente in consultare' },
      });
    }

    return items;
  });

  protected readonly playbook = computed<CompliancePlaybookItem[]>(() => {
    const steps: CompliancePlaybookItem[] = [];

    if (this.authz.hasRole('registrator')) {
      steps.push(
        {
          id: 'registrator-1',
          title: 'Verifica intrarile care cer publicare',
          summary: 'Identifica documentele si hotararile care trebuie mutate din registru in circuitul de publicare sau arhivare.',
          actorName: 'Registrator',
        },
        {
          id: 'registrator-2',
          title: 'Coreleaza dovada cu publicarea',
          summary: 'Asigura-te ca fiecare publicare are traseu, referinta si dovada atasata in registrul de conformitate.',
          actorName: 'Registrator',
        },
      );
    }

    if (this.authz.hasRole('gdpr_officer')) {
      steps.push({
        id: 'gdpr-1',
        title: 'Revizuieste anonimizarea',
        summary: 'Verifica documentele care ies din zona interna si confirma ca datele sensibile sunt tratate corect.',
        actorName: 'GDPR',
      });
    }

    if (this.authz.hasRole('inspector')) {
      steps.push({
        id: 'inspector-1',
        title: 'Pregateste verificarea institutionala',
        summary: 'Porneste din registrele de conformitate si regulamente pentru a identifica rapid ce documente trebuie cerute.',
        actorName: 'Inspector',
      });
    }

    if (!steps.length) {
      steps.push({
        id: 'default-1',
        title: 'Deschide registrul de conformitate',
        summary: 'Incepe cu cerintele si publicările, apoi coboara in documentele manageriale sau in regulamentele asociate.',
        actorName: 'Coordonare',
      });
    }

    return steps;
  });
}
