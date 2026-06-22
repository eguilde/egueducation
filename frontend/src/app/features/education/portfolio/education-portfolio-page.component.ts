import { ChangeDetectionStrategy, Component } from '@angular/core';

import { EDUCATION_PORTFOLIO_SECTION } from '../shared/education-config';
import { EducationDomainWorkspaceComponent } from '../shared/education-domain-workspace.component';

@Component({
  selector: 'app-education-portfolio-page',
  standalone: true,
  imports: [EducationDomainWorkspaceComponent],
  template: `<app-education-domain-workspace [section]="section" />`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EducationPortfolioPageComponent {
  protected readonly section = EDUCATION_PORTFOLIO_SECTION;
}
