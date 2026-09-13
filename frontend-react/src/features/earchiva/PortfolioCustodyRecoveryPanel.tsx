import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from 'react';
import { Button } from '@primereact/ui/button';
import { Card } from '@primereact/ui/card';
import { DataTable } from '@primereact/ui/datatable';
import { Dialog } from '@primereact/ui/dialog';
import { InputText } from '@primereact/ui/inputtext';
import { Message } from '@primereact/ui/message';
import { ProgressSpinner } from '@primereact/ui/progressspinner';
import { Select } from '@primereact/ui/select';
import { Tag } from '@primereact/ui/tag';
import { Textarea } from '@primereact/ui/textarea';
import type { SelectValueChangeEvent } from 'primereact/select';
import type {
  ArchiveApi,
  PortfolioCustodyRecoveryDisposition,
  PortfolioCustodyRecoveryOperation,
  PortfolioCustodyRecoveryQuery,
  PortfolioCustodyRecoverySort,
  PortfolioCustodyRecoveryStatus,
  ReconcilePortfolioCustodyInput,
} from './api';

const PAGE_SIZES = [10, 20, 50];
const statusOptions: Array<{ label: string; value: PortfolioCustodyRecoveryStatus | '' }> = [
  { label: 'Toate stările', value: '' },
  { label: 'Necesită recuperare', value: 'stored' },
  { label: 'În coadă', value: 'queued' },
  { label: 'În procesare', value: 'leased' },
  { label: 'Finalizată', value: 'committed' },
  { label: 'Blocată', value: 'blocked' },
  { label: 'Eșuată definitiv', value: 'deadletter' },
];
const dispositionOptions: Array<{ label: string; value: PortfolioCustodyRecoveryDisposition | '' }> = [
  { label: 'Toate destinațiile', value: '' },
  { label: 'Acces profesor', value: 'teacher_access' },
  { label: 'Numai arhiva instituției', value: 'institution_archive_only' },
];
const emptyPage = { items: [] as PortfolioCustodyRecoveryOperation[], total: 0, page: 1, pageSize: 20 };
const initialQuery = (): PortfolioCustodyRecoveryQuery => ({ page: 1, pageSize: 20, sort: 'created_at', direction: 'desc' });
const isRunning = (status: PortfolioCustodyRecoveryStatus) => status === 'queued' || status === 'leased';
const statusSeverity = (status: PortfolioCustodyRecoveryStatus) => status === 'committed' ? 'success' : status === 'blocked' || status === 'deadletter' ? 'danger' : status === 'stored' ? 'warn' : 'info';
const dispositionLabel = (value?: PortfolioCustodyRecoveryDisposition | null) => value === 'teacher_access' ? 'Acces profesor' : value === 'institution_archive_only' ? 'Numai arhiva instituției' : 'Neselectată';
const displayDate = (value?: string) => value ? new Intl.DateTimeFormat('ro-RO', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '—';
const dateFilterValue = (value?: string) => value?.slice(0, 10) ?? '';
const dateBoundary = (value: string, end = false) => value ? `${value}T${end ? '23:59:59.999' : '00:00:00.000'}Z` : undefined;

export function PortfolioCustodyRecoveryPanel({ api }: { api: ArchiveApi }) {
  const [query, setQuery] = useState<PortfolioCustodyRecoveryQuery>(initialQuery);
  const [page, setPage] = useState(emptyPage);
  const [selected, setSelected] = useState<PortfolioCustodyRecoveryOperation>();
  const [tracking, setTracking] = useState<PortfolioCustodyRecoveryOperation>();
  const [detailOpen, setDetailOpen] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const sequence = useRef(0);

  const load = useCallback(async () => {
    const current = ++sequence.current;
    setLoading(true); setError(undefined);
    try {
      const result = await api.portfolioCustodyRecoveries(query);
      if (current === sequence.current) setPage(result);
    } catch {
      if (current === sequence.current) setError('Operațiile de recuperare a custodiei nu au putut fi încărcate.');
    } finally {
      if (current === sequence.current) setLoading(false);
    }
  }, [api, query]);
  useEffect(() => { void load(); return () => { sequence.current += 1; }; }, [load]);

  useEffect(() => {
    if (!tracking?.operation_id || !isRunning(tracking.status)) return;
    let active = true;
    const poll = async () => {
      try {
        const next = await api.portfolioCustodyRecovery(tracking.intent_id, tracking.operation_id!);
        if (!active) return;
        setTracking(next);
        if (!isRunning(next.status)) {
          setNotice(next.status === 'committed' ? 'Recuperarea a fost finalizată și verificată de server.' : 'Recuperarea s-a oprit; consultați starea și codul de eroare.');
          await load();
        }
      } catch { if (active) setError('Starea operației de recuperare nu a putut fi verificată.'); }
    };
    const timer = window.setInterval(() => void poll(), 2500);
    void poll();
    return () => { active = false; window.clearInterval(timer); };
  }, [api, load, tracking?.intent_id, tracking?.operation_id, tracking?.status]);

  const lastPage = Math.max(1, Math.ceil(page.total / page.pageSize));
  const update = (patch: Partial<PortfolioCustodyRecoveryQuery>) => setQuery((current) => ({ ...current, ...patch, page: 1 }));
  const sort = (field: PortfolioCustodyRecoverySort) => setQuery((current) => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === 'asc' ? 'desc' : 'asc' }));
  const submit = async (input: ReconcilePortfolioCustodyInput) => {
    if (!selected) return;
    setSaving(true); setError(undefined); setNotice(undefined);
    try {
      const operation = await api.reconcilePortfolioCustody(selected.intent_id, input);
      setSelected(undefined); setTracking(operation);
      setNotice('Cererea a fost acceptată în coada durabilă. Finalizarea este verificată separat.');
      await load();
    } catch {
      setError('Cererea a fost refuzată. Metadatele trebuie să coincidă exact cu încărcarea inițială, iar drepturile sunt reverificate în baza de date.');
    } finally { setSaving(false); }
  };
  const inspect = async (row: PortfolioCustodyRecoveryOperation) => {
    if (!row.operation_id) return;
    setTracking(row); setDetailOpen(true); setError(undefined);
    try { setTracking(await api.portfolioCustodyRecovery(row.intent_id, row.operation_id)); }
    catch { setError('Detaliile operației de recuperare nu au putut fi încărcate.'); }
  };

  return <Card.Root><Card.Body><Card.Title>Recuperare custodie portofoliu</Card.Title><Card.Subtitle>Adoptarea verificată a versiunilor PDF deja scrise în storage WORM, fără reducerea retenției sau a blocajelor juridice.</Card.Subtitle><Card.Content>
    <div className="flex min-h-0 flex-col gap-3">
      {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
      {notice && <Message.Root severity="info"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}
      {tracking && <Message.Root severity={tracking.status === 'committed' ? 'success' : isRunning(tracking.status) ? 'info' : 'warn'}><Message.Content><Message.Text>Operație {tracking.operation_id}: {tracking.status}. O acceptare HTTP 202 nu este prezentată ca finalizare.</Message.Text></Message.Content></Message.Root>}
      <div className="min-h-72 overflow-hidden rounded-border border border-surface">
        <DataTable.Root data={page.items as unknown as Record<string, unknown>[]} dataKey="intent_id" scrollable className="max-h-[min(62dvh,42rem)] min-h-64 overflow-auto">
          <DataTable.Table><DataTable.THead className="sticky top-0 z-10">
            <DataTable.THeadRow>
              <DataTable.THeadCell><SortButton label="Intent" field="intent_id" query={query} onSort={sort} /></DataTable.THeadCell>
              <DataTable.THeadCell><SortButton label="Portofoliu" field="portfolio_id" query={query} onSort={sort} /></DataTable.THeadCell>
              <DataTable.THeadCell><SortButton label="Titlu" field="title" query={query} onSort={sort} /></DataTable.THeadCell>
              <DataTable.THeadCell><SortButton label="Fișier original" field="original_file_name" query={query} onSort={sort} /></DataTable.THeadCell>
              <DataTable.THeadCell><SortButton label="Destinație" field="disposition" query={query} onSort={sort} /></DataTable.THeadCell>
              <DataTable.THeadCell><SortButton label="Stare" field="status" query={query} onSort={sort} /></DataTable.THeadCell>
              <DataTable.THeadCell frozen alignFrozen="right"><span className="flex items-center justify-between gap-2"><span>Acțiuni</span><Button iconOnly rounded size="small" variant="text" aria-label="Reîncarcă recuperările" title="Reîncarcă" disabled={loading} onClick={() => void load()}><i className="pi pi-refresh" aria-hidden="true" /></Button></span></DataTable.THeadCell>
            </DataTable.THeadRow>
            <DataTable.THeadRow>
              <DataTable.THeadCell><div className="flex min-w-44 flex-col gap-1"><InputText aria-label="Filtru intent" value={query.intent_id ?? ''} onChange={(event: ChangeEvent<HTMLInputElement>) => update({ intent_id: event.target.value || undefined })} /><InputText aria-label="Creat de la" type="date" value={dateFilterValue(query.created_from)} onChange={(event: ChangeEvent<HTMLInputElement>) => update({ created_from: dateBoundary(event.target.value) })} /><InputText aria-label="Creat până la" type="date" value={dateFilterValue(query.created_to)} onChange={(event: ChangeEvent<HTMLInputElement>) => update({ created_to: dateBoundary(event.target.value, true) })} /></div></DataTable.THeadCell>
              <DataTable.THeadCell><InputText aria-label="Filtru portofoliu" value={query.portfolio_id ?? ''} onChange={(event: ChangeEvent<HTMLInputElement>) => update({ portfolio_id: event.target.value || undefined })} /></DataTable.THeadCell>
              <DataTable.THeadCell><InputText aria-label="Filtru titlu recuperare" value={query.title ?? ''} onChange={(event: ChangeEvent<HTMLInputElement>) => update({ title: event.target.value || undefined })} /></DataTable.THeadCell>
              <DataTable.THeadCell><InputText aria-label="Filtru fișier original recuperare" value={query.original_file_name ?? ''} onChange={(event: ChangeEvent<HTMLInputElement>) => update({ original_file_name: event.target.value || undefined })} /></DataTable.THeadCell>
              <DataTable.THeadCell><SelectFilter aria="Filtru destinație recuperare" value={query.disposition ?? ''} options={dispositionOptions} onValue={(value) => update({ disposition: value as PortfolioCustodyRecoveryDisposition || undefined })} /></DataTable.THeadCell>
              <DataTable.THeadCell><SelectFilter aria="Filtru stare recuperare" value={query.status ?? ''} options={statusOptions} onValue={(value) => update({ status: value as PortfolioCustodyRecoveryStatus || undefined })} /></DataTable.THeadCell>
              <DataTable.THeadCell frozen alignFrozen="right" aria-label="Filtre acțiuni" />
            </DataTable.THeadRow>
          </DataTable.THead><DataTable.TBody>{({ item, index }) => { const row = item as unknown as PortfolioCustodyRecoveryOperation; return <DataTable.Row key={`${row.intent_id}:${row.operation_id ?? 'stored'}`} index={index}>
            <DataTable.Cell><span className="block max-w-44 truncate" title={row.intent_id}>{row.intent_id}</span><small>{displayDate(row.created_at)}</small></DataTable.Cell>
            <DataTable.Cell><span className="block max-w-44 truncate" title={row.portfolio_id}>{row.portfolio_id}</span></DataTable.Cell>
            <DataTable.Cell>{row.title || '—'}</DataTable.Cell><DataTable.Cell>{row.original_file_name || '—'}</DataTable.Cell>
            <DataTable.Cell>{dispositionLabel(row.disposition)}</DataTable.Cell><DataTable.Cell><Tag value={row.status} severity={statusSeverity(row.status)} />{row.last_error_code && <small className="block">{row.last_error_code}</small>}</DataTable.Cell>
            <DataTable.Cell frozen alignFrozen="right"><div className="flex justify-end gap-1">{(row.status === 'stored' || row.status === 'blocked' || row.status === 'deadletter') && <Button size="small" variant="text" aria-label={`Recuperează intent ${row.intent_id}`} onClick={() => setSelected(row)}>{row.status === 'stored' ? 'Recuperează' : 'Reîncearcă'}</Button>}{row.operation_id && <Button iconOnly rounded size="small" variant="text" aria-label={`Urmărește operația ${row.operation_id}`} title="Urmărește" onClick={() => void inspect(row)}><i className="pi pi-eye" aria-hidden="true" /></Button>}</div></DataTable.Cell>
          </DataTable.Row>; }}</DataTable.TBody></DataTable.Table>
        </DataTable.Root>
        {!loading && !page.items.length && <Message.Root severity="info"><Message.Content><Message.Text>Nu există versiuni în custodie pentru filtrele curente.</Message.Text></Message.Content></Message.Root>}
        {loading && !page.items.length && <div className="flex min-h-64 items-center justify-center" role="status" aria-label="Se încarcă recuperările"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div>}
      </div>
      <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2">
        <span>{page.total ? `${(page.page - 1) * page.pageSize + 1}–${Math.min(page.page * page.pageSize, page.total)} din ${page.total}` : '0 rezultate'}</span>
        <div className="flex items-center gap-2"><SelectFilter aria="Rezultate recuperare pe pagină" value={String(query.pageSize)} options={PAGE_SIZES.map((value) => ({ label: `${value}/pagină`, value: String(value) }))} onValue={(value) => setQuery((current) => ({ ...current, page: 1, pageSize: Number(value) }))} /><Button size="small" variant="outlined" disabled={loading || query.page <= 1} onClick={() => setQuery((current) => ({ ...current, page: current.page - 1 }))}>Anterior</Button><Button size="small" variant="outlined" disabled={loading || query.page >= lastPage} onClick={() => setQuery((current) => ({ ...current, page: current.page + 1 }))}>Următor</Button></div>
        <span>{loading ? 'Se actualizează…' : ''}</span>
      </div>
    </div>
    {selected && <RecoveryDialog value={selected} saving={saving} onClose={() => !saving && setSelected(undefined)} onSubmit={submit} />}
    {detailOpen && tracking && <OperationDetailDialog value={tracking} onClose={() => setDetailOpen(false)} />}
  </Card.Content></Card.Body></Card.Root>;
}

