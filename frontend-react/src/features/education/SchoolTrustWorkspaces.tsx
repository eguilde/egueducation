import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { Popover } from "@primereact/ui/popover";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import type { ContractClient } from "../../api/client";
import type { components, operations } from "../../api/generated";

type ReportCatalogItem = components["schemas"]["SchoolReportCatalogItem"];
type ReportCode = ReportCatalogItem["code"];
type ReportPage = components["schemas"]["EducationPageOfSchoolReportRow"];
type ReportRow = components["schemas"]["SchoolReportRow"];
type SignatureEvidence = components["schemas"]["SignedArtifactEvidence"];
type SignaturePage = components["schemas"]["EducationPageOfSignedArtifactEvidence"];
type SignatureRequest = components["schemas"]["SubmitSignedArtifactEvidenceRequest"];
type ArtifactType = SignatureRequest["artifact_type"];
type EligibleArtifact = components["schemas"]["EligibleSignedArtifact"];
type EligibleArchiveVersion = components["schemas"]["EligibleSignatureArchiveVersion"];

type SortDirection = "asc" | "desc";
type PageQuery = { page: number; pageSize: number; sort?: string; direction?: SortDirection; filters: Record<string, string> };
type ReportTableItem = { [key: string]: unknown; id: string; cells: Record<string, string> };
type SignatureTableItem = { [key: string]: unknown; id: string; evidence: SignatureEvidence };
type ReportFilterParams = NonNullable<operations["get_api_education_reports_reportcode_csv"]["parameters"]["query"]>;

const initialQuery = (): PageQuery => ({ page: 1, pageSize: 20, filters: {} });
const Spinner = ({ label = "Se încarcă" }: { label?: string }) => <ProgressSpinner.Root aria-label={label}><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;
const requireData = <T,>(result: { data?: T; error?: unknown; response: Response }, code: string): T => {
  if (result.data !== undefined) return result.data;
  throw new Error(`${code}_${result.response.status}`);
};
const displayValue = (value: unknown): string => {
  if (value === null || value === undefined || value === "") return "—";
  if (typeof value === "boolean") return value ? "Da" : "Nu";
  if (typeof value === "number") return new Intl.NumberFormat("ro-RO").format(value);
  if (typeof value === "string" && /^\d{4}-\d{2}-\d{2}T/.test(value)) {
    const parsed = new Date(value);
    if (!Number.isNaN(parsed.getTime())) return new Intl.DateTimeFormat("ro-RO", { dateStyle: "medium", timeStyle: "short" }).format(parsed);
  }
  return String(value);
};
const reportItem = (row: ReportRow, index: number): ReportTableItem => ({
  id: `report-${index}`,
  cells: Object.fromEntries(Object.entries(row).map(([key, value]) => [key, displayValue(value)])),
});
const queryWithFilters = (query: PageQuery) => ({
  page: query.page,
  pageSize: query.pageSize,
  sort: query.sort,
  direction: query.direction,
  ...Object.fromEntries(Object.entries(query.filters).filter(([, value]) => value.trim()).map(([key, value]) => [`filter.${key}`, value])),
});
const signatureStatusLabel = (status?: string) => ({ pending: "În așteptare", valid: "Validă", invalid: "Invalidă", error: "Indisponibilă" })[status ?? ""] ?? "Neverificată";
const signatureSeverity = (status?: string): "success" | "danger" | "warn" | "secondary" => status === "valid" ? "success" : status === "invalid" ? "danger" : status === "pending" ? "warn" : "secondary";
const artifactTypeOptions: Array<{ value: ArtifactType; label: string }> = [
  { value: "decision", label: "Decizie" },
  { value: "publication", label: "Publicare" },
  { value: "managerial_document", label: "Document managerial" },
  { value: "meeting_document", label: "Document de ședință" },
  { value: "meeting_minute", label: "Proces-verbal" },
  { value: "meeting_resolution", label: "Hotărâre de ședință" },
];
const artifactTypeLabel = (value: string) => artifactTypeOptions.find((option) => option.value === value)?.label ?? value;

function SelectField({ label, value, options, placeholder, disabled, onValue }: { label: string; value: string | null; options: Array<{ label: string; value: string }>; placeholder: string; disabled?: boolean; onValue(value: string): void }) {
  return <label className="flex flex-col gap-1"><span>{label}</span><Select.Root value={value} options={options} optionLabel="label" optionValue="value" disabled={disabled} onValueChange={(event: SelectValueChangeEvent) => onValue(String(event.value ?? ""))}><Select.Trigger aria-label={label}><Select.Value placeholder={placeholder} /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>;
}

