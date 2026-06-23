import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AvatarModule } from 'primeng/avatar';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { FieldsetModule } from 'primeng/fieldset';
import { DividerModule } from 'primeng/divider';
import { InputTextModule } from 'primeng/inputtext';
import { MessageModule } from 'primeng/message';
import { TableModule } from 'primeng/table';
import { SelectModule } from 'primeng/select';
import { TagModule } from 'primeng/tag';
import { TooltipModule } from 'primeng/tooltip';

import { AuthService } from '../../core/auth/auth.service';
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

interface ProfileDatum {
  label: string;
  value: string;
}

type ProfileUserLike = {
  sub?: string;
  email?: string;
  name?: string;
  phone_number?: string;
  locale?: unknown;
  roles?: string[];
};

@Component({
  selector: 'app-profile-page',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    AvatarModule,
    ButtonModule,
    CardModule,
    FieldsetModule,
    DividerModule,
    InputTextModule,
    MessageModule,
    TableModule,
    SelectModule,
    TagModule,
    TooltipModule,
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
              <p-tag [value]="methodLabel(method)" severity="info" />
            }
          </div>
        </div>
      </header>

      @if (profileUser()) {
        <div class="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
          <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
          <ng-template pTemplate="title">Date utilizator</ng-template>
          <ng-template pTemplate="subtitle">Datele de identitate primite din providerul OIDC și profilul instituțional.</ng-template>
          <div class="mb-3 rounded-2xl border border-surface-200 bg-surface-50 px-4 py-3 text-sm text-muted-color dark:border-surface-800 dark:bg-surface-950">
            <span class="font-semibold text-color">{{ displayName() }}</span>
            <span class="mx-2">•</span>
            <span>{{ userEmail() }}</span>
          </div>

          <p-table [value]="profileDetails()" [tableStyle]="{ 'min-width': '100%' }" styleClass="p-datatable-sm p-datatable-striped">
            <ng-template pTemplate="body" let-row>
              <tr>
                <td class="w-1/3 font-semibold text-muted-color">{{ row.label }}</td>
                <td class="break-words text-color">{{ row.value }}</td>
              </tr>
            </ng-template>
          </p-table>

          <p-divider />

          <form class="grid gap-4 md:grid-cols-2" [formGroup]="profileForm" (ngSubmit)="saveProfile()">
            <label class="profile-field">
              <span>Nume afișat</span>
              <input pInputText formControlName="name" />
            </label>
            <label class="profile-field">
              <span>Email</span>
              <input pInputText [value]="userEmail()" readonly />
            </label>
            <label class="profile-field">
              <span>Telefon</span>
              <input pInputText formControlName="phone_number" />
            </label>
            <label class="profile-field">
              <span>Limbă</span>
              <p-select appendTo="body" [options]="localeOptions" formControlName="locale" />
            </label>
            <div class="md:col-span-2 flex items-center justify-between gap-3">
              <p-message severity="success" variant="simple" size="small" [text]="profileMessage()" />
              <p-button type="submit" label="Salvează profilul" icon="pi pi-save" [loading]="profileSaving()" [disabled]="profileForm.invalid" />
            </div>
          </form>

          <p-divider />

          <p-fieldset legend="Roluri și context">
            <div class="grid gap-4">
              <p-table [value]="roleRows()" [tableStyle]="{ 'min-width': '100%' }" styleClass="p-datatable-sm">
                <ng-template pTemplate="body" let-row>
                  <tr>
                    <td class="w-1/3 font-semibold text-muted-color">{{ row.label }}</td>
                    <td>
                      @if (row.kind === 'taglist') {
                        <div class="flex flex-wrap gap-2">
                          @for (item of row.values; track item) {
                            <p-tag [value]="item" severity="secondary" />
                          }
                        </div>
                      } @else {
                        <span class="text-color">{{ row.value }}</span>
                      }
                    </td>
                  </tr>
                </ng-template>
              </p-table>
            </div>
          </p-fieldset>
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
      } @else if (authenticated()) {
        <p-card styleClass="border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
          <ng-template pTemplate="title">Date de profil în curs de încărcare</ng-template>
          <ng-template pTemplate="subtitle">Sesiunea este autenticată, dar profilul instituțional nu a fost încă preluat.</ng-template>
          <p-message severity="info" variant="simple" [text]="'Sincronizăm datele contului cu backendul. Reîncarcă pagina dacă mesajul persistă.'" />
        </p-card>
      } @else {
        <p-card styleClass="border border-surface bg-surface-0 shadow-sm dark:bg-surface-900">
          <ng-template pTemplate="title">Autentificare necesară</ng-template>
          <ng-template pTemplate="subtitle">Profilul este disponibil numai după autentificare.</ng-template>
          <p-button label="Autentificare" icon="pi pi-sign-in" (onClick)="signIn()" />
        </p-card>
      }
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
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
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
  protected readonly authenticated = computed(() => this.auth.isAuthenticated());
  protected readonly sessionUser = computed(() => this.authz.session()?.user ?? null);
  protected readonly profileUser = computed<ProfileUserLike | null>(() => (this.sessionUser() ?? this.auth.profile() ?? null) as ProfileUserLike | null);
  protected readonly institutionalName = computed(() => this.authz.institutionName() || 'Nespecificată');
  protected readonly profileDetails = computed<ProfileDatum[]>(() => {
    const user = this.profileUser();
    return [
      { label: 'Nume', value: this.toText(user?.name) },
      { label: 'Email', value: this.toText(user?.email) },
      { label: 'Telefon', value: this.toText(user?.phone_number) },
      { label: 'Limbă', value: this.toLocale(user?.locale) },
      { label: 'Subject OIDC', value: this.toText(user?.sub) },
      { label: 'Instituție', value: this.institutionalName() },
    ];
  });
  protected readonly roleRows = computed(() => {
    const roles = this.sessionUser()?.roles ?? this.auth.profile()?.roles ?? [];
    return [
      { label: 'Roluri', kind: 'taglist' as const, values: roles.length > 0 ? roles.map((role) => this.authz.roleLabel(role)) : ['Fără roluri în sesiune'] },
      { label: 'Autentificare', kind: 'text' as const, value: this.authMethods().length > 0 ? this.authMethods().map((method) => this.methodLabel(method)).join(', ') : 'Nespecificată' },
    ];
  });

  protected readonly profileForm = this.fb.nonNullable.group({
    name: ['', Validators.required],
    phone_number: [''],
    locale: this.fb.nonNullable.control<'ro' | 'en'>('ro'),
  });

  constructor() {
    effect(() => {
      if (this.auth.isAuthenticated() && !this.authz.session()) {
        void this.authz.reload();
      }

      const user = this.profileUser();
      if (!user) {
        return;
      }
      this.profileForm.patchValue({
        name: this.toText(user.name),
        phone_number: this.toText(user.phone_number),
        locale: this.toLocale(user.locale),
      }, { emitEvent: false });
    });
    this.loadPasskeys();
  }

  protected userInitial(): string {
    const user = this.profileUser();
    return this.toText(user?.name || user?.email || user?.sub, 'U').slice(0, 1).toUpperCase();
  }

  protected authMethods(): string[] {
    return this.authz.session()?.authentication ?? [];
  }

  protected methodLabel(method: string): string {
    const labels: Record<string, string> = {
      sms: 'SMS',
      passkey: 'Passkey',
      eudi_wallet: 'EUDI Wallet',
      password: 'Parolă',
    };
    return labels[method] ?? method;
  }

  protected async signIn(): Promise<void> {
    await this.auth.login(this.router.url);
  }

  protected displayName(): string {
    const user = this.profileUser();
    return this.toText(user?.name || user?.email || user?.sub, 'Cont utilizator');
  }

  protected userEmail(): string {
    return this.toText(this.profileUser()?.email);
  }

  protected userPhone(): string {
    return this.toText(this.profileUser()?.phone_number);
  }

  protected userSub(): string {
    return this.toText(this.profileUser()?.sub);
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

  private toText(value: unknown, fallback = '-'): string {
    if (typeof value !== 'string') {
      return fallback;
    }
    const trimmed = value.trim();
    return trimmed || fallback;
  }

  private toLocale(value: unknown): 'ro' | 'en' {
    return value === 'en' ? 'en' : 'ro';
  }
}
