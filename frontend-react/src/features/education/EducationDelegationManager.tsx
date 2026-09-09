import { useCallback, useEffect, useMemo, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";

export type EducationDelegation = components["schemas"]["EducationDelegation"];
export type DelegationPrincipal = components["schemas"]["EducationDelegationEligibleAdjunct"];
export type DelegationOffer = components["schemas"]["OfferEducationDelegationRequest"];
export type DelegationPage = components["schemas"]["EducationPageOfEducationDelegation"];
export type DelegationQuery = {
  page: number;
  pageSize: number;
  sort?: string;
  direction?: "asc" | "desc";
  filters?: Record<string, string>;
};
export type DelegationCapabilities = {
  read: boolean;
  offer: boolean;
  accept: boolean;
  revoke: boolean;
  expire: boolean;
};
export type EducationDelegationApi = {
  list(query: DelegationQuery): Promise<DelegationPage>;
  eligibleAdjuncts(): Promise<DelegationPrincipal[]>;
  eligiblePermissions(): Promise<string[]>;
  create(input: DelegationOffer): Promise<EducationDelegation>;
  accept(id: string): Promise<EducationDelegation>;
  revoke(id: string): Promise<EducationDelegation>;
  expire(id: string): Promise<EducationDelegation>;
};

const requireData = async <T,>(result: { data?: T; error?: unknown; response: Response }): Promise<T> => {
  if (result.data !== undefined) return result.data;
  const code = typeof result.error === "object" && result.error !== null && "code" in result.error
    ? String((result.error as { code?: unknown }).code)
    : `education_delegation_${result.response.status}`;
  throw new Error(code);
};

export function createEducationDelegationApi(client: ContractClient): EducationDelegationApi {
  return {
    list: async (query) => requireData(await client.GET("/api/education/delegations", {
      params: { query: {
        page: query.page,
        pageSize: query.pageSize,
        sort: query.sort,
        direction: query.direction,
        "filter.status": query.filters?.status || undefined,
        "filter.permission_code": query.filters?.permission_code || undefined,
        "filter.delegate_user_id": query.filters?.delegate_user_id || undefined,
        "filter.delegate_name": query.filters?.delegate_name || undefined,
      } },
    })),
    eligibleAdjuncts: async () => requireData(await client.GET("/api/education/delegations/eligible-adjuncts")),
    eligiblePermissions: async () => requireData(await client.GET("/api/education/delegations/eligible-permissions")),
    create: async (body) => requireData(await client.POST("/api/education/delegations", { body })),
    accept: async (delegationID) => requireData(await client.POST("/api/education/delegations/{delegationID}/accept", { params: { path: { delegationID } } })),
    revoke: async (delegationID) => requireData(await client.POST("/api/education/delegations/{delegationID}/revoke", { params: { path: { delegationID } } })),
    expire: async (delegationID) => requireData(await client.POST("/api/education/delegations/{delegationID}/expire", { params: { path: { delegationID } } })),
  };
}

const initialQuery = (): DelegationQuery => ({ page: 1, pageSize: 20, filters: {} });
const emptyPage = (query: DelegationQuery): DelegationPage => ({ items: [], total: 0, page: query.page, pageSize: query.pageSize });
const statusLabel = (status: string) => ({ offered: "Oferită", accepted: "Acceptată", revoked: "Revocată", expired: "Expirată" }[status] ?? status);
const statusSeverity = (status: string): "success" | "warn" | "danger" | "secondary" => status === "accepted" ? "success" : status === "offered" ? "warn" : status === "revoked" ? "danger" : "secondary";

function OfferDialog({ adjuncts, permissions, onClose, onCreate }: {
  adjuncts: DelegationPrincipal[];
  permissions: string[];
  onClose: () => void;
  onCreate: (input: DelegationOffer) => Promise<void>;
}) {
  const [delegateUserID, setDelegateUserID] = useState("");
  const [permissionCode, setPermissionCode] = useState("");
  const [validFrom, setValidFrom] = useState("");
  const [validUntil, setValidUntil] = useState("");
  const [notes, setNotes] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const save = async () => {
    setSaving(true);
    setError(undefined);
    try {
      await onCreate({
        delegate_user_id: delegateUserID,
        permission_code: permissionCode,
        resource_type: "institution",
        valid_from: validFrom || undefined,
        valid_until: validUntil || undefined,
        notes,
      });
      onClose();
    } catch {
      setError("Delegarea nu a putut fi oferită. Serverul a respins drepturile, destinatarul sau perioada.");
    } finally {
      setSaving(false);
    }
  };
  return (
    <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}>
      <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,38rem)]">
        <Dialog.Header><Dialog.Title>Oferă delegare educațională</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header>
        <Dialog.Content><div className="flex flex-col gap-3">
          {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
          <Message.Root severity="info"><Message.Content><Message.Text>Delegatorul, instituția și dovada temporală sunt stabilite de server. Destinatarii și drepturile sunt limitate la opțiunile eligibile.</Message.Text></Message.Content></Message.Root>
          <label className="flex flex-col gap-1"><span>Director adjunct *</span>
            <Select.Root value={delegateUserID || null} options={adjuncts.map((item) => ({ label: `${item.name} · ${item.email}`, value: item.user_id }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setDelegateUserID(String(event.value ?? ""))}>
              <Select.Trigger aria-label="Director adjunct"><Select.Value placeholder="Selectați destinatarul" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
            </Select.Root>
          </label>
          <label className="flex flex-col gap-1"><span>Drept delegat *</span>
            <Select.Root value={permissionCode || null} options={permissions.map((code) => ({ label: code, value: code }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setPermissionCode(String(event.value ?? ""))}>
              <Select.Trigger aria-label="Drept delegat"><Select.Value placeholder="Selectați dreptul" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
            </Select.Root>
          </label>
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1"><span>Valabil de la</span><InputText aria-label="Valabil de la" type="date" value={validFrom} onChange={(event: ChangeEvent<HTMLInputElement>) => setValidFrom(event.target.value)} /></label>
            <label className="flex flex-col gap-1"><span>Valabil până la</span><InputText aria-label="Valabil până la" type="date" min={validFrom || undefined} value={validUntil} onChange={(event: ChangeEvent<HTMLInputElement>) => setValidUntil(event.target.value)} /></label>
          </div>
          <label className="flex flex-col gap-1"><span>Observații</span><InputText value={notes} onChange={(event: ChangeEvent<HTMLInputElement>) => setNotes(event.target.value)} /></label>
        </div></Dialog.Content>
        <Dialog.Footer><div className="flex justify-end gap-2"><Button severity="secondary" variant="outlined" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !delegateUserID || !permissionCode} onClick={() => void save()}>{saving ? "Se salvează…" : "Oferă delegarea"}</Button></div></Dialog.Footer>
      </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
    </Dialog.Root>
  );
}

export function EducationDelegationManager({ api, capabilities, onChanged }: { api: EducationDelegationApi; capabilities: DelegationCapabilities; onChanged?: () => void | Promise<void> }) {
  const [query, setQuery] = useState(initialQuery);
  const [page, setPage] = useState<DelegationPage>(() => emptyPage(query));
  const [adjuncts, setAdjuncts] = useState<DelegationPrincipal[]>([]);
  const [permissions, setPermissions] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [offerOpen, setOfferOpen] = useState(false);
  const load = useCallback(async () => {
    if (!capabilities.read) { setPage(emptyPage(query)); return; }
    setLoading(true); setError(undefined);
    try { setPage(await api.list(query)); }
    catch { setError("Delegările nu au putut fi încărcate."); }
    finally { setLoading(false); }
  }, [api, capabilities.read, query]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => {
    if (!capabilities.offer) return;
    void Promise.all([api.eligibleAdjuncts(), api.eligiblePermissions()])
      .then(([users, codes]) => { setAdjuncts(users); setPermissions(codes); })
      .catch(() => setError("Opțiunile eligibile pentru delegare nu au putut fi încărcate."));
  }, [api, capabilities.offer]);
  const run = async (operation: (id: string) => Promise<EducationDelegation>, id: string) => {
    setLoading(true); setError(undefined);
    try { await operation(id); await load(); await onChanged?.(); }
    catch { setError("Tranziția a fost respinsă de server. Verificați rolul, perioada și starea delegării."); }
    finally { setLoading(false); }
  };
  const lastPage = Math.max(1, Math.ceil(page.total / page.pageSize));
  const fields: Array<[string, keyof EducationDelegation]> = [["Delegator", "delegator_name"], ["Delegat", "delegate_name"], ["Drept", "permission_code"], ["De la", "valid_from"], ["Până la", "valid_until"]];
  const filter = (field: string, value: string) => setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } }));
  const sort = (field: string) => setQuery((current) => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" }));
  const canOffer = capabilities.offer && adjuncts.length > 0 && permissions.length > 0;
  return <Card.Root><Card.Body><Card.Title>Delegări de autoritate</Card.Title><Card.Content><div className="flex min-h-0 flex-col gap-3">
    {!capabilities.read ? <Message.Root severity="info"><Message.Content><Message.Text>Nu aveți drept de consultare a delegărilor.</Message.Text></Message.Content></Message.Root> : <>
      {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
      <div className="min-h-64 overflow-hidden rounded-border border border-surface">
        {loading && !page.items.length ? <div className="flex min-h-64 items-center justify-center"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> :
          <DataTable.Root data={page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(55dvh,36rem)] min-h-56 overflow-auto"><DataTable.Table>
            <DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>
              {fields.map(([header, field]) => <DataTable.THeadCell key={header}><div className="flex flex-col gap-1"><Button size="small" variant="text" severity="secondary" onClick={() => sort(String(field))}>{header}{query.sort === field ? query.direction === "asc" ? " ↑" : " ↓" : ""}</Button>{field === "permission_code" || field === "delegate_name" ? <InputText aria-label={`Filtru ${header}`} value={query.filters?.[field] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => filter(field, event.target.value)} placeholder="Filtru" /> : null}</div></DataTable.THeadCell>)}
              <DataTable.THeadCell><div className="flex flex-col gap-1"><span>Stare</span><InputText aria-label="Filtru Stare" value={query.filters?.status ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => filter("status", event.target.value)} placeholder="Filtru" /></div></DataTable.THeadCell>
              <DataTable.THeadCell frozen alignFrozen="right"><span className="flex items-center justify-between gap-2"><span>Acțiuni</span>{capabilities.offer && <Button iconOnly rounded size="small" aria-label="Oferă delegare" title={canOffer ? "Oferă delegare" : "Nu există opțiuni eligibile"} disabled={!canOffer} onClick={() => setOfferOpen(true)}><i className="pi pi-plus" aria-hidden="true" /></Button>}</span></DataTable.THeadCell>
            </DataTable.THeadRow></DataTable.THead>
            <DataTable.TBody>{({ item, index }) => { const delegation = item as unknown as EducationDelegation; return <DataTable.Row key={delegation.id} index={index}>
              {fields.map(([header, field]) => <DataTable.Cell key={header}>{delegation[field] || "—"}</DataTable.Cell>)}
              <DataTable.Cell><Tag value={statusLabel(delegation.status)} severity={statusSeverity(delegation.status)} /></DataTable.Cell>
              <DataTable.Cell frozen alignFrozen="right"><div className="flex gap-1">
                {capabilities.accept && delegation.status === "offered" && <Button iconOnly rounded size="small" variant="text" severity="success" aria-label={`Acceptă ${delegation.permission_code}`} onClick={() => void run(api.accept, delegation.id)}><i className="pi pi-check" aria-hidden="true" /></Button>}
                {capabilities.revoke && ["offered", "accepted"].includes(delegation.status) && <Button iconOnly rounded size="small" variant="text" severity="danger" aria-label={`Revocă ${delegation.permission_code}`} onClick={() => void run(api.revoke, delegation.id)}><i className="pi pi-times" aria-hidden="true" /></Button>}
                {capabilities.expire && delegation.status === "accepted" && <Button iconOnly rounded size="small" variant="text" severity="warn" aria-label={`Înregistrează expirarea ${delegation.permission_code}`} onClick={() => void run(api.expire, delegation.id)}><i className="pi pi-clock" aria-hidden="true" /></Button>}
              </div></DataTable.Cell>
            </DataTable.Row>; }}</DataTable.TBody>
          </DataTable.Table></DataTable.Root>}
      </div>
      <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2"><span>{page.total ? `${(page.page - 1) * page.pageSize + 1}–${Math.min(page.page * page.pageSize, page.total)} din ${page.total}` : "0 rezultate"}</span><div className="flex items-center gap-2"><Select.Root value={String(query.pageSize)} options={[10,20,50].map((value) => ({ label: `${value}/pagină`, value: String(value) }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setQuery((current) => ({ ...current, page: 1, pageSize: Number(event.value) }))}><Select.Trigger aria-label="Rezultate pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button size="small" severity="secondary" variant="outlined" disabled={loading || query.page <= 1} onClick={() => setQuery((current) => ({ ...current, page: current.page - 1 }))}>Anterior</Button><Button size="small" severity="secondary" variant="outlined" disabled={loading || query.page >= lastPage} onClick={() => setQuery((current) => ({ ...current, page: current.page + 1 }))}>Următor</Button></div><span>{loading ? "Se actualizează…" : ""}</span></div>
    </>}
  </div>{offerOpen && <OfferDialog adjuncts={adjuncts} permissions={permissions} onClose={() => setOfferOpen(false)} onCreate={async (input) => { await api.create(input); await load(); await onChanged?.(); }} />}</Card.Content></Card.Body></Card.Root>;
}
