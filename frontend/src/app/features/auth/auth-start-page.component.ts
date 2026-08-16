import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { ActivatedRoute } from '@angular/router';
import { MessageModule } from 'primeng/message';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-auth-start-page',
  standalone: true,
  imports: [TranslocoPipe, MessageModule],
  template: `
    <main class="min-h-dvh overflow-hidden bg-surface-50 text-color dark:bg-surface-950">
      <section
        class="mx-auto grid min-h-dvh w-full max-w-7xl items-center gap-8 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10"
      >
        <div class="space-y-7">
          <div
            class="inline-flex items-center rounded-full border border-primary bg-surface-0 px-4 py-2 text-sm font-semibold text-primary shadow-sm dark:bg-surface-900"
          >
            {{ 'auth.redirectBadge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-4xl text-4xl font-black tracking-[-0.045em] text-color md:text-6xl">
              {{ 'auth.redirecting' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-muted-color">
              {{ 'auth.callbackMessage' | transloco }}
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
          <div class="rounded-[1.75rem] border border-surface bg-surface-0 p-6 dark:bg-surface-950">
            <p class="text-xs font-black uppercase tracking-[0.24em] text-primary">
              {{ 'auth.redirectBadge' | transloco }}
            </p>
            <h2 class="mt-3 text-2xl font-black tracking-[-0.035em] text-color">
              {{ 'auth.redirecting' | transloco }}
            </h2>
            <p class="mt-3 text-sm leading-7 text-muted-color">
              {{ 'auth.callbackMessage' | transloco }}
            </p>

            <div class="mt-6 grid gap-3">
              <div
                class="flex items-center gap-3 rounded-2xl border border-surface bg-surface-0 px-4 py-3 dark:bg-surface-950"
              >
                <span
                  class="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-200"
                  >1</span
                >
                <div>
                  <p class="text-sm font-bold text-color">OIDC handshake</p>
                  <p class="text-xs leading-5 text-muted-color">
                    Cererea este transferată către providerul de identitate.
                  </p>
                </div>
              </div>
              <div
                class="flex items-center gap-3 rounded-2xl border border-surface bg-surface-0 px-4 py-3 dark:bg-surface-950"
              >
                <span
                  class="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-200"
                  >2</span
                >
                <div>
                  <p class="text-sm font-bold text-color">Secure redirect</p>
                  <p class="text-xs leading-5 text-muted-color">
                    Vei reveni la exact același return URL după autentificare.
                  </p>
                </div>
              </div>
            </div>

            @if (error()) {
              <p-message severity="error" styleClass="mt-5 w-full">{{ error() }}</p-message>
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
      this.error.set(
        error instanceof Error ? error.message : 'Authentication redirect could not be started.',
      );
    }
  }
}
