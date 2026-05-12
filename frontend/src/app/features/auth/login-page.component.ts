import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { Router } from '@angular/router';

import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AppApiService } from '../../core/api/app-api.service';
import { AuthService } from '../../core/auth/auth.service';
import { AuthzService } from '../../core/authz/authz.service';
import { AuthShellComponent } from '../../shared/auth-shell/auth-shell.component';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    AuthShellComponent,
    MatButtonModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSnackBarModule,
  ],
  templateUrl: './login-page.component.html',
  styleUrl: './login-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginPageComponent {
  private readonly api = inject(AppApiService);
  private readonly auth = inject(AuthService);
  private readonly authz = inject(AuthzService);
  private readonly fb = inject(FormBuilder);
  private readonly snackBar = inject(MatSnackBar);
  private readonly router = inject(Router);
  private readonly transloco = inject(TranslocoService);

  protected readonly methods = toSignal(this.api.authMethods(), { initialValue: { methods: [] } });
  protected readonly uiConfig = toSignal(this.api.authUiConfig(), {
    initialValue: {
      auth_flow: 'redirect',
      default_locale: 'ro',
      available_locales: ['ro', 'en'],
      theme_family: 'material3-expressive',
      theme_brand: 'red-rose',
      oidc_issuer: '',
      oidc_client_id: '',
      desktop_client_id: '',
      sms_otp_enabled: false,
      passkey_enabled: false,
      eudi_wallet_enabled: false,
      gdpr_features_enabled: true,
    },
  });
  protected readonly enabledMethods = computed(() => this.methods().methods.filter((method) => method.enabled));
  protected readonly secondaryMethods = computed(() =>
    this.enabledMethods().filter((method) => method.code !== 'oidc_redirect'),
  );
  protected readonly smsState = signal<'idle' | 'sent' | 'verifying'>('idle');
  protected readonly maskedPhone = signal('');

  protected readonly smsRequestForm = this.fb.group({
    identifier: this.fb.nonNullable.control('', [Validators.required, this.validateIdentifier]),
  });

  protected readonly smsVerifyForm = this.fb.group({
    identifier: this.fb.nonNullable.control('', [Validators.required, this.validateIdentifier]),
    code: this.fb.nonNullable.control('', [Validators.required, Validators.minLength(6), Validators.maxLength(6)]),
  });

  protected async signIn(): Promise<void> {
    await this.auth.login();
  }

  protected requestSmsOtp(): void {
    if (this.smsRequestForm.invalid) {
      this.smsRequestForm.markAllAsTouched();
      return;
    }

    const identifier = this.smsRequestForm.controls.identifier.getRawValue();
    this.api.requestSmsOtp({ identifier }).subscribe({
      next: (response) => {
        this.smsState.set('sent');
        this.maskedPhone.set(response.masked_phone);
        this.smsVerifyForm.patchValue({ identifier });
        this.snackBar.open(
          this.transloco.translate('auth.smsOtp.messages.sent'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('auth.smsOtp.messages.sendFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected verifySmsOtp(): void {
    if (this.smsVerifyForm.invalid) {
      this.smsVerifyForm.markAllAsTouched();
      return;
    }

    this.smsState.set('verifying');
    const payload = this.smsVerifyForm.getRawValue();
    this.api.verifySmsOtp(payload).subscribe({
      next: async () => {
        await this.authz.reload();
        this.snackBar.open(
          this.transloco.translate('auth.smsOtp.messages.verified'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.smsState.set('sent');
        await this.router.navigateByUrl(this.auth.consumeReturnUrl());
      },
      error: () => {
        this.smsState.set('sent');
        this.snackBar.open(
          this.transloco.translate('auth.smsOtp.messages.verifyFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected methodIcon(code: string): string {
    switch (code) {
      case 'sms_otp':
        return 'sms';
      case 'passkey':
        return 'fingerprint';
      case 'eudi_wallet':
        return 'id_card';
      default:
        return 'shield_lock';
    }
  }

  private validateIdentifier(control: AbstractControl<string>): ValidationErrors | null {
    const value = control.value?.trim() ?? '';
    if (!value) {
      return null;
    }

    const looksLikeEmail = value.includes('@');
    if (looksLikeEmail) {
      return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) ? null : { identifier: true };
    }

    const digits = value.replace(/\D/g, '');
    return digits.length >= 10 ? null : { identifier: true };
  }
}
