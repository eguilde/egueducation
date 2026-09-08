import { useCallback, useEffect, useState, type ChangeEvent } from 'react';
import { Button } from '@primereact/ui/button';
import { Card } from '@primereact/ui/card';
import { DataTable } from '@primereact/ui/datatable';
import { Dialog } from '@primereact/ui/dialog';
import { InputText } from '@primereact/ui/inputtext';
import { Message } from '@primereact/ui/message';
import { ProgressSpinner } from '@primereact/ui/progressspinner';
import { Select } from '@primereact/ui/select';
import { Tabs, type TabsRootChangeEvent } from '@primereact/ui/tabs';
import { Tag } from '@primereact/ui/tag';
import { Textarea } from '@primereact/ui/textarea';
import { Eye, Refresh, SortAlt } from '@primeicons/react';
import type { SelectValueChangeEvent } from 'primereact/select';
import { calendarDateLabel, canonicalStatus, statusLabel } from '../registratura/workflow';
import { createWorkflowApi, type FluxDocument, type FluxPage, type FluxQuery, type FluxStats, type WorkflowApi, type WorkflowAssignees } from './api';

type TabName = 'queue' | 'mapa' | 'pipeline';
type ViewState = { page: FluxPage; loading: boolean; error?: string };
type Filters = Pick<FluxQuery, 'nr_doc' | 'continut' | 'emitent' | 'compartiment' | 'tip'>;
const blankPage = (): FluxPage => ({ items: [], total: 0, page: 1, pageSize: 20 });
const initialViews = (): Record<TabName, ViewState> => ({ queue: { page: blankPage(), loading: true }, mapa: { page: blankPage(), loading: false }, pipeline: { page: blankPage(), loading: false } });
const initialFilters = (): Record<TabName, Filters> => ({ queue: {}, mapa: {}, pipeline: {} });
const defaultQuery = (): Record<TabName, FluxQuery> => ({
  queue: { page: 1, pageSize: 20, sort: 'registered_at', direction: 'desc' },
  mapa: { page: 1, pageSize: 20, sort: 'registered_at', direction: 'desc', mapa_filter: 'mine' },
  pipeline: { page: 1, pageSize: 20, sort: 'registered_at', direction: 'desc' },
});
const Spinner = () => <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;
const actionLabels: Record<string, string> = { assign_department: 'Alocă compartiment', claim: 'Preia document', assign_user: 'Atribuie utilizator', send_for_approval: 'Trimite la aprobare', approve: 'Aprobă', reject: 'Respinge' };
const actionsFor = (status: string) => ({ INCOMING: ['assign_department'], ALOCAT_COMPARTIMENT: ['claim', 'assign_user'], IN_LUCRU: ['assign_user', 'send_for_approval'], FLUX_APROBARE: ['approve', 'reject'] }[canonicalStatus(status)] ?? []);
const severityFor = (status?: string) => ({ FINALIZAT: 'success', IN_LUCRU: 'info', FLUX_APROBARE: 'warn', ANULAT: 'danger' }[canonicalStatus(status ?? '')] ?? 'secondary') as 'success' | 'info' | 'warn' | 'danger' | 'secondary';

