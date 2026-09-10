import { useCallback, useEffect, useState, type ChangeEvent, type ReactNode } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Checkbox } from "@primereact/ui/checkbox";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";

type Transfer = components["schemas"]["PortfolioTransferEvent"];
type TransferPage = components["schemas"]["EducationPageOfPortfolioTransferEvent"];
type CreateTransfer = components["schemas"]["CreateIntertenantPortfolioTransferRequest"];
type PortfolioTransferDestination = components["schemas"]["PortfolioTransferDestination"];
export type DestinationTenant = { code: string; label: string };
export type TransferCapabilities = { send: boolean; receive: boolean };
export type TransferQuery = { page: number; pageSize: number; sort?: string; direction?: "asc" | "desc"; filters?: Record<string, string> };
type TransferAdvanceAction = "mark_sent" | "confirm_received" | "close_transfer";
export type IntertenantPortfolioTransferApi = {
  destinations(): Promise<DestinationTenant[]>;
  outgoing(portfolioID: string, query: TransferQuery): Promise<TransferPage>;
  create(portfolioID: string, input: CreateTransfer): Promise<Transfer>;
  advance(portfolioID: string, transferID: string, action: TransferAdvanceAction): Promise<Transfer>;
  inbox(query: TransferQuery): Promise<TransferPage>;
  accept(transferID: string): Promise<Transfer>;
};

const requireData = async <T,>(result: { data?: T; error?: unknown; response: Response }): Promise<T> => {
  if (result.data !== undefined) return result.data;
  const code = typeof result.error === "object" && result.error !== null && "code" in result.error ? String((result.error as { code?: unknown }).code) : `transfer_${result.response.status}`;
  throw new Error(code);
};
const queryParams = (query: TransferQuery) => ({ page: query.page, pageSize: query.pageSize, sort: query.sort, direction: query.direction, "filter.transfer_code": query.filters?.transfer_code || undefined, "filter.source_institution": query.filters?.source_institution || undefined, "filter.status": query.filters?.status || undefined });

/** Strict generated OpenAPI client adapter; no browser request uses an untyped route string. */
export function createIntertenantPortfolioTransferApi(client: ContractClient): IntertenantPortfolioTransferApi {
  return {
    destinations: async () => {
      const destinations = await requireData<PortfolioTransferDestination[]>(
        await client.GET("/api/education/portfolio-transfer-destinations"),
      );
      return destinations.map((destination) => ({
        code: destination.tenant_code,
        label: destination.short_name || destination.display_name,
      }));
    },
    outgoing: async (portfolioID, query) => requireData(await client.GET("/api/education/portfolios/records/{recordID}/transfers", { params: { path: { recordID: portfolioID }, query: queryParams(query) } })),
    create: async (portfolioID, body) => requireData(await client.POST("/api/education/portfolios/records/{recordID}/transfers", { params: { path: { recordID: portfolioID } }, body })),
    advance: async (portfolioID, transferID, action) => requireData(await client.POST("/api/education/portfolios/records/{recordID}/transfers/{itemID}/advance", { params: { path: { recordID: portfolioID, itemID: transferID } }, body: { action } })),
    inbox: async (query) => requireData(await client.GET("/api/education/portfolio-transfer-inbox", { params: { query: queryParams(query) } })),
    accept: async (transferID) => requireData(await client.POST("/api/education/portfolio-transfer-inbox/{itemID}/accept", { params: { path: { itemID: transferID } } })),
  };
}

