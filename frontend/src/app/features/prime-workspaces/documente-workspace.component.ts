import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ButtonModule } from 'primeng/button';
import { ButtonGroupModule } from 'primeng/buttongroup';
import { CardModule } from 'primeng/card';
import { ChipModule } from 'primeng/chip';
import { DatePickerModule } from 'primeng/datepicker';
import { DialogModule } from 'primeng/dialog';
import { DrawerModule } from 'primeng/drawer';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { SelectButtonModule } from 'primeng/selectbutton';
import { TableLazyLoadEvent, TableModule } from 'primeng/table';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TextareaModule } from 'primeng/textarea';
import { TooltipModule } from 'primeng/tooltip';
import { TreeModule } from 'primeng/tree';
import { TreeNode } from 'primeng/api';

interface RegistryRow {
  id: string;
  nrDoc: string;
  tip: string;
  continut: string;
  emitent: string;
  destinatar: string;
  dataIntrare: string;
  dataIesire: string;
  status: string;
  compartimente: string;
  nrExtern: string;
  activitate: string;
}

interface WorkflowRow {
  id: string;
  nrDoc: string;
  data: string;
  continut: string;
  emitent: string;
  compartiment: string;
  pas: string;
  termen: string;
  status: string;
}

interface ArchiveSearchRow {
  id: string;
  nr: string;
  data: string;
  subiect: string;
  dosar: string;
  tip: string;
}

interface ArchiveFileRow {
  id: string;
  fisier: string;
  ocr: string;
  sursa: string;
  data: string;
}

