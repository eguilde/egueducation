import { BreakpointObserver } from '@angular/cdk/layout';
import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { map } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatToolbarModule } from '@angular/material/toolbar';

import { AuthService } from '../core/auth/auth.service';
import { AuthzService } from '../core/authz/authz.service';
import { ThemeService } from '../core/ui/theme.service';

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
    MatButtonModule,
    MatButtonToggleModule,
    MatDividerModule,
    MatExpansionModule,
    MatIconModule,
    MatListModule,
    MatSidenavModule,
    MatToolbarModule,
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
  protected readonly theme = inject(ThemeService);

  protected readonly navItems: NavItem[] = [
    {
      icon: 'dashboard',
      labelKey: 'nav.dashboard',
      route: '/dashboard',
      permission: 'dashboard.read',
      module: 'dashboard',
    },
    {
      icon: 'folder_open',
      labelKey: 'nav.documente',
      route: '/documente/dashboard',
      permission: ['registratura.read', 'workflow.read', 'earchiva.read'],
      permissionMode: 'any',
      module: ['registratura', 'workflow', 'earchiva'],
      moduleMode: 'any',
    },
    {
      icon: 'school',
      labelKey: 'nav.education',
      route: '/education/dashboard',
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
    {
      icon: 'policy',
      labelKey: 'nav.gdpr',
      route: '/gdpr/dashboard',
      permission: ['gdpr.read', 'gdpr.policies.read', 'gdpr.requests.read'],
      permissionMode: 'all',
      module: 'gdpr',
    },
    { icon: 'admin_panel_settings', labelKey: 'nav.admin', route: '/admin/dashboard', permission: 'admin.read', module: 'admin' },
  ];
  protected readonly visibleNavItems = computed(() =>
    this.navItems.filter((item) =>
      this.canAccess(item.permission, item.module, item.permissionMode, item.moduleMode),
    ),
  );

  protected readonly mobileOpen = signal(false);
  private readonly handset = toSignal(
    this.breakpoints.observe('(max-width: 1024px)').pipe(map((state) => state.matches)),
    { initialValue: false },
  );
  protected readonly isHandset = computed(() => this.handset());
  protected readonly sidenavMode = computed<'side' | 'over'>(() => (this.isHandset() ? 'over' : 'side'));
  protected readonly sidenavOpened = computed(() => (this.isHandset() ? this.mobileOpen() : true));

  protected toggleNav(): void {
    if (this.isHandset()) {
      this.mobileOpen.update((open) => !open);
    }
  }

  protected closeMobileNav(): void {
    if (this.isHandset()) {
      this.mobileOpen.set(false);
    }
  }

  protected async signIn(): Promise<void> {
    await this.router.navigate(['/login']);
  }

  protected async signOut(): Promise<void> {
    await this.router.navigate(['/auth/logout']);
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
