import { TitleCasePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TranslocoPipe } from '@jsverse/transloco';
import { ButtonModule } from 'primeng/button';
import { SelectButtonModule } from 'primeng/selectbutton';
import { TooltipModule } from 'primeng/tooltip';

import { ThemeService } from '../core/ui/theme.service';

@Component({
  selector: 'app-theme-panel',
  imports: [TitleCasePipe, FormsModule, TranslocoPipe, ButtonModule, SelectButtonModule, TooltipModule],
  template: `
    <div class="flex flex-col gap-3 text-sm" role="group" [attr.aria-label]="'appearance.title' | transloco">
      <fieldset class="m-0 flex flex-col gap-1.5 border-0 p-0">
        <legend class="font-medium" id="theme-mode-label">{{ 'appearance.mode' | transloco }}</legend>
        <p-selectbutton
          [options]="darkModeOptions"
          [ngModel]="theme.isDarkMode()"
          optionLabel="icon"
          optionValue="value"
          size="small"
          aria-labelledby="theme-mode-label"
          (ngModelChange)="theme.setDarkMode($event)"
        >
          <ng-template #item let-item>
            <i [class]="item.icon" aria-hidden="true"></i>
          </ng-template>
        </p-selectbutton>
      </fieldset>

      <fieldset class="m-0 flex flex-col gap-1.5 border-0 p-0">
        <legend class="font-medium" id="primary-color-label">{{ 'appearance.primary' | transloco }}</legend>
        <div class="flex flex-wrap gap-1" role="radiogroup" aria-labelledby="primary-color-label">
          @for (color of theme.primaryColors; track color.name) {
            <button
              type="button"
              role="radio"
              class="size-7 cursor-pointer rounded-full border-0 outline outline-1 outline-offset-1 transition-all"
              [class.outline-primary]="theme.selectedPrimaryColor() === color.name"
              [class.outline-transparent]="theme.selectedPrimaryColor() !== color.name"
              [style.backgroundColor]="color.palette['500']"
              [pTooltip]="color.name | titlecase"
              tooltipPosition="top"
              [attr.aria-checked]="theme.selectedPrimaryColor() === color.name"
              [attr.aria-label]="('appearance.primary' | transloco) + ': ' + (color.name | titlecase)"
              (click)="theme.setPrimaryColor(color.name)"
            ></button>
          }
        </div>
      </fieldset>

      <fieldset class="m-0 flex flex-col gap-1.5 border-0 p-0">
        <legend class="font-medium" id="surface-color-label">{{ 'appearance.surface' | transloco }}</legend>
        <div class="flex flex-wrap gap-1" role="radiogroup" aria-labelledby="surface-color-label">
          @for (surface of theme.surfaces; track surface.name) {
            <button
              type="button"
              role="radio"
              class="size-7 cursor-pointer rounded-full border-0 outline outline-1 outline-offset-1 transition-all"
              [class.outline-primary]="theme.selectedSurface() === surface.name"
              [class.outline-transparent]="theme.selectedSurface() !== surface.name"
              [style.backgroundColor]="surface.palette['500']"
              [pTooltip]="surface.name | titlecase"
              tooltipPosition="top"
              [attr.aria-checked]="theme.selectedSurface() === surface.name"
              [attr.aria-label]="('appearance.surface' | transloco) + ': ' + (surface.name | titlecase)"
              (click)="theme.setSurface(surface.name)"
            ></button>
          }
        </div>
      </fieldset>

      <div class="flex items-center justify-between gap-3 pt-1">
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
  protected readonly darkModeOptions = [
    { icon: 'pi pi-sun', value: false },
    { icon: 'pi pi-moon', value: true },
  ];
  protected readonly languageOptions = [
    { label: 'RO', value: 'ro' },
    { label: 'EN', value: 'en' },
  ];
}
