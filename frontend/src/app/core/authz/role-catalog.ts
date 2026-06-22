export interface FeatureAccessRule {
  feature: string;
  roles: string[];
  permissions: string[];
  description: string;
}

export const FEATURE_ACCESS_RULES: FeatureAccessRule[] = [
  {
    feature: 'documente',
    roles: ['admin', 'super_admin', 'director', 'secretar', 'registrator'],
    permissions: ['registratura.read', 'workflow.read', 'earchiva.read'],
    description: 'Listarea și navigarea documentelor registraturii.',
  },
  {
    feature: 'documente_create_edit',
    roles: ['admin', 'super_admin', 'director', 'secretar', 'registrator'],
    permissions: ['registratura.manage'],
    description: 'Creare, editare și anulare documente.',
  },
  {
    feature: 'registre',
    roles: ['admin', 'super_admin', 'director', 'secretar'],
    permissions: ['registratura.read'],
    description: 'Administrarea registrelor și numerotării.',
  },
  {
    feature: 'workflow',
    roles: ['admin', 'super_admin', 'director', 'secretar', 'profesor', 'inspector'],
    permissions: ['workflow.read'],
    description: 'Urmărirea și tranziția taskurilor workflow.',
  },
  {
    feature: 'earchiva',
    roles: ['admin', 'super_admin', 'director', 'secretar'],
    permissions: ['earchiva.read'],
    description: 'Înregistrări arhivistice și evidențe eArhivă.',
  },
  {
    feature: 'education',
    roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
    permissions: [],
    description: 'Modulele educaționale de linie și consultare.',
  },
  {
    feature: 'admin',
    roles: ['admin', 'super_admin', 'director'],
    permissions: ['admin.read'],
    description: 'Administrare platformă și catalog RBAC.',
  },
  {
    feature: 'gdpr',
    roles: ['admin', 'super_admin', 'director', 'gdpr_officer'],
    permissions: [],
    description: 'Operațiuni GDPR și protecția datelor.',
  },
];
