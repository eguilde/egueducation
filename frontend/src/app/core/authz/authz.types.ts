export interface SessionModule {
  code: string;
  active: boolean;
}

export interface SessionUser {
  id: string;
  sub: string;
  name: string;
  email: string;
  phone_number?: string;
  locale: 'ro' | 'en';
  roles: string[];
}

export interface SessionContext {
  user: SessionUser;
  institution_id: string;
  institution_name: string;
  permissions: string[];
  modules: SessionModule[];
  authentication: string[];
  gdpr_capabilities: string[];
}
