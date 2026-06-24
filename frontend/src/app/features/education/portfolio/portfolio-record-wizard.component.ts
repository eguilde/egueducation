import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
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
import { finalize, firstValueFrom } from 'rxjs';

import { AuthzService } from '../../../core/authz/authz.service';
import { AuditTrailPanelComponent } from '../shared/wizard/audit-trail-panel.component';
import { DeadlineBadgeComponent } from '../shared/wizard/deadline-badge.component';
import { DocumentPreviewPanelComponent } from '../shared/wizard/document-preview-panel.component';
import { WizardSummaryPanelComponent } from '../shared/wizard/wizard-summary-panel.component';
import { WizardStepperComponent } from '../shared/wizard/wizard-stepper.component';
import { EducationWizardStepState } from '../shared/wizard/wizard.models';

interface PortfolioWizardPayload {
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
  selector: 'app-portfolio-record-wizard',
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
    AuditTrailPanelComponent,
    DeadlineBadgeComponent,
    DocumentPreviewPanelComponent,
    WizardStepperComponent,
    WizardSummaryPanelComponent,
  ],
  providers: [MessageService],
  template: `
    <p-toast />

    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-folder-open"></i>
              Portofolii profesionale
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard portofoliu cadru didactic</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru deschiderea unui portofoliu CD, cu retentie, trasabilitate si controale minime de
              autenticitate si consimtamant.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux portofoliu" status="active" />
            <p-button
              [routerLink]="['/education', 'portfolio']"
              label="Inapoi la portofolii"
              icon="pi pi-arrow-left"
              severity="secondary"
              [outlined]="true"
            />
          </div>
        </div>
      </div>

      <app-wizard-stepper [steps]="steps()" />

      <div class="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
        <form class="space-y-4" [formGroup]="form">
          @if (currentStep() === 0) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">1. Titular si an scolar</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim cui apartine portofoliul si in ce an scolar este urmarit procedural.
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

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Structura si retentie</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Alegem statusul initial, structura estimata si calendarul minim pentru retentie si actualizare.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status initial</span>
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
                    <span class="text-sm font-medium">Numar estimat sectiuni</span>
                    <p-inputnumber formControlName="sectionCount" inputId="sectionCount" [min]="0" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Custode</span>
                    <input pInputText formControlName="custodian" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Ultima actualizare</span>
                    <p-datepicker formControlName="lastUpdatedOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Retentie pana la</span>
                    <p-datepicker formControlName="retentionUntil" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Declaratii si note</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Bifam elementele minime de conformitate si notele de pornire ale portofoliului.
                  </p>
                </div>

                <div class="grid gap-3">
                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                    <p-checkbox formControlName="authenticityDeclared" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Declaratie de autenticitate disponibila</div>
                      <p class="m-0 text-sm text-muted-color">
                        Portofoliul poate porni cu confirmarea ca documentele vor fi asumate de titular.
                      </p>
                    </div>
                  </label>

                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                    <p-checkbox formControlName="consentCaptured" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Consimtamant / informare GDPR capturat(a)</div>
                      <p class="m-0 text-sm text-muted-color">
                        Marcheaza ca titularul a fost informat in legatura cu prelucrarea datelor relevante.
                      </p>
                    </div>
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Note initiale</span>
                    <textarea pTextarea formControlName="notes" rows="5"></textarea>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel
              title="4. Confirmare portofoliu"
              description="Revizuire finala inainte de a crea portofoliul si de a porni checklist-ul de completare."
              [items]="summaryItems()"
            />

            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button
                  label="Creeaza portofoliul"
                  icon="pi pi-check"
                  [loading]="submitting()"
                  (onClick)="submit()"
                />
                <p-button
                  label="Anuleaza"
                  icon="pi pi-times"
                  severity="secondary"
                  [outlined]="true"
                  [routerLink]="['/education', 'portfolio']"
                />
              </div>
            </div>
          }

          <div class="flex flex-wrap justify-between gap-3">
            <p-button
              label="Pasul anterior"
              icon="pi pi-arrow-left"
              severity="secondary"
              [outlined]="true"
              [disabled]="currentStep() === 0 || submitting()"
              (onClick)="goPrevious()"
            />

            <p-button
              label="Pasul urmator"
              icon="pi pi-arrow-right"
              iconPos="right"
              [disabled]="currentStep() >= maxStepIndex || !canAdvance() || submitting()"
              (onClick)="goNext()"
            />
          </div>
        </form>

        <div class="grid gap-4">
          <app-document-preview-panel
            title="Pachet rezultat"
            description="Portofoliul creat pregateste descrierea de baza pentru opis, checklist, transfer si revizuiri."
            [documents]="documentPreviewItems()"
          />

          <app-audit-trail-panel
            title="Traseu recomandat"
            description="Pasii operationali de urmat imediat dupa deschiderea portofoliului."
            [events]="auditTrailItems()"
          />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PortfolioRecordWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;

  protected readonly statusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'In completare', value: 'in_progress' },
    { label: 'In verificare', value: 'review' },
    { label: 'Validat', value: 'validated' },
    { label: 'Returnat', value: 'returned' },
  ];

  protected readonly transferOptions = [
    { label: 'Fara transfer', value: 'none' },
    { label: 'Predare', value: 'handover' },
    { label: 'Preluare', value: 'takeover' },
    { label: 'Finalizat', value: 'completed' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    ownerName: ['', [Validators.required, Validators.minLength(3)]],
    ownerRole: ['', Validators.required],
    schoolYear: ['2025-2026', Validators.required],
    status: ['draft', Validators.required],
    sectionCount: [0, Validators.required],
    lastUpdatedOn: [new Date(), Validators.required],
    retentionUntil: [new Date(new Date().getFullYear() + 2, 7, 31), Validators.required],
    transferStatus: ['none', Validators.required],
    authenticityDeclared: [false],
    consentCaptured: [false],
    custodian: [''],
    notes: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'owner', label: 'Titular', description: 'Titular si an scolar.', status: this.stepStatus(0) },
    { key: 'structure', label: 'Structura', description: 'Status, transfer si retentie.', status: this.stepStatus(1) },
    { key: 'compliance', label: 'Conformitate', description: 'Declaratii si note.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'owner', label: 'Titular', value: this.form.controls.ownerName.value || '-' },
    { key: 'role', label: 'Functie', value: this.form.controls.ownerRole.value || '-' },
    { key: 'year', label: 'An scolar', value: this.form.controls.schoolYear.value || '-' },
    {
      key: 'status',
      label: 'Status',
      value: this.statusOptions.find((option) => option.value === this.form.controls.status.value)?.label || '-',
    },
    {
      key: 'transfer',
      label: 'Transfer',
      value: this.transferOptions.find((option) => option.value === this.form.controls.transferStatus.value)?.label || '-',
    },
    { key: 'sections', label: 'Sectiuni estimate', value: String(this.form.controls.sectionCount.value) },
    { key: 'custodian', label: 'Custode', value: this.form.controls.custodian.value || '-' },
    { key: 'auth', label: 'Autenticitate', value: this.form.controls.authenticityDeclared.value ? 'Da' : 'Nu' },
    { key: 'consent', label: 'Consimtamant', value: this.form.controls.consentCaptured.value ? 'Da' : 'Nu' },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    {
      id: 'portfolio',
      title: 'Inregistrare portofoliu',
      summary: 'Punct de plecare pentru opis, checklist, documente si revizuiri institutionale.',
      status: 'active',
    },
    {
      id: 'checklist',
      title: 'Checklist de completare',
      summary: 'Poate fi completat ulterior pentru urmarirea lipsurilor si a probelor documentare.',
      status: 'pending',
    },
    {
      id: 'retention',
      title: 'Retentie si transfer',
      summary: 'Datele setate aici vor ajuta la controlul custodiei si al transferurilor.',
      status: 'scheduled',
    },
  ]);

  protected readonly auditTrailItems = computed(() => [
    {
      id: 'trail-1',
      title: 'Deschidere portofoliu',
      summary: 'Secretariatul sau directorul creeaza portofoliul initial pentru cadrul didactic.',
      actorName: 'Director / Secretariat',
    },
    {
      id: 'trail-2',
      title: 'Completare si opis',
      summary: 'Titularul si administratorii documenteaza continutul prin sectiuni, opis si documente atasate.',
      actorName: 'Titular portofoliu',
    },
    {
      id: 'trail-3',
      title: 'Review si transfer',
      summary: 'Portofoliul poate intra apoi in verificare, returnare pentru completari sau transfer procedural.',
      actorName: 'Management educational',
    },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['ownerName', 'ownerRole', 'schoolYear']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['status', 'sectionCount', 'lastUpdatedOn', 'retentionUntil', 'transferStatus']);
    }
    return true;
  }

  protected goNext(): void {
    if (!this.canAdvance() || this.currentStep() >= this.maxStepIndex) {
      this.markCurrentStepTouched();
      return;
    }
    this.currentStep.update((value) => Math.min(this.maxStepIndex, value + 1));
  }

  protected goPrevious(): void {
    this.currentStep.update((value) => Math.max(0, value - 1));
  }

  protected async submit(): Promise<void> {
    if (this.submitting()) {
      return;
    }

    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.messages.add({
        severity: 'warn',
        summary: 'Date incomplete',
        detail: 'Completeaza campurile obligatorii inainte de a crea portofoliul.',
      });
      return;
    }

    this.submitting.set(true);

    try {
      await firstValueFrom(
        this.http.post('/api/education/portfolios/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))),
      );

      this.messages.add({
        severity: 'success',
        summary: 'Portofoliul a fost creat',
        detail: 'Inregistrarea este gata pentru completare, opis si review institutional.',
      });

      setTimeout(() => {
        void this.router.navigate(['/education', 'portfolio']);
      }, 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({
        severity: 'error',
        summary: 'Crearea a esuat',
        detail: 'Portofoliul nu a putut fi salvat. Verifica datele si incearca din nou.',
      });
    }
  }

  private buildPayload(): PortfolioWizardPayload {
    const value = this.form.getRawValue();
    return {
      owner_name: value.ownerName.trim(),
      owner_role: value.ownerRole.trim(),
      school_year: value.schoolYear.trim(),
      status: value.status,
      section_count: value.sectionCount,
      last_updated_on: this.formatDate(value.lastUpdatedOn),
      retention_until: this.formatDate(value.retentionUntil),
      transfer_status: value.transferStatus,
      authenticity_declared: value.authenticityDeclared,
      consent_captured: value.consentCaptured,
      custodian: value.custodian.trim(),
      notes: value.notes.trim(),
    };
  }

  private stepStatus(index: number): EducationWizardStepState['status'] {
    if (index < this.currentStep()) {
      return 'completed';
    }
    if (index === this.currentStep()) {
      return 'active';
    }
    return 'pending';
  }

  private markCurrentStepTouched(): void {
    if (this.currentStep() === 0) {
      ['ownerName', 'ownerRole', 'schoolYear'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['status', 'sectionCount', 'lastUpdatedOn', 'retentionUntil', 'transferStatus'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
    }
  }

  private isStepValid(controlNames: string[]): boolean {
    return controlNames.every((name) => this.form.controls[name as keyof typeof this.form.controls].valid);
  }

  private formatDate(value: Date): string {
    const year = value.getFullYear();
    const month = `${value.getMonth() + 1}`.padStart(2, '0');
    const day = `${value.getDate()}`.padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
}
