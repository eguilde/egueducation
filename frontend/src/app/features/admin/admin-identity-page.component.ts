import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest } from 'rxjs';
import { switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatTabsModule } from '@angular/material/tabs';

import { AdminApiService } from '../../core/api/admin-api.service';
import {
  AdminOIDCClient,
  AdminOIDCConsentGrant,
  AdminOIDCSession,
  UpsertAdminOIDCClientRequest,
} from '../../core/api/api.types';
import { HasPermissionDirective } from '../../shared/authz/has-permission.directive';
import {
  ServerTableColumn,
  ServerTableComponent,
  ServerTableFilterState,
  ServerTableRowAction,
  ServerTableSortState,
} from '../../shared/server-table/server-table.component';

@Component({
  selector: 'app-admin-identity-page',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatPaginatorModule,
    MatSnackBarModule,
    MatTabsModule,
    HasPermissionDirective,
    ServerTableComponent,
  ],
  templateUrl: './admin-identity-page.component.html',
  styleUrl: './admin-identity-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminIdentityPageComponent {
  private readonly adminApi = inject(AdminApiService);
  private readonly fb = inject(FormBuilder);
  private readonly transloco = inject(TranslocoService);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly clientTableState = signal({
    page: 1,
    pageSize: 10,
    sort: 'client_id' as string | undefined,
    direction: 'asc' as 'asc' | 'desc' | undefined,
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly consentTableState = signal({
    page: 1,
    pageSize: 10,
    sort: 'granted_at' as string | undefined,
    direction: 'desc' as 'asc' | 'desc' | undefined,
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });
  protected readonly sessionTableState = signal({
    page: 1,
    pageSize: 10,
    sort: 'created_at' as string | undefined,
    direction: 'desc' as 'asc' | 'desc' | undefined,
    filters: {} as Record<string, string>,
    refreshToken: 0,
  });

  protected readonly selectedClientId = signal<string | null>(null);
  protected readonly selectedConsentId = signal<string | null>(null);
  protected readonly selectedSessionId = signal<string | null>(null);
  protected readonly activeRegister = signal<'clients' | 'consents' | 'sessions'>('clients');
  protected readonly activePanel = signal<'client-create' | 'client-details' | 'consent-details' | 'session-details'>(
    'client-create',
  );

  protected readonly clientForm = this.fb.group({
    client_id: this.fb.nonNullable.control('', [Validators.required]),
    client_name: this.fb.nonNullable.control('', [Validators.required]),
    public_client: this.fb.nonNullable.control(true),
    require_pkce: this.fb.nonNullable.control(true),
    active: this.fb.nonNullable.control(true),
    redirect_uris_text: this.fb.nonNullable.control('', [Validators.required]),
  });

  protected readonly clientsResponse = toSignal(
    combineLatest([toObservable(this.clientTableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.oidcClients({
          page: state.page,
          pageSize: state.pageSize,
          sort: state.sort,
          direction: state.direction,
          filters: state.filters,
        }),
      ),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly consentsResponse = toSignal(
    combineLatest([toObservable(this.consentTableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.oidcConsents({
          page: state.page,
          pageSize: state.pageSize,
          sort: state.sort,
          direction: state.direction,
          filters: state.filters,
        }),
      ),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly sessionsResponse = toSignal(
    combineLatest([toObservable(this.sessionTableState), this.transloco.langChanges$]).pipe(
      switchMap(([state]) =>
        this.adminApi.oidcSessions({
          page: state.page,
          pageSize: state.pageSize,
          sort: state.sort,
          direction: state.direction,
          filters: state.filters,
        }),
      ),
    ),
    { initialValue: { items: [], total: 0, page: 1, pageSize: 10 } },
  );

  protected readonly clientRows = computed(() => this.clientsResponse().items);
  protected readonly consentRows = computed(() => this.consentsResponse().items);
  protected readonly sessionRows = computed(() => this.sessionsResponse().items);
  protected readonly selectedClient = computed(
    () => this.clientRows().find((row) => row.client_id === this.selectedClientId()) ?? null,
  );
  protected readonly selectedConsent = computed(
    () => this.consentRows().find((row) => row.id === this.selectedConsentId()) ?? null,
  );
  protected readonly selectedSession = computed(
    () => this.sessionRows().find((row) => row.token_id === this.selectedSessionId()) ?? null,
  );
  protected readonly clientRowActions = computed<ServerTableRowAction<AdminOIDCClient>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);
  protected readonly consentRowActions = computed<ServerTableRowAction<AdminOIDCConsentGrant>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);
  protected readonly sessionRowActions = computed<ServerTableRowAction<AdminOIDCSession>[]>(() => [
    { key: 'open', icon: 'open_in_new', label: this.transloco.translate('common.open') },
  ]);

  protected readonly clientColumns = computed<ServerTableColumn<AdminOIDCClient>[]>(() => [
    {
      key: 'client_id',
      label: this.transloco.translate('admin.identity.clients.columns.clientId'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'client_name',
      label: this.transloco.translate('admin.identity.clients.columns.clientName'),
      sortable: true,
    },
    {
      key: 'redirect_uris',
      label: this.transloco.translate('admin.identity.clients.columns.redirectUris'),
      formatter: (row) => row.redirect_uris.length.toString(),
    },
    {
      key: 'active',
      label: this.transloco.translate('admin.identity.clients.columns.active'),
      sortable: true,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.identity.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.identity.boolean.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`admin.identity.boolean.${row.active ? 'yes' : 'no'}`),
    },
    {
      key: 'created_at',
      label: this.transloco.translate('admin.identity.clients.columns.createdAt'),
      sortable: true,
      formatter: (row) => this.formatDate(row.created_at),
    },
  ]);

  protected readonly consentColumns = computed<ServerTableColumn<AdminOIDCConsentGrant>[]>(() => [
    {
      key: 'client_id',
      label: this.transloco.translate('admin.identity.consents.columns.clientId'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'subject_name',
      label: this.transloco.translate('admin.identity.consents.columns.subject'),
      sortable: false,
      filterKey: 'subject',
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
      formatter: (row) => row.subject_name || row.subject_email || row.subject,
    },
    {
      key: 'scope',
      label: this.transloco.translate('admin.identity.consents.columns.scope'),
      formatter: (row) => row.scope,
    },
    {
      key: 'granted_at',
      label: this.transloco.translate('admin.identity.consents.columns.grantedAt'),
      sortable: true,
      formatter: (row) => this.formatDate(row.granted_at),
    },
  ]);

  protected readonly sessionColumns = computed<ServerTableColumn<AdminOIDCSession>[]>(() => [
    {
      key: 'client_id',
      label: this.transloco.translate('admin.identity.sessions.columns.clientId'),
      sortable: true,
      sticky: true,
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
    },
    {
      key: 'subject_name',
      label: this.transloco.translate('admin.identity.sessions.columns.subject'),
      sortable: false,
      filterKey: 'subject',
      filter: { type: 'text', placeholder: this.transloco.translate('table.filters.contains') },
      formatter: (row) => row.subject_name || row.subject_email || row.subject,
    },
    {
      key: 'scope',
      label: this.transloco.translate('admin.identity.sessions.columns.scope'),
      formatter: (row) => row.scope,
    },
    {
      key: 'expires_at',
      label: this.transloco.translate('admin.identity.sessions.columns.expiresAt'),
      sortable: true,
      formatter: (row) => this.formatDate(row.expires_at),
    },
    {
      key: 'revoked',
      label: this.transloco.translate('admin.identity.sessions.columns.revoked'),
      sortable: false,
      filter: {
        type: 'select',
        placeholder: this.transloco.translate('table.filters.any'),
        options: [
          { value: 'true', label: this.transloco.translate('admin.identity.boolean.yes') },
          { value: 'false', label: this.transloco.translate('admin.identity.boolean.no') },
        ],
      },
      formatter: (row) => this.transloco.translate(`admin.identity.boolean.${row.revoked ? 'yes' : 'no'}`),
    },
  ]);

  protected selectClient(item: AdminOIDCClient): void {
    this.selectedClientId.set(item.client_id);
    this.activeRegister.set('clients');
    this.activePanel.set('client-details');
    this.clientForm.reset({
      client_id: item.client_id,
      client_name: item.client_name,
      public_client: item.public_client,
      require_pkce: item.require_pkce,
      active: item.active,
      redirect_uris_text: item.redirect_uris.join('\n'),
    });
  }

  protected resetClientForm(): void {
    this.selectedClientId.set(null);
    this.activePanel.set('client-create');
    this.clientForm.reset({
      client_id: '',
      client_name: '',
      public_client: true,
      require_pkce: true,
      active: true,
      redirect_uris_text: '',
    });
  }

  protected saveClient(): void {
    if (this.clientForm.invalid) {
      this.clientForm.markAllAsTouched();
      return;
    }
    const raw = this.clientForm.getRawValue();
    const payload: UpsertAdminOIDCClientRequest = {
      client_id: raw.client_id.trim(),
      client_name: raw.client_name.trim(),
      public_client: raw.public_client,
      require_pkce: raw.require_pkce,
      active: raw.active,
      redirect_uris: raw.redirect_uris_text
        .split(/\r?\n|,/)
        .map((value) => value.trim())
        .filter((value) => value.length > 0),
    };
    this.adminApi.saveOidcClient(payload).subscribe({
      next: (saved) => {
        this.selectedClientId.set(saved.client_id);
        this.activePanel.set('client-details');
        this.snackBar.open(
          this.transloco.translate('admin.identity.messages.clientSaved'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.clientTableState.update((state) => ({ ...state, page: 1, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.identity.messages.clientSaveFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected selectConsent(item: AdminOIDCConsentGrant): void {
    this.activeRegister.set('consents');
    this.activePanel.set('consent-details');
    this.selectedConsentId.set(item.id);
  }

  protected revokeSelectedConsent(): void {
    const id = this.selectedConsentId();
    if (!id) {
      return;
    }
    this.adminApi.revokeOidcConsent({ id }).subscribe({
      next: () => {
        this.selectedConsentId.set(null);
        this.snackBar.open(
          this.transloco.translate('admin.identity.messages.consentRevoked'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.consentTableState.update((state) => ({ ...state, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.identity.messages.consentRevokeFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected selectSession(item: AdminOIDCSession): void {
    this.activeRegister.set('sessions');
    this.activePanel.set('session-details');
    this.selectedSessionId.set(item.token_id);
  }

  protected openClientCreatePanel(): void {
    this.activeRegister.set('clients');
    this.activePanel.set('client-create');
  }

  protected onClientActionClick(event: { action: string; row: AdminOIDCClient }): void {
    if (event.action === 'open') {
      this.selectClient(event.row);
    }
  }

  protected onConsentActionClick(event: { action: string; row: AdminOIDCConsentGrant }): void {
    if (event.action === 'open') {
      this.selectConsent(event.row);
    }
  }

  protected onSessionActionClick(event: { action: string; row: AdminOIDCSession }): void {
    if (event.action === 'open') {
      this.selectSession(event.row);
    }
  }

  protected revokeSelectedSession(): void {
    const tokenId = this.selectedSessionId();
    if (!tokenId) {
      return;
    }
    this.adminApi.revokeOidcSession({ token_id: tokenId }).subscribe({
      next: () => {
        this.selectedSessionId.set(null);
        this.snackBar.open(
          this.transloco.translate('admin.identity.messages.sessionRevoked'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.sessionTableState.update((state) => ({ ...state, refreshToken: state.refreshToken + 1 }));
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('admin.identity.messages.sessionRevokeFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }

  protected onClientPageChange(event: PageEvent): void {
    this.clientTableState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onConsentPageChange(event: PageEvent): void {
    this.consentTableState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onSessionPageChange(event: PageEvent): void {
    this.sessionTableState.update((state) => ({ ...state, page: event.pageIndex + 1, pageSize: event.pageSize }));
  }

  protected onClientFilterChange(filters: ServerTableFilterState): void {
    this.clientTableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onConsentFilterChange(filters: ServerTableFilterState): void {
    this.consentTableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onSessionFilterChange(filters: ServerTableFilterState): void {
    this.sessionTableState.update((state) => ({ ...state, page: 1, filters }));
  }

  protected onClientSortChange(sort: ServerTableSortState): void {
    this.clientTableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: (sort.direction as 'asc' | 'desc') || undefined,
    }));
  }

  protected onConsentSortChange(sort: ServerTableSortState): void {
    this.consentTableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: (sort.direction as 'asc' | 'desc') || undefined,
    }));
  }

  protected onSessionSortChange(sort: ServerTableSortState): void {
    this.sessionTableState.update((state) => ({
      ...state,
      page: 1,
      sort: sort.active || undefined,
      direction: (sort.direction as 'asc' | 'desc') || undefined,
    }));
  }

  private formatDate(value: string): string {
    if (!value) {
      return '—';
    }
    return new Intl.DateTimeFormat(this.transloco.getActiveLang(), {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date(value));
  }
}