function TableFooter({ page, total, pageSize, loading, onPage, onPageSize }: { page: number; total: number; pageSize: number; loading: boolean; onPage(page: number): void; onPageSize(pageSize: number): void }) {
  const first = total ? (page - 1) * pageSize + 1 : 0;
  const last = Math.min(page * pageSize, total);
  return <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface py-2"><span>{total ? `${first} – ${last} din ${total}` : "0 rezultate"}</span><div className="flex items-center gap-2"><Select.Root value={pageSize} options={[10, 20, 50, 100].map((value) => ({ label: String(value), value }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => onPageSize(Number(event.value))}><Select.Trigger aria-label="Rânduri pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button size="small" variant="outlined" disabled={loading || page <= 1} onClick={() => onPage(page - 1)}>Anterior</Button><Button size="small" variant="outlined" disabled={loading || page * pageSize >= total} onClick={() => onPage(page + 1)}>Următor</Button></div></div>;
}

function SortableFilterHeader({ label, field, query, onSort, onFilter }: { label: string; field: string; query: PageQuery; onSort(field: string): void; onFilter(field: string, value: string): void }) {
  return <div className="flex min-w-32 flex-col gap-1"><Button size="small" variant="text" severity="secondary" aria-label={`Sortează după ${label}`} onClick={() => onSort(field)}>{label}{query.sort === field ? query.direction === "asc" ? " ↑" : " ↓" : ""}</Button><InputText aria-label={`Filtru ${label}`} value={query.filters[field] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => onFilter(field, event.target.value)} /></div>;
}

async function reportCatalog(client: ContractClient) {
  return requireData(await client.GET("/api/education/reports"), "school_report_catalog");
}

async function reportRows(client: ContractClient, reportCode: ReportCode, query: PageQuery) {
  return requireData(await client.GET("/api/education/reports/{reportCode}", { params: { path: { reportCode }, query: queryWithFilters(query) } }), "school_report");
}

async function reportDownload(client: ContractClient, reportCode: ReportCode, format: "csv" | "pdf", filters: Record<string, string>): Promise<Blob> {
  const query = Object.fromEntries(Object.entries(filters).filter(([, value]) => value.trim()).map(([key, value]) => [`filter.${key}`, value])) as ReportFilterParams;
  const params = { path: { reportCode }, query };
  const result = format === "csv"
    ? await client.GET("/api/education/reports/{reportCode}/csv", { params, parseAs: "blob" })
    : await client.GET("/api/education/reports/{reportCode}/pdf", { params, parseAs: "blob" });
  if (result.error || !result.response.ok) throw new Error(`school_report_download_${result.response.status}`);
  const data: unknown = result.data;
  if (!(data instanceof Blob)) throw new Error("school_report_download_invalid_body");
  return data;
}

