import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { NgOptimizedImage } from '@angular/common';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatIconModule } from '@angular/material/icon';

import { ThemeService } from '../../core/ui/theme.service';

@Component({
  selector: 'app-auth-shell',
  imports: [NgOptimizedImage, TranslocoPipe, MatButtonModule, MatButtonToggleModule, MatIconModule],
  templateUrl: './auth-shell.component.html',
  styleUrl: './auth-shell.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthShellComponent {
  protected readonly theme = inject(ThemeService);

  readonly eyebrow = input<string>('');
  readonly title = input.required<string>();
  readonly subtitle = input<string>('');
  readonly visualKicker = input<string>('eGuilde Identity');
  readonly visualTitle = input.required<string>();
  readonly visualBody = input.required<string>();
  readonly accent = input<'rose' | 'amber'>('rose');

  protected readonly accentClass = computed(() =>
    this.accent() === 'amber' ? 'auth-shell--accent-amber' : 'auth-shell--accent-rose',
  );
}
