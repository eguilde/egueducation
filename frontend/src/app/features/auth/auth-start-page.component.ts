import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { TranslocoPipe } from '@jsverse/transloco';
import { ActivatedRoute } from '@angular/router';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-auth-start-page',
  standalone: true,
  imports: [TranslocoPipe],
  template: `
    <main class="app-auth-shell">
      <section class="mx-auto grid min-h-dvh w-full max-w-7xl items-center gap-8 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10">
        <div class="space-y-7">
          <div class="app-auth-badge px-4 py-2 text-sm font-semibold">
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
              <article class="app-auth-glass-card rounded-[1.35rem] p-4">
                <p class="app-auth-accent text-xs font-black uppercase tracking-[0.18em]">
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

        <aside class="app-auth-panel rounded-[2rem] p-5">
          <div class="app-auth-panel-inner rounded-[1.75rem] p-6">
            <p class="app-auth-accent text-xs font-black uppercase tracking-[0.24em]">
              {{ 'auth.redirectBadge' | transloco }}
            </p>
            <h2 class="mt-3 text-2xl font-black tracking-[-0.035em] text-color">
              {{ 'auth.redirecting' | transloco }}
            </h2>
            <p class="mt-3 text-sm leading-7 text-muted-color">
              {{ 'auth.callbackMessage' | transloco }}
            </p>

            <div class="mt-6 grid gap-3">
              <div class="app-auth-panel-inner flex items-center gap-3 rounded-2xl px-4 py-3">
                <span class="app-auth-accent-soft flex h-10 w-10 items-center justify-center rounded-2xl">1</span>
                <div>
                  <p class="text-sm font-bold text-color">OIDC handshake</p>
                  <p class="text-xs leading-5 text-muted-color">Cererea este transferată către providerul de identitate.</p>
                </div>
              </div>
              <div class="app-auth-panel-inner flex items-center gap-3 rounded-2xl px-4 py-3">
                <span class="app-auth-accent-soft flex h-10 w-10 items-center justify-center rounded-2xl">2</span>
                <div>
                  <p class="text-sm font-bold text-color">Secure redirect</p>
                  <p class="text-xs leading-5 text-muted-color">Vei reveni la exact același return URL după autentificare.</p>
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
