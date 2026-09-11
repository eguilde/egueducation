import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Checkbox } from "@primereact/ui/checkbox";
import { Dialog } from "@primereact/ui/dialog";
import { DataTable } from "@primereact/ui/datatable";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Tag } from "@primereact/ui/tag";
import { Textarea } from "@primereact/ui/textarea";
import { Select } from "@primereact/ui/select";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import type { EducationApi, EducationListQuery, EducationPage, EducationRecord, OwnPortfolio, OwnPortfolioArchiveDocument, OwnPortfolioDocumentInput, OwnPortfolioInput, PortfolioDeclarationEvidence, PortfolioDeclarationTemplate, PortfolioDocument, PortfolioSection } from "./types";
import { PortfolioExportManifestPanel } from "./PortfolioExportManifestPanel";
import { PortfolioDocumentVersionHistoryDialog } from "./PortfolioDocumentVersionHistoryDialog";

const today = () => new Date().toISOString().slice(0, 10);
const defaultInput = (): OwnPortfolioInput => ({
  school_year: "",
  last_updated_on: today(),
  notes: "",
});

const editable = (portfolio: OwnPortfolio) =>
  portfolio.status === "draft" || portfolio.status === "returned";

const severity = (status: string): "success" | "warn" | "info" | "danger" | "secondary" => {
  if (status === "validated") return "success";
  if (status === "returned") return "warn";
  if (status === "submitted") return "info";
  if (status === "archived") return "secondary";
  return "secondary";
};

const statusLabel = (status: string) => ({
  draft: "Ciornă", returned: "Returnat pentru completare", submitted: "Trimis spre verificare",
  validated: "Validat", transferred: "Transferat", archived: "Arhivat",
}[status] ?? status);
const declarationLabel = (type: string) => type === "authenticity" ? "Declarație de autenticitate" : "Informare privind datele personale";
const submitErrorMessage = (error: unknown) => {
  const code = error instanceof Error ? error.message : "";
  if (code.includes("incomplete") || code.includes("blocker")) return "Portofoliul nu poate fi trimis: există cerințe obligatorii neîndeplinite. Consultați checklistul.";
  return "Trimiterea nu a putut fi efectuată. Verificați declarațiile și checklistul, apoi încercați din nou.";
};

type RelatedResource = "documents" | "checklist" | "opis" | "reviews";
const relatedLabels: Record<RelatedResource, string> = { documents: "Documente", checklist: "Checklist", opis: "Opis", reviews: "Revizuiri" };

// These flags mirror the closed query contract in EducationApi.  A header
// control is only rendered when this specific own-portfolio endpoint actually
// forwards that field to the server; otherwise it would be a decorative,
// misleading client-side control.
type RelatedColumn = { field: string; label: string; value: (item: EducationRecord) => string; filterable?: boolean; sortable?: boolean };
const relatedColumns: Record<RelatedResource, RelatedColumn[]> = {
  documents: [
    { field: "section_code", label: "Secțiune", value: (item) => String(item.section_code ?? ""), filterable: true, sortable: true },
    { field: "component_code", label: "Componentă", value: (item) => String(item.component_code ?? ""), filterable: true, sortable: true },
    { field: "document_title", label: "Document", value: (item) => String(item.document_title ?? ""), filterable: true, sortable: true },
    { field: "description", label: "Descriere", value: (item) => String(item.description ?? ""), filterable: true, sortable: true },
    { field: "school_year", label: "An școlar", value: (item) => String(item.school_year ?? ""), filterable: true, sortable: true },
    { field: "subject_discipline", label: "Disciplină", value: (item) => String(item.subject_discipline ?? ""), filterable: true, sortable: true },
    { field: "applicable_class", label: "Clasă", value: (item) => String(item.applicable_class ?? ""), filterable: true, sortable: true },
    { field: "competencies", label: "Competențe", value: (item) => Array.isArray(item.competencies) ? item.competencies.join(", ") : String(item.competencies ?? ""), filterable: true },
    { field: "evidence_type", label: "Tip dovadă", value: (item) => String(item.evidence_type ?? ""), filterable: true, sortable: true },
    { field: "issued_on", label: "Data", value: (item) => String(item.issued_on ?? ""), filterable: true, sortable: true },
    { field: "authenticity_status", label: "Stare", value: (item) => String(item.authenticity_status ?? ""), filterable: true, sortable: true },
    { field: "archive_version_no", label: "Versiune arhivă", value: (item) => item.archive_version_no ? `v${String(item.archive_version_no)}` : "—", filterable: true, sortable: true },
    { field: "archive_sha256", label: "Hash arhivă", value: (item) => String(item.archive_sha256 ?? ""), filterable: true },
  ],
  checklist: [
    { field: "requirement_code", label: "Cod", value: (item) => String(item.requirement_code ?? ""), filterable: true, sortable: true },
    { field: "section_code", label: "Secțiune", value: (item) => String(item.section_code ?? ""), sortable: true },
    { field: "status", label: "Stare", value: (item) => String(item.status ?? ""), sortable: true },
    { field: "mandatory", label: "Obligatoriu", value: (item) => item.mandatory ? "Da" : "Nu" },
    { field: "checked_by", label: "Verificat de", value: (item) => String(item.checked_by ?? "") },
  ],
  opis: [
    { field: "section_code", label: "Secțiune", value: (item) => String(item.section_code ?? ""), filterable: true, sortable: true },
    { field: "component_code", label: "Componentă", value: (item) => String(item.component_code ?? ""), sortable: true },
    { field: "entry_title", label: "Document", value: (item) => String(item.entry_title ?? ""), sortable: true },
    { field: "document_reference", label: "Referință", value: (item) => String(item.document_reference ?? ""), sortable: true },
  ],
  reviews: [
    { field: "review_code", label: "Cod", value: (item) => String(item.review_code ?? ""), filterable: true, sortable: true },
    { field: "review_stage", label: "Etapă", value: (item) => String(item.review_stage ?? ""), sortable: true },
    { field: "outcome", label: "Rezultat", value: (item) => String(item.outcome ?? ""), sortable: true },
    { field: "reviewer_name", label: "Evaluator", value: (item) => String(item.reviewer_name ?? ""), sortable: true },
  ],
};

