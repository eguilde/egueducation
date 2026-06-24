import { CommonModule } from '@angular/common';
import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

import { AuthzService } from '../../../core/authz/authz.service';

interface PortfolioSelfServiceCard {
  key: string;
  label: string;
  value: string;
  icon: string;
}

interface PortfolioPreviewRow {
  id: string;
  portfolio_code: string;
  owner_name: string;
  owner_role: string;
  school_year: string;
  status: string;
  section_count: number;
  transfer_status: string;
  retention_until: string;
  last_updated_on?: string;
}

interface PortfolioSelfServiceAction {
  key: string;
  label: string;
  summary: string;
  route: string;
  queryParams: Record<string, string>;
  severity?: 'primary' | 'secondary';
}

@Component({
  selector: 'app-education-portfolio-self-service',
  standalone: true,
  imports: [CommonModule, RouterLink, ButtonModule, CardModule, TagModule],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 overflow-auto p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-folder-open"></i>
              Portofoliu personal
            </div>
            <h1 class="m-0 text-3xl font-semibold">Portofoliul meu</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Ecran dedicat pentru profesor, cu acces direct la portofoliul propriu, opis, transfer digital si punctele
              de verificare utile pentru completare.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            @for (role of roleTags(); track role) {
              <p-tag [value]="role" severity="contrast" />
            }
            <p-tag [value]="institutionName()" severity="secondary" />
            <p-button
              [routerLink]="['/education', 'dashboard', 'teacher']"
              label="Inapoi la cockpit"
              icon="pi pi-arrow-left"
              severity="secondary"
              [outlined]="true"
            />
          </div>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        @for (card of cards(); track card.key) {
          <p-card styleClass="h-full border border-surface bg-surface-0 shadow-sm">
            <div class="flex items-center justify-between gap-3">
              <div class="space-y-1">
                <p class="m-0 text-sm text-muted-color">{{ card.label }}</p>
                <strong class="block text-lg font-semibold">{{ card.value }}</strong>
              </div>
              <span class="grid size-12 place-items-center rounded-2xl border border-surface-200 bg-surface-50">
                <i [class]="card.icon"></i>
              </span>
            </div>
          </p-card>
        }
      </div>

      <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="space-y-1">
            <h2 class="m-0 text-xl font-semibold">Ultimul portofoliu gasit</h2>
            <p class="m-0 text-sm text-muted-color">
              Afisam datele de datare si stare pentru portofoliul asociat profesorului curent.
            </p>
          </div>
          <p-tag [value]="previewStatusLabel()" [severity]="previewSeverity()" />
        </div>

        @if (loadingPreview()) {
          <p class="mt-4 m-0 text-sm text-muted-color">Se incarca portofoliul asociat profesorului...</p>
        } @else if (portfolioPreview(); as preview) {
          <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-xl border border-surface p-3">
              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Cod portofoliu</div>
              <div class="mt-2 text-sm font-medium">{{ preview.portfolio_code }}</div>
            </div>
            <div class="rounded-xl border border-surface p-3">
              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Datare automata</div>
              <div class="mt-2 text-sm font-medium">{{ preview.last_updated_on || '-' }}</div>
              <div class="mt-1 text-xs text-muted-color">Se actualizeaza la sincronizarea opisului.</div>
            </div>
            <div class="rounded-xl border border-surface p-3">
              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Retentie pana la</div>
              <div class="mt-2 text-sm font-medium">{{ preview.retention_until || '-' }}</div>
            </div>
            <div class="rounded-xl border border-surface p-3">
              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Transfer</div>
              <div class="mt-2 text-sm font-medium">{{ displayTransferStatus(preview.transfer_status) }}</div>
            </div>
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            @if (preview.status === 'draft' || preview.status === 'returned') {
              <p-button
                label="Trimite la verificare"
                icon="pi pi-send"
                [loading]="actionLoading() === 'submit'"
                (onClick)="advancePortfolioStatus('submit')"
              />
            }
            @if (preview.status === 'submitted' && canManagePortfolio()) {
              <p-button
                label="Valideaza portofoliul"
                icon="pi pi-check"
                [loading]="actionLoading() === 'validate'"
                (onClick)="advancePortfolioStatus('validate')"
              />
              <p-button
                label="Returneaza pentru completari"
                icon="pi pi-replay"
                severity="secondary"
                [outlined]="true"
                [loading]="actionLoading() === 'return'"
                (onClick)="advancePortfolioStatus('return')"
              />
            }
            @if (preview.status === 'submitted') {
              @if (canManagePortfolio()) {
                <p-button
                  label="Arhiveaza portofoliul"
                  icon="pi pi-box"
                  severity="secondary"
                  [outlined]="true"
                  [loading]="actionLoading() === 'archive'"
                  (onClick)="advancePortfolioStatus('archive')"
                />
              } @else {
                <p-button
                  label="Revino la completare"
                  icon="pi pi-undo"
                  severity="secondary"
                  [outlined]="true"
                  [loading]="actionLoading() === 'withdraw'"
                  (onClick)="advancePortfolioStatus('withdraw')"
                />
              }
            }
          </div>
        } @else {
          <p class="mt-4 m-0 text-sm text-muted-color">Nu exista inca un portofoliu listat pentru titularul curent.</p>
        }

        <div class="mt-4 rounded-xl border border-surface p-4">
          <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Circuit procedural</div>
          <div class="mt-3 grid gap-2 md:grid-cols-4">
            @for (step of workflowSteps(); track step.key) {
              <div class="rounded-lg border border-surface p-3">
                <div class="flex items-center justify-between gap-2">
                  <div class="text-sm font-medium">{{ step.label }}</div>
                  <p-tag [value]="step.state" [severity]="step.severity" />
                </div>
                <p class="mt-2 mb-0 text-sm text-muted-color">{{ step.summary }}</p>
              </div>
            }
          </div>
        </div>
      </section>

      <div class="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="space-y-1">
            <h2 class="m-0 text-xl font-semibold">Actiuni principale</h2>
            <p class="m-0 text-sm text-muted-color">
              Deschidem direct fluxurile pe care profesorul le foloseste pentru a lucra cu propriul portofoliu.
            </p>
          </div>

          <div class="mt-4 grid gap-3">
            @for (action of actions(); track action.key) {
              <article class="rounded-xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="font-medium">{{ action.label }}</div>
                    <p-tag [value]="action.severity === 'primary' ? 'Prioritar' : 'Suport'" severity="secondary" />
                  </div>
                  <p class="m-0 text-sm text-muted-color">{{ action.summary }}</p>
                </div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <p-button
                    [routerLink]="action.route"
                    [queryParams]="action.queryParams"
                    [label]="action.label"
                    [severity]="action.severity ?? 'secondary'"
                    [outlined]="action.severity !== 'primary'"
                    icon="pi pi-arrow-right"
                  />
                </div>
              </article>
            }
          </div>
        </section>

        <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="space-y-1">
            <h2 class="m-0 text-xl font-semibold">Traseu recomandat</h2>
            <p class="m-0 text-sm text-muted-color">
              Ordinea minima recomandata pentru a mentine portofoliul actualizat si valorificabil.
            </p>
          </div>

          <div class="mt-4 grid gap-3">
            @for (item of playbook(); track item.key) {
              <article class="rounded-xl border border-surface p-4">
                <div class="space-y-2">
                  <div class="flex flex-wrap items-start justify-between gap-3">
                    <div class="font-medium">{{ item.label }}</div>
                    <p-tag [value]="item.step" severity="contrast" />
                  </div>
                  <p class="m-0 text-sm text-muted-color">{{ item.summary }}</p>
                </div>
              </article>
            }
          </div>
        </section>
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationPortfolioSelfServiceComponent {
  private readonly http = inject(HttpClient);
  private readonly authz = inject(AuthzService);

  protected readonly institutionName = this.authz.institutionName;
  protected readonly ownerName = computed(() => this.authz.user()?.name?.trim() || 'Profesor');
  protected readonly roleTags = computed(() =>
    this.authz.roles().length ? this.authz.roles().map((role) => this.authz.roleLabel(role)) : ['Profesor'],
  );
  protected readonly canManagePortfolio = computed(() => this.authz.hasPermission('education.portfolios.manage'));
  protected readonly portfolioPreview = signal<PortfolioPreviewRow | null>(null);
  protected readonly loadingPreview = signal(true);
  protected readonly actionLoading = signal('');

  protected readonly cards = computed<PortfolioSelfServiceCard[]>(() => [
    {
      key: 'owner',
      label: 'Titular curent',
      value: this.ownerName(),
      icon: 'pi pi-user',
    },
    {
      key: 'portfolio',
      label: 'Portofoliu disponibil',
      value: this.authz.hasPermission('education.portfolios.read') ? 'Da' : 'Fara acces',
      icon: 'pi pi-folder-open',
    },
    {
      key: 'opis',
      label: 'Opis si indexare',
      value: 'Regenerabil',
      icon: 'pi pi-list',
    },
    {
      key: 'transfer',
      label: 'Transfer digital',
      value: 'Urmarit procedural',
      icon: 'pi pi-send',
    },
  ]);

  protected readonly actions = computed<PortfolioSelfServiceAction[]>(() => [
    {
      key: 'portfolio-list',
      label: 'Deschide portofoliul meu',
      summary: 'Listeaza portofoliile filtrate pe titularul curent si deschide direct registrul relevant.',
      route: '/education/portfolio',
      queryParams: {
        resource: 'portfolios',
        filter_owner_name: this.ownerName(),
        presetLabel: 'Portofoliul meu',
      },
      severity: 'primary',
    },
    {
      key: 'portfolio-wizard',
      label: 'Actualizeaza portofoliul',
      summary: 'Pornește wizard-ul de creare sau completare, cu sectiuni si declaratii minime.',
      route: '/education/portfolio/wizard',
      queryParams: {} as Record<string, string>,
    },
    {
      key: 'portfolio-workflow',
      label: 'Deschide fluxul ghidat',
      summary: 'Acceseaza wizard-ul care permite trimiterea la verificare, revenirea la completare si salvarea starii curente.',
      route: '/education/portfolio/workflow',
      queryParams: {
        owner_name: this.ownerName(),
      },
    },
    {
      key: 'evaluations',
      label: 'Verifica evaluarea mea',
      summary: 'Trece in zona de personal pentru evaluari, criterii si comunicarea rezultatului.',
      route: '/education/personnel',
      queryParams: {
        resource: 'evaluations',
        presetLabel: 'Evaluarile mele',
      },
    },
  ]);

  protected readonly playbook = computed(() => [
    {
      key: 'step-1',
      step: '1',
      label: 'Actualizeaza documentele',
      summary: 'Porneste cu documentele si sectiunile care lipsesc sau sunt expirate.',
    },
    {
      key: 'step-2',
      step: '2',
      label: 'Regenereaza opisul',
      summary: 'Folosește actiunea dedicata pentru a reconstrui indexarea din documentele deja existente.',
    },
    {
      key: 'step-3',
      step: '3',
      label: 'Urmareste transferul',
      summary: 'Verifica statusul de transfer, predarea, receptia si eventualele blocaje procedurale.',
    },
    {
      key: 'step-4',
      step: '4',
      label: 'Legatura cu evaluarea',
      summary: 'Treci spre evaluare si folosirea portofoliului in fluxurile metodologice ulterioare.',
    },
  ]);

  protected readonly previewStatusLabel = computed(() => {
    const preview = this.portfolioPreview();
    if (!preview) {
      return 'Fara date';
    }
    if (preview.status === 'validated') {
      return 'Validat';
    }
    if (preview.status === 'returned') {
      return 'Returnat';
    }
    if (preview.status === 'submitted') {
      return 'In verificare';
    }
    return 'In lucru';
  });

  protected readonly previewSeverity = computed<'success' | 'warn' | 'secondary'>(() => {
    const preview = this.portfolioPreview();
    if (!preview) {
      return 'secondary';
    }
    if (preview.status === 'validated') {
      return 'success';
    }
    if (preview.status === 'returned' || preview.status === 'submitted') {
      return 'warn';
    }
    return 'secondary';
  });

  protected readonly workflowSteps = computed(() => {
    const preview = this.portfolioPreview();
    const status = preview?.status ?? 'draft';
    const isManage = this.authz.hasPermission('education.portfolios.manage');

    return [
      {
        key: 'draft',
        label: 'Completare',
        summary: 'Documentele si sectiunile sunt aduse la zi inainte de trimitere.',
        state: status === 'draft' || status === 'returned' ? 'Activ' : 'Inchis',
        severity: status === 'draft' || status === 'returned' ? ('success' as const) : ('secondary' as const),
      },
      {
        key: 'submitted',
        label: 'Verificare',
        summary: isManage ? 'Directorul sau secretariatul poate confirma sau returna.' : 'Asteptam verificarea institutionala.',
        state: status === 'submitted' ? 'Activ' : 'Inchis',
        severity: status === 'submitted' ? ('warn' as const) : ('secondary' as const),
      },
      {
        key: 'validated',
        label: 'Validare',
        summary: 'Portofoliul devine eligibil pentru raportare si valorificare.',
        state: status === 'validated' ? 'Activ' : 'Inchis',
        severity: status === 'validated' ? ('success' as const) : ('secondary' as const),
      },
      {
        key: 'archived',
        label: 'Arhivare',
        summary: 'Circuitul se incheie si portofoliul intra in regim de pastrare.',
        state: status === 'archived' ? 'Activ' : 'Inchis',
        severity: status === 'archived' ? ('secondary' as const) : ('secondary' as const),
      },
    ];
  });

  async ngOnInit(): Promise<void> {
    await this.refreshPreview();
  }

  protected async advancePortfolioStatus(action: 'submit' | 'withdraw' | 'validate' | 'return' | 'archive'): Promise<void> {
    const preview = this.portfolioPreview();
    if (!preview || this.actionLoading()) {
      return;
    }

    const nextStatus =
      action === 'submit'
        ? 'submitted'
        : action === 'validate'
          ? 'validated'
          : action === 'archive'
            ? 'archived'
            : 'draft';
    this.actionLoading.set(action);

    try {
      await firstValueFrom(
        this.http.patch<PortfolioPreviewRow>(`/api/education/portfolios/records/${preview.id}`, {
          owner_name: preview.owner_name,
          owner_role: preview.owner_role,
          school_year: preview.school_year,
          status: nextStatus,
          section_count: preview.section_count,
          last_updated_on: preview.last_updated_on || preview.retention_until || new Date().toISOString().slice(0, 10),
          retention_until: preview.retention_until,
          transfer_status: preview.transfer_status,
          authenticity_declared: true,
          consent_captured: true,
          custodian: 'Secretariat',
          notes: `Actualizat procedural prin fluxul profesorului: ${action}.`,
        }),
      );
    } finally {
      this.actionLoading.set('');
    }

    await this.refreshPreview();
  }

  protected displayTransferStatus(value: string): string {
    switch (value) {
      case 'prepared':
        return 'Pregatit';
      case 'sent':
        return 'Trimis';
      case 'received':
        return 'Receptionat';
      case 'none':
      default:
        return 'Fara transfer';
    }
  }

  private async refreshPreview(): Promise<void> {
    this.loadingPreview.set(true);
    try {
      const params = new HttpParams()
        .set('page', '1')
        .set('pageSize', '5')
        .set('filter.owner_name', this.ownerName())
        .set('sort', 'last_updated_on')
        .set('direction', 'desc');
      const response = await firstValueFrom(
        this.http.get<{ items?: PortfolioPreviewRow[] }>('/api/education/portfolios/records', { params }),
      );
      this.portfolioPreview.set((response.items ?? [])[0] ?? null);
    } catch {
      this.portfolioPreview.set(null);
    } finally {
      this.loadingPreview.set(false);
    }
  }
}
