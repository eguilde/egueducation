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
    <main class="app-auth-shell">
      <section class="mx-auto grid min-h-dvh w-full max-w-5xl items-center gap-8 px-5 py-8 md:grid-cols-[0.9fr_1.1fr] md:px-8 lg:px-10">
        <div class="space-y-6">
          <div class="app-auth-badge px-4 py-2 text-sm font-semibold">
            {{ 'auth.redirectBadge' | transloco }}
          </div>
          <h1 class="max-w-2xl text-4xl font-black tracking-[-0.045em] text-color md:text-6xl">
            {{ 'auth.redirecting' | transloco }}
          </h1>
          <p class="max-w-xl text-lg leading-8 text-muted-color">
            {{ 'auth.callbackMessage' | transloco }}
          </p>
        </div>

        <div class="app-auth-panel rounded-[2rem] p-6">
          <div class="app-auth-panel-inner rounded-[1.75rem] p-6 text-center">
            <p class="app-auth-accent text-xs font-black uppercase tracking-[0.24em]">
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
      await this.authz.bootstrapAuthenticated();
      await this.router.navigateByUrl(this.auth.consumeReturnUrl());
    } catch (error) {
      this.error.set(error instanceof Error ? error.message : 'Authentication failed');
    }
  }
}
