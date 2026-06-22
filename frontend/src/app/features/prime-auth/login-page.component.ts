import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { InputOtpModule } from 'primeng/inputotp';
import { InputTextModule } from 'primeng/inputtext';
import { MessageModule } from 'primeng/message';
import { TagModule } from 'primeng/tag';

import { AppApiService } from '../../core/api/app-api.service';

type LoginStep = 'phone' | 'code';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    RouterLink,
    TranslocoPipe,
    ButtonModule,
    CardModule,
    InputOtpModule,
    InputTextModule,
    MessageModule,
    TagModule,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <main class="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(225,29,72,0.18),_transparent_32rem),linear-gradient(135deg,_#fff7f8_0%,_#fff_44%,_#ffe4e9_100%)] text-slate-950 dark:bg-[radial-gradient(circle_at_top_left,_rgba(225,29,72,0.28),_transparent_32rem),linear-gradient(135deg,_#120509_0%,_#0f172a_58%,_#260711_100%)] dark:text-white">
      <section class="mx-auto grid min-h-screen w-full max-w-7xl items-center gap-10 px-5 py-8 md:grid-cols-[1.05fr_0.95fr] md:px-8 lg:px-10">
        <div class="space-y-8">
          <div class="inline-flex items-center rounded-full border border-rose-200 bg-white/75 px-4 py-2 text-sm font-semibold text-rose-700 shadow-sm backdrop-blur dark:border-rose-400/30 dark:bg-white/10 dark:text-rose-100">
            {{ 'authLogin.badge' | transloco }}
          </div>

          <div class="space-y-5">
            <h1 class="max-w-4xl text-4xl font-black tracking-[-0.045em] text-slate-950 md:text-6xl dark:text-white">
              {{ 'authLogin.title' | transloco }}
            </h1>
            <p class="max-w-2xl text-lg leading-8 text-slate-700 dark:text-slate-200">
              {{ 'authLogin.subtitle' | transloco }}
            </p>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            @for (method of methods; track method.code) {
              <article class="rounded-[1.4rem] border border-rose-100 bg-white/85 p-4 shadow-sm shadow-rose-950/5 backdrop-blur dark:border-white/10 dark:bg-white/10">
                <p-tag [value]="method.badge | transloco" [severity]="method.primary ? 'danger' : 'secondary'" />
                <h2 class="mt-3 text-lg font-black text-slate-950 dark:text-white">{{ method.title | transloco }}</h2>
                <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ method.body | transloco }}</p>
                <div class="mt-4">
                  @if (method.code === 'oidc_redirect' || method.code === 'eudi_wallet') {
                    <a
                      routerLink="/auth/start"
                      [queryParams]="{ returnUrl: returnUrl }"
                      class="inline-flex w-full items-center justify-center rounded-2xl bg-rose-700 px-4 py-2.5 text-sm font-bold text-white shadow-lg shadow-rose-900/15 transition hover:bg-rose-800 focus:outline-none focus:ring-4 focus:ring-rose-300"
                    >
                      {{ 'auth.login' | transloco }}
                    </a>
                  } @else if (method.code === 'sms_otp') {
                    <p class="text-sm font-semibold text-slate-700 dark:text-slate-200">
                      {{ 'authLogin.phoneHint' | transloco }}
                    </p>
                  } @else if (method.code === 'passkey') {
                    <p-button
                      class="block"
                      type="button"
                      styleClass="w-full justify-center"
                      icon="pi pi-key"
                      [label]="'auth.methods.passkey' | transloco"
                      [loading]="passkeyBusy()"
                      (onClick)="loginWithPasskey()"
                    />
                  }
                </div>
              </article>
            }
          </div>
          @if (passkeyMessage()) {
            <div class="rounded-[1.4rem] border border-rose-100 bg-white/80 p-4 text-sm text-slate-700 shadow-sm backdrop-blur dark:border-white/10 dark:bg-white/10 dark:text-slate-200">
              {{ passkeyMessage() }}
            </div>
          }
        </div>

        <div class="space-y-4">
          <div class="rounded-[1.4rem] border border-rose-100 bg-white/80 p-5 shadow-sm backdrop-blur dark:border-white/10 dark:bg-white/10">
            <p class="text-xs font-black uppercase tracking-[0.18em] text-rose-700 dark:text-rose-200">
              {{ 'auth.redirectBadge' | transloco }}
            </p>
            <h2 class="mt-2 text-2xl font-black text-slate-950 dark:text-white">
              {{ 'authLogin.title' | transloco }}
            </h2>
            <p class="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
              {{ 'authLogin.subtitle' | transloco }}
            </p>
          </div>

          <div class="space-y-4">
            <div class="rounded-[1.75rem] border border-surface bg-surface-0 shadow-2xl shadow-rose-950/10 dark:bg-surface-900">
              <div class="bg-primary px-7 py-8 text-primary-contrast">
                <p class="text-sm font-semibold uppercase tracking-[0.24em] opacity-75">
                  {{ 'authLogin.cardBadge' | transloco }}
                </p>
                <h2 class="mt-3 text-3xl font-black tracking-tight">{{ 'authLogin.cardTitle' | transloco }}</h2>
                <p class="mt-3 max-w-[26rem] text-sm leading-6 text-white/80">{{ 'authLogin.cardSubtitle' | transloco }}</p>
              </div>

              <form class="grid gap-5 p-5" [formGroup]="form" (ngSubmit)="step() === 'phone' ? requestCode() : verifyCode()">
                @if (error()) {
                  <p-message severity="error" [text]="error()! | transloco" />
                }

                @if (step() === 'phone') {
                  <label class="grid gap-2">
                    <span class="text-sm font-semibold text-surface-700 dark:text-surface-200">{{ 'authLogin.phoneLabel' | transloco }}</span>
                    <input
                      pInputText
                      formControlName="identifier"
                      autocomplete="tel"
                      inputmode="tel"
                      class="w-full"
                      [placeholder]="'authLogin.phonePlaceholder' | transloco"
                    />
                    <p class="text-sm leading-6 text-surface-500 dark:text-surface-300">
                      {{ 'authLogin.phoneHint' | transloco }}
                    </p>
                  </label>
                  <p-button
                    type="submit"
                    styleClass="w-full justify-center"
                    icon="pi pi-send"
                    [label]="'authLogin.sendCode' | transloco"
                    [loading]="loading()"
                  />
                } @else {
                  <div class="rounded-2xl border border-surface bg-highlight p-4 text-sm text-highlight-color">
                    {{ 'authLogin.codeSent' | transloco: { phone: maskedPhone() || form.controls.identifier.value } }}
                  </div>
                  <label class="grid gap-3">
                    <span class="text-sm font-semibold text-surface-700 dark:text-surface-200">{{ 'authLogin.codeLabel' | transloco }}</span>
                    <p-inputotp formControlName="code" [length]="6" [integerOnly]="true" styleClass="justify-center" />
                    <p class="text-sm leading-6 text-surface-500 dark:text-surface-300">
                      {{ 'authLogin.codeHint' | transloco }}
                    </p>
                  </label>
                  <div class="grid gap-3 sm:grid-cols-2">
                    <p-button
                      type="button"
                      severity="secondary"
                      styleClass="w-full justify-center"
                      icon="pi pi-arrow-left"
                      [label]="'authLogin.backToPhone' | transloco"
                      (onClick)="back()"
                    />
                    <p-button
                      type="submit"
                      styleClass="w-full justify-center"
                      icon="pi pi-check"
                      [label]="'authLogin.verify' | transloco"
                      [loading]="loading()"
                    />
                  </div>
                  <button
                    type="button"
                    class="text-left text-sm font-semibold text-rose-700 underline-offset-4 hover:underline dark:text-rose-200"
                    (click)="requestCode()"
                  >
                    {{ 'authLogin.resendCode' | transloco }}
                  </button>
                }
              </form>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <a
                routerLink="/auth/register"
                [queryParams]="{ returnUrl: returnUrl }"
                class="rounded-[1.4rem] border border-rose-100 bg-white/80 p-4 shadow-sm backdrop-blur transition hover:-translate-y-0.5 hover:shadow-lg dark:border-white/10 dark:bg-white/10"
              >
                <p class="text-xs font-black uppercase tracking-[0.18em] text-rose-700 dark:text-rose-200">{{ 'auth.register.badge' | transloco }}</p>
                <h3 class="mt-2 text-lg font-black text-slate-950 dark:text-white">{{ 'auth.register.title' | transloco }}</h3>
                <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ 'auth.register.subtitle' | transloco }}</p>
              </a>
              <a
                routerLink="/auth/start"
                [queryParams]="{ returnUrl: returnUrl }"
                class="rounded-[1.4rem] border border-slate-200 bg-white/80 p-4 shadow-sm backdrop-blur transition hover:-translate-y-0.5 hover:shadow-lg dark:border-white/10 dark:bg-white/10"
              >
                <p class="text-xs font-black uppercase tracking-[0.18em] text-slate-600 dark:text-slate-300">{{ 'auth.redirectBadge' | transloco }}</p>
                <h3 class="mt-2 text-lg font-black text-slate-950 dark:text-white">{{ 'auth.loginTitle' | transloco }}</h3>
                <p class="mt-1 text-sm leading-6 text-slate-600 dark:text-slate-300">{{ 'auth.loginBody' | transloco }}</p>
              </a>
            </div>
          </div>
        </div>
      </section>
    </main>
  `,
})
export class LoginPageComponent {
  private readonly api = inject(AppApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly fb = inject(FormBuilder);

  readonly loading = signal(false);
  readonly error = signal<string | null>(null);
  readonly step = signal<LoginStep>('phone');
  readonly maskedPhone = signal('');
  readonly passkeyBusy = signal(false);
  readonly passkeyMessage = signal('');
  readonly returnUrl = this.route.snapshot.queryParamMap.get('returnUrl') || '/dashboard';

  readonly methods = [
    {
      code: 'oidc_redirect',
      badge: 'auth.primaryMethod',
      title: 'auth.methods.oidc_redirect',
      body: 'auth.methodDescriptions.oidc_redirect',
      primary: true,
    },
    {
      code: 'sms_otp',
      badge: 'authLogin.badge',
      title: 'auth.methods.sms_otp',
      body: 'auth.methodDescriptions.sms_otp',
      primary: false,
    },
    {
      code: 'passkey',
      badge: 'auth.methods.passkey',
      title: 'auth.methods.passkey',
      body: 'auth.methodDescriptions.passkey',
      primary: false,
    },
    {
      code: 'eudi_wallet',
      badge: 'auth.methods.eudi_wallet',
      title: 'auth.methods.eudi_wallet',
      body: 'auth.methodDescriptions.eudi_wallet',
      primary: false,
    },
  ] as const;

  readonly form = this.fb.nonNullable.group({
    identifier: ['', [Validators.required, Validators.pattern(/^[0-9+\s().-]{7,20}$/)]],
    code: ['', [Validators.required, Validators.minLength(6), Validators.maxLength(6), Validators.pattern(/^\d{6}$/)]],
  });

  loginWithPasskey(): void {
    if (!navigator.credentials?.get || !window.PublicKeyCredential) {
      this.passkeyMessage.set('Browserul nu suportă WebAuthn/passkey în acest context.');
      return;
    }

    this.passkeyBusy.set(true);
    this.passkeyMessage.set('');
    this.api.beginPasskeyLogin().subscribe({
      next: async (response) => {
        try {
          const credential = await navigator.credentials.get({
            publicKey: {
              challenge: this.base64UrlToBuffer(response.options.challenge),
              rpId: response.options.rp.id,
              timeout: response.options.timeout,
              userVerification: response.options.userVerification,
              allowCredentials: (response.options.allowCredentials ?? []).map((entry) => ({
                id: this.base64UrlToBuffer(entry.id),
                type: entry.type,
              })),
            },
          });

          if (!(credential instanceof PublicKeyCredential)) {
            throw new Error('Credential response is not a public key credential.');
          }

          const assertion = credential.response as AuthenticatorAssertionResponse;
          await new Promise<void>((resolve, reject) => {
            this.api.finishPasskeyLogin({
              challenge: response.options.challenge,
              credential_id: credential.id,
              response: {
                clientDataJSON: this.bufferToBase64Url(assertion.clientDataJSON),
                authenticatorData: this.bufferToBase64Url(assertion.authenticatorData),
                signature: this.bufferToBase64Url(assertion.signature),
                userHandle: assertion.userHandle ? this.bufferToBase64Url(assertion.userHandle) : '',
                type: credential.type,
              },
            }).subscribe({
              next: () => {
                window.location.assign(this.returnUrl);
                resolve();
              },
              error: () => reject(new Error('passkey_login_failed')),
            });
          });
        } catch {
          this.passkeyBusy.set(false);
          this.passkeyMessage.set('Autentificarea cu passkey a fost anulată sau respinsă de browser.');
        }
      },
      error: () => {
        this.passkeyBusy.set(false);
        this.passkeyMessage.set('Backendul nu a putut genera challenge-ul passkey.');
      },
    });
  }

  requestCode(): void {
    if (this.form.controls.identifier.invalid) {
      this.form.controls.identifier.markAsTouched();
      this.error.set('authLogin.phoneInvalid');
      return;
    }

    this.loading.set(true);
    this.error.set(null);
    const phoneNumber = this.form.controls.identifier.value.trim();
    this.api.requestSmsOtp({ phone_number: phoneNumber }).subscribe({
      next: (response) => {
        this.loading.set(false);
        this.maskedPhone.set(response.masked_phone);
        this.step.set('code');
        this.form.controls.code.setValue('');
      },
      error: () => {
        this.loading.set(false);
        this.error.set('authLogin.requestFailed');
      },
    });
  }

  verifyCode(): void {
    if (this.form.controls.identifier.invalid) {
      this.form.controls.identifier.markAsTouched();
      this.error.set('authLogin.phoneInvalid');
      return;
    }

    if (this.form.controls.code.invalid) {
      this.form.controls.code.markAsTouched();
      this.error.set('authLogin.codeRequired');
      return;
    }

    this.loading.set(true);
    this.error.set(null);
    const phoneNumber = this.form.controls.identifier.value.trim();
    this.api.verifySmsOtp({
      phone_number: phoneNumber,
      code: this.form.controls.code.value.trim(),
    }).subscribe({
      next: () => {
        window.location.assign(this.returnUrl);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('authLogin.verifyFailed');
      },
    });
  }

  private base64UrlToBuffer(value: string): ArrayBuffer {
    const base64 = value.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(value.length / 4) * 4, '=');
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }
    return bytes.buffer;
  }

  private bufferToBase64Url(buffer: ArrayBuffer): string {
    if (!buffer) {
      return '';
    }
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (const byte of bytes) {
      binary += String.fromCharCode(byte);
    }
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
  }

  back(): void {
    this.step.set('phone');
    this.error.set(null);
    this.maskedPhone.set('');
    this.form.controls.code.setValue('');
  }
}