const emptyPage = <T,>(): EducationPage<T> => ({ items: [], total: 0, page: 1, pageSize: 10 });

function TablePager({ page, pageSize, total, disabled, onChange }: { page: number; pageSize: number; total: number; disabled: boolean; onChange: (page: number, pageSize: number) => void }) {
  const lastPage = Math.max(1, Math.ceil(total / pageSize));
  return <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2" aria-label="Paginare">
    <span>{total ? `${(page - 1) * pageSize + 1} - ${Math.min(page * pageSize, total)} din ${total}` : "0 rezultate"}</span>
    <div className="flex items-center gap-2">
      <Select.Root value={pageSize} options={[10, 20, 50, 100].map((value) => ({ label: String(value), value }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => onChange(1, Number(event.value))}>
        <Select.Trigger aria-label="Înregistrări pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger>
        <Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
      </Select.Root>
      <Button iconOnly rounded variant="outlined" severity="secondary" aria-label="Pagina anterioară" disabled={disabled || page <= 1} onClick={() => onChange(page - 1, pageSize)}><i className="pi pi-angle-left" aria-hidden="true" /></Button>
      <span>Pagina {page} din {lastPage}</span>
      <Button iconOnly rounded variant="outlined" severity="secondary" aria-label="Pagina următoare" disabled={disabled || page >= lastPage} onClick={() => onChange(page + 1, pageSize)}><i className="pi pi-angle-right" aria-hidden="true" /></Button>
    </div>
  </div>;
}

function RelatedRecords({ api, portfolioID, resource, canAdd, canDelete, revision, onAdd, onDelete }: { api: EducationApi; portfolioID: string; resource: RelatedResource; canAdd: boolean; canDelete: boolean; revision: number; onAdd?: () => void; onDelete?: (id: string) => void }) {
  const columns = relatedColumns[resource];
  const [data, setData] = useState<EducationPage<EducationRecord>>(emptyPage);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [sort, setSort] = useState<{ field: string; direction: "asc" | "desc" }>({ field: columns[0].field, direction: "asc" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const [versionDocument, setVersionDocument] = useState<Pick<PortfolioDocument, "id" | "document_title">>();
  const ready = useRef(false);
  const query = useMemo<EducationListQuery>(() => ({ page, pageSize, sort: sort.field, direction: sort.direction, filters }), [page, pageSize, sort, filters]);
  const load = useCallback(async (next: EducationListQuery) => {
    setLoading(true); setError(false);
    try { setData(await api.ownPortfolioRelated(portfolioID, resource, next)); }
    catch { setData(emptyPage()); setError(true); }
    finally { setLoading(false); }
  }, [api, portfolioID, resource]);
  useEffect(() => {
    const delay = ready.current ? 350 : 0;
    ready.current = true;
    const timer = window.setTimeout(() => void load(query), delay);
    return () => window.clearTimeout(timer);
  }, [load, query, revision]);
  return <Card.Root><Card.Body><Card.Title>{relatedLabels[resource]}</Card.Title><Card.Content><div className="flex flex-col gap-2">
    {error && <Message.Root severity="error"><Message.Content><Message.Text>Datele nu au putut fi încărcate.</Message.Text></Message.Content></Message.Root>}
    <DataTable.Root data={data.items as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(28rem,55dvh)] min-h-64 overflow-auto">
      <DataTable.Table>
        <DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>
          {columns.map((column) => <DataTable.THeadCell key={column.field}>
            {column.sortable ? <Button variant="text" size="small" aria-label={`Sortează după ${column.label}`} onClick={() => { setPage(1); setSort((current) => ({ field: column.field, direction: current.field === column.field && current.direction === "asc" ? "desc" : "asc" })); }}>{column.label}{sort.field === column.field ? sort.direction === "asc" ? " ↑" : " ↓" : ""}</Button> : <span>{column.label}</span>}
            {column.filterable && <InputText aria-label={`Filtru ${column.label}`} className="mt-1 w-full" value={filters[column.field] ?? ""} placeholder="Filtru" onChange={(event: ChangeEvent<HTMLInputElement>) => { setPage(1); setFilters((current) => ({ ...current, [column.field]: event.target.value })); }} />}
          </DataTable.THeadCell>)}
          <DataTable.THeadCell frozen alignFrozen="right"><span className="flex items-center justify-between gap-2"><span>Acțiuni</span>{resource === "documents" && canAdd && <Button iconOnly rounded size="small" aria-label="Adaugă document" title="Adaugă document" onClick={onAdd}><i className="pi pi-plus" aria-hidden="true" /></Button>}</span></DataTable.THeadCell>
        </DataTable.THeadRow></DataTable.THead>
        <DataTable.TBody>{({ item, index }) => { const record = item as EducationRecord; return <DataTable.Row key={record.id} index={index}>{columns.map((column) => <DataTable.Cell key={column.field}>{column.field === "status" || column.field === "authenticity_status" || column.field === "outcome" ? <Tag value={column.value(record) || "—"} severity={column.value(record) === "validated" || column.value(record) === "complete" ? "success" : "secondary"} /> : column.value(record) || "—"}</DataTable.Cell>)}<DataTable.Cell frozen alignFrozen="right">{resource === "documents" ? <div className="flex items-center gap-1"><Button iconOnly rounded size="small" variant="text" severity="secondary" aria-label={`Istoric versiuni ${String(record.document_title ?? "document")}`} title="Istoric versiuni" onClick={() => setVersionDocument({ id: record.id, document_title: String(record.document_title ?? "Document") })}><i className="pi pi-history" aria-hidden="true" /></Button>{canDelete && <Button iconOnly rounded size="small" variant="outlined" severity="danger" aria-label={`Șterge ${String(record.document_title ?? "document")}`} title="Șterge" onClick={() => onDelete?.(record.id)}><i className="pi pi-trash" aria-hidden="true" /></Button>}</div> : <span>—</span>}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody>
      </DataTable.Table>
    </DataTable.Root>
    {!loading && data.items.length === 0 && <Message.Root severity="info"><Message.Content><Message.Text>Nu există înregistrări pentru filtrele curente.</Message.Text></Message.Content></Message.Root>}
    <TablePager page={page} pageSize={pageSize} total={data.total} disabled={loading} onChange={(nextPage, nextSize) => { setPage(nextPage); setPageSize(nextSize); }} />
    {versionDocument && <PortfolioDocumentVersionHistoryDialog title={`Istoric versiuni — ${versionDocument.document_title}`} load={(query) => api.ownPortfolioDocumentVersions(portfolioID, versionDocument.id, query)} onClose={() => setVersionDocument(undefined)} />}
  </div></Card.Content></Card.Body></Card.Root>;
}

function PortfolioForm({ value, saving, onCancel, onSave }: {
  value: OwnPortfolioInput; saving: boolean; onCancel: () => void; onSave: (value: OwnPortfolioInput) => void;
}) {
  const [form, setForm] = useState(value);
  const set = <K extends keyof OwnPortfolioInput>(key: K, next: OwnPortfolioInput[K]) => setForm((current) => ({ ...current, [key]: next }));
  const valid = Boolean(form.school_year.trim() && form.last_updated_on);
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onCancel()}>
    <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup>
      <Dialog.Header><Dialog.Title>Portofoliu profesional</Dialog.Title><Dialog.Close aria-label="Închide formularul" /></Dialog.Header>
      <Dialog.Content><div className="flex flex-col gap-3">
        <Message.Root severity="info"><Message.Content><Message.Text>Identitatea titularului, instituția, custodia și starea oficială sunt stabilite exclusiv de server. Completați numai conținutul propriului portofoliu.</Message.Text></Message.Content></Message.Root>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-1"><label htmlFor="portfolio-school-year">An școlar *</label><InputText id="portfolio-school-year" value={form.school_year} placeholder="2026-2027" onChange={(event: ChangeEvent<HTMLInputElement>) => set("school_year", event.target.value)} /></div>
          <div className="flex flex-col gap-1"><label htmlFor="portfolio-updated">Data actualizării *</label><InputText id="portfolio-updated" type="date" value={form.last_updated_on} onChange={(event: ChangeEvent<HTMLInputElement>) => set("last_updated_on", event.target.value)} /></div>
        </div>
        <div className="flex flex-col gap-1"><label htmlFor="portfolio-notes">Observații</label><Textarea id="portfolio-notes" value={form.notes} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => set("notes", event.target.value)} /></div>
      </div></Dialog.Content>
      <Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onCancel}>Renunță</Button><Button disabled={saving || !valid} onClick={() => onSave(form)}>{saving ? "Se salvează…" : "Salvează ciorna"}</Button></div></Dialog.Footer>
    </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
  </Dialog.Root>;
}

function PortfolioDocumentForm({ saving, archiveDocuments, portfolioSections, schoolYear, onCancel, onSave }: { saving: boolean; archiveDocuments: readonly OwnPortfolioArchiveDocument[]; portfolioSections: readonly PortfolioSection[]; schoolYear: string; onCancel: () => void; onSave: (input: OwnPortfolioDocumentInput) => void }) {
  const [form, setForm] = useState<OwnPortfolioDocumentInput>({ section_code: "", component_code: "", document_title: "", description: "", school_year: schoolYear, subject_discipline: "", applicable_class: "", competencies: [], evidence_type: "", issued_on: today(), added_on: today(), chronological_index: 1, sensitive_data: false, file_reference: "", notes: "" });
  const [competenciesText, setCompetenciesText] = useState("");
  const set = <K extends keyof OwnPortfolioDocumentInput>(key: K, value: OwnPortfolioDocumentInput[K]) => setForm((current) => ({ ...current, [key]: value }));
  const valid = Boolean(form.section_code.trim() && form.component_code.trim() && form.document_title.trim() && form.description.trim() && form.school_year.trim() && form.subject_discipline.trim() && form.applicable_class.trim() && competenciesText.trim() && form.evidence_type.trim() && form.issued_on && form.added_on && form.file_reference.trim());
  const archiveOptions = archiveDocuments.map((document) => ({ label: `${document.title} · v${document.current_version_no}`, value: `archive://${document.id}` }));
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onCancel()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,56rem)]"><Dialog.Header><Dialog.Title>Adaugă document în portofoliu</Dialog.Title><Dialog.Close aria-label="Închide formular document" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Selectați componenta din catalogul oficial și un document eArhivă autorizat din instituția curentă. Serverul fixează proveniența, versiunea și hash-ul arhivistic.</Message.Text></Message.Content></Message.Root><div className="grid gap-3 sm:grid-cols-2"><div className="flex flex-col gap-1 sm:col-span-2"><label>Componentă din catalog *</label><Select.Root value={form.section_code && form.component_code ? `${form.section_code}::${form.component_code}` : null} options={portfolioSections.map((section) => ({ label: `${String(section.section_code)} · ${String(section.component_code)} — ${String(section.label_ro ?? "")}`, value: `${String(section.section_code)}::${String(section.component_code)}` }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => { const [sectionCode, componentCode] = String(event.value ?? "").split("::"); setForm((current) => ({ ...current, section_code: sectionCode ?? "", component_code: componentCode ?? "", evidence_type: current.evidence_type || "document", sensitive_data: Boolean(portfolioSections.find((section) => `${section.section_code}::${section.component_code}` === event.value)?.sensitive_data) })); }}><Select.Trigger aria-label="Componentă din catalog"><Select.Value placeholder={portfolioSections.length ? "Alegeți componenta legală" : "Catalogul nu este disponibil"} /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div><div className="flex flex-col gap-1"><label htmlFor="document-title">Titlu *</label><InputText id="document-title" value={form.document_title} onChange={(event: ChangeEvent<HTMLInputElement>) => set("document_title", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-evidence">Tip dovadă *</label><InputText id="document-evidence" value={form.evidence_type} onChange={(event: ChangeEvent<HTMLInputElement>) => set("evidence_type", event.target.value)} /></div><div className="flex flex-col gap-1 sm:col-span-2"><label htmlFor="document-description">Descriere pedagogică *</label><Textarea id="document-description" value={form.description} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => set("description", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-school-year">An școlar *</label><InputText id="document-school-year" value={form.school_year} onChange={(event: ChangeEvent<HTMLInputElement>) => set("school_year", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-subject">Disciplina *</label><InputText id="document-subject" value={form.subject_discipline} onChange={(event: ChangeEvent<HTMLInputElement>) => set("subject_discipline", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-class">Clasa aplicabilă *</label><InputText id="document-class" value={form.applicable_class} onChange={(event: ChangeEvent<HTMLInputElement>) => set("applicable_class", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-competencies">Competențe *</label><InputText id="document-competencies" value={competenciesText} placeholder="Separate prin virgulă" onChange={(event: ChangeEvent<HTMLInputElement>) => setCompetenciesText(event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-issued">Data emiterii *</label><InputText id="document-issued" type="date" value={form.issued_on} onChange={(event: ChangeEvent<HTMLInputElement>) => set("issued_on", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-added">Data adăugării *</label><InputText id="document-added" type="date" value={form.added_on} onChange={(event: ChangeEvent<HTMLInputElement>) => set("added_on", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-index">Index cronologic</label><InputText id="document-index" type="number" min="1" value={String(form.chronological_index)} onChange={(event: ChangeEvent<HTMLInputElement>) => set("chronological_index", Math.max(1, Number(event.target.value) || 1))} /></div><div className="flex flex-col gap-1"><label>Document eArhivă autorizat *</label><Select.Root value={form.file_reference || null} options={archiveOptions} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => set("file_reference", String(event.value ?? ""))}><Select.Trigger aria-label="Document eArhivă autorizat"><Select.Value placeholder={archiveOptions.length ? "Alegeți documentul" : "Nu există documente eligibile"} /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div></div><label className="flex items-start gap-2"><Checkbox.Root aria-label="Document cu date sensibile" checked={form.sensitive_data} onCheckedChange={() => set("sensitive_data", !form.sensitive_data)}><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>Documentul conține date cu caracter personal sau alte date sensibile.</span></label><div className="flex flex-col gap-1"><label htmlFor="document-notes">Observații</label><Textarea id="document-notes" value={form.notes} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => set("notes", event.target.value)} /></div></div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onCancel}>Renunță</Button><Button disabled={saving || !valid || archiveOptions.length === 0 || portfolioSections.length === 0} onClick={() => onSave({ ...form, competencies: competenciesText.split(",").map((value) => value.trim()).filter(Boolean) })}>{saving ? "Se salvează…" : "Adaugă document"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

/** Self-service teacher portfolio. Its API only exposes resources owned by the authenticated subject. */
export function TeacherPortfolioWorkspace({ api, canManageOwn }: { api: EducationApi; canManageOwn: boolean }) {
  const [items, setItems] = useState<OwnPortfolio[]>([]);
  const [selected, setSelected] = useState<OwnPortfolio>();
  const [editing, setEditing] = useState<OwnPortfolio>();
  const [creating, setCreating] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [addingDocument, setAddingDocument] = useState(false);
  const [archiveDocuments, setArchiveDocuments] = useState<OwnPortfolioArchiveDocument[]>([]);
  const [portfolioSections, setPortfolioSections] = useState<PortfolioSection[]>([]);
  const [declarations, setDeclarations] = useState<PortfolioDeclarationEvidence>({ templates: [], acknowledgements: [] });
  const [declarationsLoading, setDeclarationsLoading] = useState(false);
  const [declarationTemplate, setDeclarationTemplate] = useState<PortfolioDeclarationTemplate>();
  const [declarationConfirmed, setDeclarationConfirmed] = useState(false);
  const [relatedRevision, setRelatedRevision] = useState(0);
  const [confirmation, setConfirmation] = useState<{ kind: "submit" | "delete"; documentID?: string }>();
  const load = useCallback(async () => {
    setLoading(true); setError(undefined);
    try {
      const result = await api.ownPortfolios({ page: 1, pageSize: 100, sort: "school_year", direction: "desc" });
      setItems(result.items);
      setSelected((current) => result.items.find((item) => item.id === current?.id) ?? result.items[0]);
    } catch { setError("Portofoliul propriu nu a putut fi încărcat. Încercați din nou."); }
    finally { setLoading(false); }
  }, [api]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => {
    let active = true;
    if (!selected) {
      setDeclarations({ templates: [], acknowledgements: [] });
      return () => { active = false; };
    }
    setDeclarationsLoading(true);
    void api.ownPortfolioDeclarations(selected.id).then((result) => {
      if (active) setDeclarations(result);
    }).catch(() => {
      if (active) setDeclarations({ templates: [], acknowledgements: [] });
    }).finally(() => {
      if (active) setDeclarationsLoading(false);
    });
    return () => { active = false; };
  }, [api, selected?.id]);
  useEffect(() => {
    let active = true;
    void Promise.all([
      api.ownPortfolioArchiveDocuments({ page: 1, pageSize: 100, sort: "title", direction: "asc" }),
      api.portfolioSections({ page: 1, pageSize: 100, sort: "sort_order", direction: "asc" }),
    ]).then(([documents, sections]) => {
      if (!active) return;
      setArchiveDocuments(documents.items.filter((document) => document.current_version_no > 0));
      setPortfolioSections(sections.items);
    }).catch(() => { if (active) { setArchiveDocuments([]); setPortfolioSections([]); } });
    return () => { active = false; };
  }, [api]);
  const save = async (input: OwnPortfolioInput) => {
    setSaving(true); setError(undefined);
    try {
      const result = editing ? await api.updateOwnPortfolio(editing.id, input) : await api.createOwnPortfolio(input);
      setEditing(undefined); setCreating(false); setNotice(editing ? "Ciorna a fost actualizată." : "Ciorna a fost creată.");
      await load(); setSelected(result);
    } catch { setError("Portofoliul nu a putut fi salvat. Verificați datele și drepturile active."); }
    finally { setSaving(false); }
  };
  const submit = async () => {
    if (!selected) return;
    setSaving(true); setError(undefined);
    try { const result = await api.submitOwnPortfolio(selected.id); setSelected(result); setItems((current) => current.map((item) => item.id === result.id ? result : item)); setNotice("Portofoliul a fost trimis spre verificare."); setConfirmation(undefined); }
    catch (error) { setError(submitErrorMessage(error)); }
    finally { setSaving(false); }
  };
  const openEdit = async (portfolio: OwnPortfolio) => {
    setError(undefined);
    try { setEditing(await api.ownPortfolio(portfolio.id)); } catch { setError("Detaliile portofoliului nu au putut fi încărcate."); }
  };
  const regenerateOpis = async () => {
    if (!selected) return;
    setSaving(true); setError(undefined);
    try { await api.regenerateOwnPortfolioOpis(selected.id); setNotice("Opisul a fost regenerat."); setRelatedRevision((current) => current + 1); }
    catch { setError("Opisul nu a putut fi regenerat."); }
    finally { setSaving(false); }
  };
  const addDocument = async (input: OwnPortfolioDocumentInput) => {
    if (!selected) return;
    setSaving(true); setError(undefined);
    try { await api.createOwnPortfolioDocument(selected.id, input); setAddingDocument(false); setNotice("Referința documentului a fost adăugată."); setRelatedRevision((current) => current + 1); }
    catch { setError("Documentul nu a putut fi adăugat. Verificați referința și drepturile active."); }
    finally { setSaving(false); }
  };
  const deleteDocument = async (documentID: string) => {
    if (!selected || !editable(selected) || !canManageOwn) return;
    setSaving(true); setError(undefined);
    try { await api.deleteOwnPortfolioDocument(selected.id, documentID); setNotice("Documentul a fost șters din ciornă."); setConfirmation(undefined); setRelatedRevision((current) => current + 1); }
    catch { setError("Documentul nu a putut fi șters."); }
    finally { setSaving(false); }
  };
  const acknowledgeDeclaration = async () => {
    if (!selected || !declarationTemplate || !declarationConfirmed) return;
    setSaving(true); setError(undefined);
    try {
      await api.acknowledgeOwnPortfolioDeclaration(selected.id, declarationTemplate.declaration_type);
      setDeclarations(await api.ownPortfolioDeclarations(selected.id));
      setDeclarationTemplate(undefined); setDeclarationConfirmed(false);
      setNotice("Confirmarea declarației a fost înregistrată cu textul și versiunea afișate.");
    } catch {
      setError("Confirmarea declarației nu a putut fi înregistrată.");
    } finally { setSaving(false); }
  };
  const acknowledged = (template: PortfolioDeclarationTemplate) => declarations.acknowledgements.some((item) => item.declaration_type === template.declaration_type && item.declaration_version === template.declaration_version && item.declaration_text === template.declaration_text);
  const declarationsComplete = declarations.templates.length === 2 && declarations.templates.every(acknowledged);
  const checklist = selected ? [
    ["Secțiuni", `${selected.section_count} raportate de server`, true],
    ["Declarații", declarationsComplete ? "Confirmate" : "Lipsesc confirmări", declarationsComplete],
    ["Actualizare", selected.last_updated_on || "Lipsește", Boolean(selected.last_updated_on)],
  ] as const : [];
  return <section aria-label="Portofoliul meu profesional" className="flex flex-col gap-4">
    <Card.Root><Card.Body><Card.Title>Portofoliul meu profesional</Card.Title><Card.Content><div className="flex flex-col gap-3"><p>Consultați și actualizați numai portofoliile alocate contului autentificat în instituția curentă.</p><div className="flex flex-wrap gap-2"><Button variant="outlined" severity="secondary" disabled={loading || saving} onClick={() => void load()}>Actualizează</Button>{canManageOwn && <Button disabled={saving} onClick={() => setCreating(true)}>Portofoliu nou</Button>}</div>{!canManageOwn && <Message.Root severity="info"><Message.Content><Message.Text>Aveți acces de consultare. Dreptul de modificare a propriului portofoliu nu este activ.</Message.Text></Message.Content></Message.Root>}</div></Card.Content></Card.Body></Card.Root>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {notice && <Message.Root severity="success"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}
    {loading ? <div className="flex justify-center p-6" role="status"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> : items.length === 0 ? <Card.Root><Card.Body><Card.Content><Message.Root severity="info"><Message.Content><Message.Text>Nu există încă un portofoliu profesional pentru contul dvs. {canManageOwn ? "Creați o ciornă pentru anul școlar curent." : "Contactați administratorul instituției dacă trebuie alocat un portofoliu."}</Message.Text></Message.Content></Message.Root></Card.Content></Card.Body></Card.Root> : <div className="grid gap-4 lg:grid-cols-[minmax(16rem,0.8fr)_minmax(0,1.5fr)]"><Card.Root><Card.Body><Card.Title>Portofoliile mele</Card.Title><Card.Content><div className="flex flex-col gap-2">{items.map((item) => <Button key={item.id} variant={selected?.id === item.id ? undefined : "outlined"} severity={selected?.id === item.id ? undefined : "secondary"} onClick={() => setSelected(item)}><span className="flex w-full items-center justify-between gap-2"><span>{item.school_year}</span><Tag value={statusLabel(item.status)} severity={severity(item.status)} /></span></Button>)}</div></Card.Content></Card.Body></Card.Root>
      {selected && <Card.Root><Card.Body><Card.Title>{selected.portfolio_code}</Card.Title><Card.Content><div className="flex flex-col gap-4"><div className="flex flex-wrap items-center justify-between gap-2"><div><strong>{selected.owner_name}</strong><p>{selected.owner_role} · An școlar {selected.school_year}</p></div><Tag value={statusLabel(selected.status)} severity={severity(selected.status)} /></div><div className="grid gap-3 xl:grid-cols-3"><Card.Root><Card.Body><Card.Title>Stare de completare</Card.Title><Card.Content><ul className="flex list-none flex-col gap-2 p-0 m-0">{checklist.map(([label, value, complete]) => <li key={label} className="flex items-center justify-between gap-2"><span>{label}</span><Tag value={value} severity={complete ? "success" : "warn"} /></li>)}</ul></Card.Content></Card.Body></Card.Root><Card.Root><Card.Body><Card.Title>Flux și acțiunea următoare</Card.Title><Card.Content><p>{editable(selected) ? "Completați checklistul, salvați ciorna și trimiteți portofoliul spre verificare." : selected.status === "submitted" ? "Portofoliul este în verificare instituțională." : selected.status === "validated" ? "Portofoliul este validat; îl puteți consulta." : "Așteptați următoarea acțiune a instituției."}</p><p>Custodie: gestionată de instituție · Transfer: {selected.transfer_status || "neinițiat"}</p></Card.Content></Card.Body></Card.Root><Card.Root><Card.Body><Card.Title>Declarații obligatorii</Card.Title><Card.Content><div className="flex flex-col gap-2">{declarationsLoading ? <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root> : declarations.templates.length === 0 ? <Message.Root severity="warn"><Message.Content><Message.Text>Declarațiile oficiale nu sunt disponibile.</Message.Text></Message.Content></Message.Root> : declarations.templates.map((template) => { const accepted = acknowledged(template); return <div key={`${template.declaration_type}:${template.declaration_version}`} className="flex flex-wrap items-center justify-between gap-2"><div><strong>{declarationLabel(template.declaration_type)}</strong><p>Versiunea {template.declaration_version}</p></div>{accepted ? <Tag value="Confirmată" severity="success" /> : <Button size="small" variant="outlined" disabled={!editable(selected) || !canManageOwn || saving} onClick={() => { setDeclarationTemplate(template); setDeclarationConfirmed(false); }}>Citește și confirmă</Button>}</div>; })}</div></Card.Content></Card.Body></Card.Root></div><div className="grid gap-3 xl:grid-cols-2"><RelatedRecords api={api} portfolioID={selected.id} resource="documents" canAdd={Boolean(editable(selected) && canManageOwn && !saving)} canDelete={Boolean(editable(selected) && canManageOwn && !saving)} revision={relatedRevision} onAdd={() => setAddingDocument(true)} onDelete={(id) => setConfirmation({ kind: "delete", documentID: id })} /><RelatedRecords api={api} portfolioID={selected.id} resource="checklist" canAdd={false} canDelete={false} revision={relatedRevision} /><RelatedRecords api={api} portfolioID={selected.id} resource="opis" canAdd={false} canDelete={false} revision={relatedRevision} /><RelatedRecords api={api} portfolioID={selected.id} resource="reviews" canAdd={false} canDelete={false} revision={relatedRevision} /></div>{editable(selected) && canManageOwn && <div className="flex flex-wrap gap-2"><Button size="small" variant="outlined" disabled={saving} onClick={() => void regenerateOpis()}><i className="pi pi-refresh" aria-hidden="true" /> Regenerare opis</Button><Button variant="outlined" severity="secondary" disabled={saving} onClick={() => void openEdit(selected)}><i className="pi pi-pencil" aria-hidden="true" /> Editează ciorna</Button><Button disabled={saving || !declarationsComplete} onClick={() => setConfirmation({ kind: "submit" })}><i className="pi pi-send" aria-hidden="true" /> Trimite spre verificare</Button></div>}</div></Card.Content></Card.Body></Card.Root>}</div>}
    {selected && <PortfolioExportManifestPanel portfolioID={selected.id} api={api} canExport={canManageOwn} />}
    {(creating || editing) && <PortfolioForm value={editing ?? defaultInput()} saving={saving} onCancel={() => { setCreating(false); setEditing(undefined); }} onSave={(value) => void save(value)} />}
    {addingDocument && selected && <PortfolioDocumentForm saving={saving} archiveDocuments={archiveDocuments} portfolioSections={portfolioSections} schoolYear={selected.school_year} onCancel={() => setAddingDocument(false)} onSave={(value) => void addDocument(value)} />}
    {declarationTemplate && <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && setDeclarationTemplate(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>{declarationLabel(declarationTemplate.declaration_type)}</Dialog.Title><Dialog.Close aria-label="Închide declarația" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Text oficial · versiunea {declarationTemplate.declaration_version} · aplicabilă din {declarationTemplate.effective_from}</Message.Text></Message.Content></Message.Root><p>{declarationTemplate.declaration_text}</p><small>Sursă: {declarationTemplate.source_ref}</small><label className="flex items-start gap-2"><Checkbox.Root aria-label="Confirm declarația afișată" checked={declarationConfirmed} onCheckedChange={() => setDeclarationConfirmed((current) => !current)}><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>Confirm în mod explicit textul și versiunea afișate.</span></label></div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={() => setDeclarationTemplate(undefined)}>Renunță</Button><Button disabled={saving || !declarationConfirmed} onClick={() => void acknowledgeDeclaration()}>{saving ? "Se înregistrează…" : "Confirmă declarația"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>}
    {confirmation && <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && setConfirmation(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>{confirmation.kind === "submit" ? "Trimite portofoliul?" : "Elimină documentul?"}</Dialog.Title><Dialog.Close aria-label="Închide confirmarea" /></Dialog.Header><Dialog.Content><Message.Root severity={confirmation.kind === "submit" ? "info" : "warn"}><Message.Content><Message.Text>{confirmation.kind === "submit" ? "După trimitere, ciorna devine disponibilă verificării instituționale și nu mai poate fi modificată până la returnare." : "Referința va fi eliminată din ciornă. Documentul original rămâne în eArhivă."}</Message.Text></Message.Content></Message.Root></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={() => setConfirmation(undefined)}>Renunță</Button><Button severity={confirmation.kind === "delete" ? "danger" : undefined} disabled={saving} onClick={() => confirmation.kind === "submit" ? void submit() : void deleteDocument(confirmation.documentID ?? "")}>{saving ? "Se procesează…" : confirmation.kind === "submit" ? "Confirmă trimiterea" : "Confirmă eliminarea"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>}
  </section>;
}
