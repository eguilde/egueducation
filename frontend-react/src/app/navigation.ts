export interface NavItem { label: string; icon: string; to: string; permission?: string; permissions?: string[]; module?: string; delegatedEducation?: boolean }
export const navigation: NavItem[] = [
  { label: 'Acasă', icon: 'pi pi-home', to: '/' },
  { label: 'Registratură', icon: 'pi pi-inbox', to: '/registratura', permission: 'registratura.read', module: 'registratura' },
  { label: 'Flux documente', icon: 'pi pi-send', to: '/flux-documente', permission: 'workflow.read', module: 'workflow' },
  { label: 'eArhivă', icon: 'pi pi-folder-open', to: '/earchiva', permission: 'earchiva.read' },
  { label: 'Școală', icon: 'pi pi-building-columns', to: '/scoala', permission: 'education.read', module: 'education' },
  { label: 'Clase și elevi', icon: 'pi pi-users', to: '/scoala/clase', permissions: ['education.classes.read', 'education.classes.manage', 'education.classes.read_assigned'], module: 'education' },
  { label: 'Secretariat', icon: 'pi pi-briefcase', to: '/scoala/secretariat', permission: 'education.cockpit.secretariat.read', module: 'education' },
  { label: 'Resurse umane', icon: 'pi pi-id-card', to: '/scoala/hr', permission: 'education.cockpit.hr.read', module: 'education' },
  { label: 'Cockpit comisie', icon: 'pi pi-sitemap', to: '/scoala/committee-cockpit', permission: 'education.cockpit.committee.read', module: 'education' },
  { label: 'Inspector', icon: 'pi pi-shield', to: '/scoala/inspector', permission: 'education.cockpit.inspector.read', module: 'education' },
  { label: 'Rapoarte școlare', icon: 'pi pi-chart-bar', to: '/scoala/reports', permissions: ['education.portfolios.school.read', 'education.portfolios.read', 'education.evaluations.read', 'education.governance.read', 'education.personnel.files.read', 'education.compliance.read'], module: 'education' },
  { label: 'Semnături și dovezi', icon: 'pi pi-verified', to: '/scoala/signatures', permissions: ['education.signatures.read', 'education.signatures.manage', 'education.signatures.validate'], module: 'education' },
  { label: 'Portofolii instituționale', icon: 'pi pi-folder-open', to: '/scoala/portfolios', permissions: ['education.portfolios.school.read', 'education.portfolios.read'], module: 'education' },
  { label: 'Portofoliul meu', icon: 'pi pi-folder', to: '/scoala/portfolio/me', permission: 'education.portfolios.read_own', module: 'education' },
  { label: 'Resurse delegate', icon: 'pi pi-share-alt', to: '/scoala/resurse-delegate', module: 'education', delegatedEducation: true },
  { label: 'Administrare', icon: 'pi pi-cog', to: '/administrare', permission: 'admin.read' }
];
