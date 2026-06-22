import { ChangeDetectionStrategy, Component } from '@angular/core';
import { InputTextModule } from 'primeng/inputtext';
import { TableModule } from 'primeng/table';
import { TabsModule } from 'primeng/tabs';

@Component({
  selector: 'app-gdpr-workspace',
  imports: [InputTextModule, TableModule, TabsModule],
  template: `
    <section class="flex h-[calc(100dvh-6rem)] min-h-0 flex-col gap-3">
      <header class="rounded-3xl border border-surface-200 bg-white p-4 shadow-sm dark:border-surface-800 dark:bg-surface-900">
        <h1 class="m-0 text-2xl font-black tracking-[-0.035em]">GDPR</h1>
        <p class="m-0 mt-1 text-sm text-surface-500">Solicitari, politici de retentie, export si publicare controlata.</p>
      </header>
      <p-tabs value="requests" class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <p-tablist>
          <p-tab value="requests">Solicitari</p-tab>
          <p-tab value="retention">Retentie</p-tab>
          <p-tab value="exports">Exporturi</p-tab>
          <p-tab value="publication">Publicare</p-tab>
        </p-tablist>
        <p-tabpanels class="min-h-0 flex-1 overflow-hidden">
          @for (tab of ['requests','retention','exports','publication']; track tab) {
            <p-tabpanel [value]="tab" class="h-full min-h-0">
              <div class="h-full rounded-3xl border border-surface-200 bg-white shadow-sm dark:border-surface-800 dark:bg-surface-900">
                <p-table [value]="[]" [paginator]="true" [rows]="25" [scrollable]="true" scrollHeight="flex" styleClass="p-datatable-sm p-datatable-gridlines">
                  <ng-template pTemplate="header">
                    <tr><th>Cod</th><th>Subiect</th><th>Status</th><th>Termen</th><th>Actiuni</th></tr>
                    <tr><th><input pInputText class="w-full" /></th><th><input pInputText class="w-full" /></th><th></th><th></th><th></th></tr>
                  </ng-template>
                  <ng-template pTemplate="emptymessage">
                    <tr><td colspan="5" class="py-8 text-center text-surface-500">Nu exista inregistrari.</td></tr>
                  </ng-template>
                </p-table>
              </div>
            </p-tabpanel>
          }
        </p-tabpanels>
      </p-tabs>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GdprWorkspaceComponent {}
