import { useCallback, useEffect, useMemo, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Tag } from "@primereact/ui/tag";
import { Textarea } from "@primereact/ui/textarea";
import type { EducationDelegationGrant } from "../../auth/AuthProvider";
import type { components } from "../../api/generated";
import type { EducationApi, EducationRecordsDomain } from "./types";

type DelegatedResourceType = Exclude<EducationDelegationGrant["resource_type"], "institution">;
type DelegatedResourceApi = Pick<EducationApi, "governanceMeetingDetail" | "recordDetail" | "saveGovernanceMeeting" | "updateRecord">;

export interface DelegatedEducationResourcesWorkspaceProps {
  /** Snapshot received from the authenticated, server-authorized grant endpoint. */
  grants: readonly EducationDelegationGrant[];
  api: DelegatedResourceApi;
  /** The route obtains this from /api/me; server-side RLS remains authoritative. */
  institutionID?: string;
  onRefresh?: () => Promise<void>;
}

type ScopedResource = {
  key: string;
  resourceType: DelegatedResourceType;
  resourceId: string;
  permissions: string[];
};

type LoadedResource = ScopedResource & {
  detail: Record<string, unknown>;
  label: string;
  status: string;
};

const domainForResource: Record<Exclude<DelegatedResourceType, "meeting">, EducationRecordsDomain> = {
  portfolio: "portfolios",
  decision: "decisions",
  regulation: "regulations",
  personnel: "personnel",
};

const resourceLabels: Record<DelegatedResourceType, string> = {
  portfolio: "Portofoliu",
  meeting: "Ședință",
  decision: "Decizie",
  regulation: "Reglementare",
  personnel: "Dosar de personal",
};

function toRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" ? value as Record<string, unknown> : {};
}

function firstText(record: Record<string, unknown>, keys: readonly string[]): string | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return undefined;
}

function resourceLabel(resource: ScopedResource, detail: Record<string, unknown>): string {
  return firstText(detail, [
    "title", "name", "record_code", "meeting_code", "decision_code", "regulation_code", "personnel_code", "full_name",
  ]) ?? `${resourceLabels[resource.resourceType]} ${resource.resourceId}`;
}

function resourceStatus(detail: Record<string, unknown>): string {
  return firstText(detail, ["status", "lifecycle_status", "publication_status", "state"]) ?? "Disponibil";
}

/**
 * Produces a closed resource set from a server snapshot.  It intentionally
 * excludes institution grants: those grant a broad capability and belong in
 * the normal school workspaces, never in this exact-resource inbox.
 */
export function delegatedResources(grants: readonly EducationDelegationGrant[]): ScopedResource[] {
  const grouped = new Map<string, ScopedResource>();
  for (const grant of grants) {
    if (grant.resource_type === "institution") continue;
    const key = `${grant.resource_type}:${grant.resource_id}`;
    const existing = grouped.get(key);
    if (existing) {
      if (!existing.permissions.includes(grant.permission_code)) existing.permissions.push(grant.permission_code);
      continue;
    }
    grouped.set(key, {
      key,
      resourceType: grant.resource_type,
      resourceId: grant.resource_id,
      permissions: [grant.permission_code],
    });
  }
  return [...grouped.values()].sort((left, right) => left.key.localeCompare(right.key));
}

async function loadExactResource(api: DelegatedResourceApi, resource: ScopedResource, institutionID?: string): Promise<LoadedResource> {
  const detail = resource.resourceType === "meeting"
    ? await api.governanceMeetingDetail(resource.resourceId)
    : await api.recordDetail(domainForResource[resource.resourceType], resource.resourceId);
  const record = toRecord(detail);
  if (institutionID && record.institution_id !== institutionID) throw new Error("delegated_resource_institution_mismatch");
  return { ...resource, detail: record, label: resourceLabel(resource, record), status: resourceStatus(record) };
}

type EditableField = { key: string; label: string; multiline?: boolean; type?: "date" | "text" };
type EditValues = Record<string, string>;

