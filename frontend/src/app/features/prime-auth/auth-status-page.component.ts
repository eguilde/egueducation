import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';

@Component({
  selector: 'app-auth-status-page',
  imports: [RouterLink, TranslocoPipe, ButtonModule],
  template: `
    <main class="min-h-dvh overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.18),_transparent_30rem),linear-gradient(135deg,_#fff7f8_0%,_#ffffff_50%,_#ffe4e9_100%)] text-slate-950 dark:bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.28),_transparent_30rem),linear-gradient(135deg,_#0f172a_0%,_#020617_56%,_#260711_100%)] dark:text-white">
      <section class="mx-auto grid min-h-dvh w-full max-w-6xl items-center gap-8 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10">
        <div class="space-y-7">
          <div class="inline-flex items-center rounded-full border border-rose-200 bg-white/75 px-4 py-2 text-sm font-semibold text-rose-700 shadow-sm backdrop-blur dark:border-rose-400/30 dark:bg-white/10 dark:text-rose-100">
            {{ 'auth.redirectBadge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-3xl text-4xl font-black tracking-[-0.045em] text-slate-950 md:text-6xl dark:text-white">
              {{ 'auth.loggedOutTitle' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-slate-700 dark:text-slate-200">
              {{ 'auth.loggedOutBody' | transloco }}
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
          <div class="rounded-[1.75rem] border border-rose-100 bg-white/90 p-6 text-center dark:border-white/10 dark:bg-slate-950/45">
            <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">
              <i class="pi pi-check text-2xl"></i>
            </div>
            <h2 class="mt-4 text-2xl font-black tracking-[-0.035em] text-slate-950 dark:text-white">
              {{ 'auth.loggedOutTitle' | transloco }}
            </h2>
            <p class="mt-3 text-sm leading-7 text-slate-600 dark:text-slate-300">
              {{ 'auth.loggedOutBody' | transloco }}
            </p>

            <div class="mt-6 grid gap-3">
              <a routerLink="/" class="inline-flex">
                <p-button [label]="'common.open' | transloco" icon="pi pi-home" styleClass="w-full justify-center" />
              </a>
              <p class="text-xs leading-6 text-slate-500 dark:text-slate-400">
                You can safely close this tab after returning to the application.
              </p>
            </div>
          </div>
        </aside>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthStatusPageComponent {
  protected readonly highlights = [
    {
      kicker: 'Session',
      title: 'Logout complete',
      body: 'The backend provider has cleared the active login session.',
    },
    {
      kicker: 'OIDC',
      title: 'Provider-owned flow',
      body: 'Logout is handled by the identity provider, not the SPA.',
    },
    {
      kicker: 'Return',
      title: 'Back to app',
      body: 'Use the primary button to return safely to the originating app.',
    },
  ] as const;
}
