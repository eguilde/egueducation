import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { CheckboxModule } from 'primeng/checkbox';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { MessageModule } from 'primeng/message';
import { MultiSelectModule } from 'primeng/multiselect';
import { SelectModule } from 'primeng/select';
import { TableModule } from 'primeng/table';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TooltipModule } from 'primeng/tooltip';

import {
  RegistraturaAdminApiService,
  RegistraturaDepartment,
  RegistraturaOrganization,
  RegistraturaOrganizationChartNode,
  RegistraturaPartyAdmin,
  RegistraturaRegistryAdmin,
  RegistraturaUserAssignment,
} from './registratura-admin-api.service';

interface AdminUserOption {
  id: string;
  name?: string;
  email?: string;
}

type Resource =
  | 'departments'
  | 'registries'
  | 'physical'
  | 'legal'
  | 'institution'
  | 'organizations'
  | 'chart'
  | 'assignments';

@Component({
  selector: 'app-registratura-admin-workspace',
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    CardModule,
    CheckboxModule,
    DialogModule,
    InputTextModule,
    MessageModule,
    MultiSelectModule,
    SelectModule,
    TableModule,
    TabsModule,
    TagModule,
    TooltipModule,
  ],
  template: `
    <section
      class="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4"
      aria-label="Administrare registratură"
    >
      <header
        class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 class="m-0 text-lg font-semibold">Administrare registratură</h2>
            <p class="m-0 mt-1 text-sm text-muted-color">
              Structura, registrele și corespondenții sunt configurați în contextul instituției
              active.
            </p>
          </div>
          <p-tag value="Izolat pe instituție" icon="pi pi-shield" severity="info" />
        </div>
      </header>

      <p-tabs [value]="activeResource()" (valueChange)="selectResource($event)">
        <p-tablist class="overflow-x-auto">
          @for (tab of resourceTabs; track tab.value) {
            <p-tab [value]="tab.value"
              ><i [class]="tab.icon + ' mr-2'" aria-hidden="true"></i>{{ tab.label }}</p-tab
            >
          }
        </p-tablist>
        <p-tabpanels class="p-0">
          @for (tab of resourceTabs; track tab.value) {
            <p-tabpanel [value]="tab.value" class="p-0">
              @if (tab.value === 'chart') {
                <ng-container [ngTemplateOutlet]="chartTemplate" />
              } @else if (tab.value === 'assignments') {
                <ng-container [ngTemplateOutlet]="assignmentsTemplate" />
              } @else {
                <ng-container [ngTemplateOutlet]="tableTemplate" />
              }
            </p-tabpanel>
          }
        </p-tabpanels>
      </p-tabs>
    </section>

    <ng-template #tableTemplate>
      <section
        class="mt-4 rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900"
      >
        <div class="mb-3 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 class="m-0 text-base font-semibold">{{ currentTitle() }}</h3>
            <p class="m-0 mt-1 text-sm text-muted-color">{{ currentDescription() }}</p>
          </div>
          <div class="flex gap-2">
            <p-button
              icon="pi pi-refresh"
              label="Reîncarcă"
              severity="secondary"
              [outlined]="true"
              [loading]="loading()"
              (onClick)="loadCurrent()"
            />
            <p-button icon="pi pi-plus" [label]="addLabel()" (onClick)="openCreate()" />
          </div>
        </div>
        <div class="mb-3 flex gap-2">
          <label class="sr-only" for="registratura-admin-search">Caută</label>
          <input
            id="registratura-admin-search"
            pInputText
            class="w-full md:max-w-md"
            placeholder="Caută în lista curentă"
            [(ngModel)]="query"
            (keyup.enter)="loadCurrent()"
          />
          <p-button icon="pi pi-search" ariaLabel="Caută" (onClick)="loadCurrent()" />
        </div>
        @if (error()) {
          <p-message class="mb-3" severity="error">{{ error() }}</p-message>
        }
        <p-table
          [value]="rows()"
          [loading]="loading()"
          styleClass="p-datatable-sm p-datatable-striped"
          [scrollable]="true"
          scrollHeight="36rem"
        >
          <ng-template pTemplate="header">
            <tr>
              @for (column of columns(); track column) {
                <th>{{ column }}</th>
              }
              <th class="w-28 text-center">Acțiuni</th>
            </tr>
          </ng-template>
          <ng-template pTemplate="body" let-row>
            <tr>
              @for (column of columns(); track column) {
                <td>{{ cell(row, column) }}</td>
              }
              <td class="text-center whitespace-nowrap">
                <p-button
                  icon="pi pi-pencil"
                  [rounded]="true"
                  [text]="true"
                  ariaLabel="Editează"
                  pTooltip="Editează"
                  (onClick)="openEdit(row)"
                />
                <p-button
                  icon="pi pi-trash"
                  severity="danger"
                  [rounded]="true"
                  [text]="true"
                  ariaLabel="Șterge"
                  pTooltip="Șterge"
                  (onClick)="remove(row)"
                />
              </td>
            </tr>
          </ng-template>
          <ng-template pTemplate="emptymessage">
            <tr>
              <td [attr.colspan]="columns().length + 1" class="py-8 text-center text-muted-color">
                Nu există înregistrări pentru acest context sau pentru filtrul curent.
              </td>
            </tr>
          </ng-template>
        </p-table>
      </section>
    </ng-template>

    <ng-template #chartTemplate>
      <section
        class="mt-4 rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900"
      >
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 class="m-0 text-base font-semibold">Organigramă</h3>
            <p class="m-0 mt-1 text-sm text-muted-color">
              Ierarhia compartimentelor și utilizatorii alocați, în contextul instituției active.
            </p>
          </div>
          <div class="flex items-center gap-2">
            <p-button
              icon="pi pi-minus"
              ariaLabel="Micșorează organigrama"
              [text]="true"
              (onClick)="decreaseZoom()"
            /><span class="w-12 text-center text-sm" aria-live="polite">{{ zoom() }}%</span
            ><p-button
              icon="pi pi-plus"
              ariaLabel="Mărește organigrama"
              [text]="true"
              (onClick)="increaseZoom()"
            /><p-button
              label="Reset"
              severity="secondary"
              [outlined]="true"
              (onClick)="zoom.set(100)"
            /><p-button
              icon="pi pi-refresh"
              ariaLabel="Reîncarcă organigrama"
              [loading]="chartLoading()"
              (onClick)="loadChart()"
            />
          </div>
        </div>
        @if (chartError()) {
          <p-message severity="error">{{ chartError() }}</p-message>
        }
        @if (chartLoading()) {
          <div class="py-8 text-center text-muted-color">Se încarcă organigrama…</div>
        } @else if (chart().length === 0) {
          <div
            class="rounded-xl border border-dashed border-surface-300 p-8 text-center text-muted-color"
          >
            Nu sunt configurate compartimente pentru organigramă.
          </div>
        } @else {
          <div class="min-h-72 overflow-auto pb-3">
            <div
              class="flex min-w-max gap-6 p-2"
              [style.transform]="'scale(' + zoom() / 100 + ')'"
              [style.transform-origin]="'top left'"
            >
              @for (node of chart(); track node.id) {
                <ng-container
                  [ngTemplateOutlet]="nodeTemplate"
                  [ngTemplateOutletContext]="{ $implicit: node }"
                />
              }
            </div>
          </div>
        }
      </section>
    </ng-template>

    <ng-template #nodeTemplate let-node>
      <article
        class="min-w-60 max-w-72 rounded-xl border border-surface bg-surface-0 p-4 dark:bg-surface-900"
      >
        <div class="font-semibold">{{ node.name }}</div>
        @if (node.role_tag) {
          <p-tag [value]="node.role_tag" severity="secondary" />
        }
        <div class="mt-2 flex flex-wrap gap-1">
          @for (user of node.users; track user.id) {
            <p-tag [value]="user.name || user.email || 'Utilizator'" severity="info" />
          } @empty {
            <span class="text-xs text-muted-color">Niciun utilizator alocat</span>
          }
        </div>
        @if (node.children?.length) {
          <div class="mt-4 flex gap-4 border-t border-dashed border-surface pt-4">
            @for (child of node.children; track child.id) {
              <ng-container
                [ngTemplateOutlet]="nodeTemplate"
                [ngTemplateOutletContext]="{ $implicit: child }"
              />
            }
          </div>
        }
      </article>
    </ng-template>

    <ng-template #assignmentsTemplate>
      <section
        class="mt-4 rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900"
      >
        <div class="mb-3">
          <h3 class="m-0 text-base font-semibold">Atribuiri utilizator</h3>
          <p class="m-0 mt-1 text-sm text-muted-color">
            Compartimentele și organizația sunt atribuiri locale instituției active; nu modifică
            rolurile globale.
          </p>
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Utilizator</span
            ><p-select
              [options]="users()"
              optionLabel="name"
              optionValue="id"
              placeholder="Selectează utilizator"
              [(ngModel)]="assignmentUserId"
              (onChange)="loadAssignments()"
              [filter]="true"
              filterBy="name,email"
          /></label>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Organizație</span
            ><p-select
              [options]="organizationOptions()"
              optionLabel="name"
              optionValue="id"
              placeholder="Fără organizație"
              [(ngModel)]="assignment.organization_id"
              [showClear]="true"
          /></label>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color md:col-span-2"
            ><span>Compartimente</span
            ><p-multiselect [options]="departmentOptions()" optionLabel="name" optionValue="id" [(ngModel)]="assignment.department_ids" placeholder="Selectează compartimente" [filter]="true"
          /></label>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Compartiment principal</span
            ><p-select
              [options]="departmentOptions()"
              optionLabel="name"
              optionValue="id"
              placeholder="Fără compartiment principal"
              [(ngModel)]="assignment.primary_department_id"
              [showClear]="true"
          /></label>
        </div>
        @if (assignmentError()) {
          <p-message class="mt-3" severity="error">{{ assignmentError() }}</p-message>
        }
        <div class="mt-4 flex justify-end">
          <p-button
            label="Salvează atribuiri"
            icon="pi pi-save"
            [disabled]="!assignmentUserId"
            [loading]="assignmentSaving()"
            (onClick)="saveAssignments()"
          />
        </div>
      </section>
    </ng-template>

    <p-dialog
      [visible]="dialogOpen()"
      (visibleChange)="dialogOpen.set($event)"
      [header]="dialogTitle()"
      [modal]="true"
      styleClass="w-[94vw] max-w-3xl"
    >
      <div class="grid gap-3" [ngSwitch]="activeResource()">
        <ng-container *ngSwitchCase="'departments'">
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Nume *</span><input pInputText [(ngModel)]="departmentDraft.name"
          /></label>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Descriere</span><input pInputText [(ngModel)]="departmentDraft.description"
          /></label>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Compartiment părinte</span
            ><p-select
              [options]="departmentOptions()"
              optionLabel="name"
              optionValue="id"
              [(ngModel)]="departmentDraft.parent_id"
              [showClear]="true"
          /></label>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Etichetă rol</span><input pInputText [(ngModel)]="departmentDraft.role_tag"
          /></label>
        </ng-container>
        <ng-container *ngSwitchCase="'registries'">
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Nume *</span><input pInputText [(ngModel)]="registryDraft.name"
          /></label>
          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Prefix</span><input pInputText [(ngModel)]="registryDraft.prefix" /></label
            ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Tip</span
              ><p-select
                [options]="registryTypes"
                optionLabel="label"
                optionValue="value"
                [(ngModel)]="registryDraft.registry_type"
            /></label>
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Start</span
              ><input pInputText type="number" [(ngModel)]="registryDraft.start_number" /></label
            ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Curent</span
              ><input pInputText type="number" [(ngModel)]="registryDraft.current_number" /></label
            ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Următor</span
              ><input pInputText type="number" [(ngModel)]="registryDraft.next_number"
            /></label>
          </div>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Compartimente</span
            ><p-multiselect [options]="departmentOptions()" optionLabel="name" optionValue="id" [(ngModel)]="registryDraft.department_ids" placeholder="Selectează compartimente" [filter]="true"
          /></label>
          <div class="flex items-center gap-2">
            <p-checkbox
              inputId="registry-default"
              [binary]="true"
              [(ngModel)]="registryDraft.is_default"
            /><label for="registry-default">Registru implicit</label>
          </div>
        </ng-container>
        <ng-container *ngSwitchCase="'organizations'">
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Nume *</span><input pInputText [(ngModel)]="organizationDraft.name" /></label
          ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Descriere</span
            ><input pInputText [(ngModel)]="organizationDraft.description" /></label
          ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>Compartimente</span
            ><p-multiselect [options]="departmentOptions()" optionLabel="name" optionValue="id" [(ngModel)]="organizationDraft.department_ids" placeholder="Selectează compartimente" [filter]="true"
          /></label>
          <div class="flex gap-4">
            <div class="flex items-center gap-2">
              <p-checkbox
                inputId="organization-active"
                [binary]="true"
                [(ngModel)]="organizationDraft.active"
              /><label for="organization-active">Activă</label>
            </div>
            <div class="flex items-center gap-2">
              <p-checkbox
                inputId="organization-default"
                [binary]="true"
                [(ngModel)]="organizationDraft.is_default"
              /><label for="organization-default">Implicită</label>
            </div>
          </div>
        </ng-container>
        <ng-container *ngSwitchDefault>
          <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
            ><span>{{ activeResource() === 'physical' ? 'Nume' : 'Denumire' }} *</span
            ><input pInputText [(ngModel)]="partyDraft.display_name"
          /></label>
          @if (activeResource() === 'physical') {
            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
                ><span>Prenume</span><input pInputText [(ngModel)]="partyDraft.first_name" /></label
              ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
                ><span>Nume</span><input pInputText [(ngModel)]="partyDraft.last_name"
              /></label>
            </div>
            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Data nașterii</span><input pInputText type="date" [(ngModel)]="partyDraft.birth_date" /></label>
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Locul nașterii</span><input pInputText [(ngModel)]="partyDraft.birth_place" /></label>
            </div>
          }
          @if (activeResource() !== 'physical') {
            <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Denumire legală</span><input pInputText [(ngModel)]="partyDraft.legal_name"
            /></label>
          }
          <div class="grid gap-3 md:grid-cols-2">
            <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Identificator</span
              ><input pInputText [(ngModel)]="partyDraft.identifier_code" /></label
            ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>CUI / Cod fiscal</span
              ><input pInputText [(ngModel)]="partyDraft.tax_id" /></label
            ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Email</span><input pInputText [(ngModel)]="partyDraft.email" /></label
            ><label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"
              ><span>Telefon</span><input pInputText [(ngModel)]="partyDraft.phone_number"
            /></label>
          </div>
          @if (activeResource() === 'legal') {
            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Nr. Registrul Comerțului</span><input pInputText [(ngModel)]="partyDraft.trade_register_number" /></label>
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Reprezentant legal</span><input pInputText [(ngModel)]="partyDraft.legal_representative" /></label>
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Formă juridică</span><input pInputText [(ngModel)]="partyDraft.legal_form" /></label>
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Capital social</span><input pInputText [(ngModel)]="partyDraft.share_capital" /></label>
            </div>
          }
          @if (activeResource() === 'institution') {
            <div class="grid gap-3 md:grid-cols-2">
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Tip instituție</span><input pInputText [(ngModel)]="partyDraft.institution_type" /></label>
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color"><span>Nivel instituție</span><input pInputText [(ngModel)]="partyDraft.institution_level" /></label>
              <label class="flex flex-col gap-1.5 text-sm font-semibold text-muted-color md:col-span-2"><span>Website</span><input pInputText type="url" [(ngModel)]="partyDraft.website" /></label>
            </div>
          }
          <div class="flex gap-4">
            <div class="flex items-center gap-2">
              <p-checkbox
                inputId="party-active"
                [binary]="true"
                [(ngModel)]="partyDraft.active"
              /><label for="party-active">Activ</label>
            </div>
            <div class="flex items-center gap-2">
              <p-checkbox
                inputId="party-default-organization"
                [binary]="true"
                [(ngModel)]="partyDraft.is_default_organization"
              /><label for="party-default-organization">Instituție implicită</label>
            </div>
          </div>
        </ng-container>
        @if (dialogError()) {
          <p-message severity="error">{{ dialogError() }}</p-message>
        }
      </div>
      <ng-template pTemplate="footer"
        ><p-button
          label="Renunță"
          severity="secondary"
          [outlined]="true"
          (onClick)="dialogOpen.set(false)" /><p-button
          label="Salvează"
          icon="pi pi-save"
          [loading]="saving()"
          [disabled]="!draftValid()"
          (onClick)="saveDraft()"
      /></ng-template>
    </p-dialog>
  `,
  host: { class: 'block min-h-0' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RegistraturaAdminWorkspaceComponent implements OnInit {
  readonly users = input<AdminUserOption[]>([]);
  readonly initialResource = input<Resource>('departments');
  protected readonly api = inject(RegistraturaAdminApiService);
  protected readonly activeResource = signal<Resource>('departments');
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly rows = signal<Array<Record<string, unknown>>>([]);
  protected query = '';
  protected readonly dialogOpen = signal(false);
  protected readonly saving = signal(false);
  protected readonly dialogError = signal('');
  protected readonly chart = signal<RegistraturaOrganizationChartNode[]>([]);
  protected readonly chartLoading = signal(false);
  protected readonly chartError = signal('');
  protected readonly zoom = signal(100);
  protected readonly departments = signal<RegistraturaDepartment[]>([]);
  protected readonly organizations = signal<RegistraturaOrganization[]>([]);
  protected assignmentUserId = '';
  protected assignment: Omit<RegistraturaUserAssignment, 'user_id'> = {
    department_ids: [],
    primary_department_id: null,
    organization_id: null,
  };
  protected readonly assignmentSaving = signal(false);
  protected readonly assignmentError = signal('');
  protected departmentDraft: Partial<RegistraturaDepartment> = {};
  protected registryDraft: Partial<RegistraturaRegistryAdmin> = {
    registry_type: 'public',
    start_number: 1,
    current_number: 0,
    next_number: 1,
    is_default: false,
  };
  protected organizationDraft: Partial<RegistraturaOrganization> = {
    active: true,
    is_default: false,
  };
  protected partyDraft: Partial<RegistraturaPartyAdmin> = {
    active: true,
    is_default_organization: false,
  };
  protected readonly registryTypes = [
    { label: 'Public', value: 'public' },
    { label: 'Privat', value: 'private' },
  ];
  protected readonly resourceTabs: Array<{ value: Resource; label: string; icon: string }> = [
    { value: 'departments', label: 'Compartimente', icon: 'pi pi-sitemap' },
    { value: 'registries', label: 'Registre', icon: 'pi pi-book' },
    { value: 'physical', label: 'Persoane fizice', icon: 'pi pi-id-card' },
    { value: 'legal', label: 'Persoane juridice', icon: 'pi pi-building' },
    { value: 'institution', label: 'Instituții publice', icon: 'pi pi-landmark' },
    { value: 'organizations', label: 'Organizații', icon: 'pi pi-warehouse' },
    { value: 'chart', label: 'Organigramă', icon: 'pi pi-share-alt' },
    { value: 'assignments', label: 'Atribuiri', icon: 'pi pi-users' },
  ];
  protected readonly departmentOptions = computed(() =>
    this.departments().map((item) => ({ id: item.id, name: item.name })),
  );
  protected readonly organizationOptions = computed(() =>
    this.organizations()
      .filter((item) => item.active)
      .map((item) => ({ id: item.id, name: item.name })),
  );

  ngOnInit(): void {
    this.activeResource.set(this.initialResource());
    this.loadReferenceData();
    this.loadCurrent();
  }

  protected selectResource(value: string | number | undefined): void {
    const next = String(value) as Resource;
    this.activeResource.set(next);
    if (next === 'chart') {
      this.loadChart();
      return;
    }
    if (next !== 'assignments') {
      this.loadCurrent();
    }
  }

  protected columns = computed(() => {
    switch (this.activeResource()) {
      case 'departments':
        return ['Nume', 'Descriere', 'Părinte', 'Rol', 'Utilizatori'];
      case 'registries':
        return ['Nume', 'Prefix', 'Tip', 'Următorul număr', 'Implicit'];
      case 'organizations':
        return ['Nume', 'Descriere', 'Stare', 'Implicit', 'Compartimente'];
      case 'physical':
        return ['Nume', 'Identificator', 'Email', 'Telefon', 'Stare'];
      case 'legal':
        return ['Denumire', 'CUI', 'Email', 'Telefon', 'Stare'];
      default:
        return ['Denumire', 'Tip', 'Email', 'Telefon', 'Implicit'];
    }
  });
  protected currentTitle = computed(
    () =>
      this.resourceTabs.find((tab) => tab.value === this.activeResource())?.label ?? 'Administrare',
  );
  protected currentDescription = computed(
    () =>
      (
        ({
          departments: 'Compartimentele sunt asociate utilizatorilor, registrelor și documentelor.',
          registries:
            'Registrele sunt vizibile numai utilizatorilor autorizați în instituția activă.',
          physical: 'Persoane fizice disponibile pentru emitent și destinatar.',
          legal: 'Persoane juridice disponibile pentru emitent și destinatar.',
          institution: 'Instituții publice și instituția implicită a tenantului.',
          organizations: 'Organizații locale, active și implicite, cu compartimente alocate.',
        }) as Partial<Record<Resource, string>>
      )[this.activeResource()] ?? '',
  );
  protected addLabel = computed(
    () => `Adaugă ${this.currentTitle().replace(/^./, (letter) => letter.toLowerCase())}`,
  );
  protected dialogTitle = computed(
    () =>
      `${this.editingId() ? 'Editează' : 'Adaugă'} ${this.currentTitle().replace(/^./, (letter) => letter.toLowerCase())}`,
  );
  protected readonly editingId = signal<string | number | null>(null);
  protected draftValid = computed(() => {
    const resource = this.activeResource();
    if (resource === 'departments') return Boolean(this.departmentDraft.name?.trim());
    if (resource === 'registries') return Boolean(this.registryDraft.name?.trim());
    if (resource === 'organizations') return Boolean(this.organizationDraft.name?.trim());
    return Boolean(this.partyDraft.display_name?.trim());
  });

  protected cell(row: Record<string, unknown>, column: string): string {
    const mappings: Record<string, keyof typeof row> = {
      Nume: 'name',
      Descriere: 'description',
      Părinte: 'parent_id',
      Rol: 'role_tag',
      Utilizatori: 'user_count',
      Prefix: 'prefix',
      Tip: 'registry_type',
      'Următorul număr': 'next_number',
      Implicit: 'is_default',
      Stare: 'active',
      Compartimente: 'department_ids',
      Identificator: 'identifier_code',
      Denumire: 'display_name',
      CUI: 'tax_id',
      Email: 'email',
      Telefon: 'phone_number',
    };
    const value = row[mappings[column]];
    if (Array.isArray(value)) return value.length ? String(value.length) : '-';
    if (typeof value === 'boolean') return value ? 'Da' : 'Nu';
    return value === null || value === undefined || value === '' ? '-' : String(value);
  }

  protected async loadCurrent(): Promise<void> {
    const resource = this.activeResource();
    if (resource === 'chart' || resource === 'assignments') return;
    this.loading.set(true);
    this.error.set('');
    try {
      const query = { page: 1, pageSize: 100, query: this.query };
      const response =
        resource === 'departments'
          ? await firstValueFrom(this.api.departments(query))
          : resource === 'registries'
            ? await firstValueFrom(this.api.registries(query))
            : resource === 'organizations'
              ? await firstValueFrom(this.api.organizations(query))
              : await firstValueFrom(this.api.parties(resource, query));
      this.rows.set(response.items as unknown as Array<Record<string, unknown>>);
    } catch {
      this.rows.set([]);
      this.error.set(
        'Datele nu au putut fi încărcate. Verifică drepturile și conexiunea la serviciul de administrare.',
      );
    } finally {
      this.loading.set(false);
    }
  }

  protected async loadReferenceData(): Promise<void> {
    try {
      this.departments.set(
        (await firstValueFrom(this.api.departments({ page: 1, pageSize: 250 }))).items,
      );
    } catch {
      this.departments.set([]);
    }
    try {
      this.organizations.set(
        (await firstValueFrom(this.api.organizations({ page: 1, pageSize: 250 }))).items,
      );
    } catch {
      this.organizations.set([]);
    }
  }

  protected openCreate(): void {
    this.editingId.set(null);
    this.resetDraft();
    this.dialogError.set('');
    this.dialogOpen.set(true);
  }
  protected openEdit(row: Record<string, unknown>): void {
    this.editingId.set((row['id'] as string | number) ?? null);
    this.dialogError.set('');
    const resource = this.activeResource();
    if (resource === 'departments')
      this.departmentDraft = { ...(row as unknown as RegistraturaDepartment) };
    else if (resource === 'registries') {
      this.registryDraft = { ...(row as unknown as RegistraturaRegistryAdmin) };
    } else if (resource === 'organizations') {
      this.organizationDraft = { ...(row as unknown as RegistraturaOrganization) };
    } else this.partyDraft = { ...(row as unknown as RegistraturaPartyAdmin) };
    this.dialogOpen.set(true);
  }
  protected async saveDraft(): Promise<void> {
    if (!this.draftValid()) return;
    this.saving.set(true);
    this.dialogError.set('');
    const resource = this.activeResource();
    try {
      if (resource === 'departments')
        await firstValueFrom(this.api.saveDepartment(this.departmentDraft));
      else if (resource === 'registries')
        await firstValueFrom(
          this.api.saveRegistry({
            ...this.registryDraft,
            department_ids: this.registryDraft.department_ids ?? [],
          }),
        );
      else if (resource === 'organizations')
        await firstValueFrom(
          this.api.saveOrganization({
            ...this.organizationDraft,
            department_ids: this.organizationDraft.department_ids ?? [],
          }),
        );
      else
        await firstValueFrom(
          this.api.saveParty({
            ...this.partyDraft,
            party_type: resource as RegistraturaPartyAdmin['party_type'],
          }),
        );
      this.dialogOpen.set(false);
      await this.loadReferenceData();
      await this.loadCurrent();
    } catch {
      this.dialogError.set(
        'Salvarea a eșuat. Nu s-a aplicat nicio modificare confirmată. Verifică datele și drepturile de administrare.',
      );
    } finally {
      this.saving.set(false);
    }
  }
  protected async remove(row: Record<string, unknown>): Promise<void> {
    const id = row['id'];
    if (
      id === undefined ||
      !window.confirm(
        'Confirmi ștergerea? Elementele folosite de documente pot fi refuzate de server.',
      )
    )
      return;
    this.error.set('');
    try {
      const resource = this.activeResource();
      if (resource === 'departments') await firstValueFrom(this.api.deleteDepartment(String(id)));
      else if (resource === 'registries') await firstValueFrom(this.api.deleteRegistry(Number(id)));
      else if (resource === 'organizations')
        await firstValueFrom(this.api.deleteOrganization(String(id)));
      else await firstValueFrom(this.api.deleteParty(String(id)));
      await this.loadReferenceData();
      await this.loadCurrent();
    } catch {
      this.error.set(
        'Ștergerea a fost refuzată. Datele referite trebuie păstrate sau migrate înainte de eliminare.',
      );
    }
  }
  protected async loadChart(): Promise<void> {
    this.chartLoading.set(true);
    this.chartError.set('');
    try {
      this.chart.set(await firstValueFrom(this.api.chart()));
    } catch {
      this.chart.set([]);
      this.chartError.set('Organigrama nu a putut fi încărcată.');
    } finally {
      this.chartLoading.set(false);
    }
  }
  protected async loadAssignments(): Promise<void> {
    if (!this.assignmentUserId) return;
    this.assignmentError.set('');
    try {
      const item = await firstValueFrom(this.api.userAssignments(this.assignmentUserId));
      this.assignment = {
        department_ids: item.department_ids,
        primary_department_id: item.primary_department_id ?? null,
        organization_id: item.organization_id ?? null,
      };
    } catch {
      this.assignment = { department_ids: [], primary_department_id: null, organization_id: null };
      this.assignmentError.set('Atribuirile utilizatorului nu au putut fi încărcate.');
    }
  }
  protected async saveAssignments(): Promise<void> {
    if (!this.assignmentUserId) return;
    this.assignmentSaving.set(true);
    this.assignmentError.set('');
    try {
      const assignment = {
        ...this.assignment,
        department_ids: this.assignment.department_ids,
      };
      await firstValueFrom(this.api.saveUserAssignments(this.assignmentUserId, assignment));
      this.assignment = assignment;
    } catch {
      this.assignmentError.set(
        'Atribuirile nu au putut fi salvate. Compartimentele și organizația trebuie să aparțină aceleiași instituții.',
      );
    } finally {
      this.assignmentSaving.set(false);
    }
  }
  protected decreaseZoom(): void {
    this.zoom.set(Math.max(60, this.zoom() - 15));
  }
  protected increaseZoom(): void {
    this.zoom.set(Math.min(150, this.zoom() + 15));
  }
  private resetDraft(): void {
    this.departmentDraft = {};
    this.registryDraft = {
      registry_type: 'public',
      start_number: 1,
      current_number: 0,
      next_number: 1,
      is_default: false,
    };
    this.organizationDraft = { active: true, is_default: false };
    this.partyDraft = { active: true, is_default_organization: false };
  }
}
