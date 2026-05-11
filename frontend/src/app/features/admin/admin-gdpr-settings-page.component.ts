import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { AdminApiService } from '../../core/api/admin-api.service';
import { AdminGdprSetting, UpdateAdminGdprSettingRequest } from '../../core/api/api.types';

@Component({
  selector: 'app-admin-gdpr-settings-page',
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
  templateUrl: './admin-gdpr-settings-page.component.html',
  styleUrl: './admin-gdpr-settings-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminGdprSettingsPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly fb = inject(FormBuilder);
  private readonly transloco = inject(TranslocoService);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly refreshToken = signal(0);

  protected readonly response = toSignal(toObservable(this.refreshToken).pipe(switchMap(() => this.adminApi.gdprSettings())), {
    initialValue: { items: [], total: 0, page: 1, pageSize: 0 },
  });

  protected readonly items = computed(() => this.response().items);
  protected readonly selected = signal<AdminGdprSetting | null>(null);
  protected readonly codeOptions = [
    'publication_anonymization_required',
    'subject_export_requires_approval',
    'default_response_sla_days',
    'retention_review_notice_days',
    'portfolio_consent_required',
    'portfolio_authenticity_required',
  ];

  protected readonly form = this.fb.group({
    code: this.fb.nonNullable.control('publication_anonymization_required', [Validators.required]),
    value_type: this.fb.nonNullable.control<'text' | 'bool' | 'int'>('bool', [Validators.required]),
    value_text: this.fb.nonNullable.control(''),
    value_bool: this.fb.nonNullable.control(true),
    value_int: this.fb.nonNullable.control(30, [Validators.required, Validators.min(0)]),
  });

  protected readonly currentType = computed(() => this.form.controls.value_type.getRawValue());

  protected selectSetting(item: AdminGdprSetting): void {
    this.selected.set(item);
    this.form.reset({
      code: item.code,
      value_type: item.value_type,
      value_text: item.value_text,
      value_bool: item.value_bool,
      value_int: item.value_int,
    });
  }

  protected save(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    const payload: UpdateAdminGdprSettingRequest = {
      code: raw.code,
      value_type: raw.value_type,
      value_text: raw.value_text,
      value_bool: raw.value_bool,
      value_int: raw.value_int,
    };

    this.adminApi.saveGdprSetting(payload).subscribe({
      next: () => {
        this.refreshToken.update((value) => value + 1);
        this.snackBar.open(
          this.transloco.translate('admin.gdprSettings.messages.saved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.gdprSettings.messages.saveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
