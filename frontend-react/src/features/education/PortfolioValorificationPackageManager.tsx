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

type Package = components["schemas"]["PortfolioValorificationPackage"];
type PackagePage = components["schemas"]["EducationPageOfPortfolioValorificationPackage"];
type CreatePackage = components["schemas"]["CreatePortfolioValorificationPackageRequest"];
type Scope = CreatePackage["scope"];
type Purpose = CreatePackage["purpose"];
type Source = components["schemas"]["PortfolioValorificationEligibleSource"];
type ArchiveVersion = components["schemas"]["PortfolioValorificationEligibleArchiveVersion"];
type Evidence = components["schemas"]["PortfolioValorificationPackageDocument"];

export type ValorificationCapabilities = { read: boolean; manage: boolean };
export type ValorificationQuery = {
  page: number;
  pageSize: number;
  sort?: "created_at" | "scope" | "purpose" | "status";
  direction?: "asc" | "desc";
  filters?: { scope?: string; purpose?: string; status?: string };
};

export type PortfolioValorificationPackageApi = {
  list(portfolioID: string, query: ValorificationQuery): Promise<PackagePage>;
  sources(portfolioID: string, scope: Scope): Promise<Source[]>;
  archiveVersions(portfolioID: string): Promise<ArchiveVersion[]>;
  create(portfolioID: string, scope: Scope, purpose: Purpose, sourceID: string): Promise<Package>;
  attach(portfolioID: string, packageID: string, version: ArchiveVersion): Promise<void>;
  advance(portfolioID: string, packageID: string, action: "submit" | "validate" | "complete"): Promise<Package>;
  evidence(portfolioID: string, packageID: string): Promise<Evidence[]>;
};

const requireData = async <T,>(result: { data?: T; error?: unknown; response: Response }): Promise<T> => {
  if (result.data !== undefined) return result.data;
  throw new Error(`valorification_${result.response.status}`);
};

const queryParams = (query: ValorificationQuery) => ({
  page: query.page,
  pageSize: query.pageSize,
  sort: query.sort,
  direction: query.direction,
  "filter.scope": query.filters?.scope || undefined,
  "filter.purpose": query.filters?.purpose || undefined,
  "filter.status": query.filters?.status || undefined,
});

/** Uses only literal operations and DTOs generated from the backend OpenAPI document. */
export function createPortfolioValorificationPackageApi(client: ContractClient): PortfolioValorificationPackageApi {
  return {
    list: async (portfolioID, query) => requireData(await client.GET(
      "/api/education/portfolios/records/{recordID}/valorification-packages",
      { params: { path: { recordID: portfolioID }, query: queryParams(query) } },
    )),
    sources: async (portfolioID, scope) => requireData(await client.GET(
      "/api/education/portfolios/records/{recordID}/valorification-packages/eligible-sources",
      { params: { path: { recordID: portfolioID }, query: { scope } } },
    )),
    archiveVersions: async (portfolioID) => requireData(await client.GET(
      "/api/education/portfolios/records/{recordID}/valorification-packages/eligible-archive-versions",
      { params: { path: { recordID: portfolioID } } },
    )),
    create: async (portfolioID, scope, purpose, sourceID) => {
      const body: CreatePackage = scope === "evaluare_profesionala"
        ? { scope, purpose, source_evaluation_id: sourceID }
        : scope === "mobilitate"
          ? { scope, purpose, source_mobility_case_id: sourceID }
          : { scope, purpose, source_merit_grant_id: sourceID };
      return requireData(await client.POST(
        "/api/education/portfolios/records/{recordID}/valorification-packages",
        { params: { path: { recordID: portfolioID } }, body },
      ));
    },
    attach: async (portfolioID, packageID, version) => {
      await requireData(await client.POST(
        "/api/education/portfolios/records/{recordID}/valorification-packages/{itemID}/documents",
        {
          params: { path: { recordID: portfolioID, itemID: packageID } },
          body: { archive_document_id: version.archive_document_id, archive_version_id: version.archive_version_id },
        },
      ));
    },
    advance: async (portfolioID, packageID, action) => requireData(await client.POST(
      "/api/education/portfolios/records/{recordID}/valorification-packages/{itemID}/advance",
      { params: { path: { recordID: portfolioID, itemID: packageID } }, body: { action } },
    )),
    evidence: async (portfolioID, packageID) => requireData(await client.GET(
      "/api/education/portfolios/records/{recordID}/valorification-packages/{itemID}/documents",
      { params: { path: { recordID: portfolioID, itemID: packageID } } },
    )),
  };
}

const scopes: Array<{ value: Scope; label: string }> = [
  { value: "evaluare_profesionala", label: "Evaluare profesională" },
  { value: "mobilitate", label: "Mobilitate" },
  { value: "gradatie_merit", label: "Gradație de merit" },
];

