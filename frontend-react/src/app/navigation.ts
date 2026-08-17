export interface NavItem { label: string; icon: string; to: string; permission?: string; module?: string }
export const navigation: NavItem[] = [
  { label: 'Acasă', icon: 'pi pi-home', to: '/' },
  { label: 'Registratură', icon: 'pi pi-inbox', to: '/registratura', permission: 'registratura.read', module: 'registratura' },
  { label: 'Flux documente', icon: 'pi pi-send', to: '/flux-documente', permission: 'workflow.read', module: 'workflow' },
  { label: 'eArhivă', icon: 'pi pi-folder-open', to: '/earchiva', permission: 'earchiva.read' },
  { label: 'Școală', icon: 'pi pi-building-columns', to: '/scoala', permission: 'education.read', module: 'education' },
  { label: 'Administrare', icon: 'pi pi-cog', to: '/administrare', permission: 'admin.read' }
];
