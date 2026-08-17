import { useCallback, useEffect, useState, type ChangeEvent } from 'react';
import { Button } from '@primereact/ui/button';
import { Card } from '@primereact/ui/card';
import { Dialog } from '@primereact/ui/dialog';
import { InputText } from '@primereact/ui/inputtext';
import { Message } from '@primereact/ui/message';
import { ProgressSpinner } from '@primereact/ui/progressspinner';
import { Select } from '@primereact/ui/select';
import { Tag } from '@primereact/ui/tag';
import { Textarea } from '@primereact/ui/textarea';
import type { SelectValueChangeEvent } from 'primereact/select';
import type { ArchiveApi, ArchiveClassificationField, ArchiveClassificationReview, ArchiveClassificationReviewFilterState, ArchiveFinalClassification } from './api';

const PAGE_SIZE = 25;
const Spinner = () => <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;

const suggestedClassification = (review: ArchiveClassificationReview): ArchiveFinalClassification => ({
  category: review.suggestion.category.value ?? '', fond: review.suggestion.fond.value ?? '', series: review.suggestion.series.value ?? '',
  document_type: review.suggestion.document_type.value ?? '', document_date: review.suggestion.document_date.value ?? '', document_number: review.suggestion.document_number.value ?? '',
});

const fields: Array<[keyof ArchiveFinalClassification, string]> = [
  ['category', 'Categorie'], ['fond', 'Fond'], ['series', 'Serie'], ['document_type', 'Tip document'], ['document_date', 'Data documentului'], ['document_number', 'Număr document'],
];

function FieldEvidence({ label, field }: { label: string; field: ArchiveClassificationField }) {
  return <div className="flex flex-col gap-1"><strong>{label}</strong><span>{field.value || 'Nedeterminat'}</span><small>Sursă: {field.source} · încredere: {Math.round(field.confidence * 100)}%</small>{field.evidence && <small>Indiciu: {field.evidence}</small>}</div>;
}