const purposes: Array<{ value: Purpose; scope: Scope; label: string }> = [
  { value: "licentiere", scope: "evaluare_profesionala", label: "Licențiere în cariera didactică" },
  { value: "debut", scope: "evaluare_profesionala", label: "Debut în cariera didactică" },
  { value: "definitivat", scope: "evaluare_profesionala", label: "Definitivat" },
  { value: "grad_ii", scope: "evaluare_profesionala", label: "Grad didactic II" },
  { value: "grad_i", scope: "evaluare_profesionala", label: "Grad didactic I" },
  { value: "evaluare_profesionala", scope: "evaluare_profesionala", label: "Evaluare profesională" },
  { value: "dezvoltare_profesionala", scope: "evaluare_profesionala", label: "Dezvoltare profesională" },
  { value: "inspectie_scolara", scope: "evaluare_profesionala", label: "Inspecție școlară" },
  { value: "evaluare_externa_calitate", scope: "evaluare_profesionala", label: "Evaluare externă a calității" },
  { value: "distinctie_premiu", scope: "evaluare_profesionala", label: "Distincție sau premiu" },
  { value: "mobilitate", scope: "mobilitate", label: "Mobilitate" },
  { value: "gradatie_merit", scope: "gradatie_merit", label: "Gradație de merit" },
  { value: "distinctie_premiu", scope: "gradatie_merit", label: "Distincție sau premiu" },
];

const purposeLabel = (purpose: Purpose) => purposes.find((option) => option.value === purpose)?.label ?? purpose;
const scopeLabel = (scope: Scope) => scopes.find((option) => option.value === scope)?.label ?? scope;
const statusLabel = (status: Package["status"]) => ({ draft: "Ciornă", submitted: "Depus", validated: "Validat", completed: "Finalizat" })[status];
const statusSeverity = (status: Package["status"]): "secondary" | "warn" | "success" => status === "completed" ? "success" : status === "draft" ? "secondary" : "warn";
const formatDateTime = (value: string) => new Intl.DateTimeFormat("ro-RO", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
const queryInitial = (): ValorificationQuery => ({ page: 1, pageSize: 20, filters: {} });

function SelectField({ label, ariaLabel, value, options, placeholder, onValue }: {
  label: string;
  ariaLabel: string;
  value: string | null;
  options: Array<{ label: string; value: string }>;
  placeholder?: string;
  onValue(value: string): void;
}) {
  return <label className="flex flex-col gap-1"><span>{label}</span><Select.Root value={value} options={options} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => onValue(String(event.value ?? ""))}><Select.Trigger aria-label={ariaLabel}><Select.Value placeholder={placeholder} /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>;
}

