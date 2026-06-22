import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { AvatarModule } from 'primeng/avatar';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DividerModule } from 'primeng/divider';
import { InputTextModule } from 'primeng/inputtext';
import { MessageModule } from 'primeng/message';
import { SelectModule } from 'primeng/select';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../core/authz/authz.service';

interface UpdateProfileRequest {
  name: string;
  phone_number: string;
  locale: 'ro' | 'en';
}

interface PasskeyCredentialSummary {
  id: string;
  credential_id: string;
  label: string;
  created_at: string;
  last_used_at?: string | null;
}

interface PasskeyRegistrationOptions {
  rp: { id: string; name: string };
  user: { id: string; name: string; displayName: string };
  challenge: string;
  pubKeyCredParams: PublicKeyCredentialParameters[];
  timeout: number;
  attestation: AttestationConveyancePreference;
}

interface FinishPasskeyRegistrationRequest {
  challenge: string;
  credential_id: string;
  response: unknown;
}

@Component({
  selector: 'app-profile-page',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    AvatarModule,
    ButtonModule,
    CardModule,
    DividerModule,
    InputTextModule,
    MessageModule,
    SelectModule,
    TagModule,
  ],
  template: `
    <section class="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <header class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm dark:bg-surface-900">
        <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div class="flex items-center gap-4">
            <p-avatar [label]="userInitial()" shape="circle" size="xlarge" styleClass="bg-primary text-primary-contrast" />
            <div class="min-w-0">
              <h1 class="m-0 text-2xl font-black tracking-[-0.03em]">Profil utilizator</h1>
              <p class="m-0 mt-1 text-sm text-muted-color">
                Gestionează datele contului, metodele moderne de autentificare și portofelul EUDI.
              </p>
            </div>
          </div>
          <div class="flex flex-wrap gap-2">
            @for (method of authMethods(); track method) {
              <p-tag [value]="method" severity="info" />
            }
          </div>
        </div>
      </header>

      <div class="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
        <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
          <ng-template pTemplate="title">Date utilizator</ng-template>
          <ng-template pTemplate="subtitle">Datele de identitate primite din providerul OIDC și profilul instituțional.</ng-template>

          @if (authz.user(); as user) {
            <form class="grid gap-4 md:grid-cols-2" [formGroup]="profileForm" (ngSubmit)="saveProfile()">
              <label class="profile-field">
                <span>Nume afișat</span>
                <input pInputText formControlName="name" />
              </label>
              <label class="profile-field">
                <span>Email</span>
                <input pInputText [value]="user.email" readonly />
              </label>
              <label class="profile-field">
                <span>Telefon</span>
                <input pInputText formControlName="phone_number" />
              </label>
              <label class="profile-field">
                <span>Limbă</span>
                <p-select appendTo="body" [options]="localeOptions" formControlName="locale" />
              </label>
              <label class="profile-field md:col-span-2">
                <span>Subject OIDC</span>
                <input pInputText [value]="user.sub" readonly />
              </label>
              <div class="md:col-span-2 flex items-center justify-between gap-3">
                <p-message severity="success" variant="simple" size="small" [text]="profileMessage()" />
                <p-button type="submit" label="Salvează profilul" icon="pi pi-save" [loading]="profileSaving()" [disabled]="profileForm.invalid" />
              </div>
            </form>
          }

          <p-divider />

          <div>
            <h3 class="m-0 mb-2 text-base font-bold">Roluri și context</h3>
            <div class="flex flex-wrap gap-2">
              @for (role of authz.user()?.roles ?? []; track role) {
                <p-tag [value]="role" severity="secondary" />
              } @empty {
                <span class="text-sm text-muted-color">Nu există roluri expuse în sesiune.</span>
              }
            </div>
            <p class="m-0 mt-3 text-sm text-muted-color">
              Instituție: {{ authz.institutionName() || '-' }}
            </p>
          </div>
        </p-card>

        <div class="grid gap-4">
          <p-card styleClass="border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
            <ng-template pTemplate="title">Passkey / WebAuthn</ng-template>
            <ng-template pTemplate="subtitle">Adaugă o cheie de acces pentru autentificare fără parolă și step-up pe acțiuni sensibile.</ng-template>
            <div class="flex flex-col gap-3">
              <div class="flex flex-wrap gap-2">
                @for (passkey of passkeys(); track passkey.id) {
                  <p-tag [value]="passkey.label" severity="success" />
                } @empty {
                  <p-tag value="Nicio passkey activă" severity="warn" />
                }
              </div>
              <p-message *ngIf="passkeyMessage()" severity="info" variant="simple" size="small" [text]="passkeyMessage()" />
              <button pButton type="button" class="w-fit" [disabled]="passkeyBusy()" (click)="addPasskey()">
                <i class="pi pi-key"></i>
                <span>{{ passkeyBusy() ? 'Se pregătește...' : 'Adaugă passkey' }}</span>
              </button>
            </div>
          </p-card>

          <p-card styleClass="border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
            <ng-template pTemplate="title">Portofel EUDI</ng-template>
            <ng-template pTemplate="subtitle">Activează conectarea cu wallet EUDI pentru identitate digitală și atribute verificate.</ng-template>
            <div class="flex flex-col gap-3">
              <p-tag [value]="eudiStatus()" [severity]="eudiStatus() === 'activ' ? 'success' : 'info'" />
              <p-message *ngIf="eudiMessage()" severity="info" variant="simple" size="small" [text]="eudiMessage()" />
              <p-button label="Activează EUDI wallet" icon="pi pi-wallet" [loading]="eudiBusy()" (onClick)="activateEudiWallet()" />
            </div>
          </p-card>
        </div>
      </div>
    </section>
  `,
  styles: `
    .profile-field {
      display: flex;
      flex-direction: column;
      gap: 0.4rem;
      color: var(--p-text-muted-color);
      font-size: 0.875rem;
      font-weight: 600;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfilePageComponent {
  private readonly http = inject(HttpClient);
  private readonly fb = inject(FormBuilder);
  protected readonly authz = inject(AuthzService);
  protected readonly localeOptions = [{ label: 'Română', value: 'ro' }, { label: 'English', value: 'en' }];
  protected readonly passkeys = signal<PasskeyCredentialSummary[]>([]);
  protected readonly profileSaving = signal(false);
  protected readonly passkeyBusy = signal(false);
  protected readonly eudiBusy = signal(false);
  protected readonly profileMessage = signal('');
  protected readonly passkeyMessage = signal('');
  protected readonly eudiMessage = signal('');
  protected readonly eudiStatus = computed(() => this.authMethods().includes('eudi_wallet') ? 'activ' : 'neactivat');

  protected readonly profileForm = this.fb.nonNullable.group({
    name: ['', Validators.required],
    phone_number: [''],
    locale: this.fb.nonNullable.control<'ro' | 'en'>('ro'),
  });

  constructor() {
    effect(() => {
      const user = this.authz.user();
      if (!user) {
        return;
      }
      this.profileForm.patchValue({
        name: user.name ?? '',
        phone_number: user.phone_number ?? '',
        locale: user.locale ?? 'ro',
      }, { emitEvent: false });
    });
    this.loadPasskeys();
  }

  protected userInitial(): string {
    const user = this.authz.user();
    return (user?.name || user?.email || user?.sub || 'U').slice(0, 1).toUpperCase();
  }

  protected authMethods(): string[] {
    return this.authz.session()?.authentication ?? [];
  }

  protected saveProfile(): void {
    if (this.profileForm.invalid) {
      this.profileForm.markAllAsTouched();
      return;
    }
    this.profileSaving.set(true);
    this.profileMessage.set('');
    this.http.put('/api/profile', this.profileForm.getRawValue() satisfies UpdateProfileRequest).subscribe({
      next: async () => {
        await this.authz.reload();
        this.profileSaving.set(false);
        this.profileMessage.set('Profilul a fost salvat.');
      },
      error: () => {
        this.profileSaving.set(false);
        this.profileMessage.set('Profilul nu a putut fi salvat.');
      },
    });
  }

  protected loadPasskeys(): void {
    this.http.get<PasskeyCredentialSummary[]>('/api/passkeys').subscribe({
      next: (items) => this.passkeys.set(items ?? []),
      error: () => this.passkeys.set([]),
    });
  }

  protected addPasskey(): void {
    if (!navigator.credentials?.create || !window.PublicKeyCredential) {
      this.passkeyMessage.set('Browserul nu suportă WebAuthn/passkey în acest context.');
      return;
    }
    this.passkeyBusy.set(true);
    this.passkeyMessage.set('');
    this.http.post<PasskeyRegistrationOptions>('/api/passkeys/register-options', {}).subscribe({
      next: async (options) => {
        try {
          const credential = await navigator.credentials.create({
            publicKey: {
              ...options,
              challenge: this.base64UrlToBuffer(options.challenge),
              user: {
                ...options.user,
                id: this.base64UrlToBuffer(options.user.id),
              },
            },
          });
          if (!(credential instanceof PublicKeyCredential)) {
            throw new Error('Credential response is not a public key credential.');
          }
          const response = credential.response as AuthenticatorAttestationResponse;
          const payload: FinishPasskeyRegistrationRequest = {
            challenge: options.challenge,
            credential_id: credential.id,
            response: {
              clientDataJSON: this.bufferToBase64Url(response.clientDataJSON),
              attestationObject: this.bufferToBase64Url(response.attestationObject),
              transports: response.getTransports?.() ?? [],
              type: credential.type,
            },
          };
          this.http.post('/api/passkeys/register-finish', payload).subscribe({
            next: async () => {
              this.passkeyBusy.set(false);
              this.passkeyMessage.set('Passkey a fost adăugată.');
              this.loadPasskeys();
              await this.authz.reload();
            },
            error: () => {
              this.passkeyBusy.set(false);
              this.passkeyMessage.set('Înregistrarea passkey nu a putut fi finalizată.');
            },
          });
        } catch {
          this.passkeyBusy.set(false);
          this.passkeyMessage.set('Înregistrarea passkey a fost anulată sau respinsă de browser.');
        }
      },
      error: () => {
        this.passkeyBusy.set(false);
        this.passkeyMessage.set('Backendul nu a putut genera challenge-ul passkey.');
      },
    });
  }

  protected activateEudiWallet(): void {
    this.eudiBusy.set(true);
    this.eudiMessage.set('');
    this.http.post('/api/eudi-wallet/activate', {}).subscribe({
      next: async () => {
        await this.authz.reload();
        this.eudiBusy.set(false);
        this.eudiMessage.set('Wallet EUDI a fost activat pentru cont.');
      },
      error: () => {
        this.eudiBusy.set(false);
        this.eudiMessage.set('Wallet EUDI nu a putut fi activat.');
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
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (const byte of bytes) {
      binary += String.fromCharCode(byte);
    }
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
  }
}
