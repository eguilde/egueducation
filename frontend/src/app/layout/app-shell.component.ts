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
  permission?: string;
  module?: string;
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
    { icon: 'dashboard', labelKey: 'nav.dashboard', route: '/dashboard', permission: 'dashboard.read', module: 'dashboard' },
    { icon: 'inbox', labelKey: 'nav.registratura', route: '/registratura', permission: 'registratura.read', module: 'registratura' },
    { icon: 'account_tree', labelKey: 'nav.workflow', route: '/workflow', permission: 'workflow.read', module: 'workflow' },
    { icon: 'inventory_2', labelKey: 'nav.earchiva', route: '/earchiva', permission: 'earchiva.read', module: 'earchiva' },
    { icon: 'school', labelKey: 'nav.education', route: '/education', permission: 'education.read', module: 'education' },
    { icon: 'gavel', labelKey: 'nav.educationDecisions', route: '/education/decisions', permission: 'education.decisions.read', module: 'education' },
    { icon: 'fact_check', labelKey: 'nav.educationManagerial', route: '/education/managerial', permission: 'education.managerial.read', module: 'education' },
    { icon: 'policy', labelKey: 'nav.educationRegulations', route: '/education/regulations', permission: 'education.regulations.read', module: 'education' },
    { icon: 'badge', labelKey: 'nav.educationPersonnel', route: '/education/personnel', permission: 'education.personnel.read', module: 'education' },
    { icon: 'assignment_turned_in', labelKey: 'nav.educationEvaluations', route: '/education/evaluations', permission: 'education.evaluations.read', module: 'education' },
    { icon: 'verified_user', labelKey: 'nav.educationDeclarations', route: '/education/declarations', permission: 'education.declarations.read', module: 'education' },
    { icon: 'swap_horiz', labelKey: 'nav.educationMobility', route: '/education/mobility', permission: 'education.mobility.read', module: 'education' },
    { icon: 'military_tech', labelKey: 'nav.educationGradatii', route: '/education/gradatii', permission: 'education.gradatii.read', module: 'education' },
    { icon: 'folder_shared', labelKey: 'nav.educationPortfolios', route: '/education/portfolios', permission: 'education.portfolios.read', module: 'education' },
    { icon: 'policy', labelKey: 'nav.gdpr', route: '/gdpr', permission: 'gdpr.read', module: 'gdpr' },
    { icon: 'download', labelKey: 'nav.gdprExports', route: '/gdpr/exports', permission: 'gdpr.exports.read', module: 'gdpr' },
    { icon: 'visibility_lock', labelKey: 'nav.gdprPublication', route: '/gdpr/publication-reviews', permission: 'gdpr.publication.read', module: 'gdpr' },
    { icon: 'admin_panel_settings', labelKey: 'nav.admin', route: '/admin', permission: 'admin.read', module: 'admin' },
  ];
  protected readonly visibleNavItems = computed(() =>
    this.navItems.filter((item) => this.canAccess(item.permission, item.module)),
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

  protected canAccess(permission?: string, moduleCode?: string): boolean {
    const permissionOk = !permission || this.authz.hasPermission(permission);
    const moduleOk = !moduleCode || this.authz.hasModule(moduleCode);
    return permissionOk && moduleOk;
  }
}
