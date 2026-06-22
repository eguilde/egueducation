import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
import { MessageModule } from 'primeng/message';

type RegisterStep = 'account-type' | 'consent' | 'entry';
type AccountType = 'staff' | 'community';
type EntryRoute = '/auth/start';

@Component({
  selector: 'app-register-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    TranslocoPipe,
    ButtonModule,
    CardModule,
    CheckboxModule,
    MessageModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="min-h-dvh overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.18),_transparent_32rem),linear-gradient(135deg,_#fff7f8_0%,_#ffffff_46%,_#ffe4e9_100%)] text-slate-950 dark:bg-[radial-gradient(circle_at_top_left,_rgba(244,63,94,0.28),_transparent_30rem),linear-gradient(135deg,_#19040a_0%,_#0f172a_58%,_#260711_100%)] dark:text-white">
      <section class="mx-auto grid min-h-dvh w-full max-w-7xl items-center gap-8 px-5 py-8 md:grid-cols-[1.02fr_0.98fr] md:px-8 lg:px-10">
        <div class="space-y-8">
          <div class="inline-flex items-center rounded-full border border-rose-200 bg-white/75 px-4 py-2 text-sm font-semibold text-rose-700 shadow-sm backdrop-blur dark:border-rose-400/30 dark:bg-white/10 dark:text-rose-100">
            {{ 'auth.register.badge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-4xl text-4xl font-black tracking-[-0.045em] text-slate-950 md:text-6xl dark:text-white">
              {{ 'auth.register.title' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-slate-700 dark:text-slate-200">
              {{ 'auth.register.subtitle' | transloco }}
            </p>
          </div>

          <div class="grid gap-3">
            @for (item of journey; track item.title) {
              <article class="rounded-[1.5rem] border border-white/70 bg-white/72 p-5 shadow-lg shadow-rose-950/5 backdrop-blur dark:border-white/10 dark:bg-white/10">
                <div class="flex items-start gap-4">
                  <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">
                    <i [class]="item.icon"></i>
                  </span>
                  <div class="space-y-1">
                    <p class="text-xs font-black uppercase tracking-[0.2em] text-rose-700 dark:text-rose-200">
                      {{ item.kicker | transloco }}
                    </p>
                    <h2 class="text-lg font-black text-slate-950 dark:text-white">
                      {{ item.title | transloco }}
                    </h2>
                    <p class="text-sm leading-6 text-slate-600 dark:text-slate-300">
                      {{ item.body | transloco }}
                    </p>
                  </div>
                </div>
              </article>
            }
          </div>

          <div class="flex flex-col gap-3 sm:flex-row">
            <a
              routerLink="/auth/start"
              [queryParams]="{ returnUrl: returnUrl }"
              class="inline-flex items-center justify-center rounded-2xl border border-rose-200 bg-white/70 px-6 py-3 text-base font-bold text-rose-900 shadow-sm backdrop-blur transition hover:bg-white focus:outline-none focus:ring-4 focus:ring-rose-200 dark:border-white/10 dark:bg-white/10 dark:text-white dark:hover:bg-white/15"
            >
              {{ 'auth.register.backToLogin' | transloco }}
            </a>
            <a
              routerLink="/auth/start"
              [queryParams]="{ returnUrl: returnUrl }"
              class="inline-flex items-center justify-center rounded-2xl bg-rose-700 px-6 py-3 text-base font-bold text-white shadow-xl shadow-rose-900/20 transition hover:bg-rose-800 focus:outline-none focus:ring-4 focus:ring-rose-300"
            >
              {{ 'auth.register.oidcCta' | transloco }}
            </a>
          </div>
        </div>

        <aside class="rounded-[2rem] border border-white/70 bg-white/72 p-4 shadow-2xl shadow-rose-950/10 backdrop-blur-xl dark:border-white/10 dark:bg-white/10">
          <p-card styleClass="overflow-hidden rounded-[1.75rem] border border-surface bg-surface-0 shadow-xl dark:bg-surface-900">
            <ng-template pTemplate="header">
              <div class="bg-primary px-7 py-8 text-primary-contrast">
                <p class="text-sm font-semibold uppercase tracking-[0.24em] opacity-75">
                  {{ 'auth.register.cardBadge' | transloco }}
                </p>
                <h2 class="mt-3 text-3xl font-black tracking-tight">
                  {{ 'auth.register.cardTitle' | transloco }}
                </h2>
                <p class="mt-3 max-w-[24rem] text-sm leading-6 text-white/80">
                  {{ 'auth.register.cardSubtitle' | transloco }}
                </p>
              </div>
            </ng-template>

            <form class="grid gap-5 p-5 md:p-6" [formGroup]="consentForm">
              <div class="flex flex-wrap gap-2">
                @for (step of steps; track step.key) {
                  <span
                    class="rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em]"
                    [class.border-rose-200]="step.key === currentStep()"
                    [class.bg-rose-700]="step.key === currentStep()"
                    [class.text-white]="step.key === currentStep()"
                    [class.border-surface]="step.key !== currentStep()"
                    [class.bg-surface-50]="step.key !== currentStep()"
                    [class.text-surface-600]="step.key !== currentStep()"
                  >
                    {{ step.label | transloco }}
                  </span>
                }
              </div>

              @if (currentStep() === 'account-type') {
                <div class="grid gap-4 sm:grid-cols-2">
                  @for (item of accountTypes; track item.code) {
                    <button
                      type="button"
                      class="group flex h-full flex-col rounded-[1.5rem] border border-rose-100 bg-white/85 p-5 text-left shadow-sm transition hover:-translate-y-0.5 hover:border-rose-200 hover:shadow-lg focus:outline-none focus:ring-4 focus:ring-rose-200 dark:border-white/10 dark:bg-slate-950/40"
                      (click)="selectAccountType(item.code)"
                    >
                      <div class="flex items-start gap-4">
                        <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">
                          <i [class]="item.icon"></i>
                        </span>
                        <div class="space-y-1">
                          <p class="text-xs font-black uppercase tracking-[0.2em] text-rose-700 dark:text-rose-200">
                            {{ item.kicker | transloco }}
                          </p>
                          <h3 class="text-lg font-black text-slate-950 dark:text-white">
                            {{ item.title | transloco }}
                          </h3>
                          <p class="text-sm leading-6 text-slate-600 dark:text-slate-300">
                            {{ item.body | transloco }}
                          </p>
                        </div>
                      </div>
                    </button>
                  }
                </div>

                <p-message severity="info" [text]="'auth.register.info' | transloco" />
              } @else if (currentStep() === 'consent') {
                <div class="rounded-[1.5rem] border border-surface bg-surface-50 p-4 dark:bg-white/5">
                  <p class="text-xs font-black uppercase tracking-[0.2em] text-rose-700 dark:text-rose-200">
                    {{ 'auth.register.selectedAccount' | transloco }}
                  </p>
                  <div class="mt-3 flex items-start gap-3">
                    <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">
                      <i [class]="selectedAccount()?.icon ?? ''"></i>
                    </span>
                    <div class="space-y-1">
                      <h3 class="text-lg font-black text-slate-950 dark:text-white">
                        {{ selectedAccount()?.title | transloco }}
                      </h3>
                      <p class="text-sm leading-6 text-slate-600 dark:text-slate-300">
                        {{ selectedAccount()?.body | transloco }}
                      </p>
                    </div>
                  </div>
                </div>

                <p-message severity="info" [text]="'auth.register.info' | transloco" />

                <div class="grid gap-3 rounded-[1.5rem] border border-surface bg-white/70 p-4 dark:bg-white/5">
                  <label class="flex items-start gap-3">
                    <p-checkbox formControlName="provisioningAcknowledged" inputId="register-provisioning" [binary]="true" />
                    <span class="text-sm leading-6 text-slate-700 dark:text-slate-200">
                      {{ 'auth.register.consent.items.provisioning' | transloco }}
                    </span>
                  </label>
                  <label class="flex items-start gap-3">
                    <p-checkbox formControlName="flowAcknowledged" inputId="register-flow" [binary]="true" />
                    <span class="text-sm leading-6 text-slate-700 dark:text-slate-200">
                      {{ 'auth.register.consent.items.flow' | transloco }}
                    </span>
                  </label>
                </div>

                <div class="grid gap-3 sm:grid-cols-2">
                  <p-button
                    type="button"
                    severity="secondary"
                    styleClass="w-full justify-center"
                    icon="pi pi-arrow-left"
                    [label]="'auth.register.backToAccountType' | transloco"
                    (onClick)="backToAccountType()"
                  />
                  <p-button
                    type="button"
                    styleClass="w-full justify-center"
                    icon="pi pi-arrow-right"
                    [label]="'auth.register.consentCta' | transloco"
                    [disabled]="consentForm.invalid"
                    (onClick)="continueToEntry()"
                  />
                </div>
              } @else {
                <div class="rounded-[1.5rem] border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900 dark:border-amber-400/20 dark:bg-amber-500/10 dark:text-amber-100">
                  {{ 'auth.register.entry.info' | transloco }}
                </div>

                <div class="grid gap-4 sm:grid-cols-2">
                  @for (entry of entryPoints; track entry.code) {
                    <article class="rounded-[1.5rem] border border-rose-100 bg-white/85 p-5 shadow-sm dark:border-white/10 dark:bg-slate-950/40">
                      <div class="flex items-start gap-4">
                        <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-rose-700/10 text-rose-700 dark:bg-white/10 dark:text-rose-100">
                          <i [class]="entry.icon"></i>
                        </span>
                        <div class="space-y-1">
                          <p class="text-xs font-black uppercase tracking-[0.2em] text-rose-700 dark:text-rose-200">
                            {{ entry.kicker | transloco }}
                          </p>
                          <h3 class="text-lg font-black text-slate-950 dark:text-white">
                            {{ entry.title | transloco }}
                          </h3>
                          <p class="text-sm leading-6 text-slate-600 dark:text-slate-300">
                            {{ entry.body | transloco }}
                          </p>
                        </div>
                      </div>

                      <p-button
                        type="button"
                        styleClass="mt-5 w-full justify-center"
                        [severity]="entry.code === 'classic' ? 'primary' : 'secondary'"
                        [icon]="entry.code === 'classic' ? 'pi pi-mobile' : 'pi pi-id-card'"
                        [label]="entry.cta | transloco"
                        (onClick)="goTo(entry.route)"
                      />
                    </article>
                  }
                </div>

                <div class="grid gap-3 sm:grid-cols-2">
                  <p-button
                    type="button"
                    severity="secondary"
                    styleClass="w-full justify-center"
                    icon="pi pi-arrow-left"
                    [label]="'auth.register.backToConsent' | transloco"
                    (onClick)="backToConsent()"
                  />
                  <a
                    routerLink="/auth/start"
                    [queryParams]="{ returnUrl: returnUrl }"
                    class="inline-flex items-center justify-center rounded-2xl border border-rose-200 bg-white/70 px-6 py-3 text-base font-bold text-rose-900 shadow-sm backdrop-blur transition hover:bg-white focus:outline-none focus:ring-4 focus:ring-rose-200 dark:border-white/10 dark:bg-white/10 dark:text-white dark:hover:bg-white/15"
                  >
                    {{ 'auth.register.backToLogin' | transloco }}
                  </a>
                </div>
              }
            </form>
          </p-card>
        </aside>
      </section>
    </main>
  `,
})
export class RegisterPageComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  readonly returnUrl = this.route.snapshot.queryParamMap.get('returnUrl') || '/dashboard';
  readonly currentStep = signal<RegisterStep>('account-type');
  readonly selectedAccountType = signal<AccountType | null>(null);

  readonly accountTypes = [
    {
      code: 'staff',
      icon: 'pi pi-building text-lg',
      kicker: 'auth.register.accountTypes.staff.kicker',
      title: 'auth.register.accountTypes.staff.title',
      body: 'auth.register.accountTypes.staff.body',
    },
    {
      code: 'community',
      icon: 'pi pi-users text-lg',
      kicker: 'auth.register.accountTypes.community.kicker',
      title: 'auth.register.accountTypes.community.title',
      body: 'auth.register.accountTypes.community.body',
    },
  ] as const;

  readonly selectedAccount = computed(
    () => this.accountTypes.find((item) => item.code === this.selectedAccountType()) ?? null,
  );

  readonly journey = [
    {
      icon: 'pi pi-id-card text-lg',
      kicker: 'auth.register.journey.account.kicker',
      title: 'auth.register.journey.account.title',
      body: 'auth.register.journey.account.body',
    },
    {
      icon: 'pi pi-check-circle text-lg',
      kicker: 'auth.register.journey.consent.kicker',
      title: 'auth.register.journey.consent.title',
      body: 'auth.register.journey.consent.body',
    },
    {
      icon: 'pi pi-arrow-right text-lg',
      kicker: 'auth.register.journey.entry.kicker',
      title: 'auth.register.journey.entry.title',
      body: 'auth.register.journey.entry.body',
    },
  ] as const;

  readonly steps = [
    { key: 'account-type', label: 'auth.register.steps.accountType' },
    { key: 'consent', label: 'auth.register.steps.consent' },
    { key: 'entry', label: 'auth.register.steps.entry' },
  ] as const;

  readonly entryPoints = [
    {
      code: 'classic' as const,
      icon: 'pi pi-mobile text-lg',
      kicker: 'auth.register.entry.classic.kicker',
      title: 'auth.register.entry.classic.title',
      body: 'auth.register.entry.classic.body',
      cta: 'auth.register.entry.classic.cta',
      route: '/auth/start' as const,
    },
    {
      code: 'wallet' as const,
      icon: 'pi pi-id-card text-lg',
      kicker: 'auth.register.entry.wallet.kicker',
      title: 'auth.register.entry.wallet.title',
      body: 'auth.register.entry.wallet.body',
      cta: 'auth.register.entry.wallet.cta',
      route: '/auth/start' as const,
    },
  ] as const;

  readonly consentForm = this.fb.nonNullable.group({
    provisioningAcknowledged: [false, Validators.requiredTrue],
    flowAcknowledged: [false, Validators.requiredTrue],
  });

  selectAccountType(type: AccountType): void {
    this.selectedAccountType.set(type);
    this.currentStep.set('consent');
    this.consentForm.reset({
      provisioningAcknowledged: false,
      flowAcknowledged: false,
    });
  }

  backToAccountType(): void {
    this.currentStep.set('account-type');
    this.selectedAccountType.set(null);
    this.consentForm.reset({
      provisioningAcknowledged: false,
      flowAcknowledged: false,
    });
  }

  continueToEntry(): void {
    if (this.consentForm.invalid) {
      this.consentForm.markAllAsTouched();
      return;
    }

    this.currentStep.set('entry');
  }

  backToConsent(): void {
    this.currentStep.set('consent');
  }

  goTo(route: EntryRoute): void {
    void this.router.navigateByUrl(`${route}?returnUrl=${encodeURIComponent(this.returnUrl)}`);
  }
}
