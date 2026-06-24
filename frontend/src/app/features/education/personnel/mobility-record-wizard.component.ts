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

interface MobilityWizardPayload {
  employee_code: string;
  full_name: string;
  school_year: string;
  request_type: string;
  stage: string;
  status: string;
  source_school: string;
  destination_school: string;
  submitted_on: string;
  reviewed_by: string;
  notes: string;
}

@Component({
  selector: 'app-mobility-record-wizard',
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
              <i class="pi pi-directions-alt"></i>
              Mobilitate personal
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard caz de mobilitate</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru deschiderea unui caz de mobilitate cu traseu procedural, unitati implicate si responsabil de analiza.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux mobilitate" status="active" />
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
                  <h2 class="m-0 text-xl font-semibold">1. Titular si solicitare</h2>
                  <p class="m-0 text-sm text-muted-color">Definim persoana, anul scolar si tipul solicitarii de mobilitate.</p>
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
                    <span class="text-sm font-medium">Tip solicitare</span>
                    <p-select formControlName="requestType" [options]="requestTypeOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Etapa si status</h2>
                  <p class="m-0 text-sm text-muted-color">Stabilim stadiul procedural si statusul operational al cazului.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Etapa</span>
                    <p-select formControlName="stage" [options]="stageOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status</span>
                    <p-select formControlName="status" [options]="statusOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Data depunerii</span>
                    <p-datepicker formControlName="submittedOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Analizat de</span>
                    <input pInputText formControlName="reviewedBy" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Unitati si observatii</h2>
                  <p class="m-0 text-sm text-muted-color">Notam unitatea sursa, destinatia si contextul procedural minim.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Unitate sursa</span>
                    <input pInputText formControlName="sourceSchool" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Unitate destinatie</span>
                    <input pInputText formControlName="destinationSchool" />
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
            <app-wizard-summary-panel title="4. Confirmare caz de mobilitate" description="Revizuire finala inainte de a crea cazul de mobilitate." [items]="summaryItems()" />
            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button label="Creeaza cazul" icon="pi pi-check" [loading]="submitting()" (onClick)="submit()" />
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
          <app-document-preview-panel title="Pachet rezultat" description="Cazul creat va putea alimenta documente, punctaje, contestatii si decizii finale." [documents]="documentPreviewItems()" />
          <app-audit-trail-panel title="Traseu recomandat" description="Secventa tipica dupa deschiderea unui caz de mobilitate." [events]="auditTrailItems()" />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MobilityRecordWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;

  protected readonly requestTypeOptions = [
    { label: 'Transfer', value: 'transfer' },
    { label: 'Pretransfer', value: 'pretransfer' },
    { label: 'Detasare', value: 'detasare' },
    { label: 'Restrangere activitate', value: 'restrangere' },
  ];
  protected readonly stageOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'Depus', value: 'submitted' },
    { label: 'Analiza', value: 'review' },
    { label: 'Decizie', value: 'decision' },
  ];
  protected readonly statusOptions = [
    { label: 'Deschis', value: 'open' },
    { label: 'In lucru', value: 'in_progress' },
    { label: 'Finalizat', value: 'closed' },
    { label: 'Respins', value: 'rejected' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    employeeCode: ['', Validators.required],
    fullName: ['', [Validators.required, Validators.minLength(3)]],
    schoolYear: ['2025-2026', Validators.required],
    requestType: ['transfer', Validators.required],
    stage: ['draft', Validators.required],
    status: ['open', Validators.required],
    sourceSchool: [''],
    destinationSchool: [''],
    submittedOn: [new Date(), Validators.required],
    reviewedBy: [''],
    notes: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'identity', label: 'Titular', description: 'Titular si solicitare.', status: this.stepStatus(0) },
    { key: 'tracking', label: 'Status', description: 'Etapa si analiza.', status: this.stepStatus(1) },
    { key: 'schools', label: 'Unitati', description: 'Sursa, destinatie, note.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'employee', label: 'Cod angajat', value: this.form.controls.employeeCode.value || '-' },
    { key: 'name', label: 'Nume', value: this.form.controls.fullName.value || '-' },
    { key: 'type', label: 'Tip solicitare', value: this.optionLabel(this.requestTypeOptions, this.form.controls.requestType.value) },
    { key: 'stage', label: 'Etapa', value: this.optionLabel(this.stageOptions, this.form.controls.stage.value) },
    { key: 'status', label: 'Status', value: this.optionLabel(this.statusOptions, this.form.controls.status.value) },
    { key: 'destination', label: 'Destinatie', value: this.form.controls.destinationSchool.value || '-' },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    { id: 'mobility', title: 'Dosar mobilitate', summary: 'Cazul de baza pentru documente, punctaj, contestatii si decizie.', status: 'active' },
    { id: 'workflow', title: 'Flux procedural', summary: 'Etapa si statusul pot alimenta un traseu de avizare si decizie.', status: 'pending' },
    { id: 'reporting', title: 'Raportare operationala', summary: 'Cazul va putea fi inclus in cockpit si in rapoarte de personal.', status: 'scheduled' },
  ]);

  protected readonly auditTrailItems = computed(() => [
    { id: 'trail-1', title: 'Deschidere caz', summary: 'Cazul de mobilitate este creat pentru cadrul didactic selectat.', actorName: 'Director / Secretariat' },
    { id: 'trail-2', title: 'Analiza si documente', summary: 'Se completeaza actele, punctajele si eventualele observatii sau contestatii.', actorName: 'Comisie / Management personal' },
    { id: 'trail-3', title: 'Decizie finala', summary: 'Cazul poate ajunge la decizie si la comunicarea rezultatului.', actorName: 'Management educational' },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) return this.isStepValid(['employeeCode', 'fullName', 'schoolYear', 'requestType']);
    if (this.currentStep() === 1) return this.isStepValid(['stage', 'status', 'submittedOn']);
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
      this.messages.add({ severity: 'warn', summary: 'Date incomplete', detail: 'Completeaza campurile obligatorii inainte de a crea cazul.' });
      return;
    }
    this.submitting.set(true);
    try {
      await firstValueFrom(this.http.post('/api/education/mobility/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))));
      this.messages.add({ severity: 'success', summary: 'Cazul a fost creat', detail: 'Inregistrarea de mobilitate este disponibila in registru.' });
      setTimeout(() => void this.router.navigate(['/education', 'personnel']), 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({ severity: 'error', summary: 'Crearea a esuat', detail: 'Cazul de mobilitate nu a putut fi salvat.' });
    }
  }
  private buildPayload(): MobilityWizardPayload {
    const value = this.form.getRawValue();
    return {
      employee_code: value.employeeCode.trim(),
      full_name: value.fullName.trim(),
      school_year: value.schoolYear.trim(),
      request_type: value.requestType,
      stage: value.stage,
      status: value.status,
      source_school: value.sourceSchool.trim(),
      destination_school: value.destinationSchool.trim(),
      submitted_on: this.formatDate(value.submittedOn),
      reviewed_by: value.reviewedBy.trim(),
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
      ['employeeCode', 'fullName', 'schoolYear', 'requestType'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['stage', 'status', 'submittedOn'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
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
