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

interface MeritWizardPayload {
  full_name: string;
  role_title: string;
  school_year: string;
  category: string;
  status: string;
  score: number;
  committee_name: string;
  decision_date: string;
  funded: boolean;
  notes: string;
}

@Component({
  selector: 'app-merit-record-wizard',
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
              <i class="pi pi-star"></i>
              Gradatii de merit
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard dosar gradatie</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru deschiderea unui dosar de gradatie de merit cu punctaj, comisie si stare de finantare.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux gradatii" status="active" />
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
                  <h2 class="m-0 text-xl font-semibold">1. Candidat si categorie</h2>
                  <p class="m-0 text-sm text-muted-color">Definim cadrul didactic, functia, anul si categoria dosarului.</p>
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
                    <span class="text-sm font-medium">An scolar</span>
                    <input pInputText formControlName="schoolYear" />
                  </label>
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Categorie</span>
                    <p-select formControlName="category" [options]="categoryOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Evaluare si comisie</h2>
                  <p class="m-0 text-sm text-muted-color">Stabilim statusul, punctajul si comisia responsabila.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status</span>
                    <p-select formControlName="status" [options]="statusOptions" optionLabel="label" optionValue="value" appendTo="body" />
                  </label>
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Punctaj</span>
                    <p-inputnumber formControlName="score" inputId="score" [min]="0" [max]="100" [minFractionDigits]="0" [maxFractionDigits]="2" />
                  </label>
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Comisie</span>
                    <input pInputText formControlName="committeeName" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Decizie si finantare</h2>
                  <p class="m-0 text-sm text-muted-color">Completam data deciziei, starea de finantare si observatiile initiale.</p>
                </div>
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Data deciziei</span>
                    <p-datepicker formControlName="decisionDate" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>
                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                    <p-checkbox formControlName="funded" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Include finantare</div>
                      <p class="m-0 text-sm text-muted-color">Marcheaza daca dosarul este asociat unei finantari aprobate.</p>
                    </div>
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
            <app-wizard-summary-panel title="4. Confirmare dosar gradatie" description="Revizuire finala inainte de a crea dosarul." [items]="summaryItems()" />
            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button label="Creeaza dosarul" icon="pi pi-check" [loading]="submitting()" (onClick)="submit()" />
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
          <app-document-preview-panel title="Pachet rezultat" description="Dosarul creat poate alimenta scoruri detaliate, contestatii si decizii finale." [documents]="documentPreviewItems()" />
          <app-audit-trail-panel title="Traseu recomandat" description="Secventa tipica dupa deschiderea unui dosar de gradatie." [events]="auditTrailItems()" />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MeritRecordWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;

  protected readonly categoryOptions = [
    { label: 'Predare', value: 'predare' },
    { label: 'Conducere', value: 'conducere' },
    { label: 'Indrumare si control', value: 'control' },
    { label: 'Didactic auxiliar', value: 'auxiliar' },
  ];
  protected readonly statusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'Depus', value: 'submitted' },
    { label: 'Evaluat', value: 'evaluated' },
    { label: 'Finantat', value: 'funded' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    fullName: ['', [Validators.required, Validators.minLength(3)]],
    roleTitle: ['', Validators.required],
    schoolYear: ['2025-2026', Validators.required],
    category: ['predare', Validators.required],
    status: ['draft', Validators.required],
    score: [0, Validators.required],
    committeeName: [''],
    decisionDate: [new Date(), Validators.required],
    funded: [false],
    notes: [''],
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'candidate', label: 'Candidat', description: 'Nume, functie, categorie.', status: this.stepStatus(0) },
    { key: 'scoring', label: 'Evaluare', description: 'Status, punctaj, comisie.', status: this.stepStatus(1) },
    { key: 'decision', label: 'Decizie', description: 'Data, finantare, note.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'name', label: 'Nume', value: this.form.controls.fullName.value || '-' },
    { key: 'role', label: 'Functie', value: this.form.controls.roleTitle.value || '-' },
    { key: 'category', label: 'Categorie', value: this.optionLabel(this.categoryOptions, this.form.controls.category.value) },
    { key: 'status', label: 'Status', value: this.optionLabel(this.statusOptions, this.form.controls.status.value) },
    { key: 'score', label: 'Punctaj', value: String(this.form.controls.score.value) },
    { key: 'funded', label: 'Finantat', value: this.form.controls.funded.value ? 'Da' : 'Nu' },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    { id: 'merit', title: 'Dosar gradatie', summary: 'Inregistrarea de baza pentru evaluare, punctaj si finantare.', status: 'active' },
    { id: 'appeals', title: 'Contestatii si clarificari', summary: 'Poate fi extins ulterior cu contestatii si comunicari oficiale.', status: 'pending' },
    { id: 'reporting', title: 'Raportare manageriala', summary: 'Datele pot alimenta cockpitul si rapoartele pe personal.', status: 'scheduled' },
  ]);

  protected readonly auditTrailItems = computed(() => [
    { id: 'trail-1', title: 'Deschidere dosar', summary: 'Dosarul de gradatie este creat pentru candidatul selectat.', actorName: 'Director / Secretariat' },
    { id: 'trail-2', title: 'Evaluare si punctare', summary: 'Comisia poate completa punctaje si documente justificative.', actorName: 'Comisie evaluare' },
    { id: 'trail-3', title: 'Decizie si finantare', summary: 'Dosarul poate merge spre decizie, finantare si comunicare.', actorName: 'Management educational' },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) return this.isStepValid(['fullName', 'roleTitle', 'schoolYear', 'category']);
    if (this.currentStep() === 1) return this.isStepValid(['status', 'score']);
    if (this.currentStep() === 2) return this.isStepValid(['decisionDate']);
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
      this.messages.add({ severity: 'warn', summary: 'Date incomplete', detail: 'Completeaza campurile obligatorii inainte de a crea dosarul.' });
      return;
    }
    this.submitting.set(true);
    try {
      await firstValueFrom(this.http.post('/api/education/gradatii/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))));
      this.messages.add({ severity: 'success', summary: 'Dosarul a fost creat', detail: 'Inregistrarea gradatiei este disponibila in registru.' });
      setTimeout(() => void this.router.navigate(['/education', 'personnel']), 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({ severity: 'error', summary: 'Crearea a esuat', detail: 'Dosarul de gradatie nu a putut fi salvat.' });
    }
  }
  private buildPayload(): MeritWizardPayload {
    const value = this.form.getRawValue();
    return {
      full_name: value.fullName.trim(),
      role_title: value.roleTitle.trim(),
      school_year: value.schoolYear.trim(),
      category: value.category,
      status: value.status,
      score: value.score,
      committee_name: value.committeeName.trim(),
      decision_date: this.formatDate(value.decisionDate),
      funded: value.funded,
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
      ['fullName', 'roleTitle', 'schoolYear', 'category'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['status', 'score'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 2) {
      this.form.controls.decisionDate.markAsTouched();
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
