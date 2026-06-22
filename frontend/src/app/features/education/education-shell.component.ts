import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter, map } from 'rxjs/operators';

import { TabsModule } from 'primeng/tabs';

import { AuthzService } from '../../core/authz/authz.service';
import { EDUCATION_ROUTE_TABS } from './shared/education-config';

@Component({
  selector: 'app-education-shell',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    TabsModule,
  ],
  template: `
    <section class="flex h-[calc(100dvh-6rem)] min-h-0 flex-col overflow-hidden">
      <div class="shrink-0 px-3 pt-3">
        <p-tabs [value]="activeTab()" (valueChange)="navigate(coerceTabValue($event, 'dashboard'))">
          <p-tablist>
            @for (tab of tabs(); track tab.path) {
              <p-tab [value]="tab.path">
                <span class="inline-flex items-center gap-2">
                  <i [class]="tab.icon"></i>
                  {{ tab.label }}
                </span>
              </p-tab>
            }
          </p-tablist>
        </p-tabs>
      </div>

      <div class="min-h-0 flex-1 overflow-hidden">
        <router-outlet />
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationShellComponent {
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly authz = inject(AuthzService);

  protected readonly tabs = computed(() =>
    EDUCATION_ROUTE_TABS.filter((tab) => tab.permissions.some((permission) => this.authz.hasPermission(permission))),
  );
  private readonly currentUrl = toSignal(
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
      map(() => this.router.url),
    ),
    { initialValue: this.router.url },
  );

  protected readonly activeTab = computed(() => {
    const parts = this.currentUrl().split('/');
    return parts[2] || 'dashboard';
  });

  protected navigate(path: string): void {
    void this.router.navigate([path], { relativeTo: this.route });
  }

  protected coerceTabValue(value: unknown, fallback: string): string {
    const normalized = String(value ?? '').trim();
    return normalized || fallback;
  }
}
