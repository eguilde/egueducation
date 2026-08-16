import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

export interface RegistraturaAdminPage<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface RegistraturaDepartment {
  id: string;
  name: string;
  description?: string;
  parent_id?: string | null;
  role_tag?: string;
  user_count?: number;
}

export interface RegistraturaOrganization {
  id: string;
  name: string;
  description?: string;
  active: boolean;
  is_default: boolean;
  department_ids: string[];
}

export interface RegistraturaPartyAdmin {
  id: string;
  party_type: 'physical' | 'legal' | 'institution';
  display_name: string;
  first_name?: string;
  last_name?: string;
  legal_name?: string;
  identifier_code?: string;
  tax_id?: string;
  email?: string;
  phone_number?: string;
  birth_date?: string | null;
  birth_place?: string;
  trade_register_number?: string;
  legal_representative?: string;
  legal_form?: string;
  share_capital?: string;
  institution_type?: string;
  institution_level?: string;
  website?: string;
  active: boolean;
  is_default_organization?: boolean;
}

export interface RegistraturaRegistryAdmin {
  id: number;
  name: string;
  prefix: string;
  start_number: number;
  current_number: number;
  next_number: number;
  registry_type: 'public' | 'private';
  is_default: boolean;
  department_ids: string[];
}

export interface RegistraturaUserAssignment {
  user_id: string;
  department_ids: string[];
  primary_department_id?: string | null;
  organization_id?: string | null;
}

export interface RegistraturaOrganizationChartNode extends RegistraturaDepartment {
  users: Array<{ id: string; name: string; email?: string }>;
  children: RegistraturaOrganizationChartNode[];
}

export interface RegistraturaAdminQuery {
  page?: number;
  pageSize?: number;
  query?: string;
  party_type?: RegistraturaPartyAdmin['party_type'];
}

@Injectable({ providedIn: 'root' })
export class RegistraturaAdminApiService {
  private readonly http = inject(HttpClient);

  departments(query: RegistraturaAdminQuery = {}) {
    return this.http.get<RegistraturaAdminPage<RegistraturaDepartment>>('/api/registratura/admin/departments', { params: this.params(query) });
  }

  saveDepartment(payload: Partial<RegistraturaDepartment>) {
    return payload.id
      ? this.http.patch<RegistraturaDepartment>(`/api/registratura/admin/departments/${payload.id}`, payload)
      : this.http.post<RegistraturaDepartment>('/api/registratura/admin/departments', payload);
  }

  deleteDepartment(id: string) {
    return this.http.delete<void>(`/api/registratura/admin/departments/${id}`);
  }

  organizations(query: RegistraturaAdminQuery = {}) {
    return this.http.get<RegistraturaAdminPage<RegistraturaOrganization>>('/api/registratura/admin/organizations', { params: this.params(query) });
  }

  saveOrganization(payload: Partial<RegistraturaOrganization>) {
    return payload.id
      ? this.http.patch<RegistraturaOrganization>(`/api/registratura/admin/organizations/${payload.id}`, payload)
      : this.http.post<RegistraturaOrganization>('/api/registratura/admin/organizations', payload);
  }

  deleteOrganization(id: string) {
    return this.http.delete<void>(`/api/registratura/admin/organizations/${id}`);
  }

  parties(type: RegistraturaPartyAdmin['party_type'], query: RegistraturaAdminQuery = {}) {
    return this.http.get<RegistraturaAdminPage<RegistraturaPartyAdmin>>('/api/registratura/parties', {
      params: this.params({ ...query, party_type: type }),
    });
  }

  saveParty(payload: Partial<RegistraturaPartyAdmin>) {
    return payload.id
      ? this.http.patch<RegistraturaPartyAdmin>(`/api/registratura/parties/${payload.id}`, payload)
      : this.http.post<RegistraturaPartyAdmin>('/api/registratura/parties', payload);
  }

  deleteParty(id: string) {
    return this.http.delete<void>(`/api/registratura/parties/${id}`);
  }

  registries(query: RegistraturaAdminQuery = {}) {
    return this.http.get<RegistraturaAdminPage<RegistraturaRegistryAdmin>>('/api/registratura/admin/registries', { params: this.params(query) });
  }

  saveRegistry(payload: Partial<RegistraturaRegistryAdmin>) {
    return payload.id !== undefined
      ? this.http.patch<RegistraturaRegistryAdmin>(`/api/registratura/admin/registries/${payload.id}`, payload)
      : this.http.post<RegistraturaRegistryAdmin>('/api/registratura/admin/registries', payload);
  }

  deleteRegistry(id: number) {
    return this.http.delete<void>(`/api/registratura/admin/registries/${id}`);
  }

  chart() {
    return this.http.get<RegistraturaOrganizationChartNode[]>('/api/registratura/admin/organization-chart');
  }

  userAssignments(userId: string) {
    return this.http.get<RegistraturaUserAssignment>(`/api/registratura/admin/users/${userId}/assignments`);
  }

  saveUserAssignments(userId: string, payload: Omit<RegistraturaUserAssignment, 'user_id'>) {
    return this.http.put<RegistraturaUserAssignment>(`/api/registratura/admin/users/${userId}/assignments`, payload);
  }

  private params(query: RegistraturaAdminQuery): HttpParams {
    let params = new HttpParams();
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== null && value !== '') {
        params = params.set(key, String(value));
      }
    }
    return params;
  }
}
