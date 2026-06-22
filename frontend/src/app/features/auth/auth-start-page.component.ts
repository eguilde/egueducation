import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { ActivatedRoute } from '@angular/router';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-auth-start-page',
  standalone: true,
  imports: [TranslocoPipe],
  template: `
    <main class="min-h-dvh overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.18),_transparent_30rem),linear-gradient(135deg,_#fff7f8_0%,_#ffffff_50%,_#ffe4e9_100%)] text-slate-950 dark:bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.28),_transparent_30rem),linear-gradient(135deg,_#0f172a_0%,_#020617_56%,_#260711_100%)] dark:text-white">
      <section class="mx-auto grid min-h-dvh w-full max-w-7xl items-center gap-8 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10">
        <div class="space-y-7">
          <div class="inline-flex items-center rounded-full border border-rose-200 bg-white/75 px-4 py-2 text-sm font-semibold text-rose-700 shadow-sm backdrop-blur dark:border-rose-400/30 dark:bg-white/10 dark:text-rose-100">
            {{ 'auth.redirectBadge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-4xl text-4xl font-black tracking-[-0.045em] text-slate-950 md:text-6xl dark:text-white">
              {{ 'auth.redirecting' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-slate-700 dark:text-slate-200">
              {{ 'auth.callbackMessage' | transloco }}
            </p>
          </div>

          <div class="grid gap-3 sm:grid-cols-3">
            @for (item of highlights; track item.title) {
              <article class="rounded-[1.35rem] border border-white/70 bg-white/75 p-4 shadow-lg shadow-rose-950/5 backdrop-blur dark:border-white/10 dark:bg-white/10">
                <p class="text-xs font-black uppercase tracking-[0.18em] text-rose-700 dark:text-rose-200">
                  {{ item.kicker }}
                </p>
                <h2 class="mt-2 text-sm font-black text-slate-950 dark:text-white">
                  {{ item.title }}
                </h2>
                <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
                  {{ item.body }}
                </p>
              </article>
            }
          </div>
        </div>

        <aside class="rounded-[2rem] border border-white/70 bg-white/72 p-5 shadow-2xl shadow-rose-950/10 backdrop-blur-xl dark:border-white/10 dark:bg-white/10">
          <div class="rounded-[1.75rem] border border-rose-100 bg-white/90 p-6 dark:border-white/10 dark:bg-slate-950/45">
            <p class="text-xs font-black uppercase tracking-[0.24em] text-rose-700 dark:text-rose-200">
              {{ 'auth.redirectBadge' | transloco }}
            </p>
            <h2 class="mt-3 text-2xl font-black tracking-[-0.035em] text-slate-950 dark:text-white">
              {{ 'auth.redirecting' | transloco }}
            </h2>
            <p class="mt-3 text-sm leading-7 text-slate-600 dark:text-slate-300">
              {{ 'auth.callbackMessage' | transloco }}
            </p>

            <div class="mt-6 grid gap-3">
              <div class="flex items-center gap-3 rounded-2xl border border-rose-100 bg-rose-50/80 px-4 py-3 dark:border-white/10 dark:bg-white/5">
                <span class="flex h-10 w-10 items-center justify-center rounded-2xl bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">1</span>
                <div>
                  <p class="text-sm font-bold text-slate-950 dark:text-white">OIDC handshake</p>
                  <p class="text-xs leading-5 text-slate-600 dark:text-slate-300">Cererea este transferată către providerul de identitate.</p>
                </div>
              </div>
              <div class="flex items-center gap-3 rounded-2xl border border-rose-100 bg-rose-50/80 px-4 py-3 dark:border-white/10 dark:bg-white/5">
                <span class="flex h-10 w-10 items-center justify-center rounded-2xl bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">2</span>
                <div>
                  <p class="text-sm font-bold text-slate-950 dark:text-white">Secure redirect</p>
                  <p class="text-xs leading-5 text-slate-600 dark:text-slate-300">Vei reveni la exact același return URL după autentificare.</p>
                </div>
              </div>
            </div>

            @if (error()) {
              <p class="mt-5 rounded-2xl bg-rose-100 px-4 py-3 text-sm font-semibold text-rose-800 dark:bg-rose-950/50 dark:text-rose-100">
                {{ error() }}
              </p>
            }
          </div>
        </aside>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthStartPageComponent {
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  protected readonly error = signal<string | null>(null);
  protected readonly highlights = [
    {
      kicker: 'Redirect',
      title: 'Continuing sign-in',
      body: 'The authentication request is being handed off to the OIDC provider.',
    },
    {
      kicker: 'Flow',
      title: 'OIDC provider',
      body: 'Login, consent and logout are all handled by the backend interaction pages.',
    },
    {
      kicker: 'Responsive',
      title: 'Mobile friendly',
      body: 'The layout adapts cleanly on phones, tablets and desktops.',
    },
  ] as const;

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
