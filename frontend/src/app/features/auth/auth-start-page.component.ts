import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { ActivatedRoute } from '@angular/router';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-auth-start-page',
  standalone: true,
  imports: [TranslocoPipe],
  template: `
    <main class="grid min-h-screen place-items-center bg-rose-50 px-6 text-center text-slate-950 dark:bg-slate-950 dark:text-white">
      <section class="max-w-md rounded-[2rem] border border-rose-100 bg-white/80 p-8 shadow-2xl shadow-rose-950/10 dark:border-white/10 dark:bg-white/10">
        <p class="text-sm font-black uppercase tracking-[0.2em] text-rose-700 dark:text-rose-200">
          {{ 'auth.redirectBadge' | transloco }}
        </p>
        <h1 class="mt-3 text-3xl font-black tracking-[-0.035em]">
          {{ 'auth.redirecting' | transloco }}
        </h1>
        <p class="mt-3 leading-7 text-slate-600 dark:text-slate-300">
          {{ 'auth.callbackMessage' | transloco }}
        </p>
        @if (error()) {
          <p class="mt-4 rounded-2xl bg-rose-100 px-4 py-3 text-sm font-semibold text-rose-800 dark:bg-rose-950/50 dark:text-rose-100">
            {{ error() }}
          </p>
        }
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthStartPageComponent {
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  protected readonly error = signal<string | null>(null);

  constructor() {
    void this.start();
  }

  private async start(): Promise<void> {
    try {
      const returnUrl = this.route.snapshot.queryParamMap.get('returnUrl') || '/dashboard';
      await this.auth.login(returnUrl);
    } catch (error) {
      this.error.set(error instanceof Error ? error.message : 'Authentication redirect could not be started.');
    }
  }
}