export function WorkflowWorkspace({ api = createWorkflowApi(), canTransition = false, canManage = false }: { api?: WorkflowApi; canTransition?: boolean; canManage?: boolean }) {
  const [active, setActive] = useState<TabName>('queue');
  const [queries, setQueries] = useState(defaultQuery);
  const [filters, setFilters] = useState(initialFilters);
  const [views, setViews] = useState(initialViews);
  const [stats, setStats] = useState<FluxStats>([]);
  const [showArchived, setShowArchived] = useState(false);
  const [selected, setSelected] = useState<FluxDocument>();
  const [assignees, setAssignees] = useState<WorkflowAssignees>();
  const [action, setAction] = useState<string>();
  const [assigneeId, setAssigneeId] = useState('');
  const [comment, setComment] = useState('');
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState<string>();

  const load = useCallback(async (tab: TabName, override?: FluxQuery) => {
    const query = override ?? queries[tab];
    setViews((current) => ({ ...current, [tab]: { ...current[tab], loading: true, error: undefined } }));
    try {
      const page = await (tab === 'queue' ? api.queue(query) : tab === 'mapa' ? api.mapa(query) : api.pipeline(query));
      setViews((current) => ({ ...current, [tab]: { page, loading: false } }));
    } catch {
      setViews((current) => ({ ...current, [tab]: { ...current[tab], loading: false, error: 'Lista nu a putut fi încărcată. Reîncercați.' } }));
    }
  }, [api, queries]);
  const loadStats = useCallback(async () => { try { setStats(await api.pipelineStats()); } catch { setStats([]); } }, [api]);
  useEffect(() => { void load(active); if (active === 'pipeline') void loadStats(); }, [active, load, loadStats]);

  const updateQuery = (tab: TabName, patch: Partial<FluxQuery>) => {
    const next = { ...queries[tab], ...patch, page: patch.page ?? 1, ...filters[tab] };
    setQueries((current) => ({ ...current, [tab]: next }));
    void load(tab, next);
  };
  const updateFilter = (tab: TabName, name: keyof Filters, value: string) => {
    const nextFilters = { ...filters[tab], [name]: value };
    setFilters((current) => ({ ...current, [tab]: nextFilters }));
    const next = { ...queries[tab], ...nextFilters, page: 1 };
    setQueries((current) => ({ ...current, [tab]: next }));
    void load(tab, next);
  };
  const open = async (document: FluxDocument) => {
    setSelected(document); setAction(undefined); setAssigneeId(''); setComment('');
    if (canTransition && !assignees) { try { setAssignees(await api.assignees()); } catch { setAssignees({ departments: [], users: [] }); } }
  };
  const applyAction = async () => {
    if (!selected || !action) return;
    const expectedVersion = selected.workflow_version;
    if (!expectedVersion || expectedVersion < 1) {
      setNotice('Versiunea documentului lipsește. Reîncărcați lista înainte de a aplica acțiunea.');
      return;
    }
    if (action === 'reject' && comment.trim().length < 10) return;
    if (['assign_department', 'assign_user', 'send_for_approval'].includes(action) && !assigneeId) return;
    setSaving(true);
    try {
      const target = action === 'assign_department'
        ? { department_id: assigneeId }
        : ['assign_user', 'send_for_approval'].includes(action)
          ? { user_id: assigneeId }
          : {};
      await api.transition(String(selected.id), {
        action: action as 'assign_department' | 'assign_user' | 'claim' | 'send_for_approval' | 'approve' | 'reject',
        expected_version: expectedVersion,
        ...target,
        ...(comment.trim() ? { note: comment.trim() } : {}),
      });
      setNotice('Acțiunea fluxului a fost aplicată.'); setSelected(undefined); setAction(undefined);
      await Promise.all([load('queue'), load('mapa'), load('pipeline'), loadStats()]);
    } catch { setNotice('Acțiunea a fost refuzată de server sau documentul a fost modificat între timp.'); }
    finally { setSaving(false); }
  };
  const pipelineStats = showArchived ? stats : stats.filter((item) => !['FINALIZAT', 'ANULAT'].includes(canonicalStatus(item.status ?? '')));

  return <section aria-label="Flux documente" className="flex min-h-0 flex-1 flex-col gap-3">
    {notice && <Message.Root severity="info"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}
    <Tabs.Root value={active} onValueChange={(event: TabsRootChangeEvent) => setActive((String(event.value) as TabName) ?? 'queue')}>
      <Tabs.List aria-label="Vederi flux documente"><Tabs.Tab value="queue">Coada mea</Tabs.Tab>{canTransition && <Tabs.Tab value="mapa">Mapă semnături</Tabs.Tab>}{canManage && <Tabs.Tab value="pipeline">Evidență completă</Tabs.Tab>}<Tabs.Indicator /></Tabs.List>
      <Tabs.Panel value="queue"><FluxTable tab="queue" title="Coada mea" view={views.queue} query={queries.queue} filters={filters.queue} onFilter={updateFilter} onQuery={updateQuery} onOpen={open} onReload={() => void load('queue')} /></Tabs.Panel>
      {canTransition && <Tabs.Panel value="mapa"><div className="mt-3 flex flex-wrap gap-2"><MapaFilter active={queries.mapa.mapa_filter ?? 'mine'} onChange={(value) => updateQuery('mapa', { mapa_filter: value === 'all' ? undefined : value })} /></div><FluxTable tab="mapa" title="Mapă semnături" view={views.mapa} query={queries.mapa} filters={filters.mapa} onFilter={updateFilter} onQuery={updateQuery} onOpen={open} onReload={() => void load('mapa')} /></Tabs.Panel>}
      {canManage && <Tabs.Panel value="pipeline"><div className="mt-3 flex flex-wrap items-center gap-2">{pipelineStats.map((item) => <Button key={item.status} size="small" aria-label={`Filtrează status ${statusLabel(item.status ?? '')}`} variant={queries.pipeline.status === item.status ? undefined : 'outlined'} severity="secondary" onClick={() => updateQuery('pipeline', { status: queries.pipeline.status === item.status ? undefined : item.status })}><Tag value={statusLabel(item.status ?? '')} severity={severityFor(item.status)} /> {item.count ?? 0}</Button>)}<Button size="small" variant="outlined" severity="secondary" onClick={() => setShowArchived((value) => !value)}>{showArchived ? 'Ascunde arhivate' : 'Arată arhivate'}</Button></div><FluxTable tab="pipeline" title="Evidență completă" view={views.pipeline} query={queries.pipeline} filters={filters.pipeline} onFilter={updateFilter} onQuery={updateQuery} onOpen={open} onReload={() => { void load('pipeline'); void loadStats(); }} /></Tabs.Panel>}
    </Tabs.Root>
    <DocumentPanel document={selected} assignees={assignees} canTransition={canTransition} action={action} assigneeId={assigneeId} comment={comment} saving={saving} onAction={setAction} onAssignee={setAssigneeId} onComment={setComment} onClose={() => setSelected(undefined)} onApply={() => void applyAction()} />
  </section>;
}

