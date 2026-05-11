import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { AuthShellComponent } from '../../shared/auth-shell/auth-shell.component';

@Component({
  selector: 'app-logged-out-page',
  standalone: true,
  imports: [RouterLink, TranslocoPipe, AuthShellComponent, MatButtonModule, MatIconModule],
  templateUrl: './logged-out-page.component.html',
  styleUrl: './logged-out-page.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoggedOutPageComponent {}
