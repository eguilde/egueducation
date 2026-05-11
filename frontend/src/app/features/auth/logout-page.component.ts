import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

import { AuthService } from '../../core/auth/auth.service';
import { AuthShellComponent } from '../../shared/auth-shell/auth-shell.component';

@Component({
  selector: 'app-logout-page',
  imports: [TranslocoPipe, AuthShellComponent, MatButtonModule, MatIconModule],
  templateUrl: './logout-page.component.html',
  styleUrl: './logout-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LogoutPageComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly transloco = inject(TranslocoService);

  protected readonly busy = signal(false);
  protected readonly error = signal<string | null>(null);

  protected async confirmLogout(): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    try {
      await this.auth.logout();
    } catch {
      this.error.set(this.transloco.translate('auth.logoutConfirm.error'));
      this.busy.set(false);
    }
  }

  protected async cancel(): Promise<void> {
    await this.router.navigateByUrl('/dashboard');
  }
}
