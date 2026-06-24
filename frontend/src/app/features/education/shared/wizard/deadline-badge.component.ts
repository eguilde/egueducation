import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TagModule } from 'primeng/tag';

import { educationStatusSeverity } from '../education-status.helpers';

@Component({
  selector: 'app-deadline-badge',
  standalone: true,
  imports: [TagModule],
  template: `
    <p-tag [value]="label()" [severity]="severity()" />
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DeadlineBadgeComponent {
  readonly label = input.required<string>();
  readonly status = input<string>('');

  protected severity() {
    return educationStatusSeverity(this.status() || this.label());
  }
}
