import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatCardModule } from '@angular/material/card';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-callback-page',
  standalone: true,
  imports: [TranslocoPipe, MatCardModule, MatProgressSpinnerModule],
  template: `
    <div class="callback-page">
      <mat-card appearance="outlined" class="callback-page__card">
        <mat-progress-spinner diameter="44" mode="indeterminate" />
        <h1>{{ 'auth.redirecting' | transloco }}</h1>
        @if (error()) {
          <p>{{ error() }}</p>
        } @else {
          <p>{{ 'auth.callbackMessage' | transloco }}</p>
        }
      </mat-card>
    </div>
  `,
  styles: `
    .callback-page {
      min-height: 100dvh;
      display: grid;
      place-items: center;
      padding: 1.5rem;
      background: linear-gradient(180deg, var(--surface-page) 0%, var(--surface-page-alt) 100%);
    }

    .callback-page__card {
      width: min(100%, 28rem);
      display: grid;
      gap: 1rem;
      place-items: center;
      text-align: center;
      border-radius: 1.5rem;
      padding: 2rem;
      background: var(--surface-card);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CallbackPageComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected readonly error = signal<string | null>(null);

  constructor() {
    void this.finish();
  }

  private async finish(): Promise<void> {
    try {
      await this.auth.handleCallback(new URLSearchParams(window.location.search));
      await this.router.navigateByUrl('/dashboard');
    } catch (error) {
      this.error.set(error instanceof Error ? error.message : 'Authentication failed');
    }
  }
}
