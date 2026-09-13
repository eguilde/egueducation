import { useCallback, useEffect, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { FileUpload } from "@primereact/ui/fileupload";
import { Message } from "@primereact/ui/message";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import type { AdmissionApi, AdmissionApplication, AdmissionApplicationDetail, AdmissionArchiveVersion, AdmissionAuthorizationOption, AdmissionCampaign, AdmissionCampaignContext, AdmissionCriterionAssessment, AdmissionDecision, AdmissionLegalArtifactSlot, AdmissionLegalPreparation, AdmissionLegalPreparationArtifact, AdmissionLookup, AdmissionPage, AdmissionQuery, AdmissionSignerAuthorization, AdmissionSignerPermission, ArchiveVersionInput, CampaignInput } from "./api";
import type { AdmissionCapabilities } from "./AdmissionWorkspace";
type View = "campaigns" | "applications" | "outcomes" | "signers";
const initialQuery = (sort: string): AdmissionQuery => ({ page: 1, pageSize: 20, sort, direction: "asc", filters: {} });
const severity = (value: string) => value === "open" || value === "admitted" || value === "accepted" || value === "met" || value === "resolved" ? "success" : value === "rejected" || value === "cancelled" || value === "not_met" ? "danger" : value.includes("review") || value === "waitlisted" ? "warn" : "secondary";
const archiveRef = (item: AdmissionArchiveVersion): ArchiveVersionInput => ({ document_id: item.document_id, version_id: item.version_id });
function usePage<T>(loader: (query: AdmissionQuery) => Promise<AdmissionPage<T>>, initialSort: string) {
    const [query, setQuery] = useState(() => initialQuery(initialSort));
    const [page, setPage] = useState<AdmissionPage<T>>({ items: [], total: 0, page: 1, pageSize: 20 });
    const [error, setError] = useState<string>();
    const reload = useCallback(async () => { try {
        setError(undefined);
        setPage(await loader(query));
    }
    catch {
        setError("Datele nu au putut fi încărcate.");
    } }, [loader, query]);
    useEffect(() => { const timer = window.setTimeout(() => void reload(), 160); return () => window.clearTimeout(timer); }, [reload]);
    return { query, page, error, reload, filter: (field: string, value: string) => setQuery(old => ({ ...old, page: 1, filters: { ...old.filters, [field]: value } })), sort: (field: string) => setQuery(old => ({ ...old, page: 1, sort: field, direction: old.sort === field && old.direction === "asc" ? "desc" : "asc" })), paginate: (page: number, pageSize = query.pageSize) => setQuery(old => ({ ...old, page, pageSize })) };
}
type AnyPage = ReturnType<typeof usePage<unknown>>;
function ColumnHeader({ label, field, page }: {
    label: string;
    field: string;
    page: AnyPage;
}) { return <div className="flex min-w-28 flex-col gap-1"><Button size="small" variant="text" severity="secondary" aria-label={`Sortează după ${label}`} onClick={() => page.sort(field)}>{label}{page.query.sort === field ? page.query.direction === "asc" ? " ↑" : " ↓" : ""}</Button><InputText aria-label={`Filtru ${label}`} value={page.query.filters[field] ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => page.filter(field, e.target.value)}/></div>; }
function Paginator({ page }: {
    page: AnyPage;
}) { const last = Math.max(1, Math.ceil(page.page.total / page.page.pageSize)); return <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2"><span>{page.page.total ? `${(page.page.page - 1) * page.page.pageSize + 1}–${Math.min(page.page.page * page.page.pageSize, page.page.total)} din ${page.page.total}` : "0 rezultate"}</span><div className="flex items-center gap-2"><Select.Root value={String(page.query.pageSize)} options={[10, 20, 50].map(n => ({ label: `${n}/pagină`, value: String(n) }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => page.paginate(1, Number(e.value))}><Select.Trigger aria-label="Rezultate pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button size="small" variant="outlined" disabled={page.query.page <= 1} onClick={() => page.paginate(page.query.page - 1)}>Anterior</Button><Button size="small" variant="outlined" disabled={page.query.page >= last} onClick={() => page.paginate(page.query.page + 1)}>Următor</Button></div></div>; }
function DialogShell({ title, onClose, children }: {
    title: string;
    onClose: () => void;
    children: ReactNode;
}) { return <Dialog.Root open onOpenChange={(e: {
    value?: boolean;
}) => !e.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,46rem)]"><Dialog.Header><Dialog.Title>{title}</Dialog.Title><Dialog.Close aria-label="Închide dialogul"/></Dialog.Header><Dialog.Content>{children}</Dialog.Content></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>; }
function LookupSelect({ label, value, load, onChange }: {
    label: string;
    value: string;
    load: (q: string) => Promise<AdmissionLookup[]>;
    onChange: (id: string) => void;
}) { const [items, setItems] = useState<AdmissionLookup[]>([]); const [q, setQ] = useState(""); useEffect(() => { const timer = window.setTimeout(() => void load(q).then(setItems).catch(() => setItems([])), 150); return () => window.clearTimeout(timer); }, [load, q]); return <label className="flex flex-col gap-1"><span>{label}</span><InputText aria-label={`Caută ${label}`} value={q} onChange={(e: ChangeEvent<HTMLInputElement>) => setQ(e.target.value)}/><Select.Root value={value || null} options={items.filter(x => !x.disabled).map(x => ({ label: x.label, value: x.id }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => onChange(String(e.value ?? ""))}><Select.Trigger aria-label={label}><Select.Value placeholder="Selectați"/><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>; }
function ArchiveSelect({ api, purpose, value, onChange, label = "Versiune arhivă WORM eligibilă *" }: {
    api: AdmissionApi;
    purpose: "application_document" | "decision" | "appeal" | "appeal_resolution";
    value?: AdmissionArchiveVersion;
    onChange: (item?: AdmissionArchiveVersion) => void;
    label?: string;
}) { const [items, setItems] = useState<AdmissionArchiveVersion[]>([]); const [q, setQ] = useState(""); useEffect(() => { const timer = window.setTimeout(() => void api.listArchiveVersions({ purpose, q }).then(xs => setItems(xs.filter(x => x.eligible))).catch(() => setItems([])), 150); return () => window.clearTimeout(timer); }, [api, purpose, q]); return <label className="flex flex-col gap-1"><span>{label}</span><InputText aria-label={`Caută ${label}`} value={q} onChange={(e: ChangeEvent<HTMLInputElement>) => setQ(e.target.value)}/><Select.Root value={value?.version_id ?? null} options={items.map(x => ({ label: x.label, value: x.version_id }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => onChange(items.find(x => x.version_id === e.value))}><Select.Trigger aria-label={label}><Select.Value placeholder="Selectați versiunea păstrată"/><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>; }
function CampaignContextSelect({ api, value, onChange }: {
    api: AdmissionApi;
    value?: AdmissionCampaignContext;
    onChange: (item?: AdmissionCampaignContext) => void;
}) { const [items, setItems] = useState<AdmissionCampaignContext[]>([]); const [q, setQ] = useState(""); useEffect(() => { const timer = window.setTimeout(() => void api.listCampaignContexts({ q }).then(setItems).catch(() => setItems([])), 150); return () => window.clearTimeout(timer); }, [api, q]); return <label className="flex flex-col gap-1"><span>Ofertă, locație și autorizare *</span><InputText aria-label="Caută context autorizat" value={q} onChange={(e: ChangeEvent<HTMLInputElement>) => setQ(e.target.value)}/><Select.Root value={value?.id ?? null} options={items.map(x => ({ label: `${x.label} · ${x.shift}`, value: x.id }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => onChange(items.find(x => x.id === e.value))}><Select.Trigger aria-label="Context autorizat"><Select.Value placeholder="Selectați contextul"/><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>; }
function AuthorizationSelect({ api, value, onChange }: {
    api: AdmissionApi;
    value?: AdmissionAuthorizationOption;
    onChange: (item?: AdmissionAuthorizationOption) => void;
}) {
    const [items, setItems] = useState<AdmissionAuthorizationOption[]>([]);
    const [q, setQ] = useState("");
    useEffect(() => {
        const timer = window.setTimeout(() => void api.listAuthorizations({ q }).then(setItems).catch(() => setItems([])), 150);
        return () => window.clearTimeout(timer);
    }, [api, q]);
    return <label className="flex flex-col gap-1"><span>Autorizare ofertă *</span><InputText aria-label="Caută autorizare" value={q} onChange={(e: ChangeEvent<HTMLInputElement>) => setQ(e.target.value)}/><Select.Root value={value?.id ?? null} options={items.map(x => ({ label: x.label, value: x.id }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => onChange(items.find(x => x.id === e.value))}><Select.Trigger aria-label="Autorizare ofertă"><Select.Value placeholder="Selectați autorizarea"/><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>;
}
function CampaignContextDialog({ api, onClose, onSaved }: {
    api: AdmissionApi;
    onClose: () => void;
    onSaved: () => Promise<void>;
}) {
    const [classID, setClassID] = useState("");
    const [authorization, setAuthorization] = useState<AdmissionAuthorizationOption>();
    const [schoolYear, setSchoolYear] = useState("");
    const [effectiveFrom, setEffectiveFrom] = useState("");
    const [effectiveTo, setEffectiveTo] = useState("");
    const [saving, setSaving] = useState(false);
    const save = async () => {
        if (!classID || !authorization || !schoolYear.trim() || !effectiveFrom)
            return;
        setSaving(true);
        try {
            await api.createCampaignContext({ class_id: classID, offering_id: authorization.offering_id, location_id: authorization.location_id, authorization_id: authorization.id, school_year: schoolYear.trim(), shift: authorization.shift, effective_from: effectiveFrom, effective_to: effectiveTo || undefined });
            await onSaved();
            onClose();
        }
        finally {
            setSaving(false);
        }
    };
    return <DialogShell title="Context autorizat de admitere" onClose={onClose}><div className="flex flex-col gap-3"><LookupSelect label="Clasă" value={classID} load={q => api.listClasses({ q })} onChange={setClassID}/><AuthorizationSelect api={api} value={authorization} onChange={setAuthorization}/><label className="flex flex-col gap-1"><span>An școlar *</span><InputText aria-label="An școlar context" value={schoolYear} onChange={(e: ChangeEvent<HTMLInputElement>) => setSchoolYear(e.target.value)}/></label><label className="flex flex-col gap-1"><span>Valabil de la *</span><InputText aria-label="Valabil de la" type="date" value={effectiveFrom} onChange={(e: ChangeEvent<HTMLInputElement>) => setEffectiveFrom(e.target.value)}/></label><label className="flex flex-col gap-1"><span>Valabil până la</span><InputText aria-label="Valabil până la" type="date" value={effectiveTo} onChange={(e: ChangeEvent<HTMLInputElement>) => setEffectiveTo(e.target.value)}/></label><div className="flex justify-end"><Button disabled={saving || !classID || !authorization || !schoolYear.trim() || !effectiveFrom} onClick={() => void save()}>{saving ? "Se salvează…" : "Creează contextul"}</Button></div></div></DialogShell>;
}
function CampaignWizard({ api, onClose, onSaved }: {
    api: AdmissionApi;
    onClose: () => void;
    onSaved: () => Promise<void>;
}) {
    const [context, setContext] = useState<AdmissionCampaignContext>();
    const [source, setSource] = useState("");
    const [form, setForm] = useState({ code: "", title: "", school_year: "", capacity_limit: "", student_place_limit: "", students_per_group: "", capacity_unit: "students" as "students" | "study_groups", opens_on: "", closes_on: "", decision_due_on: "" });
    const [saving, setSaving] = useState(false);
    const set = (key: keyof typeof form, value: string) => setForm(old => ({ ...old, [key]: value }));
    const groupSize = Number(form.students_per_group);
    const capacity = Number(form.capacity_limit);
    const studentPlaces = Number(form.student_place_limit);
    const validGroupCapacity = form.capacity_unit === "students" || (groupSize >= 1 && studentPlaces <= capacity * groupSize);
    const save = async () => {
        if (!context || !source || !form.code.trim() || !form.title.trim() || !form.school_year.trim() || form.school_year.trim() !== context.school_year || !form.opens_on || !form.closes_on || capacity < 1 || studentPlaces < 1 || !validGroupCapacity)
            return;
        setSaving(true);
        try {
            const input: CampaignInput = { code: form.code.trim(), title: form.title.trim(), school_year: form.school_year.trim(), source_id: source, capacity_limit: capacity, student_place_limit: studentPlaces, capacity_unit: form.capacity_unit, capacity_basis: form.capacity_unit === "study_groups" ? { students_per_group: groupSize } : {}, shift: context.shift, opens_on: form.opens_on, closes_on: form.closes_on, decision_due_on: form.decision_due_on || undefined, offering_id: context.offering_id, location_id: context.location_id, authorization_id: context.authorization_id, class_offering_context_id: context.class_offering_context_id, criteria: [], document_requirements: [] };
            await api.createCampaign(input);
            await onSaved();
            onClose();
        }
        finally {
            setSaving(false);
        }
    };
    return <DialogShell title="Campanie de admitere" onClose={onClose}><div className="flex flex-col gap-3"><CampaignContextSelect api={api} value={context} onChange={item => { setContext(item); if (item) set("school_year", item.school_year); }}/><LookupSelect label="Referință legală" value={source} load={q => api.listRegulatorySources({ q })} onChange={setSource}/>{([["code", "Cod", "text"], ["title", "Titlu", "text"], ["school_year", "An școlar", "text"], ["opens_on", "Deschide la", "date"], ["closes_on", "Închide la", "date"], ["decision_due_on", "Decizie până la", "date"], ["capacity_limit", "Capacitate", "number"], ["student_place_limit", "Locuri elevi", "number"]] as const).map(([key, label, type]) => <label key={key} className="flex flex-col gap-1"><span>{label}</span><InputText aria-label={label} type={type} value={form[key]} onChange={(e: ChangeEvent<HTMLInputElement>) => set(key, e.target.value)}/></label>)}<Select.Root value={form.capacity_unit} options={[{ label: "Elevi", value: "students" }, { label: "Grupe de studiu", value: "study_groups" }]} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("capacity_unit", e.value === "study_groups" ? "study_groups" : "students")}><Select.Trigger aria-label="Unitatea capacității"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>{form.capacity_unit === "study_groups" && <label className="flex flex-col gap-1"><span>Elevi per grupă *</span><InputText aria-label="Elevi per grupă" type="number" value={form.students_per_group} onChange={(e: ChangeEvent<HTMLInputElement>) => set("students_per_group", e.target.value)}/>{!validGroupCapacity && <small>Locurile pentru elevi nu pot depăși numărul de grupe × elevi per grupă.</small>}</label>}<div className="flex justify-end"><Button disabled={saving || !context || !source || !validGroupCapacity} onClick={() => void save()}>{saving ? "Se salvează…" : "Creează campania"}</Button></div></div></DialogShell>;
}
function CampaignConfig({ api, campaign, onClose }: {
    api: AdmissionApi;
    campaign: AdmissionCampaign;
    onClose: () => void;
}) {
    const [mode, setMode] = useState<"criterion" | "document">("criterion");
    const [code, setCode] = useState("");
    const [title, setTitle] = useState("");
    const [saving, setSaving] = useState(false);
    const save = async () => {
        if (!code.trim() || !title.trim())
            return;
        setSaving(true);
        try {
            if (mode === "criterion") {
                const existing = await api.listCriteria(campaign.id, { ...initialQuery("ordinal"), pageSize: 100 });
                const ordinal = Math.max(0, ...existing.items.map(item => item.ordinal)) + 1;
                await api.addCriterion(campaign.id, { code: code.trim(), title: title.trim(), kind: "eligibility", required: true, weight: 0, ordinal, rule_snapshot: {} });
            }
            else {
                const existing = await api.listDocumentRequirements(campaign.id, { ...initialQuery("ordinal"), pageSize: 100 });
                const ordinal = Math.max(0, ...existing.items.map(item => item.ordinal)) + 1;
                await api.addDocumentRequirement(campaign.id, { code: code.trim(), title: title.trim(), required: true, allowed_mime_types: ["application/pdf"], ordinal });
            }
            setCode("");
            setTitle("");
        }
        finally {
            setSaving(false);
        }
    };
    return <DialogShell title={`Configurare · ${campaign.code}`} onClose={onClose}><div className="flex flex-col gap-3"><div className="flex gap-2"><Button variant={mode === "criterion" ? undefined : "outlined"} onClick={() => setMode("criterion")}>Criteriu</Button><Button variant={mode === "document" ? undefined : "outlined"} onClick={() => setMode("document")}>Document</Button></div><InputText aria-label="Cod configurare" value={code} onChange={(e: ChangeEvent<HTMLInputElement>) => setCode(e.target.value)}/><InputText aria-label="Titlu configurare" value={title} onChange={(e: ChangeEvent<HTMLInputElement>) => setTitle(e.target.value)}/><div className="flex justify-end"><Button disabled={saving || !code.trim() || !title.trim()} onClick={() => void save()}>Adaugă</Button></div></div></DialogShell>;
}
function DSSRetentionDialog({ api, capabilities, onClose }: { api: AdmissionApi; capabilities: AdmissionCapabilities; onClose: () => void }) {
    const [current, setCurrent] = useState<Awaited<ReturnType<AdmissionApi["currentDSSRetentionPolicy"]>>>();
    const [loaded, setLoaded] = useState(false);
    const [sourceID, setSourceID] = useState("");
    const [days, setDays] = useState("");
    const [ruleVersionID, setRuleVersionID] = useState("");
    const [effectiveFrom, setEffectiveFrom] = useState(() => new Date().toISOString().slice(0, 10));
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string>();
    useEffect(() => {
        void api.currentDSSRetentionPolicy().then(policy => {
            setCurrent(policy);
            setSourceID(policy.source_id);
            setRuleVersionID(policy.rule_version_id);
            setDays(String(policy.minimum_retention_days));
            setEffectiveFrom(policy.effective_from);
        }).catch(() => setCurrent(undefined)).finally(() => setLoaded(true));
    }, [api]);
    const propose = async () => {
        const minimumRetentionDays = Number(days);
        if (!sourceID || !effectiveFrom || !Number.isInteger(minimumRetentionDays) || minimumRetentionDays < 1 || minimumRetentionDays > 36500)
            return;
        setSaving(true);
        setError(undefined);
        try {
            const rule = await api.proposeRetentionRule({ artifact_kind: "admission_dss", source_id: sourceID, effective_from: effectiveFrom, effective_to: null, minimum_retention_days: minimumRetentionDays });
            setRuleVersionID(rule.id);
        }
        catch {
            setError("Regula de retenție nu a putut fi propusă.");
        }
        finally {
            setSaving(false);
        }
    };
    const approve = async () => {
        if (!ruleVersionID)
            return;
        setSaving(true);
        setError(undefined);
        try {
            await api.approveRetentionRule({ rule_version_id: ruleVersionID });
        }
        catch {
            setError("Aprobarea necesită un al doilea director și o regulă încă propusă.");
        }
        finally {
            setSaving(false);
        }
    };
    const activate = async () => {
        if (!ruleVersionID || !effectiveFrom)
            return;
        setSaving(true);
        setError(undefined);
        try {
            setCurrent(await api.configureDSSRetentionPolicy({ rule_version_id: ruleVersionID, effective_from: effectiveFrom }));
        }
        catch {
            setError("Politica poate fi activată numai dintr-o regulă aprobată și activă.");
        }
        finally {
            setSaving(false);
        }
    };
    return <DialogShell title="Autoritate și politică retenție DSS" onClose={onClose}><div className="flex flex-col gap-3">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}{loaded && <Message.Root severity={current ? "info" : "warn"}><Message.Content><Message.Text>{current ? `Politica activă: minimum ${current.minimum_retention_days} zile, de la ${current.effective_from}.` : "Nu există încă o politică activă. Emiterea deciziilor rămâne blocată până la configurare."}</Message.Text></Message.Content></Message.Root>}{capabilities.retentionManage && <><LookupSelect label="Sursă legală retenție" value={sourceID} load={q => api.listRegulatorySources({ q })} onChange={setSourceID}/><label className="flex flex-col gap-1"><span>Retenție minimă (zile) *</span><InputText aria-label="Retenție minimă zile" type="number" min={1} max={36500} value={days} onChange={(e: ChangeEvent<HTMLInputElement>) => setDays(e.target.value)}/></label></>}<label className="flex flex-col gap-1"><span>Aplicabilă de la *</span><InputText aria-label="Politică aplicabilă de la" type="date" max={new Date().toISOString().slice(0, 10)} value={effectiveFrom} onChange={(e: ChangeEvent<HTMLInputElement>) => setEffectiveFrom(e.target.value)}/></label><label className="flex flex-col gap-1"><span>ID versiune regulă *</span><InputText aria-label="ID versiune regulă retenție" value={ruleVersionID} onChange={(e: ChangeEvent<HTMLInputElement>) => setRuleVersionID(e.target.value)}/></label><Message.Root severity="secondary"><Message.Content><Message.Text>Propunerea și aprobarea trebuie realizate de directori diferiți. Activarea folosește exclusiv regula aprobată; durata și sursa nu pot fi suprascrise.</Message.Text></Message.Content></Message.Root><div className="flex flex-wrap justify-end gap-2">{capabilities.retentionManage && <Button variant="outlined" disabled={saving || !sourceID || !effectiveFrom || !days} onClick={() => void propose()}>Propune regula</Button>}{capabilities.retentionApprove && <Button variant="outlined" disabled={saving || !ruleVersionID} onClick={() => void approve()}>Aprobă regula</Button>}{capabilities.retentionManage && <Button disabled={saving || !ruleVersionID || !effectiveFrom} onClick={() => void activate()}>{saving ? "Se salvează…" : "Activează politica"}</Button>}</div></div></DialogShell>;
}
function CampaignTable({ api, capabilities }: {
    api: AdmissionApi;
    capabilities: AdmissionCapabilities;
}) {
    const page = usePage<AdmissionCampaign>(q => api.listCampaigns(q), "opens_on");
    const [creating, setCreating] = useState(false);
    const [creatingContext, setCreatingContext] = useState(false);
    const [configuringRetention, setConfiguringRetention] = useState(false);
    const [selected, setSelected] = useState<AdmissionCampaign>();
    const nextStatus = (campaign: AdmissionCampaign) => campaign.status === "draft" ? "published" : campaign.status === "published" ? "open" : campaign.status === "open" ? "closed" : campaign.status === "closed" || campaign.status === "cancelled" ? "archived" : undefined;
    const transition = async (campaign: AdmissionCampaign) => {
        const status = nextStatus(campaign);
        if (!status)
            return;
        await api.transitionCampaign(campaign.id, { status, expected_version: campaign.expected_version });
        await page.reload();
    };
    return <><Card.Root><Card.Body><Card.Title>Campanii</Card.Title><Card.Content>{page.error && <Message.Root severity="error"><Message.Content><Message.Text>{page.error}</Message.Text></Message.Content></Message.Root>}<div className="min-h-52 overflow-hidden rounded-border border-surface"><DataTable.Root data={page.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(60dvh,34rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Cod" field="code" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Campanie" field="title" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Stare" field="status" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell>Schimb</DataTable.THeadCell><DataTable.THeadCell><div className="flex flex-wrap justify-end gap-1">{(capabilities.retentionManage || capabilities.retentionApprove) && <Button size="small" variant="outlined" aria-label="Configurează retenția DSS" onClick={() => setConfiguringRetention(true)}>Retenție DSS</Button>}{capabilities.contextManage && <Button size="small" variant="outlined" aria-label="Adaugă context autorizat" onClick={() => setCreatingContext(true)}>Context</Button>}{capabilities.manage && <Button size="small" aria-label="Adaugă campanie" onClick={() => setCreating(true)}>Adaugă</Button>}</div></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as AdmissionCampaign; const status = nextStatus(x); return <DataTable.Row index={index} key={x.id}><DataTable.Cell>{x.code}</DataTable.Cell><DataTable.Cell>{x.title}</DataTable.Cell><DataTable.Cell><Tag value={x.status} severity={severity(x.status)}/></DataTable.Cell><DataTable.Cell>{x.shift}</DataTable.Cell><DataTable.Cell><div className="flex flex-wrap justify-end gap-1">{capabilities.manage && x.status === "draft" && <Button size="small" variant="text" onClick={() => setSelected(x)}>Configurare</Button>}{capabilities.manage && status && <Button size="small" variant="text" aria-label={`${status} ${x.code}`} onClick={() => void transition(x)}>{status === "published" ? "Publică" : status === "open" ? "Deschide" : status === "closed" ? "Închide" : "Arhivează"}</Button>}</div></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={page as AnyPage}/></Card.Content></Card.Body></Card.Root>{configuringRetention && <DSSRetentionDialog api={api} capabilities={capabilities} onClose={() => setConfiguringRetention(false)}/>} {creatingContext && <CampaignContextDialog api={api} onClose={() => setCreatingContext(false)} onSaved={page.reload}/>} {creating && <CampaignWizard api={api} onClose={() => setCreating(false)} onSaved={page.reload}/>} {selected && <CampaignConfig api={api} campaign={selected} onClose={() => setSelected(undefined)}/>}</>;
}
function CreateApplication({ api, onClose, onSaved }: {
    api: AdmissionApi;
    onClose: () => void;
    onSaved: () => Promise<void>;
}) { const [campaign, setCampaign] = useState(""); const [candidate, setCandidate] = useState(""); const [no, setNo] = useState(""); const save = async () => { if (!campaign || !candidate || !no.trim())
    return; await api.createApplication({ campaign_id: campaign, application_no: no.trim(), candidate_party_id: candidate, consent_snapshot: { confirmed: true } }); await onSaved(); onClose(); }; return <DialogShell title="Aplicație nouă" onClose={onClose}><div className="flex flex-col gap-3"><LookupSelect label="Campanie" value={campaign} load={async (q) => (await api.listCampaigns({ ...initialQuery("opens_on"), pageSize: 50, filters: { title: q } })).items.map(x => ({ id: x.id, label: `${x.code} · ${x.title}` }))} onChange={setCampaign}/><LookupSelect label="Candidat" value={candidate} load={q => api.listCandidateParties({ q })} onChange={setCandidate}/><InputText aria-label="Număr aplicație" value={no} onChange={(e: ChangeEvent<HTMLInputElement>) => setNo(e.target.value)}/><div className="flex justify-end"><Button disabled={!campaign || !candidate || !no.trim()} onClick={() => void save()}>Creează aplicația</Button></div></div></DialogShell>; }
function LegalPayload({ preparation }: { preparation: AdmissionLegalPreparation }) {
    return <><LegalPayloadContent preparation={preparation}/><Message.Root severity="info"><Message.Content><Message.Text>Retenție minimă: {preparation.minimum_retention_days} zile. Documentul semnat trebuie păstrat cel puțin până la {new Date(preparation.required_retention_until).toLocaleString("ro-RO")}. Termenul este stabilit de politica aprobată și nu poate fi modificat aici.</Message.Text></Message.Content></Message.Root></>;
}

function LegalPayloadContent({ preparation }: { preparation: AdmissionLegalPreparation }) {
    const download = (name: string, payloadBase64?: string | null) => {
        if (!payloadBase64) return;
        const encoded = atob(payloadBase64);
        const bytes = Uint8Array.from(encoded, character => character.charCodeAt(0));
        const url = URL.createObjectURL(new Blob([bytes], { type: "application/json" }));
        const anchor = document.createElement("a"); anchor.href = url; anchor.download = name; anchor.click(); URL.revokeObjectURL(url);
    };
    return <div className="flex flex-col gap-2" aria-label="Payload juridic pregătit"><Message.Root severity="info"><Message.Content><Message.Text>Semnați exact fiecare payload descărcat în serviciul DSS. O versiune WORM semnată diferită va fi refuzată la finalizare.</Message.Text></Message.Content></Message.Root><div className="rounded-border border border-surface p-3"><div className="flex flex-wrap items-center justify-between gap-2"><div><div className="text-sm font-semibold">Payload principal · SHA-256</div><code className="break-all text-sm">{preparation.canonical_payload_sha256}</code></div><Button size="small" variant="outlined" disabled={!preparation.canonical_payload_base64} onClick={() => download(`admission-${preparation.id}-payload.json`, preparation.canonical_payload_base64)}>Descarcă payload</Button></div></div><pre className="m-0 max-h-56 overflow-auto rounded-border border border-surface p-3 text-xs">{JSON.stringify(preparation.canonical_payload, null, 2)}</pre>{preparation.resulting_decision_payload && <><div className="rounded-border border border-surface p-3"><div className="flex flex-wrap items-center justify-between gap-2"><div><div className="text-sm font-semibold">Decizie rezultată · SHA-256</div><code className="break-all text-sm">{preparation.resulting_decision_payload_sha256}</code></div><Button size="small" variant="outlined" disabled={!preparation.resulting_decision_payload_base64} onClick={() => download(`admission-${preparation.id}-resulting-decision.json`, preparation.resulting_decision_payload_base64)}>Descarcă decizia</Button></div></div><pre className="m-0 max-h-56 overflow-auto rounded-border border border-surface p-3 text-xs">{JSON.stringify(preparation.resulting_decision_payload, null, 2)}</pre></>}<div className="text-sm">Expiră: {preparation.expires_at}</div></div>;
}

const artifactArchiveVersion = (value: AdmissionLegalPreparationArtifact): AdmissionArchiveVersion => {
    return {
        id: value.version.id,
        document_id: value.document.id,
        version_id: value.version.id,
        version_no: value.version.version_no,
        original_file_name: value.document.original_file_name,
        mime_type: value.document.mime_type,
        retention_until: value.retention_until,
        sha256: value.version.source_sha256,
        eligible: value.document.status === "ready",
        label: `${value.document.title || value.document.original_file_name} · v${value.version.version_no}`,
    };
};

const artifactStatus = (value?: AdmissionLegalPreparationArtifact) => value?.document.status;
const newIdempotencyKey = () => globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`;
const artifactKeyName = (preparationID: string, slot: AdmissionLegalArtifactSlot, digest: string) => `admission-worm:${preparationID}:${slot}:${digest}`;
const fileDigest = async (file: File) => {
    const bytes = await file.arrayBuffer();
    const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
    return Array.from(new Uint8Array(digest), value => value.toString(16).padStart(2, "0")).join("");
};

/**
 * A preparation owns each artifact slot. The browser never selects a generic
 * archive version or retention value: it can only submit the signed PDF to
 * that immutable server-side slot and wait for the archived version to be ready.
 */
export function LegalPreparationArtifactUpload({ api, preparationID, slot, label, disabled, onReady }: {
    api: AdmissionApi;
    preparationID: string;
    slot: AdmissionLegalArtifactSlot;
    label: string;
    disabled?: boolean;
    onReady: (archive?: AdmissionArchiveVersion) => void;
}) {
    const [file, setFile] = useState<File>();
    const [artifact, setArtifact] = useState<AdmissionLegalPreparationArtifact>();
    const [idempotencyKey, setIdempotencyKey] = useState<string>();
    const [uploading, setUploading] = useState(false);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string>();
    const [pollGeneration, setPollGeneration] = useState(0);
    const attempts = useRef(0);
    const ready = artifact && artifactStatus(artifact) === "ready" ? artifactArchiveVersion(artifact) : undefined;
    const intentID = artifact?.intent_id;
    const status = artifactStatus(artifact);

    useEffect(() => {
        const controller = new AbortController();
        attempts.current = 0;
        setLoading(true); setError(undefined); setArtifact(undefined); setFile(undefined); setIdempotencyKey(undefined); onReady(undefined);
        void api.getLegalPreparationArtifact(preparationID, slot, controller.signal).then((existing) => {
            if (!controller.signal.aborted && existing) setArtifact(existing);
        }).catch(() => {
            if (!controller.signal.aborted) setError("Documentul asociat nu a putut fi încărcat.");
        }).finally(() => {
            if (!controller.signal.aborted) setLoading(false);
        });
        return () => controller.abort();
    }, [api, onReady, preparationID, slot]);

    useEffect(() => {
        if (!artifact) return;
        if (artifactStatus(artifact) === "ready") { onReady(artifactArchiveVersion(artifact)); return; }
        onReady(undefined);
        if (artifactStatus(artifact) === "failed") { setError("Procesarea documentului a eșuat în arhivă."); return; }
        const controller = new AbortController();
        let timer: number | undefined;
        const poll = async () => {
            try {
                const current = await api.getLegalPreparationArtifact(preparationID, slot, controller.signal);
                if (controller.signal.aborted) return;
                if (!current) { setError("Documentul asociat nu mai este disponibil."); return; }
                setArtifact(current);
                if (artifactStatus(current) === "ready" || artifactStatus(current) === "failed") return;
                attempts.current += 1;
                if (attempts.current >= 20) { setError("Procesarea durează prea mult. Reîncercați verificarea mai târziu."); return; }
                timer = window.setTimeout(() => void poll(), 1500);
            } catch {
                if (!controller.signal.aborted) setError("Starea procesării nu a putut fi verificată.");
            }
        };
        timer = window.setTimeout(() => void poll(), 800);
        return () => { controller.abort(); if (timer !== undefined) window.clearTimeout(timer); };
    }, [api, intentID, onReady, pollGeneration, preparationID, slot, status]);

    const refresh = async () => {
        setLoading(true); setError(undefined);
        try {
            const current = await api.getLegalPreparationArtifact(preparationID, slot);
            if (!current) setError("Documentul asociat nu a fost încă confirmat de server.");
            else { setArtifact(current); if (artifactStatus(current) !== "ready" && artifactStatus(current) !== "failed") setPollGeneration(value => value + 1); }
        } catch { setError("Starea procesării nu a putut fi verificată."); }
        finally { setLoading(false); }
    };
    const select = async (event: { files: File[] }) => {
        const selected = event.files[0];
        if (!selected) return;
        if (selected.type !== "application/pdf" && !/\.pdf$/i.test(selected.name)) { setError("Este acceptat exclusiv un fișier PDF."); return; }
        const digest = await fileDigest(selected);
        const storageKey = artifactKeyName(preparationID, slot, digest);
        const key = globalThis.sessionStorage?.getItem(storageKey) ?? newIdempotencyKey();
        globalThis.sessionStorage?.setItem(storageKey, key);
        setFile(selected); setIdempotencyKey(key); setArtifact(undefined); setError(undefined); onReady(undefined);
    };
    const upload = async () => {
        if (!file || !idempotencyKey) return;
        setUploading(true); setError(undefined);
        try { setArtifact(await api.uploadLegalPreparationArtifact(preparationID, slot, file, idempotencyKey)); }
        catch { setError("Încărcarea a fost refuzată. Reîncercați cu același fișier."); }
        finally { setUploading(false); }
    };
    return <section className="flex flex-col gap-2 rounded-border border border-surface p-3" aria-label={label}>
        <div className="flex flex-wrap items-center justify-between gap-2"><strong>{label}</strong>{artifact && <Tag value={artifactStatus(artifact) ?? "queued"} severity={ready ? "success" : error ? "danger" : "warn"}/>}</div>
        {loading ? <span aria-live="polite">Se verifică documentul asociat…</span> : <><FileUpload.Root name={`${slot}-signed-pdf`} customUpload accept="application/pdf,.pdf" maxFileSize={100 * 1024 * 1024} disabled={disabled || uploading || Boolean(artifact)} onSelect={(event: { files: File[] }) => void select(event)}><FileUpload.Trigger>Alege PDF semnat</FileUpload.Trigger><FileUpload.Content /></FileUpload.Root>{artifact && !ready && <Message.Root severity="warn"><Message.Content><Message.Text>Slotul WORM este deja rezervat; pentru un alt document anulați pregătirea și creați una nouă.</Message.Text></Message.Content></Message.Root>}{file && <span className="truncate">{file.name}</span>}{artifact && <span>Retenție stabilită de server: {new Date(artifact.retention_until).toLocaleString("ro-RO")}</span>}{!ready && <div className="flex flex-wrap gap-2"><Button size="small" disabled={disabled || uploading || !file} onClick={() => void upload()}>{uploading ? "Se încarcă…" : "Încarcă documentul semnat"}</Button>{error && file && <Button size="small" variant="outlined" disabled={uploading} onClick={() => void upload()}>Reîncearcă</Button>}{(error || artifact) && <Button size="small" variant="text" disabled={loading || uploading} onClick={() => void refresh()}>Verifică starea</Button>}</div>}</>}
        {ready && <Message.Root severity="success"><Message.Content><Message.Text>Versiunea WORM este pregătită pentru finalizare.</Message.Text></Message.Content></Message.Root>}
        {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    </section>;
}

function DecisionDialog({ api, detail, onClose, onSaved }: {
    api: AdmissionApi;
    detail: AdmissionApplicationDetail;
    onClose: () => void;
    onSaved: () => Promise<void>;
}) {
    const [outcome, setOutcome] = useState<AdmissionDecision["outcome"]>("admitted");
    const [decisionNo, setDecisionNo] = useState("");
    const [rationale, setRationale] = useState("");
    const [deadline, setDeadline] = useState("");
    const [ranking, setRanking] = useState("");
    const [preparation, setPreparation] = useState<AdmissionLegalPreparation>();
    const [archive, setArchive] = useState<AdmissionArchiveVersion>();
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string>();
    const prepare = async () => {
        if (!decisionNo.trim() || !rationale.trim()) return;
        setSaving(true); setError(undefined);
        try { setPreparation(await api.prepareDecision(detail.application.id, { outcome, decision_no: decisionNo.trim(), rationale: rationale.trim(), appeal_deadline: deadline || undefined, ranking_value: ranking ? Number(ranking) : undefined, expected_version: detail.application.expected_version })); }
        catch { setError("Pregătirea deciziei a eșuat. Reîncărcați datele și încercați din nou."); }
        finally { setSaving(false); }
    };
    const finalize = async () => {
        if (!preparation || !archive) return;
        setSaving(true); setError(undefined);
        try { await api.finalizeDecision(detail.application.id, { preparation_id: preparation.id, archive: archiveRef(archive) }); await onSaved(); onClose(); }
        catch { setError("Finalizarea a fost refuzată. Verificați semnătura, versiunea WORM și perioada de valabilitate."); }
        finally { setSaving(false); }
    };
    const cancel = async () => { if (!preparation) return; setSaving(true); setError(undefined); try { await api.cancelLegalPreparation(preparation.id); onClose(); } catch { setError("Anularea pregătirii a eșuat; rezervarea rămâne activă până la expirare."); } finally { setSaving(false); } };
    return <DialogShell title="Emitere decizie" onClose={onClose}><div className="flex flex-col gap-3"><div className="flex flex-wrap gap-2"><Tag value={preparation ? "2. semnare și finalizare" : "1. date decizie"} severity="info"/><Tag value="3. evidență WORM" severity="secondary"/></div>{!preparation ? <><InputText aria-label="Număr decizie" value={decisionNo} onChange={(e: ChangeEvent<HTMLInputElement>) => setDecisionNo(e.target.value)}/><Select.Root value={outcome} options={["admitted", "waitlisted", "rejected"].map(x => ({ label: x, value: x }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => setOutcome(e.value as AdmissionDecision["outcome"])}><Select.Trigger aria-label="Rezultat decizie"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><InputText aria-label="Motivare decizie" value={rationale} onChange={(e: ChangeEvent<HTMLInputElement>) => setRationale(e.target.value)}/><InputText aria-label="Termen contestație" type="date" value={deadline} onChange={(e: ChangeEvent<HTMLInputElement>) => setDeadline(e.target.value)}/><InputText aria-label="Punctaj/rang" type="number" value={ranking} onChange={(e: ChangeEvent<HTMLInputElement>) => setRanking(e.target.value)}/></> : <><LegalPayload preparation={preparation}/><LegalPreparationArtifactUpload api={api} preparationID={preparation.id} slot="primary" label="PDF semnat · decizie" disabled={saving} onReady={setArchive}/></>}{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="flex flex-wrap justify-between gap-2">{preparation && <Button severity="danger" variant="text" disabled={saving} onClick={() => void cancel()}>Anulează pregătirea</Button>}<Button disabled={saving || (!preparation && (!decisionNo.trim() || !rationale.trim())) || (!!preparation && !archive)} onClick={() => void (preparation ? finalize() : prepare())}>{saving ? "Se procesează…" : preparation ? "Finalizează decizia semnată" : "Pregătește pentru semnare"}</Button></div></div></DialogShell>;
}
function ApplicationDetail({ api, applicationID, capabilities, onClose, onChanged }: {
    api: AdmissionApi;
    applicationID: string;
    capabilities: AdmissionCapabilities;
    onClose: () => void;
    onChanged: () => Promise<void>;
}) { const [detail, setDetail] = useState<AdmissionApplicationDetail>(); const [document, setDocument] = useState<import("./api").AdmissionApplicationDocument>(); const [assessment, setAssessment] = useState<AdmissionCriterionAssessment>(); const [deciding, setDeciding] = useState(false); const [enrolling, setEnrolling] = useState(false); const [appeal, setAppeal] = useState<AdmissionDecision>(); const [archive, setArchive] = useState<AdmissionArchiveVersion>(); const [note, setNote] = useState(""); const [outcome, setOutcome] = useState<Exclude<AdmissionCriterionAssessment["outcome"], "pending">>("met"); const reload = useCallback(async () => setDetail(await api.getApplication(applicationID)), [api, applicationID]); useEffect(() => { void reload(); }, [reload]); if (!detail)
    return <DialogShell title="Aplicație" onClose={onClose}>Se încarcă…</DialogShell>; const app = detail.application; const transition = async (status: "submitted" | "under_review") => { await api.transitionApplication(app.id, { status, expected_version: app.expected_version }); await reload(); await onChanged(); }; const assessDocument = async (status: "accepted" | "waived") => { if (!document || (status === "accepted" && !archive))
    return; await api.assessDocument(app.id, document.id, { status, review_note: note, expected_version: document.expected_version, archive: archive ? archiveRef(archive) : undefined }); setDocument(undefined); setArchive(undefined); setNote(""); await reload(); }; const assessCriterion = async () => { if (!assessment || !note.trim())
    return; await api.assessCriterion(app.id, { criterion_id: assessment.criterion_id, outcome, rationale: note.trim(), evidence_snapshot: {}, expected_version: assessment.expected_version }); setAssessment(undefined); setNote(""); await reload(); }; return <DialogShell title={`Aplicație · ${app.application_no}`} onClose={onClose}><div className="flex flex-col gap-4"><div className="flex flex-wrap gap-2"><Tag value={app.status} severity={severity(app.status)}/>{capabilities.manage && app.status === "draft" && <Button onClick={() => void transition("submitted")}>Depune</Button>}{capabilities.manage && app.status === "submitted" && <Button onClick={() => void transition("under_review")}>Începe verificarea</Button>}{capabilities.decide && app.status === "under_review" && <Button onClick={() => setDeciding(true)}>Emite decizie</Button>}</div><div><h2 className="m-0 text-base font-semibold">Documente necesare</h2>{detail.documents.map(x => <div className="flex items-center justify-between gap-2" key={x.id}><span>{x.document_kind} <Tag value={x.status} severity={severity(x.status)}/></span>{capabilities.manage && ["requested", "submitted"].includes(x.status) && <Button size="small" variant="text" onClick={() => setDocument(x)}>Verifică</Button>}</div>)}</div><div><h2 className="m-0 text-base font-semibold">Criterii</h2>{detail.assessments.map(x => <div className="flex items-center justify-between gap-2" key={x.id}><span>{x.title} <Tag value={x.outcome} severity={severity(x.outcome)}/></span>{capabilities.manage && <Button size="small" variant="text" onClick={() => setAssessment(x)}>Evaluează</Button>}</div>)}</div><div><h2 className="m-0 text-base font-semibold">Decizii</h2>{detail.decisions.map(x => <div className="flex flex-wrap items-center justify-between gap-2" key={x.id}><span>{x.decision_no} · {x.outcome}</span>{capabilities.appealsManage && <Button size="small" variant="text" onClick={() => setAppeal(x)}>Contestație</Button>}{capabilities.manage && x.outcome === "admitted" && app.status === "admitted" && <Button size="small" onClick={() => setEnrolling(true)}>Înmatriculează</Button>}</div>)}</div></div>{document && <DialogShell title="Verificare document" onClose={() => setDocument(undefined)}><div className="flex flex-col gap-3"><InputText aria-label="Notă document" value={note} onChange={(e: ChangeEvent<HTMLInputElement>) => setNote(e.target.value)}/><ArchiveSelect api={api} purpose="application_document" value={archive} onChange={setArchive}/><div className="flex justify-end gap-2"><Button severity="secondary" onClick={() => void assessDocument("waived")}>Dispensă</Button><Button disabled={!archive} onClick={() => void assessDocument("accepted")}>Acceptă</Button></div></div></DialogShell>}{assessment && <DialogShell title="Evaluare criteriu" onClose={() => setAssessment(undefined)}><div className="flex flex-col gap-3"><Select.Root value={outcome} options={["met", "not_met", "not_applicable", "indeterminate"].map(x => ({ label: x, value: x }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => setOutcome(e.value as Exclude<AdmissionCriterionAssessment["outcome"], "pending">)}><Select.Trigger aria-label="Rezultat criteriu"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><InputText aria-label="Motivare criteriu" value={note} onChange={(e: ChangeEvent<HTMLInputElement>) => setNote(e.target.value)}/><div className="flex justify-end"><Button disabled={!note.trim()} onClick={() => void assessCriterion()}>Salvează evaluarea</Button></div></div></DialogShell>}{deciding && <DecisionDialog api={api} detail={detail} onClose={() => setDeciding(false)} onSaved={async () => { await reload(); await onChanged(); }}/>}{appeal && <AppealDialog api={api} app={app} decision={appeal} onClose={() => setAppeal(undefined)} onSaved={reload}/>}{enrolling && <EnrolDialog api={api} app={app} onClose={() => setEnrolling(false)} onSaved={reload}/>}</DialogShell>; }
function AppealDialog({ api, app, decision, onClose, onSaved }: {
    api: AdmissionApi;
    app: AdmissionApplication;
    decision: AdmissionDecision;
    onClose: () => void;
    onSaved: () => Promise<void>;
}) { const [no, setNo] = useState(""); const [statement, setStatement] = useState(""); const [archive, setArchive] = useState<AdmissionArchiveVersion>(); const save = async () => { if (!no.trim() || !statement.trim() || !archive)
    return; await api.createAppeal(app.id, { decision_id: decision.id, appeal_no: no.trim(), submitted_by_party_id: app.candidate_party_id, statement: statement.trim(), archive: archiveRef(archive) }); await onSaved(); onClose(); }; return <DialogShell title="Contestație" onClose={onClose}><div className="flex flex-col gap-3"><InputText aria-label="Număr contestație" value={no} onChange={(e: ChangeEvent<HTMLInputElement>) => setNo(e.target.value)}/><InputText aria-label="Motivare contestație" value={statement} onChange={(e: ChangeEvent<HTMLInputElement>) => setStatement(e.target.value)}/><ArchiveSelect api={api} purpose="appeal" value={archive} onChange={setArchive}/><div className="flex justify-end"><Button disabled={!no.trim() || !statement.trim() || !archive} onClick={() => void save()}>Înregistrează contestația</Button></div></div></DialogShell>; }
function EnrolDialog({ api, app, onClose, onSaved }: {
    api: AdmissionApi;
    app: AdmissionApplication;
    onClose: () => void;
    onSaved: () => Promise<void>;
}) { const [code, setCode] = useState(""); const [from, setFrom] = useState(""); const save = async () => { if (!code.trim() || !from)
    return; await api.enrolApplication(app.id, { student_code: code.trim(), enrolled_from: from, expected_version: app.expected_version }); await onSaved(); onClose(); }; return <DialogShell title="Înmatriculare" onClose={onClose}><div className="flex flex-col gap-3"><InputText aria-label="Cod elev" value={code} onChange={(e: ChangeEvent<HTMLInputElement>) => setCode(e.target.value)}/><InputText aria-label="Dată înmatriculare" type="date" value={from} onChange={(e: ChangeEvent<HTMLInputElement>) => setFrom(e.target.value)}/><div className="flex justify-end"><Button disabled={!code.trim() || !from} onClick={() => void save()}>Înmatriculează</Button></div></div></DialogShell>; }
function ApplicationsTable({ api, capabilities }: {
    api: AdmissionApi;
    capabilities: AdmissionCapabilities;
}) { const page = usePage<AdmissionApplication>(q => api.listApplications(q), "submitted_at"); const [creating, setCreating] = useState(false); const [detail, setDetail] = useState<string>(); return <><Card.Root><Card.Body><Card.Title>Aplicații</Card.Title><Card.Content><div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={page.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(60dvh,34rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Nr. aplicație" field="application_no" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Stare" field="status" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><div className="flex justify-end">{capabilities.manage && <Button size="small" aria-label="Adaugă aplicație" onClick={() => setCreating(true)}>Adaugă</Button>}</div></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as AdmissionApplication; return <DataTable.Row index={index} key={x.id}><DataTable.Cell>{x.application_no}</DataTable.Cell><DataTable.Cell><Tag value={x.status} severity={severity(x.status)}/></DataTable.Cell><DataTable.Cell><Button size="small" variant="text" aria-label={`Deschide ${x.application_no}`} onClick={() => setDetail(x.id)}>Detalii</Button></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={page as AnyPage}/></Card.Content></Card.Body></Card.Root>{creating && <CreateApplication api={api} onClose={() => setCreating(false)} onSaved={page.reload}/>}{detail && <ApplicationDetail api={api} applicationID={detail} capabilities={capabilities} onClose={() => setDetail(undefined)} onChanged={page.reload}/>}</>; }
/** Retained only while old deep links are migrated; it is not rendered. */
function LegacyOutcomesTable({ api, capabilities }: {
    api: AdmissionApi;
    capabilities: AdmissionCapabilities;
}) { const decisions = usePage<AdmissionDecision>(q => api.listDecisions(q), "decided_at"); const appeals = usePage<import("./api").AdmissionAppeal>(q => api.listAppeals(q), "submitted_at"); const [selected, setSelected] = useState<import("./api").AdmissionAppeal>(); const [archive, setArchive] = useState<AdmissionArchiveVersion>(); const [rationale, setRationale] = useState(""); const [outcome, setOutcome] = useState<"upheld" | "partially_upheld" | "dismissed" | "withdrawn">("dismissed"); const [decisionNo, setDecisionNo] = useState(""); const resolve = async () => { if (!selected || !archive || !rationale.trim() || ((outcome === "upheld" || outcome === "partially_upheld") && !decisionNo.trim()))
    return; const detail = await api.getApplication(selected.application_id); await api.resolveAppeal(selected.id, { outcome, rationale: rationale.trim(), application_expected_version: detail.application.expected_version, resulting_decision_no: decisionNo.trim() || undefined, resulting_outcome: outcome === "upheld" || outcome === "partially_upheld" ? "admitted" : undefined, expected_version: selected.expected_version, archive: archiveRef(archive) }); setSelected(undefined); await appeals.reload(); }; return <div className="grid gap-3 xl:grid-cols-2"><Card.Root><Card.Body><Card.Title>Decizii emise</Card.Title><Card.Content><div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={decisions.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(48dvh,28rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Decizie" field="decision_no" page={decisions as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Rezultat" field="outcome" page={decisions as AnyPage}/></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as AdmissionDecision; return <DataTable.Row index={index} key={x.id}><DataTable.Cell>{x.decision_no}</DataTable.Cell><DataTable.Cell><Tag value={x.outcome} severity={severity(x.outcome)}/></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={decisions as AnyPage}/></Card.Content></Card.Body></Card.Root><Card.Root><Card.Body><Card.Title>Contestații</Card.Title><Card.Content><div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={appeals.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(48dvh,28rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Contestație" field="appeal_no" page={appeals as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Stare" field="status" page={appeals as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell>Acțiuni</DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as import("./api").AdmissionAppeal; return <DataTable.Row index={index} key={x.id}><DataTable.Cell>{x.appeal_no}</DataTable.Cell><DataTable.Cell><Tag value={x.status} severity={severity(x.status)}/></DataTable.Cell><DataTable.Cell>{capabilities.appealsManage && (x.status === "submitted" || x.status === "under_review") && <Button size="small" variant="text" onClick={() => setSelected(x)}>Soluționează</Button>}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={appeals as AnyPage}/></Card.Content></Card.Body></Card.Root>{selected && <DialogShell title={`Soluționare · ${selected.appeal_no}`} onClose={() => setSelected(undefined)}><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Serverul verifică separarea: soluționatorul trebuie să fie diferit de emitentul deciziei.</Message.Text></Message.Content></Message.Root><Select.Root value={outcome} options={["dismissed", "upheld", "partially_upheld", "withdrawn"].map(x => ({ label: x, value: x }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => setOutcome(e.value as typeof outcome)}><Select.Trigger aria-label="Rezultat contestație"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>{(outcome === "upheld" || outcome === "partially_upheld") && <InputText aria-label="Număr decizie rezultată" value={decisionNo} onChange={(e: ChangeEvent<HTMLInputElement>) => setDecisionNo(e.target.value)}/>}<InputText aria-label="Motivare soluționare" value={rationale} onChange={(e: ChangeEvent<HTMLInputElement>) => setRationale(e.target.value)}/><ArchiveSelect api={api} purpose="appeal_resolution" value={archive} onChange={setArchive}/><div className="flex justify-end"><Button disabled={!archive || !rationale.trim() || ((outcome === "upheld" || outcome === "partially_upheld") && !decisionNo.trim())} onClick={() => void resolve()}>Înregistrează soluția</Button></div></div></DialogShell>}</div>; }
function AppealResolutionDialog({ api, appeal, onClose, onSaved }: { api: AdmissionApi; appeal: import("./api").AdmissionAppeal; onClose: () => void; onSaved: () => Promise<void> }) {
    const [outcome, setOutcome] = useState<"upheld" | "partially_upheld" | "dismissed" | "withdrawn">("dismissed");
    const [rationale, setRationale] = useState("");
    const [decisionNo, setDecisionNo] = useState("");
    const [preparation, setPreparation] = useState<AdmissionLegalPreparation>();
    const [archive, setArchive] = useState<AdmissionArchiveVersion>();
    const [resultingDecisionArchive, setResultingDecisionArchive] = useState<AdmissionArchiveVersion>();
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string>();
    const prepare = async () => {
        if (!rationale.trim() || ((outcome === "upheld" || outcome === "partially_upheld") && !decisionNo.trim())) return;
        setSaving(true); setError(undefined);
        try {
            const detail = await api.getApplication(appeal.application_id);
            setPreparation(await api.prepareAppealResolution(appeal.id, { outcome, rationale: rationale.trim(), application_expected_version: detail.application.expected_version, resulting_decision_no: decisionNo.trim() || undefined, resulting_outcome: outcome === "upheld" || outcome === "partially_upheld" ? "admitted" : undefined, expected_version: appeal.expected_version }));
        } catch { setError("Pregătirea soluției a eșuat. Verificați starea contestației și drepturile de soluționare."); }
        finally { setSaving(false); }
    };
    const finalize = async () => {
        if (!preparation || !archive) return;
        setSaving(true); setError(undefined);
        try { await api.finalizeAppealResolution(appeal.id, { preparation_id: preparation.id, archive: archiveRef(archive), resulting_decision_archive: resultingDecisionArchive ? archiveRef(resultingDecisionArchive) : undefined }); await onSaved(); onClose(); }
        catch { setError("Finalizarea a fost refuzată. Selectați documentul WORM semnat pentru payloadul afișat."); }
        finally { setSaving(false); }
    };
    const cancel = async () => { if (!preparation) return; setSaving(true); setError(undefined); try { await api.cancelLegalPreparation(preparation.id); onClose(); } catch { setError("Anularea pregătirii a eșuat; rezervarea rămâne activă până la expirare."); } finally { setSaving(false); } };
    const requiresResultingDecision = Boolean(preparation?.resulting_decision_id);
    return <DialogShell title={`Soluționare · ${appeal.appeal_no}`} onClose={onClose}><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Soluționatorul trebuie să fie diferit de emitentul deciziei. Semnătura este verificată contra autorizării active a certificatului.</Message.Text></Message.Content></Message.Root>{!preparation ? <><Select.Root value={outcome} options={["dismissed", "upheld", "partially_upheld", "withdrawn"].map(x => ({ label: x, value: x }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => setOutcome(e.value as typeof outcome)}><Select.Trigger aria-label="Rezultat contestație"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>{(outcome === "upheld" || outcome === "partially_upheld") && <InputText aria-label="Număr decizie rezultată" value={decisionNo} onChange={(e: ChangeEvent<HTMLInputElement>) => setDecisionNo(e.target.value)}/>}<InputText aria-label="Motivare soluționare" value={rationale} onChange={(e: ChangeEvent<HTMLInputElement>) => setRationale(e.target.value)}/></> : <><LegalPayload preparation={preparation}/><LegalPreparationArtifactUpload api={api} preparationID={preparation.id} slot="primary" label="PDF semnat · soluție contestație" disabled={saving} onReady={setArchive}/>{requiresResultingDecision && <LegalPreparationArtifactUpload api={api} preparationID={preparation.id} slot="resulting_decision" label="PDF semnat · decizie rezultată" disabled={saving} onReady={setResultingDecisionArchive}/>}</>}{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="flex flex-wrap justify-between gap-2">{preparation && <Button variant="text" severity="danger" disabled={saving} onClick={() => void cancel()}>Anulează pregătirea</Button>}<Button disabled={saving || (!preparation && (!rationale.trim() || ((outcome === "upheld" || outcome === "partially_upheld") && !decisionNo.trim()))) || (!!preparation && (!archive || (requiresResultingDecision && !resultingDecisionArchive)))} onClick={() => void (preparation ? finalize() : prepare())}>{saving ? "Se procesează…" : preparation ? "Finalizează soluția semnată" : "Pregătește pentru semnare"}</Button></div></div></DialogShell>;
}

function OutcomesTable({ api, capabilities }: { api: AdmissionApi; capabilities: AdmissionCapabilities }) {
    const decisions = usePage<AdmissionDecision>(q => api.listDecisions(q), "decided_at");
    const appeals = usePage<import("./api").AdmissionAppeal>(q => api.listAppeals(q), "submitted_at");
    const [selected, setSelected] = useState<import("./api").AdmissionAppeal>();
    return <div className="grid gap-3 xl:grid-cols-2"><Card.Root><Card.Body><Card.Title>Decizii emise</Card.Title><Card.Content><div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={decisions.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(48dvh,28rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Decizie" field="decision_no" page={decisions as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Rezultat" field="outcome" page={decisions as AnyPage}/></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as AdmissionDecision; return <DataTable.Row index={index} key={x.id}><DataTable.Cell>{x.decision_no}</DataTable.Cell><DataTable.Cell><Tag value={x.outcome} severity={severity(x.outcome)}/></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={decisions as AnyPage}/></Card.Content></Card.Body></Card.Root><Card.Root><Card.Body><Card.Title>Contestații</Card.Title><Card.Content><div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={appeals.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(48dvh,28rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Contestație" field="appeal_no" page={appeals as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Stare" field="status" page={appeals as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell>Acțiuni</DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as import("./api").AdmissionAppeal; return <DataTable.Row index={index} key={x.id}><DataTable.Cell>{x.appeal_no}</DataTable.Cell><DataTable.Cell><Tag value={x.status} severity={severity(x.status)}/></DataTable.Cell><DataTable.Cell>{capabilities.appealsManage && (x.status === "submitted" || x.status === "under_review") && <Button size="small" variant="text" onClick={() => setSelected(x)}>Soluționează</Button>}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={appeals as AnyPage}/></Card.Content></Card.Body></Card.Root>{selected && <AppealResolutionDialog api={api} appeal={selected} onClose={() => setSelected(undefined)} onSaved={appeals.reload}/>}</div>;
}

function SignerAuthorizationDialog({ api, onClose, onSaved }: { api: AdmissionApi; onClose: () => void; onSaved: () => Promise<void> }) {
    const [certificate, setCertificate] = useState(""); const [userID, setUserID] = useState(""); const [permission, setPermissionRaw] = useState<AdmissionSignerPermission>("education.admissions.decide"); const setPermission = (value: string) => { if (value === "education.admissions.decide" || value === "education.admissions.appeals.manage") setPermissionRaw(value); }; const [validUntil, setValidUntil] = useState(""); const [saving, setSaving] = useState(false); const [error, setError] = useState<string>();
    const save = async () => { if (!/^[a-fA-F0-9]{64}$/.test(certificate.trim()) || !userID.trim() || !validUntil) return; setSaving(true); setError(undefined); try { await api.proposeSignerAuthorization({ certificate_sha256: certificate.trim().toLowerCase(), user_id: userID.trim(), permission_code: permission, valid_until: validUntil }); await onSaved(); onClose(); } catch { setError("Propunerea autorizării certificatului a fost refuzată."); } finally { setSaving(false); } };
    return <DialogShell title="Propune autorizare certificat" onClose={onClose}><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Un alt administrator trebuie să aprobe această autorizare. Certificatul este identificat exclusiv prin SHA-256.</Message.Text></Message.Content></Message.Root><InputText aria-label="SHA-256 certificat" placeholder="64 caractere hexazecimale" value={certificate} onChange={(e: ChangeEvent<HTMLInputElement>) => setCertificate(e.target.value)}/><InputText aria-label="Utilizator certificat" value={userID} onChange={(e: ChangeEvent<HTMLInputElement>) => setUserID(e.target.value)}/><InputText aria-label="Permisiune certificat" value={permission} onChange={(e: ChangeEvent<HTMLInputElement>) => setPermission(e.target.value)}/><InputText aria-label="Valabil până la" type="date" value={validUntil} onChange={(e: ChangeEvent<HTMLInputElement>) => setValidUntil(e.target.value)}/>{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="flex justify-end"><Button disabled={saving || !/^[a-fA-F0-9]{64}$/.test(certificate.trim()) || !userID.trim() || !permission.trim() || !validUntil} onClick={() => void save()}>{saving ? "Se salvează…" : "Propune autorizarea"}</Button></div></div></DialogShell>;
}

function LegacySignerAuthorizations({ api }: { api: AdmissionApi }) {
    const page = usePage<AdmissionSignerAuthorization>(q => api.listSignerAuthorizations(q), "valid_until"); const [creating, setCreating] = useState(false); const [error, setError] = useState<string>();
    const approve = async (item: AdmissionSignerAuthorization) => { try { setError(undefined); await api.approveSignerAuthorization({ proposal_id: item.proposal_id || item.id }); await page.reload(); } catch { setError("Aprobarea autorizării a fost refuzată."); } };
    const revoke = async (item: AdmissionSignerAuthorization) => { try { setError(undefined); await api.revokeSignerAuthorization(item.id, { expected_version: item.expected_version, reason: "revocare inițiată din interfața administrativă" }); await page.reload(); } catch { setError("Revocarea autorizării a fost refuzată."); } };
    return <Card.Root><Card.Body><Card.Title>Autorizări certificate DSS</Card.Title><Card.Content>{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={page.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(60dvh,34rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Certificat" field="certificate_sha256" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Stare" field="status" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><div className="flex justify-end"><Button size="small" aria-label="Adaugă autorizare certificat" onClick={() => setCreating(true)}>Adaugă</Button></div></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as AdmissionSignerAuthorization; return <DataTable.Row index={index} key={x.id}><DataTable.Cell><span className="block max-w-52 truncate" title={x.certificate_sha256}>{x.certificate_sha256}</span></DataTable.Cell><DataTable.Cell><Tag value={x.status} severity={severity(x.status)}/></DataTable.Cell><DataTable.Cell><div className="flex justify-end gap-1">{x.status === "proposed" && <Button size="small" variant="text" onClick={() => void approve(x)}>Aprobă</Button>}{x.status === "active" && <Button size="small" severity="danger" variant="text" onClick={() => void revoke(x)}>Revocă</Button>}</div></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={page as AnyPage}/>{creating && <SignerAuthorizationDialog api={api} onClose={() => setCreating(false)} onSaved={page.reload}/>}</Card.Content></Card.Body></Card.Root>;
}

function SignerRevocationDialog({ api, item, onClose, onSaved }: { api: AdmissionApi; item: AdmissionSignerAuthorization; onClose: () => void; onSaved: () => Promise<void> }) {
    const [reason, setReason] = useState(""); const [saving, setSaving] = useState(false); const [error, setError] = useState<string>();
    const revoke = async () => { if (!reason.trim()) return; setSaving(true); setError(undefined); try { await api.revokeSignerAuthorization(item.id, { expected_version: item.expected_version, reason: reason.trim() }); await onSaved(); onClose(); } catch { setError("Revocarea a fost refuzată; actualizați tabelul și încercați din nou."); } finally { setSaving(false); } };
    return <DialogShell title="Revocă autorizare certificat" onClose={onClose}><div className="flex flex-col gap-3"><Message.Root severity="warn"><Message.Content><Message.Text>Revocarea împiedică utilizarea certificatului pentru finalizări viitoare. Operațiile deja finalizate rămân auditate.</Message.Text></Message.Content></Message.Root><InputText aria-label="Motiv revocare" value={reason} onChange={(e: ChangeEvent<HTMLInputElement>) => setReason(e.target.value)}/>{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="flex justify-end"><Button severity="danger" disabled={saving || !reason.trim()} onClick={() => void revoke()}>{saving ? "Se revocă…" : "Confirmă revocarea"}</Button></div></div></DialogShell>;
}

function SignerAuthorizations({ api, capabilities }: { api: AdmissionApi; capabilities: Pick<AdmissionCapabilities, "signerAuthorizationManage" | "signerAuthorizationApprove"> }) {
    const page = usePage<AdmissionSignerAuthorization>(q => api.listSignerAuthorizations(q), "valid_until"); const [creating, setCreating] = useState(false); const [revoking, setRevoking] = useState<AdmissionSignerAuthorization>(); const [error, setError] = useState<string>();
    const approve = async (item: AdmissionSignerAuthorization) => { try { setError(undefined); await api.approveSignerAuthorization({ proposal_id: item.proposal_id || item.id }); await page.reload(); } catch { setError("Aprobarea autorizării a fost refuzată."); } };
    return <Card.Root><Card.Body><Card.Title>Autorizări certificate DSS</Card.Title><Card.Content>{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="min-h-52 overflow-hidden rounded-border border border-surface"><DataTable.Root data={page.page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(60dvh,34rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><ColumnHeader label="Certificat" field="certificate_sha256" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Permisiune" field="permission_code" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><ColumnHeader label="Stare" field="status" page={page as AnyPage}/></DataTable.THeadCell><DataTable.THeadCell><div className="flex justify-end">{capabilities.signerAuthorizationManage && <Button size="small" aria-label="Adaugă autorizare certificat" onClick={() => setCreating(true)}>Adaugă</Button>}</div></DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const x = item as unknown as AdmissionSignerAuthorization; return <DataTable.Row index={index} key={x.id}><DataTable.Cell><span className="block max-w-52 truncate" title={x.certificate_sha256}>{x.certificate_sha256}</span></DataTable.Cell><DataTable.Cell>{x.permission_code}</DataTable.Cell><DataTable.Cell><Tag value={x.status} severity={severity(x.status)}/></DataTable.Cell><DataTable.Cell><div className="flex justify-end gap-1">{capabilities.signerAuthorizationApprove && x.status === "proposed" && <Button size="small" variant="text" onClick={() => void approve(x)}>Aprobă</Button>}{capabilities.signerAuthorizationManage && x.status === "active" && <Button size="small" severity="danger" variant="text" onClick={() => setRevoking(x)}>Revocă</Button>}</div></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div><Paginator page={page as AnyPage}/>{creating && <SignerAuthorizationDialog api={api} onClose={() => setCreating(false)} onSaved={page.reload}/>} {revoking && <SignerRevocationDialog api={api} item={revoking} onClose={() => setRevoking(undefined)} onSaved={page.reload}/>}</Card.Content></Card.Body></Card.Root>;
}

export function AdmissionVertical({ api, capabilities }: {
    api: AdmissionApi;
    capabilities: AdmissionCapabilities;
}) { const [view, setView] = useState<View>("campaigns"); const canManageSigners = Boolean(capabilities.signerAuthorizationManage); const canApproveSigners = Boolean(capabilities.signerAuthorizationApprove); const canAccessSigners = canManageSigners || canApproveSigners; if (!capabilities.read)
    return <Message.Root severity="warn"><Message.Content><Message.Text>Nu aveți dreptul de acces la admitere.</Message.Text></Message.Content></Message.Root>; return <section className="flex min-h-0 flex-col gap-3" aria-label="Admitere"><div className="flex flex-wrap items-center justify-between gap-2"><div><h1 className="m-0 text-xl font-semibold">Admitere</h1><p className="m-0">Campanii, dosare, decizii și contestații</p></div><div className="flex flex-wrap gap-2"><Button variant={view === "campaigns" ? undefined : "outlined"} onClick={() => setView("campaigns")}>Campanii</Button>{capabilities.piiRead && <Button variant={view === "applications" ? undefined : "outlined"} onClick={() => setView("applications")}>Aplicații</Button>}{capabilities.piiRead && <Button variant={view === "outcomes" ? undefined : "outlined"} onClick={() => setView("outcomes")}>Decizii și contestații</Button>}{canAccessSigners && <Button variant={view === "signers" ? undefined : "outlined"} onClick={() => setView("signers")}>Certificate DSS</Button>}</div></div>{view === "campaigns" ? <CampaignTable api={api} capabilities={capabilities}/> : view === "applications" && capabilities.piiRead ? <ApplicationsTable api={api} capabilities={capabilities}/> : view === "outcomes" && capabilities.piiRead ? <OutcomesTable api={api} capabilities={capabilities}/> : view === "signers" && canAccessSigners ? <SignerAuthorizations api={api} capabilities={{ signerAuthorizationManage: canManageSigners, signerAuthorizationApprove: canApproveSigners }}/> : <Message.Root severity="warn"><Message.Content><Message.Text>Accesul la datele personale din admitere necesită și dreptul de citire Registratură.</Message.Text></Message.Content></Message.Root>}</section>; }
