import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DatePickerModule } from 'primeng/datepicker';
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

interface DeclarationWizardPayload {
  employee_code: string;
  full_name: string;
  declaration_type: string;
  status: string;
  school_year: string;
  submitted_on: string;
  valid_until: string;
  summary: string;
}

@Component({
  selector: 'app-declaration-record-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
    DatePickerModule,
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
              <i class="pi pi-file-edit"></i>
              Declaratii si adeverinte
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard declaratie</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru inregistrarea unei declaratii institutionale asociate cadrului didactic.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux declaratii" status="active" />
            <p-button [routerLink]="['/education', 'personnel']" label="Inapoi la personal" icon="pi pi-arrow-left" severity="secondary" [outlined]="true" />
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
                  <h2 class="m-0 text-xl font-semibold">1. Titular si tip</h2>
                  <p class="m-0 text-sm text-muted-color">Stabilim persoana, tipul declaratiei si anul scolar.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Cod angajat</span>
                    <input pInputText formControlName="employeeCode" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">An scolar</span>
                    <input pInputText formControlName="schoolYear" />
                  </label>
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Nume complet</span>
                    <input pInputText formControlName="fullName" />
                  </label>
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Tip declaratie</span>
                    <p-select formControlName="declarationType" [options]="typeOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Status si calendar</h2>
                  <p class="m-0 text-sm text-muted-color">Definim starea documentului si intervalul lui de valabilitate.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status</span>
                    <p-select formControlName="status" [options]="statusOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Data depunerii</span>
                    <p-datepicker formControlName="submittedOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Valabila pana la</span>
                    <p-datepicker formControlName="validUntil" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Rezumat</h2>
                  <p class="m-0 text-sm text-muted-color">Notam pe scurt scopul sau contextul declaratiei.</p>
                </div>
                <label class="space-y-2">
                  <span class="text-sm font-medium">Rezumat</span>
                  <textarea pTextarea formControlName="summary" rows="5"></textarea>
                </label>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel title="4. Confirmare declaratie" description="Revizuire finala inainte de a crea declaratia." [items]="summaryItems()" />
            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button label="Creeaza declaratia" icon="pi pi-check" [loading]="submitting()" (onClick)="submit()" />
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
          <app-document-preview-panel title="Pachet rezultat" description="Declaratia va putea fi urmarita pentru expirare, revalidare si audit." [documents]="documentPreviewItems()" />
          <app-audit-trail-panel title="Traseu recomandat" description="Pasii operationali dupa inregistrarea declaratiei." [events]="auditTrailItems()" />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DeclarationRecordWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;

  protected readonly typeOptions = [
    { label: 'Autenticitate', value: 'authenticity' },
    { label: 'Consimtamant', value: 'consent' },
    { label: 'Conflict de interese', value: 'conflict_of_interest' },
    { label: 'Adeverinta', value: 'certificate' },
  ];
  protected readonly statusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'Depusa', value: 'submitted' },
    { label: 'Verificata', value: 'verified' },
    { label: 'Expirata', value: 'expired' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    employeeCode: ['', Validators.required],
    fullName: ['', [Validators.required, Validators.minLength(3)]],
    declarationType: ['authenticity', Validators.required],
    status: ['draft', Validators.required],
    schoolYear: ['2025-2026', Validators.required],
    submittedOn: [new Date(), Validators.required],
    validUntil: [new Date(new Date().getFullYear() + 1, 11, 31), Validators.required],
    summary: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'identity', label: 'Titular', description: 'Titular si tip.', status: this.stepStatus(0) },
    { key: 'calendar', label: 'Calendar', description: 'Status si valabilitate.', status: this.stepStatus(1) },
    { key: 'summary', label: 'Rezumat', description: 'Context operational.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);
  protected readonly summaryItems = computed(() => [
    { key: 'employee', label: 'Cod angajat', value: this.form.controls.employeeCode.value || '-' },
    { key: 'name', label: 'Nume', value: this.form.controls.fullName.value || '-' },
    { key: 'type', label: 'Tip', value: this.optionLabel(this.typeOptions, this.form.controls.declarationType.value) },
    { key: 'status', label: 'Status', value: this.optionLabel(this.statusOptions, this.form.controls.status.value) },
    { key: 'year', label: 'An scolar', value: this.form.controls.schoolYear.value || '-' },
  ]);
  protected readonly documentPreviewItems = computed(() => [
    { id: 'declaration', title: 'Inregistrare declaratie', summary: 'Documentul este disponibil pentru urmarire si audit institutional.', status: 'active' },
    { id: 'validity', title: 'Calendar valabilitate', summary: 'Datele de depunere si expirare pot alimenta alerte si revalidari.', status: 'scheduled' },
    { id: 'compliance', title: 'Conformitate personal', summary: 'Declaratia contribuie la istoricul de conformitate al cadrului didactic.', status: 'pending' },
  ]);
  protected readonly auditTrailItems = computed(() => [
    { id: 'trail-1', title: 'Creare declaratie', summary: 'Documentul este inregistrat pentru cadrul didactic selectat.', actorName: 'Secretariat / Management personal' },
    { id: 'trail-2', title: 'Verificare si evidenta', summary: 'Statusul poate evolua spre depus, verificat sau expirat.', actorName: 'Responsabil administrativ' },
    { id: 'trail-3', title: 'Audit si revalidare', summary: 'Declaratia ramane usor de gasit pentru audit si reinnoire.', actorName: 'Management educational' },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) return this.isStepValid(['employeeCode', 'fullName', 'declarationType', 'schoolYear']);
    if (this.currentStep() === 1) return this.isStepValid(['status', 'submittedOn', 'validUntil']);
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
      this.messages.add({ severity: 'warn', summary: 'Date incomplete', detail: 'Completeaza campurile obligatorii inainte de a crea declaratia.' });
      return;
    }
    this.submitting.set(true);
    try {
      await firstValueFrom(this.http.post('/api/education/declarations/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))));
      this.messages.add({ severity: 'success', summary: 'Declaratia a fost creata', detail: 'Inregistrarea declaratiei este disponibila in registru.' });
      setTimeout(() => void this.router.navigate(['/education', 'personnel']), 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({ severity: 'error', summary: 'Crearea a esuat', detail: 'Declaratia nu a putut fi salvata.' });
    }
  }
  private buildPayload(): DeclarationWizardPayload {
    const value = this.form.getRawValue();
    return {
      employee_code: value.employeeCode.trim(),
      full_name: value.fullName.trim(),
      declaration_type: value.declarationType,
      status: value.status,
      school_year: value.schoolYear.trim(),
      submitted_on: this.formatDate(value.submittedOn),
      valid_until: this.formatDate(value.validUntil),
      summary: value.summary.trim(),
    };
  }
  private stepStatus(index: number): EducationWizardStepState['status'] {
    if (index < this.currentStep()) return 'completed';
    if (index === this.currentStep()) return 'active';
    return 'pending';
  }
  private markCurrentStepTouched(): void {
    if (this.currentStep() === 0) {
      ['employeeCode', 'fullName', 'declarationType', 'schoolYear'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['status', 'submittedOn', 'validUntil'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
    }
  }
  private isStepValid(controlNames: string[]): boolean {
    return controlNames.every((name) => this.form.controls[name as keyof typeof this.form.controls].valid);
  }
  private optionLabel(options: Array<{ label: string; value: string }>, value: string): string {
    return options.find((option) => option.value === value)?.label ?? '-';
  }
  private formatDate(value: Date): string {
    const year = value.getFullYear();
    const month = `${value.getMonth() + 1}`.padStart(2, '0');
    const day = `${value.getDate()}`.padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
}
