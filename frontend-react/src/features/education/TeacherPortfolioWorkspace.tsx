import { useCallback, useEffect, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Checkbox } from "@primereact/ui/checkbox";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Tag } from "@primereact/ui/tag";
import { Textarea } from "@primereact/ui/textarea";
import { Select } from "@primereact/ui/select";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import type { EducationApi, EducationRecord, OwnPortfolio, OwnPortfolioArchiveDocument, OwnPortfolioDocumentInput, OwnPortfolioInput } from "./types";

const today = () => new Date().toISOString().slice(0, 10);
const defaultInput = (): OwnPortfolioInput => ({
  school_year: "",
  section_count: 0,
  last_updated_on: today(),
  authenticity_declared: false,
  consent_captured: false,
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
const submitErrorMessage = (error: unknown) => {
  const code = error instanceof Error ? error.message : "";
  if (code.includes("incomplete") || code.includes("blocker")) return "Portofoliul nu poate fi trimis: există cerințe obligatorii neîndeplinite. Consultați checklistul.";
  return "Trimiterea nu a putut fi efectuată. Verificați declarațiile și checklistul, apoi încercați din nou.";
};

type RelatedResource = "documents" | "checklist" | "opis" | "reviews";
const relatedLabels: Record<RelatedResource, string> = { documents: "Documente", checklist: "Checklist", opis: "Opis", reviews: "Revizuiri" };

function RelatedRecords({ resource, items, canDelete, onDelete }: { resource: RelatedResource; items: readonly Record<string, unknown>[]; canDelete: boolean; onDelete?: (id: string) => void }) {
  return <Card.Root><Card.Body><Card.Title>{relatedLabels[resource]}</Card.Title><Card.Content><div className="flex flex-col gap-2">{items.length === 0 ? <p>Nu există înregistrări vizibile.</p> : items.map((item) => {
    const id = String(item.id ?? "");
    const title = String(item.document_title ?? item.requirement_label ?? item.opis_title ?? item.review_code ?? item.title ?? id);
    const detail = String(item.section_code ?? item.requirement_code ?? item.component_code ?? item.review_stage ?? "");
    const status = String(item.status ?? item.authenticity_status ?? item.validation_status ?? "");
    return <Card.Root key={id}><Card.Body><Card.Content><div className="flex flex-wrap items-center justify-between gap-2"><div><strong>{title}</strong>{detail && <p>{detail}</p>}{resource === "documents" && Boolean(item.file_reference) && <p>Referință: {String(item.file_reference)}</p>}</div><div className="flex items-center gap-2">{status && <Tag value={status} severity={status === "validated" || status === "complete" ? "success" : "secondary"} />}{canDelete && resource === "documents" && <Button size="small" severity="danger" variant="outlined" onClick={() => onDelete?.(id)}>Șterge</Button>}</div></div></Card.Content></Card.Body></Card.Root>;
  })}</div></Card.Content></Card.Body></Card.Root>;
}

function PortfolioForm({ value, saving, onCancel, onSave }: {
  value: OwnPortfolioInput; saving: boolean; onCancel: () => void; onSave: (value: OwnPortfolioInput) => void;
}) {
  const [form, setForm] = useState(value);
  const set = <K extends keyof OwnPortfolioInput>(key: K, next: OwnPortfolioInput[K]) => setForm((current) => ({ ...current, [key]: next }));
  const valid = Boolean(form.school_year.trim() && form.last_updated_on && form.authenticity_declared && form.consent_captured);
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
        <label className="flex items-start gap-2"><Checkbox.Root aria-label="Declarație de autenticitate" checked={form.authenticity_declared} onCheckedChange={() => set("authenticity_declared", !form.authenticity_declared)}><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>Declar autenticitatea documentelor și informațiilor încărcate. *</span></label>
        <label className="flex items-start gap-2"><Checkbox.Root aria-label="Acord prelucrare date" checked={form.consent_captured} onCheckedChange={() => set("consent_captured", !form.consent_captured)}><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>Confirm prelucrarea datelor în scopul administrării și evaluării portofoliului. *</span></label>
      </div></Dialog.Content>
      <Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onCancel}>Renunță</Button><Button disabled={saving || !valid} onClick={() => onSave(form)}>{saving ? "Se salvează…" : "Salvează ciorna"}</Button></div></Dialog.Footer>
    </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
  </Dialog.Root>;
}

function PortfolioDocumentForm({ saving, archiveDocuments, onCancel, onSave }: { saving: boolean; archiveDocuments: readonly OwnPortfolioArchiveDocument[]; onCancel: () => void; onSave: (input: OwnPortfolioDocumentInput) => void }) {
  const [form, setForm] = useState<OwnPortfolioDocumentInput>({ section_code: "", component_code: "", document_title: "", evidence_type: "", issued_on: today(), added_on: today(), chronological_index: 1, sensitive_data: false, file_reference: "", notes: "" });
  const set = <K extends keyof OwnPortfolioDocumentInput>(key: K, value: OwnPortfolioDocumentInput[K]) => setForm((current) => ({ ...current, [key]: value }));
  const valid = Boolean(form.section_code.trim() && form.component_code.trim() && form.document_title.trim() && form.evidence_type.trim() && form.issued_on && form.added_on && form.file_reference.trim());
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onCancel()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Adaugă document în portofoliu</Dialog.Title><Dialog.Close aria-label="Închide formular document" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Selectați un document eArhivă autorizat, din instituția curentă. Serverul verifică existența și versiunea stocată curentă.</Message.Text></Message.Content></Message.Root><div className="grid gap-3 sm:grid-cols-2"><div className="flex flex-col gap-1"><label htmlFor="document-section">Secțiune *</label><InputText id="document-section" value={form.section_code} onChange={(event: ChangeEvent<HTMLInputElement>) => set("section_code", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-component">Componentă *</label><InputText id="document-component" value={form.component_code} onChange={(event: ChangeEvent<HTMLInputElement>) => set("component_code", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-title">Titlu *</label><InputText id="document-title" value={form.document_title} onChange={(event: ChangeEvent<HTMLInputElement>) => set("document_title", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-evidence">Tip dovadă *</label><InputText id="document-evidence" value={form.evidence_type} onChange={(event: ChangeEvent<HTMLInputElement>) => set("evidence_type", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-issued">Data emiterii *</label><InputText id="document-issued" type="date" value={form.issued_on} onChange={(event: ChangeEvent<HTMLInputElement>) => set("issued_on", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-added">Data adăugării *</label><InputText id="document-added" type="date" value={form.added_on} onChange={(event: ChangeEvent<HTMLInputElement>) => set("added_on", event.target.value)} /></div><div className="flex flex-col gap-1"><label htmlFor="document-index">Index cronologic</label><InputText id="document-index" type="number" min="1" value={String(form.chronological_index)} onChange={(event: ChangeEvent<HTMLInputElement>) => set("chronological_index", Math.max(1, Number(event.target.value) || 1))} /></div><div className="flex flex-col gap-1"><label>Document eArhivă autorizat *</label><Select.Root value={form.file_reference || null} options={archiveDocuments.map((document) => ({ label: `${document.title} (${document.id})`, value: `archive://${document.id}` }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => set("file_reference", String(event.value ?? ""))}><Select.Trigger aria-label="Document eArhivă autorizat"><Select.Value placeholder={archiveDocuments.length ? "Alegeți documentul" : "Nu există documente eligibile"} /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div></div><label className="flex items-start gap-2"><Checkbox.Root aria-label="Document cu date sensibile" checked={form.sensitive_data} onCheckedChange={() => set("sensitive_data", !form.sensitive_data)}><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>Documentul conține date cu caracter personal sau alte date sensibile.</span></label><div className="flex flex-col gap-1"><label htmlFor="document-notes">Observații</label><Textarea id="document-notes" value={form.notes} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => set("notes", event.target.value)} /></div></div></Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onCancel}>Renunță</Button><Button disabled={saving || !valid || archiveDocuments.length === 0} onClick={() => onSave(form)}>{saving ? "Se salvează…" : "Adaugă document"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
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
  const [related, setRelated] = useState<Record<RelatedResource, EducationRecord[]>>({ documents: [], checklist: [], opis: [], reviews: [] });
  const load = useCallback(async () => {
    setLoading(true); setError(undefined);
    try {
      const result = await api.ownPortfolios();
      setItems(result.items);
      setSelected((current) => result.items.find((item) => item.id === current?.id) ?? result.items[0]);
    } catch { setError("Portofoliul propriu nu a putut fi încărcat. Încercați din nou."); }
    finally { setLoading(false); }
  }, [api]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => {
    let active = true;
    void api.ownPortfolioArchiveDocuments().then((result) => {
      if (active) setArchiveDocuments(result.items.filter((document) => document.current_version_no > 0));
    }).catch(() => { if (active) setArchiveDocuments([]); });
    return () => { active = false; };
  }, [api]);
  useEffect(() => {
    if (!selected) { setRelated({ documents: [], checklist: [], opis: [], reviews: [] }); return; }
    let active = true;
    void Promise.all((["documents", "checklist", "opis", "reviews"] as RelatedResource[]).map(async (resource) => [resource, (await api.ownPortfolioRelated(selected.id, resource)).items] as const))
      .then((entries) => { if (active) setRelated(Object.fromEntries(entries) as Record<RelatedResource, EducationRecord[]>); })
      .catch(() => { if (active) setRelated({ documents: [], checklist: [], opis: [], reviews: [] }); });
    return () => { active = false; };
  }, [api, selected?.id]);
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
    try { const result = await api.submitOwnPortfolio(selected.id); setSelected(result); setItems((current) => current.map((item) => item.id === result.id ? result : item)); setNotice("Portofoliul a fost trimis spre verificare."); }
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
    try { await api.regenerateOwnPortfolioOpis(selected.id); setNotice("Opisul a fost regenerat."); const result = await api.ownPortfolioRelated(selected.id, "opis"); setRelated((current) => ({ ...current, opis: result.items })); }
    catch { setError("Opisul nu a putut fi regenerat."); }
    finally { setSaving(false); }
  };
  const addDocument = async (input: OwnPortfolioDocumentInput) => {
    if (!selected) return;
    setSaving(true); setError(undefined);
    try { await api.createOwnPortfolioDocument(selected.id, input); setAddingDocument(false); setNotice("Referința documentului a fost adăugată."); const result = await api.ownPortfolioRelated(selected.id, "documents"); setRelated((current) => ({ ...current, documents: result.items })); }
    catch { setError("Documentul nu a putut fi adăugat. Verificați referința și drepturile active."); }
    finally { setSaving(false); }
  };
  const deleteDocument = async (documentID: string) => {
    if (!selected || !editable(selected) || !canManageOwn) return;
    setSaving(true); setError(undefined);
    try { await api.deleteOwnPortfolioDocument(selected.id, documentID); setNotice("Documentul a fost șters din ciornă."); const result = await api.ownPortfolioRelated(selected.id, "documents"); setRelated((current) => ({ ...current, documents: result.items })); }
    catch { setError("Documentul nu a putut fi șters."); }
    finally { setSaving(false); }
  };
  const checklist = selected ? [
    ["Secțiuni", `${selected.section_count} raportate de server`, true],
    ["Autenticitate", selected.authenticity_declared ? "Declarată" : "Lipsește", selected.authenticity_declared],
    ["Acord date", selected.consent_captured ? "Confirmat" : "Lipsește", selected.consent_captured],
    ["Actualizare", selected.last_updated_on || "Lipsește", Boolean(selected.last_updated_on)],
  ] as const : [];
  return <section aria-label="Portofoliul meu profesional" className="flex flex-col gap-4">
    <Card.Root><Card.Body><Card.Title>Portofoliul meu profesional</Card.Title><Card.Content><div className="flex flex-col gap-3"><p>Consultați și actualizați numai portofoliile alocate contului autentificat în instituția curentă.</p><div className="flex flex-wrap gap-2"><Button variant="outlined" severity="secondary" disabled={loading || saving} onClick={() => void load()}>Actualizează</Button>{canManageOwn && <Button disabled={saving} onClick={() => setCreating(true)}>Portofoliu nou</Button>}</div>{!canManageOwn && <Message.Root severity="info"><Message.Content><Message.Text>Aveți acces de consultare. Dreptul de modificare a propriului portofoliu nu este activ.</Message.Text></Message.Content></Message.Root>}</div></Card.Content></Card.Body></Card.Root>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {notice && <Message.Root severity="success"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}
    {loading ? <div className="flex justify-center p-6" role="status"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> : items.length === 0 ? <Card.Root><Card.Body><Card.Content><Message.Root severity="info"><Message.Content><Message.Text>Nu există încă un portofoliu profesional pentru contul dvs. {canManageOwn ? "Creați o ciornă pentru anul școlar curent." : "Contactați administratorul instituției dacă trebuie alocat un portofoliu."}</Message.Text></Message.Content></Message.Root></Card.Content></Card.Body></Card.Root> : <div className="grid gap-4 lg:grid-cols-[minmax(16rem,0.8fr)_minmax(0,1.5fr)]"><Card.Root><Card.Body><Card.Title>Portofoliile mele</Card.Title><Card.Content><div className="flex flex-col gap-2">{items.map((item) => <Button key={item.id} variant={selected?.id === item.id ? undefined : "outlined"} severity={selected?.id === item.id ? undefined : "secondary"} onClick={() => setSelected(item)}><span className="flex w-full items-center justify-between gap-2"><span>{item.school_year}</span><Tag value={statusLabel(item.status)} severity={severity(item.status)} /></span></Button>)}</div></Card.Content></Card.Body></Card.Root>
      {selected && <Card.Root><Card.Body><Card.Title>{selected.portfolio_code}</Card.Title><Card.Content><div className="flex flex-col gap-4"><div className="flex flex-wrap items-center justify-between gap-2"><div><strong>{selected.owner_name}</strong><p>{selected.owner_role} · An școlar {selected.school_year}</p></div><Tag value={statusLabel(selected.status)} severity={severity(selected.status)} /></div><div className="grid gap-3 sm:grid-cols-2"><Card.Root><Card.Body><Card.Title>Stare de completare</Card.Title><Card.Content><ul className="flex list-none flex-col gap-2 p-0 m-0">{checklist.map(([label, value, complete]) => <li key={label} className="flex items-center justify-between gap-2"><span>{label}</span><Tag value={value} severity={complete ? "success" : "warn"} /></li>)}</ul></Card.Content></Card.Body></Card.Root><Card.Root><Card.Body><Card.Title>Flux și acțiunea următoare</Card.Title><Card.Content><p>{editable(selected) ? "Completați checklistul, salvați ciorna și trimiteți portofoliul spre verificare." : selected.status === "submitted" ? "Portofoliul este în verificare instituțională." : selected.status === "validated" ? "Portofoliul este validat; îl puteți consulta." : "Așteptați următoarea acțiune a instituției."}</p><p>Custodie: gestionată de instituție · Transfer: {selected.transfer_status || "neinițiat"}</p></Card.Content></Card.Body></Card.Root></div><div className="grid gap-3 lg:grid-cols-2"><RelatedRecords resource="documents" items={related.documents} canDelete={Boolean(editable(selected) && canManageOwn && !saving)} onDelete={(id) => void deleteDocument(id)} /><RelatedRecords resource="checklist" items={related.checklist} canDelete={false} /><RelatedRecords resource="opis" items={related.opis} canDelete={false} /><RelatedRecords resource="reviews" items={related.reviews} canDelete={false} /></div>{editable(selected) && canManageOwn && <div className="flex flex-wrap gap-2"><Button size="small" variant="outlined" disabled={saving} onClick={() => setAddingDocument(true)}>Adaugă document</Button><Button size="small" variant="outlined" disabled={saving} onClick={() => void regenerateOpis()}>Regenerează opisul</Button><Button variant="outlined" severity="secondary" disabled={saving} onClick={() => void openEdit(selected)}>Editează ciorna</Button><Button disabled={saving || !selected.authenticity_declared || !selected.consent_captured} onClick={() => void submit()}>{saving ? "Se trimite…" : "Trimite spre verificare"}</Button></div>}</div></Card.Content></Card.Body></Card.Root>}</div>}
    {(creating || editing) && <PortfolioForm value={editing ?? defaultInput()} saving={saving} onCancel={() => { setCreating(false); setEditing(undefined); }} onSave={(value) => void save(value)} />}
    {addingDocument && <PortfolioDocumentForm saving={saving} archiveDocuments={archiveDocuments} onCancel={() => setAddingDocument(false)} onSave={(value) => void addDocument(value)} />}
  </section>;
}
