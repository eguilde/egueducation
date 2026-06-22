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
    <main class="grid min-h-dvh place-items-center bg-surface-50 p-6 dark:bg-surface-950">
      <div class="callback-page__card rounded-[1.5rem] border border-surface bg-surface-0 p-6 shadow-sm dark:bg-surface-900">
        <p-progress-spinner strokeWidth="4" ariaLabel="loading" />
        @if (error()) {
          <div class="callback-page__error">
            <p>{{ error() }}</p>
          </div>
        } @else {
          <p>{{ 'auth.callbackMessage' | transloco }}</p>
        }
      </div>
    </main>
  `,
  styles: `
    .callback-page__card {
      display: grid;
      gap: 1rem;
      place-items: center;
      text-align: center;
    }

    .callback-page__card p {
      margin: 0;
      color: var(--p-text-muted-color);
    }

    .callback-page__error {
      display: grid;
      gap: 0.5rem;
      justify-items: center;
      color: var(--p-red-600);
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
