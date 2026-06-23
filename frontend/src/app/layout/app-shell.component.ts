import { BreakpointObserver } from '@angular/cdk/layout';
import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { map } from 'rxjs/operators';

import { ButtonModule } from 'primeng/button';
import { DrawerModule } from 'primeng/drawer';
import { PopoverModule } from 'primeng/popover';
import { ToolbarModule } from 'primeng/toolbar';

import { AuthService } from '../core/auth/auth.service';
import { AppBrandingService } from '../core/branding/app-branding.service';
import { AuthzService } from '../core/authz/authz.service';
import { ThemeService } from '../core/ui/theme.service';
import { ThemePanelComponent } from './theme-panel.component';

interface NavItem {
  icon: string;
  labelKey: string;
  route: string;
  permission?: string | string[];
  permissionMode?: 'all' | 'any';
  module?: string | string[];
  moduleMode?: 'all' | 'any';
}

@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    TranslocoPipe,
    ButtonModule,
    DrawerModule,
    PopoverModule,
    ThemePanelComponent,
    ToolbarModule,
  ],
  templateUrl: './app-shell.component.html',
  styleUrl: './app-shell.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppShellComponent {
  private readonly breakpoints = inject(BreakpointObserver);
  private readonly router = inject(Router);
  protected readonly auth = inject(AuthService);
  protected readonly authz = inject(AuthzService);
  protected readonly branding = inject(AppBrandingService);
  protected readonly theme = inject(ThemeService);

  protected readonly navItems: NavItem[] = [
    {
      icon: 'pi pi-chart-line',
      labelKey: 'nav.dashboard',
      route: '/dashboard',
      permission: 'dashboard.read',
      module: 'dashboard',
    },
    {
      icon: 'pi pi-folder-open',
      labelKey: 'nav.documente',
      route: '/documente',
      permission: ['registratura.read', 'workflow.read', 'earchiva.read'],
      permissionMode: 'any',
      module: ['registratura', 'workflow', 'earchiva'],
      moduleMode: 'any',
    },
    {
      icon: 'pi pi-book',
      labelKey: 'nav.registre',
      route: '/registre',
      permission: 'registratura.read',
      module: 'registratura',
    },
    {
      icon: 'pi pi-sitemap',
      labelKey: 'nav.workflow',
      route: '/workflow',
      permission: 'workflow.read',
      module: 'workflow',
    },
    {
      icon: 'pi pi-inbox',
      labelKey: 'nav.earchiva',
      route: '/earchiva',
      permission: 'earchiva.read',
      module: 'earchiva',
    },
    {
      icon: 'pi pi-graduation-cap',
      labelKey: 'nav.education',
      route: '/education',
      permission: [
        'education.governance.read',
        'education.personnel.read',
        'education.decisions.read',
        'education.regulations.read',
        'education.managerial.read',
        'education.evaluations.read',
        'education.declarations.read',
        'education.mobility.read',
        'education.gradatii.read',
        'education.portfolios.read',
      ],
      permissionMode: 'any',
      module: 'education',
    },
    { icon: 'pi pi-cog', labelKey: 'nav.admin', route: '/admin', permission: 'admin.read', module: 'admin' },
  ];
  protected readonly visibleNavItems = computed(() =>
    this.navItems.filter((item) =>
      this.canAccess(item.permission, item.module, item.permissionMode, item.moduleMode),
    ),
  );

  protected readonly drawerVisible = signal(false);
  protected readonly authenticated = computed(() => this.auth.isAuthenticated());
  private readonly handset = toSignal(
    this.breakpoints.observe('(max-width: 1024px)').pipe(map((state) => state.matches)),
    { initialValue: false },
  );
  protected readonly isHandset = computed(() => this.handset());

  constructor() {
    effect(() => {
      if (!this.authenticated()) {
        this.drawerVisible.set(false);
        return;
      }
      this.drawerVisible.set(!this.isHandset());
    });
  }

  protected toggleDrawer(): void {
    this.drawerVisible.update((open) => !open);
  }

  protected closeMobileNav(): void {
    if (this.isHandset()) {
      this.drawerVisible.set(false);
    }
  }

  protected async signIn(): Promise<void> {
    await this.auth.login(this.router.url);
  }

  protected async signOut(): Promise<void> {
    await this.auth.logout();
  }

  protected canAccess(
    permission?: string | string[],
    moduleCode?: string | string[],
    permissionMode: 'all' | 'any' = 'any',
    moduleMode: 'all' | 'any' = 'any',
  ): boolean {
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
    return permissionOk && moduleOk;
  }
}
