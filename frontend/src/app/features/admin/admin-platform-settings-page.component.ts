import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AppApiService } from '../../core/api/app-api.service';
import { AdminAuthMethodSetting, AdminModuleSetting } from '../../core/api/api.types';

@Component({
  selector: 'app-admin-platform-settings-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSnackBarModule,
  ],
  templateUrl: './admin-platform-settings-page.component.html',
  styleUrl: './admin-platform-settings-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminPlatformSettingsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly appApi = inject(AppApiService);
  private readonly fb = inject(FormBuilder);
  private readonly transloco = inject(TranslocoService);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly methodsResponse = toSignal(this.adminApi.authMethods(), {
    initialValue: { items: [], total: 0, page: 1, pageSize: 0 },
  });
  protected readonly modulesResponse = toSignal(this.adminApi.modules(), {
    initialValue: { items: [], total: 0, page: 1, pageSize: 0 },
  });
  protected readonly uiConfig = toSignal(this.appApi.authUiConfig(), {
    initialValue: {
      auth_flow: 'redirect',
      default_locale: 'ro',
      available_locales: ['ro', 'en'] as Array<'ro' | 'en'>,
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

  protected readonly methodItems = computed(() => this.methodsResponse().items);
  protected readonly moduleItems = computed(() => this.modulesResponse().items);

  protected readonly methodForm = this.fb.group({
    code: this.fb.nonNullable.control('oidc_redirect', [Validators.required]),
    enabled: this.fb.nonNullable.control(true),
    primary_method: this.fb.nonNullable.control(false),
    sort_order: this.fb.nonNullable.control(10, [Validators.required]),
  });

  protected readonly moduleForm = this.fb.group({
    code: this.fb.nonNullable.control('dashboard', [Validators.required]),
    active: this.fb.nonNullable.control(true),
  });

  protected readonly selectedMethod = signal<AdminAuthMethodSetting | null>(null);
  protected readonly selectedModule = signal<AdminModuleSetting | null>(null);

  protected selectMethod(item: AdminAuthMethodSetting): void {
    this.selectedMethod.set(item);
    this.methodForm.reset({
      code: item.code,
      enabled: item.enabled,
      primary_method: item.primary_method,
      sort_order: item.sort_order,
    });
  }

  protected selectModule(item: AdminModuleSetting): void {
    this.selectedModule.set(item);
    this.moduleForm.reset({
      code: item.code,
      active: item.active,
    });
  }

  protected saveMethod(): void {
    if (this.methodForm.invalid) {
      this.methodForm.markAllAsTouched();
      return;
    }
    this.adminApi.saveAuthMethod(this.methodForm.getRawValue()).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('admin.platform.messages.methodSaved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.platform.messages.methodSaveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected saveModule(): void {
    if (this.moduleForm.invalid) {
      this.moduleForm.markAllAsTouched();
      return;
    }
    this.adminApi.saveModule(this.moduleForm.getRawValue()).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('admin.platform.messages.moduleSaved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.platform.messages.moduleSaveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