export function ClassificationReviewPanel({ api, canReview }: { api: ArchiveApi; canReview: boolean }) {
  const [state, setState] = useState<ArchiveClassificationReviewFilterState>('pending_review');
  const [reviews, setReviews] = useState<ArchiveClassificationReview[]>([]);
  const [total, setTotal] = useState(0); const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false); const [error, setError] = useState<string>();
  const [selected, setSelected] = useState<ArchiveClassificationReview>();
  const [draft, setDraft] = useState<ArchiveFinalClassification>(); const [note, setNote] = useState(''); const [saving, setSaving] = useState(false);
  const load = useCallback(async () => {
    if (!canReview) return;
    setLoading(true); setError(undefined);
    try { const result = await api.classificationReviews({ state, page: String(page), pageSize: String(PAGE_SIZE) }); setReviews(result.items); setTotal(result.total); }
    catch { setError('Revizuirile de clasificare nu au putut fi încărcate. Reîncercați.'); }
    finally { setLoading(false); }
  }, [api, canReview, page, state]);
  useEffect(() => { if (canReview) void load(); }, [canReview, load]);
  if (!canReview) return null;
  const openCorrection = (review: ArchiveClassificationReview) => { setSelected(review); setDraft(suggestedClassification(review)); setNote(''); };
  const close = () => { if (!saving) { setSelected(undefined); setDraft(undefined); setNote(''); } };
  const approve = async (review: ArchiveClassificationReview) => {
    setSaving(true); setError(undefined);
    try { await api.approveClassificationReview(review.id, { revision: review.revision, note: '' }); await load(); }
    catch { setError('Aprobarea nu a fost salvată. Clasificarea poate fi deja revizuită de alt utilizator; reîncărcați lista.'); }
    finally { setSaving(false); }
  };
  const correct = async () => {
    if (!selected || !draft || !draft.category.trim() || !draft.fond.trim() || !draft.series.trim() || !draft.document_type.trim()) return;
    setSaving(true); setError(undefined);
    try { await api.correctClassificationReview(selected.id, { revision: selected.revision, note: note.trim(), classification: draft }); close(); await load(); }
    catch { setError('Corectarea nu a fost salvată. Clasificarea poate fi deja revizuită de alt utilizator; reîncărcați lista.'); }
    finally { setSaving(false); }
  };
  const updateDraft = (field: keyof ArchiveFinalClassification, value: string) => setDraft((current) => current ? { ...current, [field]: value } : current);
  return <Card.Root><Card.Body><Card.Title>Revizuire clasificări OCR</Card.Title><Card.Content><div className="flex flex-col gap-4">
    <div className="flex flex-wrap items-center gap-2"><Select.Root value={state} options={[{ label: 'În așteptare', value: 'pending_review' }, { label: 'Necesită atenție', value: 'needs_review' }]} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => { setState(event.value as ArchiveClassificationReviewFilterState); setPage(1); }}><Select.Trigger aria-label="Stare revizuiri clasificare"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button variant="outlined" disabled={loading || saving} onClick={() => void load()}>Actualizează</Button></div>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {loading ? <div className="flex justify-center p-6"><Spinner /></div> : reviews.length === 0 ? <Message.Root severity="info"><Message.Content><Message.Text>Nu există clasificări care necesită revizuire pentru filtrul ales.</Message.Text></Message.Content></Message.Root> : <div className="flex flex-col gap-3">{reviews.map((review) => <Card.Root key={review.id}><Card.Body><Card.Content><div className="flex flex-col gap-3"><div className="flex flex-wrap items-center justify-between gap-2"><div><strong>Document {review.document_id}</strong><div><small>Versiune {review.version_id} · revizia {review.revision} · generat {review.generated_at}</small></div></div><div className="flex gap-2"><Tag value={review.state} severity={review.state === 'needs_review' ? 'warn' : 'info'} /><Tag value={`Încredere ${Math.round(review.suggestion_confidence * 100)}%`} /></div></div><div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3"><FieldEvidence label="Categorie" field={review.suggestion.category} /><FieldEvidence label="Fond" field={review.suggestion.fond} /><FieldEvidence label="Serie" field={review.suggestion.series} /><FieldEvidence label="Tip document" field={review.suggestion.document_type} /><FieldEvidence label="Data documentului" field={review.suggestion.document_date} /><FieldEvidence label="Număr document" field={review.suggestion.document_number} /></div><small>Regulă: {review.suggestion_source} · confirmare umană obligatorie</small><div className="flex flex-wrap gap-2"><Button disabled={saving} onClick={() => void approve(review)}>Aprobă propunerea</Button><Button variant="outlined" disabled={saving} onClick={() => openCorrection(review)}>Corectează</Button></div></div></Card.Content></Card.Body></Card.Root>)}</div>}
    <div className="flex items-center justify-between gap-2"><span>{total} revizuiri · pagina {page}</span><div className="flex gap-2"><Button size="small" variant="outlined" disabled={loading || page <= 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>Anterior</Button><Button size="small" variant="outlined" disabled={loading || reviews.length < PAGE_SIZE} onClick={() => setPage((value) => value + 1)}>Următor</Button></div></div>
  </div></Card.Content></Card.Body>
  <Dialog.Root open={Boolean(selected)} onOpenChange={(event: { value?: boolean }) => !event.value && close()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Corectează clasificarea OCR</Dialog.Title><Dialog.Close aria-label="Închide corectarea" /></Dialog.Header><Dialog.Content>{selected && draft && <div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Completați câmpurile structurale. Propunerea OCR este păstrată separat ca proveniență, iar corectarea este asociată reviziei {selected.revision}.</Message.Text></Message.Content></Message.Root><div className="grid gap-3 md:grid-cols-2">{fields.map(([key, label]) => <div key={key} className="flex flex-col gap-1"><label htmlFor={`classification-${key}`}>{label}{['category', 'fond', 'series', 'document_type'].includes(key) ? ' *' : ''}</label><InputText id={`classification-${key}`} aria-label={label} value={draft[key] ?? ''} onChange={(event: ChangeEvent<HTMLInputElement>) => updateDraft(key, event.target.value)} /></div>)}</div><label htmlFor="classification-note">Notă de revizuire</label><Textarea id="classification-note" aria-label="Notă de revizuire" value={note} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setNote(event.target.value)} /></div>}</Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" disabled={saving} onClick={close}>Renunță</Button><Button disabled={saving || !draft?.category.trim() || !draft.fond.trim() || !draft.series.trim() || !draft.document_type.trim()} onClick={() => void correct()}>{saving ? 'Se salvează…' : 'Salvează corectarea'}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
  </Card.Root>;
}
