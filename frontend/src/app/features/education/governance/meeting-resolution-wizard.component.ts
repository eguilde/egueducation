import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
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

interface GovernanceMeetingOption {
  id: string;
  school_year: string;
  organism: string;
  title: string;
  status: string;
  meeting_date: string;
}

interface GovernanceVoteOption {
  id: string;
  meeting_id: string;
  subject_title: string;
  agenda_order: number;
  outcome: string;
}

interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

interface ResolutionWizardPayload {
  vote_id: string;
  title: string;
  resolution_type: string;
  publication_status: string;
  anonymization_state: string;
  issued_on: string;
  signed_by: string;
  notes: string;
}

@Component({
  selector: 'app-meeting-resolution-wizard',
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
              <i class="pi pi-verified"></i>
              Guvernanta educationala
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard hotarare si aviz</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru transformarea unui vot intr-un act formal, cu publicare si anonimizare controlata.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <p-tag [value]="institutionName()" severity="secondary" />
            <app-deadline-badge label="Flux asistat" status="active" />
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
                  <h2 class="m-0 text-xl font-semibold">1. Sedinta si vot</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Alegem sedinta si votul sursa care fundamenteaza hotararea sau avizul formal.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Sedinta</span>
                    <p-select
                      formControlName="meetingId"
                      [options]="meetingOptions()"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                      [loading]="loadingMeetings()"
                      placeholder="Selecteaza sedinta"
                      (onChange)="onMeetingChange()"
                    />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Vot sursa</span>
                    <p-select
                      formControlName="voteId"
                      [options]="voteOptions()"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                      [loading]="loadingVotes()"
                      placeholder="Selecteaza votul"
                    />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Act si emitere</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Definim tipul actului, titulatura lui si datele minime de emitere.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Titlu hotarare / aviz</span>
                    <input pInputText formControlName="title" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Tip act</span>
                    <p-select
                      formControlName="resolutionType"
                      [options]="resolutionTypeOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Data emiterii</span>
                    <p-datepicker formControlName="issuedOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Semnat de</span>
                    <input pInputText formControlName="signedBy" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Publicare si anonimizare</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim starea de publicare si nivelul de anonimizare necesar pentru actul emis.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status publicare</span>
                    <p-select
                      formControlName="publicationStatus"
                      [options]="publicationStatusOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status anonimizare</span>
                    <p-select
                      formControlName="anonymizationState"
                      [options]="anonymizationStateOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Note</span>
                    <textarea pTextarea formControlName="notes" rows="4"></textarea>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel
              title="4. Confirmare hotarare"
              description="Verificare finala inainte de a salva actul formal in sedinta selectata."
              [items]="summaryItems()"
            />

            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button
                  label="Creeaza hotararea"
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
            title="Pachet procedural"
            description="Actul formal devine sursa pentru publicare, anonimizare si evidenta de guvernanta."
            [documents]="documentPreviewItems()"
          />

          <app-audit-trail-panel
            title="Traseu recomandat"
            description="Ordinea recomandata dupa generarea hotararii sau avizului."
            [events]="auditTrailItems()"
          />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MeetingResolutionWizardComponent {
  private readonly fb = inject(FormBuilder);
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly currentStep = signal(0);
  protected readonly submitting = signal(false);
  protected readonly maxStepIndex = 3;
  protected readonly loadingMeetings = signal(false);
  protected readonly loadingVotes = signal(false);
  protected readonly meetings = signal<GovernanceMeetingOption[]>([]);
  protected readonly votes = signal<GovernanceVoteOption[]>([]);

  protected readonly resolutionTypeOptions = [
    { label: 'Hotarare', value: 'hotarare' },
    { label: 'Decizie', value: 'decizie' },
    { label: 'Aviz', value: 'aviz' },
  ];

  protected readonly publicationStatusOptions = [
    { label: 'Intern', value: 'intern' },
    { label: 'Pregatit pentru publicare', value: 'pregatit_publicare' },
    { label: 'Publicat', value: 'publicat' },
  ];

  protected readonly anonymizationStateOptions = [
    { label: 'Necesara', value: 'necesara' },
    { label: 'Finalizata', value: 'finalizata' },
    { label: 'Nu este necesara', value: 'nu_este_necesara' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    meetingId: ['', Validators.required],
    voteId: ['', Validators.required],
    title: ['', [Validators.required, Validators.minLength(5)]],
    resolutionType: ['hotarare', Validators.required],
    publicationStatus: ['intern', Validators.required],
    anonymizationState: ['necesara', Validators.required],
    issuedOn: [new Date(), Validators.required],
    signedBy: [''],
    notes: [''],
  });

  constructor() {
    const meetingId = this.route.snapshot.queryParamMap.get('meetingId');
    const voteId = this.route.snapshot.queryParamMap.get('voteId');
    if (meetingId) {
      this.form.controls.meetingId.setValue(meetingId);
    }
    if (voteId) {
      this.form.controls.voteId.setValue(voteId);
    }
    void this.loadMeetings();
  }

  protected readonly meetingOptions = computed(() =>
    this.meetings().map((meeting) => ({
      value: meeting.id,
      label: `${this.organismLabel(meeting.organism)} • ${meeting.meeting_date} • ${meeting.title}`,
    })),
  );

  protected readonly voteOptions = computed(() =>
    this.votes().map((vote) => ({
      value: vote.id,
      label: `${vote.agenda_order}. ${vote.subject_title} (${vote.outcome})`,
    })),
  );

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'source', label: 'Sursa', description: 'Sedinta si votul sursa.', status: this.stepStatus(0) },
    { key: 'issuance', label: 'Emitere', description: 'Tip act si emitere.', status: this.stepStatus(1) },
    { key: 'publication', label: 'Publicare', description: 'Publicare si anonimizare.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'meeting', label: 'Sedinta', value: this.selectedMeetingLabel() || '-' },
    { key: 'vote', label: 'Vot sursa', value: this.selectedVoteLabel() || '-' },
    { key: 'title', label: 'Titlu', value: this.form.controls.title.value || '-' },
    {
      key: 'type',
      label: 'Tip act',
      value: this.resolutionTypeOptions.find((option) => option.value === this.form.controls.resolutionType.value)?.label || '-',
    },
    {
      key: 'publication',
      label: 'Publicare',
      value: this.publicationStatusOptions.find((option) => option.value === this.form.controls.publicationStatus.value)?.label || '-',
    },
    {
      key: 'anonymization',
      label: 'Anonimizare',
      value: this.anonymizationStateOptions.find((option) => option.value === this.form.controls.anonymizationState.value)?.label || '-',
    },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    {
      id: 'resolution',
      title: 'Act formal de guvernanta',
      summary: 'Documentul va fi legat de votul sursa si inregistrat in sedinta selectata.',
      status: 'active',
    },
    {
      id: 'publication',
      title: 'Publicare institutionala',
      summary: 'Starea de publicare va putea alimenta registrul de publicatii si cockpit-ul directorului.',
      status: 'pending',
    },
    {
      id: 'anonymization',
      title: 'Control anonimizare',
      summary: 'Hotararea va fi urmarita si din perspectiva protectiei datelor, cand este cazul.',
      status: 'scheduled',
    },
  ]);

  protected readonly auditTrailItems = computed(() => [
    {
      id: 'trail-1',
      title: 'Act derivat din vot',
      summary: 'Directorul sau secretarul transforma un rezultat de vot intr-o hotarare sau intr-un aviz formal.',
      actorName: 'Director / Secretar',
    },
    {
      id: 'trail-2',
      title: 'Publicare si anonimizare',
      summary: 'Actul este pregatit pentru publicare interna sau externa, cu nivelul necesar de anonimizare.',
      actorName: 'Secretar / GDPR',
    },
    {
      id: 'trail-3',
      title: 'Trasabilitate de guvernanta',
      summary: 'Actul ramane legat de sedinta si de votul sursa pentru audit si raportare manageriala.',
      actorName: 'Management educational',
    },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['meetingId', 'voteId']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['title', 'resolutionType', 'issuedOn']);
    }
    if (this.currentStep() === 2) {
      return this.isStepValid(['publicationStatus', 'anonymizationState']);
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
        detail: 'Completeaza campurile obligatorii inainte de a crea hotararea.',
      });
      return;
    }

    const meetingId = this.form.controls.meetingId.value;
    const payload = this.buildPayload();
    this.submitting.set(true);

    try {
      await firstValueFrom(
        this.http
          .post(`/api/education/governance/meetings/${meetingId}/resolutions`, payload)
          .pipe(finalize(() => this.submitting.set(false))),
      );

      this.messages.add({
        severity: 'success',
        summary: 'Hotararea a fost creata',
        detail: 'Actul formal a fost legat de sedinta si de votul selectat.',
      });

      setTimeout(() => {
        void this.router.navigate(['/education', 'governance']);
      }, 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({
        severity: 'error',
        summary: 'Crearea a esuat',
        detail: 'Hotararea nu a putut fi salvata. Verifica datele si incearca din nou.',
      });
    }
  }

  protected onMeetingChange(): void {
    this.form.controls.voteId.setValue('');
    void this.loadVotesForMeeting(this.form.controls.meetingId.value);
  }

  private async loadMeetings(): Promise<void> {
    this.loadingMeetings.set(true);
    try {
      const response = await firstValueFrom(
        this.http.get<PagedResponse<GovernanceMeetingOption>>('/api/education/governance/meetings?page=1&pageSize=100'),
      );
      this.meetings.set(response.items ?? []);
      if (this.form.controls.meetingId.value) {
        await this.loadVotesForMeeting(this.form.controls.meetingId.value);
      }
    } catch {
      this.meetings.set([]);
      this.votes.set([]);
    } finally {
      this.loadingMeetings.set(false);
    }
  }

  private async loadVotesForMeeting(meetingId: string): Promise<void> {
    if (!meetingId) {
      this.votes.set([]);
      return;
    }
    this.loadingVotes.set(true);
    try {
      const response = await firstValueFrom(
        this.http.get<PagedResponse<GovernanceVoteOption>>(`/api/education/governance/meetings/${meetingId}/votes?page=1&pageSize=100`),
      );
      this.votes.set(response.items ?? []);
      if (this.form.controls.voteId.value && !this.votes().some((vote) => vote.id === this.form.controls.voteId.value)) {
        this.form.controls.voteId.setValue('');
      }
    } catch {
      this.votes.set([]);
    } finally {
      this.loadingVotes.set(false);
    }
  }

  private selectedMeetingLabel(): string {
    const meeting = this.meetings().find((item) => item.id === this.form.controls.meetingId.value);
    return meeting ? `${this.organismLabel(meeting.organism)} • ${meeting.title} • ${meeting.meeting_date}` : '';
  }

  private selectedVoteLabel(): string {
    const vote = this.votes().find((item) => item.id === this.form.controls.voteId.value);
    return vote ? `${vote.agenda_order}. ${vote.subject_title} (${vote.outcome})` : '';
  }

  private buildPayload(): ResolutionWizardPayload {
    const value = this.form.getRawValue();
    return {
      vote_id: value.voteId,
      title: value.title.trim(),
      resolution_type: value.resolutionType,
      publication_status: value.publicationStatus,
      anonymization_state: value.anonymizationState,
      issued_on: this.formatDate(value.issuedOn),
      signed_by: value.signedBy.trim(),
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
      ['meetingId', 'voteId'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 1) {
      ['title', 'resolutionType', 'issuedOn'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
      return;
    }
    if (this.currentStep() === 2) {
      ['publicationStatus', 'anonymizationState'].forEach((key) => this.form.controls[key as keyof typeof this.form.controls].markAsTouched());
    }
  }

  private isStepValid(controlNames: string[]): boolean {
    return controlNames.every((name) => this.form.controls[name as keyof typeof this.form.controls].valid);
  }

  private organismLabel(value: string): string {
    switch (value) {
      case 'ca':
        return 'CA';
      case 'cp':
        return 'CP';
      case 'ceac':
        return 'CEAC';
      case 'cfdcd':
        return 'CFDCD';
      default:
        return value;
    }
  }

  private formatDate(value: Date): string {
    const year = value.getFullYear();
    const month = `${value.getMonth() + 1}`.padStart(2, '0');
    const day = `${value.getDate()}`.padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
}
