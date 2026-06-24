import { ChangeDetectionStrategy, Component } from '@angular/core';

import { EDUCATION_GOVERNANCE_SECTION } from '../shared/education-config';
import { EducationDomainWorkspaceComponent } from '../shared/education-domain-workspace.component';

@Component({
  selector: 'app-education-governance-page',
  standalone: true,
  imports: [EducationDomainWorkspaceComponent],
  template: `
    <section class="flex h-full min-h-0 flex-col gap-4 p-3">
      <div class="rounded-3xl border border-surface bg-surface-0 p-5 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="space-y-2">
            <div class="inline-flex items-center gap-2 text-sm font-medium text-muted-color">
              <i class="pi pi-sitemap"></i>
              Guvernanta educationala
            </div>
            <h1 class="m-0 text-3xl font-semibold">Guvernanta si trasee procedurale</h1>
            <p class="m-0 max-w-4xl text-sm text-muted-color">
              Registru operational pentru sedinte, membri, minute, hotarari si documentele aferente organismelor de conducere.
            </p>
          </div>
        </div>
      </div>

      <div class="min-h-0 flex-1">
        <app-education-domain-workspace [section]="section" />
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationGovernancePageComponent {
  protected readonly section = EDUCATION_GOVERNANCE_SECTION;
}