function CreatePackageWizard({ portfolioID, api, onClose, onSaved }: { portfolioID: string; api: PortfolioValorificationPackageApi; onClose(): void; onSaved(): Promise<void> }) {
  const [scope, setScope] = useState<Scope>("evaluare_profesionala");
  const [purpose, setPurpose] = useState<Purpose>("evaluare_profesionala");
  const [sources, setSources] = useState<Source[]>([]);
  const [sourceID, setSourceID] = useState("");
  const [created, setCreated] = useState<Package>();
  const [versions, setVersions] = useState<ArchiveVersion[]>([]);
  const [versionID, setVersionID] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const availablePurposes = purposes.filter((option) => option.scope === scope);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setSourceID("");
    setError(undefined);
    void api.sources(portfolioID, scope)
      .then((items) => { if (active) setSources(items); })
      .catch(() => { if (active) setError("Sursele eligibile nu au putut fi încărcate."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [api, portfolioID, scope]);

  const create = async () => {
    if (!sourceID) return;
    setSaving(true);
    setError(undefined);
    try {
      const item = await api.create(portfolioID, scope, purpose, sourceID);
      setCreated(item);
      setLoading(true);
      setVersions(await api.archiveVersions(portfolioID));
    } catch {
      setError("Pachetul nu a putut fi creat.");
    } finally {
      setSaving(false);
      setLoading(false);
    }
  };

  const attach = async () => {
    const version = versions.find((item) => item.archive_version_id === versionID);
    if (!created || !version) return;
    setSaving(true);
    setError(undefined);
    try {
      await api.attach(portfolioID, created.id, version);
      await onSaved();
      onClose();
    } catch {
      setError("Versiunea arhivată nu a putut fi atașată.");
    } finally {
      setSaving(false);
    }
  };

  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,42rem)]"><Dialog.Header><Dialog.Title>{created ? "Atașează versiune eArhivă" : "Creează pachet de valorificare"}</Dialog.Title><Dialog.Close aria-label="Închide asistentul" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<Message.Root severity="info"><Message.Content><Message.Text>{created ? "Alegeți versiunea probatorie din eArhivă." : "Selectați scopul juridic și sursa eligibilă furnizată de server."}</Message.Text></Message.Content></Message.Root>{loading ? <div className="flex justify-center p-4"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> : created ? <SelectField label="Versiune eArhivă *" ariaLabel="Versiune eArhivă" value={versionID || null} options={versions.map((item) => ({ label: `${item.title} · versiunea ${item.version_no}`, value: item.archive_version_id }))} placeholder="Alegeți versiunea" onValue={setVersionID} /> : <><SelectField label="Domeniu sursă *" ariaLabel="Domeniu valorificare" value={scope} options={scopes} onValue={(value) => { const next = value as Scope; setScope(next); setPurpose(purposes.find((item) => item.scope === next)?.value ?? "evaluare_profesionala"); }} /><SelectField label="Scop juridic *" ariaLabel="Scop juridic" value={purpose} options={availablePurposes} onValue={(value) => setPurpose(value as Purpose)} /><SelectField label="Sursă eligibilă *" ariaLabel="Sursă eligibilă" value={sourceID || null} options={sources.map((item) => ({ label: item.label, value: item.id }))} placeholder={sources.length ? "Alegeți sursa" : "Nu există surse eligibile"} onValue={setSourceID} /></>}</div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button severity="secondary" variant="outlined" disabled={saving} onClick={onClose}>Renunță</Button>{created ? <Button disabled={saving || !versionID} onClick={() => void attach()}>{saving ? "Se atașează…" : "Atașează versiunea"}</Button> : <Button disabled={saving || loading || !sourceID} onClick={() => void create()}>{saving ? "Se creează…" : "Continuă"}</Button>}</div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

function EvidenceDialog({ portfolioID, packageID, api, onClose }: { portfolioID: string; packageID: string; api: PortfolioValorificationPackageApi; onClose(): void }) {
  const [items, setItems] = useState<Evidence[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => { void api.evidence(portfolioID, packageID).then(setItems).catch(() => setError("Dovezile nu au putut fi încărcate.")); }, [api, packageID, portfolioID]);
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,42rem)]"><Dialog.Header><Dialog.Title>Dovezi arhivate</Dialog.Title><Dialog.Close aria-label="Închide dovezile" /></Dialog.Header><Dialog.Content>{error ? <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root> : <div className="flex flex-col gap-2">{items.length ? items.map((item) => <Card.Root key={item.id}><Card.Body><Card.Content><div className="flex flex-col gap-1"><span>Versiunea {item.archive_version_no}</span><span className="break-all">SHA-256: {item.archive_sha256}</span><span className="break-all">{item.archive_source_bucket}/{item.archive_source_object_key}</span></div></Card.Content></Card.Body></Card.Root>) : <Message.Root severity="info"><Message.Content><Message.Text>Nu există încă dovezi atașate.</Message.Text></Message.Content></Message.Root>}</div>}</Dialog.Content></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

type PackageTableItem = Record<string, unknown> & { id: string; entry: Package };
const isPackageTableItem = (item: Record<string, unknown>): item is PackageTableItem => typeof item.id === "string" && typeof item.entry === "object" && item.entry !== null;

