export interface AdminPermissions {
  dashboard: boolean;
  usersRead: boolean;
  usersManage: boolean;
  rolesRead: boolean;
  rolesManage: boolean;
  modulesRead: boolean;
  modulesManage: boolean;
}

/** A permission check backed by the authenticated `/api/me` effective permissions. */
export type PermissionCheck = (permission: string) => boolean;

export interface Dashboard {
  stats: Record<string, number>;
  modules: ModuleSetting[];
  admin_sections: string[];
  warnings: string[];
}

export interface ModuleSetting {
  code: string;
  active: boolean;
}

export interface AdminUser {
  id: string;
  sub: string;
  name: string;
  email: string;
  phone: string;
  position: string;
  locale: string;
  status: string;
  email_verified: boolean;
  phone_verified: boolean;
  preferred_otp_channel: string;
  last_login_at: string;
}

export interface Role {
  code: string;
  label: string;
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

/** Lossless envelope for dynamically configured administration resources.
 * The API intentionally permits institution-specific fields in these records. */
export interface AdminResource {
  id?: string;
  code?: string;
  name?: string;
  label?: string;
  [key: string]: unknown;
}

export interface UpsertUserInput {
  id?: string;
  name: string;
  email: string;
  phone: string;
  locale: "ro" | "en";
  status: "active" | "inactive";
  email_verified: boolean;
  phone_verified: boolean;
  preferred_otp_channel: "sms" | "email";
}

export interface AdminApi {
  dashboard(): Promise<Dashboard>;
  users(query?: string): Promise<Page<AdminUser>>;
  saveUser(input: UpsertUserInput): Promise<AdminUser>;
  roles(): Promise<Page<Role>>;
  modules(): Promise<Page<ModuleSetting>>;
  saveModule(input: ModuleSetting): Promise<ModuleSetting>;
  resource(path: AdminResourcePath): Promise<Page<AdminResource>>;
	/** Saves only resources that have a corresponding backend POST operation. */
	saveResource(path: AdminWritableResourcePath, input: Record<string, unknown>): Promise<AdminResource>;
}

export type AdminResourcePath =
  | "audit" | "auth-methods" | "dossier-requirements" | "education-taxonomies"
  | "gdpr-settings" | "memberships" | "nomenclatures" | "oidc/clients"
  | "org-units" | "permissions" | "permissions/assignments" | "position-roles"
  | "positions" | "role-assignments" | "role-permissions" | "roles" | "modules" | "workflow-definitions"
  | "gdpr/config" | "gdpr/dashboard" | "gdpr/exports" | "gdpr/publication-reviews"
  | "gdpr/retention-policies" | "gdpr/subject-requests";

export type AdminWritableResourcePath =
	| "auth-methods" | "dossier-requirements" | "education-taxonomies" | "gdpr-settings"
	| "memberships" | "modules" | "nomenclatures" | "oidc/clients" | "org-units"
	| "permissions/assignments" | "position-roles" | "positions" | "role-assignments"
	| "role-permissions" | "roles" | "workflow-definitions"
	| "gdpr/exports" | "gdpr/publication-reviews" | "gdpr/retention-policies" | "gdpr/subject-requests";
