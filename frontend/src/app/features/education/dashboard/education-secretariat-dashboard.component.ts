import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';
import { EDUCATION_SECTIONS } from '../shared/education-config';

interface SecretariatSectionState {
  key: string;
  label: string;
  icon: string;
  description: string;
  resources: Array<{
    key: string;
    label: string;
    route: string;
    canCreate: boolean;
    createRoute?: string;
  }>;
}

interface SecretariatMetricCard {
  key: string;
  label: string;
  icon: string;
  value: number | string;
}

interface SecretariatPlaybookItem {
  id: string;
  title: string;
  summary: string;
  actorName: string;
}

@Component({
  selector: 'app-education-secretariat-dashboard',
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
              <i class="pi pi-inbox"></i>
              Secretariat educational
            </div>
            <h1 class="m-0 text-3xl font-semibold">Cockpit secretariat</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Intrare rapida pentru programare, verificare, inregistrare si urmarire operationala a domeniilor educationale
              pe care secretariatul le gestioneaza zi de zi.
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

      <div class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="flex items-center justify-between gap-3">
            <div class="space-y-1">
              <h2 class="m-0 text-xl font-semibold">Acces rapid</h2>
              <p class="m-0 text-sm text-muted-color">
                Zonele si actiunile disponibile sunt filtrate dupa permisiunile curente si dupa ce este relevant pentru
                secretariat.
              </p>
            </div>
            <p-tag [value]="sections().length + ' domenii'" severity="secondary" />
          </div>

          <div class="mt-4 grid gap-3">
            @for (section of sections(); track section.key) {
              <article class="rounded-xl border border-surface p-4">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="space-y-2">
                    <div class="inline-flex items-center gap-2 font-semibold">
                      <i [class]="section.icon"></i>
                      <span>{{ section.label }}</span>
                    </div>
                    <p class="m-0 text-sm text-muted-color">{{ section.description }}</p>
                  </div>
                  <p-tag [value]="section.resources.length + ' registre'" severity="secondary" />
                </div>

                <div class="mt-3 flex flex-wrap gap-2">
                  @for (resource of section.resources; track resource.key) {
                    <p-tag [value]="resource.label" severity="contrast" />
                  }
                </div>

                <div class="mt-4 flex flex-wrap gap-2">
                  <p-button
                    [routerLink]="['/education', section.key]"
                    label="Deschide zona"
                    icon="pi pi-arrow-right"
                    size="small"
                  />

                  @for (resource of section.resources; track resource.key) {
                    @if (resource.canCreate && resource.createRoute) {
                      <p-button
                        [routerLink]="resource.createRoute"
                        [label]="'Adauga ' + resource.label"
                        icon="pi pi-plus"
                        size="small"
                        severity="secondary"
                        [outlined]="true"
                      />
                    }
                  }
                </div>
              </article>
            }
          </div>
        </section>

        <div class="grid gap-4">
          <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
            <div class="space-y-1">
              <h2 class="m-0 text-xl font-semibold">Traseu de lucru</h2>
              <p class="m-0 text-sm text-muted-color">
                Ordinea de lucru sugerata pentru ziua curenta, in functie de rolurile si permisiunile disponibile.
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
              <h2 class="m-0 text-xl font-semibold">Roluri active</h2>
              <p class="m-0 text-sm text-muted-color">
                Utilizatorul curent este afisat aici exact asa cum il vede secretariatul.
              </p>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              @for (role of roleTags(); track role) {
                <p-tag [value]="role" severity="contrast" />
              }
            </div>
          </section>
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationSecretariatDashboardComponent {
  private readonly authz = inject(AuthzService);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly roleTags = computed(() =>
    this.authz.roles().length
      ? this.authz.roles().map((role) => this.authz.roleLabel(role))
      : ['Secretariat'],
  );

  protected readonly sections = computed<SecretariatSectionState[]>(() =>
    EDUCATION_SECTIONS
      .map((section) => ({
        key: section.key,
        label: section.label,
        icon: section.icon,
        description: section.description,
        resources: section.resources
          .filter((resource) => this.authz.hasPermission(resource.readPermission))
          .map((resource) => ({
            key: resource.key,
            label: resource.label,
            route: `/education/${section.key}`,
            canCreate: Boolean(resource.createWizardRoute && resource.managePermission && this.authz.hasPermission(resource.managePermission)),
            createRoute: resource.createWizardRoute,
          })),
      }))
      .filter((section) => section.resources.length > 0),
  );

  protected readonly metrics = computed<SecretariatMetricCard[]>(() => {
    const accessibleSections = this.sections();
    const accessibleResources = accessibleSections.reduce((total, section) => total + section.resources.length, 0);
    const creatableResources = accessibleSections.reduce(
      (total, section) => total + section.resources.filter((resource) => resource.canCreate).length,
      0,
    );

    return [
      {
        key: 'sections',
        label: 'Domenii vizibile',
        icon: 'pi pi-sitemap',
        value: accessibleSections.length,
      },
      {
        key: 'resources',
        label: 'Registre deschise',
        icon: 'pi pi-database',
        value: accessibleResources,
      },
      {
        key: 'create',
        label: 'Actiuni rapide',
        icon: 'pi pi-plus-circle',
        value: creatableResources,
      },
      {
        key: 'role',
        label: 'Identitate curenta',
        icon: 'pi pi-user',
        value: this.roleTags().join(' / '),
      },
    ];
  });

  protected readonly playbook = computed<SecretariatPlaybookItem[]>(() => {
    const steps: SecretariatPlaybookItem[] = [];

    if (this.authz.hasRole('secretar')) {
      steps.push(
        {
          id: 'secretar-1',
          title: 'Pregateste sedintele si minutele',
          summary: 'Intrarea principala pentru CA, minute, voturi si hotarari care trebuie verificate inainte de publicare.',
          actorName: 'Secretar',
        },
        {
          id: 'secretar-2',
          title: 'Verifica documentele manageriale',
          summary: 'Urmarirea dosarelor aflate in lucru si a traseului de avizare pentru documentele institutionale.',
          actorName: 'Secretar',
        },
      );
    }

    if (this.authz.hasRole('registrator')) {
      steps.push(
        {
          id: 'registrator-1',
          title: 'Receptioneaza si indexeaza inregistrari',
          summary: 'Fluxul zilnic pentru intrari, documente si materiale care intra in circuitul operational.',
          actorName: 'Registrator',
        },
        {
          id: 'registrator-2',
          title: 'Trimite catre verificare si arhiva',
          summary: 'Redirectioneaza dosarele catre zonele potrivite si inchide traseele pregatite pentru arhivare.',
          actorName: 'Registrator',
        },
      );
    }

    if (!steps.length) {
      steps.push({
        id: 'default-1',
        title: 'Deschide domeniul activ',
        summary: 'Alege registrul potrivit din stanga si continua cu lista filtrata pentru rolul curent.',
        actorName: 'Coordonare',
      });
    }

    return steps;
  });
}
