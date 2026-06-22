import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { ThemeService } from '../../core/ui/theme.service';

@Component({
  selector: 'app-home-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe],
  template: `
    <main class="min-h-dvh overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.22),_transparent_34rem),linear-gradient(135deg,_#fff7f8_0%,_#fff_46%,_#ffe4e9_100%)] text-slate-950 dark:bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.3),_transparent_32rem),linear-gradient(135deg,_#19040a_0%,_#0f172a_58%,_#260711_100%)] dark:text-white">
      <section class="mx-auto grid min-h-dvh w-full max-w-7xl items-center gap-10 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10">
        <div class="space-y-8">
          <div class="inline-flex items-center rounded-full border border-rose-200 bg-white/75 px-4 py-2 text-sm font-semibold text-rose-700 shadow-sm backdrop-blur dark:border-rose-400/30 dark:bg-white/10 dark:text-rose-100">
            {{ 'home.badge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-4xl text-4xl font-black tracking-[-0.045em] text-slate-950 md:text-6xl dark:text-white">
              {{ 'home.title' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-slate-700 dark:text-slate-200">
              {{ 'home.subtitle' | transloco }}
            </p>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row">
            <a
              routerLink="/auth/start"
              class="inline-flex items-center justify-center rounded-2xl bg-rose-700 px-6 py-3 text-base font-bold text-white shadow-xl shadow-rose-900/20 transition hover:bg-rose-800 focus:outline-none focus:ring-4 focus:ring-rose-300"
            >
              {{ 'auth.login' | transloco }}
            </a>
            <a
              routerLink="/auth/register"
              class="inline-flex items-center justify-center rounded-2xl border border-rose-200 bg-white/70 px-6 py-3 text-base font-bold text-rose-800 shadow-sm backdrop-blur transition hover:bg-white focus:outline-none focus:ring-4 focus:ring-rose-200 dark:border-white/15 dark:bg-white/10 dark:text-white dark:hover:bg-white/15">
              {{ 'home.register' | transloco }}
            </a>
          </div>

          <div class="flex flex-wrap gap-2">
            @for (item of trustItems; track item) {
              <span class="rounded-full border border-slate-200 bg-white/60 px-3 py-1 text-sm font-medium text-slate-700 dark:border-white/15 dark:bg-white/10 dark:text-slate-100">
                {{ item | transloco }}
              </span>
            }
          </div>
        </div>

        <aside class="rounded-[2rem] border border-white/70 bg-white/72 p-5 shadow-2xl shadow-rose-950/10 backdrop-blur-xl dark:border-white/10 dark:bg-white/10">
          <div class="grid gap-4">
            @for (card of cards; track card.title) {
              <article class="rounded-[1.5rem] border border-rose-100 bg-white/82 p-5 shadow-sm dark:border-white/10 dark:bg-slate-950/40">
                <p class="text-xs font-black uppercase tracking-[0.2em] text-rose-700 dark:text-rose-200">
                  {{ card.kicker | transloco }}
                </p>
                <h2 class="mt-3 text-xl font-black text-slate-950 dark:text-white">
                  {{ card.title | transloco }}
                </h2>
                <p class="mt-2 leading-7 text-slate-600 dark:text-slate-300">
                  {{ card.body | transloco }}
                </p>
              </article>
            }
          </div>
        </aside>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomePageComponent {
  protected readonly theme = inject(ThemeService);

  protected readonly trustItems = ['home.trust.oidc', 'home.trust.rbac', 'home.trust.gdpr'];
  protected readonly cards = [
    {
      kicker: 'home.cards.documents.kicker',
      title: 'home.cards.documents.title',
      body: 'home.cards.documents.body',
    },
    {
      kicker: 'home.cards.education.kicker',
      title: 'home.cards.education.title',
      body: 'home.cards.education.body',
    },
    {
      kicker: 'home.cards.admin.kicker',
      title: 'home.cards.admin.title',
      body: 'home.cards.admin.body',
    },
  ];

}
