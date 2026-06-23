import { BreakpointObserver } from '@angular/cdk/layout';
import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { map } from 'rxjs/operators';

import { ButtonModule } from 'primeng/button';
import { DrawerModule } from 'primeng/drawer';
import { DividerModule } from 'primeng/divider';
import { PopoverModule } from 'primeng/popover';
import { ToolbarModule } from 'primeng/toolbar';

import { AuthService } from '../core/auth/auth.service';
import { AppBrandingService } from '../core/branding/app-branding.service';
import { AuthzService } from '../core/authz/authz.service';
import { ThemeService } from '../core/ui/theme.service';
import { ThemePanelComponent } from './theme-panel.component';

interface NavItem {
  id: string;
  icon: string;
  label: string;
  route: string;
  roles?: string | string[];
  rolesMode?: 'all' | 'any';
  permission?: string | string[];
  permissionMode?: 'all' | 'any';
  module?: string | string[];
  moduleMode?: 'all' | 'any';
}

interface NavSection {
  title: string;
  description: string;
  items: NavItem[];
}

@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    ButtonModule,
    DrawerModule,
    DividerModule,
    PopoverModule,
    ThemePanelComponent,
    ToolbarModule,
  ],
  templateUrl: './app-shell.component.html',
  styleUrl: './app-shell.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppShellComponent {
  private readonly breakpoints = inject(BreakpointObserver);
  private readonly router = inject(Router);
  protected readonly auth = inject(AuthService);
  protected readonly authz = inject(AuthzService);
  protected readonly branding = inject(AppBrandingService);
  protected readonly theme = inject(ThemeService);

  protected readonly navSections: NavSection[] = [
    {
      title: 'Start',
      description: 'Punctul de intrare în aplicație.',
      items: [
        {
          id: 'home',
          icon: 'pi pi-home',
          label: 'Acasă',
          route: '/',
        },
      ],
    },
    {
      title: 'Registratură / Flux / Arhivă',
      description: 'Documente, trasee, registre și evidență arhivistică.',
      items: [
        {
          id: 'documente',
          icon: 'pi pi-folder-open',
          label: 'Documente',
          route: '/documente',
          roles: ['admin', 'super_admin', 'director', 'secretar', 'registrator'],
          rolesMode: 'any',
          permission: ['registratura.read', 'workflow.read', 'earchiva.read'],
          permissionMode: 'any',
          module: ['registratura', 'workflow', 'earchiva'],
          moduleMode: 'any',
        },
        {
          id: 'registre',
          icon: 'pi pi-book',
          label: 'Registre',
          route: '/registre',
          roles: ['admin', 'super_admin', 'director', 'secretar'],
          rolesMode: 'any',
          permission: 'registratura.read',
          module: 'registratura',
        },
        {
          id: 'workflow',
          icon: 'pi pi-sitemap',
          label: 'Flux documente',
          route: '/workflow',
          roles: ['admin', 'super_admin', 'director', 'secretar', 'profesor', 'inspector'],
          rolesMode: 'any',
          permission: 'workflow.read',
          module: 'workflow',
        },
        {
          id: 'earchiva',
          icon: 'pi pi-inbox',
          label: 'Arhivă electronică',
          route: '/earchiva',
          roles: ['admin', 'super_admin', 'director', 'secretar'],
          rolesMode: 'any',
          permission: 'earchiva.read',
          module: 'earchiva',
        },
      ],
    },
    {
      title: 'Educație',
      description: 'Guvernanță, personal, portofolii și conformitate educațională.',
      items: [
        {
          id: 'education-dashboard',
          icon: 'pi pi-chart-bar',
          label: 'Panou educație',
          route: '/education/dashboard',
          roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
          rolesMode: 'any',
          permission: ['education.read', 'education.governance.read', 'education.personnel.read', 'education.evaluations.read', 'education.declarations.read', 'education.mobility.read', 'education.gradatii.read', 'education.portfolios.read'],
          permissionMode: 'any',
          module: 'education',
        },
        {
          id: 'education-governance',
          icon: 'pi pi-users',
          label: 'Guvernanță',
          route: '/education/governance',
          roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
          rolesMode: 'any',
          permission: ['education.governance.read', 'education.decisions.read', 'education.managerial.read', 'education.regulations.read'],
          permissionMode: 'any',
          module: 'education',
        },
        {
          id: 'education-personnel',
          icon: 'pi pi-id-card',
          label: 'Personal',
          route: '/education/personnel',
          roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
          rolesMode: 'any',
          permission: ['education.personnel.read', 'education.evaluations.read', 'education.declarations.read', 'education.mobility.read', 'education.gradatii.read'],
          permissionMode: 'any',
          module: 'education',
        },
        {
          id: 'education-portfolio',
          icon: 'pi pi-folder-open',
          label: 'Portofolii',
          route: '/education/portfolio',
          roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
          rolesMode: 'any',
          permission: 'education.portfolios.read',
          module: 'education',
        },
        {
          id: 'education-compliance',
          icon: 'pi pi-shield',
          label: 'Conformitate',
          route: '/education/compliance',
          roles: ['admin', 'super_admin', 'director', 'profesor', 'inspector'],
          rolesMode: 'any',
          permission: 'education.read',
          module: 'education',
        },
      ],
    },
    {
      title: 'Administrare',
      description: 'RBAC, profil și setări de platformă.',
      items: [
        {
          id: 'admin',
          icon: 'pi pi-cog',
          label: 'Administrare',
          route: '/admin',
          roles: ['admin', 'super_admin', 'director'],
          rolesMode: 'any',
          permission: 'admin.read',
          module: 'admin',
        },
        {
          id: 'gdpr',
          icon: 'pi pi-shield',
          label: 'GDPR',
          route: '/gdpr',
          roles: ['admin', 'super_admin', 'director', 'gdpr_officer'],
          rolesMode: 'any',
          module: 'gdpr',
        },
      ],
    },
  ];
  protected readonly drawerSections = computed<NavSection[]>(() => {
    if (!this.authenticated()) {
      return [];
    }

    if (this.authz.hasFullAccess()) {
      return this.navSections;
    }

    return this.navSections
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => this.canAccess(item.roles, item.permission, item.module, item.rolesMode, item.permissionMode, item.moduleMode)),
      }))
      .filter((section) => section.items.length > 0);
  });

  protected readonly drawerVisible = signal(false);
  protected readonly authenticated = computed(() => this.auth.isAuthenticated());
  protected readonly userLabel = computed(() =>
    this.authz.user()?.name?.trim() ||
    this.auth.profile()?.name?.trim() ||
    this.authz.user()?.email?.trim() ||
    this.auth.profile()?.email?.trim() ||
    this.authz.user()?.sub?.trim() ||
    this.auth.profile()?.sub?.trim() ||
    'Contul meu',
  );
  private readonly desktop = toSignal(
    this.breakpoints.observe('(min-width: 1280px)').pipe(map((state) => state.matches)),
    { initialValue: false },
  );
  protected readonly isDesktop = computed(() => this.desktop());
  private lastDesktopState = false;

  constructor() {
    effect(() => {
      const authenticated = this.authenticated();
      const desktop = this.isDesktop();

      if (!authenticated) {
        this.drawerVisible.set(false);
        this.lastDesktopState = desktop;
        return;
      }

      if (desktop) {
        this.drawerVisible.set(true);
      } else if (this.lastDesktopState) {
        this.drawerVisible.set(false);
      }

      this.lastDesktopState = desktop;
    });
  }

  protected toggleDrawer(): void {
    if (this.isDesktop()) {
      this.drawerVisible.set(true);
      return;
    }
    this.drawerVisible.update((open) => !open);
  }

  protected closeMobileNav(): void {
    if (!this.isDesktop()) {
      this.drawerVisible.set(false);
    }
  }

  protected async signIn(): Promise<void> {
    await this.auth.login(this.router.url);
  }

  protected async signOut(): Promise<void> {
    await this.auth.logout();
  }

  protected openProfile(): void {
    void this.router.navigateByUrl('/profile');
    this.closeMobileNav();
  }

  protected canAccess(
    roles?: string | string[],
    permission?: string | string[],
    moduleCode?: string | string[],
    rolesMode: 'all' | 'any' = 'any',
    permissionMode: 'all' | 'any' = 'any',
    moduleMode: 'all' | 'any' = 'any',
  ): boolean {
    if (this.authz.hasFullAccess()) {
      return true;
    }

    const roleOk = !roles || (
      Array.isArray(roles)
        ? (rolesMode === 'all'
            ? roles.every((value) => this.authz.hasRole(value))
            : roles.some((value) => this.authz.hasRole(value)))
        : this.authz.hasRole(roles)
    );
    const permissionOk = !permission || (
      Array.isArray(permission)
        ? (permissionMode === 'all'
            ? permission.every((value) => this.authz.hasPermission(value))
            : permission.some((value) => this.authz.hasPermission(value)))
        : this.authz.hasPermission(permission)
    );
    const moduleOk = !moduleCode || (
      Array.isArray(moduleCode)
        ? (moduleMode === 'all'
            ? moduleCode.every((value) => this.authz.hasModule(value))
            : moduleCode.some((value) => this.authz.hasModule(value)))
        : this.authz.hasModule(moduleCode)
    );
    return roleOk && permissionOk && moduleOk;
  }
}
