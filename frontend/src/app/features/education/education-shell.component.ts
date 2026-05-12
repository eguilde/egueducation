import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterOutlet } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';
import { filter, map, startWith } from 'rxjs';

import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';

import { AuthzService } from '../../core/authz/authz.service';

interface WorkspaceTab {
  icon: string;
  labelKey: string;
  route: string;
  permission: string;
}

interface WorkspaceSection {
  descriptionKey: string;
  icon: string;
  id: 'dashboard' | 'governance' | 'personnel' | 'portfolios';
  labelKey: string;
  route: string;
  tabs: WorkspaceTab[];
}

@Component({
  selector: 'app-education-shell',
  standalone: true,
  imports: [RouterOutlet, RouterLink, TranslocoPipe, MatIconModule, MatTabsModule],
  template: `
    <section class="grid gap-5">
      <header class="grid gap-2 rounded-[1.5rem] border border-slate-200/60 bg-white/80 p-5 shadow-sm shadow-rose-100/40 dark:border-slate-800 dark:bg-slate-950/60">
        <p class="m-0 text-[0.78rem] font-bold uppercase tracking-[0.16em] text-rose-600">
          {{ activeSection().labelKey | transloco }}
        </p>
        <div class="grid gap-2 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
          <div class="grid gap-1">
            <h1 class="m-0 text-2xl font-semibold text-slate-900 dark:text-slate-50">
              {{ 'educationWorkspace.title' | transloco }}
            </h1>
            <p class="m-0 max-w-4xl text-sm leading-6 text-slate-600 dark:text-slate-300">
              {{ activeSection().descriptionKey | transloco }}
            </p>
          </div>
          <div class="hidden items-center gap-2 rounded-full bg-rose-50 px-3 py-2 text-sm font-medium text-rose-700 dark:bg-rose-950/40 dark:text-rose-200 lg:flex">
            <mat-icon class="!text-[1.15rem]">{{ activeSection().icon }}</mat-icon>
            <span>{{ activeSection().labelKey | transloco }}</span>
          </div>
        </div>
      </header>

      <nav mat-tab-nav-bar [tabPanel]="panel" class="overflow-x-auto rounded-[1.25rem] border border-slate-200/70 bg-white/80 px-2 dark:border-slate-800 dark:bg-slate-950/60">
        @for (section of visibleSections(); track section.id) {
          <a
            mat-tab-link
            [routerLink]="section.route"
            [active]="activeSection().id === section.id"
          >
            <mat-icon>{{ section.icon }}</mat-icon>
            <span>{{ section.labelKey | transloco }}</span>
          </a>
        }
      </nav>

      @if (contextTabs().length > 0) {
        <nav class="flex flex-wrap gap-2 rounded-[1.25rem] border border-slate-200/70 bg-white/75 p-3 dark:border-slate-800 dark:bg-slate-950/50">
          @for (tab of contextTabs(); track tab.route) {
            <a
              [routerLink]="tab.route"
              class="inline-flex min-h-10 items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition-colors duration-150"
              [class.border-rose-300]="isTabActive(tab.route)"
              [class.bg-rose-50]="isTabActive(tab.route)"
              [class.text-rose-700]="isTabActive(tab.route)"
              [class.dark:border-rose-700]="isTabActive(tab.route)"
              [class.dark:bg-rose-950/40]="isTabActive(tab.route)"
              [class.dark:text-rose-200]="isTabActive(tab.route)"
              [class.border-slate-200]="!isTabActive(tab.route)"
              [class.bg-slate-50]="!isTabActive(tab.route)"
              [class.text-slate-700]="!isTabActive(tab.route)"
              [class.dark:border-slate-700]="!isTabActive(tab.route)"
              [class.dark:bg-slate-900]="!isTabActive(tab.route)"
              [class.dark:text-slate-200]="!isTabActive(tab.route)"
            >
              <mat-icon class="!text-[1.1rem]">{{ tab.icon }}</mat-icon>
              <span>{{ tab.labelKey | transloco }}</span>
            </a>
          }
        </nav>
      }

      <mat-tab-nav-panel #panel>
        <section class="min-w-0">
          <router-outlet />
        </section>
      </mat-tab-nav-panel>
    </section>
  `,
  styles: [`
    :host ::ng-deep .mdc-tab {
      min-width: max-content;
    }

    :host ::ng-deep .mdc-tab__text-label {
      display: inline-flex;
      align-items: center;
      gap: 0.45rem;
      font-weight: 600;
    }
  `],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationShellComponent {
  private readonly authz = inject(AuthzService);
  private readonly router = inject(Router);
  private readonly fallbackSection: WorkspaceSection = {
    id: 'dashboard',
    icon: 'dashboard',
    labelKey: 'nav.dashboard',
    descriptionKey: 'educationWorkspace.subtitle',
    route: '/education/dashboard',
    tabs: [],
  };

  private readonly url = toSignal(
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
      startWith(null),
      map(() => this.router.url.split('?')[0].split('#')[0]),
    ),
    { initialValue: this.router.url.split('?')[0].split('#')[0] },
  );

  private readonly sections: readonly WorkspaceSection[] = [
    {
      id: 'dashboard',
      icon: 'dashboard',
      labelKey: 'nav.dashboard',
      descriptionKey: 'educationWorkspace.subtitle',
      route: '/education/dashboard',
      tabs: [],
    },
    {
      id: 'governance',
      icon: 'gavel',
      labelKey: 'educationWorkspace.tabs.governance',
      descriptionKey: 'educationWorkspace.cards.governanceBody',
      route: '/education/governance',
      tabs: [
        { icon: 'groups', labelKey: 'educationWorkspace.tabs.governance', route: '/education/governance', permission: 'education.governance.read' },
        { icon: 'gavel', labelKey: 'nav.educationDecisions', route: '/education/decisions', permission: 'education.decisions.read' },
        { icon: 'policy', labelKey: 'nav.educationRegulations', route: '/education/regulations', permission: 'education.regulations.read' },
        { icon: 'folder_special', labelKey: 'nav.educationManagerial', route: '/education/managerial', permission: 'education.managerial.read' },
      ],
    },
    {
      id: 'personnel',
      icon: 'badge',
      labelKey: 'nav.educationPersonnel',
      descriptionKey: 'educationWorkspace.cards.personnelBody',
      route: '/education/personnel',
      tabs: [
        { icon: 'badge', labelKey: 'nav.educationPersonnel', route: '/education/personnel', permission: 'education.personnel.read' },
        { icon: 'assignment_turned_in', labelKey: 'nav.educationEvaluations', route: '/education/evaluations', permission: 'education.evaluations.read' },
        { icon: 'verified_user', labelKey: 'nav.educationDeclarations', route: '/education/declarations', permission: 'education.declarations.read' },
        { icon: 'swap_horiz', labelKey: 'nav.educationMobility', route: '/education/mobility', permission: 'education.mobility.read' },
        { icon: 'military_tech', labelKey: 'nav.educationGradatii', route: '/education/gradatii', permission: 'education.gradatii.read' },
      ],
    },
    {
      id: 'portfolios',
      icon: 'folder_shared',
      labelKey: 'nav.educationPortfolios',
      descriptionKey: 'educationWorkspace.cards.portfolioBody',
      route: '/education/portfolios',
      tabs: [
        { icon: 'folder_shared', labelKey: 'nav.educationPortfolios', route: '/education/portfolios', permission: 'education.portfolios.read' },
      ],
    },
  ];

  protected readonly visibleSections = computed(() =>
    this.sections.filter((section) => {
      if (section.id === 'dashboard') {
        return this.authz.hasModule('education');
      }
      return section.tabs.some((tab) => this.authz.hasPermission(tab.permission) && this.authz.hasModule('education'));
    }).map((section) => ({
      ...section,
      tabs: section.tabs.filter((tab) => this.authz.hasPermission(tab.permission) && this.authz.hasModule('education')),
    })),
  );

  protected readonly activeSection = computed(() => {
    const currentUrl = this.url();
    return this.visibleSections().find((section) =>
      section.id === 'dashboard'
        ? currentUrl === section.route
        : section.tabs.some((tab) => currentUrl === tab.route),
    ) ?? this.visibleSections()[0] ?? this.fallbackSection;
  });

  protected readonly contextTabs = computed(() =>
    this.activeSection()?.id === 'dashboard' ? [] : (this.activeSection()?.tabs ?? []),
  );

  protected isTabActive(route: string): boolean {
    return this.url() === route;
  }
}