@Component({
  selector: 'app-documente-workspace',
  imports: [
    CommonModule,
    FormsModule,
    ButtonModule,
    ButtonGroupModule,
    CardModule,
    ChipModule,
    DatePickerModule,
    DialogModule,
    DrawerModule,
    IconFieldModule,
    InputIconModule,
    InputTextModule,
    SelectModule,
    SelectButtonModule,
    TableModule,
    TabsModule,
    TagModule,
    TextareaModule,
    TooltipModule,
    TreeModule,
  ],
  template: `
    <section class="document-workspace flex h-[calc(100dvh-6rem)] min-h-0 flex-col overflow-hidden">
      <p-tabs value="registratura" class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <p-tablist class="shrink-0">
          <p-tab value="registratura">Registratură</p-tab>
          <p-tab value="flux">Flux documente</p-tab>
          <p-tab value="arhiva">eArhivă</p-tab>
        </p-tablist>

        <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
          <p-tabpanel value="registratura" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
              @if (registraturaFilterOpen()) {
                <section class="search-panel mb-3 shrink-0">
                  <header class="search-panel-header">
                    <div class="flex items-center gap-2 font-semibold">
                      <i class="pi pi-filter"></i>
                      <span>Filtrare documente</span>
                    </div>
                    <p-button icon="pi pi-times" [text]="true" [rounded]="true" severity="secondary" size="small" (onClick)="registraturaFilterOpen.set(false)" />
                  </header>
                  <div class="search-panel-content">
                    <div class="search-row">
                      <label class="search-field">
                        <span><i class="pi pi-hashtag"></i> Nr. document</span>
                        <input pInputText placeholder="ex: 123" />
                      </label>
                      <label class="search-field">
                        <span><i class="pi pi-tag"></i> Tip document</span>
                        <p-select appendTo="body" [options]="tipOptions" placeholder="Toate" [showClear]="true" />
                      </label>
                      <label class="search-field">
                        <span><i class="pi pi-file"></i> Nr. extern</span>
                        <input pInputText placeholder="ex: ABC-123" />
                      </label>
                    </div>
                    <div class="search-row">
                      <label class="search-field search-field-wide">
                        <span><i class="pi pi-send"></i> Emitent</span>
                        <input pInputText placeholder="Caută emitent..." />
                      </label>
                      <label class="search-field search-field-wide">
                        <span><i class="pi pi-inbox"></i> Destinatar</span>
                        <input pInputText placeholder="Caută destinatar..." />
                      </label>
                    </div>
                    <div class="search-row">
                      <label class="search-field search-field-wide">
                        <span><i class="pi pi-calendar-plus"></i> Data intrare</span>
                        <div class="date-range">
                          <p-datepicker appendTo="body" dateFormat="dd.mm.yy" placeholder="De la" [showIcon]="true" />
                          <span>—</span>
                          <p-datepicker appendTo="body" dateFormat="dd.mm.yy" placeholder="Până la" [showIcon]="true" />
                        </div>
                      </label>
                      <label class="search-field search-field-wide">
                        <span><i class="pi pi-calendar-minus"></i> Data ieșire</span>
                        <div class="date-range">
                          <p-datepicker appendTo="body" dateFormat="dd.mm.yy" placeholder="De la" [showIcon]="true" />
                          <span>—</span>
                          <p-datepicker appendTo="body" dateFormat="dd.mm.yy" placeholder="Până la" [showIcon]="true" />
                        </div>
                      </label>
                    </div>
                  </div>
                  <footer class="search-panel-footer">
                    <p-button icon="pi pi-refresh" label="Resetare" severity="secondary" [outlined]="true" size="small" />
                    <p-button icon="pi pi-search" label="Caută documente" size="small" />
                  </footer>
                </section>
              }

              <div class="mb-3 grid shrink-0 grid-cols-[auto_1fr_auto] items-center gap-2">
                <div class="justify-self-start">
                  <p-button
                    icon="pi pi-search"
                    severity="secondary"
                    [outlined]="!registraturaFilterOpen()"
                    [rounded]="true"
                    size="small"
                    ariaLabel="Caută și filtrează documente"
                    (onClick)="registraturaFilterOpen.update((open) => !open)"
                  />
                </div>
                <p-buttongroup styleClass="justify-self-center">
                  <p-button icon="pi pi-download" label="Intrare" severity="info" size="small" (onClick)="openDocumentDialog('intrare')" />
                  <p-button icon="pi pi-upload" label="Ieșire" severity="success" size="small" (onClick)="openDocumentDialog('iesire')" />
                  <p-button icon="pi pi-clone" label="Multiplu" severity="secondary" size="small" (onClick)="openBatchDialog()" />
                </p-buttongroup>
                <div class="flex items-center gap-2 justify-self-end">
                  <span class="hidden text-sm text-muted-color sm:inline">{{ registryTotal() }} documente</span>
                  <p-button icon="pi pi-file-pdf" label="Export PDF" severity="secondary" [outlined]="true" size="small" (onClick)="openPdfExportDialog()" />
                </div>
              </div>

              <p-table
                class="flex min-h-0 flex-1 flex-col overflow-hidden"
                styleClass="p-datatable-sm p-datatable-gridlines registry-table"
                [value]="registryRows()"
                [lazy]="true"
                [loading]="loading()"
                [scrollable]="true"
                scrollHeight="flex"
                [stripedRows]="true"
                [paginator]="true"
                [rows]="20"
                [rowsPerPageOptions]="[10,20,50,100]"
                [totalRecords]="registryTotal()"
                [showCurrentPageReport]="true"
                currentPageReportTemplate="Se afișează {first} - {last} din {totalRecords} documente"
                dataKey="id"
                [expandedRowKeys]="expandedRows()"
                (onLazyLoad)="loadRegistry($event)"
              >
                <ng-template pTemplate="header">
                  <tr>
                    <th style="width:3rem"></th>
                    <th pSortableColumn="nrDoc" style="width:7rem">Nr. Doc <p-sortIcon field="nrDoc" /></th>
                    <th pSortableColumn="tip" style="width:7rem">Tip <p-sortIcon field="tip" /></th>
                    <th pSortableColumn="continut">Conținut <p-sortIcon field="continut" /></th>
                    <th pSortableColumn="emitent" style="width:12rem">Emitent <p-sortIcon field="emitent" /></th>
                    <th pSortableColumn="destinatar" style="width:12rem">Destinatar <p-sortIcon field="destinatar" /></th>
                    <th pSortableColumn="dataIntrare" style="width:9rem">Intrare <p-sortIcon field="dataIntrare" /></th>
                    <th pSortableColumn="dataIesire" style="width:9rem">Ieșire <p-sortIcon field="dataIesire" /></th>
                    <th pSortableColumn="status" style="width:9rem">Status <p-sortIcon field="status" /></th>
                    <th style="width:12rem">Acțiuni</th>
                  </tr>
                  <tr class="filter-row">
                    <th></th>
                    <th><input pInputText class="w-full filter-input" placeholder="Nr..." /></th>
                    <th><input pInputText class="w-full filter-input" placeholder="Tip..." /></th>
                    <th><input pInputText class="w-full filter-input" placeholder="Conținut..." /></th>
                    <th><input pInputText class="w-full filter-input" placeholder="Emitent..." /></th>
                    <th><input pInputText class="w-full filter-input" placeholder="Destinatar..." /></th>
                    <th></th>
                    <th></th>
                    <th><p-select appendTo="body" class="w-full" [options]="statusOptions" placeholder="Status" [showClear]="true" /></th>
                    <th></th>
                  </tr>
                </ng-template>
                <ng-template pTemplate="body" let-document let-expanded="expanded">
                  <tr>
                    <td>
                      <p-button [pRowToggler]="document" [text]="true" [rounded]="true" severity="secondary" [icon]="expanded ? 'pi pi-chevron-down' : 'pi pi-chevron-right'" pTooltip="Detalii" />
                    </td>
                    <td class="font-mono">{{ document.nrDoc }}</td>
                    <td><p-tag [value]="document.tip" [severity]="document.tip === 'Intrare' ? 'warn' : 'success'" /></td>
                    <td><div class="max-w-[28rem] truncate" [pTooltip]="document.continut">{{ document.continut }}</div></td>
                    <td>{{ document.emitent }}</td>
                    <td>{{ document.destinatar }}</td>
                    <td>{{ document.dataIntrare }}</td>
                    <td>{{ document.dataIesire }}</td>
                    <td><p-tag [value]="document.status" severity="info" /></td>
                    <td>
                      <div class="flex justify-center gap-1">
                        <p-button icon="pi pi-history" [rounded]="true" [text]="true" severity="info" size="small" pTooltip="Istoric" />
                        <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" size="small" pTooltip="Editează" />
                        <p-button icon="pi pi-print" [rounded]="true" [text]="true" severity="secondary" size="small" pTooltip="Imprimă" />
                        <p-button icon="pi pi-share-alt" [rounded]="true" [text]="true" severity="warn" size="small" pTooltip="Flux workflow" />
                      </div>
                    </td>
                  </tr>
                </ng-template>
                <ng-template pTemplate="expandedrow" let-document>
                  <tr>
                    <td colspan="10">
                      <div class="grid gap-4 p-4 md:grid-cols-2 lg:grid-cols-4">
                        <div><span class="font-semibold">Compartimente:</span><br />{{ document.compartimente || '-' }}</div>
                        <div><span class="font-semibold">Nr. extern:</span><br />{{ document.nrExtern || '-' }}</div>
                        <div><span class="font-semibold">Activitate:</span><br />{{ document.activitate || '-' }}</div>
                        <div><span class="font-semibold">Status:</span><br />{{ document.status }}</div>
                      </div>
                    </td>
                  </tr>
                </ng-template>
                <ng-template pTemplate="emptymessage">
                  <tr><td colspan="10" class="p-8 text-center text-muted-color">Nu s-au găsit documente</td></tr>
                </ng-template>
              </p-table>

              <p-dialog
                [visible]="documentDialogOpen()"
                (visibleChange)="documentDialogOpen.set($event)"
                [modal]="true"
                [draggable]="false"
                [resizable]="true"
                [maximizable]="true"
                header="Adaugă document nou"
                styleClass="registry-dialog"
                [style]="{ width: 'min(72rem, 94vw)' }"
                [contentStyle]="{ 'max-height': '78dvh', overflow: 'auto' }"
              >
                <div class="flex justify-center pb-3">
                  <p-selectbutton
                    [options]="registrationModeOptions"
                    [ngModel]="registrationMode()"
                    optionLabel="label"
                    optionValue="value"
                    (ngModelChange)="registrationMode.set($event)"
                  />
                </div>

                <div class="grid gap-4 lg:grid-cols-2">
                  <section class="dialog-section">
                    <h3>Date document</h3>
                    <label class="search-field">
                      <span>Conținut <strong class="text-primary">*</strong></span>
                      <textarea pTextarea rows="5" placeholder="Descrierea documentului"></textarea>
                    </label>
                    <div class="grid gap-3 md:grid-cols-2">
                      <label class="search-field">
                        <span>Tip document</span>
                        <p-select appendTo="body" [options]="tipOptions" [ngModel]="registrationMode()" optionLabel="label" optionValue="value" />
                      </label>
                      <label class="search-field">
                        <span>Compartiment</span>
                        <p-select appendTo="body" [options]="compartimentOptions" placeholder="Alege compartiment" />
                      </label>
                    </div>
                    <label class="search-field">
                      <span>Activitate / observații</span>
                      <input pInputText placeholder="Activitate, termen, context intern" />
                    </label>
                  </section>

                  <section class="dialog-section">
                    <h3>Părți și corespondență</h3>
                    <label class="search-field">
                      <span>Emitent <strong class="text-primary">*</strong></span>
                      <input pInputText placeholder="Caută sau adaugă emitent" />
                    </label>
                    <label class="search-field">
                      <span>Destinatar <strong class="text-primary">*</strong></span>
                      <input pInputText placeholder="Caută sau adaugă destinatar" />
                    </label>
                    <div class="grid gap-3 md:grid-cols-2">
                      <label class="search-field">
                        <span>Nr. extern</span>
                        <input pInputText placeholder="Nr. extern" />
                      </label>
                      <label class="search-field">
                        <span>Data nr. extern</span>
                        <p-datepicker appendTo="body" dateFormat="dd.mm.yy" [showIcon]="true" />
                      </label>
                    </div>
                  </section>
                </div>

                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="documentDialogOpen.set(false)" />
                    <p-button label="Salvează document" icon="pi pi-check" (onClick)="documentDialogOpen.set(false)" />
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
                [style]="{ width: 'min(56rem, 94vw)' }"
              >
                <div class="grid gap-4 md:grid-cols-2">
                  <label class="search-field">
                    <span>Număr documente</span>
                    <input pInputText type="number" min="1" max="20" placeholder="1 - 20" />
                  </label>
                  <label class="search-field">
                    <span>Registru</span>
                    <p-select appendTo="body" [options]="registruOptions" placeholder="Alege registrul" />
                  </label>
                  <label class="search-field">
                    <span>Data intrare</span>
                    <p-datepicker appendTo="body" dateFormat="dd.mm.yy" [showIcon]="true" />
                  </label>
                  <label class="search-field">
                    <span>Compartiment</span>
                    <p-select appendTo="body" [options]="compartimentOptions" placeholder="Compartiment responsabil" />
                  </label>
                  <label class="search-field md:col-span-2">
                    <span>Conținut / observații comune</span>
                    <textarea pTextarea rows="4" placeholder="Descriere opțională aplicată documentelor generate"></textarea>
                  </label>
                  <div class="md:col-span-2 rounded-2xl border border-primary-200 bg-primary-50 p-4 text-sm text-primary-900 dark:border-primary-700 dark:bg-primary-950 dark:text-primary-100">
                    <div class="mb-1 flex items-center gap-2 font-semibold">
                      <i class="pi pi-info-circle"></i>
                      Generarea multiplă trebuie să consume numerotarea server-side
                    </div>
                    <p class="m-0">
                      UI-ul păstrează fluxul Costești; următorul contract backend trebuie să creeze atomar N poziții în registrul ales și să returneze numerele generate.
                    </p>
                  </div>
                </div>

                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="batchDialogOpen.set(false)" />
                    <p-button label="Generează" icon="pi pi-clone" (onClick)="batchDialogOpen.set(false)" />
                  </div>
                </ng-template>
              </p-dialog>

              <p-dialog
                [visible]="pdfExportDialogOpen()"
                (visibleChange)="pdfExportDialogOpen.set($event)"
                [modal]="true"
                [draggable]="false"
                header="Export PDF registratură"
                styleClass="registry-dialog"
                [style]="{ width: 'min(38rem, 94vw)' }"
              >
                <div class="grid gap-4">
                  <p class="m-0 text-sm text-muted-color">
                    Exportă registrul curent într-un PDF pe intervalul selectat, similar fluxului Costești.
                  </p>
                  <div class="date-range">
                    <p-datepicker appendTo="body" dateFormat="dd.mm.yy" placeholder="De la" [showIcon]="true" />
                    <span>—</span>
                    <p-datepicker appendTo="body" dateFormat="dd.mm.yy" placeholder="Până la" [showIcon]="true" />
                  </div>
                </div>
                <ng-template pTemplate="footer">
                  <div class="flex justify-end gap-2">
                    <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="pdfExportDialogOpen.set(false)" />
                    <p-button label="Exportă PDF" icon="pi pi-file-pdf" (onClick)="pdfExportDialogOpen.set(false)" />
                  </div>
                </ng-template>
              </p-dialog>
            </div>
          </p-tabpanel>

          <p-tabpanel value="flux" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
              <p-tabs value="queue" class="flex min-h-0 flex-1 flex-col overflow-hidden">
                <p-tablist class="shrink-0">
                  <p-tab value="queue">Coada mea</p-tab>
                  <p-tab value="signatures">Mapă semnături</p-tab>
                  <p-tab value="pipeline">Evidență completă</p-tab>
                </p-tablist>
                <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0 pt-3">
                  @for (tab of workflowTabs; track tab.value) {
                    <p-tabpanel [value]="tab.value" class="flex min-h-0 flex-1 overflow-hidden p-0">
                      <div class="flex min-h-0 flex-1 flex-col overflow-hidden">
                        @if (tab.value === 'pipeline') {
                          <div class="mb-2 flex shrink-0 flex-wrap items-center gap-2">
                            @for (stat of workflowStats; track stat.status) {
                              <p-card styleClass="cursor-pointer min-w-[7rem]">
                                <div class="flex flex-col items-center gap-1">
                                  <p-tag [value]="stat.label" [severity]="stat.severity" />
                                  <span class="text-xl font-bold">{{ stat.count }}</span>
                                </div>
                              </p-card>
                            }
                            <p-button label="Arată arhivate" severity="secondary" size="small" />
                          </div>
                        }

                        <p-table
                          class="flex min-h-0 flex-1 flex-col overflow-hidden"
                          styleClass="p-datatable-sm p-datatable-gridlines workflow-table"
                          [value]="workflowRows()"
                          [lazy]="true"
                          [loading]="loading()"
                          [paginator]="true"
                          [rows]="20"
                          [totalRecords]="workflowTotal()"
                          [rowsPerPageOptions]="[20,50,100]"
                          [showCurrentPageReport]="true"
                          currentPageReportTemplate="{first}–{last} din {totalRecords}"
                          [scrollable]="true"
                          scrollHeight="flex"
                          (onLazyLoad)="loadWorkflow($event)"
                        >
                          <ng-template pTemplate="header">
                            <tr>
                              <th pSortableColumn="nrDoc" style="width:7rem">Nr. <p-sortIcon field="nrDoc" /></th>
                              <th pSortableColumn="data" style="width:7rem">Dată <p-sortIcon field="data" /></th>
                              <th pSortableColumn="continut">Conținut <p-sortIcon field="continut" /></th>
                              <th pSortableColumn="emitent" style="width:12rem">Emitent <p-sortIcon field="emitent" /></th>
                              <th pSortableColumn="compartiment" style="width:13rem">Compartiment <p-sortIcon field="compartiment" /></th>
                              <th pSortableColumn="pas" style="width:12rem">Pas <p-sortIcon field="pas" /></th>
                              <th style="width:9rem">Status</th>
                              <th style="width:5rem"></th>
                            </tr>
                            <tr class="filter-row">
                              <th><input pInputText class="w-full filter-input" placeholder="Nr..." /></th>
                              <th></th>
                              <th><input pInputText class="w-full filter-input" placeholder="Conținut..." /></th>
                              <th><input pInputText class="w-full filter-input" placeholder="Emitent..." /></th>
                              <th><input pInputText class="w-full filter-input" placeholder="Compartiment..." /></th>
                              <th><input pInputText class="w-full filter-input" placeholder="Pas..." /></th>
                              <th></th>
                              <th></th>
                            </tr>
                          </ng-template>
                          <ng-template pTemplate="body" let-doc>
                            <tr>
                              <td>{{ doc.nrDoc }}</td>
                              <td>{{ doc.data }}</td>
                              <td class="max-w-xs truncate" [title]="doc.continut">{{ doc.continut }}</td>
                              <td>{{ doc.emitent }}</td>
                              <td>{{ doc.compartiment }}</td>
                              <td>{{ doc.pas }}</td>
                              <td><p-tag [value]="doc.status" severity="info" /></td>
                              <td><p-button icon="pi pi-eye" [rounded]="true" size="small" severity="secondary" pTooltip="Deschide" (onClick)="workflowPanelOpen.set(true)" /></td>
                            </tr>
                          </ng-template>
                          <ng-template pTemplate="emptymessage">
                            <tr><td colspan="8" class="p-8 text-center text-muted-color">Niciun document în {{ tab.label.toLowerCase() }}.</td></tr>
                          </ng-template>
                        </p-table>
                      </div>
                    </p-tabpanel>
                  }
                </p-tabpanels>
              </p-tabs>
            </div>
          </p-tabpanel>

          <p-tabpanel value="arhiva" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col overflow-hidden p-4">
              <p-tabs value="cautare" class="flex min-h-0 flex-1 flex-col overflow-hidden">
                <p-tablist class="shrink-0">
                  <p-tab value="cautare">Căutare</p-tab>
                  <p-tab value="dosare">Dosare</p-tab>
                  <p-tab value="ingestie">Ingestie</p-tab>
                  <p-tab value="nomenclator">Nomenclator</p-tab>
                </p-tablist>
                <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0 pt-3">
                  <p-tabpanel value="cautare" class="flex min-h-0 flex-1 overflow-hidden p-0">
                    <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden">
                      <div class="flex shrink-0 gap-2">
                        <p-iconfield class="flex-1">
                          <p-inputicon styleClass="pi pi-search" />
                          <input pInputText placeholder="Căutare în conținut, număr, emitent, subiect..." class="w-full" />
                        </p-iconfield>
                        <p-button label="Caută" icon="pi pi-search" />
                      </div>
                      <div class="flex shrink-0 flex-wrap items-center gap-2">
                        <p-select appendTo="body" [options]="archiveCategoryOptions" placeholder="Categorie" [showClear]="true" />
                        <p-select appendTo="body" [options]="archiveYearOptions" placeholder="An" [showClear]="true" />
                        <p-chip label="Flux documente" [removable]="true" />
                      </div>
                      <p-table class="flex min-h-0 flex-1 flex-col overflow-hidden" styleClass="p-datatable-sm" [value]="archiveSearchRows()" [lazy]="true" [totalRecords]="archiveSearchTotal()" [rows]="20" [paginator]="true" [scrollable]="true" scrollHeight="flex">
                        <ng-template pTemplate="header">
                          <tr><th style="width:90px">Nr.</th><th style="width:80px">Dată</th><th>Subiect / Conținut</th><th style="width:150px">Dosar</th><th style="width:90px">Tip</th><th style="width:70px">Acțiuni</th></tr>
                        </ng-template>
                        <ng-template pTemplate="body" let-item>
                          <tr><td class="font-mono text-sm">{{ item.nr }}</td><td>{{ item.data }}</td><td>{{ item.subiect }}</td><td>{{ item.dosar }}</td><td><p-tag [value]="item.tip" severity="success" /></td><td><p-button icon="pi pi-eye" [text]="true" size="small" pTooltip="Vizualizează" /></td></tr>
                        </ng-template>
                        <ng-template pTemplate="emptymessage"><tr><td colspan="6" class="p-8 text-center text-muted-color">Niciun document găsit</td></tr></ng-template>
                      </p-table>
                    </div>
                  </p-tabpanel>

                  <p-tabpanel value="dosare" class="flex min-h-0 flex-1 overflow-hidden p-0">
                    <div class="flex min-h-0 flex-1 overflow-hidden">
                      <aside class="flex w-72 min-w-64 flex-col gap-3 border-r border-surface p-4">
                        <p-select appendTo="body" [options]="archiveYearOptions" placeholder="An arhivistic" />
                        <p-button label="Dosar nou" icon="pi pi-plus" size="small" />
                        <p-tree [value]="archiveTree" selectionMode="single" class="min-h-0 flex-1 overflow-y-auto" styleClass="border-none p-0" />
                      </aside>
                      <main class="flex min-w-0 flex-1 flex-col gap-4 p-4">
                        <div class="flex flex-wrap items-start justify-between gap-3">
                          <div>
                            <h2 class="m-0 text-lg font-semibold">Alege un dosar arhivistic</h2>
                            <p class="m-0 text-sm text-muted-color">Fond, serie, termen de păstrare și fișiere asociate.</p>
                          </div>
                          <p-tag value="deschis" severity="success" />
                        </div>
                        <p-table [value]="archiveFileRows()" styleClass="p-datatable-sm" [rows]="20" [paginator]="false">
                          <ng-template pTemplate="header"><tr><th>Fișier</th><th>Status OCR</th><th>Sursă</th><th>Dată</th><th>Acțiuni</th></tr></ng-template>
                          <ng-template pTemplate="body" let-file><tr><td>{{ file.fisier }}</td><td><p-tag [value]="file.ocr" severity="info" /></td><td>{{ file.sursa }}</td><td>{{ file.data }}</td><td><p-button icon="pi pi-eye" [text]="true" size="small" /></td></tr></ng-template>
                          <ng-template pTemplate="emptymessage"><tr><td colspan="5" class="p-8 text-center text-muted-color">Niciun fișier în acest dosar</td></tr></ng-template>
                        </p-table>
                      </main>
                    </div>
                  </p-tabpanel>

                  <p-tabpanel value="ingestie" class="p-0">
                    <div class="grid place-items-center rounded-2xl border border-dashed border-surface p-10 text-center text-muted-color">
                      <i class="pi pi-upload mb-3 text-3xl"></i>
                      <p class="m-0 font-semibold">Zonă ingestie documente arhivă</p>
                      <p class="m-0 text-sm">Încărcare manuală, OCR, clasificare și atașare la dosar arhivistic.</p>
                    </div>
                  </p-tabpanel>
                  <p-tabpanel value="nomenclator" class="p-0">
                    <div class="grid place-items-center rounded-2xl border border-dashed border-surface p-10 text-center text-muted-color">Nomenclator arhivistic: fonduri, serii, termene și reguli.</div>
                  </p-tabpanel>
                </p-tabpanels>
              </p-tabs>
            </div>
          </p-tabpanel>
        </p-tabpanels>
      </p-tabs>

      <p-drawer [visible]="workflowPanelOpen()" (visibleChange)="workflowPanelOpen.set($event)" position="right" header="Flux document" [style]="{ width: 'min(42rem, 100vw)' }">
        <div class="grid gap-3">
          <p-card>
            <h3 class="m-0 mb-2">Acțiuni flux</h3>
            <div class="flex flex-wrap gap-2">
              <p-button label="Aprobă" icon="pi pi-check" />
              <p-button label="Returnează" icon="pi pi-replay" severity="warn" />
              <p-button label="Atribuie" icon="pi pi-user-plus" severity="secondary" />
            </div>
          </p-card>
          <p-card><p class="m-0 text-muted-color">Aici vin timeline-ul, comentariile, atașamentele și tranzițiile reale ale workflow-ului.</p></p-card>
        </div>
      </p-drawer>
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
      color: var(--p-text-color);
    }

    :host ::ng-deep .p-datatable,
    :host ::ng-deep .p-datatable-table-container {
      min-height: 0;
    }

    :host ::ng-deep .p-datatable-thead {
      position: sticky;
      top: 0;
      z-index: 3;
    }

    :host ::ng-deep .registry-table,
    :host ::ng-deep .workflow-table,
    :host ::ng-deep .registry-table .p-datatable-table-container,
    :host ::ng-deep .workflow-table .p-datatable-table-container,
    :host ::ng-deep .registry-table .p-datatable-table,
    :host ::ng-deep .workflow-table .p-datatable-table,
    :host ::ng-deep .registry-table .p-datatable-tbody > tr,
    :host ::ng-deep .workflow-table .p-datatable-tbody > tr,
    :host ::ng-deep .registry-table .p-datatable-tbody > tr > td,
    :host ::ng-deep .workflow-table .p-datatable-tbody > tr > td {
      background: var(--p-content-background);
      color: var(--p-text-color);
    }

    :host-context(.app-dark) ::ng-deep .registry-table,
    :host-context(.app-dark) ::ng-deep .workflow-table,
    :host-context(.app-dark) ::ng-deep .registry-table .p-datatable-table-container,
    :host-context(.app-dark) ::ng-deep .workflow-table .p-datatable-table-container,
    :host-context(.app-dark) ::ng-deep .registry-table .p-datatable-table,
    :host-context(.app-dark) ::ng-deep .workflow-table .p-datatable-table,
    :host-context(.app-dark) ::ng-deep .registry-table .p-datatable-tbody > tr,
    :host-context(.app-dark) ::ng-deep .workflow-table .p-datatable-tbody > tr,
    :host-context(.app-dark) ::ng-deep .registry-table .p-datatable-tbody > tr > td,
    :host-context(.app-dark) ::ng-deep .workflow-table .p-datatable-tbody > tr > td {
      background: var(--p-surface-900);
    }

    :host ::ng-deep .registry-table .p-datatable-thead > tr > th,
    :host ::ng-deep .workflow-table .p-datatable-thead > tr > th {
      background: var(--p-surface-50);
      color: var(--p-text-color);
    }

    :host ::ng-deep .filter-row > th {
      top: 2.65rem;
      z-index: 4;
      background: var(--p-surface-50);
    }

    :host-context(.app-dark) ::ng-deep .registry-table .p-datatable-thead > tr > th,
    :host-context(.app-dark) ::ng-deep .workflow-table .p-datatable-thead > tr > th,
    :host-context(.app-dark) ::ng-deep .filter-row > th {
      background: var(--p-surface-800);
      color: var(--p-text-color);
    }

    :host ::ng-deep .p-paginator {
      flex-shrink: 0;
      border-top: 1px solid var(--p-content-border-color);
      background: var(--p-content-background);
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

    .search-field-wide {
      min-width: min(24rem, 100%);
    }

    .date-range {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .filter-input {
      height: 2rem;
      font-size: 0.8125rem;
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
      color: var(--p-text-color);
    }

    :host-context(.app-dark) .dialog-section {
      background: var(--p-surface-900);
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DocumenteWorkspaceComponent {
  protected readonly loading = signal(false);
  protected readonly registraturaFilterOpen = signal(false);
  protected readonly documentDialogOpen = signal(false);
  protected readonly batchDialogOpen = signal(false);
  protected readonly pdfExportDialogOpen = signal(false);
  protected readonly registrationMode = signal<'intrare' | 'iesire'>('intrare');
  protected readonly workflowPanelOpen = signal(false);
  protected readonly expandedRows = signal<Record<string, boolean>>({});
  protected readonly registryRows = signal<RegistryRow[]>([]);
  protected readonly workflowRows = signal<WorkflowRow[]>([]);
  protected readonly archiveSearchRows = signal<ArchiveSearchRow[]>([]);
  protected readonly archiveFileRows = signal<ArchiveFileRow[]>([]);
  protected readonly registryTotal = signal(0);
  protected readonly workflowTotal = signal(0);
  protected readonly archiveSearchTotal = signal(0);

  protected readonly tipOptions = [
    { label: 'Intrare', value: 'intrare' },
    { label: 'Ieșire', value: 'iesire' },
    { label: 'Multiplu', value: 'multiplu' },
  ];
  protected readonly registrationModeOptions = [
    { label: 'Intrare', value: 'intrare' },
    { label: 'Ieșire', value: 'iesire' },
  ];
  protected readonly compartimentOptions = [
    { label: 'Secretariat', value: 'secretariat' },
    { label: 'Direcțiune', value: 'directiune' },
    { label: 'Contabilitate', value: 'contabilitate' },
    { label: 'Resurse umane', value: 'resurse_umane' },
  ];
  protected readonly registruOptions = [
    { label: 'Registru Intrări', value: 'intrari' },
    { label: 'Registru Ieșiri', value: 'iesiri' },
    { label: 'Registru General', value: 'general' },
  ];
  protected readonly statusOptions = [
    { label: 'În lucru', value: 'in_lucru' },
    { label: 'Finalizat', value: 'finalizat' },
    { label: 'Anulat', value: 'anulat' },
  ];
  protected readonly workflowTabs = [
    { value: 'queue', label: 'Coada mea' },
    { value: 'signatures', label: 'Mapă semnături' },
    { value: 'pipeline', label: 'Evidență completă' },
  ];
  protected readonly workflowStats = [
    { status: 'draft', label: 'Draft', count: 0, severity: 'secondary' as const },
    { status: 'approval', label: 'Avizare', count: 0, severity: 'info' as const },
    { status: 'blocked', label: 'Blocate', count: 0, severity: 'danger' as const },
  ];
  protected readonly archiveCategoryOptions = [
    { label: 'Toate categoriile', value: '' },
    { label: 'Documente curente', value: 'current' },
    { label: 'Hotărâri CA/CP', value: 'decisions' },
  ];
  protected readonly archiveYearOptions = [
    { label: '2026', value: 2026 },
    { label: '2025', value: 2025 },
    { label: '2024', value: 2024 },
  ];
  protected readonly archiveTree: TreeNode[] = [
    { label: 'Fond Școală', expanded: true, children: [{ label: 'Registratură 2026' }, { label: 'Decizii CA/CP' }] },
    { label: 'Personal', children: [{ label: 'Dosare cadre didactice' }] },
  ];

  protected loadRegistry(_event: TableLazyLoadEvent): void {
    this.registryRows.set([]);
    this.registryTotal.set(0);
  }

  protected loadWorkflow(_event: TableLazyLoadEvent): void {
    this.workflowRows.set([]);
    this.workflowTotal.set(0);
  }

  protected openDocumentDialog(mode: 'intrare' | 'iesire'): void {
    this.registrationMode.set(mode);
    this.documentDialogOpen.set(true);
  }

  protected openBatchDialog(): void {
    this.batchDialogOpen.set(true);
  }

  protected openPdfExportDialog(): void {
    this.pdfExportDialogOpen.set(true);
  }
}
