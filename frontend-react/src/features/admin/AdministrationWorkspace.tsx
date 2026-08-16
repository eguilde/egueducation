import { useCallback, useEffect, useMemo, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select } from "@primereact/ui/select";
import type { SelectValueChangeEvent } from "primereact/select";
import { Tag } from "@primereact/ui/tag";
import { Tabs } from "@primereact/ui/tabs";
import { Plus, Refresh } from "@primeicons/react";
import { createAdminApi } from "./api";
import type { AdminApi, AdminPermissions, AdminResource, AdminResourcePath, AdminUser, AdminWritableResourcePath, Dashboard, ModuleSetting, Page, PermissionCheck, Role, UpsertUserInput } from "./types";

const emptyUser = (): UpsertUserInput => ({
  name: "", email: "", phone: "", locale: "ro", status: "active",
  email_verified: false, phone_verified: false, preferred_otp_channel: "sms",
});
const defaultPermissions: AdminPermissions = {
  dashboard: false, usersRead: false, usersManage: false,
  rolesRead: false, rolesManage: false, modulesRead: false, modulesManage: false,
};
const Spinner = () => <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;

export interface AdministrationWorkspaceProps {
  api?: AdminApi;
  permissions?: Partial<AdminPermissions>;
  canAccess?: PermissionCheck;
  institutionName: string;
}

