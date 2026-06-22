import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  AdminDashboardResponse,
  AdminMembership,
  AdminOrgUnit,
  AdminPermission,
  AdminPermissionAssignment,
  AdminPermissionAssignmentFilters,
  AdminPosition,
  AdminPositionRoleAssignment,
  AdminPositionRoleAssignmentFilters,
  AdminRole,
  AdminDossierRequirement,
  AdminDossierRequirementFilters,
  AdminEducationTaxonomy,
  AdminEducationTaxonomyFilters,
  AdminAuthMethodSetting,
  AdminAuditFilters,
  AdminAuditEvent,
  AdminGdprSetting,
  AdminModuleSetting,
  AdminNomenclature,
  AdminNomenclatureFilters,
  AdminOIDCClient,
  AdminOIDCConsentGrant,
  AdminOIDCSession,
  AdminWorkflowDefinition,
  AdminWorkflowDefinitionFilters,
  AdminUser,
  AdminUserRoleAssignment,
  AdminRolePermissionAssignment,
  AdminRolePermissionAssignmentFilters,
  AdminUserFilters,
  CreateAdminDossierRequirementRequest,
  CreateAdminEducationTaxonomyRequest,
  CreateAdminNomenclatureRequest,
  CreateAdminWorkflowDefinitionRequest,
  PagedResponse,
  RevokeAdminOIDCConsentRequest,
  RevokeAdminOIDCSessionRequest,
  TableQuery,
  UpsertAdminMembershipRequest,
  UpsertAdminOrgUnitRequest,
  UpsertAdminOIDCClientRequest,
  UpsertAdminPermissionAssignmentRequest,
  UpsertAdminPositionRoleAssignmentRequest,
  UpsertAdminPositionRequest,
  UpsertAdminRoleRequest,
  UpsertAdminRolePermissionAssignmentRequest,
  UpsertAdminUserRequest,
  UpsertAdminUserRoleAssignmentRequest,
  UpdateAdminAuthMethodSettingRequest,
  UpdateAdminGdprSettingRequest,
  UpdateAdminModuleSettingRequest,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class AdminApiService {
  private readonly http = inject(HttpClient);

  dashboard() {
    return this.http.get<AdminDashboardResponse>('/api/admin/dashboard');
  }

  users(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<AdminUser>>('/api/admin/users', { params });
  }

  userFilters() {
    return this.http.get<AdminUserFilters>('/api/admin/users/filters');
  }

  saveUser(payload: UpsertAdminUserRequest) {
    return this.http.post<AdminUser>('/api/admin/users', payload);
  }

  roles(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminRole>>('/api/admin/roles', { params });
  }

  saveRole(payload: UpsertAdminRoleRequest) {
    return this.http.post<AdminRole>('/api/admin/roles', payload);
  }

  roleAssignments(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminUserRoleAssignment>>('/api/admin/role-assignments', { params });
  }

  saveRoleAssignment(payload: UpsertAdminUserRoleAssignmentRequest) {
    return this.http.post<AdminUserRoleAssignment>('/api/admin/role-assignments', payload);
  }

  rolePermissionAssignments(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminRolePermissionAssignment>>('/api/admin/role-permissions', { params });
  }

  rolePermissionAssignmentFilters() {
    return this.http.get<AdminRolePermissionAssignmentFilters>('/api/admin/role-permissions/filters');
  }

  saveRolePermissionAssignment(payload: UpsertAdminRolePermissionAssignmentRequest) {
    return this.http.post<AdminRolePermissionAssignment>('/api/admin/role-permissions', payload);
  }

  orgUnits(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminOrgUnit>>('/api/admin/org-units', { params });
  }

  saveOrgUnit(payload: UpsertAdminOrgUnitRequest) {
    return this.http.post<AdminOrgUnit>('/api/admin/org-units', payload);
  }

  memberships(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminMembership>>('/api/admin/memberships', { params });
  }

  saveMembership(payload: UpsertAdminMembershipRequest) {
    return this.http.post<AdminMembership>('/api/admin/memberships', payload);
  }

  positions(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminPosition>>('/api/admin/positions', { params });
  }

  savePosition(payload: UpsertAdminPositionRequest) {
    return this.http.post<AdminPosition>('/api/admin/positions', payload);
  }

  positionRoleAssignments(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminPositionRoleAssignment>>('/api/admin/position-roles', { params });
  }

  positionRoleAssignmentFilters() {
    return this.http.get<AdminPositionRoleAssignmentFilters>('/api/admin/position-roles/filters');
  }

  savePositionRoleAssignment(payload: UpsertAdminPositionRoleAssignmentRequest) {
    return this.http.post<AdminPositionRoleAssignment>('/api/admin/position-roles', payload);
  }

  permissions(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminPermission>>('/api/admin/permissions', { params });
  }

  permissionAssignments(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminPermissionAssignment>>('/api/admin/permissions/assignments', { params });
  }

  permissionAssignmentFilters() {
    return this.http.get<AdminPermissionAssignmentFilters>('/api/admin/permissions/assignments/filters');
  }

  savePermissionAssignment(payload: UpsertAdminPermissionAssignmentRequest) {
    return this.http.post<AdminPermissionAssignment>('/api/admin/permissions/assignments', payload);
  }

  dossierRequirements(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<AdminDossierRequirement>>('/api/admin/dossier-requirements', { params });
  }

  dossierRequirementFilters() {
    return this.http.get<AdminDossierRequirementFilters>('/api/admin/dossier-requirements/filters');
  }

  saveDossierRequirement(payload: CreateAdminDossierRequirementRequest) {
    return this.http.post<AdminDossierRequirement>('/api/admin/dossier-requirements', payload);
  }

  educationTaxonomies(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<AdminEducationTaxonomy>>('/api/admin/education-taxonomies', { params });
  }

  educationTaxonomyFilters() {
    return this.http.get<AdminEducationTaxonomyFilters>('/api/admin/education-taxonomies/filters');
  }

  saveEducationTaxonomy(payload: CreateAdminEducationTaxonomyRequest) {
    return this.http.post<AdminEducationTaxonomy>('/api/admin/education-taxonomies', payload);
  }

  workflowDefinitions(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<AdminWorkflowDefinition>>('/api/admin/workflow-definitions', { params });
  }

  workflowDefinitionFilters() {
    return this.http.get<AdminWorkflowDefinitionFilters>('/api/admin/workflow-definitions/filters');
  }

  saveWorkflowDefinition(payload: CreateAdminWorkflowDefinitionRequest) {
    return this.http.post<AdminWorkflowDefinition>('/api/admin/workflow-definitions', payload);
  }

  nomenclatures(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<AdminNomenclature>>('/api/admin/nomenclatures', { params });
  }

  nomenclatureFilters() {
    return this.http.get<AdminNomenclatureFilters>('/api/admin/nomenclatures/filters');
  }

  saveNomenclature(payload: CreateAdminNomenclatureRequest) {
    return this.http.post<AdminNomenclature>('/api/admin/nomenclatures', payload);
  }

  authMethods() {
    return this.http.get<PagedResponse<AdminAuthMethodSetting>>('/api/admin/auth-methods');
  }

  saveAuthMethod(payload: UpdateAdminAuthMethodSettingRequest) {
    return this.http.post<AdminAuthMethodSetting>('/api/admin/auth-methods', payload);
  }

  modules() {
    return this.http.get<PagedResponse<AdminModuleSetting>>('/api/admin/modules');
  }

  saveModule(payload: UpdateAdminModuleSettingRequest) {
    return this.http.post<AdminModuleSetting>('/api/admin/modules', payload);
  }

  oidcClients(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminOIDCClient>>('/api/admin/oidc/clients', { params });
  }

  saveOidcClient(payload: UpsertAdminOIDCClientRequest) {
    return this.http.post<AdminOIDCClient>('/api/admin/oidc/clients', payload);
  }

  oidcConsents(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminOIDCConsentGrant>>('/api/admin/oidc/consents', { params });
  }

  revokeOidcConsent(payload: RevokeAdminOIDCConsentRequest) {
    return this.http.post<{ status: string }>('/api/admin/oidc/consents/revoke', payload);
  }

  oidcSessions(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminOIDCSession>>('/api/admin/oidc/sessions', { params });
  }

  revokeOidcSession(payload: RevokeAdminOIDCSessionRequest) {
    return this.http.post<{ status: string }>('/api/admin/oidc/sessions/revoke', payload);
  }

  auditEvents(query: TableQuery) {
    let params = new HttpParams().set('page', String(query.page)).set('pageSize', String(query.pageSize));
    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }
    return this.http.get<PagedResponse<AdminAuditEvent>>('/api/admin/audit', { params });
  }

  auditFilters() {
    return this.http.get<AdminAuditFilters>('/api/admin/audit/filters');
  }

  gdprSettings() {
    return this.http.get<PagedResponse<AdminGdprSetting>>('/api/admin/gdpr-settings');
  }

  saveGdprSetting(payload: UpdateAdminGdprSettingRequest) {
    return this.http.post<AdminGdprSetting>('/api/admin/gdpr-settings', payload);
  }
}
