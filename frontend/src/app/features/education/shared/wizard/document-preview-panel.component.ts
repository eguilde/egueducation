import { CommonModule } from '@angular/common';
import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TagModule } from 'primeng/tag';

import { educationStatusSeverity } from '../education-status.helpers';
import { EducationWizardDocumentPreview } from './wizard.models';

@Component({
  selector: 'app-document-preview-panel',
  standalone: true,
  imports: [CommonModule, TagModule],
  template: `
    <section class="rounded-2xl border border-surface bg-surface-0 p-4 shadow-sm">
      <div class="space-y-1">
        <h3 class="m-0 text-lg font-semibold">{{ title() }}</h3>
        @if (description()) {
          <p class="m-0 text-sm text-muted-color">{{ description() }}</p>
        }
      </div>

      <div class="mt-4 space-y-3">
        @for (document of documents(); track document.id) {
          <div class="rounded-xl border border-surface p-3">
            <div class="flex items-start justify-between gap-3">
              <div class="space-y-1">
                <div class="font-medium">{{ document.title }}</div>
                @if (document.summary) {
                  <p class="m-0 text-sm text-muted-color">{{ document.summary }}</p>
                }
              </div>
              @if (document.status) {
                <p-tag [value]="document.status" [severity]="statusSeverity(document.status)" />
              }
            </div>
          </div>
        } @empty {
          <div class="rounded-xl border border-dashed border-surface p-4 text-sm text-muted-color">
            Nu exista documente generate pentru aceasta etapa.
          </div>
        }
      </div>
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DocumentPreviewPanelComponent {
  readonly title = input.required<string>();
  readonly description = input<string>('');
  readonly documents = input<EducationWizardDocumentPreview[]>([]);

  protected statusSeverity(value: string) {
    return educationStatusSeverity(value);
  }
}
