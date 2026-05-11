import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';

import { AppApiService } from '../../core/api/app-api.service';
import { AuthzService } from '../../core/authz/authz.service';

@Component({
  selector: 'app-dashboard-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, MatButtonModule, MatCardModule, MatChipsModule, MatIconModule],
  templateUrl: './dashboard-page.component.html',
  styleUrl: './dashboard-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DashboardPageComponent {
  private readonly api = inject(AppApiService);
  protected readonly authz = inject(AuthzService);

  protected readonly meta = toSignal(this.api.meta(), {
    initialValue: {
      name: 'EguEducation',
      default_locale: 'ro',
      available_locales: ['ro', 'en'],
      theme: {
        family: 'material3-expressive',
        brand: 'red-rose',
      },
    },
  });

  protected readonly cards = [
    { icon: 'inbox', titleKey: 'dashboard.cards.registratura.title', bodyKey: 'dashboard.cards.registratura.body' },
    { icon: 'account_tree', titleKey: 'dashboard.cards.workflow.title', bodyKey: 'dashboard.cards.workflow.body' },
    { icon: 'inventory_2', titleKey: 'dashboard.cards.archive.title', bodyKey: 'dashboard.cards.archive.body' },
    { icon: 'school', titleKey: 'dashboard.cards.education.title', bodyKey: 'dashboard.cards.education.body' },
  ];

  protected readonly themeBadge = computed(
    () => `${this.meta().theme.family} / ${this.meta().theme.brand}`,
  );
}
