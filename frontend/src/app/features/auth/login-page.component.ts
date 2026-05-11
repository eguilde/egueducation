import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';

import { AppApiService } from '../../core/api/app-api.service';
import { AuthService } from '../../core/auth/auth.service';

@Component({
  selector: 'app-login-page',
  standalone: true,
  imports: [TranslocoPipe, MatButtonModule, MatCardModule, MatChipsModule, MatIconModule],
  templateUrl: './login-page.component.html',
  styleUrl: './login-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginPageComponent {
  private readonly api = inject(AppApiService);
  private readonly auth = inject(AuthService);

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

  protected async signIn(): Promise<void> {
    await this.auth.login();
  }
}
