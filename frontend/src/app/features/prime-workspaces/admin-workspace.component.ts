import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { TableLazyLoadEvent, TableModule } from 'primeng/table';
import { TabsModule } from 'primeng/tabs';
import { TagModule } from 'primeng/tag';
import { TooltipModule } from 'primeng/tooltip';

import { AdminApiService } from '../../core/api/admin-api.service';
import {
  AdminPositionRoleAssignment,
  AdminPositionRoleAssignmentFilters,
  AdminRole,
  AdminRolePermissionAssignment,
  AdminRolePermissionAssignmentFilters,
  AdminUser,
  AdminUserRoleAssignment,
  PagedResponse,
  TableQuery,
  UpsertAdminPositionRoleAssignmentRequest,
  UpsertAdminRolePermissionAssignmentRequest,
  UpsertAdminRoleRequest,
  UpsertAdminUserRequest,
  UpsertAdminUserRoleAssignmentRequest,
} from '../../core/api/api.types';
import { AuthzService } from '../../core/authz/authz.service';
import { FEATURE_ACCESS_RULES } from '../../core/authz/role-catalog';

interface AdminTab {
  value: string;
  label: string;
  icon: string;
  status: 'wired' | 'contract-missing';
  description: string;
}

interface AdminRoleFilterState {
  code: string;
  label: string;
}

interface AdminUserRoleAssignmentFilterState {
  user_name: string;
  role_code: string;
}

interface AdminRolePermissionAssignmentFilterState {
  role_code: string;
  permission_code: string;
}

interface AdminPositionRoleAssignmentFilterState {
  position_code: string;
  role_code: string;
}

