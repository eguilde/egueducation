import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';

import { EducationWizardAuditEvent } from './wizard.models';

@Component({
  selector: 'app-audit-trail-panel',
  standalone: true,
  imports: [CommonModule],
  template: `
    <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
      <div class="space-y-1">
        <h3 class="m-0 text-lg font-semibold">{{ title() }}</h3>
        @if (description()) {
          <p class="m-0 text-sm text-muted-color">{{ description() }}</p>
        }
      </div>

      <div class="mt-4 space-y-3">
        @for (event of events(); track event.id) {
          <article class="rounded-xl border border-surface p-3">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="font-medium">{{ event.title }}</div>
              @if (event.happenedOn) {
                <div class="text-xs text-muted-color">{{ event.happenedOn }}</div>
              }
            </div>
            <p class="mb-0 mt-1 text-sm text-muted-color">{{ event.summary }}</p>
            @if (event.actorName) {
              <div class="mt-2 text-xs text-muted-color">Actor: {{ event.actorName }}</div>
            }
          </article>
        } @empty {
          <div class="rounded-xl border border-dashed border-surface p-4 text-sm text-muted-color">
            Nu exista evenimente de audit disponibile.
          </div>
        }
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuditTrailPanelComponent {
  readonly title = input.required<string>();
  readonly description = input<string>('');
  readonly events = input<EducationWizardAuditEvent[]>([]);
}
