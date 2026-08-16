import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';

@Component({
  selector: 'app-auth-status-page',
  imports: [RouterLink, TranslocoPipe, ButtonModule],
  template: `
    <main class="min-h-dvh overflow-hidden bg-surface-50 text-color dark:bg-surface-950">
      <section
        class="mx-auto grid min-h-dvh w-full max-w-6xl items-center gap-8 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10"
      >
        <div class="space-y-7">
          <div
            class="inline-flex items-center rounded-full border border-primary bg-surface-0 px-4 py-2 text-sm font-semibold text-primary shadow-sm dark:bg-surface-900"
          >
            {{ 'auth.redirectBadge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-3xl text-4xl font-black tracking-[-0.045em] text-color md:text-6xl">
              {{ 'auth.loggedOutTitle' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-muted-color">
              {{ 'auth.loggedOutBody' | transloco }}
            </p>
          </div>

          <div class="grid gap-3 sm:grid-cols-3">
            @for (item of highlights; track item.title) {
              <article
                class="rounded-[1.35rem] border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900"
              >
                <p class="text-xs font-black uppercase tracking-[0.18em] text-primary">
                  {{ item.kicker }}
                </p>
                <h2 class="mt-2 text-sm font-black text-color">
                  {{ item.title }}
                </h2>
                <p class="mt-2 text-sm leading-6 text-muted-color">
                  {{ item.body }}
                </p>
              </article>
            }
          </div>
        </div>

        <aside
          class="rounded-[2rem] border border-surface bg-surface-0 p-5 shadow-lg dark:bg-surface-900"
        >
          <div
            class="rounded-[1.75rem] border border-surface bg-surface-0 p-6 text-center dark:bg-surface-950"
          >
            <div
              class="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-200"
            >
              <i class="pi pi-check text-2xl"></i>
            </div>
            <h2 class="mt-4 text-2xl font-black tracking-[-0.035em] text-color">
              {{ 'auth.loggedOutTitle' | transloco }}
            </h2>
            <p class="mt-3 text-sm leading-7 text-muted-color">
              {{ 'auth.loggedOutBody' | transloco }}
            </p>

            <div class="mt-6 grid gap-3">
              <a routerLink="/" class="inline-flex">
                <p-button
                  [label]="'common.open' | transloco"
                  icon="pi pi-home"
                  styleClass="w-full justify-center"
                />
              </a>
              <p class="text-xs leading-6 text-muted-color">
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