const editableFields: Record<DelegatedResourceType, readonly EditableField[]> = {
  decision: [{ key: "title", label: "Titlu" }, { key: "summary", label: "Rezumat", multiline: true }, { key: "legal_basis", label: "Temei legal", multiline: true }, { key: "status", label: "Stare" }, { key: "publication_status", label: "Stare publicare" }],
  regulation: [{ key: "title", label: "Titlu" }, { key: "summary", label: "Rezumat", multiline: true }, { key: "owner_name", label: "Responsabil" }, { key: "approval_status", label: "Stare aprobare" }, { key: "status", label: "Stare" }, { key: "review_due_on", label: "Revizuire până la", type: "date" }],
  personnel: [{ key: "full_name", label: "Nume" }, { key: "role_title", label: "Funcție" }, { key: "assigned_unit", label: "Compartiment" }, { key: "email", label: "E-mail" }, { key: "phone", label: "Telefon" }, { key: "employment_type", label: "Tip încadrare" }, { key: "evaluation_status", label: "Stare evaluare" }, { key: "mobility_stage", label: "Etapă mobilitate" }, { key: "status", label: "Stare" }, { key: "notes", label: "Note", multiline: true }],
  portfolio: [{ key: "owner_name", label: "Titular" }, { key: "owner_role", label: "Rol titular" }, { key: "custodian", label: "Custode" }, { key: "status", label: "Stare" }, { key: "transfer_status", label: "Stare transfer" }, { key: "notes", label: "Note", multiline: true }],
  meeting: [{ key: "title", label: "Titlu" }, { key: "summary", label: "Rezumat", multiline: true }, { key: "location", label: "Loc" }, { key: "meeting_date", label: "Data ședinței", type: "date" }, { key: "meeting_type", label: "Tip ședință" }, { key: "organism", label: "Organism" }, { key: "status", label: "Stare" }],
};

function text(record: Record<string, unknown>, field: string): string | undefined {
  const value = record[field];
  return typeof value === "string" ? value : undefined;
}

function required(record: Record<string, unknown>, field: string): string {
  const value = text(record, field);
  if (!value?.trim()) throw new Error(`delegated_resource_missing_${field}`);
  return value;
}

function optional(record: Record<string, unknown>, field: string): string | undefined {
  const value = text(record, field);
  return value?.trim() ? value : undefined;
}

function booleanValue(record: Record<string, unknown>, field: string): boolean | undefined {
  const value = record[field];
  return typeof value === "boolean" ? value : undefined;
}

function numberValue(record: Record<string, unknown>, field: string): number | undefined {
  const value = record[field];
  return typeof value === "number" ? value : undefined;
}

function valuesFor(resource: LoadedResource): EditValues {
  return Object.fromEntries(editableFields[resource.resourceType].map((field) => [field.key, text(resource.detail, field.key) ?? ""]));
}

function patched(detail: Record<string, unknown>, values: EditValues, field: string): string {
  const value = values[field]?.trim();
  if (!value) throw new Error(`delegated_resource_missing_${field}`);
  return value;
}

function patchedEnum<const T extends readonly string[]>(detail: Record<string, unknown>, values: EditValues, field: string, allowed: T): T[number] {
  const value = patched(detail, values, field);
  if (!(allowed as readonly string[]).includes(value)) throw new Error(`delegated_resource_invalid_${field}`);
  return value as T[number];
}