function SortButton({ label, field, query, onSort }: { label: string; field: PortfolioCustodyRecoverySort; query: PortfolioCustodyRecoveryQuery; onSort: (field: PortfolioCustodyRecoverySort) => void }) {
  return <Button size="small" variant="text" severity="secondary" aria-label={`Sortează după ${label}`} onClick={() => onSort(field)}>{label}{query.sort === field ? query.direction === 'asc' ? ' ↑' : ' ↓' : ''}</Button>;
}

function SelectFilter({ aria, value, options, onValue }: { aria: string; value: string; options: Array<{ label: string; value: string }>; onValue: (value: string) => void }) {
  return <Select.Root value={value} options={options} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => onValue(String(event.value ?? ''))}><Select.Trigger aria-label={aria}><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>;
}

function RecoveryDialog({ value, saving, onClose, onSubmit }: { value: PortfolioCustodyRecoveryOperation; saving: boolean; onClose: () => void; onSubmit: (input: ReconcilePortfolioCustodyInput) => Promise<void> }) {
  const [form, setForm] = useState<ReconcilePortfolioCustodyInput>({ reason: '', disposition: 'teacher_access', title: '', original_file_name: '', document_date: undefined });
  const valid = form.reason.trim().length >= 10 && form.title.trim().length > 0 && form.original_file_name.trim().length > 0;
  const set = <K extends keyof ReconcilePortfolioCustodyInput>(key: K, next: ReconcilePortfolioCustodyInput[K]) => setForm((current) => ({ ...current, [key]: next }));
  const warning = useMemo(() => form.disposition === 'teacher_access'
    ? 'Serverul reverifică titularul, starea portofoliului și dreptul profesorului. Numai după commit documentul devine accesibil în portofoliu.'
    : 'Documentul rămâne exclusiv în arhiva instituției. Nu se creează atașament sau drept de acces pentru profesor.', [form.disposition]);
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,44rem)]"><Dialog.Header><Dialog.Title>Recuperare verificată a custodiei</Dialog.Title><Dialog.Close aria-label="Închide recuperarea" disabled={saving} /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">
    <Message.Root severity="warn"><Message.Content><Message.Text>Storage-ul WORM și legal hold nu sunt modificate de această operație. Metadatele trebuie să reproducă exact încărcarea inițială.</Message.Text></Message.Content></Message.Root>
    <span>Intent: <strong>{value.intent_id}</strong></span>
    <label className="flex flex-col gap-1"><span>Destinație finală *</span><Select.Root value={form.disposition} options={dispositionOptions.filter((item) => item.value)} optionLabel="label" optionValue="value" disabled={saving} onValueChange={(event: SelectValueChangeEvent) => set('disposition', event.value as PortfolioCustodyRecoveryDisposition)}><Select.Trigger aria-label="Destinație finală recuperare"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>
    <Message.Root severity="info"><Message.Content><Message.Text>{warning}</Message.Text></Message.Content></Message.Root>
    <label className="flex flex-col gap-1"><span>Titlul inițial *</span><InputText aria-label="Titlul inițial al încărcării" value={form.title} disabled={saving} onChange={(event: ChangeEvent<HTMLInputElement>) => set('title', event.target.value)} /></label>
    <label className="flex flex-col gap-1"><span>Numele original al fișierului *</span><InputText aria-label="Numele original al fișierului" value={form.original_file_name} disabled={saving} onChange={(event: ChangeEvent<HTMLInputElement>) => set('original_file_name', event.target.value)} /></label>
    <label className="flex flex-col gap-1"><span>Data documentului</span><InputText aria-label="Data documentului recuperat" type="date" value={form.document_date ?? ''} disabled={saving} onChange={(event: ChangeEvent<HTMLInputElement>) => set('document_date', event.target.value || undefined)} /></label>
    <label className="flex flex-col gap-1"><span>Motiv documentat * (minimum 10 caractere)</span><Textarea aria-label="Motiv recuperare custodie" value={form.reason} disabled={saving} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => set('reason', event.target.value)} /></label>
  </div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !valid} onClick={() => void onSubmit({ ...form, reason: form.reason.trim(), title: form.title.trim(), original_file_name: form.original_file_name.trim() })}>{saving ? 'Se transmite…' : 'Trimite pentru recuperare'}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

