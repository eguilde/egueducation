import { ChangeDetectionStrategy, Component } from '@angular/core';

import { EDUCATION_GOVERNANCE_SECTION } from '../shared/education-config';
import { EducationDomainWorkspaceComponent } from '../shared/education-domain-workspace.component';

@Component({
  selector: 'app-education-governance-page',
  standalone: true,
  imports: [EducationDomainWorkspaceComponent],
  template: `<app-education-domain-workspace [section]="section" />`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationGovernancePageComponent {
  protected readonly section = EDUCATION_GOVERNANCE_SECTION;
}