export function PortfolioValorificationPackageManager({ portfolioID, client, capabilities }: { portfolioID: string; client: ContractClient; capabilities: ValorificationCapabilities }) {
  const api = useMemo(() => createPortfolioValorificationPackageApi(client), [client]);
  const [query, setQuery] = useState(queryInitial);
  const [page, setPage] = useState<PackagePage>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const [creating, setCreating] = useState(false);
  const [evidenceID, setEvidenceID] = useState<string>();

  const load = useCallback(async () => {
    if (!capabilities.read) return;
    setLoading(true);
    setError(undefined);
    try { setPage(await api.list(portfolioID, query)); }
    catch { setError("Pachetele nu au putut fi încărcate."); }
    finally { setLoading(false); }
  }, [api, capabilities.read, portfolioID, query]);
  useEffect(() => { void load(); }, [load]);

  const advance = async (item: Package, action: "submit" | "validate" | "complete") => {
    setLoading(true);
    setError(undefined);
    try { await api.advance(portfolioID, item.id, action); await load(); }
    catch { setError("Tranziția nu a putut fi aplicată. Verificați starea și drepturile."); }
    finally { setLoading(false); }
  };

  if (!capabilities.read) return <Message.Root severity="info"><Message.Content><Message.Text>Nu aveți capabilitatea de consultare a pachetelor de valorificare.</Message.Text></Message.Content></Message.Root>;
  const filters = query.filters ?? {};
  const lastPage = Math.max(1, Math.ceil(page.total / page.pageSize));
  const tableItems: PackageTableItem[] = page.items.map((entry) => ({ id: entry.id, entry }));
  const sort = (field: NonNullable<ValorificationQuery["sort"]>) => setQuery((current) => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" }));
  const filter = (field: "scope" | "purpose" | "status", value: string) => setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } }));

  return <Card.Root><Card.Body><Card.Title>Pachete de valorificare</Card.Title><Card.Content><div className="flex min-h-0 flex-col gap-3">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="min-h-64 overflow-hidden rounded-border border border-surface">{loading && !page.items.length ? <div className="flex min-h-64 items-center justify-center"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> : <DataTable.Root data={tableItems} dataKey="id" scrollable className="max-h-[min(55dvh,36rem)] min-h-56 overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>{([{ field: "scope", label: "Domeniu sursă", aria: "Filtru domeniu" }, { field: "purpose", label: "Scop juridic", aria: "Filtru scop juridic" }, { field: "status", label: "Stare", aria: "Filtru stare" }] as const).map((column) => <DataTable.THeadCell key={column.field}><div className="flex min-w-36 flex-col gap-1"><Button size="small" variant="text" severity="secondary" onClick={() => sort(column.field)}>{column.label}{query.sort === column.field ? query.direction === "asc" ? " ↑" : " ↓" : ""}</Button><InputText aria-label={column.aria} value={filters[column.field] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => filter(column.field, event.target.value)} placeholder="Filtru" /></div></DataTable.THeadCell>)}<DataTable.THeadCell>Creat de</DataTable.THeadCell><DataTable.THeadCell>Actualizat</DataTable.THeadCell><DataTable.THeadCell frozen alignFrozen="right"><span className="flex items-center justify-between gap-2"><span>Acțiuni</span>{capabilities.manage && <Button iconOnly rounded size="small" aria-label="Adaugă pachet" title="Adaugă pachet" onClick={() => setCreating(true)}><i className="pi pi-plus" aria-hidden="true" /></Button>}</span></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { if (!isPackageTableItem(item)) return null; const entry = item.entry; const transition = entry.status === "draft" ? "submit" : entry.status === "submitted" ? "validate" : entry.status === "validated" ? "complete" : undefined; return <DataTable.Row key={entry.id} index={index}><DataTable.Cell>{scopeLabel(entry.scope)}</DataTable.Cell><DataTable.Cell>{purposeLabel(entry.purpose)}</DataTable.Cell><DataTable.Cell><Tag value={statusLabel(entry.status)} severity={statusSeverity(entry.status)} /></DataTable.Cell><DataTable.Cell>{entry.created_by_subject}</DataTable.Cell><DataTable.Cell>{formatDateTime(entry.completed_at ?? entry.validated_at ?? entry.submitted_at ?? entry.created_at)}</DataTable.Cell><DataTable.Cell frozen alignFrozen="right"><div className="flex gap-1"><Button iconOnly rounded size="small" variant="text" aria-label={`Vezi dovezi ${purposeLabel(entry.purpose)}`} title="Vezi dovezi" onClick={() => setEvidenceID(entry.id)}><i className="pi pi-eye" aria-hidden="true" /></Button>{capabilities.manage && transition && <Button iconOnly rounded size="small" variant="text" aria-label={`${transition} ${purposeLabel(entry.purpose)}`} title={transition} onClick={() => void advance(entry, transition)}><i className="pi pi-arrow-right" aria-hidden="true" /></Button>}</div></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>}</div><div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2"><span>{page.total ? `${(page.page - 1) * page.pageSize + 1}–${Math.min(page.page * page.pageSize, page.total)} din ${page.total}` : "0 rezultate"}</span><div className="flex items-center gap-2"><Select.Root value={String(query.pageSize)} options={[10, 20, 50].map((value) => ({ label: `${value}/pagină`, value: String(value) }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setQuery((current) => ({ ...current, page: 1, pageSize: Number(event.value) }))}><Select.Trigger aria-label="Rezultate pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button size="small" severity="secondary" variant="outlined" disabled={loading || query.page <= 1} onClick={() => setQuery((current) => ({ ...current, page: current.page - 1 }))}>Anterior</Button><Button size="small" severity="secondary" variant="outlined" disabled={loading || query.page >= lastPage} onClick={() => setQuery((current) => ({ ...current, page: current.page + 1 }))}>Următor</Button></div><span>{loading ? "Se actualizează…" : ""}</span></div></div>{creating && <CreatePackageWizard portfolioID={portfolioID} api={api} onClose={() => setCreating(false)} onSaved={load} />}{evidenceID && <EvidenceDialog portfolioID={portfolioID} packageID={evidenceID} api={api} onClose={() => setEvidenceID(undefined)} />}</Card.Content></Card.Body></Card.Root>;
}
