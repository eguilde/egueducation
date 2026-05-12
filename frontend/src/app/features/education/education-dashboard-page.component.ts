import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { combineLatest, map } from 'rxjs';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';

import { EducationApiService } from '../../core/api/education-api.service';

@Component({
  selector: 'app-education-dashboard-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, MatButtonModule, MatCardModule, MatIconModule],
  template: `
    <section class="grid gap-5">
      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        @for (card of statCards(); track card.labelKey) {
          <mat-card appearance="outlined" class="grid gap-3 rounded-[1.25rem] border-slate-200/70 bg-white/85 p-5 shadow-sm shadow-rose-100/40 dark:border-slate-800 dark:bg-slate-950/60">
            <mat-icon class="text-rose-600 dark:text-rose-300">{{ card.icon }}</mat-icon>
            <strong class="text-[clamp(1.8rem,2vw,2.4rem)] leading-none text-slate-900 dark:text-slate-50">{{ card.value }}</strong>
            <span class="text-sm text-slate-600 dark:text-slate-300">{{ card.labelKey | transloco }}</span>
          </mat-card>
        }
      </section>

      <section class="grid gap-4 xl:grid-cols-3">
        <mat-card appearance="outlined" class="grid gap-4 rounded-[1.25rem] border-slate-200/70 bg-white/85 p-5 shadow-sm shadow-rose-100/40 dark:border-slate-800 dark:bg-slate-950/60">
          <div class="grid gap-2">
            <p class="m-0 text-[0.78rem] font-bold uppercase tracking-[0.14em] text-rose-600">{{ 'educationWorkspace.tabs.governance' | transloco }}</p>
            <h2 class="m-0 text-lg font-semibold text-slate-900 dark:text-slate-50">{{ 'educationWorkspace.cards.governanceTitle' | transloco }}</h2>
            <p class="m-0 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ 'educationWorkspace.cards.governanceBody' | transloco }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <a mat-flat-button routerLink="/education/governance">{{ 'educationWorkspace.open' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/decisions">{{ 'nav.educationDecisions' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/regulations">{{ 'nav.educationRegulations' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/managerial">{{ 'nav.educationManagerial' | transloco }}</a>
          </div>
        </mat-card>

        <mat-card appearance="outlined" class="grid gap-4 rounded-[1.25rem] border-slate-200/70 bg-white/85 p-5 shadow-sm shadow-rose-100/40 dark:border-slate-800 dark:bg-slate-950/60">
          <div class="grid gap-2">
            <p class="m-0 text-[0.78rem] font-bold uppercase tracking-[0.14em] text-rose-600">{{ 'nav.educationPersonnel' | transloco }}</p>
            <h2 class="m-0 text-lg font-semibold text-slate-900 dark:text-slate-50">{{ 'educationWorkspace.cards.personnelTitle' | transloco }}</h2>
            <p class="m-0 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ 'educationWorkspace.cards.personnelBody' | transloco }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <a mat-flat-button routerLink="/education/personnel">{{ 'educationWorkspace.open' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/evaluations">{{ 'nav.educationEvaluations' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/declarations">{{ 'nav.educationDeclarations' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/mobility">{{ 'nav.educationMobility' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/gradatii">{{ 'nav.educationGradatii' | transloco }}</a>
          </div>
        </mat-card>

        <mat-card appearance="outlined" class="grid gap-4 rounded-[1.25rem] border-slate-200/70 bg-white/85 p-5 shadow-sm shadow-rose-100/40 dark:border-slate-800 dark:bg-slate-950/60">
          <div class="grid gap-2">
            <p class="m-0 text-[0.78rem] font-bold uppercase tracking-[0.14em] text-rose-600">{{ 'nav.educationPortfolios' | transloco }}</p>
            <h2 class="m-0 text-lg font-semibold text-slate-900 dark:text-slate-50">{{ 'educationWorkspace.cards.portfolioTitle' | transloco }}</h2>
            <p class="m-0 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ 'educationWorkspace.cards.portfolioBody' | transloco }}</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <a mat-flat-button routerLink="/education/portfolios">{{ 'educationWorkspace.open' | transloco }}</a>
            <a mat-stroked-button routerLink="/education/managerial">{{ 'nav.educationManagerial' | transloco }}</a>
          </div>
        </mat-card>
      </section>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationDashboardPageComponent {
  private readonly educationApi = inject(EducationApiService);

  protected readonly snapshot = toSignal(
    combineLatest([
      this.educationApi.dashboard(),
      this.educationApi.decisionsDashboard(),
      this.educationApi.managerialDashboard(),
    ]).pipe(
      map(([governance, decisions, managerial]) => ({
        meetings: governance.stats.total_meetings,
        scheduled: governance.stats.scheduled_meetings,
        decisions: decisions.stats.total_decisions,
        dossiers: managerial.stats.total_dossiers,
      })),
    ),
    {
      initialValue: {
        meetings: 0,
        scheduled: 0,
        decisions: 0,
        dossiers: 0,
      },
    },
  );

  protected readonly statCards = computed(() => [
    { icon: 'groups', labelKey: 'educationWorkspace.stats.meetings', value: this.snapshot().meetings },
    { icon: 'event_available', labelKey: 'educationWorkspace.stats.scheduled', value: this.snapshot().scheduled },
    { icon: 'gavel', labelKey: 'educationWorkspace.stats.decisions', value: this.snapshot().decisions },
    { icon: 'folder_special', labelKey: 'educationWorkspace.stats.dossiers', value: this.snapshot().dossiers },
  ]);
}
