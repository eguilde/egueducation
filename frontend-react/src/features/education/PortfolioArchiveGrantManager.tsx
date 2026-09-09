import { useCallback, useEffect, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import type { EducationApi, EducationListQuery, EducationPage, EligibleGovernanceUser, OwnPortfolioArchiveDocument, PortfolioAttachmentGrant } from "./types";

type QueryState = Required<Pick<EducationListQuery, "page" | "pageSize">> & Pick<EducationListQuery, "sort" | "direction"> & { filters: Record<string, string> };
const initialQuery = (): QueryState => ({ page: 1, pageSize: 20, filters: {} });

function Spinner() { return <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>; }

/** Server-side list state; a small debounce makes header filters practical. */
function usePagedRows<T extends { id: string }>(load: (query: EducationListQuery) => Promise<EducationPage<T>>) {
  const [query, setQuery] = useState<QueryState>(initialQuery);
  const [page, setPage] = useState<EducationPage<T>>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const requestId = useRef(0);
  const queryEffectReady = useRef(false);
  const refresh = useCallback(async (next = query) => {
    const id = ++requestId.current; setLoading(true); setError(undefined);
    try { const result = await load(next); if (id === requestId.current) setPage(result); }
    catch { if (id === requestId.current) setError("Datele nu au putut fi încărcate. Încercați din nou."); }
    finally { if (id === requestId.current) setLoading(false); }
  }, [load, query]);
  // The initial request must not be coupled to query changes; subsequent
  // filtering/sorting/paging goes through the debounced effect below.
  useEffect(() => { void refresh(initialQuery()); }, [load]);
  useEffect(() => {
    if (!queryEffectReady.current) { queryEffectReady.current = true; return; }
    const timeout = window.setTimeout(() => void refresh(), 300);
    return () => window.clearTimeout(timeout);
  }, [query]);
  const changeFilter = (field: string, value: string) => setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } }));
  const changeSort = (field: string) => setQuery((current) => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" }));
  return { page, query, loading, error, refresh, changeFilter, changeSort,
    changePage: (pageNumber: number) => setQuery((current) => ({ ...current, page: pageNumber })),
    changePageSize: (pageSize: number) => setQuery((current) => ({ ...current, page: 1, pageSize })),
  };
}