function MapaFilter({ active, onChange }: { active: 'mine' | 'peers' | 'all'; onChange: (value: 'mine' | 'peers' | 'all') => void }) { return <>{([['mine', 'Adresate mie'], ['peers', 'De la colegi'], ['all', 'Toate']] as const).map(([value, label]) => <Button key={value} size="small" severity="secondary" variant={active === value ? undefined : 'outlined'} onClick={() => onChange(value)}>{label}</Button>)}</>; }

function FluxTable({ tab, title, view, query, filters, onFilter, onQuery, onOpen, onReload }: { tab: TabName; title: string; view: ViewState; query: FluxQuery; filters: Filters; onFilter: (tab: TabName, name: keyof Filters, value: string) => void; onQuery: (tab: TabName, value: Partial<FluxQuery>) => void; onOpen: (document: FluxDocument) => void; onReload: () => void }) {
  const sort = (field: string) => onQuery(tab, { sort: field, direction: query.sort === field && query.direction === 'asc' ? 'desc' : 'asc' });
  const page = view.page; const first = page.total === 0 ? 0 : (page.page - 1) * page.pageSize + 1; const last = Math.min(page.total, page.page * page.pageSize);
  const input = (name: keyof Filters, label: string, placeholder: string) => <InputText aria-label={label} value={filters[name] ?? ''} onChange={(event: ChangeEvent<HTMLInputElement>) => onFilter(tab, name, event.target.value)} placeholder={placeholder} />;
  const header = (label: string, field: string) => <Button variant="text" severity="secondary" size="small" aria-label={`Sortează după ${label}`} onClick={() => sort(field)}>{label}<SortAlt /></Button>;
  return <Card.Root className="mt-3"><Card.Body><Card.Content><div className="mb-2 flex items-center justify-between gap-2"><h2>{title}</h2><Button size="small" variant="text" severity="secondary" aria-label={`Reîncarcă ${title}`} onClick={onReload}><Refresh /></Button></div>{view.error && <Message.Root severity="error"><Message.Content><Message.Text>{view.error}</Message.Text></Message.Content></Message.Root>}{view.loading ? <div className="flex justify-center p-8"><Spinner /></div> : <div className="overflow-x-auto"><DataTable.Root data={page.items as Record<string, unknown>[]} dataKey="id" scrollable><DataTable.Table><DataTable.THead><DataTable.THeadRow><DataTable.THeadCell>{header('Nr. document', 'registry_number')}</DataTable.THeadCell><DataTable.THeadCell>{header('Dată', 'entry_at')}</DataTable.THeadCell><DataTable.THeadCell>{header('Conținut', 'subject')}</DataTable.THeadCell><DataTable.THeadCell>{header('Emitent', 'correspondent')}</DataTable.THeadCell><DataTable.THeadCell>{header('Compartiment', 'department_name')}</DataTable.THeadCell><DataTable.THeadCell>{header('Tip', 'document_type')}</DataTable.THeadCell><DataTable.THeadCell>Status</DataTable.THeadCell><DataTable.THeadCell>Acțiuni</DataTable.THeadCell></DataTable.THeadRow><DataTable.THeadRow><DataTable.THeadCell>{input('nr_doc', 'Filtru Nr. document', 'Nr…')}</DataTable.THeadCell><DataTable.THeadCell /><DataTable.THeadCell>{input('continut', 'Filtru Conținut', 'Conținut…')}</DataTable.THeadCell><DataTable.THeadCell>{input('emitent', 'Filtru Emitent', 'Emitent…')}</DataTable.THeadCell><DataTable.THeadCell>{input('compartiment', 'Filtru Compartiment', 'Compartiment…')}</DataTable.THeadCell><DataTable.THeadCell>{input('tip', 'Filtru Tip', 'Tip…')}</DataTable.THeadCell><DataTable.THeadCell /><DataTable.THeadCell /></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const document = item as unknown as FluxDocument; return <DataTable.Row key={String(document.id)} index={index}><DataTable.Cell>{document.registry_number ?? '—'}</DataTable.Cell><DataTable.Cell>{calendarDateLabel(document.entry_at ?? document.registered_at)}</DataTable.Cell><DataTable.Cell><span title={document.subject}>{document.subject ?? '—'}</span></DataTable.Cell><DataTable.Cell>{document.correspondent ?? '—'}</DataTable.Cell><DataTable.Cell>{document.department_name ?? '—'}</DataTable.Cell><DataTable.Cell>{document.document_type ?? '—'}</DataTable.Cell><DataTable.Cell><Tag value={statusLabel(document.status ?? '')} severity={severityFor(document.status)} /></DataTable.Cell><DataTable.Cell><Button variant="text" rounded iconOnly aria-label={`Deschide ${document.registry_number ?? document.id}`} title="Deschide" onClick={() => onOpen(document)}><Eye /></Button></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>{page.items.length === 0 && <Message.Root severity="info"><Message.Content><Message.Text>Niciun document pentru criteriile selectate.</Message.Text></Message.Content></Message.Root>}</div>}<div className="mt-3 flex flex-wrap items-center justify-between gap-2" aria-label={`Paginare ${title}`}><span>{first}–{last} din {page.total}</span><div className="flex gap-2"><Button size="small" variant="outlined" disabled={view.loading || page.page <= 1} onClick={() => onQuery(tab, { page: page.page - 1 })}>Anterior</Button><Button size="small" variant="outlined" disabled={view.loading || last >= page.total} onClick={() => onQuery(tab, { page: page.page + 1 })}>Următor</Button></div></div></Card.Content></Card.Body></Card.Root>;
}

