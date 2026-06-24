import { CommonModule } from '@angular/common';
import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
import { DatePickerModule } from 'primeng/datepicker';
import { InputNumberModule } from 'primeng/inputnumber';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { ToastModule } from 'primeng/toast';
import { firstValueFrom } from 'rxjs';

import { AuthzService } from '../../../core/authz/authz.service';
import { WizardStepperComponent } from '../shared/wizard/wizard-stepper.component';
import { EducationWizardStepState } from '../shared/wizard/wizard.models';

interface PortfolioWorkflowRecord {
  id: string;
  portfolio_code: string;
  owner_name: string;
  owner_role: string;
  school_year: string;
  status: string;
  section_count: number;
  last_updated_on: string;
  retention_until: string;
  transfer_status: string;
  authenticity_declared: boolean;
  consent_captured: boolean;
  custodian: string;
  notes: string;
}

@Component({
  selector: 'app-education-portfolio-workflow-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
    CheckboxModule,
    DatePickerModule,
    InputNumberModule,
    InputTextModule,
    SelectModule,
    TagModule,
    TextareaModule,
    ToastModule,
    WizardStepperComponent,
  ],
  providers: [MessageService],
  template: `
    <p-toast />

    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-sitemap"></i>
              Flux portofoliu profesor
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard de completare si validare</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Ghid pentru actualizarea portofoliului existent, cu trimitere la verificare, revenire pentru completari
              si salvarea datelor curente.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="ownerLabel()" severity="contrast" />
            <p-tag [value]="institutionName()" severity="secondary" />
            <p-button
              [routerLink]="['/education', 'portfolio', 'me']"
              label="Inapoi la portofoliu"
              icon="pi pi-arrow-left"
              severity="secondary"
              [outlined]="true"
            />
          </div>
        </div>
      </div>

      <app-wizard-stepper [steps]="steps()" />

      @if (loading()) {
        <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
          <p class="m-0 text-sm text-muted-color">Se incarca portofoliul curent...</p>
        </p-card>
      } @else if (record(); as current) {
        <div class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
          <form class="space-y-4" [formGroup]="form">
            @if (activeStep() === 0) {
              <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
                <div class="space-y-5">
                  <div class="space-y-1">
                    <h2 class="m-0 text-xl font-semibold">1. Identificare</h2>
                    <p class="m-0 text-sm text-muted-color">
                      Confirmam portofoliul gasit pentru titularul curent.
                    </p>
                  </div>

                  <div class="grid gap-4 md:grid-cols-2">
                    <label class="space-y-2 md:col-span-2">
                      <span class="text-sm font-medium">Titular</span>
                      <input pInputText formControlName="ownerName" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Functie</span>
                      <input pInputText formControlName="ownerRole" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">An scolar</span>
                      <input pInputText formControlName="schoolYear" />
                    </label>
                  </div>
                </div>
              </p-card>
            }

            @if (activeStep() === 1) {
              <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
                <div class="space-y-5">
                  <div class="space-y-1">
                    <h2 class="m-0 text-xl font-semibold">2. Completare</h2>
                    <p class="m-0 text-sm text-muted-color">
                      Actualizam datele operative, retentia si urmarirea transferului.
                    </p>
                  </div>

                  <div class="grid gap-4 md:grid-cols-2">
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Status curent</span>
                      <p-select
                        formControlName="status"
                        [options]="statusOptions"
                        optionLabel="label"
                        optionValue="value"
                        appendTo="body"
                      />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Status transfer</span>
                      <p-select
                        formControlName="transferStatus"
                        [options]="transferOptions"
                        optionLabel="label"
                        optionValue="value"
                        appendTo="body"
                      />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Numar sectiuni</span>
                      <p-inputnumber formControlName="sectionCount" inputId="sectionCount" [min]="0" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Custode</span>
                      <input pInputText formControlName="custodian" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Ultima actualizare</span>
                      <p-datepicker formControlName="lastUpdatedOn" dateFormat="yy-mm-dd" appendTo="body" [showIcon]="true" />
                    </label>
                    <label class="space-y-2">
                      <span class="text-sm font-medium">Retentie pana la</span>
                      <p-datepicker formControlName="retentionUntil" dateFormat="yy-mm-dd" appendTo="body" [showIcon]="true" />
                    </label>
                  </div>
                </div>
              </p-card>
            }

            @if (activeStep() === 2) {
              <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
                <div class="space-y-5">
                  <div class="space-y-1">
                    <h2 class="m-0 text-xl font-semibold">3. Verificare</h2>
                    <p class="m-0 text-sm text-muted-color">
                      Ajustam actiunea de circuit fara a pierde continutul deja completat.
                    </p>
                  </div>

                  <div class="grid gap-3">
                    <div class="flex flex-wrap gap-2">
                      <p-button label="Trimite la verificare" icon="pi pi-send" (onClick)="setStatus('submitted')" />
                      <p-button
                        label="Revino la completare"
                        icon="pi pi-undo"
                        severity="secondary"
                        [outlined]="true"
                        (onClick)="setStatus('draft')"
                      />
                      <p-button
                        label="Valideaza"
                        icon="pi pi-check"
                        severity="success"
                        [outlined]="true"
                        (onClick)="setStatus('validated')"
                      />
                      <p-button
                        label="Returneaza"
                        icon="pi pi-replay"
                        severity="secondary"
                        [outlined]="true"
                        (onClick)="setStatus('returned')"
                      />
                      <p-button
                        label="Arhiveaza"
                        icon="pi pi-box"
                        severity="secondary"
                        [outlined]="true"
                        (onClick)="setStatus('archived')"
                      />
                    </div>

                    <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                      <p-checkbox formControlName="authenticityDeclared" [binary]="true" />
                      <div class="space-y-1">
                        <div class="font-medium">Autenticitate declarata</div>
                        <p class="m-0 text-sm text-muted-color">Se pastreaza marcajul procedural pentru documentele validate.</p>
                      </div>
                    </label>

                    <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                      <p-checkbox formControlName="consentCaptured" [binary]="true" />
                      <div class="space-y-1">
                        <div class="font-medium">Consimtamant capturat</div>
                        <p class="m-0 text-sm text-muted-color">Ajustarea de stare nu pierde urmarirea de conformitate.</p>
                      </div>
                    </label>

                    <label class="space-y-2">
                      <span class="text-sm font-medium">Note</span>
                      <textarea pTextarea formControlName="notes" rows="4"></textarea>
                    </label>
                  </div>
                </div>
              </p-card>
            }

            @if (activeStep() === 3) {
              <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
                <div class="space-y-4">
                  <div class="space-y-1">
                    <h2 class="m-0 text-xl font-semibold">4. Confirmare</h2>
                    <p class="m-0 text-sm text-muted-color">
                      Salvam schimbarea si ne intoarcem in portofoliul personal.
                    </p>
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <p-button label="Salveaza si intoarce" icon="pi pi-check" [loading]="saving()" (onClick)="save()" />
                    <p-button
                      label="Inapoi"
                      icon="pi pi-arrow-left"
                      severity="secondary"
                      [outlined]="true"
                      (onClick)="goPrevious()"
                    />
                  </div>
                </div>
              </p-card>
            }

            <div class="flex flex-wrap justify-between gap-3">
              <p-button
                label="Pasul anterior"
                icon="pi pi-arrow-left"
                severity="secondary"
                [outlined]="true"
                [disabled]="activeStep() === 0 || saving()"
                (onClick)="goPrevious()"
              />
              <p-button
                label="Pasul urmator"
                icon="pi pi-arrow-right"
                iconPos="right"
                [disabled]="activeStep() >= maxStepIndex || saving() || !canAdvance()"
                (onClick)="goNext()"
              />
            </div>
          </form>

          <div class="grid gap-4">
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-1">
                <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Portofoliu incarcat</div>
                <div class="text-lg font-semibold">{{ current.portfolio_code }}</div>
                <p class="m-0 text-sm text-muted-color">{{ current.owner_name }} - {{ current.owner_role }}</p>
              </div>
              <div class="mt-4 grid gap-3 text-sm">
                <div class="rounded-xl border border-surface p-3">
                  <div class="text-xs uppercase tracking-wide text-muted-color">Status</div>
                  <div class="mt-1 font-medium">{{ statusLabel(form.controls.status.value ?? 'draft') }}</div>
                </div>
                <div class="rounded-xl border border-surface p-3">
                  <div class="text-xs uppercase tracking-wide text-muted-color">Ultima actualizare</div>
                  <div class="mt-1 font-medium">{{ formatDateLabel(form.controls.lastUpdatedOn.value) }}</div>
                </div>
                <div class="rounded-xl border border-surface p-3">
                  <div class="text-xs uppercase tracking-wide text-muted-color">Retentie</div>
                  <div class="mt-1 font-medium">{{ formatDateLabel(form.controls.retentionUntil.value) }}</div>
                </div>
              </div>
            </p-card>
          </div>
        </div>
      } @else {
        <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
          <p class="m-0 text-sm text-muted-color">Nu am gasit un portofoliu pentru titularul curent.</p>
        </p-card>
      }
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationPortfolioWorkflowWizardComponent implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly messages = inject(MessageService);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly ownerLabel = computed(() => this.ownerNameHint() || this.authz.user()?.name?.trim() || 'Profesor');
  protected readonly activeStep = signal(0);
  protected readonly loading = signal(true);
  protected readonly saving = signal(false);
  protected readonly record = signal<PortfolioWorkflowRecord | null>(null);
  protected readonly ownerNameHint = signal('');
  protected readonly maxStepIndex = 3;

  protected readonly statusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'In verificare', value: 'submitted' },
    { label: 'Validat', value: 'validated' },
    { label: 'Returnat', value: 'returned' },
    { label: 'Arhivat', value: 'archived' },
  ];

  protected readonly transferOptions = [
    { label: 'Niciun transfer', value: 'none' },
    { label: 'Pregatit', value: 'prepared' },
    { label: 'Trimis', value: 'sent' },
    { label: 'Receptionat', value: 'received' },
  ];

  protected readonly form = this.fb.group({
    ownerName: ['', [Validators.required, Validators.minLength(3)]],
    ownerRole: ['', Validators.required],
    schoolYear: ['', Validators.required],
    status: ['draft', Validators.required],
    sectionCount: [0, [Validators.required, Validators.min(0)]],
    lastUpdatedOn: [new Date(), Validators.required],
    retentionUntil: [new Date(), Validators.required],
    transferStatus: ['none', Validators.required],
    authenticityDeclared: [false],
    consentCaptured: [false],
    custodian: [''],
    notes: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'identify', label: 'Identificare', description: 'Portofoliul curent.', status: this.stepStatus(0) },
    { key: 'complete', label: 'Completare', description: 'Date si retentie.', status: this.stepStatus(1) },
    { key: 'workflow', label: 'Verificare', description: 'Circuit procedural.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Salvare si intoarcere.', status: this.stepStatus(3) },
  ]);

  async ngOnInit(): Promise<void> {
    const ownerName = this.route.snapshot.queryParamMap.get('owner_name')
      ?? this.route.snapshot.queryParamMap.get('filter_owner_name')
      ?? this.authz.user()?.name?.trim()
      ?? '';
    this.ownerNameHint.set(ownerName);
    await this.loadCurrentPortfolio(ownerName);
  }

  protected canAdvance(): boolean {
    if (this.activeStep() === 0) {
      return !!this.record();
    }
    if (this.activeStep() === 1) {
      return this.form.controls.ownerName.valid
        && this.form.controls.ownerRole.valid
        && this.form.controls.schoolYear.valid
        && this.form.controls.sectionCount.valid
        && this.form.controls.lastUpdatedOn.valid
        && this.form.controls.retentionUntil.valid;
    }
    return true;
  }

  protected goNext(): void {
    if (!this.canAdvance() || this.activeStep() >= this.maxStepIndex) {
      return;
    }
    this.activeStep.update((value) => Math.min(this.maxStepIndex, value + 1));
  }

  protected goPrevious(): void {
    this.activeStep.update((value) => Math.max(0, value - 1));
  }

  protected setStatus(status: 'draft' | 'submitted' | 'validated' | 'returned' | 'archived'): void {
    this.form.controls.status.setValue(status);
    this.messages.add({
      severity: 'info',
      summary: 'Status actualizat',
      detail: `Portofoliul a fost setat pe ${this.statusLabel(status)}.`,
    });
  }

  protected async save(): Promise<void> {
    const current = this.record();
    if (!current || this.saving()) {
      return;
    }

    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.messages.add({
        severity: 'warn',
        summary: 'Date incomplete',
        detail: 'Verifica campurile obligatorii inainte de salvare.',
      });
      return;
    }

    this.saving.set(true);
    try {
      await firstValueFrom(this.http.patch(`/api/education/portfolios/records/${current.id}`, this.buildPayload()));
      this.messages.add({
        severity: 'success',
        summary: 'Portofoliu salvat',
        detail: 'Modificarile au fost persistate si fluxul este actualizat.',
      });
      await this.router.navigate(['/education', 'portfolio', 'me']);
    } catch {
      this.messages.add({
        severity: 'error',
        summary: 'Salvarea a esuat',
        detail: 'Portofoliul nu a putut fi actualizat.',
      });
    } finally {
      this.saving.set(false);
    }
  }

  protected statusLabel(status: string): string {
    switch (status) {
      case 'submitted':
        return 'In verificare';
      case 'validated':
        return 'Validat';
      case 'returned':
        return 'Returnat';
      case 'archived':
        return 'Arhivat';
      default:
        return 'Draft';
    }
  }

  protected formatDateLabel(value: Date | null): string {
    if (!value) {
      return '-';
    }
    return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`;
  }

  private async loadCurrentPortfolio(ownerName: string): Promise<void> {
    this.loading.set(true);
    try {
      let params = new HttpParams().set('page', '1').set('pageSize', '5');
      if (ownerName.trim()) {
        params = params.set('filter.owner_name', ownerName.trim());
      }
      const response = await firstValueFrom(this.http.get<{ items: PortfolioWorkflowRecord[] }>('/api/education/portfolios/records', { params }));
      const current = response.items?.[0] ?? null;
      this.record.set(current);
      if (current) {
        this.form.patchValue({
          ownerName: current.owner_name,
          ownerRole: current.owner_role,
          schoolYear: current.school_year,
          status: current.status,
          sectionCount: current.section_count,
          lastUpdatedOn: this.parseDate(current.last_updated_on),
          retentionUntil: this.parseDate(current.retention_until),
          transferStatus: current.transfer_status,
          authenticityDeclared: current.authenticity_declared,
          consentCaptured: current.consent_captured,
          custodian: current.custodian,
          notes: current.notes,
        });
      }
    } catch {
      this.record.set(null);
    } finally {
      this.loading.set(false);
    }
  }

  private buildPayload(): Record<string, unknown> {
    const value = this.form.getRawValue();
    return {
      owner_name: value.ownerName?.trim() ?? '',
      owner_role: value.ownerRole?.trim() ?? '',
      school_year: value.schoolYear?.trim() ?? '',
      status: value.status,
      section_count: value.sectionCount,
      last_updated_on: this.formatDateLabel(value.lastUpdatedOn),
      retention_until: this.formatDateLabel(value.retentionUntil),
      transfer_status: value.transferStatus,
      authenticity_declared: value.authenticityDeclared,
      consent_captured: value.consentCaptured,
      custodian: value.custodian?.trim() ?? '',
      notes: value.notes?.trim() ?? '',
    };
  }

  private parseDate(value: string): Date {
    const parsed = new Date(`${value}T00:00:00`);
    return Number.isNaN(parsed.getTime()) ? new Date() : parsed;
  }

  private stepStatus(index: number): EducationWizardStepState['status'] {
    if (!this.record()) {
      return 'blocked';
    }
    if (index < this.activeStep()) {
      return 'completed';
    }
    if (index === this.activeStep()) {
      return 'active';
    }
    return 'pending';
  }
}
