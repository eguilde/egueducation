import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';

import { AdminApiService } from '../../core/api/admin-api.service';

@Component({
  selector: 'app-admin-dashboard-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, MatButtonModule, MatCardModule, MatChipsModule, MatIconModule],
  templateUrl: './admin-dashboard-page.component.html',
  styleUrl: './admin-dashboard-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminDashboardPageComponent {
  private readonly adminApi = inject(AdminApiService);

  protected readonly dashboard = toSignal(this.adminApi.dashboard(), {
    initialValue: {
      stats: {
        users: 0,
        memberships: 0,
        positions: 0,
        permissions: 0,
        workflows: 0,
        archives: 0,
      },
      modules: [],
      admin_sections: [],
      warnings: [],
    },
  });

  protected readonly statCards = computed(() => [
    { icon: 'group', value: String(this.dashboard().stats.users), titleKey: 'admin.stats.users' },
    { icon: 'badge', value: String(this.dashboard().stats.positions), titleKey: 'admin.stats.positions' },
    { icon: 'account_tree', value: String(this.dashboard().stats.workflows), titleKey: 'admin.stats.workflows' },
    { icon: 'verified_user', value: String(this.dashboard().stats.permissions), titleKey: 'admin.stats.permissions' },
  ]);

  protected readonly sections = computed(() =>
    this.dashboard().admin_sections.map((section) => `admin.sections.${section}`),
  );
}
