import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
import { MessageModule } from 'primeng/message';
import { ProgressSpinnerModule } from 'primeng/progressspinner';

import { AppApiService } from '../../core/api/app-api.service';
import { AuthConsentRequestResponse } from '../../core/api/api.types';

@Component({
  selector: 'app-consent-page',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    TranslocoPipe,
    ButtonModule,
    CardModule,
    CheckboxModule,
    MessageModule,
    ProgressSpinnerModule,
  ],
  template: `
    <main class="grid min-h-dvh place-items-center bg-surface-50 px-4 py-8 text-color dark:bg-surface-950">
      <section class="w-full max-w-[42rem]">
        <p-card styleClass="overflow-hidden rounded-[1.75rem] border border-surface bg-surface-0 shadow-xl dark:bg-surface-900">
          <ng-template pTemplate="header">
            <div class="bg-primary px-7 py-8 text-primary-contrast">
              <p class="text-sm font-semibold uppercase tracking-[0.24em] opacity-75">
                {{ 'auth.consent.eyebrow' | transloco }}
              </p>
              <h1 class="mt-3 text-3xl font-black tracking-tight">{{ 'auth.consent.title' | transloco }}</h1>
              <p class="mt-3 max-w-[34rem] text-sm leading-6 text-white/80">{{ 'auth.consent.subtitle' | transloco }}</p>
            </div>
          </ng-template>

          <div class="grid gap-5 p-2">
            @if (loading()) {
              <div class="grid place-items-center gap-3 py-8">
                <p-progress-spinner strokeWidth="4" ariaLabel="loading" />
                <p class="text-sm text-surface-500">{{ 'auth.consent.loading' | transloco }}</p>
              </div>
            } @else if (error()) {
              <p-message severity="error" [text]="error()! | transloco" />
              <a routerLink="/" class="inline-flex">
                <p-button [label]="'common.open' | transloco" icon="pi pi-home" />
              </a>
            } @else if (request()) {
              <div class="rounded-2xl border border-surface bg-highlight p-4 text-sm leading-6 text-highlight-color">
                <strong>{{ request()!.client_name }}</strong> {{ 'auth.consent.clientBody' | transloco }}
              </div>

              <div class="grid gap-3">
                @for (scope of request()!.scopes; track scope.code) {
                  <label class="flex items-start gap-3 rounded-2xl border border-surface-200 p-4 dark:border-surface-800">
                    <p-checkbox
                      [binary]="true"
                      [ngModel]="selected().has(scope.code)"
                      [disabled]="scope.required"
                      (ngModelChange)="toggleScope(scope.code, $event)"
                    />
                    <span class="grid gap-1">
                      <span class="font-semibold">{{ scope.label }}</span>
                      <span class="text-xs uppercase tracking-[0.18em] text-surface-500">{{ scope.code }}</span>
                    </span>
                  </label>
                }
              </div>

              <p class="rounded-2xl bg-surface-100 p-4 text-sm leading-6 text-surface-600 dark:bg-surface-900 dark:text-surface-300">
                {{ 'auth.consent.gdpr' | transloco }}
              </p>

              <div class="grid gap-3 sm:grid-cols-2">
                <p-button
                  severity="secondary"
                  styleClass="w-full justify-center"
                  icon="pi pi-times"
                  [label]="'auth.consent.deny' | transloco"
                  [loading]="submitting()"
                  (onClick)="decide('deny')"
                />
                <p-button
                  styleClass="w-full justify-center"
                  icon="pi pi-check"
                  [label]="'auth.consent.allow' | transloco"
                  [loading]="submitting()"
                  (onClick)="decide('allow')"
                />
              </div>
            }
          </div>
        </p-card>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConsentPageComponent {
  private readonly api = inject(AppApiService);
  private readonly route = inject(ActivatedRoute);

  readonly loading = signal(true);
  readonly submitting = signal(false);
  readonly error = signal<string | null>(null);
  readonly request = signal<AuthConsentRequestResponse | null>(null);
  readonly selected = signal(new Set<string>());

  constructor() {
    this.load();
  }

  toggleScope(code: string, checked: boolean): void {
    const next = new Set(this.selected());
    if (checked) {
      next.add(code);
    } else {
      next.delete(code);
    }
    this.selected.set(next);
  }

  decide(decision: 'allow' | 'deny'): void {
    const request = this.request();
    if (!request) {
      return;
    }
    const grantedScopes = decision === 'allow' ? Array.from(this.selected()) : [];
    if (decision === 'allow' && request.scopes.some((scope) => scope.required && !this.selected().has(scope.code))) {
      this.error.set('auth.consent.messages.selectScope');
      return;
    }

    this.submitting.set(true);
    this.error.set(null);
    this.api.decideConsent({ request_id: request.request_id, decision, granted_scopes: grantedScopes }).subscribe({
      next: (response) => window.location.assign(response.redirect_to),
      error: () => {
        this.submitting.set(false);
        this.error.set('auth.consent.messages.saveFailed');
      },
    });
  }

  private load(): void {
    const requestId = this.route.snapshot.queryParamMap.get('request');
    if (!requestId) {
      this.loading.set(false);
      this.error.set('auth.consent.unavailable');
      return;
    }

    this.api.consentRequest(requestId).subscribe({
      next: (request) => {
        this.request.set(request);
        this.selected.set(new Set(request.scopes.map((scope) => scope.code)));
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('auth.consent.unavailable');
      },
    });
  }
}
