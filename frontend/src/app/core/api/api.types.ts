export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface AppMeta {
  name: string;
  default_locale: 'ro' | 'en';
  available_locales: Array<'ro' | 'en'>;
  theme: {
    family: string;
    brand: string;
  };
}

export interface AuthMethod {
  code: string;
  enabled: boolean;
  primary: boolean;
}

export interface AuthMethodsResponse {
  methods: AuthMethod[];
}

export interface AuthUiConfig {
  auth_flow: string;
  default_locale: 'ro' | 'en';
  available_locales: Array<'ro' | 'en'>;
  theme_family: string;
  theme_brand: string;
  oidc_issuer: string;
  oidc_client_id: string;
  desktop_client_id: string;
  sms_otp_enabled: boolean;
  passkey_enabled: boolean;
  eudi_wallet_enabled: boolean;
  gdpr_features_enabled: boolean;
}

export interface ModuleStatus {
  code: string;
  active: boolean;
}

export interface AdminDashboardResponse {
  stats: {
    users: number;
    memberships: number;
    positions: number;
    permissions: number;
    workflows: number;
    archives: number;
  };
  modules: ModuleStatus[];
  admin_sections: string[];
  warnings: string[];
}

export interface AdminUser {
  id: string;
  name: string;
  email: string;
  phone: string;
  position: string;
  locale: string;
  status: string;
  last_login_at: string;
}

export interface AdminUserFilters {
  positions: string[];
  statuses: string[];
  locales: string[];
}

export interface SessionContext {
  user: {
    id: string;
    sub: string;
    name: string;
    email: string;
    phone_number?: string;
    locale: 'ro' | 'en';
    roles: string[];
  };
  institution_id: string;
  institution_name: string;
  permissions: string[];
  modules: Array<{
    code: string;
    active: boolean;
  }>;
  authentication: string[];
  gdpr_capabilities: string[];
}

export interface TableQuery {
  page: number;
  pageSize: number;
  sort?: string;
  direction?: 'asc' | 'desc';
  filters?: Record<string, string>;
}