export function AdministrationWorkspace({ api = createAdminApi(), permissions, canAccess = () => false, institutionName }: AdministrationWorkspaceProps) {
  const access = { ...defaultPermissions, ...permissions };
  const [dashboard, setDashboard] = useState<Dashboard>();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [modules, setModules] = useState<ModuleSetting[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const [userForm, setUserForm] = useState<UpsertUserInput>();

  const load = useCallback(async () => {
    setLoading(true); setError(undefined);
    try {
      const [nextDashboard, nextUsers, nextRoles, nextModules] = await Promise.all([
        access.dashboard ? api.dashboard() : Promise.resolve(undefined),
        access.usersRead ? api.users(query) : Promise.resolve(undefined),
        access.rolesRead ? api.roles() : Promise.resolve(undefined),
        access.modulesRead ? api.modules() : Promise.resolve(undefined),
      ]);
      setDashboard(nextDashboard); setUsers(nextUsers?.items ?? []); setRoles(nextRoles?.items ?? []); setModules(nextModules?.items ?? []);
    } catch (reason) { setError(readError(reason)); }
    finally { setLoading(false); }
  }, [access.dashboard, access.modulesRead, access.rolesRead, access.usersRead, api, query]);
  useEffect(() => { void load(); }, [load]);

  const stats = useMemo(() => Object.entries(dashboard?.stats ?? {}), [dashboard]);
  const saveUser = async () => {
    if (!userForm || !userForm.name.trim() || !userForm.email.trim()) return;
    setSaving(true); setError(undefined);
    try { await api.saveUser({ ...userForm, name: userForm.name.trim(), email: userForm.email.trim() }); setUserForm(undefined); await load(); }
    catch (reason) { setError(readError(reason)); }
    finally { setSaving(false); }
  };
  const toggleModule = async (item: ModuleSetting) => {
    setSaving(true); setError(undefined);
    try { await api.saveModule({ ...item, active: !item.active }); await load(); }
    catch (reason) { setError(readError(reason)); }
    finally { setSaving(false); }
  };

  return <section aria-label="Administrare" className="flex flex-col gap-4">
    <div className="flex flex-col gap-1"><h1>Administrare</h1><p>Context activ: {institutionName}</p></div>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {!Object.values(access).some(Boolean) ? <Message.Root severity="warn"><Message.Content><Message.Text>Nu aveți drepturi de administrare pentru această instituție.</Message.Text></Message.Content></Message.Root> : <>
      {loading ? <div className="flex justify-center p-8"><Spinner /></div> : <Tabs.Root defaultValue="overview">
        <Tabs.List><Tabs.Tab value="overview">Panou</Tabs.Tab><Tabs.Tab value="users" disabled={!access.usersRead}>Utilizatori</Tabs.Tab><Tabs.Tab value="roles" disabled={!access.rolesRead}>Roluri</Tabs.Tab><Tabs.Tab value="modules" disabled={!access.modulesRead}>Module</Tabs.Tab><Tabs.Indicator /></Tabs.List>
        <Tabs.Panel value="overview"><div className="mt-4 flex flex-col gap-4">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{stats.map(([name, value]) => <Card.Root key={name}><Card.Body><Card.Content><p>{label(name)}</p><strong>{value}</strong></Card.Content></Card.Body></Card.Root>)}</div>
          {dashboard?.warnings.map((warning) => <Message.Root key={warning} severity="warn"><Message.Content><Message.Text>{warning}</Message.Text></Message.Content></Message.Root>)}
          <Card.Root><Card.Body><Card.Content><h2>Configurare tenant</h2><p>Setările din acest panou se aplică exclusiv instituției active. Identitățile comune între instituții nu sunt editate fără drept de administrator de platformă.</p><div className="flex flex-wrap gap-2">{dashboard?.admin_sections.map((section) => <Tag key={section} value={label(section)} severity="secondary" />)}</div></Card.Content></Card.Body></Card.Root>
        </div></Tabs.Panel>
        <Tabs.Panel value="users"><div className="mt-4 flex flex-col gap-4"><div className="flex flex-col gap-2 sm:flex-row"><InputText aria-label="Caută utilizatori" value={query} onChange={(event: ChangeEvent<HTMLInputElement>) => setQuery(event.target.value)} placeholder="Caută nume sau e-mail" /><Button variant="outlined" severity="secondary" onClick={() => void load()}><Refresh />Reîncarcă</Button>{access.usersManage && <Button onClick={() => setUserForm(emptyUser())}><Plus />Utilizator</Button>}</div><UserTable users={users} /></div></Tabs.Panel>
        <Tabs.Panel value="roles"><div className="mt-4"><RoleTable roles={roles} canManage={access.rolesManage} /></div></Tabs.Panel>
        <Tabs.Panel value="modules"><div className="mt-4"><ModuleTable modules={modules} canManage={access.modulesManage} saving={saving} onToggle={toggleModule} /></div></Tabs.Panel>
      </Tabs.Root>}
      <AdministrationResources api={api} canAccess={canAccess} />
    </>}
    <UserDialog open={Boolean(userForm)} form={userForm} saving={saving} onClose={() => setUserForm(undefined)} onChange={setUserForm} onSave={saveUser} />
  </section>;
}

const additionalResources: Array<{ path: AdminResourcePath; permission: string; label: string; description: string }> = [
	{ path: "roles", permission: "admin.roles.read", label: "Roluri configurabile", description: "Catalogul de roluri al instituției." },
  { path: "memberships", permission: "admin.memberships.read", label: "Apartenențe", description: "Utilizatori și apartenențe în instituția curentă." },
  { path: "org-units", permission: "admin.org_units.read", label: "Unități organizaționale", description: "Structura organizațională a instituției." },
  { path: "positions", permission: "admin.positions.read", label: "Funcții", description: "Funcții disponibile pentru înrolare și organigramă." },
  { path: "position-roles", permission: "admin.positions.read", label: "Roluri pe funcție", description: "Maparea funcțiilor către rolurile instituției." },
  { path: "permissions", permission: "admin.permissions.read", label: "Permisiuni", description: "Catalogul permisiunilor disponibile." },
  { path: "permissions/assignments", permission: "admin.permissions.read", label: "Atribuiri permisiuni", description: "Atribuiri efective pentru contextul instituției." },
  { path: "role-assignments", permission: "admin.roles.read", label: "Atribuiri roluri", description: "Roluri acordate membrilor instituției." },
  { path: "role-permissions", permission: "admin.roles.read", label: "Permisiuni roluri", description: "Permisiuni atașate rolurilor." },
  { path: "auth-methods", permission: "admin.auth_methods.read", label: "Metode autentificare", description: "Metode de autentificare configurate de server." },
  { path: "oidc/clients", permission: "admin.identity.read", label: "Clienți OIDC", description: "Clienți OIDC autorizați; se afișează numai metadatele permise." },
  { path: "workflow-definitions", permission: "admin.workflow_definitions.read", label: "Definiții flux", description: "Definițiile fluxurilor documentare." },
  { path: "nomenclatures", permission: "admin.nomenclatures.read", label: "Nomenclatoare", description: "Nomenclatoare configurate pentru instituție." },
  { path: "education-taxonomies", permission: "admin.education_taxonomies.read", label: "Taxonomii Școală", description: "Taxonomii pentru modulele educaționale." },
  { path: "dossier-requirements", permission: "admin.dossier_requirements.read", label: "Cerințe dosare", description: "Cerințe configurabile pentru dosare." },
  { path: "gdpr-settings", permission: "admin.gdpr_settings.read", label: "Setări GDPR", description: "Setări GDPR ale instituției." },
  { path: "audit", permission: "admin.audit.read", label: "Jurnal audit", description: "Evenimente de audit disponibile utilizatorului curent." },
  { path: "gdpr/config", permission: "gdpr.read", label: "Configurație GDPR", description: "Configurația GDPR disponibilă utilizatorului curent." },
  { path: "gdpr/dashboard", permission: "gdpr.read", label: "Panou GDPR", description: "Indicatori GDPR calculați de backend." },
  { path: "gdpr/exports", permission: "gdpr.exports.read", label: "Exporturi GDPR", description: "Solicitări și exporturi de date." },
  { path: "gdpr/publication-reviews", permission: "gdpr.publication.read", label: "Revizuiri publicare", description: "Revizuiri pentru publicare sigură." },
  { path: "gdpr/retention-policies", permission: "gdpr.policies.read", label: "Politici retenție", description: "Politici de retenție configurate." },
  { path: "gdpr/subject-requests", permission: "gdpr.requests.read", label: "Cereri persoane vizate", description: "Cereri GDPR ale persoanelor vizate." },
];

type ResourceField = { name: string; label: string; kind?: "boolean" | "number" };
const resourceEditors: Partial<Record<AdminWritableResourcePath, ResourceField[]>> = {
	roles: [{ name: "code", label: "Cod" }, { name: "label", label: "Etichetă" }],
	memberships: [{ name: "user_id", label: "ID utilizator" }, { name: "position_code", label: "Cod funcție" }, { name: "org_unit_code", label: "Cod unitate" }, { name: "organization_name", label: "Organizație" }, { name: "is_primary", label: "Principală", kind: "boolean" }, { name: "active", label: "Activă", kind: "boolean" }, { name: "start_date", label: "Dată început" }, { name: "end_date", label: "Dată sfârșit" }],
	"org-units": [{ name: "code", label: "Cod" }, { name: "name", label: "Nume" }, { name: "parent_code", label: "Cod părinte" }, { name: "active", label: "Activă", kind: "boolean" }, { name: "sort_order", label: "Ordine", kind: "number" }],
	positions: [{ name: "code", label: "Cod" }, { name: "name", label: "Nume" }, { name: "scope_module", label: "Modul" }, { name: "active", label: "Activă", kind: "boolean" }, { name: "sort_order", label: "Ordine", kind: "number" }],
	"position-roles": [{ name: "position_code", label: "Cod funcție" }, { name: "role_code", label: "Cod rol" }, { name: "assigned", label: "Atribuit", kind: "boolean" }],
	"permissions/assignments": [{ name: "permission_code", label: "Cod permisiune" }, { name: "position_code", label: "Cod funcție" }, { name: "assigned", label: "Atribuit", kind: "boolean" }],
	"role-assignments": [{ name: "user_id", label: "ID utilizator" }, { name: "role_code", label: "Cod rol" }, { name: "assigned", label: "Atribuit", kind: "boolean" }],
	"role-permissions": [{ name: "role_code", label: "Cod rol" }, { name: "permission_code", label: "Cod permisiune" }, { name: "assigned", label: "Atribuit", kind: "boolean" }],
	"auth-methods": [{ name: "code", label: "Cod" }, { name: "enabled", label: "Activă", kind: "boolean" }, { name: "primary_method", label: "Metodă primară", kind: "boolean" }, { name: "sort_order", label: "Ordine", kind: "number" }],
	modules: [{ name: "code", label: "Cod" }, { name: "active", label: "Activ", kind: "boolean" }],
	"oidc/clients": [{ name: "client_id", label: "Client ID" }, { name: "client_name", label: "Nume client" }, { name: "public_client", label: "Client public", kind: "boolean" }, { name: "require_pkce", label: "PKCE obligatoriu", kind: "boolean" }, { name: "active", label: "Activ", kind: "boolean" }, { name: "redirect_uris", label: "Redirect URI (separate prin virgulă)" }],
	"gdpr-settings": [{ name: "code", label: "Cod" }, { name: "value_type", label: "Tip valoare" }, { name: "value_text", label: "Valoare text" }, { name: "value_bool", label: "Valoare booleană", kind: "boolean" }, { name: "value_int", label: "Valoare numerică", kind: "number" }],
	"dossier-requirements": [{ name: "source_module", label: "Modul sursă" }, { name: "relation_type", label: "Tip relație" }, { name: "min_count", label: "Număr minim", kind: "number" }, { name: "required_for_readiness", label: "Necesar pregătire", kind: "boolean" }, { name: "required_for_submit", label: "Necesar transmitere", kind: "boolean" }, { name: "required_for_approve", label: "Necesar aprobare", kind: "boolean" }],
	"workflow-definitions": [{ name: "code", label: "Cod" }, { name: "name", label: "Nume" }, { name: "category", label: "Categorie" }, { name: "initial_step", label: "Pas inițial" }, { name: "sla_hours", label: "SLA ore", kind: "number" }, { name: "active", label: "Activ", kind: "boolean" }],
	nomenclatures: [{ name: "domain", label: "Domeniu" }, { name: "code", label: "Cod" }, { name: "label_ro", label: "Etichetă RO" }, { name: "label_en", label: "Etichetă EN" }, { name: "active", label: "Activ", kind: "boolean" }, { name: "sort_order", label: "Ordine", kind: "number" }],
	"education-taxonomies": [{ name: "domain", label: "Domeniu" }, { name: "code", label: "Cod" }, { name: "label_ro", label: "Etichetă RO" }, { name: "label_en", label: "Etichetă EN" }, { name: "active", label: "Activă", kind: "boolean" }, { name: "sort_order", label: "Ordine", kind: "number" }],
	"gdpr/retention-policies": [{ name: "domain_code", label: "Domeniu" }, { name: "record_category", label: "Categorie" }, { name: "retention_years", label: "Ani retenție", kind: "number" }, { name: "legal_basis", label: "Temei legal" }, { name: "status", label: "Stare" }, { name: "review_due_on", label: "Revizuire la" }, { name: "owner_name", label: "Responsabil" }, { name: "notes", label: "Note" }],
	"gdpr/subject-requests": [{ name: "subject_name", label: "Persoană vizată" }, { name: "request_type", label: "Tip cerere" }, { name: "status", label: "Stare" }, { name: "submitted_on", label: "Depusă la" }, { name: "due_on", label: "Termen" }, { name: "handled_by", label: "Responsabil" }, { name: "source_module", label: "Modul sursă" }, { name: "anonymization_required", label: "Anonimizare", kind: "boolean" }, { name: "notes", label: "Note" }],
	"gdpr/exports": [{ name: "request_id", label: "ID cerere" }, { name: "subject_name", label: "Persoană vizată" }, { name: "source_module", label: "Modul sursă" }, { name: "status", label: "Stare" }, { name: "export_format", label: "Format" }, { name: "approved_by", label: "Aprobat de" }, { name: "approved_on", label: "Aprobat la" }, { name: "generated_on", label: "Generat la" }, { name: "package_summary", label: "Rezumat pachet" }, { name: "notes", label: "Note" }],
	"gdpr/publication-reviews": [{ name: "source_module", label: "Modul sursă" }, { name: "source_record_id", label: "ID înregistrare" }, { name: "source_label", label: "Etichetă" }, { name: "anonymization_status", label: "Stare anonimizare" }, { name: "publication_status", label: "Stare publicare" }, { name: "reviewed_by", label: "Revizuit de" }, { name: "reviewed_on", label: "Revizuit la" }, { name: "legal_basis", label: "Temei legal" }, { name: "notes", label: "Note" }],
};

const resourceWritePermission: Partial<Record<AdminWritableResourcePath, string>> = {
	roles: "admin.roles.manage", memberships: "admin.memberships.manage", "org-units": "admin.org_units.manage", positions: "admin.positions.manage", "position-roles": "admin.positions.manage", "permissions/assignments": "admin.permissions.manage", "role-assignments": "admin.roles.manage", "role-permissions": "admin.roles.manage", "auth-methods": "admin.auth_methods.manage", modules: "admin.modules.manage", "oidc/clients": "admin.identity.manage", "gdpr-settings": "admin.gdpr_settings.manage", "dossier-requirements": "admin.dossier_requirements.manage", "workflow-definitions": "admin.workflow_definitions.manage", nomenclatures: "admin.nomenclatures.manage", "education-taxonomies": "admin.education_taxonomies.manage", "gdpr/retention-policies": "gdpr.policies.manage", "gdpr/subject-requests": "gdpr.requests.manage", "gdpr/exports": "gdpr.exports.manage", "gdpr/publication-reviews": "gdpr.publication.manage",
};

function resourceTitle(resource: AdminResource) {
  for (const field of ["name", "label", "code", "title", "id"]) {
    const value = resource[field];
    if (typeof value === "string" || typeof value === "number") return String(value);
  }
  return "Înregistrare";
}

function AdministrationResources({ api, canAccess }: { api: AdminApi; canAccess: PermissionCheck }) {
  const resources = useMemo(() => additionalResources.filter((resource) => canAccess(resource.permission)), [canAccess]);
  const [selected, setSelected] = useState<(typeof additionalResources)[number]>();
  const [page, setPage] = useState<Page<AdminResource>>({ items: [], total: 0, page: 1, pageSize: 50 });
  const [loading, setLoading] = useState(false);
	const [saving, setSaving] = useState(false);
	const [editorOpen, setEditorOpen] = useState(false);
	const [editorValues, setEditorValues] = useState<Record<string, string>>({});
  const [error, setError] = useState<string>();
  const load = useCallback(async () => {
    setLoading(true); setError(undefined);
    if (!selected) return;
    try { setPage(await api.resource(selected.path)); }
    catch (reason) { setError(readError(reason)); }
    finally { setLoading(false); }
  }, [api, selected]);
  useEffect(() => { void load(); }, [load]);
  useEffect(() => { setSelected((current) => resources.find((resource) => resource.path === current?.path) ?? resources[0]); }, [resources]);
  if (resources.length === 0 || !selected) return null;
	const writable = selected.path in resourceEditors ? selected.path as AdminWritableResourcePath : undefined;
	const canWrite = Boolean(writable && resourceWritePermission[writable] && canAccess(resourceWritePermission[writable]!));
	const openEditor = () => {
		if (!writable) return;
		setEditorValues(Object.fromEntries((resourceEditors[writable] ?? []).map((field) => [field.name, field.kind === "boolean" ? "false" : ""])));
		setEditorOpen(true);
	};
	const save = async () => {
		if (!writable) return;
		setSaving(true); setError(undefined);
		try {
			const fields = resourceEditors[writable] ?? [];
			const input: Record<string, unknown> = {};
			for (const field of fields) {
				const value = editorValues[field.name] ?? "";
				input[field.name] = field.kind === "boolean" ? value === "true" : field.kind === "number" ? Number(value || 0) : field.name === "redirect_uris" ? value.split(",").map((uri) => uri.trim()).filter(Boolean) : value;
			}
			await api.saveResource(writable, input); setEditorOpen(false); await load();
		} catch (reason) { setError(readError(reason)); }
		finally { setSaving(false); }
	};
  return <><Card.Root><Card.Body><Card.Title>Configurări și conformitate</Card.Title><Card.Content><div className="flex flex-col gap-4">
    <p>Secțiunile disponibile citesc datele direct din API, numai când aveți permisiunea exactă pentru endpointul respectiv.</p>
    <div className="flex flex-wrap gap-2" aria-label="Secțiuni de administrare">{resources.map((resource) => <Button key={resource.path} size="small" variant={selected.path === resource.path ? undefined : "outlined"} severity={selected.path === resource.path ? undefined : "secondary"} onClick={() => setSelected(resource)}>{resource.label}</Button>)}</div>
    <div className="flex flex-wrap items-center justify-between gap-2"><div><h2>{selected.label}</h2><p>{selected.description}</p></div>{canWrite && <Button onClick={openEditor}><Plus />Adaugă sau actualizează</Button>}</div>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {loading ? <div className="flex justify-center p-6"><Spinner /></div> : page.items.length === 0 ? <Message.Root severity="info"><Message.Content><Message.Text>Nu există date disponibile sau nu aveți dreptul de a le consulta.</Message.Text></Message.Content></Message.Root> : <DataTable.Root data={page.items as Record<string, unknown>[]}><DataTable.Table><DataTable.THead><DataTable.THeadRow><DataTable.THeadCell>Înregistrare</DataTable.THeadCell><DataTable.THeadCell>Identificator</DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const resource = item as AdminResource; return <DataTable.Row key={String(resource.id ?? resource.code ?? index)} index={index}><DataTable.Cell>{resourceTitle(resource)}</DataTable.Cell><DataTable.Cell>{String(resource.id ?? resource.code ?? "—")}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>}
  </div></Card.Content></Card.Body></Card.Root><ResourceEditor open={editorOpen} resource={writable} values={editorValues} saving={saving} onChange={(name, value) => setEditorValues((current) => ({ ...current, [name]: value }))} onClose={() => setEditorOpen(false)} onSave={save} /></>;
}

function ResourceEditor({ open, resource, values, saving, onChange, onClose, onSave }: { open: boolean; resource?: AdminWritableResourcePath; values: Record<string, string>; saving: boolean; onChange: (name: string, value: string) => void; onClose: () => void; onSave: () => void }) {
	const fields = resource ? resourceEditors[resource] ?? [] : [];
	return <Dialog.Root open={open} onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Configurare instituție</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">{fields.map((field) => field.kind === "boolean" ? <Select.Root key={field.name} value={values[field.name]} options={[{ label: `${field.label}: nu`, value: "false" }, { label: `${field.label}: da`, value: "true" }]} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => onChange(field.name, String(event.value))}><Select.Trigger><Select.Value /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root> : <InputText key={field.name} aria-label={field.label} type={field.kind === "number" ? "number" : undefined} value={values[field.name] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => onChange(field.name, event.target.value)} />)}</div></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={onClose}>Renunță</Button><Button disabled={saving} onClick={onSave}>{saving ? "Se salvează…" : "Salvează"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

function UserTable({ users }: { users: AdminUser[] }) { return <Card.Root><Card.Body><Card.Content>{users.length === 0 ? <Message.Root severity="info"><Message.Content><Message.Text>Niciun utilizator nu este disponibil pentru instituția activă.</Message.Text></Message.Content></Message.Root> : <DataTable.Root data={users as unknown as Record<string, unknown>[]} dataKey="id"><DataTable.Table><DataTable.THead><DataTable.THeadRow><DataTable.THeadCell>Nume</DataTable.THeadCell><DataTable.THeadCell>E-mail</DataTable.THeadCell><DataTable.THeadCell>Funcție</DataTable.THeadCell><DataTable.THeadCell>Stare</DataTable.THeadCell><DataTable.THeadCell>Verificare</DataTable.THeadCell></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const user = item as unknown as AdminUser; return <DataTable.Row key={user.id} index={index}><DataTable.Cell>{user.name}</DataTable.Cell><DataTable.Cell>{user.email}</DataTable.Cell><DataTable.Cell>{user.position || "—"}</DataTable.Cell><DataTable.Cell><Tag value={user.status} severity={user.status === "active" ? "success" : "secondary"} /></DataTable.Cell><DataTable.Cell>{user.email_verified ? "E-mail verificat" : "E-mail neverificat"}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>}</Card.Content></Card.Body></Card.Root>; }
function RoleTable({ roles, canManage }: { roles: Role[]; canManage: boolean }) { return <Card.Root><Card.Body><Card.Content><div className="flex items-center justify-between"><h2>Roluri</h2>{canManage && <Message.Root severity="info"><Message.Content><Message.Text>Crearea și actualizarea rolurilor sunt disponibile în secțiunea „Roluri configurabile”, prin formularul validat după DTO-ul serverului.</Message.Text></Message.Content></Message.Root>}</div>{roles.length === 0 ? <p>Nu există roluri accesibile.</p> : <div className="flex flex-wrap gap-2">{roles.map((role) => <Tag key={role.code} value={`${role.label} (${role.code})`} severity="secondary" />)}</div>}</Card.Content></Card.Body></Card.Root>; }
function ModuleTable({ modules, canManage, saving, onToggle }: { modules: ModuleSetting[]; canManage: boolean; saving: boolean; onToggle: (value: ModuleSetting) => void }) { return <Card.Root><Card.Body><Card.Content>{modules.length === 0 ? <Message.Root severity="info"><Message.Content><Message.Text>Nu există module configurabile pentru instituția activă.</Message.Text></Message.Content></Message.Root> : <div className="flex flex-col gap-3">{modules.map((module) => <div className="flex flex-wrap items-center justify-between gap-2" key={module.code}><span>{label(module.code)}</span><div className="flex items-center gap-2"><Tag value={module.active ? "Activ" : "Inactiv"} severity={module.active ? "success" : "secondary"} />{canManage && <Button size="small" variant="outlined" severity="secondary" disabled={saving} onClick={() => void onToggle(module)}>{module.active ? "Dezactivează" : "Activează"}</Button>}</div></div>)}</div>}</Card.Content></Card.Body></Card.Root>; }
function UserDialog({ open, form, saving, onClose, onChange, onSave }: { open: boolean; form?: UpsertUserInput; saving: boolean; onClose: () => void; onChange: (value: UpsertUserInput) => void; onSave: () => void }) { const set = <K extends keyof UpsertUserInput>(key: K, value: UpsertUserInput[K]) => form && onChange({ ...form, [key]: value }); return <Dialog.Root open={open} onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Utilizator nou</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><InputText aria-label="Nume" value={form?.name ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("name", e.target.value)} /><InputText aria-label="E-mail" type="email" value={form?.email ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("email", e.target.value)} /><InputText aria-label="Telefon" value={form?.phone ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("phone", e.target.value)} /><Select.Root value={form?.locale} options={[{ label: "Română", value: "ro" }, { label: "English", value: "en" }]} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => set("locale", event.value as "ro" | "en")}><Select.Trigger><Select.Value /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={onClose}>Renunță</Button><Button disabled={saving || !form?.name.trim() || !form?.email.trim()} onClick={onSave}>{saving ? "Se salvează…" : "Salvează"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>; }
function label(value: string) { return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase()); }
function readError(reason: unknown) { if (!(reason instanceof Error)) return "Operația de administrare nu a reușit."; if (reason.message === "shared_identity_platform_admin_required") return "Identitatea este folosită și în alt tenant; este necesar un administrator de platformă."; return "Operația de administrare nu a reușit. Încercați din nou."; }
