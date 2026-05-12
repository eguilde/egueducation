import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { combineLatest, map } from 'rxjs';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';

import { EarchivaApiService } from '../../core/api/earchiva-api.service';
import { RegistraturaApiService } from '../../core/api/registratura-api.service';
import { WorkflowApiService } from '../../core/api/workflow-api.service';

@Component({
  selector: 'app-documente-dashboard-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, MatButtonModule, MatCardModule, MatIconModule],
  template: `
    <section class="workspace-dashboard space-y-4">
      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        @for (card of statCards(); track card.labelKey) {
          <mat-card appearance="outlined" class="workspace-dashboard__stat-card rounded-3xl p-5">
            <mat-icon>{{ card.icon }}</mat-icon>
            <strong>{{ card.value }}</strong>
            <span>{{ card.labelKey | transloco }}</span>
          </mat-card>
        }
      </section>

      <section class="grid gap-4 lg:grid-cols-3">
        <mat-card appearance="outlined" class="workspace-dashboard__module-card rounded-3xl p-5">
          <div class="space-y-2">
            <h2>{{ 'nav.registratura' | transloco }}</h2>
            <p>{{ 'documenteWorkspace.cards.registratura' | transloco }}</p>
          </div>
          <a mat-flat-button routerLink="/documente/registratura">{{ 'documenteWorkspace.open' | transloco }}</a>
        </mat-card>

        <mat-card appearance="outlined" class="workspace-dashboard__module-card rounded-3xl p-5">
          <div class="space-y-2">
            <h2>{{ 'nav.documentFlow' | transloco }}</h2>
            <p>{{ 'documenteWorkspace.cards.workflow' | transloco }}</p>
          </div>
          <a mat-flat-button routerLink="/documente/workflow">{{ 'documenteWorkspace.open' | transloco }}</a>
        </mat-card>

        <mat-card appearance="outlined" class="workspace-dashboard__module-card rounded-3xl p-5">
          <div class="space-y-2">
            <h2>{{ 'nav.earchiva' | transloco }}</h2>
            <p>{{ 'documenteWorkspace.cards.earchiva' | transloco }}</p>
          </div>
          <a mat-flat-button routerLink="/documente/earchiva">{{ 'documenteWorkspace.open' | transloco }}</a>
        </mat-card>
      </section>
    </section>
  `,
  styles: `
    .workspace-dashboard__module-card,
    .workspace-dashboard__stat-card {
      display: grid;
      gap: 0.9rem;
    }

    .workspace-dashboard__module-card h2 {
      margin: 0;
    }

    .workspace-dashboard__module-card p {
      margin: 0;
      color: var(--text-soft);
    }

    .workspace-dashboard__stat-card mat-icon {
      color: var(--brand-600);
    }

    .workspace-dashboard__stat-card strong {
      font-size: clamp(1.8rem, 2vw, 2.4rem);
      line-height: 1;
    }

    .workspace-dashboard__stat-card span {
      color: var(--text-soft);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DocumenteDashboardPageComponent {
  private readonly registraturaApi = inject(RegistraturaApiService);
  private readonly workflowApi = inject(WorkflowApiService);
  private readonly earchivaApi = inject(EarchivaApiService);

  protected readonly snapshot = toSignal(
    combineLatest([
      this.registraturaApi.documents({ page: 1, pageSize: 1, filters: {} }),
      this.workflowApi.dashboard(),
      this.earchivaApi.dashboard(),
    ]).pipe(
      map(([registratura, workflow, archive]) => ({
        registraturaTotal: registratura.total,
        activeTasks: workflow.stats.active_tasks,
        blockedDossiers: workflow.stats.blocked_dossiers,
        archiveTotal: archive.stats.total_records,
      })),
    ),
    {
      initialValue: {
        registraturaTotal: 0,
        activeTasks: 0,
        blockedDossiers: 0,
        archiveTotal: 0,
      },
    },
  );

  protected readonly statCards = computed(() => [
    {
      icon: 'description',
      labelKey: 'documenteWorkspace.stats.registratura',
      value: this.snapshot().registraturaTotal,
    },
    {
      icon: 'account_tree',
      labelKey: 'documenteWorkspace.stats.workflow',
      value: this.snapshot().activeTasks,
    },
    {
      icon: 'warning',
      labelKey: 'documenteWorkspace.stats.blocked',
      value: this.snapshot().blockedDossiers,
    },
    {
      icon: 'inventory_2',
      labelKey: 'documenteWorkspace.stats.archive',
      value: this.snapshot().archiveTotal,
    },
  ]);
}
