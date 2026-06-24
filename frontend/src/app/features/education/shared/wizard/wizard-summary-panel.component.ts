import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';

@Component({
  selector: 'app-wizard-summary-panel',
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

      <div class="mt-4 grid gap-3 md:grid-cols-2">
        @for (item of items(); track item.key) {
          <div class="rounded-xl border border-surface p-3">
            <div class="text-xs font-semibold uppercase tracking-wide text-muted-color">{{ item.label }}</div>
            <div class="mt-1 font-medium">{{ item.value || '-' }}</div>
          </div>
        }
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WizardSummaryPanelComponent {
  readonly title = input.required<string>();
  readonly description = input<string>('');
  readonly items = input.required<Array<{ key: string; label: string; value: string }>>();
}
