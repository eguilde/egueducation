import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';

@Component({
  selector: 'app-auth-status-page',
  imports: [RouterLink, TranslocoPipe, ButtonModule],
  template: `
    <main class="grid min-h-dvh place-items-center bg-surface-50 p-6 dark:bg-surface-950">
      <section class="w-full max-w-lg rounded-[2rem] border border-surface-200 bg-white p-8 text-center shadow-xl dark:border-surface-800 dark:bg-surface-900">
        <p class="text-sm font-black uppercase tracking-[0.2em] text-primary">{{ 'auth.redirectBadge' | transloco }}</p>
        <h1 class="mt-3 text-3xl font-black">{{ 'auth.loggedOutTitle' | transloco }}</h1>
        <p class="mt-3 text-surface-600 dark:text-surface-300">{{ 'auth.loggedOutBody' | transloco }}</p>
        <a routerLink="/" class="mt-6 inline-flex">
          <p-button [label]="'common.open' | transloco" icon="pi pi-home" />
        </a>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthStatusPageComponent {}
