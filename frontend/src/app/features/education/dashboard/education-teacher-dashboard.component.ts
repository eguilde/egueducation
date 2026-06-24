import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';

interface TeacherMetricCard {
  key: string;
  label: string;
  icon: string;
  value: number | string;
}

interface TeacherQuickLink {
  key: string;
  title: string;
  summary: string;
  route: string;
  queryParams?: Record<string, string>;
}

interface TeacherPlaybookItem {
  id: string;
  title: string;
  summary: string;
  actorName: string;
}

@Component({
  selector: 'app-education-teacher-dashboard',
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
              <i class="pi pi-user"></i>
              Experienta cadru didactic
            </div>
            <h1 class="m-0 text-3xl font-semibold">Cockpit profesor</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Intrare rapida pentru portofoliu, evaluare, declaratii si fluxurile personale pe care cadrul didactic le
              actualizeaza in mod curent.
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
              <h2 class="m-0 text-xl font-semibold">Fluxuri personale</h2>
              <p class="m-0 text-sm text-muted-color">
                Traseele pe care profesorul le foloseste cel mai des pentru actualizare, verificare si depunere.
              </p>
            </div>
            <p-tag [value]="quickLinks().length + ' fluxuri'" severity="secondary" />
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
              <h2 class="m-0 text-xl font-semibold">Traseu de lucru</h2>
              <p class="m-0 text-sm text-muted-color">
                Ordinea recomandata pentru completare, verificare si predare a informatiilor proprii.
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
                Modulul este pregatit deja pentru portofolii, evaluari, declaratii si trasee administrative personale.
              </p>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              <p-tag value="Portofolii CD" severity="contrast" />
              <p-tag value="Evaluari anuale" severity="contrast" />
              <p-tag value="Declaratii" severity="secondary" />
              <p-tag value="Mobilitate" severity="secondary" />
              <p-tag value="Gradatii" severity="secondary" />
            </div>
          </section>
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationTeacherDashboardComponent {
  protected readonly authz = inject(AuthzService);
  protected readonly institutionName = this.authz.institutionName;
  protected readonly roleTags = computed(() =>
    this.authz.roles().length
      ? this.authz.roles().map((role) => this.authz.roleLabel(role))
      : ['Profesor'],
  );

  protected readonly metrics = computed<TeacherMetricCard[]>(() => [
    {
      key: 'portfolio',
      label: 'Portofoliu profesional',
      icon: 'pi pi-folder-open',
      value: this.authz.hasPermission('education.portfolios.read') ? 'Activ' : 'Fara acces',
    },
    {
      key: 'evaluations',
      label: 'Evaluare anuala',
      icon: 'pi pi-chart-line',
      value: this.authz.hasPermission('education.evaluations.read') ? 'Monitorizare' : 'Fara acces',
    },
    {
      key: 'declarations',
      label: 'Declaratii si anexe',
      icon: 'pi pi-file-edit',
      value: this.authz.hasPermission('education.declarations.read') ? 'Disponibil' : 'Fara acces',
    },
    {
      key: 'identity',
      label: 'Identitate curenta',
      icon: 'pi pi-user',
      value: this.roleTags().join(' / '),
    },
  ]);

  protected readonly quickLinks = computed<TeacherQuickLink[]>(() => {
    const items: TeacherQuickLink[] = [];

    if (this.authz.hasPermission('education.portfolios.read')) {
      items.push({
        key: 'portfolio',
        title: 'Portofoliul meu',
        summary: 'Deschide ecranul dedicat profesorului si portofoliul filtrat pe titularul curent.',
        route: '/education/portfolio/me',
        queryParams: {},
      });
    }

    if (this.authz.hasPermission('education.evaluations.read')) {
      items.push({
        key: 'evaluations',
        title: 'Evaluari si feedback',
        summary: 'Deschide evaluarea anuala, criteriile si comunicarea rezultatului pentru urmarirea progresului personal.',
        route: '/education/personnel',
        queryParams: { resource: 'evaluations', presetLabel: 'Evaluarile mele' },
      });
    }

    if (this.authz.hasPermission('education.declarations.read')) {
      items.push({
        key: 'declarations',
        title: 'Declaratii si documente asumate',
        summary: 'Vezi declaratiile depuse si documentele institutionale care trebuie confirmate sau actualizate.',
        route: '/education/personnel',
        queryParams: { resource: 'declarations', presetLabel: 'Declaratiile mele' },
      });
    }

    if (this.authz.hasPermission('education.mobility.read')) {
      items.push({
        key: 'mobility',
        title: 'Mobilitate si trasee administrative',
        summary: 'Urmareste cererile de mobilitate, documentele suport si etapele administrative asociate.',
        route: '/education/personnel',
        queryParams: { resource: 'mobility', presetLabel: 'Mobilitatea mea' },
      });
    }

    return items;
  });

  protected readonly playbook = computed<TeacherPlaybookItem[]>(() => [
    {
      id: 'teacher-1',
      title: 'Actualizeaza portofoliul',
      summary: 'Incepe cu documentele si sectiunile care au nevoie de completare sau revizie inainte de validare.',
      actorName: 'Profesor',
    },
    {
      id: 'teacher-2',
      title: 'Verifica evaluarea si criteriile',
      summary: 'Revizuieste evaluarea anuala, observa contestatiile sau comunicarea rezultatului si pregateste eventualele completari.',
      actorName: 'Profesor',
    },
    {
      id: 'teacher-3',
      title: 'Confirma declaratiile si traseele administrative',
      summary: 'Pastreaza la zi declaratiile, documentele de conformitate si eventualele cereri de mobilitate sau gradatie.',
      actorName: 'Profesor',
    },
  ]);
}