function OperationDetailDialog({ value, onClose }: { value: PortfolioCustodyRecoveryOperation; onClose: () => void }) {
  const reason = 'reason' in value ? value.reason : undefined;
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,40rem)]"><Dialog.Header><Dialog.Title>Operație de recuperare</Dialog.Title><Dialog.Close aria-label="Închide detaliile recuperării" /></Dialog.Header><Dialog.Content><dl className="grid gap-2 sm:grid-cols-2"><div><dt>Operație</dt><dd className="break-all">{value.operation_id}</dd></div><div><dt>Intent</dt><dd className="break-all">{value.intent_id}</dd></div><div><dt>Portofoliu</dt><dd className="break-all">{value.portfolio_id}</dd></div><div><dt>Stare</dt><dd><Tag value={value.status} severity={statusSeverity(value.status)} /></dd></div><div><dt>Destinație</dt><dd>{dispositionLabel(value.disposition)}</dd></div><div><dt>Actualizat</dt><dd>{displayDate(value.updated_at)}</dd></div><div className="sm:col-span-2"><dt>Motiv documentat</dt><dd>{reason || '—'}</dd></div><div className="sm:col-span-2"><dt>Cod sigur de eroare</dt><dd>{value.last_error_code || '—'}</dd></div></dl></Dialog.Content><Dialog.Footer><Button variant="outlined" onClick={onClose}>Închide</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}