export function SchoolReportsWorkspace({ client, canExport }: { client: ContractClient; canExport: boolean }) {
  const [catalog, setCatalog] = useState<ReportCatalogItem[]>([]);
  const [reportCode, setReportCode] = useState<ReportCode>();
  const [query, setQuery] = useState<PageQuery>(initialQuery);
  const [page, setPage] = useState<ReportPage>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const request = useRef(0);
  const report = catalog.find((item) => item.code === reportCode);
  const tableRows = useMemo(() => page.items.map(reportItem), [page.items]);

  useEffect(() => { let active = true; setLoading(true); void reportCatalog(client).then((value) => { if (!active) return; setCatalog(value.items); setReportCode(value.items[0]?.code); }).catch(() => active && setError("Catalogul de rapoarte nu a putut fi încărcat.")).finally(() => active && setLoading(false)); return () => { active = false; }; }, [client]);
  const refresh = useCallback(async () => { if (!reportCode) return; const current = ++request.current; setLoading(true); setError(undefined); try { const value = await reportRows(client, reportCode, query); if (current === request.current) setPage(value); } catch { if (current === request.current) setError("Raportul nu a putut fi încărcat."); } finally { if (current === request.current) setLoading(false); } }, [client, query, reportCode]);
  useEffect(() => { const timer = window.setTimeout(() => void refresh(), 250); return () => window.clearTimeout(timer); }, [refresh]);
  const setFilter = (field: string, value: string) => setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } }));
  const sort = (field: string) => setQuery((current) => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" }));
  const download = async (format: "csv" | "pdf") => { if (!reportCode) return; setError(undefined); try { const blob = await reportDownload(client, reportCode, format, query.filters); const url = URL.createObjectURL(blob); const anchor = document.createElement("a"); anchor.href = url; anchor.download = `${reportCode}.${format}`; anchor.click(); URL.revokeObjectURL(url); } catch { setError("Exportul nu a putut fi generat."); } };

  return <Card.Root><Card.Body><Card.Title>Rapoarte școlare</Card.Title><Card.Subtitle>Rapoarte generate exclusiv de server pentru instituția activă, cu filtrare, sortare și paginare server-side.</Card.Subtitle><Card.Content><div className="flex min-h-0 flex-col gap-3">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="flex flex-wrap items-end justify-between gap-2"><SelectField label="Raport" value={reportCode ?? null} options={catalog.map((item) => ({ label: item.label, value: item.code }))} placeholder="Alegeți raportul" onValue={(value) => { setReportCode(value as ReportCode); setQuery(initialQuery()); }} /><div className="flex gap-1"><Button iconOnly rounded variant="outlined" aria-label="Exportă CSV" title={canExport ? "Exportă CSV" : "Nu aveți drept de export sensibil"} disabled={!canExport || !reportCode} onClick={() => void download("csv")}><i className="pi pi-file" aria-hidden="true" /></Button><Button iconOnly rounded variant="outlined" aria-label="Exportă PDF" title={canExport ? "Exportă PDF" : "Nu aveți drept de export sensibil"} disabled={!canExport || !reportCode} onClick={() => void download("pdf")}><i className="pi pi-file-pdf" aria-hidden="true" /></Button></div></div>{report?.description && <Message.Root severity="info"><Message.Content><Message.Text>{report.description}</Message.Text></Message.Content></Message.Root>}<div className="min-h-72 overflow-hidden rounded-border border border-surface"><DataTable.Root data={tableRows} dataKey="id" scrollable className="max-h-[min(62dvh,44rem)] min-h-64 overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>{report?.columns.map((column) => <DataTable.THeadCell key={column.key}><SortableFilterHeader label={column.label} field={column.key} query={query} onSort={sort} onFilter={setFilter} /></DataTable.THeadCell>)}</DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const row = item as ReportTableItem; return <DataTable.Row key={row.id} index={index}>{report?.columns.map((column) => <DataTable.Cell key={column.key}>{row.cells[column.key] ?? "—"}</DataTable.Cell>)}</DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>{loading && !page.items.length && <div className="flex justify-center p-6" role="status"><Spinner label="Se încarcă raportul" /></div>}{!loading && reportCode && !page.items.length && <Message.Root severity="info"><Message.Content><Message.Text>Raportul nu conține rezultate pentru filtrele curente.</Message.Text></Message.Content></Message.Root>}</div><TableFooter page={page.page} total={page.total} pageSize={page.pageSize} loading={loading} onPage={(value) => setQuery((current) => ({ ...current, page: value }))} onPageSize={(value) => setQuery((current) => ({ ...current, page: 1, pageSize: value }))} /></div></Card.Content></Card.Body></Card.Root>;
}

function EvidenceDetails({ evidence, onClose }: { evidence: SignatureEvidence; onClose(): void }) {
  const fields: Array<[string, ReactNode]> = [
    ["Tip artefact", artifactTypeLabel(evidence.artifact_type)], ["Subiect semnătură", evidence.signature_subject], ["Emitent certificat", evidence.certificate_issuer], ["Serie certificat", evidence.certificate_serial], ["Format / nivel", `${evidence.signature_format} / ${evidence.signature_level}`], ["Valabil de la", displayValue(evidence.certificate_valid_from)], ["Valabil până la", displayValue(evidence.certificate_valid_until)], ["Depusă la", displayValue(evidence.submitted_at)], ["SHA-256 document", <span className="break-all">{evidence.document_sha256}</span>], ["Stocare probatorie", <span className="break-all">{evidence.storage_bucket}/{evidence.storage_object_key}</span>], ["Validare", <Tag value={signatureStatusLabel(evidence.latest_validation?.validation_status)} severity={signatureSeverity(evidence.latest_validation?.validation_status)} />],
  ];
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,48rem)]"><Dialog.Header><Dialog.Title>Detalii dovadă de semnătură</Dialog.Title><Dialog.Close aria-label="Închide detaliile" /></Dialog.Header><Dialog.Content><dl className="grid gap-3 sm:grid-cols-2">{fields.map(([label, value]) => <div key={label}><dt className="font-medium">{label}</dt><dd>{value || "—"}</dd></div>)}</dl></Dialog.Content></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