/** Maps only an already-authorized exact detail into its generated PATCH body. */
function decisionUpdateBody(detail: Record<string, unknown>, values: EditValues): components["schemas"]["CreateGovernanceDecisionRequest"] { return { decision_date: required(detail, "decision_date"), legal_basis: values.legal_basis.trim() || undefined, organism: required(detail, "organism"), publication_status: patched(detail, values, "publication_status"), school_year: required(detail, "school_year"), signed_by: optional(detail, "signed_by"), status: patched(detail, values, "status"), summary: values.summary.trim() || undefined, title: patched(detail, values, "title") }; }
function regulationUpdateBody(detail: Record<string, unknown>, values: EditValues): components["schemas"]["CreateRegulationRecordRequest"] { return { approval_status: patched(detail, values, "approval_status"), approved_on: optional(detail, "approved_on"), owner_name: values.owner_name.trim() || undefined, regulation_type: required(detail, "regulation_type"), review_due_on: patched(detail, values, "review_due_on"), school_year: required(detail, "school_year"), status: patched(detail, values, "status"), summary: values.summary.trim() || undefined, title: patched(detail, values, "title") }; }
function personnelUpdateBody(detail: Record<string, unknown>, values: EditValues): components["schemas"]["CreatePersonnelRecordRequest"] { return { app_user_id: optional(detail, "app_user_id"), assigned_unit: values.assigned_unit.trim() || undefined, email: values.email.trim() || undefined, employment_type: patchedEnum(detail, values, "employment_type", ["titular", "suplinitor", "plata_cu_ora", "auxiliar"] as const), evaluation_status: patchedEnum(detail, values, "evaluation_status", ["draft", "in_review", "finalized"] as const), full_name: patched(detail, values, "full_name"), has_portfolio: booleanValue(detail, "has_portfolio"), mobility_stage: patchedEnum(detail, values, "mobility_stage", ["none", "transfer", "detasare", "restrangere"] as const), notes: values.notes.trim() || undefined, phone: values.phone.trim() || undefined, role_title: patched(detail, values, "role_title"), school_year: required(detail, "school_year"), status: patchedEnum(detail, values, "status", ["active", "on_leave", "vacant", "inactive"] as const) }; }
function portfolioUpdateBody(detail: Record<string, unknown>, values: EditValues): components["schemas"]["UpdatePortfolioRecordRequest"] { return { authenticity_declared: booleanValue(detail, "authenticity_declared"), consent_captured: booleanValue(detail, "consent_captured"), custodian: values.custodian.trim() || undefined, last_updated_on: required(detail, "last_updated_on"), notes: values.notes.trim() || undefined, owner_name: patched(detail, values, "owner_name"), owner_personnel_id: optional(detail, "owner_personnel_id"), owner_role: patched(detail, values, "owner_role"), owner_user_id: optional(detail, "owner_user_id"), school_year: required(detail, "school_year"), section_count: numberValue(detail, "section_count"), status: patched(detail, values, "status"), transfer_status: patched(detail, values, "transfer_status") }; }

async function saveRootUpdate(api: DelegatedResourceApi, resource: LoadedResource, values: EditValues): Promise<unknown> {
  switch (resource.resourceType) {
    case "decision": return api.updateRecord("decisions", resource.resourceId, decisionUpdateBody(resource.detail, values));
    case "regulation": return api.updateRecord("regulations", resource.resourceId, regulationUpdateBody(resource.detail, values));
    case "personnel": return api.updateRecord("personnel", resource.resourceId, personnelUpdateBody(resource.detail, values));
    case "portfolio": return api.updateRecord("portfolios", resource.resourceId, portfolioUpdateBody(resource.detail, values));
    case "meeting": return api.saveGovernanceMeeting(meetingUpdateBody(resource, values), resource.resourceId);
  }
}

function meetingUpdateBody(resource: LoadedResource, values: EditValues): components["schemas"]["CreateGovernanceMeetingRequest"] {
  const detail = resource.detail;
  return { chairperson: optional(detail, "chairperson"), chairperson_user_id: required(detail, "chairperson_user_id"), location: values.location.trim() || undefined, meeting_date: patched(detail, values, "meeting_date"), meeting_type: patched(detail, values, "meeting_type"), organism: patched(detail, values, "organism"), participants_count: numberValue(detail, "participants_count"), quorum_required: numberValue(detail, "quorum_required"), school_year: required(detail, "school_year"), secretary_name: optional(detail, "secretary_name"), secretary_user_id: required(detail, "secretary_user_id"), status: patched(detail, values, "status"), summary: values.summary.trim() || undefined, title: patched(detail, values, "title") };
}

