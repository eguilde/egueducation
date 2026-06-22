import { ChangeDetectionStrategy, Component } from '@angular/core';

import { EDUCATION_COMPLIANCE_SECTION } from '../shared/education-config';
import { EducationDomainWorkspaceComponent } from '../shared/education-domain-workspace.component';

@Component({
  selector: 'app-education-compliance-page',
  standalone: true,
  imports: [EducationDomainWorkspaceComponent],
  template: `<app-education-domain-workspace [section]="section" />`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationCompliancePageComponent {
  protected readonly section = EDUCATION_COMPLIANCE_SECTION;
}