function DocumentPanel({ document, assignees, canTransition, action, assigneeId, comment, saving, onAction, onAssignee, onComment, onClose, onApply }: { document?: FluxDocument; assignees?: WorkflowAssignees; canTransition: boolean; action?: string; assigneeId: string; comment: string; saving: boolean; onAction: (value?: string) => void; onAssignee: (value: string) => void; onComment: (value: string) => void; onClose: () => void; onApply: () => void }) {
  const targetKind = action === 'assign_department' ? 'Compartiment' : action === 'assign_user' || action === 'send_for_approval' ? 'Utilizator / aprobator' : undefined;
  const options = action === 'assign_department' ? (assignees?.departments ?? []) : assignees?.users ?? [];
  const enabled = Boolean(action) && (action !== 'reject' || comment.trim().length >= 10) && (!targetKind || Boolean(assigneeId));
  return <Dialog.Root open={Boolean(document)} onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Flux document {document?.registry_number}</Dialog.Title><Dialog.Close aria-label="Închide flux document" /></Dialog.Header><Dialog.Content>{document && <div className="flex flex-col gap-3"><dl className="grid gap-2 sm:grid-cols-2"><div><dt>Conținut</dt><dd>{document.subject ?? '—'}</dd></div><div><dt>Status</dt><dd><Tag value={statusLabel(document.status ?? '')} severity={severityFor(document.status)} /></dd></div><div><dt>Compartiment</dt><dd>{document.department_name ?? '—'}</dd></div><div><dt>Responsabil</dt><dd>{document.assigned_user_name ?? '—'}</dd></div><div><dt>Aprobator</dt><dd>{document.target_approver_name ?? '—'}</dd></div><div><dt>Respingeri</dt><dd>{document.rejection_count ?? 0}</dd></div></dl>{canTransition ? <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-3"><strong>Acțiune</strong><div className="flex flex-wrap gap-2">{actionsFor(document.status ?? '').map((entry) => <Button key={entry} size="small" variant={action === entry ? undefined : 'outlined'} onClick={() => { onAction(entry); onAssignee(''); onComment(''); }}>{actionLabels[entry]}</Button>)}</div>{action && <>{targetKind && <Select.Root value={assigneeId || null} options={options} optionLabel="name" optionValue="id" onValueChange={(event: SelectValueChangeEvent) => onAssignee(String(event.value ?? ''))}><Select.Trigger aria-label={targetKind}><Select.Value placeholder={`Selectați ${targetKind.toLocaleLowerCase('ro-RO')}`} /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>}{action === 'reject' && <Textarea aria-label="Motiv respingere" value={comment} placeholder="Motiv respingere (minim 10 caractere)" onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onComment(event.target.value)} />}{action !== 'reject' && <Textarea aria-label="Notă flux" value={comment} placeholder="Notă (opțional)" onChange={(event: ChangeEvent<HTMLTextAreaElement>) => onComment(event.target.value)} />}<Button disabled={saving || !enabled} onClick={onApply}>{saving ? 'Se aplică…' : actionLabels[action]}</Button></>}</div></Card.Content></Card.Body></Card.Root> : <Message.Root severity="info"><Message.Content><Message.Text>Nu aveți dreptul de a aplica tranziții. Documentul rămâne disponibil numai pentru consultare.</Message.Text></Message.Content></Message.Root>}</div>}</Dialog.Content><Dialog.Footer><Button variant="outlined" onClick={onClose}>Închide</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}