export function canEditDelegatedResource(resource: LoadedResource): boolean {
  if (!resource.permissions.some((permission) => permission.endsWith(".manage"))) return false;
  try {
    if (resource.resourceType === "meeting") meetingUpdateBody(resource, valuesFor(resource));
    else {
      switch (resource.resourceType) {
        case "decision": decisionUpdateBody(resource.detail, valuesFor(resource)); break;
        case "regulation": regulationUpdateBody(resource.detail, valuesFor(resource)); break;
        case "personnel": personnelUpdateBody(resource.detail, valuesFor(resource)); break;
        case "portfolio": portfolioUpdateBody(resource.detail, valuesFor(resource)); break;
      }
    }
    return true;
  } catch { return false; }
}

function formatPermission(permission: string): string {
  return permission.replace(/^education\./, "").replaceAll(".", " · ");
}

function EditDialog({ resource, api, onClose, onSaved }: { resource: LoadedResource; api: DelegatedResourceApi; onClose: () => void; onSaved: (saved: LoadedResource) => void }) {
  const [values, setValues] = useState(() => valuesFor(resource));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const save = async () => {
    setSaving(true); setError(undefined);
    try {
      const updated = await saveRootUpdate(api, resource, values);
      const detail = toRecord(updated);
      if (detail.institution_id !== resource.detail.institution_id) throw new Error("delegated_resource_institution_mismatch");
      onSaved({ ...resource, detail, label: resourceLabel(resource, detail), status: resourceStatus(detail) });
      onClose();
    } catch { setError("Actualizarea nu a fost acceptată. Dreptul, tenantul și instituția sunt validate de server."); }
    finally { setSaving(false); }
  };
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}>
    <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,42rem)]">
      <Dialog.Header><Dialog.Title>Editează resursa delegată</Dialog.Title><Dialog.Close aria-label="Închide editorul" /></Dialog.Header>
      <Dialog.Content><div className="flex flex-col gap-3">
        <Message.Root severity="info"><Message.Content><Message.Text>Editarea se aplică exclusiv resursei identificate mai jos; interfața nu deschide colecții sau căutări globale.</Message.Text></Message.Content></Message.Root>
        {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
        {editableFields[resource.resourceType].map((field) => <label key={field.key} className="flex flex-col gap-1"><span>{field.label} *</span>{field.multiline ? <Textarea value={values[field.key]} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setValues((current) => ({ ...current, [field.key]: event.target.value }))} /> : <InputText type={field.type ?? "text"} value={values[field.key]} onChange={(event: ChangeEvent<HTMLInputElement>) => setValues((current) => ({ ...current, [field.key]: event.target.value }))} />}</label>)}
      </div></Dialog.Content>
      <Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving} onClick={() => void save()}>{saving ? "Se salvează…" : "Salvează"}</Button></div></Dialog.Footer>
    </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
  </Dialog.Root>;
}

function DetailDialog({ resource, api, onClose, onSaved }: { resource: LoadedResource; api: DelegatedResourceApi; onClose: () => void; onSaved: (saved: LoadedResource) => void }) {
  const fields = [
    ["Identificator", resource.resourceId],
    ["Tip", resourceLabels[resource.resourceType]],
    ["Stare", resource.status],
    ["Titlu", resource.label],
  ] as const;
  const canManage = resource.permissions.some((permission) => permission.endsWith(".manage"));
  const canEdit = canEditDelegatedResource(resource);
  const [editing, setEditing] = useState(false);
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}>
    <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,38rem)]">
      <Dialog.Header><Dialog.Title>Detalii resursă delegată</Dialog.Title><Dialog.Close aria-label="Închide detaliile" /></Dialog.Header>
      <Dialog.Content><div className="flex flex-col gap-3">
        <div className="grid gap-2 sm:grid-cols-2">{fields.map(([label, value]) => <div key={label}><strong>{label}:</strong> {value}</div>)}</div>
        <div className="flex flex-wrap gap-2">{resource.permissions.map((permission) => <Tag key={permission} value={formatPermission(permission)} severity={permission.endsWith(".manage") ? "warn" : "info"} />)}</div>
        {canManage && !canEdit && <Message.Root severity="warn"><Message.Content><Message.Text>Există un drept de administrare delegat, dar detaliul nu conține toate câmpurile necesare contractului de actualizare. Pentru a evita completarea sau accesul largit, editarea este blocată până când serverul poate furniza un contract exact.</Message.Text></Message.Content></Message.Root>}
      </div></Dialog.Content>
      <Dialog.Footer><div className="flex flex-wrap justify-end gap-2">{canEdit && <Button onClick={() => setEditing(true)}><i className="pi pi-pencil" aria-hidden="true" />Editează</Button>}<Button variant="outlined" severity="secondary" onClick={onClose}>Închide</Button></div></Dialog.Footer>
    </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
    {editing && <EditDialog resource={resource} api={api} onClose={() => setEditing(false)} onSaved={onSaved} />}
  </Dialog.Root>;
}

