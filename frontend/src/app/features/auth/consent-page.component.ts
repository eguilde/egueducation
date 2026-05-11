import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AppApiService } from '../../core/api/app-api.service';
import { AuthConsentRequestResponse } from '../../core/api/api.types';
import { AuthShellComponent } from '../../shared/auth-shell/auth-shell.component';

@Component({
  selector: 'app-consent-page',
  imports: [
    TranslocoPipe,
    RouterLink,
    AuthShellComponent,
    MatButtonModule,
    MatCheckboxModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
  ],
  templateUrl: './consent-page.component.html',
  styleUrl: './consent-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConsentPageComponent {
  private readonly api = inject(AppApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly snackBar = inject(MatSnackBar);
  private readonly transloco = inject(TranslocoService);

  protected readonly consentRequest = signal<AuthConsentRequestResponse | null>(null);
  protected readonly loading = signal(true);
  protected readonly submitting = signal(false);
  protected readonly error = signal<string | null>(null);
  protected readonly selectedScopes = signal<string[]>([]);
  protected readonly allSelected = computed(() => {
    const request = this.consentRequest();
    if (!request || request.scopes.length === 0) {
      return true;
    }
    return request.scopes.every((scope) => this.selectedScopes().includes(scope.code));
  });

  constructor() {
    void this.load();
  }

  protected toggleAll(checked: boolean): void {
    const request = this.consentRequest();
    if (!request) {
      return;
    }
    this.selectedScopes.set(checked ? request.scopes.map((scope) => scope.code) : []);
  }

  protected toggleScope(code: string, checked: boolean): void {
    this.selectedScopes.update((current) => {
      if (checked) {
        return current.includes(code) ? current : [...current, code];
      }
      return current.filter((scopeCode) => scopeCode !== code);
    });
  }

  protected approve(): void {
    void this.submit('allow');
  }

  protected deny(): void {
    void this.submit('deny');
  }

  private async load(): Promise<void> {
    const requestId = this.route.snapshot.queryParamMap.get('request');
    if (!requestId) {
      this.loading.set(false);
      this.error.set('Missing consent request');
      return;
    }

    this.api.consentRequest(requestId).subscribe({
      next: (response) => {
        this.consentRequest.set(response);
        this.selectedScopes.set(response.scopes.map((scope) => scope.code));
        this.loading.set(false);
      },
      error: () => {
        this.error.set('Consent request is no longer available');
        this.loading.set(false);
      },
    });
  }

  private async submit(decision: 'allow' | 'deny'): Promise<void> {
    const request = this.consentRequest();
    if (!request) {
      return;
    }
    if (decision === 'allow' && request.scopes.length > 0 && this.selectedScopes().length === 0) {
      this.snackBar.open(
        this.transloco.translate('auth.consent.messages.selectScope'),
        this.transloco.translate('common.close'),
        { duration: 3500 },
      );
      return;
    }

    this.submitting.set(true);
    this.api
      .decideConsent({
        request_id: request.request_id,
        decision,
        granted_scopes: this.selectedScopes(),
      })
      .subscribe({
        next: (response) => {
          window.location.href = response.redirect_to;
        },
        error: () => {
          this.submitting.set(false);
          this.snackBar.open(
            this.transloco.translate('auth.consent.messages.saveFailed'),
            this.transloco.translate('common.close'),
            { duration: 4000 },
          );
        },
      });
  }
}
