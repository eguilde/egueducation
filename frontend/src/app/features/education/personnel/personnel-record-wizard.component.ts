import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
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

interface PersonnelWizardPayload {
  full_name: string;
  role_title: string;
  employment_type: string;
  status: string;
  evaluation_status: string;
  mobility_stage: string;
  school_year: string;
  assigned_unit: string;
  phone: string;
  email: string;
  has_portfolio: boolean;
  notes: string;
}

@Component({
  selector: 'app-personnel-record-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
    CheckboxModule,
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
              <i class="pi pi-id-card"></i>
              Management personal
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard cadru didactic</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru deschiderea unei fise de personal, cu incadrare, stare operationala si context pentru
              evaluare si portofoliu.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux HR" status="active" />
            <p-button
              [routerLink]="['/education', 'personnel']"
              label="Inapoi la personal"
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
                  <h2 class="m-0 text-xl font-semibold">1. Identitate si incadrare</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Inregistram datele de baza si tipul de incadrare pentru cadrul didactic.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Nume complet</span>
                    <input pInputText formControlName="fullName" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Functie</span>
                    <input pInputText formControlName="roleTitle" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Incadrare</span>
                    <p-select formControlName="employmentType" [options]="employmentOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">An scolar</span>
                    <input pInputText formControlName="schoolYear" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Structura</span>
                    <input pInputText formControlName="assignedUnit" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Status si urmarire</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim starea de lucru, evaluarea curenta si relatia cu mobilitatea si portofoliul.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status</span>
                    <p-select formControlName="status" [options]="statusOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Evaluare</span>
                    <p-select formControlName="evaluationStatus" [options]="evaluationStatusOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Etapa mobilitate</span>
                    <p-select formControlName="mobilityStage" [options]="mobilityOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>

                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3 md:col-span-2">
                    <p-checkbox formControlName="hasPortfolio" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Are portofoliu activ</div>
                      <p class="m-0 text-sm text-muted-color">
                        Marcheaza daca persoana are deja un portofoliu deschis sau daca acesta trebuie creat ulterior.
                      </p>
                    </div>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Contact si note</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Completam datele de contact si contextul operational minim pentru managementul scolii.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Telefon</span>
                    <input pInputText formControlName="phone" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Email</span>
                    <input pInputText formControlName="email" />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Note</span>
                    <textarea pTextarea formControlName="notes" rows="5"></textarea>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel
              title="4. Confirmare fisa de personal"
              description="Revizuire finala inainte de a crea inregistrarea cadrului didactic."
              [items]="summaryItems()"
            />
            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button label="Creeaza fisa" icon="pi pi-check" [loading]="submitting()" (onClick)="submit()" />
                <p-button label="Anuleaza" icon="pi pi-times" severity="secondary" [outlined]="true" [routerLink]="['/education', 'personnel']" />
              </div>
            </div>
          }

          <div class="flex flex-wrap justify-between gap-3">
            <p-button label="Pasul anterior" icon="pi pi-arrow-left" severity="secondary" [outlined]="true" [disabled]="currentStep() === 0 || submitting()" (onClick)="goPrevious()" />
            <p-button label="Pasul urmator" icon="pi pi-arrow-right" iconPos="right" [disabled]="currentStep() >= maxStepIndex || !canAdvance() || submitting()" (onClick)="goNext()" />
          </div>
        </form>

        <div class="grid gap-4">
          <app-document-preview-panel title="Pachet rezultat" description="Fisa de personal va putea alimenta portofolii, evaluari si fluxuri de mobilitate." [documents]="documentPreviewItems()" />
          <app-audit-trail-panel title="Traseu recomandat" description="Pasii urmatori dupa deschiderea unei fise de personal." [events]="auditTrailItems()" />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PersonnelRecordWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;

  protected readonly employmentOptions = [
    { label: 'Titular', value: 'titular' },
    { label: 'Suplinitor', value: 'suplinitor' },
    { label: 'Plata cu ora', value: 'hourly' },
    { label: 'Detasat', value: 'detasat' },
  ];
  protected readonly statusOptions = [
    { label: 'Activ', value: 'active' },
    { label: 'Suspendat', value: 'suspended' },
    { label: 'Detasat', value: 'detached' },
    { label: 'Inchis', value: 'closed' },
  ];
  protected readonly evaluationStatusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'In lucru', value: 'in_progress' },
    { label: 'Finalizata', value: 'finalized' },
    { label: 'Contestata', value: 'contested' },
  ];
  protected readonly mobilityOptions = [
    { label: 'Fara etapa', value: 'none' },
    { label: 'Pretransfer', value: 'pretransfer' },
    { label: 'Transfer', value: 'transfer' },
    { label: 'Restrangere', value: 'restrangere' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    fullName: ['', [Validators.required, Validators.minLength(3)]],
    roleTitle: ['', Validators.required],
    employmentType: ['titular', Validators.required],
    status: ['active', Validators.required],
    evaluationStatus: ['draft', Validators.required],
    mobilityStage: ['none', Validators.required],
    schoolYear: ['2025-2026', Validators.required],
    assignedUnit: [''],
    phone: [''],
    email: [''],
    hasPortfolio: [false],
    notes: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'identity', label: 'Identitate', description: 'Date si incadrare.', status: this.stepStatus(0) },
    { key: 'tracking', label: 'Urmarire', description: 'Status, evaluare si portofoliu.', status: this.stepStatus(1) },
    { key: 'contact', label: 'Contact', description: 'Date de contact si note.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'name', label: 'Nume', value: this.form.controls.fullName.value || '-' },
    { key: 'role', label: 'Functie', value: this.form.controls.roleTitle.value || '-' },
    { key: 'employment', label: 'Incadrare', value: this.optionLabel(this.employmentOptions, this.form.controls.employmentType.value) },
    { key: 'status', label: 'Status', value: this.optionLabel(this.statusOptions, this.form.controls.status.value) },
    { key: 'evaluation', label: 'Evaluare', value: this.optionLabel(this.evaluationStatusOptions, this.form.controls.evaluationStatus.value) },
    { key: 'mobility', label: 'Mobilitate', value: this.optionLabel(this.mobilityOptions, this.form.controls.mobilityStage.value) },
    { key: 'portfolio', label: 'Portofoliu', value: this.form.controls.hasPortfolio.value ? 'Da' : 'Nu' },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    { id: 'personnel', title: 'Fisa personala', summary: 'Inregistrarea de baza pentru cadrul didactic si fluxurile asociate.', status: 'active' },
    { id: 'evaluation', title: 'Context evaluare', summary: 'Poate alimenta ulterior evaluarea anuala si istoricul operational.', status: 'pending' },
    { id: 'portfolio', title: 'Legatura cu portofoliul', summary: 'Statusul de portofoliu ramane vizibil pentru cockpit si verificare.', status: 'scheduled' },
  ]);

  protected readonly auditTrailItems = computed(() => [
    { id: 'trail-1', title: 'Creare fisa de personal', summary: 'Directorul sau secretariatul deschide inregistrarea cadrului didactic.', actorName: 'Director / Secretariat' },
    { id: 'trail-2', title: 'Actualizare operationala', summary: 'Se completeaza datele de incadrare, evaluare si portofoliu pe parcursul anului.', actorName: 'Management personal' },
    { id: 'trail-3', title: 'Integrare cu evaluari', summary: 'Fisa poate fi folosita drept baza pentru evaluari si alte fluxuri pe rol.', actorName: 'Management educational' },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['fullName', 'roleTitle', 'employmentType', 'schoolYear']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['status', 'evaluationStatus', 'mobilityStage']);
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
    if (this.submitting()) return;
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      this.messages.add({ severity: 'warn', summary: 'Date incomplete', detail: 'Completeaza campurile obligatorii inainte de a crea fisa.' });
      return;
    }
    this.submitting.set(true);
    try {
      await firstValueFrom(this.http.post('/api/education/personnel/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))));
      this.messages.add({ severity: 'success', summary: 'Fisa a fost creata', detail: 'Inregistrarea cadrului didactic este disponibila in registru.' });
      setTimeout(() => void this.router.navigate(['/education', 'personnel']), 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({ severity: 'error', summary: 'Crearea a esuat', detail: 'Fisa de personal nu a putut fi salvata.' });
    }
  }

  private buildPayload(): PersonnelWizardPayload {
    const value = this.form.getRawValue();
    return {
      full_name: value.fullName.trim(),
      role_title: value.roleTitle.trim(),
      employment_type: value.employmentType,
      status: value.status,
      evaluation_status: value.evaluationStatus,
      mobility_stage: value.mobilityStage,
      school_year: value.schoolYear.trim(),
      assigned_unit: value.assignedUnit.trim(),
      phone: value.phone.trim(),
      email: value.email.trim(),
      has_portfolio: value.hasPortfolio,
      notes: value.notes.trim(),
    };
  }
  private stepStatus(index: number): EducationWizardStepState['status'] {
    if (index < this.currentStep()) return 'completed';
    if (index === this.currentStep()) return 'active';
    return 'pending';
  }
  private markCurrentStepTouched(): void {
    if (this.currentStep() === 0) {
      ['fullName', 'roleTitle', 'employmentType', 'schoolYear'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['status', 'evaluationStatus', 'mobilityStage'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
    }
  }
  private isStepValid(controlNames: string[]): boolean {
    return controlNames.every((name) => this.form.controls[name as keyof typeof this.form.controls].valid);
  }
  private optionLabel(options: Array<{ label: string; value: string }>, value: string): string {
    return options.find((option) => option.value === value)?.label ?? '-';
  }
}
