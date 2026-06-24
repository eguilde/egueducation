import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { MessageService } from 'primeng/api';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
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

interface GovernanceMeetingOption {
  id: string;
  school_year: string;
  organism: string;
  title: string;
  status: string;
  meeting_date: string;
}

interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

interface VoteWizardPayload {
  subject_title: string;
  agenda_order: number;
  decision_type: string;
  votes_for: number;
  votes_against: number;
  abstentions: number;
  outcome: string;
  requires_follow_up: boolean;
  legal_basis: string;
  notes: string;
}

@Component({
  selector: 'app-meeting-vote-wizard',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    ButtonModule,
    CardModule,
    CheckboxModule,
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
              <i class="pi pi-check-square"></i>
              Guvernanta educationala
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard vot si rezultat</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru inregistrarea unui punct supus la vot, a rezultatului si a masurilor de urmarire asociate.
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
                  <h2 class="m-0 text-xl font-semibold">1. Sedinta si subiect</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Alegem sedinta si definim punctul de pe ordinea de zi care va fi supus la vot.
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
                    />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Ordine pe agenda</span>
                    <p-inputnumber formControlName="agendaOrder" inputId="voteAgendaOrder" [min]="1" />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Subiect</span>
                    <input pInputText formControlName="subjectTitle" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Tip decizie si rezultat</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim natura deciziei si rezultatul votului pentru punctul selectat.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Tip decizie</span>
                    <p-select
                      formControlName="decisionType"
                      [options]="decisionTypeOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Rezultat</span>
                    <p-select
                      formControlName="outcome"
                      [options]="outcomeOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Voturi pentru</span>
                    <p-inputnumber formControlName="votesFor" inputId="votesFor" [min]="0" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Voturi impotriva</span>
                    <p-inputnumber formControlName="votesAgainst" inputId="votesAgainst" [min]="0" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Abtineri</span>
                    <p-inputnumber formControlName="abstentions" inputId="abstentions" [min]="0" />
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Temei si follow-up</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Adaugam temeiul legal si marcajul de urmarire pentru etapele urmatoare.
                  </p>
                </div>

                <div class="grid gap-4">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Temei legal</span>
                    <textarea pTextarea formControlName="legalBasis" rows="4"></textarea>
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Masuri / note</span>
                    <textarea pTextarea formControlName="notes" rows="4"></textarea>
                  </label>
                </div>

                <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                  <p-checkbox formControlName="requiresFollowUp" [binary]="true" />
                  <div class="space-y-1">
                    <div class="font-medium">Necesita urmarire operationala</div>
                    <p class="m-0 text-sm text-muted-color">
                      Marcheaza daca votul trebuie sa continue in fluxurile de monitorizare, minuta sau hotarare derivata.
                    </p>
                  </div>
                </label>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel
              title="4. Confirmare vot"
              description="Verificare finala inainte de a salva votul in sedinta selectata."
              [items]="summaryItems()"
            />

            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button
                  label="Creeaza votul"
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
            title="Artefacte derivate"
            description="Votul poate alimenta ulterior minuta structurata si hotararea derivata."
            [documents]="documentPreviewItems()"
          />

          <app-audit-trail-panel
            title="Traseu recomandat"
            description="Ordinea recomandata dupa inregistrarea votului."
            [events]="auditTrailItems()"
          />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MeetingVoteWizardComponent {
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
  protected readonly meetings = signal<GovernanceMeetingOption[]>([]);

  protected readonly decisionTypeOptions = [
    { label: 'Hotarare', value: 'hotarare' },
    { label: 'Aviz', value: 'aviz' },
    { label: 'Informare', value: 'informare' },
    { label: 'Delegare', value: 'delegare' },
    { label: 'Aprobare', value: 'aprobare' },
  ];

  protected readonly outcomeOptions = [
    { label: 'Adoptat', value: 'adoptat' },
    { label: 'Respins', value: 'respins' },
    { label: 'Amanat', value: 'amanat' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    meetingId: ['', Validators.required],
    agendaOrder: [1, [Validators.required, Validators.min(1)]],
    subjectTitle: ['', [Validators.required, Validators.minLength(5)]],
    decisionType: ['hotarare', Validators.required],
    outcome: ['adoptat', Validators.required],
    votesFor: [0, [Validators.required, Validators.min(0)]],
    votesAgainst: [0, [Validators.required, Validators.min(0)]],
    abstentions: [0, [Validators.required, Validators.min(0)]],
    requiresFollowUp: [false],
    legalBasis: [''],
    notes: [''],
  });

  constructor() {
    const meetingId = this.route.snapshot.queryParamMap.get('meetingId');
    if (meetingId) {
      this.form.controls.meetingId.setValue(meetingId);
    }
    void this.loadMeetings();
  }

  protected readonly meetingOptions = computed(() =>
    this.meetings().map((meeting) => ({
      value: meeting.id,
      label: `${this.organismLabel(meeting.organism)} • ${meeting.meeting_date} • ${meeting.title}`,
    })),
  );

  protected readonly selectedMeetingLabel = computed(() => {
    const meeting = this.meetings().find((item) => item.id === this.form.controls.meetingId.value);
    if (!meeting) {
      return '';
    }
    return `${this.organismLabel(meeting.organism)} • ${meeting.title} • ${meeting.meeting_date}`;
  });

  protected readonly steps = computed<EducationWizardStepState[]>(() => [
    { key: 'meeting', label: 'Sedinta', description: 'Subiect si ordine de zi.', status: this.stepStatus(0) },
    { key: 'vote', label: 'Rezultat', description: 'Tip decizie si numar voturi.', status: this.stepStatus(1) },
    { key: 'basis', label: 'Temei', description: 'Temei legal si follow-up.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'meeting', label: 'Sedinta', value: this.selectedMeetingLabel() || '-' },
    { key: 'agenda', label: 'Ordine agenda', value: String(this.form.controls.agendaOrder.value) },
    { key: 'subject', label: 'Subiect', value: this.form.controls.subjectTitle.value || '-' },
    {
      key: 'decision-type',
      label: 'Tip decizie',
      value: this.decisionTypeOptions.find((option) => option.value === this.form.controls.decisionType.value)?.label || '-',
    },
    {
      key: 'outcome',
      label: 'Rezultat',
      value: this.outcomeOptions.find((option) => option.value === this.form.controls.outcome.value)?.label || '-',
    },
    {
      key: 'votes',
      label: 'Voturi',
      value: `P ${this.form.controls.votesFor.value} / I ${this.form.controls.votesAgainst.value} / A ${this.form.controls.abstentions.value}`,
    },
    { key: 'follow-up', label: 'Necesita urmarire', value: this.form.controls.requiresFollowUp.value ? 'Da' : 'Nu' },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    {
      id: 'vote',
      title: 'Rezultat de vot',
      summary: 'Inregistrarea va deveni sursa pentru raportarea deciziilor si pentru hotarari derivate.',
      status: 'active',
    },
    {
      id: 'minute',
      title: 'Minuta structurata',
      summary: this.form.controls.requiresFollowUp.value
        ? 'Votul este pregatit pentru urmarire si consemnare detaliata in minuta.'
        : 'Votul poate fi legat ulterior de o minuta detaliata.',
      status: this.form.controls.requiresFollowUp.value ? 'pending' : 'draft',
    },
    {
      id: 'resolution',
      title: 'Hotarare / aviz',
      summary: 'Daca este cazul, votul devine baza pentru actul formal rezultat.',
      status: 'scheduled',
    },
  ]);

  protected readonly auditTrailItems = computed(() => [
    {
      id: 'trail-1',
      title: 'Inregistrare vot',
      summary: 'Secretarul sau directorul inregistreaza subiectul, rezultatul si numarul voturilor.',
      actorName: 'Director / Secretar',
    },
    {
      id: 'trail-2',
      title: 'Monitorizare implementare',
      summary: 'Masurile rezultate sunt urmarite daca votul necesita follow-up operational.',
      actorName: 'Responsabil desemnat',
    },
    {
      id: 'trail-3',
      title: 'Act derivat',
      summary: 'Votul poate alimenta minuta structurata si ulterior hotararea/avizul formal.',
      actorName: 'Director / Secretar',
    },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['meetingId', 'agendaOrder', 'subjectTitle']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['decisionType', 'outcome', 'votesFor', 'votesAgainst', 'abstentions']);
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
        detail: 'Completeaza campurile obligatorii inainte de a crea votul.',
      });
      return;
    }

    const meetingId = this.form.controls.meetingId.value;
    const payload = this.buildPayload();
    this.submitting.set(true);

    try {
      await firstValueFrom(
        this.http
          .post(`/api/education/governance/meetings/${meetingId}/votes`, payload)
          .pipe(finalize(() => this.submitting.set(false))),
      );

      this.messages.add({
        severity: 'success',
        summary: 'Votul a fost creat',
        detail: 'Sedinta selectata are acum un nou rezultat de vot inregistrat.',
      });

      setTimeout(() => {
        void this.router.navigate(['/education', 'governance']);
      }, 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({
        severity: 'error',
        summary: 'Crearea a esuat',
        detail: 'Votul nu a putut fi salvat. Verifica datele si incearca din nou.',
      });
    }
  }

  private async loadMeetings(): Promise<void> {
    this.loadingMeetings.set(true);
    try {
      const response = await firstValueFrom(
        this.http.get<PagedResponse<GovernanceMeetingOption>>('/api/education/governance/meetings?page=1&pageSize=100'),
      );
      this.meetings.set(response.items ?? []);
    } catch {
      this.meetings.set([]);
      this.messages.add({
        severity: 'warn',
        summary: 'Sedintele nu au putut fi incarcate',
        detail: 'Wizard-ul are nevoie de o sedinta existenta pentru a crea un vot.',
      });
    } finally {
      this.loadingMeetings.set(false);
    }
  }

  private buildPayload(): VoteWizardPayload {
    const value = this.form.getRawValue();
    return {
      subject_title: value.subjectTitle.trim(),
      agenda_order: value.agendaOrder,
      decision_type: value.decisionType,
      votes_for: value.votesFor,
      votes_against: value.votesAgainst,
      abstentions: value.abstentions,
      outcome: value.outcome,
      requires_follow_up: value.requiresFollowUp,
      legal_basis: value.legalBasis.trim(),
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
      ['meetingId', 'agendaOrder', 'subjectTitle'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
      return;
    }
    if (this.currentStep() === 1) {
      ['decisionType', 'outcome', 'votesFor', 'votesAgainst', 'abstentions'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
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
}
