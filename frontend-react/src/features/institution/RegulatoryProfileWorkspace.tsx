import { useCallback, useEffect, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import { Pencil, Refresh } from "@primeicons/react";
import { useAuth } from "../../auth/AuthProvider";
import type { InstitutionCapabilities, InstitutionPolicyApi, PolicyCutoverPreflight, PutRegulatoryProfileInput, RegulatoryProfile } from "./api";
import { useInstitutionPolicy } from "./InstitutionPolicyProvider";
import { EducationOfferingsWorkspace } from "./EducationOfferingsWorkspace";

const legalForms = [
  { label: "Școală publică", value: "public" },
  { label: "Școală privată", value: "private" },
];
const statuses = [
  { label: "Ciornă", value: "draft" },
  { label: "Aprobat", value: "approved" },
  { label: "Activ", value: "active" },
];
const authorizationStatuses = [
  { label: "Necunoscut", value: "unknown" }, { label: "Provizoriu", value: "provisional" },
  { label: "Autorizat", value: "authorized" }, { label: "Acreditat", value: "accredited" },
  { label: "Suspendat", value: "suspended" }, { label: "Retras", value: "withdrawn" },
];
const sourceKinds = [
  { label: "Lege", value: "law" }, { label: "Hotărâre de Guvern", value: "government_decision" },
  { label: "Ordin ministerial", value: "ministerial_order" }, { label: "Autorizație", value: "authorization" },
  { label: "Acreditare", value: "accreditation" }, { label: "Decizie fondator", value: "founder_decision" },
  { label: "Contract", value: "contract" }, { label: "Alt act", value: "other" },
];
const boolOptions = [{ label: "Da", value: "true" }, { label: "Nu", value: "false" }];
const spinner = <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;

function today() { return new Date().toISOString().slice(0, 10); }
function asInput(profile: RegulatoryProfile): PutRegulatoryProfileInput {
  return {
    expected_version: profile.version,
    status: profile.status === "active" || profile.status === "approved" || profile.status === "draft" ? profile.status : "draft",
    school_legal_form: profile.school_legal_form === "public" || profile.school_legal_form === "private" ? profile.school_legal_form : "" as PutRegulatoryProfileInput["school_legal_form"],
    regulatory_profile: profile.regulatory_profile,
    authorization_status: ["provisional", "authorized", "accredited", "suspended", "withdrawn"].includes(profile.authorization_status) ? profile.authorization_status as PutRegulatoryProfileInput["authorization_status"] : "unknown",
    accreditation_reference: profile.accreditation_reference,
    authorized_levels: profile.authorized_levels,
    has_legal_personality: profile.has_legal_personality,
    tax_identifier: profile.tax_identifier,
    founder_name: profile.founder_name,
    funder_name: profile.funder_name,
    budget_authority_name: profile.budget_authority_name,
    accounting_profile: profile.accounting_profile,
    procurement_profile: profile.procurement_profile,
    payroll_profile: profile.payroll_profile,
    vat_profile: profile.vat_profile,
    program_codes: profile.program_codes,
    effective_from: profile.effective_from ?? today(),
    effective_to: profile.effective_to,
    source: { source_kind: "other", citation: profile.source_reference, article_reference: "", issuer: "", source_url: "", checksum_sha256: "" },
  };
}

function labelLegalForm(value: string | null) {
  return legalForms.find((item) => item.value === value)?.label ?? "Neclasificată";
}

export function RegulatoryProfileWorkspace({ api, canManage, canReadOfferings = false, canManageOfferings = false }: { api: InstitutionPolicyApi; canManage: boolean; canReadOfferings?: boolean; canManageOfferings?: boolean }) {
  const policy = useInstitutionPolicy();
  const { session } = useAuth();
  const [profile, setProfile] = useState<RegulatoryProfile>();
  const [capabilities, setCapabilities] = useState<InstitutionCapabilities>();
  const [preflight, setPreflight] = useState<PolicyCutoverPreflight>();
  const [preflightError, setPreflightError] = useState<string>();
  const [form, setForm] = useState<PutRegulatoryProfileInput>();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const loadGeneration = useRef(0);
  const load = useCallback(async () => {
    const generation = ++loadGeneration.current;
    setLoading(true); setError(undefined);
    setPreflightError(undefined);
    const [profileResult, capabilitiesResult, preflightResult] = await Promise.allSettled([
      api.profile(), api.capabilities(), canManage && api.cutoverPreflight ? api.cutoverPreflight() : Promise.resolve(undefined),
    ]);
    if (generation !== loadGeneration.current) return;
    if (profileResult.status === "fulfilled") {
      const nextProfile = profileResult.value;
      if (session && (nextProfile.tenant_code !== session.tenant_code || nextProfile.institution_id !== session.institution_id)) {
        setProfile(undefined); setCapabilities(undefined); setPreflight(undefined); setError("Răspunsul profilului nu aparține instituției autentificate.");
      } else {
        setProfile(nextProfile);
        if (capabilitiesResult.status === "fulfilled" && capabilitiesResult.value.tenant_code === nextProfile.tenant_code && capabilitiesResult.value.institution_id === nextProfile.institution_id) {
          setCapabilities(capabilitiesResult.value);
        } else {
          setCapabilities(undefined); setError("Profilul este disponibil, dar politicile efective nu au putut fi validate. Operațiunile reglementate rămân blocate.");
        }
        if (preflightResult.status === "fulfilled" && preflightResult.value) {
          if (preflightResult.value.tenant_code === nextProfile.tenant_code && preflightResult.value.institution_id === nextProfile.institution_id) setPreflight(preflightResult.value);
          else { setPreflight(undefined); setPreflightError("Preflight-ul nu aparține instituției autentificate."); }
        } else if (canManage) {
          setPreflight(undefined); setPreflightError("Preflight-ul migrării policy v2 nu a putut fi încărcat.");
        }
      }
    } else {
      setProfile(undefined); setCapabilities(undefined); setPreflight(undefined); setError("Profilul instituțional nu a putut fi încărcat. Reîncercați.");
    }
    setLoading(false);
  }, [api, canManage, session]);
  useEffect(() => {
    void load();
    return () => { loadGeneration.current += 1; };
  }, [load]);
  const save = async () => {
    if (!form) return;
    setSaving(true); setError(undefined);
    try { await api.saveProfile(form); setForm(undefined); await Promise.all([load(), policy.refresh()]); }
    catch (reason) {
      setError((reason as { status?: number }).status === 409 ? "Profilul a fost modificat de alt utilizator. Reîncărcați și reluați editarea." : "Profilul instituțional nu a putut fi salvat.");
    } finally { setSaving(false); }
  };
  if (loading) return <div className="flex justify-center p-8">{spinner}</div>;
  if (!profile) return <div className="flex flex-col items-start gap-2"><Message.Root severity="error"><Message.Content><Message.Text>{error ?? "Profil indisponibil"}</Message.Text></Message.Content></Message.Root><Button variant="outlined" severity="secondary" onClick={() => void load()}><Refresh />Reîncarcă</Button></div>;
  return <div className="mt-4 flex flex-col gap-3">
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {capabilities?.blocked && <Message.Root severity="warn"><Message.Content><Message.Text>{capabilities.block_reason}. {capabilities.warnings.join(" ")}</Message.Text></Message.Content></Message.Root>}
    <Card.Root><Card.Body><Card.Title><div className="flex flex-wrap items-center justify-between gap-2"><span>Profil instituțional</span><div className="flex gap-2"><Button variant="outlined" severity="secondary" onClick={() => void load()}><Refresh />Reîncarcă</Button>{canManage && <Button onClick={() => setForm(asInput(profile))}><Pencil />Versiune nouă</Button>}</div></div></Card.Title><Card.Content>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Info label="Formă juridică" value={labelLegalForm(profile.school_legal_form)} />
        <Info label="Stare profil" value={profile.status} /><Info label="Versiune" value={String(profile.version)} />
        <Info label="Profil reglementar" value={profile.regulatory_profile || "—"} />
        <Info label="Autorizare / acreditare" value={profile.authorization_status} />
        <Info label="CUI" value={profile.tax_identifier || "—"} /><Info label="Fondator" value={profile.founder_name || "—"} />
        <Info label="Finanțator / ordonator" value={profile.funder_name || profile.budget_authority_name || "—"} />
        <Info label="Contabilitate" value={profile.accounting_profile || "—"} /><Info label="Achiziții" value={profile.procurement_profile || "—"} />
        <Info label="Salarizare" value={profile.payroll_profile || "—"} /><Info label="TVA" value={profile.vat_profile || "—"} />
      </div>
      <div className="mt-3 flex flex-wrap gap-2"><Tag severity={profile.public_funding ? "info" : "secondary"}>{profile.public_funding ? "Finanțare publică" : "Fără overlay finanțare publică"}</Tag><Tag severity="secondary">{profile.is_contracting_authority ? "Autoritate contractantă" : "Nu este declarată autoritate contractantă"}</Tag><Tag severity="secondary">{profile.treasury_required ? "Trezorerie obligatorie" : "Trezorerie neobligatorie"}</Tag></div>
    </Card.Content></Card.Body></Card.Root>
    <Card.Root><Card.Body><Card.Title>Politici și capabilități efective</Card.Title><Card.Content>
      <div className="mb-3 flex flex-wrap gap-2">{capabilities?.effective_policies.map((policy) => <Tag key={policy.id} severity="secondary">{`${policy.code} v${policy.version}`}</Tag>)}</div>
      {(capabilities?.capabilities.length ?? 0) === 0 ? <Message.Root severity="info"><Message.Content><Message.Text>Nu există capabilități operaționale active pentru profilul și drepturile curente.</Message.Text></Message.Content></Message.Root> : <DataTable.Root data={(capabilities?.capabilities ?? []) as unknown as Record<string, unknown>[]}><DataTable.Table><DataTable.THead><DataTable.THeadRow><DataTable.THeadCell>Capabilitate</DataTable.THeadCell><DataTable.THeadCell>Stare</DataTable.THeadCell><DataTable.THeadCell>Motiv</DataTable.THeadCell><DataTable.THeadCell>Cerințe</DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const capability = item as unknown as InstitutionCapabilities["capabilities"][number]; return <DataTable.Row index={index} key={capability.code}><DataTable.Cell>{capability.code}</DataTable.Cell><DataTable.Cell><Tag severity={capability.enabled ? "success" : "secondary"}>{capability.enabled ? "Activă" : "Indisponibilă"}</Tag></DataTable.Cell><DataTable.Cell>{capability.reason}</DataTable.Cell><DataTable.Cell>{[...capability.required_documents, ...capability.required_approvals].join(", ") || "—"}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>}
    </Card.Content></Card.Body></Card.Root>
    {canReadOfferings && <EducationOfferingsWorkspace api={api} canManage={canManageOfferings} />}
    {canManage && <Card.Root><Card.Body><Card.Title><div className="flex flex-wrap items-center justify-between gap-2"><span>Pregătire migrare policy v2</span>{preflight && <Tag severity={preflight.structurally_ready_for_dual ? "success" : "warn"}>{preflight.structurally_ready_for_dual ? "Structură reconciliată" : "Remediere necesară"}</Tag>}</div></Card.Title><Card.Content>
      {preflightError ? <Message.Root severity="warn"><Message.Content><Message.Text>{preflightError}</Message.Text></Message.Content></Message.Root> : preflight && <>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <Info label="Fază runtime" value={preflight.phase} />
          <Info label="Profile nemapate" value={`${preflight.unmapped_profiles}/${preflight.legacy_profiles}`} />
          <Info label="Assignments nemapate" value={`${preflight.unmapped_assignments}/${preflight.legacy_assignments}`} />
          <Info label="Overrides nemapate" value={`${preflight.unmapped_overrides}/${preflight.legacy_overrides}`} />
          <Info label="Evaluări nemapate" value={`${preflight.unmapped_evaluations}/${preflight.legacy_evaluations}`} />
          <Info label="Pack-uri fără proveniență" value={String(preflight.missing_pack_provenance)} />
          <Info label="Inputuri fără dată" value={String(preflight.inputs_without_effective_date)} />
          <Info label="Decizii duplicate" value={String(preflight.inputs_with_multiple_decisions)} />
          <Info label="Consumatori nereconciliați" value={String(preflight.consumer_provenance_mismatches)} />
          <Info label="Probleme blocante" value={String(preflight.open_blocking_issues)} />
        </div>
        <div className="mt-3"><Message.Root severity="info"><Message.Content><Message.Text>Acesta este un control structural read-only. Trecerea la runtime v2 necesită separat reconciliere imuabilă, shadow comparison și aprobarea cutover-ului.</Message.Text></Message.Content></Message.Root></div>
      </>}
    </Card.Content></Card.Body></Card.Root>}
    <ProfileDialog form={form} saving={saving} onChange={setForm} onClose={() => !saving && setForm(undefined)} onSave={() => void save()} />
  </div>;
}

function Info({ label, value }: { label: string; value: string }) { return <div><span>{label}</span><div><strong>{value}</strong></div></div>; }

function ProfileDialog({ form, saving, onChange, onClose, onSave }: { form?: PutRegulatoryProfileInput; saving: boolean; onChange: (value: PutRegulatoryProfileInput) => void; onClose: () => void; onSave: () => void }) {
  const set = <K extends keyof PutRegulatoryProfileInput>(key: K, value: PutRegulatoryProfileInput[K]) => form && onChange({ ...form, [key]: value });
  const text = (key: keyof PutRegulatoryProfileInput) => (event: ChangeEvent<HTMLInputElement>) => set(key, event.target.value as never);
  return <Dialog.Root open={Boolean(form)} onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Versiune nouă profil instituțional</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content>{form && <div className="grid gap-3 sm:grid-cols-2">
    <Field label="Formă juridică"><Select.Root value={form.school_legal_form} options={legalForms} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => { const value = String(event.value) as PutRegulatoryProfileInput["school_legal_form"]; onChange({ ...form, school_legal_form: value, regulatory_profile: value === "public" ? "ro.public.preuniversity" : "ro.private.preuniversity" }); }}><Select.Trigger aria-label="Formă juridică"><Select.Value placeholder="Selectați forma juridică" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
    <Field label="Stare"><Select.Root value={form.status} options={statuses} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => set("status", String(event.value) as PutRegulatoryProfileInput["status"])}><Select.Trigger aria-label="Stare profil"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
    <Field label="Profil reglementar"><InputText value={form.regulatory_profile} onChange={text("regulatory_profile")} /></Field>
    <Field label="Statut autorizare"><Select.Root value={form.authorization_status} options={authorizationStatuses} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => set("authorization_status", String(event.value) as PutRegulatoryProfileInput["authorization_status"])}><Select.Trigger aria-label="Statut autorizare"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
    <Field label="Referință acreditare"><InputText value={form.accreditation_reference ?? ""} onChange={text("accreditation_reference")} /></Field><Field label="Niveluri autorizate"><InputText value={form.authorized_levels.join(", ")} onChange={(event: ChangeEvent<HTMLInputElement>) => set("authorized_levels", event.target.value.split(",").map((value) => value.trim()).filter(Boolean))} /></Field>
    <Field label="CUI"><InputText value={form.tax_identifier ?? ""} onChange={text("tax_identifier")} /></Field><Field label="Fondator"><InputText value={form.founder_name ?? ""} onChange={text("founder_name")} /></Field>
    <Field label="Finanțator"><InputText value={form.funder_name ?? ""} onChange={text("funder_name")} /></Field><Field label="Ordonator"><InputText value={form.budget_authority_name ?? ""} onChange={text("budget_authority_name")} /></Field>
    <Field label="Profil contabil"><InputText value={form.accounting_profile ?? ""} onChange={text("accounting_profile")} /></Field><Field label="Profil achiziții"><InputText value={form.procurement_profile ?? ""} onChange={text("procurement_profile")} /></Field>
    <Field label="Profil salarizare"><InputText value={form.payroll_profile ?? ""} onChange={text("payroll_profile")} /></Field><Field label="Profil TVA"><InputText value={form.vat_profile ?? ""} onChange={text("vat_profile")} /></Field>
    <Field label="Personalitate juridică"><BooleanSelect value={form.has_legal_personality} onChange={(value) => set("has_legal_personality", value)} /></Field>
    <Field label="În vigoare de la"><InputText type="date" value={form.effective_from} onChange={text("effective_from")} /></Field><Field label="În vigoare până la"><InputText type="date" value={form.effective_to ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => set("effective_to", event.target.value || null)} /></Field>
    <Field label="Tip sursă juridică"><Select.Root value={form.source.source_kind} options={sourceKinds} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => onChange({ ...form, source: { ...form.source, source_kind: String(event.value) as typeof form.source.source_kind } })}><Select.Trigger aria-label="Tip sursă juridică"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
    <Field label="Citare act"><InputText aria-label="Citare act" value={form.source.citation} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, citation: event.target.value } })} /></Field>
    <Field label="Articol / anexă"><InputText value={form.source.article_reference} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, article_reference: event.target.value } })} /></Field><Field label="Emitent"><InputText value={form.source.issuer} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, issuer: event.target.value } })} /></Field>
    <Field label="URL oficial HTTPS"><InputText aria-label="URL oficial HTTPS" value={form.source.source_url} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, source_url: event.target.value } })} /></Field><Field label="SHA-256 document"><InputText aria-label="SHA-256 document" value={form.source.checksum_sha256} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, checksum_sha256: event.target.value.toLowerCase() } })} /></Field>
    <Field label="Publicat la"><InputText type="date" value={form.source.published_on ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, published_on: event.target.value || null } })} /></Field><Field label="Consolidat la"><InputText type="date" value={form.source.consolidated_on ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange({ ...form, source: { ...form.source, consolidated_on: event.target.value || null } })} /></Field>
    <Field label="Programe/overlay-uri"><InputText value={form.program_codes.join(", ")} onChange={(event: ChangeEvent<HTMLInputElement>) => set("program_codes", event.target.value.split(",").map((value) => value.trim()).filter(Boolean))} /></Field>
  </div>}</Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !form?.school_legal_form || !form.regulatory_profile.trim() || !form.source.citation.trim() || ((form.status === "approved" || form.status === "active") && (!form.source.source_url.startsWith("https://") || form.source.checksum_sha256.length !== 64))} onClick={onSave}>{saving ? "Se salvează…" : "Salvează versiunea"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

function BooleanSelect({ value, onChange }: { value: boolean; onChange: (value: boolean) => void }) { return <Select.Root value={String(value)} options={boolOptions} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => onChange(String(event.value) === "true")}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>; }
function Field({ label, children }: { label: string; children: ReactNode }) { return <label className="flex flex-col gap-1"><span>{label}</span>{children}</label>; }
