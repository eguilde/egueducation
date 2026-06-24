import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, computed, effect, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { ButtonGroupModule } from 'primeng/buttongroup';
import { AutoCompleteModule, AutoCompleteCompleteEvent, AutoCompleteSelectEvent } from 'primeng/autocomplete';
import { CardModule } from 'primeng/card';
import { DatePickerModule } from 'primeng/datepicker';
import { DialogModule } from 'primeng/dialog';
import { DrawerModule } from 'primeng/drawer';
import { FieldsetModule } from 'primeng/fieldset';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { ToastModule } from 'primeng/toast';
import { TableLazyLoadEvent, TableModule } from 'primeng/table';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { TooltipModule } from 'primeng/tooltip';
import { MessageService } from 'primeng/api';
import { catchError, forkJoin, firstValueFrom, of } from 'rxjs';

import {
  BatchCreateRegistraturaDocumentRequest,
  CancelRegistraturaDocumentRequest,
  CreateRegistraturaPartyRequest,
  CreateRegistraturaDocumentRequest,
  CreateDocumentLinkRequest,
  ExportRegistraturaDocumentsRequest,
  DocumentLookupItem,
  LinkedDocument,
  RegistraturaDocument,
  RegistraturaDocumentAttachment,
  RegistraturaDocumentFilters,
  RegistraturaDocumentVersion,
  RegistraturaParty,
  RegistraturaRegistry,
  UpdateRegistraturaDocumentRequest,
  TableQuery,
} from '../../core/api/api.types';
import { DocumentLinksApiService } from '../../core/api/document-links-api.service';
import { AppApiService } from '../../core/api/app-api.service';
import { RegistraturaApiService } from '../../core/api/registratura-api.service';
import { AuthzService } from '../../core/authz/authz.service';
import { AppBootstrapConfig } from '../../core/branding/app-branding.types';

interface FilterState {
  registry_number: string;
  subject: string;
  document_type: string;
  direction: string;
  status: string;
  correspondent: string;
  assigned_to: string;
  confidentiality: string;
  registered_at_from: string;
  registered_at_to: string;
  due_date_from: string;
  due_date_to: string;
}

const emptyNewDocument = (): CreateRegistraturaDocumentRequest => ({
  subject: '',
  document_type: '',
  direction: 'intrare',
  status: 'draft',
  correspondent: '',
  assigned_to: '',
  correspondent_party_id: null,
  assigned_party_id: null,
  confidentiality: 'normal',
  summary: '',
  due_date: null,
});

const emptyBatchDocument = (): BatchCreateRegistraturaDocumentRequest => ({
  registru_id: 0,
  count: 1,
  subject: '',
  document_type: '',
  direction: 'intrare',
  status: 'draft',
  correspondent: '',
  assigned_to: '',
  correspondent_party_id: null,
  assigned_party_id: null,
  confidentiality: 'normal',
  summary: '',
  due_date: null,
});

const emptyExportRequest = (): ExportRegistraturaDocumentsRequest => ({
  registru_id: null,
  start_date: null,
  end_date: null,
});

const emptyEditDocument = (): UpdateRegistraturaDocumentRequest => ({
  registru_id: null,
  subject: '',
  document_type: '',
  direction: 'intrare',
  status: 'draft',
  correspondent: '',
  assigned_to: '',
  correspondent_party_id: null,
  assigned_party_id: null,
  confidentiality: 'normal',
  summary: '',
  due_date: null,
  change_notes: '',
});

const emptyPartyForm = (): CreateRegistraturaPartyRequest => ({
  institution_id: '',
  code: '',
  party_type: 'physical',
  display_name: '',
  short_name: '',
  first_name: '',
  last_name: '',
  legal_name: '',
  identifier_code: '',
  tax_id: '',
  phone_number: '',
  email: '',
  address_line1: '',
  address_line2: '',
  locality: '',
  county: '',
  country: 'RO',
  notes: '',
  is_default_organization: false,
  active: true,
});

const defaultRegistryFallback = (): RegistraturaRegistry => ({
  id: 1,
  nume: 'Registru General',
  prefix_nr: 'RG',
  nr_inceput: 1,
  nr_curent: '1',
  nr_urmator: '2',
  data_resetare: null,
  tip_registru: 'general',
  isDefault: true,
  created_at: '',
  updated_at: '',
});

