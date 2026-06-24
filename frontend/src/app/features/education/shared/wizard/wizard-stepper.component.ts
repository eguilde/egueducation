import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TagModule } from 'primeng/tag';

import { EducationWizardStepState } from './wizard.models';

@Component({
  selector: 'app-wizard-stepper',
  standalone: true,
  imports: [CommonModule, TagModule],
  template: `
    <nav class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      @for (step of steps(); track step.key) {
        <article class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
          <div class="flex items-start justify-between gap-3">
            <div class="space-y-1">
              <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">Pas</div>
              <div class="font-semibold text-color">{{ step.label }}</div>
              @if (step.description) {
                <p class="m-0 text-sm text-muted-color">{{ step.description }}</p>
              }
            </div>
            <p-tag [value]="statusLabel(step.status)" [severity]="statusSeverity(step.status)" />
          </div>
        </article>
      }
    </nav>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WizardStepperComponent {
  readonly steps = input.required<EducationWizardStepState[]>();

  protected statusLabel(status: EducationWizardStepState['status']): string {
    switch (status) {
      case 'active':
        return 'In lucru';
      case 'completed':
        return 'Finalizat';
      case 'blocked':
        return 'Blocat';
      default:
        return 'In asteptare';
    }
  }

  protected statusSeverity(status: EducationWizardStepState['status']): 'success' | 'warn' | 'danger' | 'secondary' {
    switch (status) {
      case 'completed':
        return 'success';
      case 'active':
        return 'warn';
      case 'blocked':
        return 'danger';
      default:
        return 'secondary';
    }
  }
}