type Column<T> = { header: string; field?: string; action?: boolean; render: (row: T) => ReactNode };
function ServerTable<T extends { id: string }>({ label, state, columns, onAdd, addLabel, emptyMessage }: { label: string; state: ReturnType<typeof usePagedRows<T>>; columns: Column<T>[]; onAdd?: () => void; addLabel?: string; emptyMessage: string }) {
  const { page, query, loading, error, changeFilter, changeSort, changePage, changePageSize } = state;
  return <div className="flex min-h-0 flex-col gap-2" aria-label={label}>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    <div className="min-h-72 overflow-hidden rounded-border border border-surface">
      {loading && page.items.length === 0 ? <div className="flex min-h-72 items-center justify-center" role="status"><Spinner /></div> : <DataTable.Root data={page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(44dvh,32rem)] min-h-64 overflow-auto">
        <DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>{columns.map((column) => <DataTable.THeadCell key={column.header} frozen={column.action || undefined} alignFrozen={column.action ? "right" : undefined}>
          <div className={column.action ? "flex items-center justify-between gap-2" : "flex flex-col gap-1"}>
            {column.field ? <Button size="small" variant="text" severity="secondary" onClick={() => changeSort(column.field!)} aria-label={`Sortează după ${column.header}`}>{column.header}{query.sort === column.field ? query.direction === "asc" ? " ↑" : " ↓" : ""}</Button> : <span>{column.header}</span>}
            {column.action && onAdd && <Button iconOnly rounded size="small" aria-label={`Adaugă ${addLabel ?? "drept"}`} title={`Adaugă ${addLabel ?? "drept"}`} onClick={onAdd}><i className="pi pi-plus" aria-hidden="true" /></Button>}
            {column.field && <InputText aria-label={`Filtru ${column.header}`} className="w-full" placeholder="Filtru" value={query.filters[column.field] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => changeFilter(column.field!, event.target.value)} />}
          </div>
        </DataTable.THeadCell>)}</DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => {
          const row = item as T; return <DataTable.Row key={row.id} index={index}>{columns.map((column) => <DataTable.Cell key={column.header} frozen={column.action || undefined} alignFrozen={column.action ? "right" : undefined}>{column.render(row)}</DataTable.Cell>)}</DataTable.Row>;
        }}</DataTable.TBody></DataTable.Table>
      </DataTable.Root>}
    </div>
    {!loading && page.items.length === 0 && <Message.Root severity="info"><Message.Content><Message.Text>{emptyMessage}</Message.Text></Message.Content></Message.Root>}
    <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2" aria-label={`Paginare ${label}`}><span>{page.total ? `${(page.page - 1) * page.pageSize + 1} – ${Math.min(page.page * page.pageSize, page.total)} din ${page.total}` : "0 rezultate"}</span><div className="flex flex-wrap items-center gap-2">
      <Select.Root value={query.pageSize} options={[10, 20, 50, 100].map((value) => ({ label: String(value), value }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => changePageSize(Number(event.value))}><Select.Trigger aria-label={`Rânduri pe pagină ${label}`}><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>
      <Button size="small" variant="outlined" disabled={loading || page.page <= 1} onClick={() => changePage(page.page - 1)}>Anterior</Button><Button size="small" variant="outlined" disabled={loading || page.page * page.pageSize >= page.total} onClick={() => changePage(page.page + 1)}>Următor</Button>
    </div></div>
  </div>;
}

function GrantDialog({ api, onClose, onGranted }: { api: EducationApi; onClose: () => void; onGranted: () => Promise<void> }) {
  const [documentId, setDocumentId] = useState(""); const [userId, setUserId] = useState(""); const [saving, setSaving] = useState(false); const [error, setError] = useState<string>();
  const documents = usePagedRows(useCallback((query: EducationListQuery) => api.eligibleAttachmentDocuments(query), [api]));
  const users = usePagedRows(useCallback((query: EducationListQuery) => api.eligibleAttachmentUsers(query), [api]));
  const grant = async () => { if (!documentId || !userId) return; setSaving(true); setError(undefined); try { await api.createAttachmentGrant({ archive_document_id: documentId, grantee_user_id: userId }); await onGranted(); onClose(); } catch { setError("Dreptul nu a putut fi acordat. Verificați utilizatorul și starea documentului."); } finally { setSaving(false); } };
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,72rem)]"><Dialog.Header><Dialog.Title>Acordă acces la document eArhivă</Dialog.Title><Dialog.Close aria-label="Închide acordarea accesului" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-4">
    <Message.Root severity="info"><Message.Content><Message.Text>Selectați un document eligibil și un utilizator din instituția curentă. Listele sunt filtrate, sortate și paginate pe server.</Message.Text></Message.Content></Message.Root>{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    <div className="grid min-h-0 gap-4 lg:grid-cols-2"><ServerTable label="Documente eArhivă eligibile" state={documents} emptyMessage="Nu există documente eligibile." columns={[
      { header: "Titlu", field: "title", render: (document) => <strong>{(document as OwnPortfolioArchiveDocument).title}</strong> }, { header: "Versiune", field: "current_version_no", render: (document) => `v${(document as OwnPortfolioArchiveDocument).current_version_no}` }, { header: "Selectare", action: true, render: (document) => <Button size="small" variant={documentId === document.id ? undefined : "outlined"} onClick={() => setDocumentId(document.id)}>{documentId === document.id ? "Selectat" : "Selectează"}</Button> },
    ]} /><ServerTable label="Utilizatori eligibili" state={users} emptyMessage="Nu există utilizatori eligibili." columns={[
      { header: "Nume", field: "name", render: (user) => <strong>{(user as EligibleGovernanceUser).name}</strong> }, { header: "Selectare", action: true, render: (user) => <Button size="small" variant={userId === user.id ? undefined : "outlined"} onClick={() => setUserId(user.id)}>{userId === user.id ? "Selectat" : "Selectează"}</Button> },
    ]} /></div>
  </div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !documentId || !userId} onClick={() => void grant()}>{saving ? "Se salvează…" : "Acordă acces"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

/** Institution-administered, tenant-scoped grants for teacher portfolio evidence. */
export function PortfolioArchiveGrantManager({ api }: { api: EducationApi }) {
  const [grantOpen, setGrantOpen] = useState(false); const [pendingRevoke, setPendingRevoke] = useState<PortfolioAttachmentGrant>(); const [saving, setSaving] = useState(false); const [notice, setNotice] = useState<string>();
  const grants = usePagedRows(useCallback((query: EducationListQuery) => api.attachmentGrants(query), [api]));
  const revoke = async () => { if (!pendingRevoke) return; setSaving(true); setNotice(undefined); try { await api.deleteAttachmentGrant(pendingRevoke.id); setPendingRevoke(undefined); setNotice("Dreptul de atașare a fost revocat."); await grants.refresh(); } catch { setNotice("Dreptul nu poate fi revocat cât timp este folosit de o dovadă depusă."); } finally { setSaving(false); } };
  const afterGranted = async () => { setNotice("Dreptul de atașare a fost acordat."); await grants.refresh(); };
  return <Card.Root><Card.Body><Card.Title>Acces documente eArhivă pentru portofolii</Card.Title><Card.Subtitle>Acordați explicit unui utilizator dreptul de a folosi un document eligibil ca dovadă în propriul portofoliu.</Card.Subtitle><Card.Content><div className="flex flex-col gap-3">
    {notice && <Message.Root severity={notice.includes("nu poate") ? "error" : "success"}><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}
    <ServerTable label="Drepturi de atașare active" state={grants} onAdd={() => setGrantOpen(true)} addLabel="drept de atașare" emptyMessage="Nu există drepturi de atașare acordate." columns={[
      { header: "Document", field: "document_title", render: (grant) => <strong>{(grant as PortfolioAttachmentGrant).document_title}</strong> }, { header: "Beneficiar", field: "grantee_name", render: (grant) => (grant as PortfolioAttachmentGrant).grantee_name }, { header: "Acordat la", field: "created_at", render: (grant) => new Intl.DateTimeFormat("ro-RO", { dateStyle: "medium", timeStyle: "short" }).format(new Date((grant as PortfolioAttachmentGrant).created_at)) }, { header: "Acțiuni", action: true, render: (grant) => <Button iconOnly rounded size="small" variant="text" severity="danger" aria-label={`Revocă dreptul pentru ${(grant as PortfolioAttachmentGrant).grantee_name}`} title="Revocă dreptul" disabled={saving} onClick={() => setPendingRevoke(grant as PortfolioAttachmentGrant)}><i className="pi pi-trash" aria-hidden="true" /></Button> },
    ]} />
  </div></Card.Content></Card.Body>{grantOpen && <GrantDialog api={api} onClose={() => setGrantOpen(false)} onGranted={afterGranted} />}<Dialog.Root open={Boolean(pendingRevoke)} onOpenChange={(event: { value?: boolean }) => !event.value && !saving && setPendingRevoke(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup aria-describedby="revoke-grant-description"><Dialog.Header><Dialog.Title>Confirmați revocarea</Dialog.Title><Dialog.Close aria-label="Închide confirmarea" /></Dialog.Header><Dialog.Content><p id="revoke-grant-description">Utilizatorul nu va mai putea folosi documentul selectat ca dovadă nouă în portofoliu. Dovezile deja depuse rămân protejate.</p></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={() => setPendingRevoke(undefined)}>Renunță</Button><Button severity="danger" disabled={saving} onClick={() => void revoke()}>{saving ? "Se revocă…" : "Revocă dreptul"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root></Card.Root>;
}