const initialQuery = (): TransferQuery => ({ page: 1, pageSize: 20, filters: {} });
const label = (item?: string) => item ? item.replaceAll("_", " ") : "—";
const transferAdvance = (item: Transfer): { action: TransferAdvanceAction; label: string; icon: string } | undefined => {
  switch (item.status) {
    case "pregatit":
      return { action: "mark_sent", label: "Marchează trimis", icon: "pi pi-send" };
    case "trimis":
      return item.routing_version === 1
        ? { action: "confirm_received", label: "Confirmă recepția", icon: "pi pi-check-circle" }
        : undefined;
    case "receptionat":
      return { action: "close_transfer", label: "Închide transferul", icon: "pi pi-verified" };
    default:
      return undefined;
  }
};
function useTransfers(load: (query: TransferQuery) => Promise<TransferPage>) { const [query, setQuery] = useState(initialQuery); const [page, setPage] = useState<TransferPage>({ items: [], total: 0, page: 1, pageSize: 20 }); const [loading, setLoading] = useState(true); const [error, setError] = useState<string>(); const refresh = useCallback(async () => { setLoading(true); setError(undefined); try { setPage(await load(query)); } catch { setError("Transferurile nu au putut fi încărcate sau permisiunea nu este disponibilă."); } finally { setLoading(false); } }, [load, query]); useEffect(() => { void refresh(); }, [refresh]); return { query, page, loading, error, refresh, filter: (field: string, value: string) => setQuery(current => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } })), sort: (field: string) => setQuery(current => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" })), setPage: (pageNumber: number) => setQuery(current => ({ ...current, page: pageNumber })), setPageSize: (pageSize: number) => setQuery(current => ({ ...current, page: 1, pageSize })) }; }
function TransferTable({ title, state, action, onAdd, addLabel, addDisabled, filterableFields, sortableFields }: { title: string; state: ReturnType<typeof useTransfers>; action?: (row: Transfer) => ReactNode; onAdd?: () => void; addLabel?: string; addDisabled?: boolean; filterableFields: readonly (keyof Transfer)[]; sortableFields: readonly (keyof Transfer)[] }) {
  const { query, page, error, loading, filter, sort, setPage, setPageSize } = state;
  const fields: Array<[string, keyof Transfer]> = [["Cod", "transfer_code"], ["Instituție sursă", "source_institution"], ["Instituție destinație", "destination_institution"], ["Stare", "status"]];
  const lastPage = Math.max(1, Math.ceil(page.total / page.pageSize));
  return <div className="flex min-h-0 flex-col gap-2">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="min-h-64 overflow-hidden rounded-border border border-surface">{loading && !page.items.length ? <div className="flex min-h-64 items-center justify-center"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> : <DataTable.Root data={page.items as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(42dvh,30rem)] min-h-56 overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>{fields.map(([heading, field]) => <DataTable.THeadCell key={heading}><div className="flex flex-col gap-1">{sortableFields.includes(field) ? <Button size="small" variant="text" severity="secondary" onClick={() => sort(String(field))}>{heading}{query.sort === field ? query.direction === "asc" ? " ↑" : " ↓" : ""}</Button> : <span>{heading}</span>}{filterableFields.includes(field) && <InputText aria-label={`Filtru ${title} ${heading}`} value={query.filters?.[String(field)] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => filter(String(field), event.target.value)} placeholder="Filtru" />}</div></DataTable.THeadCell>)}{action && <DataTable.THeadCell frozen alignFrozen="right"><span className="flex items-center justify-between gap-2"><span>Acțiuni</span>{onAdd && <Button iconOnly rounded size="small" aria-label={addLabel ?? "Adaugă"} title={addLabel ?? "Adaugă"} disabled={addDisabled} onClick={onAdd}><i className="pi pi-plus" aria-hidden="true" /></Button>}</span></DataTable.THeadCell>}</DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const row = item as Transfer; return <DataTable.Row key={row.id} index={index}>{fields.map(([heading, field]) => <DataTable.Cell key={heading}>{label(String(row[field] ?? ""))}</DataTable.Cell>)}{action && <DataTable.Cell frozen alignFrozen="right">{action(row)}</DataTable.Cell>}</DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>}</div><div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2"><span>{page.total ? `${(page.page - 1) * page.pageSize + 1}–${Math.min(page.page * page.pageSize, page.total)} din ${page.total}` : "0 rezultate"}</span><div className="flex items-center gap-2"><Select.Root value={String(query.pageSize)} options={[10, 20, 50].map(value => ({ label: `${value}/pagină`, value: String(value) }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setPageSize(Number(event.value))}><Select.Trigger aria-label={`Rezultate pe pagină ${title}`}><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button size="small" severity="secondary" variant="outlined" disabled={loading || query.page <= 1} onClick={() => setPage(query.page - 1)}>Anterior</Button><Button size="small" severity="secondary" variant="outlined" disabled={loading || query.page >= lastPage} onClick={() => setPage(query.page + 1)}>Următor</Button></div><span>{loading ? "Se actualizează…" : ""}</span></div></div>;
}
function SendDialog({ tenants, onClose, onCreate }: { tenants: DestinationTenant[]; onClose: () => void; onCreate: (input: CreateTransfer) => Promise<void> }) {
  const [tenant, setTenant] = useState("");
  const [transferType, setTransferType] = useState<CreateTransfer["transfer_type"]>("mutare");
  const [handoverOn, setHandoverOn] = useState(""); const [notes, setNotes] = useState(""); const [saving, setSaving] = useState(false);
  const save = async () => { setSaving(true); try { await onCreate({ transfer_type: transferType, destination_tenant_code: tenant, handover_on: handoverOn, notes }); onClose(); } finally { setSaving(false); } };
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,38rem)]"><Dialog.Header><Dialog.Title>Inițiază transfer inter-tenant</Dialog.Title><Dialog.Close aria-label="Închide transferul" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Tenantul și instituția de destinație sunt validate de server; identitatea expeditorului și pachetul probatoriu nu sunt furnizate de browser.</Message.Text></Message.Content></Message.Root><label className="flex flex-col gap-1"><span>Tenant destinație *</span><Select.Root value={tenant || null} options={tenants.map(item => ({ label: item.label, value: item.code }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setTenant(String(event.value ?? ""))}><Select.Trigger aria-label="Tenant destinație"><Select.Value placeholder="Alegeți tenantul" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label><label className="flex flex-col gap-1"><span>Tip transfer *</span><Select.Root value={transferType} options={["predare", "primire", "mutare", "detasare"].map(value => ({ label: label(value), value }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setTransferType(event.value as CreateTransfer["transfer_type"])}><Select.Trigger aria-label="Tip transfer"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label><label className="flex flex-col gap-1"><span>Data predării *</span><InputText type="date" value={handoverOn} onChange={(event: ChangeEvent<HTMLInputElement>) => setHandoverOn(event.target.value)} /></label><label className="flex flex-col gap-1"><span>Note</span><InputText value={notes} onChange={(event: ChangeEvent<HTMLInputElement>) => setNotes(event.target.value)} /></label></div></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !tenant || !handoverOn} onClick={() => void save()}>{saving ? "Se inițiază…" : "Inițiază transfer"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}
function AcceptDialog({ item, onClose, onAccept }: { item: Transfer; onClose: () => void; onAccept: () => Promise<void> }) { const [confirmed, setConfirmed] = useState(false); const [saving, setSaving] = useState(false); const accept = async () => { setSaving(true); try { await onAccept(); onClose(); } finally { setSaving(false); } }; return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,34rem)]"><Dialog.Header><Dialog.Title>Acceptă transferul {item.transfer_code}</Dialog.Title><Dialog.Close aria-label="Închide confirmarea" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="warn"><Message.Content><Message.Text>Acceptarea înregistrează identitatea și data recepției exclusiv pe server, în tenantul și instituția curentă.</Message.Text></Message.Content></Message.Root><p>De la: <strong>{item.source_institution}</strong></p><label className="flex items-start gap-2"><Checkbox.Root checked={confirmed} onCheckedChange={() => setConfirmed(value => !value)} aria-label="Confirm primirea pachetului probatoriu"><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>Confirm că instituția a primit pachetul probatoriu identificat mai sus și solicit înregistrarea recepției.</span></label></div></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !confirmed} onClick={() => void accept()}>{saving ? "Se acceptă…" : "Acceptă profesional"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>; }