@Component({
  selector: 'app-documente-workspace',
  imports: [
    CommonModule,
    FormsModule,
  ButtonModule,
  ButtonGroupModule,
  AutoCompleteModule,
  CardModule,
    DatePickerModule,
    DialogModule,
    DrawerModule,
    FieldsetModule,
    InputTextModule,
    SelectModule,
    TableModule,
    TabsModule,
    TagModule,
    TextareaModule,
    TooltipModule,
    ToastModule,
  ],
  providers: [MessageService],
  template: `
    <section class="document-workspace flex h-[calc(100dvh-6rem)] min-h-0 flex-col overflow-hidden">
      <p-toast />
      <p-tabs value="registratura" class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <p-tablist class="shrink-0">
          <p-tab value="registratura">Registratură</p-tab>
          <p-tab value="flux">Flux documente</p-tab>
          <p-tab value="arhiva">eArhivă</p-tab>
        </p-tablist>

        <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
          <p-tabpanel value="registratura" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
              <section class="registry-header grid shrink-0 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
                <div class="flex shrink-0 items-center justify-start">
                  <p-button
                    icon="pi pi-search"
                    severity="secondary"
                    [rounded]="true"
                    [text]="true"
                    ariaLabel="Filtre"
                    pTooltip="Filtre"
                    (onClick)="searchPanelOpen.set(true)"
                  />
                </div>

                <div class="flex min-w-0 items-center justify-center gap-2">
                  <p-buttongroup>
                    <p-button icon="pi pi-download" label="Intrare" severity="info" (onClick)="openCreateDialog('intrare')" />
                    <p-button icon="pi pi-upload" label="Ieșire" severity="success" (onClick)="openCreateDialog('iesire')" />
                    <p-button icon="pi pi-clone" label="Multiplu" severity="secondary" [outlined]="true" (onClick)="openBatchDialog()" />
                  </p-buttongroup>
                </div>

                <div class="flex shrink-0 items-center justify-end gap-2">
                  <label class="registry-selector flex shrink-0 flex-col gap-1">
                    <span class="sr-only">Registru activ</span>
                    <p-select
                      class="w-full"
                      appendTo="body"
                      [options]="registryOptions()"
                      optionLabel="nume"
                      optionValue="id"
                      [(ngModel)]="selectedRegistryModel"
                      (ngModelChange)="onRegistryChange($event)"
                      placeholder="Alege registru"
                      [showClear]="false"
                    />
                  </label>
                  <p-button
                    icon="pi pi-file-pdf"
                    severity="secondary"
                    [rounded]="true"
                    [text]="true"
                    ariaLabel="Export PDF"
                    pTooltip="Export PDF"
                    (onClick)="openExportDialog()"
                  />
                </div>
              </section>

              <p-drawer
                [visible]="searchPanelOpen()"
                (visibleChange)="searchPanelOpen.set($event)"
                [modal]="true"
                [dismissible]="true"
                [showCloseIcon]="true"
                position="right"
                styleClass="workspace-filter-drawer"
              >
                <ng-template pTemplate="headless">
                  <p-fieldset legend="Filtrare documente" class="workspace-panel h-full">
                    <div class="flex h-full min-h-0 flex-col">
                      <div class="search-panel-content flex-1 overflow-auto">
                        <div class="search-row">
                          <label class="search-field">
                            <span><i class="pi pi-hashtag"></i> Nr. document</span>
                            <input pInputText [(ngModel)]="filters.registry_number" placeholder="ex: REG-2026-0001" />
                          </label>
                          <label class="search-field">
                            <span><i class="pi pi-tag"></i> Tip document</span>
                            <p-select appendTo="body" [options]="documentTypeOptions()" [(ngModel)]="filters.document_type" placeholder="Toate" [showClear]="true" />
                          </label>
                          <label class="search-field">
                            <span><i class="pi pi-file"></i> Direcție</span>
                            <p-select appendTo="body" [options]="directionOptions()" [(ngModel)]="filters.direction" placeholder="Toate" [showClear]="true" />
                          </label>
                        </div>
                        <div class="search-row">
                        <label class="search-field">
                          <span><i class="pi pi-send"></i> Emitent</span>
                          <p-autoComplete
                            appendTo="body"
                            [(ngModel)]="selectedEmitentSearch"
                            [ngModelOptions]="{ standalone: true }"
                            [suggestions]="filteredEmitentiSearch()"
                            [forceSelection]="false"
                            [dropdown]="false"
                            [minLength]="1"
                            [delay]="150"
                            [optionLabel]="getEntityDisplayText"
                            placeholder="Caută emitent..."
                            (completeMethod)="filterEmitentiSearch($event)"
                            (onSelect)="onEmitentSearchSelect($event)"
                            (onClear)="clearEmitentSearch()"
                          >
                            <ng-template let-entity pTemplate="item">
                              <div class="flex items-center gap-2">
                                <i [class]="getEntityTypeIcon(entity) + ' text-xs'"></i>
                                <span>{{ formatEntityDisplay(entity) }}</span>
                                <span class="ml-auto text-xs text-muted-color">{{ getEntityTypeBadge(entity) }}</span>
                              </div>
                            </ng-template>
                          </p-autoComplete>
                        </label>
                        <label class="search-field">
                          <span><i class="pi pi-inbox"></i> Destinatar</span>
                          <p-autoComplete
                            appendTo="body"
                            [(ngModel)]="selectedDestinatarSearch"
                            [ngModelOptions]="{ standalone: true }"
                            [suggestions]="filteredDestinatariSearch()"
                            [forceSelection]="false"
                            [dropdown]="false"
                            [minLength]="1"
                            [delay]="150"
                            [optionLabel]="getEntityDisplayText"
                            placeholder="Caută destinatar..."
                            (completeMethod)="filterDestinatariSearch($event)"
                            (onSelect)="onDestinatarSearchSelect($event)"
                            (onClear)="clearDestinatarSearch()"
                          >
                            <ng-template let-entity pTemplate="item">
                              <div class="flex items-center gap-2">
                                <i [class]="getEntityTypeIcon(entity) + ' text-xs'"></i>
                                <span>{{ formatEntityDisplay(entity) }}</span>
                                <span class="ml-auto text-xs text-muted-color">{{ getEntityTypeBadge(entity) }}</span>
                              </div>
                            </ng-template>
                          </p-autoComplete>
                        </label>
                          <label class="search-field">
                            <span><i class="pi pi-shield"></i> Confidențialitate</span>
                            <p-select appendTo="body" [options]="confidentialityOptions()" [(ngModel)]="filters.confidentiality" placeholder="Toate" [showClear]="true" />
                          </label>
                        </div>
                        <div class="search-row">
                          <label class="search-field">
                            <span><i class="pi pi-calendar-plus"></i> Data intrare</span>
                            <div class="date-range">
                              <p-datepicker appendTo="body" [(ngModel)]="filters.registered_at_from" dateFormat="yy-mm-dd" placeholder="De la" [showIcon]="true" />
                              <span>—</span>
                              <p-datepicker appendTo="body" [(ngModel)]="filters.registered_at_to" dateFormat="yy-mm-dd" placeholder="Până la" [showIcon]="true" />
                            </div>
                          </label>
                          <label class="search-field">
                            <span><i class="pi pi-calendar-minus"></i> Data scadență</span>
                            <div class="date-range">
                              <p-datepicker appendTo="body" [(ngModel)]="filters.due_date_from" dateFormat="yy-mm-dd" placeholder="De la" [showIcon]="true" />
                              <span>—</span>
                              <p-datepicker appendTo="body" [(ngModel)]="filters.due_date_to" dateFormat="yy-mm-dd" placeholder="Până la" [showIcon]="true" />
                            </div>
                          </label>
                        </div>
                      </div>

                      <footer class="search-panel-footer">
                        <p-button icon="pi pi-refresh" label="Resetare" severity="secondary" [outlined]="true" size="small" (onClick)="resetFilters()" />
                        <p-button icon="pi pi-search" label="Caută documente" severity="primary" size="small" (onClick)="loadDocuments()" />
                      </footer>
                    </div>
                  </p-fieldset>
                </ng-template>
              </p-drawer>

              <p-table
                class="app-data-table-shell mt-3 flex min-h-0 flex-1 flex-col overflow-hidden workspace-table"
                styleClass="p-datatable-sm p-datatable-gridlines registry-table"
                [value]="documents()"
                [tableStyle]="{ width: '100%', 'min-width': '100%' }"
                [lazy]="true"
                [loading]="loading()"
                [scrollable]="true"
                scrollHeight="flex"
                [stripedRows]="true"
                [paginator]="true"
                [rows]="pageSize()"
                [rowsPerPageOptions]="[10, 20, 50, 100]"
                [totalRecords]="totalRecords()"
                [showCurrentPageReport]="true"
                currentPageReportTemplate="Se afișează {first} - {last} din {totalRecords} documente"
                dataKey="id"
                (onLazyLoad)="loadDocuments($event)"
              >
                <ng-template pTemplate="header">
                  <tr>
                    <th style="width: 3rem"></th>
                    <th pSortableColumn="registry_number" style="width: 9rem">Nr. Doc <p-sortIcon field="registry_number" /></th>
                    <th pSortableColumn="document_type" style="width: 8rem">Tip <p-sortIcon field="document_type" /></th>
                    <th pSortableColumn="subject">Conținut <p-sortIcon field="subject" /></th>
                    <th pSortableColumn="correspondent" style="width: 13rem">Emitent <p-sortIcon field="correspondent" /></th>
                    <th pSortableColumn="assigned_to" style="width: 13rem">Destinatar <p-sortIcon field="assigned_to" /></th>
                    <th pSortableColumn="registered_at" style="width: 9rem">Intrare <p-sortIcon field="registered_at" /></th>
                    <th pSortableColumn="due_date" style="width: 9rem">Ieșire <p-sortIcon field="due_date" /></th>
                    <th pSortableColumn="status" style="width: 10rem">Status <p-sortIcon field="status" /></th>
                    <th style="width: 11rem">Acțiuni</th>
                  </tr>
                </ng-template>
                <ng-template pTemplate="body" let-document>
                  <tr>
                    <td>
                      <p-button icon="pi pi-chevron-right" [text]="true" [rounded]="true" severity="secondary" size="small" pTooltip="Deschide detalii" (onClick)="openDocumentDrawer(document)" />
                    </td>
                    <td class="font-mono">{{ document.registry_number }}</td>
                    <td><p-tag [value]="document.document_type" severity="info" /></td>
                    <td class="max-w-[28rem] truncate" [pTooltip]="document.subject">{{ document.subject }}</td>
                    <td>{{ document.correspondent }}</td>
                    <td>{{ document.assigned_to || '-' }}</td>
                    <td>{{ document.registered_at }}</td>
                    <td>{{ document.due_date || '-' }}</td>
                    <td><p-tag [value]="document.status" [severity]="statusSeverity(document.status)" /></td>
                    <td>
                      <div class="flex justify-center gap-1">
                        <p-button icon="pi pi-eye" [rounded]="true" [text]="true" severity="info" size="small" pTooltip="Vizualizează" (onClick)="openDocumentDrawer(document)" />
                        <p-button icon="pi pi-folder-open" [rounded]="true" [text]="true" severity="secondary" size="small" pTooltip="Deschide pagină" (onClick)="openDocumentPage(document)" />
                        <p-button icon="pi pi-file-edit" [rounded]="true" [text]="true" severity="secondary" size="small" pTooltip="Formular editare" (onClick)="openDocumentEditPage(document)" />
                        <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" severity="secondary" size="small" pTooltip="Editează" (onClick)="openEditDialog(document)" />
                        <p-button icon="pi pi-share-alt" [rounded]="true" [text]="true" severity="warn" size="small" pTooltip="Flux" (onClick)="openWorkflowForDocument(document)" />
                        <p-button icon="pi pi-ban" [rounded]="true" [text]="true" severity="danger" size="small" pTooltip="Anulează" (onClick)="openCancelDialog(document)" />
                        <p-button icon="pi pi-print" [rounded]="true" [text]="true" severity="secondary" size="small" pTooltip="Exportă PDF" (onClick)="openExportDialogFromDocument(document)" />
                      </div>
                    </td>
                  </tr>
                </ng-template>
                <ng-template pTemplate="emptymessage">
                  <tr><td colspan="10" class="p-8 text-center text-muted-color">Nu s-au găsit documente</td></tr>
                </ng-template>
              </p-table>

              <p-dialog
                [visible]="createDialogOpen()"
                (visibleChange)="createDialogOpen.set($event)"
                [modal]="true"
                [draggable]="false"
                [resizable]="true"
                [maximizable]="true"
                header="Adaugă document nou"
                styleClass="registry-dialog"
                [style]="{ width: 'min(72rem, 94vw)' }"
                [contentStyle]="{ 'max-height': '78dvh', overflow: 'auto' }"
              >
                <div class="grid gap-4 lg:grid-cols-2">
                  <section class="dialog-section">
                    <h3>Date document</h3>
                    <label class="search-field">
                      <span>Conținut <strong class="text-primary">*</strong></span>
                      <textarea pTextarea [(ngModel)]="newDocument.subject" rows="5" placeholder="Descrierea documentului"></textarea>
                    </label>
                    <div class="grid gap-3 md:grid-cols-2">
                      <label class="search-field">
                        <span>Tip document</span>
                        <p-select appendTo="body" [options]="documentTypeOptions()" [(ngModel)]="newDocument.document_type" optionLabel="label" optionValue="value" />
                      </label>
                      <label class="search-field">
                        <span>Direcție</span>
                        <p-select appendTo="body" [options]="directionOptions()" [(ngModel)]="newDocument.direction" optionLabel="label" optionValue="value" />
                      </label>
                      <label class="search-field">
                        <span>Status</span>
                        <p-select appendTo="body" [options]="statusOptions()" [(ngModel)]="newDocument.status" optionLabel="label" optionValue="value" />
                      </label>
                      <label class="search-field">
                        <span>Confidențialitate</span>
                        <p-select appendTo="body" [options]="confidentialityOptions()" [(ngModel)]="newDocument.confidentiality" optionLabel="label" optionValue="value" />
                      </label>
                    </div>
                    <label class="search-field">
                      <span>Registru</span>
                      <p-select appendTo="body" [options]="registryOptions()" [(ngModel)]="newDocument.registru_id" optionLabel="nume" optionValue="id" placeholder="Alege registrul" />
                    </label>
                  </section>

                  <section class="dialog-section">
                    <h3>Părți și detalii</h3>
                    <label class="search-field">
                      <span>Emitent <strong class="text-primary">*</strong></span>
                      <div class="flex gap-2">
                        <p-autoComplete
                        appendTo="body"
                        class="flex-1"
                        [suggestions]="partySuggestions()"
                        optionLabel="display_name"
                        [(ngModel)]="newCorrespondentParty"
                          [forceSelection]="true"
                          [dropdown]="false"
                          [showClear]="true"
                          [minLength]="1"
                          [delay]="150"
                          placeholder="Alege emitent"
                          (completeMethod)="filterParties($event)"
                          (onSelect)="onPartySelected('new', 'correspondent', $event)"
                          (onClear)="clearPartySelection('new', 'correspondent')"
                        >
                          <ng-template let-party pTemplate="item">
                            <div class="flex items-center gap-2">
                              <i [class]="partyTypeIcon(party.party_type)"></i>
                              <span>{{ formatPartyDisplay(party) }}</span>
                              <span class="ml-auto text-xs text-muted-color">{{ partyTypeLabel(party.party_type) }}</span>
                            </div>
                          </ng-template>
                        </p-autoComplete>
                        <p-button icon="pi pi-plus" severity="secondary" [rounded]="true" [text]="true" pTooltip="Parte nouă" (onClick)="openPartyDialog('physical', 'correspondent')" />
                      </div>
                    </label>
                    <label class="search-field">
                      <span>Destinatar</span>
                      <div class="flex gap-2">
                        <p-autoComplete
                        appendTo="body"
                        class="flex-1"
                        [suggestions]="partySuggestions()"
                        optionLabel="display_name"
                        [(ngModel)]="newAssignedParty"
                          [forceSelection]="true"
                          [dropdown]="false"
                          [showClear]="true"
                          [minLength]="1"
                          [delay]="150"
                          placeholder="Alege destinatar"
                          (completeMethod)="filterParties($event)"
                          (onSelect)="onPartySelected('new', 'assigned', $event)"
                          (onClear)="clearPartySelection('new', 'assigned')"
                        >
                          <ng-template let-party pTemplate="item">
                            <div class="flex items-center gap-2">
                              <i [class]="partyTypeIcon(party.party_type)"></i>
                              <span>{{ formatPartyDisplay(party) }}</span>
                              <span class="ml-auto text-xs text-muted-color">{{ partyTypeLabel(party.party_type) }}</span>
                            </div>
                          </ng-template>
                        </p-autoComplete>
                        <p-button icon="pi pi-plus" severity="secondary" [rounded]="true" [text]="true" pTooltip="Parte nouă" (onClick)="openPartyDialog('physical', 'assigned')" />
                      </div>
                    </label>
                    <label class="search-field">
                      <span>Observații</span>
                      <textarea pTextarea [(ngModel)]="newDocument.summary" rows="5" placeholder="Context intern, activitate, termen"></textarea>
                    </label>
                    <label class="search-field">
                      <span>Data ieșire</span>
                      <p-datepicker appendTo="body" [(ngModel)]="newDocument.due_date" dateFormat="yy-mm-dd" [showIcon]="true" />
                    </label>
                  </section>
                </div>

                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="createDialogOpen.set(false)" />
                    <p-button label="Salvează document" icon="pi pi-check" (onClick)="saveDocument()" />
                  </div>
                </ng-template>
              </p-dialog>

              <p-dialog
                [visible]="batchDialogOpen()"
                (visibleChange)="batchDialogOpen.set($event)"
                [modal]="true"
                [draggable]="false"
                header="Generare documente multiple"
                styleClass="registry-dialog"
                [style]="{ width: 'min(58rem, 94vw)' }"
              >
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="search-field">
                    <span>Număr documente</span>
                    <input pInputText type="number" min="1" max="100" [(ngModel)]="batchDocument.count" />
                  </label>
                  <label class="search-field">
                    <span>Registru</span>
                    <p-select appendTo="body" [options]="registryOptions()" [(ngModel)]="batchDocument.registru_id" optionLabel="nume" optionValue="id" />
                  </label>
                  <label class="search-field">
                    <span>Direcție</span>
                    <p-select appendTo="body" [options]="directionOptions()" [(ngModel)]="batchDocument.direction" optionLabel="label" optionValue="value" />
                  </label>
                  <label class="search-field">
                    <span>Tip document</span>
                    <p-select appendTo="body" [options]="documentTypeOptions()" [(ngModel)]="batchDocument.document_type" optionLabel="label" optionValue="value" />
                  </label>
                  <label class="search-field">
                    <span>Emitent</span>
                    <div class="flex gap-2">
                      <p-autoComplete
                        appendTo="body"
                        class="flex-1"
                        [suggestions]="partySuggestions()"
                        optionLabel="display_name"
                        [(ngModel)]="batchCorrespondentParty"
                        [forceSelection]="true"
                        [dropdown]="false"
                        [showClear]="true"
                        [minLength]="1"
                        [delay]="150"
                        placeholder="Alege emitent"
                        (completeMethod)="filterParties($event)"
                        (onSelect)="onPartySelected('batch', 'correspondent', $event)"
                        (onClear)="clearPartySelection('batch', 'correspondent')"
                      >
                        <ng-template let-party pTemplate="item">
                          <div class="flex items-center gap-2">
                            <i [class]="partyTypeIcon(party.party_type)"></i>
                            <span>{{ formatPartyDisplay(party) }}</span>
                            <span class="ml-auto text-xs text-muted-color">{{ partyTypeLabel(party.party_type) }}</span>
                          </div>
                        </ng-template>
                      </p-autoComplete>
                      <p-button icon="pi pi-plus" severity="secondary" [rounded]="true" [text]="true" pTooltip="Parte nouă" (onClick)="openPartyDialog('physical', 'correspondent')" />
                    </div>
                  </label>
                  <label class="search-field">
                    <span>Destinatar</span>
                    <div class="flex gap-2">
                      <p-autoComplete
                        appendTo="body"
                        class="flex-1"
                        [suggestions]="partySuggestions()"
                        optionLabel="display_name"
                        [(ngModel)]="batchAssignedParty"
                        [forceSelection]="true"
                        [dropdown]="false"
                        [showClear]="true"
                        [minLength]="1"
                        [delay]="150"
                        placeholder="Alege destinatar"
                        (completeMethod)="filterParties($event)"
                        (onSelect)="onPartySelected('batch', 'assigned', $event)"
                        (onClear)="clearPartySelection('batch', 'assigned')"
                      >
                        <ng-template let-party pTemplate="item">
                          <div class="flex items-center gap-2">
                            <i [class]="partyTypeIcon(party.party_type)"></i>
                            <span>{{ formatPartyDisplay(party) }}</span>
                            <span class="ml-auto text-xs text-muted-color">{{ partyTypeLabel(party.party_type) }}</span>
                          </div>
                        </ng-template>
                      </p-autoComplete>
                      <p-button icon="pi pi-plus" severity="secondary" [rounded]="true" [text]="true" pTooltip="Parte nouă" (onClick)="openPartyDialog('physical', 'assigned')" />
                    </div>
                  </label>
                  <label class="search-field md:col-span-2">
                    <span>Conținut comun</span>
                    <textarea pTextarea rows="4" [(ngModel)]="batchDocument.summary"></textarea>
                  </label>
                  <div class="md:col-span-2 rounded-2xl border border-primary-200 bg-primary-50 p-4 text-sm text-primary-900 dark:border-primary-700 dark:bg-primary-950 dark:text-primary-100">
                    Batch-ul consumă numerotarea server-side din registrul ales și creează fiecare document cu un număr unic.
                  </div>
                </div>

                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="batchDialogOpen.set(false)" />
                    <p-button label="Generează" icon="pi pi-clone" (onClick)="saveBatchDocuments()" />
                  </div>
                </ng-template>
              </p-dialog>

              <p-dialog
                [visible]="exportDialogOpen()"
                (visibleChange)="exportDialogOpen.set($event)"
                [modal]="true"
                [draggable]="false"
                header="Export PDF registratură"
                styleClass="registry-dialog"
                [style]="{ width: 'min(46rem, 94vw)' }"
              >
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="search-field">
                    <span>Registru</span>
                    <p-select appendTo="body" [options]="registryOptions()" [(ngModel)]="exportRequest.registru_id" optionLabel="nume" optionValue="id" />
                  </label>
                  <label class="search-field">
                    <span>Data de la</span>
                    <p-datepicker appendTo="body" [(ngModel)]="exportRequest.start_date" dateFormat="yy-mm-dd" [showIcon]="true" />
                  </label>
                  <label class="search-field">
                    <span>Data până la</span>
                    <p-datepicker appendTo="body" [(ngModel)]="exportRequest.end_date" dateFormat="yy-mm-dd" [showIcon]="true" />
                  </label>
                </div>

                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="exportDialogOpen.set(false)" />
                    <p-button label="Exportă PDF" icon="pi pi-file-pdf" (onClick)="downloadPdf()" />
                  </div>
                </ng-template>
              </p-dialog>

              <p-dialog
                [visible]="partyDialogOpen()"
                (visibleChange)="partyDialogOpen.set($event)"
                [modal]="true"
                [draggable]="false"
                header="Parte nouă"
                styleClass="registry-dialog"
                [style]="{ width: 'min(54rem, 94vw)' }"
              >
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="search-field">
                    <span>Tip parte</span>
                    <p-select appendTo="body" [options]="partyTypeDialogOptions" [(ngModel)]="partyForm.party_type" optionLabel="label" optionValue="value" />
                  </label>
                  <label class="search-field">
                    <span>Denumire afișată</span>
                    <input pInputText [(ngModel)]="partyForm.display_name" />
                  </label>
                  @if (partyForm.party_type === 'physical') {
                    <label class="search-field">
                      <span>Nume scurt</span>
                      <input pInputText [(ngModel)]="partyForm.short_name" />
                    </label>
                    <label class="search-field">
                      <span>Prenume</span>
                      <input pInputText [(ngModel)]="partyForm.first_name" />
                    </label>
                    <label class="search-field">
                      <span>Nume</span>
                      <input pInputText [(ngModel)]="partyForm.last_name" />
                    </label>
                  }
                  @if (partyForm.party_type === 'legal' || partyForm.party_type === 'institution') {
                    <label class="search-field">
                      <span>Cod intern</span>
                      <input pInputText [(ngModel)]="partyForm.code" />
                    </label>
                    <label class="search-field">
                      <span>Nume scurt</span>
                      <input pInputText [(ngModel)]="partyForm.short_name" />
                    </label>
                    <label class="search-field">
                      <span>Denumire legală</span>
                      <input pInputText [(ngModel)]="partyForm.legal_name" />
                    </label>
                    <label class="search-field">
                      <span>ID / CNP / CUI</span>
                      <input pInputText [(ngModel)]="partyForm.identifier_code" />
                    </label>
                  }
                  <label class="search-field">
                    <span>Telefon</span>
                    <input pInputText [(ngModel)]="partyForm.phone_number" />
                  </label>
                  <label class="search-field">
                    <span>Email</span>
                    <input pInputText [(ngModel)]="partyForm.email" />
                  </label>
                  <label class="search-field md:col-span-2">
                    <span>Adresă</span>
                    <input pInputText [(ngModel)]="partyForm.address_line1" />
                  </label>
                  @if (partyForm.party_type === 'physical') {
                    <label class="search-field md:col-span-2">
                      <span>Observații</span>
                      <textarea pTextarea rows="4" [(ngModel)]="partyForm.notes"></textarea>
                    </label>
                  } @else {
                    <label class="search-field md:col-span-2">
                      <span>Observații</span>
                      <textarea pTextarea rows="4" [(ngModel)]="partyForm.notes"></textarea>
                    </label>
                  }
                </div>

                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="partyDialogOpen.set(false)" />
                    <p-button label="Salvează parte" icon="pi pi-check" (onClick)="saveParty()" />
                  </div>
                </ng-template>
              </p-dialog>
            </div>
          </p-tabpanel>

          <p-tabpanel value="flux" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-4">
              <div class="grid gap-3 md:grid-cols-3">
                <p-card>
                  <div class="text-sm text-muted-color">În workflow</div>
                  <div class="text-2xl font-semibold">{{ workflowDocuments().length }}</div>
                </p-card>
                <p-card>
                  <div class="text-sm text-muted-color">Arhivate</div>
                  <div class="text-2xl font-semibold">{{ archiveDocuments().length }}</div>
                </p-card>
                <p-card>
                  <div class="text-sm text-muted-color">Registru activ</div>
                  <div class="text-2xl font-semibold">{{ selectedRegistryLabel() }}</div>
                </p-card>
              </div>

              <p-table [value]="workflowDocuments()" styleClass="p-datatable-sm p-datatable-gridlines" [paginator]="true" [rows]="10" [scrollable]="true" scrollHeight="flex">
                <ng-template pTemplate="header">
                  <tr>
                    <th style="width:9rem">Nr.</th>
                    <th>Conținut</th>
                    <th style="width:13rem">Emitent</th>
                    <th style="width:13rem">Destinatar</th>
                    <th style="width:10rem">Status</th>
                  </tr>
                </ng-template>
                <ng-template pTemplate="body" let-document>
                  <tr>
                    <td class="font-mono">{{ document.registry_number }}</td>
                    <td>{{ document.subject }}</td>
                    <td>{{ document.correspondent }}</td>
                    <td>{{ document.assigned_to || '-' }}</td>
                    <td><p-tag [value]="document.status" [severity]="statusSeverity(document.status)" /></td>
                  </tr>
                </ng-template>
                <ng-template pTemplate="emptymessage">
                  <tr><td colspan="5" class="p-8 text-center text-muted-color">Nu există documente în workflow pentru registrul selectat.</td></tr>
                </ng-template>
              </p-table>
            </div>
          </p-tabpanel>

          <p-tabpanel value="arhiva" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="grid min-h-0 flex-1 gap-4 p-4 lg:grid-cols-[minmax(0,1fr)_minmax(24rem,30rem)]">
              <p-card class="min-h-0 overflow-hidden">
                <div class="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <h3 class="m-0 text-lg font-semibold">Documente arhivate</h3>
                    <p class="m-0 text-sm text-muted-color">Selectează un document pentru a vedea fișierele atașate.</p>
                  </div>
                </div>
                <p-table [value]="archiveDocuments()" styleClass="p-datatable-sm p-datatable-gridlines" [paginator]="true" [rows]="10" [scrollable]="true" scrollHeight="flex">
                  <ng-template pTemplate="header">
                    <tr>
                      <th style="width:9rem">Nr.</th>
                      <th>Conținut</th>
                      <th style="width:10rem">Status</th>
                      <th style="width:7rem"></th>
                    </tr>
                  </ng-template>
                  <ng-template pTemplate="body" let-document>
                    <tr>
                      <td class="font-mono">{{ document.registry_number }}</td>
                      <td>{{ document.subject }}</td>
                      <td><p-tag [value]="document.status" severity="success" /></td>
                      <td><p-button icon="pi pi-folder-open" [text]="true" size="small" (onClick)="selectArchiveDocument(document)" /></td>
                    </tr>
                  </ng-template>
                  <ng-template pTemplate="emptymessage">
                    <tr><td colspan="4" class="p-8 text-center text-muted-color">Nu există documente arhivate pentru registrul selectat.</td></tr>
                  </ng-template>
                </p-table>
              </p-card>

              <p-card class="min-h-0 overflow-hidden">
                <div class="mb-3">
                  <h3 class="m-0 text-lg font-semibold">Fișiere atașate</h3>
                  <p class="m-0 text-sm text-muted-color">{{ selectedArchiveDocument() ? selectedArchiveDocument()!.registry_number : 'Alege un document arhivat' }}</p>
                </div>
                <p-table [value]="archiveAttachments()" styleClass="p-datatable-sm" [paginator]="false">
                  <ng-template pTemplate="header">
                    <tr>
                      <th>Fișier</th>
                      <th style="width:9rem">Categorie</th>
                      <th style="width:8rem">Status</th>
                    </tr>
                  </ng-template>
                  <ng-template pTemplate="body" let-file>
                    <tr>
                      <td>
                        <div class="font-medium">{{ file.title }}</div>
                        <div class="text-xs text-muted-color">{{ file.file_name }}</div>
                      </td>
                      <td>{{ file.category }}</td>
                      <td><p-tag [value]="file.status" severity="info" /></td>
                    </tr>
                  </ng-template>
                  <ng-template pTemplate="emptymessage">
                    <tr><td colspan="3" class="p-8 text-center text-muted-color">Niciun fișier încărcat.</td></tr>
                  </ng-template>
                </p-table>
              </p-card>
            </div>
          </p-tabpanel>
        </p-tabpanels>
      </p-tabs>

      <p-drawer [visible]="documentDrawerOpen()" (visibleChange)="documentDrawerOpen.set($event)" position="right" header="Detalii document" [style]="{ width: 'min(42rem, 100vw)' }">
        @if (selectedDocument()) {
          <div class="grid gap-4">
            <p-card>
              <div class="grid gap-2 text-sm">
                <div><span class="font-semibold">Nr. document:</span> {{ selectedDocument()!.registry_number }}</div>
                <div><span class="font-semibold">Conținut:</span> {{ selectedDocument()!.subject }}</div>
                <div><span class="font-semibold">Emitent:</span> {{ selectedDocument()!.correspondent }}</div>
                <div><span class="font-semibold">Destinatar:</span> {{ selectedDocument()!.assigned_to || '-' }}</div>
                <div><span class="font-semibold">Status:</span> <p-tag [value]="selectedDocument()!.status" [severity]="statusSeverity(selectedDocument()!.status)" /></div>
              </div>
              <div class="mt-4 flex flex-wrap gap-2">
                <p-button label="Editează" icon="pi pi-pencil" severity="secondary" size="small" (onClick)="openEditDialog(selectedDocument()!)" />
                <p-button label="Anulează" icon="pi pi-ban" severity="danger" size="small" (onClick)="openCancelDialog(selectedDocument()!)" />
              </div>
            </p-card>

            <p-card>
              <h3 class="m-0 mb-2 text-base font-semibold">Versiuni</h3>
              <div class="grid gap-2 text-sm">
                @for (version of selectedDocumentVersions(); track version.id) {
                  <div class="rounded-lg border border-surface-200 p-3">
                    <div class="font-semibold">Versiunea {{ version.version_no }}</div>
                    <div>{{ version.change_notes }}</div>
                    <div class="text-xs text-muted-color">{{ version.created_at }} • {{ version.created_by }}</div>
                  </div>
                } @empty {
                  <div class="text-sm text-muted-color">Nu există versiuni înregistrate.</div>
                }
              </div>
            </p-card>

            <p-card>
              <h3 class="m-0 mb-2 text-base font-semibold">Atașamente</h3>
              <div class="grid gap-2 text-sm">
                @for (attachment of selectedDocumentAttachments(); track attachment.id) {
                  <div class="rounded-lg border border-surface-200 p-3">
                    <div class="font-semibold">{{ attachment.title }}</div>
                    <div class="text-xs text-muted-color">{{ attachment.file_name }} • {{ attachment.mime_type }}</div>
                  </div>
                } @empty {
                  <div class="text-sm text-muted-color">Nu există atașamente.</div>
                }
              </div>
            </p-card>

            <p-card>
              <h3 class="m-0 mb-2 text-base font-semibold">Legături</h3>
              <div class="grid gap-2 text-sm">
                @for (link of selectedDocumentLinks(); track link.link_id) {
                  <div class="rounded-lg border border-surface-200 p-3">
                    <div class="font-semibold">{{ link.registry_number }}</div>
                    <div>{{ link.subject }}</div>
                    <div class="flex items-center justify-between gap-2">
                      <div class="text-xs text-muted-color">{{ link.document_type }} • {{ link.relation_type }} • {{ link.status }}</div>
                      <p-button icon="pi pi-trash" severity="danger" [text]="true" size="small" pTooltip="Șterge legătura" (onClick)="deleteDocumentLink(link)" />
                    </div>
                  </div>
                } @empty {
                  <div class="text-sm text-muted-color">Nu există legături înregistrate.</div>
                }
              </div>
              <div class="mt-3">
                <p-button label="Adaugă legătură" icon="pi pi-link" severity="secondary" size="small" (onClick)="openLinkDialog(selectedDocument()!)" />
              </div>
            </p-card>
          </div>
        }
      </p-drawer>

      <p-dialog
        [visible]="linkDialogOpen()"
        (visibleChange)="linkDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        header="Leagă document"
        styleClass="registry-dialog"
        [style]="{ width: 'min(56rem, 94vw)' }"
      >
        <div class="grid gap-3">
          <label class="search-field">
            <span>Caută document</span>
            <input pInputText [ngModel]="linkSearchQuery()" (ngModelChange)="linkSearchQuery.set($event)" />
          </label>
          <div class="flex justify-end gap-2">
            <p-button label="Caută" icon="pi pi-search" severity="secondary" [outlined]="true" (onClick)="searchLinkDocuments()" />
          </div>
          <div class="grid gap-2">
            @for (item of linkLookupResults(); track item.id) {
              <div class="rounded-lg border border-surface-200 p-3 flex items-center justify-between gap-3">
                <div>
                  <div class="font-semibold">{{ item.registry_number }}</div>
                  <div class="text-sm">{{ item.subject }}</div>
                  <div class="text-xs text-muted-color">{{ item.document_type }} • {{ item.status }}</div>
                </div>
                <p-button label="Leagă" icon="pi pi-link" size="small" (onClick)="createDocumentLink(item)" />
              </div>
            } @empty {
              <div class="text-sm text-muted-color">Nicio potrivire încă.</div>
            }
          </div>
        </div>
      </p-dialog>

      <p-dialog
        [visible]="editDialogOpen()"
        (visibleChange)="editDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [resizable]="true"
        [maximizable]="true"
        header="Editează document"
        styleClass="registry-dialog"
        [style]="{ width: 'min(72rem, 94vw)' }"
        [contentStyle]="{ 'max-height': '78dvh', overflow: 'auto' }"
      >
        <div class="grid gap-4 lg:grid-cols-2">
          <section class="dialog-section">
            <h3>Date document</h3>
            <label class="search-field">
              <span>Conținut</span>
              <textarea pTextarea [(ngModel)]="editDocument.subject" rows="5"></textarea>
            </label>
            <div class="grid gap-3 md:grid-cols-2">
              <label class="search-field">
                <span>Tip document</span>
                <p-select appendTo="body" [options]="documentTypeOptions()" [(ngModel)]="editDocument.document_type" />
              </label>
              <label class="search-field">
                <span>Direcție</span>
                <p-select appendTo="body" [options]="directionOptions()" [(ngModel)]="editDocument.direction" />
              </label>
              <label class="search-field">
                <span>Status</span>
                <p-select appendTo="body" [options]="statusOptions()" [(ngModel)]="editDocument.status" />
              </label>
              <label class="search-field">
                <span>Confidențialitate</span>
                <p-select appendTo="body" [options]="confidentialityOptions()" [(ngModel)]="editDocument.confidentiality" />
              </label>
            </div>
            <label class="search-field">
              <span>Registru</span>
              <p-select appendTo="body" [options]="registryOptions()" [(ngModel)]="editDocument.registru_id" optionLabel="nume" optionValue="id" />
            </label>
          </section>
                  <section class="dialog-section">
                    <h3>Părți și notițe</h3>
                    <label class="search-field">
                      <span>Emitent</span>
                      <p-autoComplete
                        appendTo="body"
                        [suggestions]="partySuggestions()"
                        optionLabel="display_name"
                        [(ngModel)]="editCorrespondentParty"
                        [forceSelection]="true"
                        [dropdown]="false"
                        [showClear]="true"
                        [minLength]="1"
                        [delay]="150"
                        placeholder="Alege emitent"
                        (completeMethod)="filterParties($event)"
                        (onSelect)="onPartySelected('edit', 'correspondent', $event)"
                        (onClear)="clearPartySelection('edit', 'correspondent')"
                      >
                        <ng-template let-party pTemplate="item">
                          <div class="flex items-center gap-2">
                            <i [class]="partyTypeIcon(party.party_type)"></i>
                            <span>{{ formatPartyDisplay(party) }}</span>
                            <span class="ml-auto text-xs text-muted-color">{{ partyTypeLabel(party.party_type) }}</span>
                          </div>
                        </ng-template>
                      </p-autoComplete>
                    </label>
                    <label class="search-field">
                      <span>Destinatar</span>
                      <p-autoComplete
                        appendTo="body"
                        [suggestions]="partySuggestions()"
                        optionLabel="display_name"
                        [(ngModel)]="editAssignedParty"
                        [forceSelection]="true"
                        [dropdown]="false"
                        [showClear]="true"
                        [minLength]="1"
                        [delay]="150"
                        placeholder="Alege destinatar"
                        (completeMethod)="filterParties($event)"
                        (onSelect)="onPartySelected('edit', 'assigned', $event)"
                        (onClear)="clearPartySelection('edit', 'assigned')"
                      >
                        <ng-template let-party pTemplate="item">
                          <div class="flex items-center gap-2">
                            <i [class]="partyTypeIcon(party.party_type)"></i>
                            <span>{{ formatPartyDisplay(party) }}</span>
                            <span class="ml-auto text-xs text-muted-color">{{ partyTypeLabel(party.party_type) }}</span>
                          </div>
                        </ng-template>
                      </p-autoComplete>
                    </label>
            <label class="search-field">
              <span>Motiv schimbare</span>
              <textarea pTextarea [(ngModel)]="editDocument.change_notes" rows="5"></textarea>
            </label>
            <label class="search-field">
              <span>Observații</span>
              <textarea pTextarea [(ngModel)]="editDocument.summary" rows="4"></textarea>
            </label>
          </section>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="editDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" (onClick)="saveEditedDocument()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="cancelDialogOpen()"
        (visibleChange)="cancelDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        header="Anulează document"
        styleClass="registry-dialog"
        [style]="{ width: 'min(40rem, 94vw)' }"
      >
        <div class="grid gap-3">
          <label class="search-field">
            <span>Motiv anulare</span>
            <textarea pTextarea [(ngModel)]="cancelReason" rows="4"></textarea>
          </label>
          <p class="m-0 text-sm text-muted-color">Documentul va fi marcat arhivat și va primi o versiune nouă în istoric.</p>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="cancelDialogOpen.set(false)" />
            <p-button label="Anulează document" severity="danger" (onClick)="cancelSelectedDocument()" />
          </div>
        </ng-template>
      </p-dialog>
    </section>
  `,
  styles: `
    :host {
      display: block;
      min-height: 0;
    }

    :host ::ng-deep .document-workspace .p-tabs,
    :host ::ng-deep .document-workspace .p-tabpanels,
    :host ::ng-deep .document-workspace .p-tabpanel {
      display: flex;
      flex-direction: column;
      flex: 1;
      min-height: 0;
      overflow: hidden;
      background: transparent;
    }

    :host ::ng-deep .p-datatable,
    :host ::ng-deep .p-datatable-table-container {
      min-height: 0;
    }

    :host ::ng-deep .workspace-table .p-datatable-wrapper {
      display: flex;
      min-height: 0;
      flex: 1;
      flex-direction: column;
    }

    :host ::ng-deep .workspace-table .p-datatable-table-container {
      flex: 1;
      min-height: 0;
      overflow: auto;
    }

    :host ::ng-deep .workspace-table .p-datatable-paginator {
      position: sticky;
      bottom: 0;
      z-index: 4;
      border-top: 1px solid var(--p-content-border-color);
      background: var(--p-content-background);
    }

    :host ::ng-deep .p-datatable-thead {
      position: sticky;
      top: 0;
      z-index: 3;
    }

    :host ::ng-deep .workspace-panel.p-fieldset {
      margin: 0;
      border-radius: 1.25rem;
      border-color: var(--p-content-border-color);
      background: var(--p-content-background);
      overflow: hidden;
    }

    :host ::ng-deep .workspace-panel .p-fieldset-legend {
      margin-left: 1rem;
      padding-inline: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.02em;
    }

    :host-context(.app-dark) ::ng-deep .workspace-panel.p-fieldset {
      background: var(--p-surface-900);
    }

    :host ::ng-deep .registry-table .p-datatable-thead > tr > th {
      background: var(--p-surface-50);
    }

    :host-context(.app-dark) ::ng-deep .registry-table .p-datatable-thead > tr > th {
      background: var(--p-surface-800);
    }

    .search-panel {
      border: 1px solid var(--p-content-border-color);
      border-radius: var(--p-content-border-radius);
      background: var(--p-content-background);
      overflow: hidden;
    }

    .search-panel-header,
    .search-panel-footer {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: var(--p-surface-50);
    }

    :host-context(.app-dark) .search-panel {
      background: var(--p-surface-900);
    }

    :host-context(.app-dark) .search-panel-header,
    :host-context(.app-dark) .search-panel-footer {
      background: var(--p-surface-800);
    }

    .search-panel-header {
      border-bottom: 1px solid var(--p-content-border-color);
    }

    .search-panel-footer {
      border-top: 1px solid var(--p-content-border-color);
      justify-content: flex-end;
    }

    .workspace-actions {
      align-self: stretch;
      padding: 0.25rem 0;
    }

    .search-panel-content {
      display: flex;
      flex-direction: column;
      gap: 1rem;
      padding: 1rem;
    }

    .search-row {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
      gap: 1rem;
    }

    .search-field {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
      font-size: 0.8125rem;
      color: var(--p-text-muted-color);
    }

    .search-field > span {
      display: flex;
      align-items: center;
      gap: 0.45rem;
      font-weight: 600;
    }

    .date-range {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .dialog-section {
      display: flex;
      flex-direction: column;
      gap: 1rem;
      border: 1px solid var(--p-content-border-color);
      border-radius: var(--p-content-border-radius);
      background: var(--p-content-background);
      padding: 1rem;
    }

    .dialog-section h3 {
      margin: 0;
      font-size: 1rem;
      font-weight: 700;
    }

    .workspace-table {
      border-radius: 1.25rem;
    }

    .registry-header {
      grid-template-columns: auto minmax(0, 1fr) auto;
    }

    .registry-selector {
      width: 12rem;
    }

    :host ::ng-deep .workspace-table .p-datatable,
    :host ::ng-deep .workspace-table .p-datatable-wrapper,
    :host ::ng-deep .workspace-table .p-datatable-table-container {
      display: flex;
      min-height: 0;
      flex: 1 1 auto;
      flex-direction: column;
    }

    :host ::ng-deep .workspace-table .p-datatable-table {
      width: 100%;
      min-width: 100%;
    }

    :host ::ng-deep .workspace-table .p-datatable-paginator {
      padding-block: 0.125rem;
      min-height: 2.4rem;
      row-gap: 0.125rem;
    }

    :host-context(.app-dark) .dialog-section {
      background: var(--p-surface-900);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DocumenteWorkspaceComponent {
  private readonly api = inject(RegistraturaApiService);
  private readonly linksApi = inject(DocumentLinksApiService);
  private readonly authz = inject(AuthzService);
  private readonly messages = inject(MessageService);
  private readonly router = inject(Router);
  private readonly appApi = inject(AppApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly registryStorageKey = 'egueducation.registratura.selectedRegistryId';

  protected readonly loading = signal(false);
  protected readonly searchPanelOpen = signal(false);
  protected readonly createDialogOpen = signal(false);
  protected readonly batchDialogOpen = signal(false);
  protected readonly exportDialogOpen = signal(false);
  protected readonly documentDrawerOpen = signal(false);
  protected readonly editDialogOpen = signal(false);
  protected readonly cancelDialogOpen = signal(false);
  protected readonly registries = signal<RegistraturaRegistry[]>([]);
  protected readonly parties = signal<RegistraturaParty[]>([]);
  protected readonly filtersResponse = signal<RegistraturaDocumentFilters | null>(null);
  protected readonly documents = signal<RegistraturaDocument[]>([]);
  protected readonly workflowDocuments = signal<RegistraturaDocument[]>([]);
  protected readonly archiveDocuments = signal<RegistraturaDocument[]>([]);
  protected readonly archiveAttachments = signal<RegistraturaDocumentAttachment[]>([]);
  protected readonly selectedDocument = signal<RegistraturaDocument | null>(null);
  protected readonly selectedDocumentVersions = signal<RegistraturaDocumentVersion[]>([]);
  protected readonly selectedDocumentAttachments = signal<RegistraturaDocumentAttachment[]>([]);
  protected readonly selectedDocumentLinks = signal<LinkedDocument[]>([]);
  protected readonly linkDialogOpen = signal(false);
  protected readonly partyDialogOpen = signal(false);
  protected readonly partyDialogTarget = signal<'correspondent' | 'assigned' | null>(null);
  protected readonly filteredNrDocSuggestions = signal<string[]>([]);
  protected readonly filteredEmitentiSearch = signal<RegistraturaParty[]>([]);
  protected readonly filteredDestinatariSearch = signal<RegistraturaParty[]>([]);
  protected readonly linkSearchQuery = signal('');
  protected readonly linkLookupResults = signal<DocumentLookupItem[]>([]);
  protected readonly selectedArchiveDocument = signal<RegistraturaDocument | null>(null);
  protected readonly totalRecords = signal(0);
  protected readonly page = signal(1);
  protected readonly pageSize = signal(20);
  protected readonly selectedRegistryId = signal<number | null>(this.loadSavedRegistryId());
  protected selectedRegistryModel: number | null = this.loadSavedRegistryId();
  protected readonly filters: FilterState = {
    registry_number: '',
    subject: '',
    document_type: '',
    direction: '',
    status: '',
    correspondent: '',
    assigned_to: '',
    confidentiality: '',
    registered_at_from: '',
    registered_at_to: '',
    due_date_from: '',
    due_date_to: '',
  };

  protected newDocument = emptyNewDocument();
  protected batchDocument = emptyBatchDocument();
  protected exportRequest = emptyExportRequest();
  protected editDocument = emptyEditDocument();
  protected partyForm = emptyPartyForm();
  protected cancelReason = '';

  protected readonly registryOptions = computed(() => this.registries());
  protected readonly selectedRegistryLabel = computed(() => {
    const id = this.selectedRegistryId();
    return this.registries().find((registry) => registry.id === id)?.nume ?? 'Niciun registru';
  });
  protected readonly documentTypeOptions = computed(() => this.toOptions(this.filtersResponse()?.document_types ?? []));
  protected readonly directionOptions = computed(() => this.toOptions(this.filtersResponse()?.directions ?? []));
  protected readonly statusOptions = computed(() => this.toOptions(this.filtersResponse()?.statuses ?? []));
  protected readonly confidentialityOptions = computed(() => this.toOptions(this.filtersResponse()?.confidentialities ?? []));
  protected readonly partyTypeDialogOptions = [
    { label: 'Persoană fizică', value: 'physical' },
    { label: 'Persoană juridică', value: 'legal' },
    { label: 'Instituție', value: 'institution' },
  ];
  protected readonly partySuggestions = signal<RegistraturaParty[]>([]);

  protected newCorrespondentParty: RegistraturaParty | null = null;
  protected newAssignedParty: RegistraturaParty | null = null;
  protected batchCorrespondentParty: RegistraturaParty | null = null;
  protected batchAssignedParty: RegistraturaParty | null = null;
  protected editCorrespondentParty: RegistraturaParty | null = null;
  protected editAssignedParty: RegistraturaParty | null = null;
  protected selectedEmitentSearch: RegistraturaParty | null = null;
  protected selectedDestinatarSearch: RegistraturaParty | null = null;

  constructor() {
    effect(() => {
      const id = this.selectedRegistryId();
      this.selectedRegistryModel = id;
      if (id) {
        localStorage.setItem(this.registryStorageKey, String(id));
      } else {
        localStorage.removeItem(this.registryStorageKey);
      }
    });
  }

  ngOnInit(): void {
    this.loadBootstrapData();
  }

  protected onRegistryChange(registryId: number | null): void {
    this.selectedRegistryModel = registryId ?? null;
    this.selectedRegistryId.set(registryId ?? null);
    this.newDocument.registru_id = registryId ?? null;
    this.batchDocument.registru_id = registryId ?? 0;
    this.exportRequest.registru_id = registryId ?? null;
    this.selectedArchiveDocument.set(null);
    this.archiveAttachments.set([]);
    this.loadDocuments();
    this.loadWorkflowDocuments();
    this.loadArchiveDocuments();
  }

  protected openCreateDialog(mode: 'intrare' | 'iesire'): void {
    this.newDocument = emptyNewDocument();
    this.newDocument.direction = mode;
    this.newDocument.document_type = this.documentTypeOptions()[0]?.value ?? '';
    this.newDocument.status = this.statusOptions()[0]?.value ?? 'draft';
    this.newDocument.confidentiality = this.confidentialityOptions()[0]?.value ?? 'normal';
    this.newDocument.registru_id = this.selectedRegistryId();
    this.applyDefaultParties(this.newDocument, mode);
    this.syncPartySelectionsFromDocument('new', this.newDocument);
    this.createDialogOpen.set(true);
  }

  protected openBatchDialog(): void {
    this.batchDocument = emptyBatchDocument();
    this.batchDocument.registru_id = this.selectedRegistryId() ?? 0;
    this.batchDocument.document_type = this.documentTypeOptions()[0]?.value ?? '';
    this.batchDocument.status = this.statusOptions()[0]?.value ?? 'draft';
    this.batchDocument.confidentiality = this.confidentialityOptions()[0]?.value ?? 'normal';
    this.applyDefaultParties(this.batchDocument, this.batchDocument.direction);
    this.syncPartySelectionsFromDocument('batch', this.batchDocument);
    this.batchDialogOpen.set(true);
  }

  protected openExportDialog(): void {
    this.exportRequest = emptyExportRequest();
    this.exportRequest.registru_id = this.selectedRegistryId();
    this.exportDialogOpen.set(true);
  }

  protected openExportDialogFromDocument(document: RegistraturaDocument): void {
    this.exportRequest = {
      registru_id: document.registru_id ?? this.selectedRegistryId(),
      start_date: document.registered_at?.slice(0, 10) ?? null,
      end_date: document.registered_at?.slice(0, 10) ?? null,
    };
    this.exportDialogOpen.set(true);
  }

  protected resetFilters(): void {
    this.filters.registry_number = '';
    this.filters.subject = '';
    this.filters.document_type = '';
    this.filters.direction = '';
    this.filters.status = '';
    this.filters.correspondent = '';
    this.filters.assigned_to = '';
    this.filters.confidentiality = '';
    this.filters.registered_at_from = '';
    this.filters.registered_at_to = '';
    this.filters.due_date_from = '';
    this.filters.due_date_to = '';
    this.selectedEmitentSearch = null;
    this.selectedDestinatarSearch = null;
    this.filteredEmitentiSearch.set([]);
    this.filteredDestinatariSearch.set([]);
    this.filteredNrDocSuggestions.set([]);
    this.loadDocuments();
  }

  protected loadDocuments(event?: TableLazyLoadEvent, silent = false): void {
    if (!silent) {
      this.loading.set(true);
    }

    const pageSize = event?.rows ?? this.pageSize();
    const first = event?.first ?? 0;
    const page = Math.floor(first / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? String(event?.sortField[0] ?? 'registered_at') : String(event?.sortField ?? 'registered_at');
    const direction = event?.sortOrder === 1 ? 'asc' : 'desc';

    this.page.set(page);
    this.pageSize.set(pageSize);

    const filters: Record<string, string> = {};
    const registruId = this.selectedRegistryId();
    if (registruId) {
      filters['registru_id'] = String(registruId);
    }
    for (const [key, value] of Object.entries(this.filters)) {
      if (value) {
        filters[key] = this.normalizeDateOrText(value);
      }
    }

    const query: TableQuery = {
      page,
      pageSize,
      sort,
      direction,
      filters,
    };

    this.api.documents(query).subscribe({
      next: (response) => {
        this.documents.set(response.items);
        this.totalRecords.set(response.total);
        this.loading.set(false);
      },
      error: () => {
        this.documents.set([]);
        this.totalRecords.set(0);
        this.loading.set(false);
      },
    });
  }

  protected loadWorkflowDocuments(): void {
    const filters: Record<string, string> = { status: 'in_workflow' };
    const registruId = this.selectedRegistryId();
    if (registruId) {
      filters['registru_id'] = String(registruId);
    }

    this.api.documents({ page: 1, pageSize: 50, sort: 'registered_at', direction: 'desc', filters }).subscribe({
      next: (response) => this.workflowDocuments.set(response.items),
      error: () => this.workflowDocuments.set([]),
    });
  }

  protected loadArchiveDocuments(): void {
    const filters: Record<string, string> = { status: 'archived' };
    const registruId = this.selectedRegistryId();
    if (registruId) {
      filters['registru_id'] = String(registruId);
    }

    this.api.documents({ page: 1, pageSize: 50, sort: 'registered_at', direction: 'desc', filters }).subscribe({
      next: (response) => {
        this.archiveDocuments.set(response.items);
        if (!this.selectedArchiveDocument() && response.items.length > 0) {
          this.selectArchiveDocument(response.items[0]);
        }
      },
      error: () => this.archiveDocuments.set([]),
    });
  }

  protected selectArchiveDocument(document: RegistraturaDocument): void {
    this.selectedArchiveDocument.set(document);
    this.api.documentAttachments(document.id).subscribe({
      next: (attachments) => this.archiveAttachments.set(attachments),
      error: () => this.archiveAttachments.set([]),
    });
  }

  protected openDocumentDrawer(document: RegistraturaDocument): void {
    this.selectedDocument.set(document);
    this.documentDrawerOpen.set(true);

    forkJoin({
      detail: this.api.document(document.id),
      versions: this.api.documentVersions(document.id),
      attachments: this.api.documentAttachments(document.id),
      links: this.linksApi.listLinks('registratura', document.id).pipe(catchError(() => of([]))),
    }).subscribe({
      next: ({ detail, versions, attachments, links }) => {
        this.selectedDocument.set(detail);
        this.selectedDocumentVersions.set(versions);
        this.selectedDocumentAttachments.set(attachments);
        this.selectedDocumentLinks.set(links);
      },
      error: () => {
        this.selectedDocumentVersions.set([]);
        this.selectedDocumentAttachments.set([]);
        this.selectedDocumentLinks.set([]);
      },
    });
  }

  protected openWorkflowForDocument(document: RegistraturaDocument): void {
    this.openDocumentDrawer(document);
  }

  protected openDocumentPage(document: RegistraturaDocument): void {
    void this.router.navigate(['/documente', document.id]);
  }

  protected openDocumentEditPage(document: RegistraturaDocument): void {
    void this.router.navigate(['/documente', document.id, 'edit']);
  }

  protected openDocumentCreatePage(): void {
    void this.router.navigate(['/documente/new']);
  }

  protected openLinkDialog(document: RegistraturaDocument): void {
    this.selectedDocument.set(document);
    this.linkSearchQuery.set('');
    this.linkLookupResults.set([]);
    this.linkDialogOpen.set(true);
  }

  protected searchLinkDocuments(): void {
    const query = this.linkSearchQuery().trim();
    const currentDocument = this.selectedDocument();
    if (!query || !currentDocument) {
      this.linkLookupResults.set([]);
      return;
    }

    this.linksApi.lookupDocuments(query).subscribe({
      next: (items) => this.linkLookupResults.set(items.filter((item) => item.id !== currentDocument.id)),
      error: () => this.linkLookupResults.set([]),
    });
  }

  protected createDocumentLink(target: DocumentLookupItem): void {
    const sourceDocument = this.selectedDocument();
    if (!sourceDocument) {
      return;
    }

    const payload: CreateDocumentLinkRequest = {
      document_id: target.id,
      source_module: 'registratura',
      source_record_id: sourceDocument.id,
      relation_type: 'supporting',
    };

    this.linksApi.createLink(payload).subscribe({
      next: () => {
        this.linkDialogOpen.set(false);
        this.refreshDocumentDrawer(sourceDocument.id);
      },
      error: () => {
        this.linkDialogOpen.set(false);
      },
    });
  }

  protected deleteDocumentLink(link: LinkedDocument): void {
    const sourceDocument = this.selectedDocument();
    if (!sourceDocument) {
      return;
    }

    this.linksApi.deleteLink(link.link_id).subscribe({
      next: () => this.refreshDocumentDrawer(sourceDocument.id),
      error: () => undefined,
    });
  }

  protected openEditDialog(document: RegistraturaDocument): void {
    this.selectedDocument.set(document);
    this.editDocument = {
      registru_id: document.registru_id ?? this.selectedRegistryId(),
      subject: document.subject,
      document_type: document.document_type,
      direction: document.direction,
      status: document.status,
      correspondent: document.correspondent,
      assigned_to: document.assigned_to,
      correspondent_party_id: document.correspondent_party_id ?? null,
      assigned_party_id: document.assigned_party_id ?? null,
      confidentiality: document.confidentiality,
      summary: document.summary,
      due_date: document.due_date ?? null,
      change_notes: '',
    };
    this.syncPartySelectionsFromDocument('edit', this.editDocument);
    this.editDialogOpen.set(true);
  }

  protected openCancelDialog(document: RegistraturaDocument): void {
    this.selectedDocument.set(document);
    this.cancelReason = '';
    this.cancelDialogOpen.set(true);
  }

  protected saveEditedDocument(): void {
    const document = this.selectedDocument();
    if (!document) {
      return;
    }

    this.syncDocumentPartyFields('edit');

    const payload: UpdateRegistraturaDocumentRequest = {
      ...this.editDocument,
      registru_id: this.editDocument.registru_id ?? this.selectedRegistryId(),
      due_date: this.normalizeOptionalDate(this.editDocument.due_date),
    };

    this.api.updateDocument(document.id, payload).subscribe({
      next: (updated) => {
        this.editDialogOpen.set(false);
        this.selectedDocument.set(updated);
        this.loadDocuments();
        this.loadWorkflowDocuments();
        this.loadArchiveDocuments();
        this.refreshDocumentDrawer(updated.id);
      },
      error: () => {
        this.editDialogOpen.set(false);
      },
    });
  }

  protected cancelSelectedDocument(): void {
    const document = this.selectedDocument();
    if (!document) {
      return;
    }

    const payload: CancelRegistraturaDocumentRequest = {
      reason: this.cancelReason.trim() || 'Anulare document',
    };

    this.api.cancelDocument(document.id, payload).subscribe({
      next: (updated) => {
        this.cancelDialogOpen.set(false);
        this.selectedDocument.set(updated);
        this.loadDocuments();
        this.loadWorkflowDocuments();
        this.loadArchiveDocuments();
        this.refreshDocumentDrawer(updated.id);
      },
      error: () => {
        this.cancelDialogOpen.set(false);
      },
    });
  }

  protected saveDocument(): void {
    this.syncDocumentPartyFields('new');

    const payload: CreateRegistraturaDocumentRequest = {
      ...this.newDocument,
      registru_id: this.newDocument.registru_id ?? this.selectedRegistryId(),
      due_date: this.normalizeOptionalDate(this.newDocument.due_date),
    };

    this.api.createDocument(payload).subscribe({
      next: () => {
        this.createDialogOpen.set(false);
        this.loadDocuments();
        this.loadWorkflowDocuments();
        this.loadArchiveDocuments();
      },
      error: () => {
        this.createDialogOpen.set(false);
      },
    });
  }

  protected saveBatchDocuments(): void {
    this.syncDocumentPartyFields('batch');

    const payload: BatchCreateRegistraturaDocumentRequest = {
      ...this.batchDocument,
      registru_id: this.batchDocument.registru_id || this.selectedRegistryId() || 0,
      due_date: this.normalizeOptionalDate(this.batchDocument.due_date),
    };

    this.api.batchCreateDocuments(payload).subscribe({
      next: () => {
        this.batchDialogOpen.set(false);
        this.loadDocuments();
        this.loadWorkflowDocuments();
        this.loadArchiveDocuments();
      },
      error: () => {
        this.batchDialogOpen.set(false);
      },
    });
  }

  protected onPartySelected(scope: 'new' | 'batch' | 'edit', kind: 'correspondent' | 'assigned', event: AutoCompleteSelectEvent): void {
    this.assignPartySelection(scope, kind, event.value as RegistraturaParty | null);
  }

  protected clearPartySelection(scope: 'new' | 'batch' | 'edit', kind: 'correspondent' | 'assigned'): void {
    this.assignPartySelection(scope, kind, null);
  }

  protected filterNrDocSuggestions(event: AutoCompleteCompleteEvent): void {
    const query = event.query.trim().toLowerCase();
    if (query.length < 1) {
      this.filteredNrDocSuggestions.set([]);
      return;
    }
    const suggestions = Array.from(
      new Set(
        this.documents()
          .map((document) => document.registry_number)
          .filter((value) => value.toLowerCase().includes(query))
      )
    ).slice(0, 20);
    this.filteredNrDocSuggestions.set(suggestions);
  }

  protected onNrDocSelect(event: AutoCompleteSelectEvent): void {
    this.filters.registry_number = String(event.value ?? '');
  }

  protected filterEmitentiSearch(event: AutoCompleteCompleteEvent): void {
    const query = event.query.trim();
    if (query.length < 1) {
      this.filteredEmitentiSearch.set([]);
      return;
    }
    this.api.partiesLookup(query).pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: (items: RegistraturaParty[]) => this.filteredEmitentiSearch.set(items),
      error: () => this.filteredEmitentiSearch.set([]),
    });
  }

  protected filterDestinatariSearch(event: AutoCompleteCompleteEvent): void {
    const query = event.query.trim();
    if (query.length < 1) {
      this.filteredDestinatariSearch.set([]);
      return;
    }
    this.api.partiesLookup(query).pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: (items: RegistraturaParty[]) => this.filteredDestinatariSearch.set(items),
      error: () => this.filteredDestinatariSearch.set([]),
    });
  }

  protected onEmitentSearchSelect(event: AutoCompleteSelectEvent): void {
    const party = event.value as RegistraturaParty | null;
    this.selectedEmitentSearch = party;
    this.filters.correspondent = party ? this.formatPartyDisplay(party) : '';
  }

  protected onDestinatarSearchSelect(event: AutoCompleteSelectEvent): void {
    const party = event.value as RegistraturaParty | null;
    this.selectedDestinatarSearch = party;
    this.filters.assigned_to = party ? this.formatPartyDisplay(party) : '';
  }

  protected clearEmitentSearch(): void {
    this.selectedEmitentSearch = null;
    this.filters.correspondent = '';
  }

  protected clearDestinatarSearch(): void {
    this.selectedDestinatarSearch = null;
    this.filters.assigned_to = '';
  }

  protected openPartyDialog(kind: 'physical' | 'legal' | 'institution', target: 'correspondent' | 'assigned'): void {
    this.partyForm = emptyPartyForm();
    this.partyForm.party_type = kind;
    this.partyDialogTarget.set(target);
    this.partyDialogOpen.set(true);
  }

  protected async saveParty(): Promise<void> {
    if (!this.partyForm.display_name.trim()) {
      return;
    }
    let institutionId = this.authz.institutionId();
    if (!institutionId) {
      await this.authz.reload();
      institutionId = this.authz.institutionId();
    }
    if (!institutionId) {
      try {
        const bootstrap = (await firstValueFrom(this.appApi.config())) as AppBootstrapConfig;
        institutionId = bootstrap.institutionId?.trim() ?? '';
      } catch {
        institutionId = '';
      }
    }
    if (!institutionId) {
      this.messages.add({
        severity: 'error',
        summary: 'Context instituțional lipsă',
        detail: 'Nu am putut determina instituția curentă pentru această persoană.',
      });
      return;
    }
    const payload: CreateRegistraturaPartyRequest = {
      ...this.partyForm,
      institution_id: institutionId,
      display_name: this.partyForm.display_name.trim(),
      code: this.partyForm.code?.trim() || undefined,
      short_name: this.partyForm.short_name?.trim() || undefined,
      first_name: this.partyForm.first_name?.trim() || undefined,
      last_name: this.partyForm.last_name?.trim() || undefined,
      legal_name: this.partyForm.legal_name?.trim() || undefined,
      identifier_code: this.partyForm.identifier_code?.trim() || undefined,
      tax_id: this.partyForm.tax_id?.trim() || undefined,
      phone_number: this.partyForm.phone_number?.trim() || undefined,
      email: this.partyForm.email?.trim() || undefined,
      address_line1: this.partyForm.address_line1?.trim() || undefined,
      address_line2: this.partyForm.address_line2?.trim() || undefined,
      locality: this.partyForm.locality?.trim() || undefined,
      county: this.partyForm.county?.trim() || undefined,
      country: this.partyForm.country?.trim() || 'RO',
      notes: this.partyForm.notes?.trim() || undefined,
    };

    this.api.createParty(payload).subscribe({
      next: (party) => {
        this.parties.set([...this.parties(), party]);
        if (this.partyDialogTarget() === 'correspondent') {
          this.newDocument.correspondent_party_id = party.id;
          this.newDocument.correspondent = party.display_name;
          this.batchDocument.correspondent_party_id = party.id;
          this.batchDocument.correspondent = party.display_name;
          this.newCorrespondentParty = party;
          this.batchCorrespondentParty = party;
          this.editCorrespondentParty = party;
        } else if (this.partyDialogTarget() === 'assigned') {
          this.newDocument.assigned_party_id = party.id;
          this.newDocument.assigned_to = party.display_name;
          this.batchDocument.assigned_party_id = party.id;
          this.batchDocument.assigned_to = party.display_name;
          this.newAssignedParty = party;
          this.batchAssignedParty = party;
          this.editAssignedParty = party;
        }
        this.messages.add({
          severity: 'success',
          summary: 'Persoană salvată',
          detail: party.display_name,
        });
        this.partyDialogOpen.set(false);
        this.partyDialogTarget.set(null);
      },
      error: (error) => {
        this.messages.add({
          severity: 'error',
          summary: 'Salvarea a eșuat',
          detail: error?.error?.message ?? 'Nu am putut salva persoana.',
        });
      },
    });
  }

  protected downloadPdf(): void {
    const payload: ExportRegistraturaDocumentsRequest = {
      registru_id: this.exportRequest.registru_id ?? this.selectedRegistryId(),
      start_date: this.normalizeOptionalDate(this.exportRequest.start_date),
      end_date: this.normalizeOptionalDate(this.exportRequest.end_date),
    };

    this.api.exportDocuments(payload).subscribe({
      next: (blob) => {
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = 'registratura.pdf';
        anchor.click();
        URL.revokeObjectURL(url);
        this.exportDialogOpen.set(false);
      },
      error: () => {
        this.exportDialogOpen.set(false);
      },
    });
  }

  protected statusSeverity(status: string): 'info' | 'success' | 'warn' | 'danger' | 'secondary' {
    switch (status) {
      case 'registered':
      case 'finalized':
      case 'archived':
        return 'success';
      case 'in_workflow':
        return 'warn';
      case 'draft':
        return 'secondary';
      default:
        return 'info';
    }
  }

  private loadBootstrapData(): void {
    this.loading.set(true);
    forkJoin({
      registries: this.api.registries().pipe(catchError(() => of([]))),
      defaultRegistry: this.api.defaultRegistry().pipe(catchError(() => of(null))),
      filters: this.api.documentFilters().pipe(catchError(() => of(null))),
      parties: this.api.partiesLookup().pipe(catchError(() => of([]))),
    }).subscribe({
      next: ({ registries, defaultRegistry, filters, parties }) => {
        const effectiveRegistries = registries.length > 0 ? registries : [defaultRegistryFallback()];
        this.registries.set(effectiveRegistries);
        this.parties.set(parties);
        this.filtersResponse.set(filters);

        const savedId = this.selectedRegistryId();
        const validSaved = savedId !== null && effectiveRegistries.some((registry) => registry.id === savedId);
        if (!validSaved) {
          const defaultFromList = effectiveRegistries.find((registry) => registry.isDefault) ?? null;
          const fallbackRegistry = defaultFromList ?? defaultRegistry ?? effectiveRegistries[0] ?? defaultRegistryFallback();
          this.selectedRegistryId.set(fallbackRegistry?.id ?? null);
          this.selectedRegistryModel = fallbackRegistry?.id ?? null;
        }
        const currentRegistryId = this.selectedRegistryId();
        this.newDocument.registru_id = currentRegistryId;
        this.batchDocument.registru_id = currentRegistryId ?? 0;
        this.exportRequest.registru_id = currentRegistryId;
        if (parties.length > 0) {
          this.applyDefaultParties(this.newDocument, this.newDocument.direction);
          this.applyDefaultParties(this.batchDocument, this.batchDocument.direction);
          this.syncPartySelectionsFromDocument('new', this.newDocument);
          this.syncPartySelectionsFromDocument('batch', this.batchDocument);
        }

        this.loadDocuments();
        this.loadWorkflowDocuments();
        this.loadArchiveDocuments();
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
      },
    });
  }

  private toOptions(values: string[]): Array<{ label: string; value: string }> {
    return values.map((value) => ({ label: value, value }));
  }

  private applyDefaultParties(document: CreateRegistraturaDocumentRequest | BatchCreateRegistraturaDocumentRequest, direction: string): void {
    const organization = this.parties().find((party) => party.is_default_organization) ?? null;
    if (!organization) {
      return;
    }

    if (direction === 'intrare') {
      document.assigned_party_id = organization.id;
      document.assigned_to = organization.display_name;
    } else if (direction === 'iesire') {
      document.correspondent_party_id = organization.id;
      document.correspondent = organization.display_name;
    } else if (direction === 'intern') {
      document.assigned_party_id = organization.id;
      document.assigned_to = organization.display_name;
      document.correspondent_party_id = organization.id;
      document.correspondent = organization.display_name;
    }
  }

  private assignPartySelection(
    scope: 'new' | 'batch' | 'edit',
    kind: 'correspondent' | 'assigned',
    party: RegistraturaParty | null,
  ): void {
    const target = scope === 'new' ? this.newDocument : scope === 'batch' ? this.batchDocument : this.editDocument;
    if (kind === 'correspondent') {
      target.correspondent_party_id = party?.id ?? null;
      target.correspondent = party?.display_name ?? '';
      if (scope === 'new') this.newCorrespondentParty = party;
      if (scope === 'batch') this.batchCorrespondentParty = party;
      if (scope === 'edit') this.editCorrespondentParty = party;
    } else {
      target.assigned_party_id = party?.id ?? null;
      target.assigned_to = party?.display_name ?? '';
      if (scope === 'new') this.newAssignedParty = party;
      if (scope === 'batch') this.batchAssignedParty = party;
      if (scope === 'edit') this.editAssignedParty = party;
    }
  }

  private syncPartySelectionsFromDocument(
    scope: 'new' | 'batch' | 'edit',
    document: CreateRegistraturaDocumentRequest | BatchCreateRegistraturaDocumentRequest | UpdateRegistraturaDocumentRequest,
  ): void {
    this.assignPartySelection(scope, 'correspondent', this.resolveParty(document.correspondent_party_id ?? null));
    this.assignPartySelection(scope, 'assigned', this.resolveParty(document.assigned_party_id ?? null));
  }

  private syncDocumentPartyFields(scope: 'new' | 'batch' | 'edit'): void {
    const target = scope === 'new' ? this.newDocument : scope === 'batch' ? this.batchDocument : this.editDocument;
    const correspondent = scope === 'new' ? this.newCorrespondentParty : scope === 'batch' ? this.batchCorrespondentParty : this.editCorrespondentParty;
    const assigned = scope === 'new' ? this.newAssignedParty : scope === 'batch' ? this.batchAssignedParty : this.editAssignedParty;
    target.correspondent_party_id = correspondent?.id ?? target.correspondent_party_id ?? null;
    target.correspondent = correspondent?.display_name ?? target.correspondent ?? '';
    target.assigned_party_id = assigned?.id ?? target.assigned_party_id ?? null;
    target.assigned_to = assigned?.display_name ?? target.assigned_to ?? '';
  }

  protected filterParties(event: AutoCompleteCompleteEvent): void {
    const query = event.query.trim();
    if (query.length < 1) {
      this.partySuggestions.set([]);
      return;
    }
    this.api.partiesLookup(query).pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: (items: RegistraturaParty[]) => this.partySuggestions.set(items),
      error: () => this.partySuggestions.set([]),
    });
  }

  protected partyTypeLabel(value: string): string {
    switch (value) {
      case 'physical':
        return 'Persoană fizică';
      case 'legal':
        return 'Persoană juridică';
      case 'institution':
        return 'Instituție';
      default:
        return value;
    }
  }

  protected partyTypeIcon(value: string): string {
    switch (value) {
      case 'physical':
        return 'pi pi-user';
      case 'legal':
        return 'pi pi-building';
      case 'institution':
        return 'pi pi-sitemap';
      default:
        return 'pi pi-users';
    }
  }

  protected formatPartyDisplay(party: RegistraturaParty): string {
    if (!party) {
      return '';
    }
    if (party.party_type === 'physical') {
      const parts = [party.first_name?.trim(), party.last_name?.trim()].filter(Boolean);
      return parts.length > 0 ? parts.join(' ') : party.display_name;
    }
    return party.display_name || party.legal_name || party.short_name || party.code;
  }

  protected formatEntityDisplay(entity: RegistraturaParty): string {
    return this.formatPartyDisplay(entity);
  }

  protected getEntityDisplayText = (entity: RegistraturaParty): string => this.formatPartyDisplay(entity);

  protected getEntityTypeIcon(entity: RegistraturaParty): string {
    return this.partyTypeIcon(entity?.party_type ?? '');
  }

  protected getEntityTypeBadge(entity: RegistraturaParty): string {
    return this.partyTypeLabel(entity?.party_type ?? '');
  }

  private resolveParty(partyId: string | null): RegistraturaParty | null {
    if (!partyId) {
      return null;
    }
    return this.parties().find((item) => item.id === partyId) ?? null;
  }

  private normalizeDateOrText(value: unknown): string {
    if (value instanceof Date) {
      return this.formatDate(value);
    }
    return String(value);
  }

  private normalizeOptionalDate(value: unknown): string | null {
    if (!value) {
      return null;
    }
    if (value instanceof Date) {
      return this.formatDate(value);
    }
    return String(value);
  }

  private formatDate(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  private loadSavedRegistryId(): number | null {
    const raw = localStorage.getItem(this.registryStorageKey);
    if (!raw) {
      return null;
    }
    const parsed = Number(raw);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
  }

  private refreshDocumentDrawer(documentId: string): void {
    forkJoin({
      detail: this.api.document(documentId),
      versions: this.api.documentVersions(documentId),
      attachments: this.api.documentAttachments(documentId),
      links: this.linksApi.listLinks('registratura', documentId).pipe(catchError(() => of([]))),
    }).subscribe({
      next: ({ detail, versions, attachments, links }) => {
        this.selectedDocument.set(detail);
        this.selectedDocumentVersions.set(versions);
        this.selectedDocumentAttachments.set(attachments);
        this.selectedDocumentLinks.set(links);
      },
      error: () => {
        this.selectedDocumentVersions.set([]);
        this.selectedDocumentAttachments.set([]);
        this.selectedDocumentLinks.set([]);
      },
    });
  }
}
