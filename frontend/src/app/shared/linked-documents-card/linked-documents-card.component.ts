import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { toObservable, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { TranslocoPipe, TranslocoService } from '@jsverse/transloco';
import { combineLatest, of } from 'rxjs';
import { debounceTime, distinctUntilChanged, startWith, switchMap } from 'rxjs/operators';

import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

import { DocumentLinksApiService } from '../../core/api/document-links-api.service';

@Component({
  selector: 'app-linked-documents-card',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    TranslocoPipe,
    MatButtonModule,
    MatCardModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatListModule,
    MatSelectModule,
    MatSnackBarModule,
  ],
  templateUrl: './linked-documents-card.component.html',
  styleUrl: './linked-documents-card.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LinkedDocumentsCardComponent {
  private readonly api = inject(DocumentLinksApiService);
  private readonly fb = inject(FormBuilder);
  private readonly transloco = inject(TranslocoService);
  private readonly snackBar = inject(MatSnackBar);

  readonly sourceModule = input.required<string>();
  readonly sourceRecordId = input<string | null>(null);

  protected readonly refreshToken = signal(0);

  protected readonly form = this.fb.group({
    query: this.fb.nonNullable.control(''),
    document_id: this.fb.nonNullable.control('', [Validators.required]),
    relation_type: this.fb.nonNullable.control('supporting', [Validators.required]),
  });

  private readonly sourceState = computed(() => ({
    sourceModule: this.sourceModule(),
    sourceRecordId: this.sourceRecordId(),
    refreshToken: this.refreshToken(),
  }));

  protected readonly links = toSignal(
    toObservable(this.sourceState).pipe(
      switchMap((state) =>
        state.sourceRecordId ? this.api.listLinks(state.sourceModule, state.sourceRecordId) : of([]),
      ),
    ),
    { initialValue: [] },
  );

  protected readonly lookupResults = toSignal(
    combineLatest([
      this.form.controls.query.valueChanges.pipe(startWith(this.form.controls.query.getRawValue()), debounceTime(250), distinctUntilChanged()),
      toObservable(this.sourceRecordId),
    ]).pipe(
      switchMap(([query, sourceRecordId]) =>
        sourceRecordId && query.trim().length >= 2 ? this.api.lookupDocuments(query.trim()) : of([]),
      ),
    ),
    { initialValue: [] },
  );

  protected readonly canLink = computed(() => !!this.sourceRecordId());

  protected createLink(): void {
    const sourceRecordId = this.sourceRecordId();
    if (!sourceRecordId || this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    const raw = this.form.getRawValue();
    this.api
      .createLink({
        document_id: raw.document_id,
        source_module: this.sourceModule(),
        source_record_id: sourceRecordId,
        relation_type: raw.relation_type,
      })
      .subscribe({
        next: () => {
          this.snackBar.open(
            this.transloco.translate('links.messages.created'),
            this.transloco.translate('common.close'),
            { duration: 3000 },
          );
          this.form.patchValue({ document_id: '', query: '', relation_type: 'supporting' });
          this.refreshToken.update((value) => value + 1);
        },
        error: () => {
          this.snackBar.open(
            this.transloco.translate('links.messages.createFailed'),
            this.transloco.translate('common.close'),
            { duration: 4000 },
          );
        },
      });
  }

  protected deleteLink(linkId: string): void {
    this.api.deleteLink(linkId).subscribe({
      next: () => {
        this.snackBar.open(
          this.transloco.translate('links.messages.deleted'),
          this.transloco.translate('common.close'),
          { duration: 3000 },
        );
        this.refreshToken.update((value) => value + 1);
      },
      error: () => {
        this.snackBar.open(
          this.transloco.translate('links.messages.deleteFailed'),
          this.transloco.translate('common.close'),
          { duration: 4000 },
        );
      },
    });
  }
}