export function PortfolioIntertenantTransfer({ portfolioID, api, capabilities }: { portfolioID: string; api: IntertenantPortfolioTransferApi; capabilities: TransferCapabilities }) {
  const outgoing = useTransfers(useCallback(query => capabilities.send ? api.outgoing(portfolioID, query) : Promise.resolve({ items: [], total: 0, page: query.page, pageSize: query.pageSize }), [api, capabilities.send, portfolioID]));
  const inbox = useTransfers(useCallback(query => capabilities.receive ? api.inbox(query) : Promise.resolve({ items: [], total: 0, page: query.page, pageSize: query.pageSize }), [api, capabilities.receive]));
  const [destinations, setDestinations] = useState<DestinationTenant[]>([]);
  const [destinationsLoading, setDestinationsLoading] = useState(capabilities.send);
  const [destinationsError, setDestinationsError] = useState<string>();
  const [sendOpen, setSendOpen] = useState(false); const [accepting, setAccepting] = useState<Transfer>(); const [notice, setNotice] = useState<string>();
  useEffect(() => {
    let live = true;
    if (!capabilities.send) { setDestinationsLoading(false); return () => { live = false; }; }
    setDestinationsLoading(true); setDestinationsError(undefined);
    void api.destinations().then((items) => { if (live) setDestinations(items); }).catch(() => { if (live) setDestinationsError("Destinațiile autorizate nu au putut fi încărcate."); }).finally(() => { if (live) setDestinationsLoading(false); });
    return () => { live = false; };
  }, [api, capabilities.send]);
  const create = async (input: CreateTransfer) => { await api.create(portfolioID, input); setNotice("Transferul a fost inițiat ca pachet pregătit."); await outgoing.refresh(); };
  const advance = async (item: Transfer, action: TransferAdvanceAction) => { await api.advance(portfolioID, item.id!, action); setNotice(action === "mark_sent" ? "Pachetul a fost transmis către tenantul destinație." : action === "confirm_received" ? "Recepția a fost confirmată." : "Transferul a fost închis."); await outgoing.refresh(); };
  const accept = async () => { if (!accepting?.id) return; await api.accept(accepting.id); setNotice("Recepția a fost înregistrată de server."); await inbox.refresh(); };
  const outgoingFields: Array<keyof Transfer> = ["transfer_code", "source_institution", "destination_institution", "status"];
  const inboxFields: Array<keyof Transfer> = ["transfer_code", "source_institution", "status"];
  const canStart = capabilities.send && !destinationsLoading && !destinationsError && destinations.length > 0;
  return <div className="flex min-h-0 flex-col gap-4">{notice && <Message.Root severity="success"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}<div className="grid min-h-0 gap-4 xl:grid-cols-2"><Card.Root><Card.Body><Card.Title>Expediere inter-tenant</Card.Title><Card.Content><div className="flex flex-col gap-3">{capabilities.send ? <>{destinationsError && <Message.Root severity="error"><Message.Content><Message.Text>{destinationsError}</Message.Text></Message.Content></Message.Root>}{!destinationsLoading && !destinationsError && destinations.length === 0 && <Message.Root severity="info"><Message.Content><Message.Text>Nu există destinații inter-tenant autorizate pentru contextul curent.</Message.Text></Message.Content></Message.Root>}</> : <Message.Root severity="info"><Message.Content><Message.Text>Nu aveți capabilitatea de expediere inter-tenant.</Message.Text></Message.Content></Message.Root>}<TransferTable title="expedieri" state={outgoing} filterableFields={outgoingFields} sortableFields={outgoingFields} onAdd={canStart ? () => setSendOpen(true) : undefined} addLabel="Inițiază transfer" action={capabilities.send ? item => { const transition = transferAdvance(item); return transition ? <Button iconOnly rounded size="small" variant="text" aria-label={`${transition.label} ${item.transfer_code}`} title={transition.label} onClick={() => void advance(item, transition.action)}><i className={transition.icon} aria-hidden="true" /></Button> : null; } : undefined} /></div></Card.Content></Card.Body></Card.Root><Card.Root><Card.Body><Card.Title>Inbox transferuri primite</Card.Title><Card.Content>{capabilities.receive ? <TransferTable title="inbox" state={inbox} filterableFields={inboxFields} sortableFields={inboxFields} action={item => item.routing_version === 2 && item.status === "trimis" ? <Button size="small" onClick={() => setAccepting(item)}>Acceptă</Button> : null} /> : <Message.Root severity="info"><Message.Content><Message.Text>Nu aveți capabilitatea de recepție inter-tenant.</Message.Text></Message.Content></Message.Root>}</Card.Content></Card.Body></Card.Root></div>{sendOpen && <SendDialog tenants={destinations} onClose={() => setSendOpen(false)} onCreate={create} />}{accepting && <AcceptDialog item={accepting} onClose={() => setAccepting(undefined)} onAccept={accept} />}</div>;
}