@Component({
  selector: 'app-admin-workspace',
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    ButtonModule,
    DialogModule,
    InputTextModule,
    SelectModule,
    TableModule,
    TabsModule,
    TagModule,
    TooltipModule,
  ],
  template: `
    <section class="admin-workspace flex h-[calc(100dvh-6rem)] min-h-0 flex-col overflow-hidden">
      <p-tabs value="users" class="flex min-h-0 flex-1 flex-col overflow-hidden">
        <p-tablist class="shrink-0">
          @for (tab of tabs; track tab.value) {
            <p-tab [value]="tab.value">
              <span class="inline-flex items-center gap-2">
                <i [class]="tab.icon"></i>
                {{ tab.label }}
              </span>
            </p-tab>
          }
        </p-tablist>

        <p-tabpanels class="min-h-0 flex-1 overflow-hidden p-0">
          <p-tabpanel value="users" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col gap-3 p-3">
              <div class="flex shrink-0 items-center justify-between gap-3">
                <div>
                  <h2 class="m-0 text-xl font-semibold">Gestionare Utilizatori</h2>
                  <p class="m-0 mt-1 text-sm text-muted-color">Tabel server-side real pentru utilizatori, verificări, roluri și stare.</p>
                </div>
                <p-button label="Adaugă utilizator" icon="pi pi-user-plus" (onClick)="openUserDialog()" />
              </div>

              <div class="min-h-0 flex-1 overflow-hidden rounded-2xl border border-surface bg-surface-0 dark:bg-surface-900">
                <p-table
                  [value]="users()"
                  [loading]="usersLoading()"
                  [lazy]="true"
                  [paginator]="true"
                  [rows]="usersQuery().pageSize"
                  [first]="(usersQuery().page - 1) * usersQuery().pageSize"
                  [totalRecords]="usersTotal()"
                  [rowsPerPageOptions]="[10,25,50,100]"
                  [showCurrentPageReport]="true"
                  currentPageReportTemplate="Afișare {first} - {last} din {totalRecords} utilizatori"
                  [scrollable]="true"
                  scrollHeight="flex"
                  styleClass="p-datatable-sm p-datatable-gridlines admin-table"
                  (onLazyLoad)="loadUsers($event)"
                >
                  <ng-template pTemplate="header">
                    <tr>
                      <th pSortableColumn="name" style="min-width: 16rem">Utilizator <p-sortIcon field="name" /></th>
                      <th pSortableColumn="email" style="min-width: 16rem">Email <p-sortIcon field="email" /></th>
                      <th style="min-width: 10rem">Telefon</th>
                      <th style="min-width: 10rem">Poziție</th>
                      <th pSortableColumn="locale" style="width: 8rem">Limbă <p-sortIcon field="locale" /></th>
                      <th pSortableColumn="status" style="width: 9rem">Stare <p-sortIcon field="status" /></th>
                      <th style="width: 9rem" class="text-center">Acțiuni</th>
                    </tr>
                    <tr>
                      <th><input pInputText class="w-full" placeholder="Caută utilizator" [(ngModel)]="userFilters['name']" (keyup.enter)="reloadUsers()" /></th>
                      <th><input pInputText class="w-full" placeholder="Caută email" [(ngModel)]="userFilters['email']" (keyup.enter)="reloadUsers()" /></th>
                      <th><input pInputText class="w-full" placeholder="Caută telefon" [(ngModel)]="userFilters['phone']" (keyup.enter)="reloadUsers()" /></th>
                      <th><input pInputText class="w-full" placeholder="Poziție" [(ngModel)]="userFilters['position']" (keyup.enter)="reloadUsers()" /></th>
                      <th></th>
                      <th></th>
                      <th></th>
                    </tr>
                  </ng-template>
                  <ng-template pTemplate="body" let-user>
                    <tr>
                      <td>
                        <div class="flex items-center gap-3">
                          <div class="grid size-10 place-items-center rounded-xl bg-primary-100 font-bold text-primary-700">
                            {{ initials(user.name || user.email) }}
                          </div>
                          <div class="min-w-0">
                            <div class="truncate font-semibold">{{ user.name || user.email }}</div>
                            <div class="truncate text-xs text-muted-color">{{ user.sub }}</div>
                          </div>
                        </div>
                      </td>
                      <td>
                        <div>{{ user.email }}</div>
                        <span class="text-xs" [class.text-green-600]="user.email_verified" [class.text-orange-600]="!user.email_verified">
                          <i [class]="user.email_verified ? 'pi pi-check-circle' : 'pi pi-exclamation-circle'"></i>
                          {{ user.email_verified ? 'Verificat' : 'Neverificat' }}
                        </span>
                      </td>
                      <td>
                        <div>{{ user.phone || '-' }}</div>
                        <span class="text-xs" [class.text-green-600]="user.phone_verified" [class.text-orange-600]="!user.phone_verified">
                          <i [class]="user.phone_verified ? 'pi pi-check-circle' : 'pi pi-exclamation-circle'"></i>
                          {{ user.phone_verified ? 'Verificat' : 'Neverificat' }}
                        </span>
                      </td>
                      <td>{{ user.position || '-' }}</td>
                      <td><p-tag [value]="user.locale || 'ro'" severity="secondary" /></td>
                      <td><p-tag [value]="user.status" [severity]="user.status === 'active' ? 'success' : 'warn'" /></td>
                      <td class="text-center">
                        <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" pTooltip="Editează utilizator" (onClick)="openUserDialog(user)" />
                      </td>
                    </tr>
                  </ng-template>
                  <ng-template pTemplate="emptymessage">
                    <tr>
                      <td colspan="7" class="py-8 text-center text-muted-color">Nu există utilizatori pentru filtrele curente.</td>
                    </tr>
                  </ng-template>
                </p-table>
              </div>
            </div>
          </p-tabpanel>

          <p-tabpanel value="rbac" class="flex min-h-0 flex-1 overflow-hidden p-0">
            <div class="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-3">
              <div class="grid gap-4 xl:grid-cols-2">
                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <div>
                      <h2 class="m-0 text-lg font-semibold">Catalog roluri</h2>
                      <p class="m-0 mt-1 text-sm text-muted-color">Roluri de bază, extinse ulterior cu altele.</p>
                    </div>
                    <p-button label="Rol nou" icon="pi pi-plus" (onClick)="openRoleDialog()" />
                  </div>
                  <div class="mb-3 grid gap-2 md:grid-cols-2">
                    <input pInputText class="w-full" placeholder="Cod rol" [(ngModel)]="roleFilters.code" (keyup.enter)="reloadRoles()" />
                    <input pInputText class="w-full" placeholder="Etichetă rol" [(ngModel)]="roleFilters.label" (keyup.enter)="reloadRoles()" />
                  </div>
                  <p-table
                    [value]="roles()"
                    [loading]="rolesLoading()"
                    [lazy]="true"
                    [paginator]="true"
                    [rows]="rolesQuery().pageSize"
                    [first]="(rolesQuery().page - 1) * rolesQuery().pageSize"
                    [totalRecords]="rolesTotal()"
                    [rowsPerPageOptions]="[25,50,100]"
                    [showCurrentPageReport]="true"
                    currentPageReportTemplate="Afișare {first} - {last} din {totalRecords} roluri"
                    [scrollable]="true"
                    scrollHeight="flex"
                    styleClass="p-datatable-sm p-datatable-gridlines"
                    (onLazyLoad)="loadRoles($event)"
                  >
                    <ng-template pTemplate="header">
                      <tr>
                        <th>Cod</th>
                        <th>Etichetă</th>
                        <th class="text-center">Acțiuni</th>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="body" let-role>
                      <tr>
                        <td class="font-medium">{{ role.code }}</td>
                        <td>{{ role.label }}</td>
                        <td class="text-center">
                          <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" (onClick)="openRoleDialog(role)" />
                        </td>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="emptymessage">
                      <tr><td colspan="3" class="py-8 text-center text-muted-color">Nu există roluri pentru filtrele curente.</td></tr>
                    </ng-template>
                  </p-table>
                </section>

                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <div>
                      <h2 class="m-0 text-lg font-semibold">Roluri pe utilizator</h2>
                      <p class="m-0 mt-1 text-sm text-muted-color">Atribuire role pentru conturile autentificate.</p>
                    </div>
                    <p-button label="Atribuire" icon="pi pi-plus" (onClick)="openUserRoleDialog()" />
                  </div>
                  <div class="mb-3 grid gap-2 md:grid-cols-2">
                    <input pInputText class="w-full" placeholder="Utilizator" [(ngModel)]="userRoleAssignmentFilters.user_name" (keyup.enter)="reloadUserRoleAssignments()" />
                    <p-select
                      appendTo="body"
                      class="w-full"
                      [options]="roles()"
                      optionLabel="label"
                      optionValue="code"
                      placeholder="Rol"
                      [(ngModel)]="userRoleAssignmentFilters.role_code"
                      (onChange)="reloadUserRoleAssignments()"
                    />
                  </div>
                  <p-table
                    [value]="userRoleAssignments()"
                    [loading]="userRoleAssignmentsLoading()"
                    [lazy]="true"
                    [paginator]="true"
                    [rows]="userRoleAssignmentsQuery().pageSize"
                    [first]="(userRoleAssignmentsQuery().page - 1) * userRoleAssignmentsQuery().pageSize"
                    [totalRecords]="userRoleAssignmentsTotal()"
                    [rowsPerPageOptions]="[25,50,100]"
                    [showCurrentPageReport]="true"
                    currentPageReportTemplate="Afișare {first} - {last} din {totalRecords} atribuiri"
                    [scrollable]="true"
                    scrollHeight="flex"
                    styleClass="p-datatable-sm p-datatable-gridlines"
                    (onLazyLoad)="loadUserRoleAssignments($event)"
                  >
                    <ng-template pTemplate="header">
                      <tr>
                        <th>Utilizator</th>
                        <th>Rol</th>
                        <th class="text-center">Acțiuni</th>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="body" let-assignment>
                      <tr>
                        <td>
                          <div class="font-medium">{{ assignment.user_name }}</div>
                          <div class="text-xs text-muted-color">{{ assignment.user_email }}</div>
                        </td>
                        <td><p-tag [value]="assignment.role_label" severity="secondary" /></td>
                        <td class="text-center">
                          <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" (onClick)="openUserRoleDialog(assignment)" />
                          <p-button icon="pi pi-trash" severity="danger" [rounded]="true" [text]="true" (onClick)="removeUserRoleAssignment(assignment)" />
                        </td>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="emptymessage">
                      <tr><td colspan="3" class="py-8 text-center text-muted-color">Nu există atribuiri pentru filtrele curente.</td></tr>
                    </ng-template>
                  </p-table>
                </section>

                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900 xl:col-span-2">
                  <div class="mb-3">
                    <h2 class="m-0 text-lg font-semibold">Matrice roluri și funcționalități</h2>
                    <p class="m-0 mt-1 text-sm text-muted-color">Maparea explicită care guvernează accesul la modulele principale.</p>
                  </div>
                  <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                    @for (rule of featureAccessRules; track rule.feature) {
                      <article class="rounded-2xl border border-surface-200 p-4">
                        <div class="flex items-start justify-between gap-3">
                          <div>
                            <div class="font-semibold">{{ rule.feature }}</div>
                            <div class="text-xs text-muted-color">{{ rule.description }}</div>
                          </div>
                          <p-tag [value]="rule.permissions.length ? 'Permisiuni' : 'Roluri'" severity="secondary" />
                        </div>
                        <div class="mt-3 flex flex-wrap gap-2">
                          @for (role of rule.roles; track role) {
                            <p-tag [value]="authz.roleLabel(role)" severity="info" [pTooltip]="role" />
                          }
                        </div>
                      </article>
                    }
                  </div>
                </section>

                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900 xl:col-span-2">
                  <div class="mb-3">
                    <h2 class="m-0 text-lg font-semibold">Catalog roluri și permisiuni</h2>
                    <p class="m-0 mt-1 text-sm text-muted-color">Sursa de adevăr servită din backend pentru label-uri și permisiuni.</p>
                  </div>
                  <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                    @for (role of authz.roleCatalog(); track role.code) {
                      <article class="rounded-2xl border border-surface-200 p-4">
                        <div class="flex items-start justify-between gap-3">
                          <div>
                            <div class="font-semibold">{{ role.label }}</div>
                            <div class="text-xs text-muted-color">{{ role.code }}</div>
                          </div>
                          <p-tag [value]="role.permissions?.length?.toString() ?? '0'" severity="info" />
                        </div>
                        <div class="mt-3 grid gap-3">
                          <div>
                            <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-color">Poziții</div>
                            <div class="flex flex-wrap gap-2">
                              @for (position of role.positions ?? []; track position) {
                                <p-tag [value]="position" severity="success" />
                              } @empty {
                                <span class="text-xs text-muted-color">Nu are poziții asociate.</span>
                              }
                            </div>
                          </div>
                          <div>
                            <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-muted-color">Permisiuni</div>
                            <div class="flex flex-wrap gap-2">
                              @for (permission of role.permissions ?? []; track permission) {
                                <p-tag [value]="permission" severity="secondary" />
                              } @empty {
                                <span class="text-xs text-muted-color">Nu are permisiuni explicite.</span>
                              }
                            </div>
                          </div>
                        </div>
                      </article>
                    }
                  </div>
                </section>

                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900 xl:col-span-2">
                  <div class="mb-3">
                    <h2 class="m-0 text-lg font-semibold">Mapare poziții și roluri</h2>
                    <p class="m-0 mt-1 text-sm text-muted-color">Expusă direct din backend pentru a păstra paritatea dintre pozițiile operaționale și RBAC.</p>
                  </div>
                  <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                    @for (mapping of authz.rolePositions(); track mapping.position_code + ':' + mapping.role_code) {
                      <article class="rounded-2xl border border-surface-200 p-4">
                        <div class="flex items-start justify-between gap-3">
                          <div>
                            <div class="font-semibold">{{ mapping.position_name }}</div>
                            <div class="text-xs text-muted-color">{{ mapping.position_code }}</div>
                          </div>
                          <p-tag [value]="mapping.role_label" severity="info" [pTooltip]="mapping.role_code" />
                        </div>
                      </article>
                    } @empty {
                      <div class="text-sm text-muted-color">Nu există mapări poziție-rol în catalogul backend.</div>
                    }
                  </div>
                </section>

                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <div>
                      <h2 class="m-0 text-lg font-semibold">Roluri pe permisiune</h2>
                      <p class="m-0 mt-1 text-sm text-muted-color">Mapare explicită între roluri și permisiuni.</p>
                    </div>
                    <p-button label="Mapare nouă" icon="pi pi-plus" (onClick)="openRolePermissionDialog()" />
                  </div>
                  <div class="mb-3 grid gap-2 md:grid-cols-2">
                    <p-select
                      appendTo="body"
                      class="w-full"
                      [options]="rolePermissionFilterOptions()?.roles ?? []"
                      optionLabel="label"
                      optionValue="code"
                      placeholder="Rol"
                      [(ngModel)]="rolePermissionAssignmentFilters.role_code"
                      (onChange)="reloadRolePermissionAssignments()"
                    />
                    <p-select
                      appendTo="body"
                      class="w-full"
                      [options]="rolePermissionFilterOptions()?.permissions ?? []"
                      optionLabel="label"
                      optionValue="code"
                      placeholder="Permisiune"
                      [(ngModel)]="rolePermissionAssignmentFilters.permission_code"
                      (onChange)="reloadRolePermissionAssignments()"
                    />
                  </div>
                  <p-table
                    [value]="rolePermissionAssignments()"
                    [loading]="rolePermissionAssignmentsLoading()"
                    [lazy]="true"
                    [paginator]="true"
                    [rows]="rolePermissionAssignmentsQuery().pageSize"
                    [first]="(rolePermissionAssignmentsQuery().page - 1) * rolePermissionAssignmentsQuery().pageSize"
                    [totalRecords]="rolePermissionAssignmentsTotal()"
                    [rowsPerPageOptions]="[25,50,100]"
                    [showCurrentPageReport]="true"
                    currentPageReportTemplate="Afișare {first} - {last} din {totalRecords} mapări"
                    [scrollable]="true"
                    scrollHeight="flex"
                    styleClass="p-datatable-sm p-datatable-gridlines"
                    (onLazyLoad)="loadRolePermissionAssignments($event)"
                  >
                    <ng-template pTemplate="header">
                      <tr>
                        <th>Rol</th>
                        <th>Permisiune</th>
                        <th class="text-center">Acțiuni</th>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="body" let-assignment>
                      <tr>
                        <td>
                          <div class="font-medium">{{ assignment.role_label }}</div>
                          <div class="text-xs text-muted-color">{{ assignment.role_label }}</div>
                        </td>
                        <td>
                          <div class="font-medium">{{ assignment.permission_code }}</div>
                          <div class="text-xs text-muted-color">{{ assignment.permission_label }}</div>
                        </td>
                        <td class="text-center">
                          <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" (onClick)="openRolePermissionDialog(assignment)" />
                          <p-button icon="pi pi-trash" severity="danger" [rounded]="true" [text]="true" (onClick)="removeRolePermissionAssignment(assignment)" />
                        </td>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="emptymessage">
                      <tr><td colspan="3" class="py-8 text-center text-muted-color">Nu există mapări pentru filtrele curente.</td></tr>
                    </ng-template>
                  </p-table>
                </section>

                <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm dark:bg-surface-900">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <div>
                      <h2 class="m-0 text-lg font-semibold">Roluri pe poziție</h2>
                      <p class="m-0 mt-1 text-sm text-muted-color">Mapare poziții operaționale către rolurile implicite.</p>
                    </div>
                    <p-button label="Mapare nouă" icon="pi pi-plus" (onClick)="openPositionRoleDialog()" />
                  </div>
                  <div class="mb-3 grid gap-2 md:grid-cols-2">
                    <p-select
                      appendTo="body"
                      class="w-full"
                      [options]="positionRoleFilterOptions()?.positions ?? []"
                      optionLabel="name"
                      optionValue="code"
                      placeholder="Poziție"
                      [(ngModel)]="positionRoleAssignmentFilters.position_code"
                      (onChange)="reloadPositionRoleAssignments()"
                    />
                    <p-select
                      appendTo="body"
                      class="w-full"
                      [options]="positionRoleFilterOptions()?.roles ?? []"
                      optionLabel="label"
                      optionValue="code"
                      placeholder="Rol"
                      [(ngModel)]="positionRoleAssignmentFilters.role_code"
                      (onChange)="reloadPositionRoleAssignments()"
                    />
                  </div>
                  <p-table
                    [value]="positionRoleAssignments()"
                    [loading]="positionRoleAssignmentsLoading()"
                    [lazy]="true"
                    [paginator]="true"
                    [rows]="positionRoleAssignmentsQuery().pageSize"
                    [first]="(positionRoleAssignmentsQuery().page - 1) * positionRoleAssignmentsQuery().pageSize"
                    [totalRecords]="positionRoleAssignmentsTotal()"
                    [rowsPerPageOptions]="[25,50,100]"
                    [showCurrentPageReport]="true"
                    currentPageReportTemplate="Afișare {first} - {last} din {totalRecords} mapări"
                    [scrollable]="true"
                    scrollHeight="flex"
                    styleClass="p-datatable-sm p-datatable-gridlines"
                    (onLazyLoad)="loadPositionRoleAssignments($event)"
                  >
                    <ng-template pTemplate="header">
                      <tr>
                        <th>Poziție</th>
                        <th>Rol</th>
                        <th class="text-center">Acțiuni</th>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="body" let-assignment>
                      <tr>
                        <td>
                          <div class="font-medium">{{ assignment.position_code }}</div>
                          <div class="text-xs text-muted-color">{{ assignment.position_name }}</div>
                        </td>
                        <td>
                          <div class="font-medium">{{ assignment.role_label }}</div>
                          <div class="text-xs text-muted-color">{{ assignment.role_label }}</div>
                        </td>
                        <td class="text-center">
                          <p-button icon="pi pi-pencil" [rounded]="true" [text]="true" (onClick)="openPositionRoleDialog(assignment)" />
                          <p-button icon="pi pi-trash" severity="danger" [rounded]="true" [text]="true" (onClick)="removePositionRoleAssignment(assignment)" />
                        </td>
                      </tr>
                    </ng-template>
                    <ng-template pTemplate="emptymessage">
                      <tr><td colspan="3" class="py-8 text-center text-muted-color">Nu există mapări pentru filtrele curente.</td></tr>
                    </ng-template>
                  </p-table>
                </section>
              </div>
            </div>
          </p-tabpanel>

          @for (tab of missingContractTabs; track tab.value) {
            <p-tabpanel [value]="tab.value" class="h-full min-h-0 p-3">
              <section class="grid h-full place-items-center rounded-2xl border border-dashed border-surface bg-surface-0 p-8 text-center dark:bg-surface-900">
                <div class="max-w-xl">
                  <i [class]="tab.icon + ' text-4xl text-primary'"></i>
                  <h2 class="mb-2 mt-4 text-xl font-bold">{{ tab.label }}</h2>
                  <p class="text-muted-color">{{ tab.description }}</p>
                  <p-tag value="Backend contract missing" severity="warn" />
                  <p class="mt-4 text-sm text-muted-color">
                    Acest tab este păstrat ca obligație de paritate Costești. Următorul pas este să adăugăm endpointurile și tabelele reale, nu să îl ascundem.
                  </p>
                </div>
              </section>
            </p-tabpanel>
          }

          <p-tabpanel value="profile" class="h-full min-h-0 p-3">
            <section class="grid h-full place-items-center rounded-2xl border border-surface bg-surface-0 p-8 text-center dark:bg-surface-900">
              <div>
                <i class="pi pi-user text-4xl text-primary"></i>
                <h2 class="mb-2 mt-4 text-xl font-bold">Profil utilizator</h2>
                <p class="text-muted-color">Profilul complet este disponibil și din blocul de utilizator din drawer.</p>
                <p-button label="Deschide profilul" icon="pi pi-user" routerLink="/profile" />
              </div>
            </section>
          </p-tabpanel>
        </p-tabpanels>
      </p-tabs>

      <p-dialog
        [visible]="userDialogOpen()"
        (visibleChange)="userDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editingUser()?.id ? 'Editează utilizator' : 'Adaugă utilizator'"
        [style]="{ width: 'min(42rem, 94vw)' }"
      >
        <div class="grid gap-4 md:grid-cols-2">
          <label class="admin-field md:col-span-2">
            <span>Nume</span>
            <input pInputText [(ngModel)]="userForm.name" />
          </label>
          <label class="admin-field">
            <span>Email</span>
            <input pInputText [(ngModel)]="userForm.email" />
          </label>
          <label class="admin-field">
            <span>Telefon</span>
            <input pInputText [(ngModel)]="userForm.phone" />
          </label>
          <label class="admin-field">
            <span>Limbă</span>
            <p-select appendTo="body" [options]="localeOptions" [(ngModel)]="userForm.locale" />
          </label>
          <label class="admin-field">
            <span>Status</span>
            <p-select appendTo="body" [options]="statusOptions" [(ngModel)]="userForm.status" />
          </label>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="userDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" (onClick)="saveUser()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="roleDialogOpen()"
        (visibleChange)="roleDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editingRole()?.code ? 'Editează rol' : 'Adaugă rol'"
        [style]="{ width: 'min(32rem, 94vw)' }"
      >
        <div class="grid gap-4">
          <label class="admin-field">
            <span>Cod</span>
            <input pInputText [(ngModel)]="roleForm.code" [disabled]="!!editingRole()?.code" />
          </label>
          <label class="admin-field">
            <span>Etichetă</span>
            <input pInputText [(ngModel)]="roleForm.label" />
          </label>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="roleDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" (onClick)="saveRole()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="userRoleDialogOpen()"
        (visibleChange)="userRoleDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editingUserRoleAssignment() ? 'Editează atribuirea rolului' : 'Atribuire rol utilizator'"
        [style]="{ width: 'min(38rem, 94vw)' }"
      >
        <div class="grid gap-4 md:grid-cols-2">
          <label class="admin-field md:col-span-2">
            <span>Utilizator</span>
            <p-select
              appendTo="body"
              [options]="userLookup()"
              optionLabel="email"
              optionValue="id"
              [(ngModel)]="userRoleForm.user_id"
              placeholder="Alege utilizatorul"
            />
          </label>
          <label class="admin-field">
            <span>Rol</span>
            <p-select
              appendTo="body"
              [options]="roles()"
              optionLabel="label"
              optionValue="code"
              [(ngModel)]="userRoleForm.role_code"
              placeholder="Alege rolul"
            />
          </label>
          <label class="admin-field">
            <span>Stare</span>
            <p-select appendTo="body" [options]="[{ label: 'Atribuit', value: true }, { label: 'Revocat', value: false }]" [(ngModel)]="userRoleForm.assigned" />
          </label>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="userRoleDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" (onClick)="saveUserRoleAssignment()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="rolePermissionDialogOpen()"
        (visibleChange)="rolePermissionDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editingRolePermissionAssignment() ? 'Editează maparea rol-permisiune' : 'Mapare rol-permisiune'"
        [style]="{ width: 'min(38rem, 94vw)' }"
      >
        <div class="grid gap-4 md:grid-cols-2">
          <label class="admin-field">
            <span>Rol</span>
            <p-select
              appendTo="body"
              [options]="rolePermissionFilterOptions()?.roles ?? []"
              optionLabel="label"
              optionValue="code"
              [(ngModel)]="rolePermissionForm.role_code"
              placeholder="Alege rolul"
            />
          </label>
          <label class="admin-field">
            <span>Permisiune</span>
            <p-select
              appendTo="body"
              [options]="rolePermissionFilterOptions()?.permissions ?? []"
              optionLabel="label"
              optionValue="code"
              [(ngModel)]="rolePermissionForm.permission_code"
              placeholder="Alege permisiunea"
            />
          </label>
          <label class="admin-field md:col-span-2">
            <span>Stare</span>
            <p-select appendTo="body" [options]="[{ label: 'Atribuit', value: true }, { label: 'Revocat', value: false }]" [(ngModel)]="rolePermissionForm.assigned" />
          </label>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="rolePermissionDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" (onClick)="saveRolePermissionAssignment()" />
          </div>
        </ng-template>
      </p-dialog>

      <p-dialog
        [visible]="positionRoleDialogOpen()"
        (visibleChange)="positionRoleDialogOpen.set($event)"
        [modal]="true"
        [draggable]="false"
        [header]="editingPositionRoleAssignment() ? 'Editează maparea poziție-rol' : 'Mapare poziție-rol'"
        [style]="{ width: 'min(38rem, 94vw)' }"
      >
        <div class="grid gap-4 md:grid-cols-2">
          <label class="admin-field">
            <span>Poziție</span>
            <p-select
              appendTo="body"
              [options]="positionRoleFilterOptions()?.positions ?? []"
              optionLabel="name"
              optionValue="code"
              [(ngModel)]="positionRoleForm.position_code"
              placeholder="Alege poziția"
            />
          </label>
          <label class="admin-field">
            <span>Rol</span>
            <p-select
              appendTo="body"
              [options]="positionRoleFilterOptions()?.roles ?? []"
              optionLabel="label"
              optionValue="code"
              [(ngModel)]="positionRoleForm.role_code"
              placeholder="Alege rolul"
            />
          </label>
          <label class="admin-field md:col-span-2">
            <span>Stare</span>
            <p-select appendTo="body" [options]="[{ label: 'Atribuit', value: true }, { label: 'Revocat', value: false }]" [(ngModel)]="positionRoleForm.assigned" />
          </label>
        </div>
        <ng-template pTemplate="footer">
          <div class="flex justify-end gap-2">
            <p-button label="Renunță" severity="secondary" [outlined]="true" (onClick)="positionRoleDialogOpen.set(false)" />
            <p-button label="Salvează" icon="pi pi-check" (onClick)="savePositionRoleAssignment()" />
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

    :host ::ng-deep .admin-workspace .p-tabs,
    :host ::ng-deep .admin-workspace .p-tabpanels,
    :host ::ng-deep .admin-workspace .p-tabpanel {
      display: flex;
      flex: 1 1 auto;
      min-height: 0;
      flex-direction: column;
      overflow: hidden;
    }

    .admin-field {
      display: flex;
      flex-direction: column;
      gap: 0.35rem;
      color: var(--p-text-muted-color);
      font-size: 0.875rem;
      font-weight: 600;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminWorkspaceComponent {
  private readonly api = inject(AdminApiService);
  protected readonly authz = inject(AuthzService);
  protected readonly featureAccessRules = FEATURE_ACCESS_RULES;

  protected readonly tabs: AdminTab[] = [
    { value: 'users', label: 'Utilizatori', icon: 'pi pi-users', status: 'wired', description: 'Gestionare utilizatori.' },
    { value: 'rbac', label: 'RBAC', icon: 'pi pi-shield', status: 'wired', description: 'Catalog de roluri, permisiuni și mapări poziții.' },
    { value: 'compartimente', label: 'Compartimente', icon: 'pi pi-sitemap', status: 'contract-missing', description: 'Compartimente operaționale pentru registratură și fluxuri.' },
    { value: 'registre', label: 'Registre', icon: 'pi pi-book', status: 'contract-missing', description: 'Registre, prefixe, numerotare, registru implicit și compartimente asociate.' },
    { value: 'persoane-fizice', label: 'Persoane Fizice', icon: 'pi pi-id-card', status: 'contract-missing', description: 'Nomenclator persoane fizice folosit de emitent/destinatar.' },
    { value: 'persoane-juridice', label: 'Persoane Juridice', icon: 'pi pi-building', status: 'contract-missing', description: 'Nomenclator persoane juridice și date de identificare.' },
    { value: 'institutii-publice', label: 'Instituții Publice', icon: 'pi pi-landmark', status: 'contract-missing', description: 'Instituții publice, instituție implicită și ierarhii.' },
    { value: 'organizatii', label: 'Organizații', icon: 'pi pi-warehouse', status: 'contract-missing', description: 'Organizații și structuri externe/instituționale.' },
    { value: 'organigrama', label: 'Organigramă', icon: 'pi pi-share-alt', status: 'contract-missing', description: 'Structură organizațională vizuală și atribuiri.' },
    { value: 'profile', label: 'Profil utilizator', icon: 'pi pi-user', status: 'wired', description: 'Profil utilizator.' },
  ];
  protected readonly missingContractTabs = this.tabs.filter((tab) => tab.status === 'contract-missing');

  protected readonly users = signal<AdminUser[]>([]);
  protected readonly usersTotal = signal(0);
  protected readonly usersLoading = signal(false);
  protected readonly usersQuery = signal<TableQuery>({ page: 1, pageSize: 25, sort: 'name', direction: 'asc', filters: {} });
  protected readonly userDialogOpen = signal(false);
  protected readonly editingUser = signal<AdminUser | null>(null);

  protected readonly userFilters: { name: string; email: string; phone: string; position: string } = { name: '', email: '', phone: '', position: '' };
  protected readonly localeOptions = [{ label: 'RO', value: 'ro' }, { label: 'EN', value: 'en' }];
  protected readonly statusOptions = [{ label: 'Activ', value: 'active' }, { label: 'Inactiv', value: 'inactive' }];
  protected readonly userForm: UpsertAdminUserRequest = {
    name: '',
    email: '',
    phone: '',
    locale: 'ro',
    status: 'active',
    email_verified: false,
    phone_verified: false,
    preferred_otp_channel: 'sms',
  };

  protected readonly roles = signal<AdminRole[]>([]);
  protected readonly rolesTotal = signal(0);
  protected readonly rolesLoading = signal(false);
  protected readonly rolesQuery = signal<TableQuery>({ page: 1, pageSize: 100, sort: 'code', direction: 'asc', filters: {} });
  protected readonly roleFilters: AdminRoleFilterState = { code: '', label: '' };
  protected readonly roleDialogOpen = signal(false);
  protected readonly editingRole = signal<AdminRole | null>(null);
  protected readonly roleForm: UpsertAdminRoleRequest = { code: '', label: '' };

  protected readonly userRoleAssignments = signal<AdminUserRoleAssignment[]>([]);
  protected readonly userRoleAssignmentsTotal = signal(0);
  protected readonly userRoleAssignmentsLoading = signal(false);
  protected readonly userRoleAssignmentsQuery = signal<TableQuery>({ page: 1, pageSize: 25, sort: 'user_name', direction: 'asc', filters: {} });
  protected readonly userRoleAssignmentFilters: AdminUserRoleAssignmentFilterState = { user_name: '', role_code: '' };
  protected readonly userRoleDialogOpen = signal(false);
  protected readonly editingUserRoleAssignment = signal<AdminUserRoleAssignment | null>(null);
  protected readonly userRoleForm: UpsertAdminUserRoleAssignmentRequest = { user_id: '', role_code: '', assigned: true };
  protected readonly userLookup = signal<AdminUser[]>([]);

  protected readonly rolePermissionAssignments = signal<AdminRolePermissionAssignment[]>([]);
  protected readonly rolePermissionAssignmentsTotal = signal(0);
  protected readonly rolePermissionAssignmentsLoading = signal(false);
  protected readonly rolePermissionAssignmentsQuery = signal<TableQuery>({ page: 1, pageSize: 25, sort: 'role_code', direction: 'asc', filters: {} });
  protected readonly rolePermissionAssignmentFilters: AdminRolePermissionAssignmentFilterState = { role_code: '', permission_code: '' };
  protected readonly rolePermissionDialogOpen = signal(false);
  protected readonly editingRolePermissionAssignment = signal<AdminRolePermissionAssignment | null>(null);
  protected readonly rolePermissionForm: UpsertAdminRolePermissionAssignmentRequest = { role_code: '', permission_code: '', assigned: true };
  protected readonly rolePermissionFilterOptions = signal<AdminRolePermissionAssignmentFilters | null>(null);

  protected readonly positionRoleAssignments = signal<AdminPositionRoleAssignment[]>([]);
  protected readonly positionRoleAssignmentsTotal = signal(0);
  protected readonly positionRoleAssignmentsLoading = signal(false);
  protected readonly positionRoleAssignmentsQuery = signal<TableQuery>({ page: 1, pageSize: 25, sort: 'position_code', direction: 'asc', filters: {} });
  protected readonly positionRoleAssignmentFilters: AdminPositionRoleAssignmentFilterState = { position_code: '', role_code: '' };
  protected readonly positionRoleDialogOpen = signal(false);
  protected readonly editingPositionRoleAssignment = signal<AdminPositionRoleAssignment | null>(null);
  protected readonly positionRoleForm: UpsertAdminPositionRoleAssignmentRequest = { position_code: '', role_code: '', assigned: true };
  protected readonly positionRoleFilterOptions = signal<AdminPositionRoleAssignmentFilters | null>(null);

  ngOnInit(): void {
    this.loadUsers();
    this.loadRoles();
    this.loadUserRoleAssignments();
    this.loadRolePermissionAssignments();
    this.loadPositionRoleAssignments();
    this.loadRolePermissionAssignmentFilters();
    this.loadPositionRoleAssignmentFilters();
    this.loadAllUsersForAssignments();
  }

  protected loadUsers(event?: TableLazyLoadEvent): void {
    const pageSize = event?.rows ?? this.usersQuery().pageSize;
    const page = Math.floor((event?.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? event?.sortField[0] : event?.sortField;
    const query: TableQuery = {
      page,
      pageSize,
      sort: sort || this.usersQuery().sort,
      direction: event?.sortOrder === -1 ? 'desc' : 'asc',
      filters: Object.fromEntries(Object.entries(this.userFilters).filter(([, value]) => value.trim() !== '')),
    };
    this.usersQuery.set(query);
    this.usersLoading.set(true);
    this.api.users(query).subscribe({
      next: (res: PagedResponse<AdminUser>) => {
        this.users.set(res.items);
        this.usersTotal.set(res.total);
        this.usersLoading.set(false);
      },
      error: () => {
        this.users.set([]);
        this.usersTotal.set(0);
        this.usersLoading.set(false);
      },
    });
  }

  protected reloadUsers(): void {
    this.loadUsers({ first: 0, rows: this.usersQuery().pageSize });
  }

  protected openUserDialog(user?: AdminUser): void {
    this.editingUser.set(user ?? null);
    Object.assign(this.userForm, {
      id: user?.id,
      name: user?.name ?? '',
      email: user?.email ?? '',
      phone: user?.phone ?? '',
      locale: user?.locale ?? 'ro',
      status: user?.status ?? 'active',
      email_verified: user?.email_verified ?? false,
      phone_verified: user?.phone_verified ?? false,
      preferred_otp_channel: user?.preferred_otp_channel ?? 'sms',
    });
    this.userDialogOpen.set(true);
  }

  protected saveUser(): void {
    this.api.saveUser({ ...this.userForm }).subscribe({
      next: () => {
        this.userDialogOpen.set(false);
        this.reloadUsers();
      },
    });
  }

  protected loadRoles(event?: TableLazyLoadEvent): void {
    const pageSize = event?.rows ?? this.rolesQuery().pageSize;
    const page = Math.floor((event?.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? event?.sortField[0] : event?.sortField;
    const query: TableQuery = {
      page,
      pageSize,
      sort: sort || this.rolesQuery().sort,
      direction: event?.sortOrder === -1 ? 'desc' : 'asc',
      filters: Object.fromEntries(Object.entries(this.roleFilters).filter(([, value]) => value.trim() !== '')),
    };
    this.rolesQuery.set(query);
    this.rolesLoading.set(true);
    this.api.roles(query).subscribe({
      next: (res: PagedResponse<AdminRole>) => {
        this.roles.set(res.items);
        this.rolesTotal.set(res.total);
        this.rolesLoading.set(false);
      },
      error: () => {
        this.roles.set([]);
        this.rolesTotal.set(0);
        this.rolesLoading.set(false);
      },
    });
  }

  protected reloadRoles(): void {
    this.loadRoles({ first: 0, rows: this.rolesQuery().pageSize });
  }

  protected openRoleDialog(role?: AdminRole): void {
    this.editingRole.set(role ?? null);
    Object.assign(this.roleForm, {
      code: role?.code ?? '',
      label: role?.label ?? '',
    });
    this.roleDialogOpen.set(true);
  }

  protected saveRole(): void {
    this.api.saveRole({ ...this.roleForm }).subscribe({
      next: () => {
        this.roleDialogOpen.set(false);
        this.reloadRoles();
        this.loadRolePermissionAssignmentFilters();
        this.loadPositionRoleAssignmentFilters();
      },
    });
  }

  protected loadAllUsersForAssignments(): void {
    this.api.users({ page: 1, pageSize: 500, sort: 'name', direction: 'asc', filters: {} }).subscribe({
      next: (res: PagedResponse<AdminUser>) => this.userLookup.set(res.items),
      error: () => this.userLookup.set([]),
    });
  }

  protected loadUserRoleAssignments(event?: TableLazyLoadEvent): void {
    const pageSize = event?.rows ?? this.userRoleAssignmentsQuery().pageSize;
    const page = Math.floor((event?.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? event?.sortField[0] : event?.sortField;
    const query: TableQuery = {
      page,
      pageSize,
      sort: sort || this.userRoleAssignmentsQuery().sort,
      direction: event?.sortOrder === -1 ? 'desc' : 'asc',
      filters: Object.fromEntries(Object.entries(this.userRoleAssignmentFilters).filter(([, value]) => value.trim() !== '')),
    };
    this.userRoleAssignmentsQuery.set(query);
    this.userRoleAssignmentsLoading.set(true);
    this.api.roleAssignments(query).subscribe({
      next: (res: PagedResponse<AdminUserRoleAssignment>) => {
        this.userRoleAssignments.set(res.items);
        this.userRoleAssignmentsTotal.set(res.total);
        this.userRoleAssignmentsLoading.set(false);
      },
      error: () => {
        this.userRoleAssignments.set([]);
        this.userRoleAssignmentsTotal.set(0);
        this.userRoleAssignmentsLoading.set(false);
      },
    });
  }

  protected reloadUserRoleAssignments(): void {
    this.loadUserRoleAssignments({ first: 0, rows: this.userRoleAssignmentsQuery().pageSize });
  }

  protected openUserRoleDialog(assignment?: AdminUserRoleAssignment): void {
    this.editingUserRoleAssignment.set(assignment ?? null);
    Object.assign(this.userRoleForm, {
      user_id: assignment?.user_id ?? '',
      role_code: assignment?.role_code ?? '',
      assigned: true,
    });
    this.userRoleDialogOpen.set(true);
  }

  protected saveUserRoleAssignment(): void {
    this.api.saveRoleAssignment({ ...this.userRoleForm }).subscribe({
      next: () => {
        this.userRoleDialogOpen.set(false);
        this.reloadUserRoleAssignments();
      },
    });
  }

  protected removeUserRoleAssignment(assignment: AdminUserRoleAssignment): void {
    this.api.saveRoleAssignment({ user_id: assignment.user_id, role_code: assignment.role_code, assigned: false }).subscribe({
      next: () => this.reloadUserRoleAssignments(),
    });
  }

  protected loadRolePermissionAssignmentFilters(): void {
    this.api.rolePermissionAssignmentFilters().subscribe({
      next: (filters) => this.rolePermissionFilterOptions.set(filters),
      error: () => this.rolePermissionFilterOptions.set(null),
    });
  }

  protected loadRolePermissionAssignments(event?: TableLazyLoadEvent): void {
    const pageSize = event?.rows ?? this.rolePermissionAssignmentsQuery().pageSize;
    const page = Math.floor((event?.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? event?.sortField[0] : event?.sortField;
    const query: TableQuery = {
      page,
      pageSize,
      sort: sort || this.rolePermissionAssignmentsQuery().sort,
      direction: event?.sortOrder === -1 ? 'desc' : 'asc',
      filters: Object.fromEntries(Object.entries(this.rolePermissionAssignmentFilters).filter(([, value]) => value.trim() !== '')),
    };
    this.rolePermissionAssignmentsQuery.set(query);
    this.rolePermissionAssignmentsLoading.set(true);
    this.api.rolePermissionAssignments(query).subscribe({
      next: (res: PagedResponse<AdminRolePermissionAssignment>) => {
        this.rolePermissionAssignments.set(res.items);
        this.rolePermissionAssignmentsTotal.set(res.total);
        this.rolePermissionAssignmentsLoading.set(false);
      },
      error: () => {
        this.rolePermissionAssignments.set([]);
        this.rolePermissionAssignmentsTotal.set(0);
        this.rolePermissionAssignmentsLoading.set(false);
      },
    });
  }

  protected reloadRolePermissionAssignments(): void {
    this.loadRolePermissionAssignments({ first: 0, rows: this.rolePermissionAssignmentsQuery().pageSize });
  }

  protected openRolePermissionDialog(assignment?: AdminRolePermissionAssignment): void {
    this.editingRolePermissionAssignment.set(assignment ?? null);
    Object.assign(this.rolePermissionForm, {
      role_code: assignment?.role_code ?? '',
      permission_code: assignment?.permission_code ?? '',
      assigned: true,
    });
    this.rolePermissionDialogOpen.set(true);
  }

  protected saveRolePermissionAssignment(): void {
    this.api.saveRolePermissionAssignment({ ...this.rolePermissionForm }).subscribe({
      next: () => {
        this.rolePermissionDialogOpen.set(false);
        this.reloadRolePermissionAssignments();
      },
    });
  }

  protected removeRolePermissionAssignment(assignment: AdminRolePermissionAssignment): void {
    this.api.saveRolePermissionAssignment({ role_code: assignment.role_code, permission_code: assignment.permission_code, assigned: false }).subscribe({
      next: () => this.reloadRolePermissionAssignments(),
    });
  }

  protected loadPositionRoleAssignmentFilters(): void {
    this.api.positionRoleAssignmentFilters().subscribe({
      next: (filters) => this.positionRoleFilterOptions.set(filters),
      error: () => this.positionRoleFilterOptions.set(null),
    });
  }

  protected loadPositionRoleAssignments(event?: TableLazyLoadEvent): void {
    const pageSize = event?.rows ?? this.positionRoleAssignmentsQuery().pageSize;
    const page = Math.floor((event?.first ?? 0) / pageSize) + 1;
    const sort = Array.isArray(event?.sortField) ? event?.sortField[0] : event?.sortField;
    const query: TableQuery = {
      page,
      pageSize,
      sort: sort || this.positionRoleAssignmentsQuery().sort,
      direction: event?.sortOrder === -1 ? 'desc' : 'asc',
      filters: Object.fromEntries(Object.entries(this.positionRoleAssignmentFilters).filter(([, value]) => value.trim() !== '')),
    };
    this.positionRoleAssignmentsQuery.set(query);
    this.positionRoleAssignmentsLoading.set(true);
    this.api.positionRoleAssignments(query).subscribe({
      next: (res: PagedResponse<AdminPositionRoleAssignment>) => {
        this.positionRoleAssignments.set(res.items);
        this.positionRoleAssignmentsTotal.set(res.total);
        this.positionRoleAssignmentsLoading.set(false);
      },
      error: () => {
        this.positionRoleAssignments.set([]);
        this.positionRoleAssignmentsTotal.set(0);
        this.positionRoleAssignmentsLoading.set(false);
      },
    });
  }

  protected reloadPositionRoleAssignments(): void {
    this.loadPositionRoleAssignments({ first: 0, rows: this.positionRoleAssignmentsQuery().pageSize });
  }

  protected openPositionRoleDialog(assignment?: AdminPositionRoleAssignment): void {
    this.editingPositionRoleAssignment.set(assignment ?? null);
    Object.assign(this.positionRoleForm, {
      position_code: assignment?.position_code ?? '',
      role_code: assignment?.role_code ?? '',
      assigned: true,
    });
    this.positionRoleDialogOpen.set(true);
  }

  protected savePositionRoleAssignment(): void {
    this.api.savePositionRoleAssignment({ ...this.positionRoleForm }).subscribe({
      next: () => {
        this.positionRoleDialogOpen.set(false);
        this.reloadPositionRoleAssignments();
      },
    });
  }

  protected removePositionRoleAssignment(assignment: AdminPositionRoleAssignment): void {
    this.api.savePositionRoleAssignment({ position_code: assignment.position_code, role_code: assignment.role_code, assigned: false }).subscribe({
      next: () => this.reloadPositionRoleAssignments(),
    });
  }

  protected initials(value: string): string {
    return value
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase() ?? '')
      .join('') || 'U';
  }
}
