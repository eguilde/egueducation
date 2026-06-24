import { CommonModule } from '@angular/common';
import { httpResource } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { FieldsetModule } from 'primeng/fieldset';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';
import {
  EDUCATION_DASHBOARD_CARDS,
  EDUCATION_ROUTE_TABS,
  EDUCATION_SECTIONS,
} from '../shared/education-config';

interface DashboardCardState {
  key: string;
  label: string;
  icon: string;
  value: number | string;
}

@Component({
  selector: 'app-education-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ButtonModule,
    CardModule,
    FieldsetModule,
    TagModule,
  ],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-8">
        @for (card of cards(); track card.key) {
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

      <div class="grid gap-4 xl:grid-cols-[1.3fr_0.9fr]">
        <p-fieldset legend="Fluxuri educationale" styleClass="border border-surface bg-surface-0 shadow-sm">
          <div class="grid gap-3 md:grid-cols-2">
            @for (section of sections(); track section.key) {
              <article class="rounded-2xl border border-surface p-4">
                <div class="flex items-start justify-between gap-3">
                  <div class="space-y-2">
                    <div class="inline-flex items-center gap-2 font-semibold">
                      <i [class]="section.icon"></i>
                      <span>{{ section.label }}</span>
                    </div>
                    <p class="m-0 text-sm text-muted-color">{{ section.description }}</p>
                  </div>
                  <p-tag [value]="section.resources.length + ' tabele'" severity="secondary" />
                </div>
                <div class="mt-4 flex flex-wrap gap-2">
                  @for (resource of section.resources; track resource.key) {
                    <p-tag [value]="resource.label" severity="contrast" />
                  }
                </div>
                <div class="mt-4">
                  <p-button
                    [routerLink]="['/education', section.key]"
                    [label]="'Deschide ' + section.label"
                    icon="pi pi-arrow-right"
                    size="small"
                  />
                </div>
              </article>
            }
          </div>
        </p-fieldset>

        <p-fieldset legend="Organizare" styleClass="border border-surface bg-surface-0 shadow-sm">
          <div class="space-y-4">
            <p class="m-0 text-sm text-muted-color">
              Modulul este acum impartit pe domenii functionale separate, fiecare cu tabele PrimeNG server-side, filtre in antet si dialoguri dedicate.
            </p>
            @if (authz.hasRole('director')) {
              <div class="rounded-2xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="font-semibold">Cockpit director</div>
                  <p class="m-0 text-sm text-muted-color">
                    Vedere operationala orientata pe management educational, cu semnale si trasee recomandate.
                  </p>
                </div>
                <div class="mt-3">
                  <p-button
                    [routerLink]="['/education', 'dashboard', 'director']"
                    label="Deschide cockpitul"
                    icon="pi pi-briefcase"
                    size="small"
                  />
                </div>
              </div>
            }
            @if (authz.hasAnyRole(['secretar', 'registrator'])) {
              <div class="rounded-2xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="font-semibold">Cockpit secretariat</div>
                  <p class="m-0 text-sm text-muted-color">
                    Acces rapid la sedinte, inregistrari, portofolii si verificari filtrate dupa rolul operational curent.
                  </p>
                </div>
                <div class="mt-3">
                  <p-button
                    [routerLink]="['/education', 'dashboard', 'secretariat']"
                    label="Deschide cockpitul"
                    icon="pi pi-inbox"
                    size="small"
                  />
                </div>
              </div>
            }
            @if (authz.hasAnyRole(['registrator', 'gdpr_officer', 'inspector']) || authz.hasPermission('education.read')) {
              <div class="rounded-2xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="font-semibold">Cockpit conformitate</div>
                  <p class="m-0 text-sm text-muted-color">
                    Intrare orientata pe cerinte, publicari, dovada, anonimizare si trasee institutionale de control.
                  </p>
                </div>
                <div class="mt-3">
                  <p-button
                    [routerLink]="['/education', 'dashboard', 'compliance']"
                    label="Deschide cockpitul"
                    icon="pi pi-shield"
                    size="small"
                  />
                </div>
              </div>
            }
            @if (authz.hasRole('profesor')) {
              <div class="rounded-2xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="font-semibold">Cockpit profesor</div>
                  <p class="m-0 text-sm text-muted-color">
                    Intrare orientata pe portofoliu, evaluare, declaratii si traseele personale ale cadrului didactic.
                  </p>
                </div>
                <div class="mt-3">
                  <p-button
                    [routerLink]="['/education', 'dashboard', 'teacher']"
                    label="Deschide cockpitul"
                    icon="pi pi-user"
                    size="small"
                  />
                </div>
              </div>
            }
            <div class="grid gap-2">
              @for (tab of tabs(); track tab.path) {
                <a
                  class="flex items-center justify-between rounded-2xl border border-surface px-4 py-3 no-underline transition-colors hover:bg-surface-50"
                  [routerLink]="['/education', tab.path]"
                >
                  <span class="inline-flex items-center gap-2 text-color">
                    <i [class]="tab.icon"></i>
                    {{ tab.label }}
                  </span>
                  <i class="pi pi-angle-right text-muted-color"></i>
                </a>
              }
            </div>
          </div>
        </p-fieldset>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationDashboardComponent {
  protected readonly authz = inject(AuthzService);
  private readonly cardResources = EDUCATION_DASHBOARD_CARDS.map((card) => {
    const resource = this.authz.hasPermission(card.permission)
      ? httpResource<{ stats: Record<string, number> }>(() => ({
          url: card.endpoint,
          method: 'GET',
        }))
      : null;

    return { card, resource };
  });

  protected readonly tabs = computed(() =>
    EDUCATION_ROUTE_TABS.filter((tab) => tab.path !== 'dashboard' && tab.permissions.some((permission) => this.authz.hasPermission(permission))),
  );
  protected readonly sections = computed(() =>
    EDUCATION_SECTIONS
      .map((section) => ({
        ...section,
        resources: section.resources.filter((resource) => this.authz.hasPermission(resource.readPermission)),
      }))
      .filter((section) => section.resources.length > 0),
  );
  protected readonly cards = computed<DashboardCardState[]>(() =>
    this.cardResources.map(({ card, resource }) => ({
      key: card.key,
      label: card.label,
      icon: card.icon,
      value: !this.authz.hasPermission(card.permission)
        ? 'Fara acces'
        : resource?.isLoading()
          ? '-'
          : resource?.value()?.stats?.[card.statKey] ?? 0,
    })),
  );
}