export function DelegatedEducationResourcesWorkspace({ grants, api, institutionID, onRefresh }: DelegatedEducationResourcesWorkspaceProps) {
  const scoped = useMemo(() => delegatedResources(grants), [grants]);
  const scopedKey = useMemo(() => scoped.map((resource) => `${resource.key}:${resource.permissions.join(",")}`).join("|"), [scoped]);
  const [items, setItems] = useState<LoadedResource[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string>();
  const [selected, setSelected] = useState<LoadedResource>();

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    if (!scoped.length) { setItems([]); setLoading(false); return; }
    const settled = await Promise.allSettled(scoped.map((resource) => loadExactResource(api, resource, institutionID)));
    // A revoked / forbidden / missing item is fail-closed: it is not rendered,
    // and no list endpoint is used to infer or broaden its visibility.
    const visible = settled.flatMap((result) => result.status === "fulfilled" ? [result.value] : []);
    setItems(visible);
    if (!visible.length) setError("Resursele delegate nu mai sunt disponibile sau accesul a fost revocat.");
    setLoading(false);
  }, [api, scoped, scopedKey]);

  useEffect(() => { void load(); }, [load]);

  const refresh = async () => {
    setRefreshing(true);
    try { await onRefresh?.(); await load(); } finally { setRefreshing(false); }
  };

  return <div className="mx-auto flex max-w-6xl flex-col gap-4">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div><h1 className="m-0 text-xl">Resurse delegate</h1><p className="m-0 mt-1">Sunt afișate numai resursele pentru care există o delegare activă în instituția curentă.</p></div>
      <Button severity="secondary" variant="outlined" disabled={loading || refreshing} onClick={() => void refresh()}><i className="pi pi-refresh" aria-hidden="true" />Actualizează</Button>
    </div>
    {error && <Message.Root severity="warn"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {loading ? <div className="flex min-h-40 items-center justify-center" role="status"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div> : items.length ? <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {items.map((resource) => <Card.Root key={resource.key}><Card.Body><Card.Title>{resource.label}</Card.Title><Card.Subtitle>{resourceLabels[resource.resourceType]} · {resource.resourceId}</Card.Subtitle><Card.Content><div className="flex flex-col gap-3"><div className="flex flex-wrap gap-2"><Tag value={resource.status} severity="secondary" />{resource.permissions.map((permission) => <Tag key={permission} value={formatPermission(permission)} severity={permission.endsWith(".manage") ? "warn" : "info"} />)}</div><div><Button size="small" variant="outlined" onClick={() => setSelected(resource)}><i className="pi pi-eye" aria-hidden="true" />Detalii</Button></div></div></Card.Content></Card.Body></Card.Root>)}
    </div> : <Message.Root severity="info"><Message.Content><Message.Text>Nu există resurse delegate active pentru utilizatorul curent.</Message.Text></Message.Content></Message.Root>}
    {selected && <DetailDialog resource={selected} api={api} onClose={() => setSelected(undefined)} onSaved={(saved) => { setItems((current) => current.map((item) => item.key === saved.key ? saved : item)); setSelected(saved); }} />}
  </div>;
}
