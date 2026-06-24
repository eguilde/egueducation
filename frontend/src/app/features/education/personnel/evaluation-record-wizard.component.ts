import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
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

interface EvaluationWizardPayload {
  employee_code: string;
  full_name: string;
  role_title: string;
  school_year: string;
  status: string;
  score: number;
  qualification: string;
  evaluator_name: string;
  finalized_on: string;
  summary: string;
}

@Component({
  selector: 'app-evaluation-record-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
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
              <i class="pi pi-list-check"></i>
              Evaluare profesionala
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard evaluare anuala</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru deschiderea unei evaluari anuale, cu punctaj, calificativ, evaluator si trasabilitate.
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux evaluare" status="active" />
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
                  <h2 class="m-0 text-xl font-semibold">1. Cadru evaluat</h2>
                  <p class="m-0 text-sm text-muted-color">Stabilim persoana evaluata, functia si anul scolar.</p>
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
                    <span class="text-sm font-medium">Functie</span>
                    <input pInputText formControlName="roleTitle" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Rezultat si status</h2>
                  <p class="m-0 text-sm text-muted-color">Definim starea evaluarii, punctajul si calificativul initial.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status</span>
                    <p-select formControlName="status" [options]="statusOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Calificativ</span>
                    <p-select formControlName="qualification" [options]="qualificationOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Punctaj</span>
                    <p-inputnumber formControlName="score" inputId="score" [min]="0" [max]="100" [minFractionDigits]="0" [maxFractionDigits]="2" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Evaluator si rezumat</h2>
                  <p class="m-0 text-sm text-muted-color">Completam evaluatorul, data finalizarii si observatiile sintetice.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Evaluator</span>
                    <input pInputText formControlName="evaluatorName" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Data finalizarii</span>
                    <p-datepicker formControlName="finalizedOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Rezumat</span>
                    <textarea pTextarea formControlName="summary" rows="5"></textarea>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel title="4. Confirmare evaluare" description="Revizuire finala inainte de a crea evaluarea anuala." [items]="summaryItems()" />
            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button label="Creeaza evaluarea" icon="pi pi-check" [loading]="submitting()" (onClick)="submit()" />
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
          <app-document-preview-panel title="Pachet rezultat" description="Evaluarea poate alimenta fisa PDF, contestatiile si cockpitul directorului." [documents]="documentPreviewItems()" />
          <app-audit-trail-panel title="Traseu recomandat" description="Pasii operationali dupa deschiderea evaluarii." [events]="auditTrailItems()" />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EvaluationRecordWizardComponent {
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
    { label: 'In evaluare', value: 'in_review' },
    { label: 'Finalizata', value: 'finalized' },
    { label: 'Contestata', value: 'contested' },
  ];
  protected readonly qualificationOptions = [
    { label: 'Foarte bine', value: 'foarte_bine' },
    { label: 'Bine', value: 'bine' },
    { label: 'Satisfacator', value: 'satisfacator' },
    { label: 'Nesatisfacator', value: 'nesatisfacator' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    employeeCode: ['', Validators.required],
    fullName: ['', [Validators.required, Validators.minLength(3)]],
    roleTitle: ['', Validators.required],
    schoolYear: ['2025-2026', Validators.required],
    status: ['draft', Validators.required],
    score: [0, Validators.required],
    qualification: ['foarte_bine', Validators.required],
    evaluatorName: [''],
    finalizedOn: [new Date(), Validators.required],
    summary: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'employee', label: 'Cadru', description: 'Persoana evaluata.', status: this.stepStatus(0) },
    { key: 'result', label: 'Rezultat', description: 'Status, punctaj, calificativ.', status: this.stepStatus(1) },
    { key: 'evaluator', label: 'Evaluator', description: 'Evaluator si rezumat.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'employee', label: 'Cadru evaluat', value: this.form.controls.fullName.value || '-' },
    { key: 'code', label: 'Cod angajat', value: this.form.controls.employeeCode.value || '-' },
    { key: 'role', label: 'Functie', value: this.form.controls.roleTitle.value || '-' },
    { key: 'status', label: 'Status', value: this.optionLabel(this.statusOptions, this.form.controls.status.value) },
    { key: 'qualification', label: 'Calificativ', value: this.optionLabel(this.qualificationOptions, this.form.controls.qualification.value) },
    { key: 'score', label: 'Punctaj', value: String(this.form.controls.score.value) },
  ]);
  protected readonly documentPreviewItems = computed(() => [
    { id: 'evaluation', title: 'Fisa de evaluare', summary: 'Inregistrarea initiala pentru calificativ, punctaj si traseu procedural.', status: 'active' },
    { id: 'appeal', title: 'Contestatii', summary: 'Poate alimenta ulterior un flux de contestare si comunicare a rezultatului.', status: 'pending' },
    { id: 'dashboard', title: 'Cockpit director', summary: 'Statusul evaluarii contribuie la sumarul operational si la alerte.', status: 'scheduled' },
  ]);
  protected readonly auditTrailItems = computed(() => [
    { id: 'trail-1', title: 'Deschidere evaluare', summary: 'Managerul creeaza evaluarea anuala pentru cadrul didactic.', actorName: 'Director / Evaluator' },
    { id: 'trail-2', title: 'Completare criterii', summary: 'Se completeaza criterii, autoevaluare si punctaje detaliate.', actorName: 'Evaluator' },
    { id: 'trail-3', title: 'Comunicare rezultat', summary: 'Rezultatul poate fi finalizat, comunicat si eventual contestat.', actorName: 'Management personal' },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) return this.isStepValid(['employeeCode', 'fullName', 'roleTitle', 'schoolYear']);
    if (this.currentStep() === 1) return this.isStepValid(['status', 'score', 'qualification']);
    if (this.currentStep() === 2) return this.isStepValid(['finalizedOn']);
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
      this.messages.add({ severity: 'warn', summary: 'Date incomplete', detail: 'Completeaza campurile obligatorii inainte de a crea evaluarea.' });
      return;
    }
    this.submitting.set(true);
    try {
      await firstValueFrom(this.http.post('/api/education/evaluations/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))));
      this.messages.add({ severity: 'success', summary: 'Evaluarea a fost creata', detail: 'Inregistrarea evaluarii este disponibila in registru.' });
      setTimeout(() => void this.router.navigate(['/education', 'personnel']), 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({ severity: 'error', summary: 'Crearea a esuat', detail: 'Evaluarea nu a putut fi salvata.' });
    }
  }
  private buildPayload(): EvaluationWizardPayload {
    const value = this.form.getRawValue();
    return {
      employee_code: value.employeeCode.trim(),
      full_name: value.fullName.trim(),
      role_title: value.roleTitle.trim(),
      school_year: value.schoolYear.trim(),
      status: value.status,
      score: value.score,
      qualification: value.qualification,
      evaluator_name: value.evaluatorName.trim(),
      finalized_on: this.formatDate(value.finalizedOn),
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
      ['employeeCode', 'fullName', 'roleTitle', 'schoolYear'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['status', 'score', 'qualification'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 2) {
      this.form.controls.finalizedOn.markAsTouched();
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
