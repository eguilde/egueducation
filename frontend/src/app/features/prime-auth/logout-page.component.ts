import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { ProgressSpinnerModule } from 'primeng/progressspinner';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-logout-page',
  imports: [TranslocoPipe, ProgressSpinnerModule],
  template: `
    <main class="grid min-h-dvh place-items-center bg-surface-50 p-6 dark:bg-surface-950">
      <section class="rounded-[2rem] border border-surface-200 bg-white p-8 text-center shadow-xl dark:border-surface-800 dark:bg-surface-900">
        <p-progress-spinner strokeWidth="4" ariaLabel="logout" />
        <h1 class="mt-4 text-2xl font-black">{{ 'auth.logout' | transloco }}</h1>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LogoutPageComponent {
  private readonly auth = inject(AuthService);

  constructor() {
    void this.auth.logout();
  }
}