function SubmitEvidenceDialog({ client, onClose, onSaved }: { client: ContractClient; onClose(): void; onSaved(): Promise<void> }) {
  const [artifactType, setArtifactType] = useState<ArtifactType>("decision");
  const [artifactSearch, setArtifactSearch] = useState("");
  const [archiveSearch, setArchiveSearch] = useState("");
  const [artifacts, setArtifacts] = useState<EligibleArtifact[]>([]);
  const [versions, setVersions] = useState<EligibleArchiveVersion[]>([]);
  const [artifactID, setArtifactID] = useState("");
  const [versionID, setVersionID] = useState("");
  const [form, setForm] = useState({ signature_format: "PAdES" as SignatureRequest["signature_format"], signature_level: "qualified" as SignatureRequest["signature_level"], signature_subject: "", certificate_issuer: "", certificate_serial: "", certificate_valid_from: "", certificate_valid_until: "" });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  useEffect(() => { const timer = window.setTimeout(() => { setLoading(true); void client.GET("/api/education/signatures/eligible-artifacts", { params: { query: { artifactType, q: artifactSearch.trim() || undefined, page: 1, pageSize: 50 } } }).then((result) => setArtifacts(requireData(result, "eligible_artifacts").items)).catch(() => setError("Artefactele eligibile nu au putut fi încărcate.")).finally(() => setLoading(false)); }, 200); return () => window.clearTimeout(timer); }, [artifactSearch, artifactType, client]);
  useEffect(() => { const timer = window.setTimeout(() => { setLoading(true); void client.GET("/api/education/signatures/eligible-archive-versions", { params: { query: { q: archiveSearch.trim() || undefined, page: 1, pageSize: 50 } } }).then((result) => setVersions(requireData(result, "eligible_archive_versions").items)).catch(() => setError("Versiunile eArhivă eligibile nu au putut fi încărcate.")).finally(() => setLoading(false)); }, 200); return () => window.clearTimeout(timer); }, [archiveSearch, client]);
  const selectedVersion = versions.find((item) => item.version_id === versionID);
  const valid = artifactID && selectedVersion && Object.values(form).every((value) => value.trim());
  const save = async () => { if (!valid || !selectedVersion) return; setSaving(true); setError(undefined); try { const body: SignatureRequest = { artifact_type: artifactType, artifact_id: artifactID, storage_document_id: selectedVersion.document_id, storage_version_id: selectedVersion.version_id, signature_format: form.signature_format, signature_level: form.signature_level, signature_subject: form.signature_subject.trim(), certificate_issuer: form.certificate_issuer.trim(), certificate_serial: form.certificate_serial.trim(), certificate_valid_from: new Date(form.certificate_valid_from).toISOString(), certificate_valid_until: new Date(form.certificate_valid_until).toISOString() }; requireData(await client.POST("/api/education/signatures", { body }), "submit_signature_evidence"); await onSaved(); onClose(); } catch { setError("Dovada nu a putut fi înregistrată. Verificați certificatul și versiunea selectată."); } finally { setSaving(false); } };
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,52rem)]"><Dialog.Header><Dialog.Title>Înregistrează dovadă de semnătură</Dialog.Title><Dialog.Close aria-label="Închide asistentul" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<Message.Root severity="info"><Message.Content><Message.Text>Artefactul și versiunea eArhivă sunt selectate din listele instituției active. Hash-ul și locația de stocare sunt preluate și validate exclusiv de server.</Message.Text></Message.Content></Message.Root><div className="grid gap-3 md:grid-cols-2"><SelectField label="Tip artefact *" value={artifactType} options={artifactTypeOptions} placeholder="Alegeți tipul" onValue={(value) => { setArtifactType(value as ArtifactType); setArtifactID(""); }} /><div className="flex flex-col gap-1"><span>Caută artefact</span><InputText aria-label="Caută artefact" value={artifactSearch} onChange={(event: ChangeEvent<HTMLInputElement>) => setArtifactSearch(event.target.value)} /></div><SelectField label="Artefact eligibil *" value={artifactID || null} options={artifacts.map((item) => ({ label: item.label, value: item.id }))} placeholder={loading ? "Se încarcă…" : "Alegeți artefactul"} disabled={loading} onValue={setArtifactID} /><div className="flex flex-col gap-1"><span>Caută versiune eArhivă</span><InputText aria-label="Caută versiune eArhivă" value={archiveSearch} onChange={(event: ChangeEvent<HTMLInputElement>) => setArchiveSearch(event.target.value)} /></div><SelectField label="Versiune eArhivă *" value={versionID || null} options={versions.map((item) => ({ label: `${item.title} · v${item.version_no}`, value: item.version_id }))} placeholder={loading ? "Se încarcă…" : "Alegeți versiunea"} disabled={loading} onValue={setVersionID} /><SelectField label="Format semnătură *" value={form.signature_format} options={["PAdES", "XAdES", "CAdES"].map((value) => ({ label: value, value }))} placeholder="Alegeți formatul" onValue={(value) => setForm((current) => ({ ...current, signature_format: value as SignatureRequest["signature_format"] }))} /><SelectField label="Nivel semnătură *" value={form.signature_level} options={[{ label: "Calificată", value: "qualified" }, { label: "Avansată", value: "advanced" }]} placeholder="Alegeți nivelul" onValue={(value) => setForm((current) => ({ ...current, signature_level: value as SignatureRequest["signature_level"] }))} />{(["signature_subject", "certificate_issuer", "certificate_serial"] as const).map((field) => <label key={field} className="flex flex-col gap-1"><span>{{ signature_subject: "Subiect certificat *", certificate_issuer: "Emitent certificat *", certificate_serial: "Serie certificat *" }[field]}</span><InputText aria-label={{ signature_subject: "Subiect certificat", certificate_issuer: "Emitent certificat", certificate_serial: "Serie certificat" }[field]} value={form[field]} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((current) => ({ ...current, [field]: event.target.value }))} /></label>)}{(["certificate_valid_from", "certificate_valid_until"] as const).map((field) => <label key={field} className="flex flex-col gap-1"><span>{field === "certificate_valid_from" ? "Certificat valabil de la *" : "Certificat valabil până la *"}</span><InputText type="datetime-local" aria-label={field === "certificate_valid_from" ? "Certificat valabil de la" : "Certificat valabil până la"} value={form[field]} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((current) => ({ ...current, [field]: event.target.value }))} /></label>)}</div>{selectedVersion && <Message.Root severity="success"><Message.Content><Message.Text>Versiunea selectată are proveniență completă și poate fi verificată de server.</Message.Text></Message.Content></Message.Root>}</div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !valid} onClick={() => void save()}>{saving ? "Se înregistrează…" : "Înregistrează dovada"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

