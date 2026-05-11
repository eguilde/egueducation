import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

import { AuthShellComponent } from '../../shared/auth-shell/auth-shell.component';

@Component({
  selector: 'app-access-denied-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, AuthShellComponent, MatButtonModule, MatIconModule],
  templateUrl: './access-denied-page.component.html',
  styleUrl: './access-denied-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AccessDeniedPageComponent {
  private readonly transloco = inject(TranslocoService);

  protected readonly supportHint = computed(() =>
    this.transloco.translate('auth.accessDenied.supportHint'),
  );
}
