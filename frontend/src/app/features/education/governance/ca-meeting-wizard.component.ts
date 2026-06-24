import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
import { DatePickerModule } from 'primeng/datepicker';
import { DividerModule } from 'primeng/divider';
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

interface MeetingWizardPayload {
  school_year: string;
  organism: string;
  title: string;
  meeting_type: string;
  status: string;
  quorum_required: number;
  participants_count: number;
  meeting_date: string;
  location: string;
  chairperson: string;
  secretary_name: string;
  summary: string;
}

@Component({
  selector: 'app-ca-meeting-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
    CheckboxModule,
    DatePickerModule,
    DividerModule,
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
              <i class="pi pi-sitemap"></i>
              Guvernanta educationala
            </div>
            <h1 class="m-0 text-3xl font-semibold">{{ wizardTitle() }}</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              {{ wizardDescription() }}
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux pilot" status="active" />
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
                  <h2 class="m-0 text-xl font-semibold">1. Configurare sedinta</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Definim identitatea sedintei si parametrii minimali necesari pentru registru si traseul procedural.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">An scolar</span>
                    <input pInputText formControlName="schoolYear" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Organism</span>
                    <p-select
                      formControlName="organism"
                      [options]="organismOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Titlu sedinta</span>
                    <input pInputText formControlName="title" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Tip sedinta</span>
                    <p-select
                      formControlName="meetingType"
                      [options]="meetingTypeOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

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
                    <span class="text-sm font-medium">Data sedintei</span>
                    <p-datepicker formControlName="meetingDate" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Locatie</span>
                    <input pInputText formControlName="location" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Cvorum si coordonare</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim actorii-cheie si elementele pe care directorul trebuie sa le urmareasca inainte de sedinta.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Cvorum necesar</span>
                    <p-inputnumber formControlName="quorumRequired" inputId="quorumRequired" [min]="1" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Participanti estimati</span>
                    <p-inputnumber formControlName="participantsCount" inputId="participantsCount" [min]="0" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Presedinte sedinta</span>
                    <input pInputText formControlName="chairperson" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Secretar</span>
                    <input pInputText formControlName="secretaryName" />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Rezumat / scop</span>
                    <textarea pTextarea formControlName="summary" rows="5"></textarea>
                  </label>
                </div>

                <p-divider />

                <div class="grid gap-3">
                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                    <p-checkbox formControlName="convocationReady" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Convocator pregatit</div>
                      <p class="m-0 text-sm text-muted-color">
                        Exista draft de convocator si se poate porni ulterior fluxul de comunicare.
                      </p>
                    </div>
                  </label>

                  <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                    <p-checkbox formControlName="attendanceTemplateReady" [binary]="true" />
                    <div class="space-y-1">
                      <div class="font-medium">Lista de prezenta pregatita</div>
                      <p class="m-0 text-sm text-muted-color">
                        Exista sablon pentru semnaturi si pentru verificarea cvorumului.
                      </p>
                    </div>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <app-wizard-summary-panel
              title="3. Rezumat procedural"
              description="Aceasta sinteza va sta la baza dosarului de sedinta si a viitoarelor wizard-uri de minute, vot si hotarari."
              [items]="summaryItems()"
            />
          }

          @if (currentStep() === 3) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-4">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">4. Confirmare si creare</h2>
                  <p class="m-0 text-sm text-muted-color">
                    La acest pas inregistram sedinta in registru si pregatim punctul de plecare pentru documentele subordonate.
                  </p>
                </div>

                <div class="rounded-2xl border border-surface bg-surface-50 p-4 text-sm text-muted-color">
                  Dupa creare, sedinta va putea fi completata in registrul de guvernanta cu participanti, documente,
                  voturi, minute si hotarari.
                </div>

                <div class="flex flex-wrap items-center gap-3">
                  <p-button
                    label="Creeaza sedinta"
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
            </p-card>
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
            title="Documente care vor fi necesare"
            description="Setul minim recomandat pentru dosarul sedintei CA."
            [documents]="documentPreviewItems()"
          />

          <app-audit-trail-panel
            title="Traseu de lucru recomandat"
            description="Ordinea optima pentru operarea sedintei in platforma."
            [events]="auditTrailItems()"
          />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CaMeetingWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly organismMode = signal<'ca' | 'cp'>('ca');
  protected readonly maxStepIndex = 3;

  protected readonly organismOptions = [
    { label: 'Consiliu de administratie', value: 'ca' },
    { label: 'Consiliu profesoral', value: 'cp' },
    { label: 'CEAC', value: 'ceac' },
    { label: 'CFDCD', value: 'cfdcd' },
  ];

  protected readonly meetingTypeOptions = [
    { label: 'Ordinara', value: 'ordinary' },
    { label: 'Extraordinara', value: 'extraordinary' },
  ];

  protected readonly statusOptions = [
    { label: 'Draft', value: 'draft' },
    { label: 'Programata', value: 'scheduled' },
    { label: 'Desfasurata', value: 'held' },
    { label: 'Publicata', value: 'published' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    schoolYear: ['2025-2026', Validators.required],
    organism: ['ca', Validators.required],
    title: ['', [Validators.required, Validators.minLength(5)]],
    meetingType: ['ordinary', Validators.required],
    status: ['draft', Validators.required],
    meetingDate: [new Date(), Validators.required],
    location: ['', Validators.required],
    quorumRequired: [5, [Validators.required, Validators.min(1)]],
    participantsCount: [7, [Validators.required, Validators.min(0)]],
    chairperson: ['', Validators.required],
    secretaryName: ['', Validators.required],
    summary: ['', [Validators.required, Validators.minLength(10)]],
    convocationReady: [true],
    attendanceTemplateReady: [true],
  });

  protected readonly wizardTitle = computed(() => (this.organismMode() === 'cp'
    ? 'Wizard sedinta Consiliu Profesoral'
    : 'Wizard sedinta Consiliu de Administratie'));

  protected readonly wizardDescription = computed(() => (this.organismMode() === 'cp'
    ? 'Flux ghidat pentru pregatirea dosarului de sedinta CP: convocare, cvorum, proces-verbal si documentele oficiale aferente.'
    : 'Flux ghidat pentru pregatirea dosarului de sedinta: convocare, cvorum, coordonare operationala si pachetul minim de documente.'));

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    {
      key: 'setup',
      label: 'Configurare',
      description: 'Identitate, organism, data si locatie.',
      status: this.stepStatus(0),
    },
    {
      key: 'coordination',
      label: 'Coordonare',
      description: 'Cvorum, participanti si checklist procedural.',
      status: this.stepStatus(1),
    },
    {
      key: 'summary',
      label: 'Rezumat',
      description: 'Validare rapida a datelor colectate.',
      status: this.stepStatus(2),
    },
    {
      key: 'submit',
      label: 'Creare',
      description: 'Inregistrare in registrul de guvernanta.',
      status: this.stepStatus(3),
    },
  ]);

  protected readonly summaryItems = computed(() => {
    const value = this.form.getRawValue();
    return [
      { key: 'school-year', label: 'An scolar', value: value.schoolYear },
      { key: 'organism', label: 'Organism', value: this.labelFor(this.organismOptions, value.organism) },
      { key: 'title', label: 'Titlu', value: value.title || '-' },
      { key: 'meeting-date', label: 'Data', value: this.formatDate(value.meetingDate) },
      { key: 'location', label: 'Locatie', value: value.location || '-' },
      { key: 'quorum', label: 'Cvorum necesar', value: String(value.quorumRequired) },
      { key: 'participants', label: 'Participanti estimati', value: String(value.participantsCount) },
      { key: 'chairperson', label: 'Presedinte', value: value.chairperson || '-' },
      { key: 'secretary', label: 'Secretar', value: value.secretaryName || '-' },
      { key: 'readiness', label: 'Checklist minima', value: this.readinessLabel() },
    ];
  });

  protected readonly documentPreviewItems = computed(() => {
    const value = this.form.getRawValue();
    return [
      {
        id: 'convocation',
        title: 'Convocator sedinta',
        summary: value.convocationReady
          ? 'Marcat ca pregatit pentru urmatorul flux procedural.'
          : 'Trebuie generat sau atasat inainte de sedinta.',
        status: value.convocationReady ? 'active' : 'pending',
      },
      {
        id: 'attendance',
        title: 'Lista prezenta si semnaturi',
        summary: value.attendanceTemplateReady
          ? 'Sablonul de prezenta este considerat disponibil.'
          : 'Sablonul de prezenta trebuie pregatit.',
        status: value.attendanceTemplateReady ? 'active' : 'pending',
      },
      {
        id: 'agenda',
        title: 'Agenda si anexe',
        summary: 'Agenda va fi completata in registrul sedintei dupa creare.',
        status: 'draft',
      },
      {
        id: 'minutes',
        title: 'Minute si hotarari',
        summary: 'Vor deriva din sedinta si din fluxurile subordonate vot/minute.',
        status: 'scheduled',
      },
    ];
  });

  protected readonly auditTrailItems = computed(() => [
    {
      id: 'audit-1',
      title: 'Creare sedinta',
      summary: 'Directorul sau secretarul creeaza cadrul procedural si defineste datele de baza.',
      actorName: 'Director / Secretar',
    },
    {
      id: 'audit-2',
      title: 'Completare participanti si documente',
      summary: 'Dupa creare se completeaza prezenta, anexele si convocatorul in registrul de guvernanta.',
      actorName: 'Secretar',
    },
    {
      id: 'audit-3',
      title: 'Vot, minute, hotarari',
      summary: 'Sedinta devine baza pentru minute, voturi si hotarari publicabile.',
      actorName: 'Director / CA',
    },
  ]);

  constructor() {
    const organism = this.route.snapshot.queryParamMap.get('organism');
    if (organism === 'cp' || organism === 'ca') {
      this.organismMode.set(organism);
      this.form.controls.organism.setValue(organism);
    }
  }

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['schoolYear', 'organism', 'title', 'meetingType', 'status', 'meetingDate', 'location']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['quorumRequired', 'participantsCount', 'chairperson', 'secretaryName', 'summary']);
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
        detail: 'Completeaza campurile obligatorii inainte de a crea sedinta.',
      });
      return;
    }

    const payload = this.buildPayload();
    this.submitting.set(true);

    try {
      await firstValueFrom(
        this.http.post('/api/education/governance/meetings', payload).pipe(finalize(() => this.submitting.set(false))),
      );

      this.messages.add({
        severity: 'success',
        summary: 'Sedinta a fost creata',
        detail: 'Fluxul de baza a fost deschis in registrul de guvernanta.',
      });

      setTimeout(() => {
        void this.router.navigate(['/education', 'governance']);
      }, 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({
        severity: 'error',
        summary: 'Crearea a esuat',
        detail: 'Sedinta nu a putut fi inregistrata. Verifica datele si incearca din nou.',
      });
    }
  }

  private buildPayload(): MeetingWizardPayload {
    const value = this.form.getRawValue();
    return {
      school_year: value.schoolYear.trim(),
      organism: value.organism,
      title: value.title.trim(),
      meeting_type: value.meetingType,
      status: value.status,
      quorum_required: value.quorumRequired,
      participants_count: value.participantsCount,
      meeting_date: this.formatDate(value.meetingDate),
      location: value.location.trim(),
      chairperson: value.chairperson.trim(),
      secretary_name: value.secretaryName.trim(),
      summary: value.summary.trim(),
    };
  }

  private readinessLabel(): string {
    const value = this.form.getRawValue();
    const completed = [value.convocationReady, value.attendanceTemplateReady].filter(Boolean).length;
    return `${completed}/2 pregatite`;
  }

  private labelFor(options: Array<{ label: string; value: string }>, value: string): string {
    return options.find((option) => option.value === value)?.label ?? value;
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
      ['schoolYear', 'organism', 'title', 'meetingType', 'status', 'meetingDate', 'location'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
      return;
    }
    if (this.currentStep() === 1) {
      ['quorumRequired', 'participantsCount', 'chairperson', 'secretaryName', 'summary'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
    }
  }

  private isStepValid(controlNames: string[]): boolean {
    return controlNames.every((name) => this.form.controls[name as keyof typeof this.form.controls].valid);
  }

  private formatDate(value: Date | string): string {
    if (typeof value === 'string') {
      return value;
    }
    const year = value.getFullYear();
    const month = `${value.getMonth() + 1}`.padStart(2, '0');
    const day = `${value.getDate()}`.padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
}