export function SignedArtifactEvidenceWorkspace({ client, canManage, canValidate }: { client: ContractClient; canManage: boolean; canValidate: boolean }) {
  const [query, setQuery] = useState<PageQuery>(initialQuery);
  const [page, setPage] = useState<SignaturePage>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [creating, setCreating] = useState(false);
  const [detail, setDetail] = useState<SignatureEvidence>();
  const [revalidating, setRevalidating] = useState<SignatureEvidence>();
  const request = useRef(0);
  const rows: SignatureTableItem[] = useMemo(() => page.items.map((evidence) => ({ id: evidence.id, evidence })), [page.items]);
  const refresh = useCallback(async () => { const current = ++request.current; setLoading(true); setError(undefined); try { const value = requireData(await client.GET("/api/education/signatures", { params: { query: queryWithFilters(query) } }), "signature_evidence"); if (current === request.current) setPage(value); } catch { if (current === request.current) setError("Dovezile de semnătură nu au putut fi încărcate."); } finally { if (current === request.current) setLoading(false); } }, [client, query]);
  useEffect(() => { const timer = window.setTimeout(() => void refresh(), 250); return () => window.clearTimeout(timer); }, [refresh]);
  const filter = (field: string, value: string) => setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } }));
  const sort = (field: string) => setQuery((current) => ({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" }));
  const revalidate = async () => { if (!revalidating) return; setLoading(true); setError(undefined); try { const validation = requireData(await client.POST("/api/education/signatures/{evidenceID}/revalidate", { params: { path: { evidenceID: revalidating.id } } }), "signature_revalidation"); setNotice(validation.validation_status === "error" ? "Verificarea externă nu este configurată; dovada a rămas în stare sigură, fără a fi declarată validă." : `Verificare finalizată: ${signatureStatusLabel(validation.validation_status)}.`); setRevalidating(undefined); await refresh(); } catch { setError("Revalidarea nu a putut fi executată."); } finally { setLoading(false); } };
  const columns = [{ field: "submitted_at", label: "Depusă la" }, { field: "artifact_type", label: "Tip artefact" }, { field: "signature_subject", label: "Subiect certificat" }, { field: "signature_format", label: "Format" }, { field: "signature_level", label: "Nivel" }, { field: "validation_status", label: "Validare" }] as const;
  return <Card.Root><Card.Body><Card.Title>Semnături și dovezi digitale</Card.Title><Card.Subtitle>Dovezi append-only, izolate pe instituție, legate de versiuni eArhivă cu proveniență derivată de server.</Card.Subtitle><Card.Content><div className="flex min-h-0 flex-col gap-3">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}{notice && <Message.Root severity="info"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}<div className="min-h-72 overflow-hidden rounded-border border border-surface"><DataTable.Root data={rows} dataKey="id" scrollable className="max-h-[min(62dvh,44rem)] min-h-64 overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>{columns.map((column) => <DataTable.THeadCell key={column.field}><SortableFilterHeader label={column.label} field={column.field} query={query} onSort={sort} onFilter={filter} /></DataTable.THeadCell>)}<DataTable.THeadCell frozen alignFrozen="right"><div className="flex items-center justify-between gap-2"><span>Acțiuni</span>{canManage && <Button iconOnly rounded size="small" aria-label="Înregistrează dovadă" title="Înregistrează dovadă" onClick={() => setCreating(true)}><i className="pi pi-plus" aria-hidden="true" /></Button>}</div></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const evidence = (item as SignatureTableItem).evidence; const status = evidence.latest_validation?.validation_status; return <DataTable.Row key={evidence.id} index={index}><DataTable.Cell>{displayValue(evidence.submitted_at)}</DataTable.Cell><DataTable.Cell>{artifactTypeLabel(evidence.artifact_type)}</DataTable.Cell><DataTable.Cell>{evidence.signature_subject}</DataTable.Cell><DataTable.Cell>{evidence.signature_format}</DataTable.Cell><DataTable.Cell>{evidence.signature_level === "qualified" ? "Calificată" : "Avansată"}</DataTable.Cell><DataTable.Cell><Tag value={signatureStatusLabel(status)} severity={signatureSeverity(status)} /></DataTable.Cell><DataTable.Cell frozen alignFrozen="right"><Popover.Root><Popover.Trigger asChild><Button iconOnly rounded size="small" variant="text" aria-label="Acțiuni dovadă"><i className="pi pi-ellipsis-v" aria-hidden="true" /></Button></Popover.Trigger><Popover.Portal><Popover.Positioner side="left"><Popover.Popup><Popover.Content><div className="flex flex-col gap-1 p-1"><Button size="small" variant="text" onClick={() => setDetail(evidence)}><i className="pi pi-eye" aria-hidden="true" />Detalii</Button>{canValidate && <Button size="small" variant="text" onClick={() => setRevalidating(evidence)}><i className="pi pi-verified" aria-hidden="true" />Revalidează</Button>}</div></Popover.Content></Popover.Popup></Popover.Positioner></Popover.Portal></Popover.Root></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>{loading && !page.items.length && <div className="flex justify-center p-6" role="status"><Spinner label="Se încarcă dovezile" /></div>}{!loading && !page.items.length && <Message.Root severity="info"><Message.Content><Message.Text>Nu există dovezi pentru filtrele curente.</Message.Text></Message.Content></Message.Root>}</div><TableFooter page={page.page} total={page.total} pageSize={page.pageSize} loading={loading} onPage={(value) => setQuery((current) => ({ ...current, page: value }))} onPageSize={(value) => setQuery((current) => ({ ...current, page: 1, pageSize: value }))} /></div>{creating && <SubmitEvidenceDialog client={client} onClose={() => setCreating(false)} onSaved={refresh} />}{detail && <EvidenceDetails evidence={detail} onClose={() => setDetail(undefined)} />}{revalidating && <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && setRevalidating(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,34rem)]"><Dialog.Header><Dialog.Title>Revalidează dovada</Dialog.Title><Dialog.Close aria-label="Închide confirmarea" /></Dialog.Header><Dialog.Content><Message.Root severity="warn"><Message.Content><Message.Text>Serverul va adăuga o nouă verificare în istoric. Dovada existentă rămâne nemodificată.</Message.Text></Message.Content></Message.Root></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={() => setRevalidating(undefined)}>Renunță</Button><Button disabled={loading} onClick={() => void revalidate()}>Revalidează</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>}</Card.Content></Card.Body></Card.Root>;
}
