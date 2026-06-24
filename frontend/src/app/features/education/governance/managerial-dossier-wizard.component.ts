import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
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

interface ManagerialWizardPayload {
  school_year: string;
  dossier_type: string;
  title: string;
  status: string;
  owner_name: string;
  due_on: string;
  publication_required: boolean;
  summary: string;
}

@Component({
  selector: 'app-managerial-dossier-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
    CheckboxModule,
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
              <i class="pi pi-briefcase"></i>
              Management educational
            </div>
            <h1 class="m-0 text-3xl font-semibold">{{ wizardTitle() }}</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              {{ wizardDescription() }}
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux managerial" status="active" />
            <p-button
              [routerLink]="['/education', 'governance']"
              label="Inapoi la guvernanta"
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
                  <h2 class="m-0 text-xl font-semibold">1. Identitate dosar</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Alegem anul scolar, tipul de dosar si titulatura de lucru pentru registrul managerial.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">An scolar</span>
                    <input pInputText formControlName="schoolYear" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Tip dosar</span>
                    <p-select
                      formControlName="dossierType"
                      [options]="dossierTypeOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Titlu dosar</span>
                    <input pInputText formControlName="title" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Responsabil si calendar</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim titularul operational si termenul orientativ pentru finalizarea dosarului.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Responsabil</span>
                    <input pInputText formControlName="ownerName" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Termen</span>
                    <p-datepicker formControlName="dueOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Rezumat</span>
                    <textarea pTextarea formControlName="summary" rows="5"></textarea>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Guvernanta documentului</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Marcam starea initiala si daca documentul trebuie inclus ulterior in circuitul de publicare.
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

                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3 md:col-span-2">
                    <p-checkbox formControlName="publicationRequired" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Documentul necesita publicare institutionala</div>
                      <p class="m-0 text-sm text-muted-color">
                        Activeaza urmarirea lui in registrul de publicari si in cockpitul directorului.
                      </p>
                    </div>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel
              title="4. Confirmare dosar managerial"
              description="Verificare finala inainte de a crea dosarul si de a-l face disponibil pentru documente si workflow."
              [items]="summaryItems()"
            />

            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button
                  label="Creeaza dosarul"
                  icon="pi pi-check"
                  [loading]="submitting()"
                  (onClick)="submit()"
                />
                <p-button
                  label="Anuleaza"
                  icon="pi pi-times"
                  severity="secondary"
                  [outlined]="true"
                  [routerLink]="['/education', 'governance']"
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
            description="Dosarul creat devine container pentru documente, etape de avizare si monitorizare operationala."
            [documents]="documentPreviewItems()"
          />

          <app-audit-trail-panel
            title="Traseu recomandat"
            description="Ordinea tipica dupa deschiderea unui dosar managerial."
            [events]="auditTrailItems()"
          />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ManagerialDossierWizardComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;
  protected readonly dossierTypeContext = signal<'generic' | 'director' | 'adjunct'>('generic');

  protected readonly wizardTitle = computed(() => {
    const context = this.dossierTypeContext();
    if (context === 'director') {
      return 'Wizard portofoliu director';
    }
    if (context === 'adjunct') {
      return 'Wizard portofoliu director adjunct';
    }
    return 'Wizard dosar managerial';
  });

  protected readonly wizardDescription = computed(() => {
    const context = this.dossierTypeContext();
    if (context === 'director') {
      return 'Flux ghidat pentru deschiderea portofoliului directorului, cu tipul de dosar preselectat si traseu clar pentru documente, avizare si publicare.';
    }
    if (context === 'adjunct') {
      return 'Flux ghidat pentru deschiderea portofoliului directorului adjunct, cu tipul de dosar preselectat si traseu clar pentru documente, avizare si publicare.';
    }
    return 'Flux ghidat pentru deschiderea unui dosar institutional: plan anual, PDI/PAS, raport sau alt document managerial urmarit in cockpitul directorului.';
  });

  protected readonly dossierTypeOptions = [
    { label: 'Plan anual', value: 'annual_plan' },
    { label: 'PDI / PAS', value: 'strategic_plan' },
    { label: 'Raport anual', value: 'annual_report' },
    { label: 'Raport de calitate', value: 'quality_report' },
    { label: 'Organigrama', value: 'organization_chart' },
    { label: 'Orar', value: 'schedule' },
    { label: 'Incadrare', value: 'staffing' },
    { label: 'Portofoliu director', value: 'director_portfolio' },
    { label: 'Portofoliu director adjunct', value: 'adjunct_director_portfolio' },
  ];

  protected readonly statusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'In avizare', value: 'review' },
    { label: 'In aprobare', value: 'approval' },
    { label: 'Aprobat', value: 'approved' },
    { label: 'Publicat', value: 'published' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    schoolYear: ['2025-2026', Validators.required],
    dossierType: ['annual_plan', Validators.required],
    title: ['', [Validators.required, Validators.minLength(5)]],
    ownerName: [''],
    dueOn: [new Date(), Validators.required],
    summary: [''],
    status: ['draft', Validators.required],
    publicationRequired: [false],
  });

  constructor() {
    const requestedDossierType = this.normalizeDossierType(this.route.snapshot.queryParamMap.get('dossierType'));
    if (requestedDossierType) {
      this.form.controls.dossierType.setValue(requestedDossierType);
      this.dossierTypeContext.set(
        requestedDossierType === 'director_portfolio'
          ? 'director'
          : requestedDossierType === 'adjunct_director_portfolio'
            ? 'adjunct'
            : 'generic',
      );
    }
  }

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'identity', label: 'Identitate', description: 'An, tip si titlu.', status: this.stepStatus(0) },
    { key: 'owner', label: 'Calendar', description: 'Responsabil si termen.', status: this.stepStatus(1) },
    { key: 'workflow', label: 'Workflow', description: 'Status si publicare.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'year', label: 'An scolar', value: this.form.controls.schoolYear.value || '-' },
    {
      key: 'type',
      label: 'Tip dosar',
      value: this.dossierTypeOptions.find((option) => option.value === this.form.controls.dossierType.value)?.label || '-',
    },
    { key: 'title', label: 'Titlu', value: this.form.controls.title.value || '-' },
    { key: 'owner', label: 'Responsabil', value: this.form.controls.ownerName.value || '-' },
    { key: 'due', label: 'Termen', value: this.formatDate(this.form.controls.dueOn.value) },
    {
      key: 'status',
      label: 'Status initial',
      value: this.statusOptions.find((option) => option.value === this.form.controls.status.value)?.label || '-',
    },
    {
      key: 'publication',
      label: 'Publicare necesara',
      value: this.form.controls.publicationRequired.value ? 'Da' : 'Nu',
    },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    {
      id: 'dossier',
      title: 'Dosar managerial',
      summary: 'Inregistrarea initiala pentru documente, versiuni, avizare si publicare.',
      status: 'active',
    },
    {
      id: 'workflow',
      title: 'Flux de aprobare',
      summary: 'Etapele ulterioare vor alimenta jurnalul de workflow si cockpitul directorului.',
      status: 'pending',
    },
    {
      id: 'publication',
      title: 'Registru publicare',
      summary: 'Poate fi urmarit pentru publicare institutionala daca documentul o cere.',
      status: this.form.controls.publicationRequired.value ? 'scheduled' : 'pending',
    },
  ]);

  protected readonly auditTrailItems = computed(() => [
    {
      id: 'trail-1',
      title: 'Creare dosar',
      summary: 'Directorul sau responsabilul deschide dosarul si ii fixeaza identitatea minima.',
      actorName: 'Director / Responsabil',
    },
    {
      id: 'trail-2',
      title: 'Completare documente',
      summary: 'Se incarca documente, versiuni si etape de lucru relevante pentru dosarul ales.',
      actorName: 'Responsabil dosar',
    },
    {
      id: 'trail-3',
      title: 'Avizare, aprobare si publicare',
      summary: 'Dosarul poate alimenta ulterior fluxurile de avizare si publicare institutionala.',
      actorName: 'Management educational',
    },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['schoolYear', 'dossierType', 'title']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['dueOn']);
    }
    if (this.currentStep() === 2) {
      return this.isStepValid(['status']);
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
        detail: 'Completeaza campurile obligatorii inainte de a crea dosarul managerial.',
      });
      return;
    }

    this.submitting.set(true);

    try {
      await firstValueFrom(
        this.http.post('/api/education/managerial/records', this.buildPayload()).pipe(finalize(() => this.submitting.set(false))),
      );

      this.messages.add({
        severity: 'success',
        summary: 'Dosarul a fost creat',
        detail: 'Inregistrarea manageriala este gata pentru documente si workflow.',
      });

      setTimeout(() => {
        void this.router.navigate(['/education', 'governance']);
      }, 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({
        severity: 'error',
        summary: 'Crearea a esuat',
        detail: 'Dosarul managerial nu a putut fi salvat. Verifica datele si incearca din nou.',
      });
    }
  }

  private buildPayload(): ManagerialWizardPayload {
    const value = this.form.getRawValue();
    return {
      school_year: value.schoolYear.trim(),
      dossier_type: value.dossierType,
      title: value.title.trim(),
      status: value.status,
      owner_name: value.ownerName.trim(),
      due_on: this.formatDate(value.dueOn),
      publication_required: value.publicationRequired,
      summary: value.summary.trim(),
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
      ['schoolYear', 'dossierType', 'title'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      this.form.controls.dueOn.markAsTouched();
      return;
    }
    if (this.currentStep() === 2) {
      this.form.controls.status.markAsTouched();
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

  private normalizeDossierType(value: string | null): string | null {
    if (!value) {
      return null;
    }

    const matchedOption = this.dossierTypeOptions.find((option) => option.value === value);
    return matchedOption ? String(matchedOption.value) : null;
  }
}
