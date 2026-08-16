import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TranslocoPipe } from '@jsverse/transloco';
import { SelectButtonModule } from 'primeng/selectbutton';

import { ThemeService } from '../core/ui/theme.service';

@Component({
  selector: 'app-theme-panel',
  imports: [FormsModule, TranslocoPipe, SelectButtonModule],
  template: `
    <div
      class="flex flex-col gap-4 text-sm"
      role="group"
      [attr.aria-label]="'appearance.title' | transloco"
    >
      <fieldset class="m-0 flex flex-col gap-2 border-0 p-0">
        <legend class="font-medium" id="theme-mode-label">
          {{ 'appearance.mode' | transloco }}
        </legend>
        <p-selectbutton
          [options]="colorSchemeOptions"
          [ngModel]="theme.colorScheme()"
          optionLabel="icon"
          optionValue="value"
          size="small"
          aria-labelledby="theme-mode-label"
          (ngModelChange)="theme.setColorScheme($event)"
        >
          <ng-template #item let-item>
            <i [class]="item.icon" aria-hidden="true"></i>
          </ng-template>
        </p-selectbutton>
        <p class="m-0 text-xs leading-5 text-muted-color">
          Aura is the application theme; only its light and dark color schemes are selectable.
        </p>
      </fieldset>

      <div class="flex items-center justify-between gap-3 border-t border-surface pt-4">
        <span class="font-medium">{{ 'appearance.language' | transloco }}</span>
        <p-selectbutton
          [options]="languageOptions"
          [ngModel]="theme.language()"
          optionLabel="label"
          optionValue="value"
          size="small"
          (ngModelChange)="theme.setLanguage($event)"
        />
      </div>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ThemePanelComponent {
  protected readonly theme = inject(ThemeService);
  protected readonly colorSchemeOptions = [
    { icon: 'pi pi-desktop', value: 'system' },
    { icon: 'pi pi-sun', value: 'light' },
    { icon: 'pi pi-moon', value: 'dark' },
  ];
  protected readonly languageOptions = [
    { label: 'RO', value: 'ro' },
    { label: 'EN', value: 'en' },
  ];
}
