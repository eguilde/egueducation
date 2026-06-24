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

interface MinuteWizardPayload {
  agenda_order: number;
  topic_title: string;
  discussion_summary: string;
  decision_summary: string;
  responsible_party: string;
  due_on: string;
  follow_up_status: string;
  requires_publication: boolean;
  notes: string;
}

@Component({
  selector: 'app-meeting-minute-wizard',
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
              <i class="pi pi-file-edit"></i>
              Guvernanta educationala
            </div>
            <h1 class="m-0 text-3xl font-semibold">Wizard punct de minuta</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Flux ghidat pentru definirea unei discutii, a deciziei rezultate si a urmaririi operative in dosarul sedintei.
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
                    Selectam sedinta si definim punctul de pe ordinea de zi pentru care redactam minuta.
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
                    <p-inputnumber formControlName="agendaOrder" inputId="agendaOrder" [min]="1" />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Titlu subiect</span>
                    <input pInputText formControlName="topicTitle" />
                  </label>

                  @if (selectedMeetingLabel()) {
                    <div class="md:col-span-2 rounded-2xl border border-surface bg-surface-50 p-3 text-sm text-muted-color">
                      Sedinta selectata: <strong>{{ selectedMeetingLabel() }}</strong>
                    </div>
                  }
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 1) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">2. Discutii si decizie</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Capturam pe scurt contextul discutiei si rezultatul operational care trebuie urmarit.
                  </p>
                </div>

                <div class="grid gap-4">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Rezumat discutii</span>
                    <textarea pTextarea formControlName="discussionSummary" rows="5"></textarea>
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Rezumat decizie</span>
                    <textarea pTextarea formControlName="decisionSummary" rows="5"></textarea>
                  </label>
                </div>
              </div>
            </p-card>
          }

          @if (currentStep() === 2) {
            <p-card styleClass="border border-surface bg-surface-0 shadow-sm">
              <div class="space-y-5">
                <div class="space-y-1">
                  <h2 class="m-0 text-xl font-semibold">3. Urmarire si publicare</h2>
                  <p class="m-0 text-sm text-muted-color">
                    Stabilim responsabilul, termenul si daca punctul trebuie sa alimenteze ulterior o hotarare/publicare.
                  </p>
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                  <label class="space-y-2">
                    <span class="text-sm font-medium">Responsabil</span>
                    <input pInputText formControlName="responsibleParty" />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Status follow-up</span>
                    <p-select
                      formControlName="followUpStatus"
                      [options]="followUpStatusOptions"
                      optionLabel="label"
                      optionValue="value"
                      appendTo="body"
                    />
                  </label>

                  <label class="space-y-2">
                    <span class="text-sm font-medium">Scadenta</span>
                    <p-datepicker formControlName="dueOn" dateFormat="yy-mm-dd" appendTo="body" />
                  </label>

                  <label class="space-y-2 md:col-span-2">
                    <span class="text-sm font-medium">Note</span>
                    <textarea pTextarea formControlName="notes" rows="4"></textarea>
                  </label>
                </div>

                <label class="flex items-start gap-3 rounded-xl border border-surface p-3">
                  <p-checkbox formControlName="requiresPublication" [binary]="true" />
                  <div class="space-y-1">
                    <div class="font-medium">Necesita publicare / hotarare derivata</div>
                    <p class="m-0 text-sm text-muted-color">
                      Marcheaza daca punctul trebuie sa continue in fluxul de hotarare, anonimizare sau publicare.
                    </p>
                  </div>
                </label>
              </div>
            </p-card>
          }

          @if (currentStep() === 3) {
            <app-wizard-summary-panel
              title="4. Confirmare minuta"
              description="Verificare finala inainte de a inregistra punctul de minuta in sedinta selectata."
              [items]="summaryItems()"
            />

            <div class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
              <div class="flex flex-wrap items-center gap-3">
                <p-button
                  label="Creeaza punctul de minuta"
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
            description="Punctul de minuta va putea alimenta ulterior alte fluxuri procedurale."
            [documents]="documentPreviewItems()"
          />

          <app-audit-trail-panel
            title="Traseu recomandat"
            description="Ordinea recomandata de lucru dupa inregistrarea punctului de minuta."
            [events]="auditTrailItems()"
          />
        </div>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MeetingMinuteWizardComponent {
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

  protected readonly followUpStatusOptions = [
    { label: 'De stabilit', value: 'de_stabilit' },
    { label: 'In urmarire', value: 'in_urmarire' },
    { label: 'Realizat', value: 'realizat' },
    { label: 'Amanat', value: 'amanat' },
    { label: 'Inchis', value: 'inchis' },
  ];

  protected readonly form = this.fb.nonNullable.group({
    meetingId: ['', Validators.required],
    agendaOrder: [1, [Validators.required, Validators.min(1)]],
    topicTitle: ['', [Validators.required, Validators.minLength(5)]],
    discussionSummary: ['', [Validators.required, Validators.minLength(10)]],
    decisionSummary: ['', [Validators.required, Validators.minLength(10)]],
    responsibleParty: ['', Validators.required],
    followUpStatus: ['de_stabilit', Validators.required],
    dueOn: [null as Date | null],
    requiresPublication: [false],
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
    { key: 'meeting', label: 'Sedinta', description: 'Selectie sedinta si punct de agenda.', status: this.stepStatus(0) },
    { key: 'decision', label: 'Discutii', description: 'Context si decizie consemnata.', status: this.stepStatus(1) },
    { key: 'follow-up', label: 'Urmarire', description: 'Responsabil, termen si publicare.', status: this.stepStatus(2) },
    { key: 'confirm', label: 'Confirmare', description: 'Revizuire si creare.', status: this.stepStatus(3) },
  ]);

  protected readonly summaryItems = computed(() => [
    { key: 'meeting', label: 'Sedinta', value: this.selectedMeetingLabel() || '-' },
    { key: 'agenda', label: 'Ordine agenda', value: String(this.form.controls.agendaOrder.value) },
    { key: 'topic', label: 'Subiect', value: this.form.controls.topicTitle.value || '-' },
    { key: 'responsible', label: 'Responsabil', value: this.form.controls.responsibleParty.value || '-' },
    {
      key: 'follow-up',
      label: 'Status follow-up',
      value: this.followUpStatusOptions.find((option) => option.value === this.form.controls.followUpStatus.value)?.label || '-',
    },
    { key: 'due-on', label: 'Scadenta', value: this.formatDate(this.form.controls.dueOn.value) || '-' },
    { key: 'publication', label: 'Necesita publicare', value: this.form.controls.requiresPublication.value ? 'Da' : 'Nu' },
  ]);

  protected readonly documentPreviewItems = computed(() => [
    {
      id: 'minutes',
      title: 'Minuta sedintei',
      summary: 'Punctul nou va intra in minuta oficiala a sedintei selectate.',
      status: 'active',
    },
    {
      id: 'resolution',
      title: 'Hotarare derivata',
      summary: this.form.controls.requiresPublication.value
        ? 'Punctul este marcat pentru continuare in fluxul de hotarare/publicare.'
        : 'Optional, poate alimenta ulterior o hotarare daca este necesar.',
      status: this.form.controls.requiresPublication.value ? 'pending' : 'draft',
    },
    {
      id: 'follow-up',
      title: 'Urmarire operationala',
      summary: 'Responsabilul si termenul devin baza pentru monitorizarea implementarii.',
      status: 'scheduled',
    },
  ]);

  protected readonly auditTrailItems = computed(() => [
    {
      id: 'trail-1',
      title: 'Consemnare minuta',
      summary: 'Secretarul sau directorul inregistreaza punctul de discutie si decizia asociata.',
      actorName: 'Director / Secretar',
    },
    {
      id: 'trail-2',
      title: 'Urmarire masuri',
      summary: 'Masurile rezultate sunt monitorizate pana la realizare, amanare sau inchidere.',
      actorName: 'Responsabil desemnat',
    },
    {
      id: 'trail-3',
      title: 'Publicare / hotarare',
      summary: 'Daca este cazul, punctul alimenteaza fluxul de hotarare, anonimizare si publicare.',
      actorName: 'Director / Secretar',
    },
  ]);

  protected canAdvance(): boolean {
    if (this.currentStep() === 0) {
      return this.isStepValid(['meetingId', 'agendaOrder', 'topicTitle']);
    }
    if (this.currentStep() === 1) {
      return this.isStepValid(['discussionSummary', 'decisionSummary']);
    }
    if (this.currentStep() === 2) {
      return this.isStepValid(['responsibleParty', 'followUpStatus']);
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
        detail: 'Completeaza campurile obligatorii inainte de a crea punctul de minuta.',
      });
      return;
    }

    const meetingId = this.form.controls.meetingId.value;
    const payload = this.buildPayload();
    this.submitting.set(true);

    try {
      await firstValueFrom(
        this.http
          .post(`/api/education/governance/meetings/${meetingId}/minutes`, payload)
          .pipe(finalize(() => this.submitting.set(false))),
      );

      this.messages.add({
        severity: 'success',
        summary: 'Punctul de minuta a fost creat',
        detail: 'Sedinta selectata are acum un nou punct consemnat in minuta.',
      });

      setTimeout(() => {
        void this.router.navigate(['/education', 'governance']);
      }, 300);
    } catch {
      this.submitting.set(false);
      this.messages.add({
        severity: 'error',
        summary: 'Crearea a esuat',
        detail: 'Punctul de minuta nu a putut fi salvat. Verifica datele si incearca din nou.',
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
        detail: 'Wizard-ul are nevoie de o sedinta existenta pentru a crea un punct de minuta.',
      });
    } finally {
      this.loadingMeetings.set(false);
    }
  }

  private buildPayload(): MinuteWizardPayload {
    const value = this.form.getRawValue();
    return {
      agenda_order: value.agendaOrder,
      topic_title: value.topicTitle.trim(),
      discussion_summary: value.discussionSummary.trim(),
      decision_summary: value.decisionSummary.trim(),
      responsible_party: value.responsibleParty.trim(),
      due_on: this.formatDate(value.dueOn),
      follow_up_status: value.followUpStatus,
      requires_publication: value.requiresPublication,
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
      ['meetingId', 'agendaOrder', 'topicTitle'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
      return;
    }
    if (this.currentStep() === 1) {
      ['discussionSummary', 'decisionSummary'].forEach((key) =>
        this.form.controls[key as keyof typeof this.form.controls].markAsTouched(),
      );
      return;
    }
    if (this.currentStep() === 2) {
      ['responsibleParty', 'followUpStatus'].forEach((key) =>
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

  private formatDate(value: Date | null): string {
    if (!value) {
      return '';
    }
    const year = value.getFullYear();
    const month = `${value.getMonth() + 1}`.padStart(2, '0');
    const day = `${value.getDate()}`.padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
}
