import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslocoPipe } from '@jsverse/transloco';

import { MatCardModule } from '@angular/material/card';

@Component({
  selector: 'app-feature-placeholder-page',
  standalone: true,
  imports: [TranslocoPipe, MatCardModule],
  template: `
    <mat-card appearance="outlined" class="placeholder">
      <h1>{{ route.snapshot.data['titleKey'] | transloco }}</h1>
      <p>{{ route.snapshot.data['descriptionKey'] | transloco }}</p>
    </mat-card>
  `,
  styles: `
    .placeholder {
      padding: 1.5rem;
      border-radius: 1.5rem;
      background: var(--surface-card);
    }

    .placeholder h1,
    .placeholder p {
      margin: 0;
    }

    .placeholder p {
      color: var(--text-muted);
      margin-top: 0.75rem;
    }
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FeaturePlaceholderPageComponent {
  protected readonly route = inject(ActivatedRoute);
}
