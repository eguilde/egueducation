import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

import { AuthService } from '../../core/auth/auth.service';
import { AuthShellComponent } from '../../shared/auth-shell/auth-shell.component';

@Component({
  selector: 'app-callback-page',
  standalone: true,
  imports: [TranslocoPipe, AuthShellComponent, MatIconModule, MatProgressSpinnerModule],
  template: `
    <app-auth-shell
      [eyebrow]="'auth.callbackEyebrow' | transloco"
      [title]="'auth.redirecting' | transloco"
      [subtitle]="'auth.callbackMessage' | transloco"
      [visualKicker]="'auth.visual.kicker' | transloco"
      [visualTitle]="'auth.visual.title' | transloco"
      [visualBody]="'auth.visual.body' | transloco">
      <div class="callback-page__card">
        <mat-progress-spinner diameter="44" mode="indeterminate" />
        @if (error()) {
          <div class="callback-page__error">
            <mat-icon>error</mat-icon>
            <p>{{ error() }}</p>
          </div>
        } @else {
          <p>{{ 'auth.callbackMessage' | transloco }}</p>
        }
      </div>
    </app-auth-shell>
  `,
  styles: `
    .callback-page__card {
      display: grid;
      gap: 1rem;
      place-items: center;
      text-align: center;
      padding: 1.5rem;
      border-radius: 1.5rem;
      border: 1px solid rgb(148 163 184 / 0.18);
      background: rgb(255 255 255 / 0.58);
    }

    .callback-page__card p {
      margin: 0;
      color: var(--text-soft);
    }

    .callback-page__error {
      display: grid;
      gap: 0.5rem;
      justify-items: center;
      color: #be123c;
    }

    :host-context(.dark-theme) .callback-page__card {
      background: rgb(15 23 42 / 0.58);
      border-color: rgb(148 163 184 / 0.18);
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
