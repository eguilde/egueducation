import { ChangeDetectionStrategy, Component } from '@angular/core';

import { EDUCATION_PERSONNEL_SECTION } from '../shared/education-config';
import { EducationDomainWorkspaceComponent } from '../shared/education-domain-workspace.component';

@Component({
  selector: 'app-education-personnel-page',
  standalone: true,
  imports: [EducationDomainWorkspaceComponent],
  template: `<app-education-domain-workspace [section]="section" />`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationPersonnelPageComponent {
  protected readonly section = EDUCATION_PERSONNEL_SECTION;
}
