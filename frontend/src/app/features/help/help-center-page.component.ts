import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DividerModule } from 'primeng/divider';
import { FieldsetModule } from 'primeng/fieldset';
import { InputTextModule } from 'primeng/inputtext';
import { TagModule } from 'primeng/tag';

import helpContent from './education-help-content.json';

type HelpStatus = 'all' | 'covered' | 'partial';
type ChapterStatus = 'covered' | 'partial';

interface HelpQuickLink {
  label: string;
  route: string;
  icon: string;
}

interface HelpSource {
  label: string;
  short: string;
  pdf: string;
}

interface HelpChapter {
  id: string;
  status: ChapterStatus;
  title: string;
  audience: string;
  summary: string;
  inAppSteps: string[];
  outputs: string[];
  routes: string[];
}

interface HelpContent {
  title: string;
  subtitle: string;
  sources: HelpSource[];
  quickLinks: HelpQuickLink[];
  chapters: HelpChapter[];
}

const HELP_CONTENT = helpContent as HelpContent;

@Component({
  selector: 'app-help-center-page',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    ButtonModule,
    CardModule,
    DividerModule,
    FieldsetModule,
    InputTextModule,
    TagModule,
  ],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-3">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-question-circle"></i>
              Manual operational in aplicatie
            </div>
            <h1 class="m-0 text-3xl font-semibold">{{ content.title }}</h1>
            <p class="m-0 max-w-4xl text-sm leading-6 text-muted-color">
              {{ content.subtitle }}
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <p-button
              routerLink="/education/dashboard"
              icon="pi pi-chart-bar"
              label="Panou educație"
              size="small"
            />
            <p-button
              routerLink="/education/governance"
              icon="pi pi-users"
              label="Guvernanță"
              severity="secondary"
              size="small"
            />
            <p-button
              routerLink="/education/portfolio"
              icon="pi pi-folder-open"
              label="Portofolii"
              severity="secondary"
              size="small"
            />
          </div>
        </div>
      </div>

      <div class="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <p-fieldset legend="Surse și acoperire" styleClass="border border-surface bg-surface-0 shadow-sm">
          <div class="grid gap-3 md:grid-cols-2">
            @for (source of content.sources; track source.pdf) {
              <article class="rounded-2xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="flex items-center gap-2 font-semibold">
                    <i class="pi pi-file-pdf"></i>
                    <span>{{ source.label }}</span>
                  </div>
                  <p class="m-0 text-sm text-muted-color">{{ source.short }}</p>
                  <p class="m-0 text-xs text-muted-color">{{ source.pdf }}</p>
                </div>
              </article>
            }
          </div>

          <p-divider />

          <div class="grid gap-3 sm:grid-cols-3">
            <div class="rounded-2xl border border-surface p-4">
              <p class="m-0 text-xs uppercase tracking-[0.18em] text-muted-color">Capitole</p>
              <strong class="mt-2 block text-3xl">{{ stats().total }}</strong>
            </div>
            <div class="rounded-2xl border border-surface p-4">
              <p class="m-0 text-xs uppercase tracking-[0.18em] text-muted-color">Acoperite</p>
              <strong class="mt-2 block text-3xl">{{ stats().covered }}</strong>
            </div>
            <div class="rounded-2xl border border-surface p-4">
              <p class="m-0 text-xs uppercase tracking-[0.18em] text-muted-color">De rafinat</p>
              <strong class="mt-2 block text-3xl">{{ stats().partial }}</strong>
            </div>
          </div>
        </p-fieldset>

        <p-fieldset legend="Cum se folosește" styleClass="border border-surface bg-surface-0 shadow-sm">
          <div class="space-y-4">
            <ol class="m-0 grid gap-3 pl-5 text-sm leading-6 text-color">
              <li>Caută capitolul după nume, rol sau rută.</li>
              <li>Deschide secțiunea relevantă și urmează pașii operaționali.</li>
              <li>Verifică rezultatul așteptat și documentele exportabile.</li>
              <li>Folosește rutele rapide când trebuie să intri direct în flux.</li>
            </ol>

            <div class="grid gap-2 sm:grid-cols-2">
              @for (link of content.quickLinks; track link.route) {
                <a
                  [routerLink]="link.route"
                  class="flex items-center justify-between rounded-2xl border border-surface px-4 py-3 no-underline transition-colors hover:bg-surface-50"
                >
                  <span class="inline-flex items-center gap-2 text-color">
                    <i [class]="link.icon"></i>
                    <span>{{ link.label }}</span>
                  </span>
                  <i class="pi pi-angle-right text-muted-color"></i>
                </a>
              }
            </div>
          </div>
        </p-fieldset>
      </div>

      <p-fieldset legend="Căutare și filtrare" styleClass="border border-surface bg-surface-0 shadow-sm">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex-1">
            <label class="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-muted-color">
              Filtru text
            </label>
            <input
              pInputText
              class="w-full"
              type="search"
              [value]="query()"
              placeholder="Caută după titlu, public sau rută"
              (input)="onQueryInput($event)"
            />
          </div>

          <div class="flex flex-wrap gap-2">
            <p-button
              label="Toate"
              size="small"
              [severity]="statusFilter() === 'all' ? 'primary' : 'secondary'"
              (onClick)="setStatusFilter('all')"
            />
            <p-button
              label="Acoperite"
              size="small"
              [severity]="statusFilter() === 'covered' ? 'success' : 'secondary'"
              (onClick)="setStatusFilter('covered')"
            />
            <p-button
              label="Parțiale"
              size="small"
              [severity]="statusFilter() === 'partial' ? 'warn' : 'secondary'"
              (onClick)="setStatusFilter('partial')"
            />
          </div>
        </div>
      </p-fieldset>

      <div class="grid gap-4 xl:grid-cols-2">
        @for (chapter of filteredChapters(); track chapter.id) {
          <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
            <div class="space-y-4">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-2">
                  <div class="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-color">
                    <i class="pi pi-book"></i>
                    {{ chapter.audience }}
                  </div>
                  <h2 class="m-0 text-xl font-semibold">{{ chapter.title }}</h2>
                </div>
                <p-tag
                  [value]="chapter.status === 'covered' ? 'Acoperit' : 'De rafinat'"
                  [severity]="chapter.status === 'covered' ? 'success' : 'warn'"
                />
              </div>

              <p class="m-0 text-sm leading-6 text-muted-color">{{ chapter.summary }}</p>

              <div class="grid gap-4 md:grid-cols-2">
                <div class="rounded-2xl border border-surface p-4">
                  <p class="m-0 text-xs font-semibold uppercase tracking-[0.18em] text-muted-color">Pași în aplicație</p>
                  <ol class="mt-3 space-y-2 pl-5 text-sm leading-6 text-color">
                    @for (step of chapter.inAppSteps; track step) {
                      <li>{{ step }}</li>
                    }
                  </ol>
                </div>

                <div class="rounded-2xl border border-surface p-4">
                  <p class="m-0 text-xs font-semibold uppercase tracking-[0.18em] text-muted-color">Ieșiri așteptate</p>
                  <ul class="mt-3 space-y-2 pl-5 text-sm leading-6 text-color">
                    @for (output of chapter.outputs; track output) {
                      <li>{{ output }}</li>
                    }
                  </ul>
                </div>
              </div>

              <div class="space-y-2">
                <p class="m-0 text-xs font-semibold uppercase tracking-[0.18em] text-muted-color">Rute asociate</p>
                <div class="flex flex-wrap gap-2">
                  @for (route of chapter.routes; track route) {
                    <a
                      [routerLink]="route"
                      class="rounded-full border border-surface px-3 py-1 text-xs font-medium text-color no-underline transition-colors hover:bg-surface-50"
                    >
                      {{ route }}
                    </a>
                  }
                </div>
              </div>
            </div>
          </p-card>
        } @empty {
          <div class="rounded-2xl border border-dashed border-surface-300 p-6 text-sm text-muted-color dark:border-surface-700">
            Nu am găsit capitole pentru filtrul curent. Resetează căutarea sau schimbă starea.
          </div>
        }
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HelpCenterPageComponent {
  protected readonly content = HELP_CONTENT;
  protected readonly query = signal('');
  protected readonly statusFilter = signal<HelpStatus>('all');

  protected readonly filteredChapters = computed(() => {
    const query = this.query().trim().toLowerCase();
    const status = this.statusFilter();

    return this.content.chapters.filter((chapter) => {
      const matchesStatus = status === 'all' || chapter.status === status;
      const haystack = [
        chapter.title,
        chapter.audience,
        chapter.summary,
        chapter.id,
        ...chapter.routes,
        ...chapter.inAppSteps,
        ...chapter.outputs,
      ]
        .join(' ')
        .toLowerCase();

      return matchesStatus && (!query || haystack.includes(query));
    });
  });

  protected readonly stats = computed(() => ({
    total: this.content.chapters.length,
    covered: this.content.chapters.filter((chapter) => chapter.status === 'covered').length,
    partial: this.content.chapters.filter((chapter) => chapter.status === 'partial').length,
  }));

  protected onQueryInput(event: Event): void {
    const value = (event.target as HTMLInputElement | null)?.value ?? '';
    this.query.set(value);
  }

  protected setStatusFilter(value: HelpStatus): void {
    this.statusFilter.set(value);
  }
}
