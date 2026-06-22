import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { ProgressSpinnerModule } from 'primeng/progressspinner';

import { AuthService } from '../../core/auth/auth.service';
import { AuthzService } from '../../core/authz/authz.service';

@Component({
  selector: 'app-callback-page',
  standalone: true,
  imports: [TranslocoPipe, ProgressSpinnerModule],
  template: `
    <main class="min-h-dvh overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.18),_transparent_30rem),linear-gradient(135deg,_#fff7f8_0%,_#ffffff_50%,_#ffe4e9_100%)] text-slate-950 dark:bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.28),_transparent_30rem),linear-gradient(135deg,_#0f172a_0%,_#020617_56%,_#260711_100%)] dark:text-white">
      <section class="mx-auto grid min-h-dvh w-full max-w-5xl items-center gap-8 px-5 py-8 md:grid-cols-[0.9fr_1.1fr] md:px-8 lg:px-10">
        <div class="space-y-6">
          <div class="inline-flex items-center rounded-full border border-rose-200 bg-white/75 px-4 py-2 text-sm font-semibold text-rose-700 shadow-sm backdrop-blur dark:border-rose-400/30 dark:bg-white/10 dark:text-rose-100">
            {{ 'auth.redirectBadge' | transloco }}
          </div>
          <h1 class="max-w-2xl text-4xl font-black tracking-[-0.045em] text-slate-950 md:text-6xl dark:text-white">
            {{ 'auth.redirecting' | transloco }}
          </h1>
          <p class="max-w-xl text-lg leading-8 text-slate-700 dark:text-slate-200">
            {{ 'auth.callbackMessage' | transloco }}
          </p>
        </div>

        <div class="rounded-[2rem] border border-white/70 bg-white/75 p-6 shadow-2xl shadow-rose-950/10 backdrop-blur-xl dark:border-white/10 dark:bg-white/10">
          <div class="rounded-[1.75rem] border border-rose-100 bg-white/90 p-6 text-center dark:border-white/10 dark:bg-slate-950/45">
            <p class="text-xs font-black uppercase tracking-[0.24em] text-rose-700 dark:text-rose-200">
              Processing
            </p>
            <div class="mt-5 flex justify-center">
              <p-progress-spinner strokeWidth="4" ariaLabel="loading" />
            </div>
            @if (error()) {
              <div class="callback-page__error mt-6">
                <p>{{ error() }}</p>
              </div>
            } @else {
              <div class="callback-page__success mt-6">
                <p>Finalizing the OIDC session and returning you to the application.</p>
              </div>
            }
          </div>
        </div>
      </section>
    </main>
  `,
  styles: `
    .callback-page__error {
      color: var(--p-red-600);
    }

    .callback-page__success {
      color: var(--p-text-muted-color);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CallbackPageComponent {
  private readonly auth = inject(AuthService);
  private readonly authz = inject(AuthzService);
  private readonly router = inject(Router);

  protected readonly error = signal<string | null>(null);

  constructor() {
    void this.finish();
  }

  private async finish(): Promise<void> {
    try {
      await this.auth.handleCallback(new URLSearchParams(window.location.search));
      await this.authz.reload();
      await this.router.navigateByUrl(this.auth.consumeReturnUrl());
    } catch (error) {
      this.error.set(error instanceof Error ? error.message : 'Authentication failed');
    }
  }
}
