import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { MessageModule } from 'primeng/message';

import { AuthService } from '../../core/auth/auth.service';
import { AuthzService } from '../../core/authz/authz.service';

@Component({
  selector: 'app-callback-page',
  standalone: true,
  imports: [TranslocoPipe, ProgressSpinnerModule, MessageModule],
  template: `
    <main class="min-h-dvh overflow-hidden bg-surface-50 text-color dark:bg-surface-950">
      <section
        class="mx-auto grid min-h-dvh w-full max-w-5xl items-center gap-8 px-5 py-8 md:grid-cols-[0.9fr_1.1fr] md:px-8 lg:px-10"
      >
        <div class="space-y-6">
          <div
            class="inline-flex items-center rounded-full border border-primary bg-surface-0 px-4 py-2 text-sm font-semibold text-primary shadow-sm dark:bg-surface-900"
          >
            {{ 'auth.redirectBadge' | transloco }}
          </div>
          <h1 class="max-w-2xl text-4xl font-black tracking-[-0.045em] text-color md:text-6xl">
            {{ 'auth.redirecting' | transloco }}
          </h1>
          <p class="max-w-xl text-lg leading-8 text-muted-color">
            {{ 'auth.callbackMessage' | transloco }}
          </p>
        </div>

        <div
          class="rounded-[2rem] border border-surface bg-surface-0 p-6 shadow-lg dark:bg-surface-900"
        >
          <div
            class="rounded-[1.75rem] border border-surface bg-surface-0 p-6 text-center dark:bg-surface-950"
          >
            <p class="text-xs font-black uppercase tracking-[0.24em] text-primary">Processing</p>
            <div class="mt-5 flex justify-center">
              <p-progress-spinner strokeWidth="4" ariaLabel="loading" />
            </div>
            @if (error()) {
              <p-message severity="error" styleClass="mt-6 w-full">{{ error() }}</p-message>
            } @else {
              <p-message severity="info" styleClass="mt-6 w-full">
                Finalizing the OIDC session and returning you to the application.
              </p-message>
            }
          </div>
        </div>
      </section>
    </main>
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
