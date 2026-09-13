import { useCallback, useEffect, useState, type ChangeEvent, type ReactNode } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputNumber, type InputNumberRootValueChangeEvent } from "@primereact/ui/inputnumber";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import { Tabs } from "@primereact/ui/tabs";
import { Tag } from "@primereact/ui/tag";
import type {
  CreateEducationOfferingInput,
  CreateOfferingAuthorizationInput,
  CreateSchoolLocationInput,
  EducationOffering,
  EducationOfferingPage,
  InstitutionCatalogQuery,
  InstitutionPolicyApi,
  OfferingAuthorization,
  OfferingAuthorizationPage,
  SchoolLocation,
  SchoolLocationPage,
  UpdateEducationOfferingInput,
  UpdateSchoolLocationInput,
} from "./api";

const emptyPage = <T,>(): { items: T[]; total: number; page: number; pageSize: number } => ({ items: [], total: 0, page: 1, pageSize: 10 });
const query = (): InstitutionCatalogQuery => ({ page: 1, pageSize: 10, direction: "asc", filters: {} });
const today = () => new Date().toISOString().slice(0, 10);
const key = () => globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random()}`;
const spinner = <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;
const booleanOptions = [{ label: "Da", value: "true" }, { label: "Nu", value: "false" }];
const authorizationStatuses = ["provisional", "accredited", "suspended", "withdrawn", "expired"].map((value) => ({ label: value, value }));
const capacityUnits = [{ label: "Elevi", value: "students" }, { label: "Formațiuni de studiu", value: "study_groups" }];
const studyShifts = [{ label: "Zi", value: "day" }, { label: "După-amiază", value: "afternoon" }, { label: "Seară", value: "evening" }];
const sourceKinds = ["authorization", "accreditation", "law", "government_decision", "ministerial_order", "founder_decision", "contract", "other"].map((value) => ({ label: value, value }));

type CatalogPage<T> = { items: T[]; total: number; page: number; pageSize: number };
export async function loadAllCatalogOptions<T>(loader: (query: InstitutionCatalogQuery) => Promise<CatalogPage<T>>): Promise<T[]> {
  const items: T[] = [];
  for (let page = 1; page <= 1000; page += 1) {
    const result = await loader({ page, pageSize: 50, direction: "asc", filters: {} });
    items.push(...result.items);
    if (items.length >= result.total) return items;
    if (!result.items.length) throw new Error("institution_catalog_option_pagination_stalled");
  }
  throw new Error("institution_catalog_option_pagination_limit");
}

export function EducationOfferingsWorkspace({ api, canManage }: { api: InstitutionPolicyApi; canManage: boolean }) {
  const supported = api.locations && api.offerings && api.authorizations && api.createLocation && api.createOffering && api.createAuthorization;
  const [locations, setLocations] = useState<SchoolLocationPage>(emptyPage());
  const [offerings, setOfferings] = useState<EducationOfferingPage>(emptyPage());
  const [authorizations, setAuthorizations] = useState<OfferingAuthorizationPage>(emptyPage());
  const [locationOptions, setLocationOptions] = useState<SchoolLocation[]>([]);
  const [offeringOptions, setOfferingOptions] = useState<EducationOffering[]>([]);
  const [locationQuery, setLocationQuery] = useState(query);
  const [offeringQuery, setOfferingQuery] = useState(query);
  const [authorizationQuery, setAuthorizationQuery] = useState(query);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const [locationForm, setLocationForm] = useState<CreateSchoolLocationInput>();
  const [offeringForm, setOfferingForm] = useState<CreateEducationOfferingInput>();
  const [locationEdit, setLocationEdit] = useState<{ item: SchoolLocation; value: UpdateSchoolLocationInput }>();
  const [offeringEdit, setOfferingEdit] = useState<{ item: EducationOffering; value: UpdateEducationOfferingInput }>();
  const [authorizationForm, setAuthorizationForm] = useState<CreateOfferingAuthorizationInput>();

  const load = useCallback(async () => {
    if (!supported) return;
    setLoading(true); setError(undefined);
    const results = await Promise.allSettled([
      api.locations!(locationQuery),
      api.offerings!(offeringQuery),
      api.authorizations!(authorizationQuery),
      canManage ? loadAllCatalogOptions(api.locations!) : Promise.resolve([] as SchoolLocation[]),
      canManage ? loadAllCatalogOptions(api.offerings!) : Promise.resolve([] as EducationOffering[]),
    ]);
    if (results[0].status === "fulfilled") setLocations(results[0].value); else setError("Locațiile nu au putut fi încărcate.");
    if (results[1].status === "fulfilled") setOfferings(results[1].value); else setError("Ofertele educaționale nu au putut fi încărcate.");
    if (results[2].status === "fulfilled") setAuthorizations(results[2].value); else setError("Deciziile de autorizare nu au putut fi încărcate.");
    if (results[3].status === "fulfilled") setLocationOptions(results[3].value); else setError("Lista completă de locații nu a putut fi încărcată.");
    if (results[4].status === "fulfilled") setOfferingOptions(results[4].value); else setError("Lista completă de oferte nu a putut fi încărcată.");
    setLoading(false);
  }, [api, authorizationQuery, canManage, locationQuery, offeringQuery, supported]);
  useEffect(() => { void load(); }, [load]);

  if (!supported) return null;
  const run = async (operation: () => Promise<unknown>, close: () => void) => {
    setSaving(true); setError(undefined);
    try { await operation(); close(); await load(); }
    catch (reason) { setError((reason as { status?: number }).status === 409 ? "Datele s-au schimbat sau intervalul se suprapune. Reîncărcați și reluați operația." : "Operația nu a putut fi salvată."); }
    finally { setSaving(false); }
  };

  return <Card.Root><Card.Body><Card.Title><div className="flex flex-wrap items-center justify-between gap-2"><span>Ofertă educațională și autorizări</span><Button size="small" variant="outlined" severity="secondary" disabled={loading} onClick={() => void load()}><i className="pi pi-refresh" aria-hidden="true" />Reîncarcă</Button></div></Card.Title><Card.Subtitle>Autorizarea este urmărită distinct pe ofertă, locație și interval; forma juridică a școlii nu o înlocuiește.</Card.Subtitle><Card.Content>
    <div className="flex min-h-0 flex-col gap-3">
      {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
      {loading && !locations.items.length && !offerings.items.length && !authorizations.items.length ? <div className="flex justify-center p-8" role="status">{spinner}</div> : <Tabs.Root defaultValue="authorizations">
        <Tabs.List><Tabs.Tab value="authorizations">Autorizări</Tabs.Tab><Tabs.Tab value="offerings">Oferte</Tabs.Tab><Tabs.Tab value="locations">Locații</Tabs.Tab><Tabs.Indicator /></Tabs.List>
        <Tabs.Panel value="authorizations"><div className="mt-3"><AuthorizationTable page={authorizations} query={authorizationQuery} setQuery={setAuthorizationQuery} canManage={canManage} onAdd={() => setAuthorizationForm(newAuthorization())} onReplace={(item) => setAuthorizationForm(replacementAuthorization(item))} /></div></Tabs.Panel>
        <Tabs.Panel value="offerings"><div className="mt-3"><OfferingTable page={offerings} query={offeringQuery} setQuery={setOfferingQuery} canManage={canManage} onAdd={() => setOfferingForm(newOffering())} onEdit={(item) => setOfferingEdit({ item, value: { expected_version: item.expected_version, title: item.title, active: item.active, effective_to: item.effective_to } })} /></div></Tabs.Panel>
        <Tabs.Panel value="locations"><div className="mt-3"><LocationTable page={locations} query={locationQuery} setQuery={setLocationQuery} canManage={canManage} onAdd={() => setLocationForm(newLocation())} onEdit={(item) => setLocationEdit({ item, value: { expected_version: item.expected_version, name: item.name, address: item.address, active: item.active, effective_to: item.effective_to } })} /></div></Tabs.Panel>
      </Tabs.Root>}
    </div>
    {locationForm && <LocationDialog value={locationForm} saving={saving} onChange={setLocationForm} onClose={() => setLocationForm(undefined)} onSave={() => void run(() => api.createLocation!(locationForm), () => setLocationForm(undefined))} />}
    {locationEdit && api.updateLocation && <LocationEditDialog item={locationEdit.item} value={locationEdit.value} saving={saving} onChange={(value) => setLocationEdit({ ...locationEdit, value })} onClose={() => setLocationEdit(undefined)} onSave={() => void run(() => api.updateLocation!(locationEdit.item.id, locationEdit.value), () => setLocationEdit(undefined))} />}
    {offeringForm && <OfferingDialog value={offeringForm} saving={saving} onChange={setOfferingForm} onClose={() => setOfferingForm(undefined)} onSave={() => void run(() => api.createOffering!(offeringForm), () => setOfferingForm(undefined))} />}
    {offeringEdit && api.updateOffering && <OfferingEditDialog item={offeringEdit.item} value={offeringEdit.value} saving={saving} onChange={(value) => setOfferingEdit({ ...offeringEdit, value })} onClose={() => setOfferingEdit(undefined)} onSave={() => void run(() => api.updateOffering!(offeringEdit.item.id, offeringEdit.value), () => setOfferingEdit(undefined))} />}
    {authorizationForm && <AuthorizationDialog value={authorizationForm} locations={locationOptions} offerings={offeringOptions} saving={saving} onChange={setAuthorizationForm} onClose={() => setAuthorizationForm(undefined)} onSave={() => void run(() => api.createAuthorization!(authorizationForm), () => setAuthorizationForm(undefined))} />}
  </Card.Content></Card.Body></Card.Root>;
}

type QuerySetter = (value: InstitutionCatalogQuery | ((current: InstitutionCatalogQuery) => InstitutionCatalogQuery)) => void;
function updateFilter(setQuery: QuerySetter, field: string, value: string) { setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value } })); }
function sort(setQuery: QuerySetter, current: InstitutionCatalogQuery, field: string) { setQuery({ ...current, page: 1, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc" }); }
function SortHeader({ label, field, query: value, setQuery }: { label: string; field: string; query: InstitutionCatalogQuery; setQuery: QuerySetter }) { return <Button size="small" variant="text" severity="secondary" aria-label={`Sortează după ${label}`} onClick={() => sort(setQuery, value, field)}>{label}{value.sort === field ? value.direction === "asc" ? " ↑" : " ↓" : ""}</Button>; }
function Filter({ label, field, query: value, setQuery }: { label: string; field: string; query: InstitutionCatalogQuery; setQuery: QuerySetter }) { return <InputText aria-label={`Filtru ${label}`} value={value.filters?.[field] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => updateFilter(setQuery, field, event.target.value)} />; }
function AddHeader({ label, canManage, onAdd }: { label: string; canManage: boolean; onAdd: () => void }) { return <div className="flex items-center justify-between gap-2"><span>Acțiuni</span>{canManage && <Button iconOnly rounded size="small" aria-label={label} title={label} onClick={onAdd}><i className="pi pi-plus" aria-hidden="true" /></Button>}</div>; }
function Pager({ page, query: value, setQuery }: { page: { total: number; page: number; pageSize: number }; query: InstitutionCatalogQuery; setQuery: QuerySetter }) {
  const first = page.total ? (page.page - 1) * page.pageSize + 1 : 0; const last = Math.min(page.page * page.pageSize, page.total);
  return <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface bg-surface-ground pt-2"><span>{first}–{last} din {page.total}</span><div className="flex items-center gap-2"><Select.Root value={String(value.pageSize)} options={[10, 20, 50].map((size) => ({ label: `${size}/pagină`, value: String(size) }))} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setQuery({ ...value, page: 1, pageSize: Number(event.value) })}><Select.Trigger aria-label="Rânduri pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><Button size="small" variant="outlined" disabled={page.page <= 1} onClick={() => setQuery({ ...value, page: value.page - 1 })}>Anterior</Button><Button size="small" variant="outlined" disabled={last >= page.total} onClick={() => setQuery({ ...value, page: value.page + 1 })}>Următor</Button></div></div>;
}
function LocationTable({ page, query: value, setQuery, canManage, onAdd, onEdit }: { page: SchoolLocationPage; query: InstitutionCatalogQuery; setQuery: QuerySetter; canManage: boolean; onAdd: () => void; onEdit: (item: SchoolLocation) => void }) {
  return <div className="flex min-h-0 flex-col gap-2"><div className="max-h-[min(55dvh,38rem)] overflow-auto rounded-border border border-surface"><DataTable.Root data={page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><SortHeader label="Cod" field="code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Locație" field="name" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell>Adresă</DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Activă" field="active" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell>Interval</DataTable.THeadCell><DataTable.THeadCell><AddHeader label="Adaugă locație" canManage={canManage} onAdd={onAdd} /></DataTable.THeadCell></DataTable.THeadRow><DataTable.THeadRow><DataTable.THeadCell><Filter label="Cod" field="code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><Filter label="Locație" field="name" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell /><DataTable.THeadCell><Select.Root value={value.filters?.active || null} options={booleanOptions} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => updateFilter(setQuery, "active", String(event.value ?? ""))}><Select.Trigger aria-label="Filtru Activă"><Select.Value placeholder="Toate" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></DataTable.THeadCell><DataTable.THeadCell /><DataTable.THeadCell /></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const row = item as unknown as SchoolLocation; return <DataTable.Row index={index} key={row.id}><DataTable.Cell>{row.code}</DataTable.Cell><DataTable.Cell>{row.name}</DataTable.Cell><DataTable.Cell>{row.address || "—"}</DataTable.Cell><DataTable.Cell><Tag severity={row.active ? "success" : "secondary"}>{row.active ? "Activă" : "Inactivă"}</Tag></DataTable.Cell><DataTable.Cell>{row.effective_from} – {row.effective_to ?? "prezent"}</DataTable.Cell><DataTable.Cell>{canManage && <Button iconOnly rounded size="small" variant="text" aria-label={`Editează locația ${row.name}`} title="Editează" onClick={() => onEdit(row)}><i className="pi pi-pencil" aria-hidden="true" /></Button>}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div>{!page.items.length && <Empty /> }<Pager page={page} query={value} setQuery={setQuery} /></div>;
}
function OfferingTable({ page, query: value, setQuery, canManage, onAdd, onEdit }: { page: EducationOfferingPage; query: InstitutionCatalogQuery; setQuery: QuerySetter; canManage: boolean; onAdd: () => void; onEdit: (item: EducationOffering) => void }) {
  return <div className="flex min-h-0 flex-col gap-2"><div className="max-h-[min(55dvh,38rem)] overflow-auto rounded-border border border-surface"><DataTable.Root data={page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell><SortHeader label="Cod" field="code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Titlu" field="title" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Nivel" field="education_level" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell>Limbă</DataTable.THeadCell><DataTable.THeadCell>Stare</DataTable.THeadCell><DataTable.THeadCell>Interval</DataTable.THeadCell><DataTable.THeadCell><AddHeader label="Adaugă ofertă" canManage={canManage} onAdd={onAdd} /></DataTable.THeadCell></DataTable.THeadRow><DataTable.THeadRow><DataTable.THeadCell><Filter label="Cod" field="code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><Filter label="Titlu" field="title" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><Filter label="Nivel" field="education_level" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell /><DataTable.THeadCell /><DataTable.THeadCell /><DataTable.THeadCell /></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const row = item as unknown as EducationOffering; return <DataTable.Row index={index} key={row.id}><DataTable.Cell>{row.code}</DataTable.Cell><DataTable.Cell>{row.title}</DataTable.Cell><DataTable.Cell>{row.education_level}{row.specialization_code ? ` / ${row.specialization_code}` : ""}</DataTable.Cell><DataTable.Cell>{row.language_code}</DataTable.Cell><DataTable.Cell><Tag severity={row.active ? "success" : "secondary"}>{row.active ? "Activă" : "Inactivă"}</Tag></DataTable.Cell><DataTable.Cell>{row.effective_from} – {row.effective_to ?? "prezent"}</DataTable.Cell><DataTable.Cell>{canManage && <Button iconOnly rounded size="small" variant="text" aria-label={`Editează oferta ${row.title}`} title="Editează" onClick={() => onEdit(row)}><i className="pi pi-pencil" aria-hidden="true" /></Button>}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div>{!page.items.length && <Empty />}<Pager page={page} query={value} setQuery={setQuery} /></div>;
}
function AuthorizationTable({ page, query: value, setQuery, canManage, onAdd, onReplace }: { page: OfferingAuthorizationPage; query: InstitutionCatalogQuery; setQuery: QuerySetter; canManage: boolean; onAdd: () => void; onReplace: (value: OfferingAuthorization) => void }) {
  return <div className="flex min-h-0 flex-col gap-2">
    <div className="max-h-[min(55dvh,38rem)] overflow-auto rounded-border border border-surface">
      <DataTable.Root data={page.items as unknown as Record<string, unknown>[]} dataKey="id" scrollable>
        <DataTable.Table>
          <DataTable.THead className="sticky top-0 z-10">
            <DataTable.THeadRow><DataTable.THeadCell><SortHeader label="Ofertă" field="offering_code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Locație" field="location_code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Stare" field="status" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell>Decizie</DataTable.THeadCell><DataTable.THeadCell>Capacitate</DataTable.THeadCell><DataTable.THeadCell><SortHeader label="Interval" field="effective_from" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><AddHeader label="Adaugă decizie" canManage={canManage} onAdd={onAdd} /></DataTable.THeadCell></DataTable.THeadRow>
            <DataTable.THeadRow><DataTable.THeadCell><Filter label="Ofertă" field="offering_code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><Filter label="Locație" field="location_code" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><Filter label="Stare" field="status" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell><Filter label="Decizie" field="decision_reference" query={value} setQuery={setQuery} /></DataTable.THeadCell><DataTable.THeadCell /><DataTable.THeadCell /><DataTable.THeadCell /></DataTable.THeadRow>
          </DataTable.THead>
          <DataTable.TBody>{({ item, index }) => {
            const row = item as unknown as OfferingAuthorization;
            const severity = row.status === "accredited" || row.status === "provisional" ? "success" : row.status === "authorized" ? "warn" : "danger";
            return <DataTable.Row index={index} key={row.id}><DataTable.Cell><strong>{row.offering_code}</strong><div>{row.offering_title}</div></DataTable.Cell><DataTable.Cell>{row.location_name}</DataTable.Cell><DataTable.Cell><Tag severity={severity}>{row.status}</Tag></DataTable.Cell><DataTable.Cell>{row.decision_reference}<div><small>{row.authority_name}</small></div></DataTable.Cell><DataTable.Cell>{row.capacity == null ? "—" : `${row.capacity} ${row.capacity_unit === "study_groups" ? "formațiuni" : "elevi"}`}<div><small>{row.shift}</small></div></DataTable.Cell><DataTable.Cell>{row.effective_from} – {row.effective_to ?? "prezent"}</DataTable.Cell><DataTable.Cell>{canManage && <Button iconOnly rounded size="small" variant="text" aria-label={`Înlocuiește decizia ${row.decision_reference}`} title="Decizie nouă" onClick={() => onReplace(row)}><i className="pi pi-history" aria-hidden="true" /></Button>}</DataTable.Cell></DataTable.Row>;
          }}</DataTable.TBody>
        </DataTable.Table>
      </DataTable.Root>
    </div>
    {!page.items.length && <Empty />}
    <Pager page={page} query={value} setQuery={setQuery} />
  </div>;
}
function Empty() { return <Message.Root severity="info"><Message.Content><Message.Text>Nu există înregistrări pentru filtrele selectate.</Message.Text></Message.Content></Message.Root>; }
function Field({ label, children }: { label: string; children: ReactNode }) { return <label className="flex flex-col gap-1"><span>{label}</span>{children}</label>; }
function DialogFrame({ title, saving, valid, onClose, onSave, children }: { title: string; saving: boolean; valid: boolean; onClose: () => void; onSave: () => void; children: ReactNode }) { return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && !saving && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,54rem)]"><Dialog.Header><Dialog.Title>{title}</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content>{children}</Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !valid} onClick={onSave}>{saving ? "Se salvează…" : "Salvează"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>; }

function newLocation(): CreateSchoolLocationInput { return { code: "", name: "", address: "", active: true, effective_from: today(), effective_to: null, idempotency_key: key() }; }
function newOffering(): CreateEducationOfferingInput { return { code: "", education_level: "", specialization_code: "", language_code: "ro", title: "", active: true, effective_from: today(), effective_to: null, idempotency_key: key() }; }
function newAuthorization(): CreateOfferingAuthorizationInput { return { offering_id: "", location_id: "", status: "provisional", authority_name: "ARACIP", decision_reference: "", capacity: null, capacity_unit: "students", shift: "day", effective_from: today(), effective_to: null, source: { source_kind: "authorization", citation: "", article_reference: "", issuer: "ARACIP", source_url: "", published_on: null, consolidated_on: null, checksum_sha256: "" }, replaces_authorization_id: null, expected_version: null, idempotency_key: key() }; }
function replacementAuthorization(value: OfferingAuthorization): CreateOfferingAuthorizationInput { return { ...newAuthorization(), offering_id: value.offering_id, location_id: value.location_id, authority_name: value.authority_name, capacity: value.capacity, capacity_unit: value.capacity_unit, shift: value.shift, replaces_authorization_id: value.id, expected_version: value.expected_version }; }

function LocationDialog({ value, saving, onChange, onClose, onSave }: { value: CreateSchoolLocationInput; saving: boolean; onChange: (value: CreateSchoolLocationInput) => void; onClose: () => void; onSave: () => void }) {
  const set = (field: keyof CreateSchoolLocationInput, next: unknown) => onChange({ ...value, [field]: next });
  return <DialogFrame title="Locație nouă" saving={saving} valid={Boolean(value.code.trim() && value.name.trim() && value.effective_from && (value.active || value.effective_to))} onClose={onClose} onSave={onSave}><div className="grid gap-3 sm:grid-cols-2"><Field label="Cod"><InputText value={value.code} onChange={(e: ChangeEvent<HTMLInputElement>) => set("code", e.target.value)} /></Field><Field label="Denumire"><InputText value={value.name} onChange={(e: ChangeEvent<HTMLInputElement>) => set("name", e.target.value)} /></Field><Field label="Adresă"><InputText value={value.address} onChange={(e: ChangeEvent<HTMLInputElement>) => set("address", e.target.value)} /></Field><Field label="Activă"><Select.Root value={String(value.active)} options={booleanOptions} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("active", e.value === "true")}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field><Field label="În vigoare de la"><InputText type="date" value={value.effective_from} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_from", e.target.value)} /></Field><Field label="În vigoare până la"><InputText type="date" value={value.effective_to ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_to", e.target.value || null)} /></Field></div></DialogFrame>;
}
function OfferingDialog({ value, saving, onChange, onClose, onSave }: { value: CreateEducationOfferingInput; saving: boolean; onChange: (value: CreateEducationOfferingInput) => void; onClose: () => void; onSave: () => void }) {
  const set = (field: keyof CreateEducationOfferingInput, next: unknown) => onChange({ ...value, [field]: next });
  return <DialogFrame title="Ofertă educațională nouă" saving={saving} valid={Boolean(value.code.trim() && value.title.trim() && value.education_level.trim() && value.effective_from && (value.active || value.effective_to))} onClose={onClose} onSave={onSave}><div className="grid gap-3 sm:grid-cols-2"><Field label="Cod"><InputText value={value.code} onChange={(e: ChangeEvent<HTMLInputElement>) => set("code", e.target.value)} /></Field><Field label="Titlu"><InputText value={value.title} onChange={(e: ChangeEvent<HTMLInputElement>) => set("title", e.target.value)} /></Field><Field label="Nivel"><InputText value={value.education_level} onChange={(e: ChangeEvent<HTMLInputElement>) => set("education_level", e.target.value)} /></Field><Field label="Specializare"><InputText value={value.specialization_code} onChange={(e: ChangeEvent<HTMLInputElement>) => set("specialization_code", e.target.value)} /></Field><Field label="Limbă"><InputText value={value.language_code} onChange={(e: ChangeEvent<HTMLInputElement>) => set("language_code", e.target.value)} /></Field><Field label="Activă"><Select.Root value={String(value.active)} options={booleanOptions} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("active", e.value === "true")}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field><Field label="În vigoare de la"><InputText type="date" value={value.effective_from} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_from", e.target.value)} /></Field><Field label="În vigoare până la"><InputText type="date" value={value.effective_to ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_to", e.target.value || null)} /></Field></div></DialogFrame>;
}

function LocationEditDialog({ item, value, saving, onChange, onClose, onSave }: { item: SchoolLocation; value: UpdateSchoolLocationInput; saving: boolean; onChange: (value: UpdateSchoolLocationInput) => void; onClose: () => void; onSave: () => void }) {
  const set = (field: keyof UpdateSchoolLocationInput, next: unknown) => onChange({ ...value, [field]: next });
  return <DialogFrame title={`Editează locația ${item.code}`} saving={saving} valid={Boolean(value.expected_version > 0 && value.name.trim() && (value.active || value.effective_to))} onClose={onClose} onSave={onSave}><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Codul și începutul valabilității rămân imuabile. Dezactivarea cere o dată de încetare și nu poate invalida autorizări existente.</Message.Text></Message.Content></Message.Root><div className="grid gap-3 sm:grid-cols-2"><Field label="Cod"><InputText value={item.code} disabled /></Field><Field label="În vigoare de la"><InputText type="date" value={item.effective_from} disabled /></Field><Field label="Denumire"><InputText value={value.name} onChange={(e: ChangeEvent<HTMLInputElement>) => set("name", e.target.value)} /></Field><Field label="Adresă"><InputText value={value.address} onChange={(e: ChangeEvent<HTMLInputElement>) => set("address", e.target.value)} /></Field><Field label="Activă"><Select.Root value={String(value.active)} options={booleanOptions} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("active", e.value === "true")}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field><Field label="În vigoare până la"><InputText type="date" value={value.effective_to ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_to", e.target.value || null)} /></Field></div></div></DialogFrame>;
}

function OfferingEditDialog({ item, value, saving, onChange, onClose, onSave }: { item: EducationOffering; value: UpdateEducationOfferingInput; saving: boolean; onChange: (value: UpdateEducationOfferingInput) => void; onClose: () => void; onSave: () => void }) {
  const set = (field: keyof UpdateEducationOfferingInput, next: unknown) => onChange({ ...value, [field]: next });
  return <DialogFrame title={`Editează oferta ${item.code}`} saving={saving} valid={Boolean(value.expected_version > 0 && value.title.trim() && (value.active || value.effective_to))} onClose={onClose} onSave={onSave}><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Nivelul, specializarea, limba și începutul valabilității rămân imuabile. Dezactivarea cere o dată de încetare și nu poate invalida autorizări existente.</Message.Text></Message.Content></Message.Root><div className="grid gap-3 sm:grid-cols-2"><Field label="Cod"><InputText value={item.code} disabled /></Field><Field label="Nivel / specializare"><InputText value={`${item.education_level}${item.specialization_code ? ` / ${item.specialization_code}` : ""}`} disabled /></Field><Field label="Titlu"><InputText value={value.title} onChange={(e: ChangeEvent<HTMLInputElement>) => set("title", e.target.value)} /></Field><Field label="Activă"><Select.Root value={String(value.active)} options={booleanOptions} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("active", e.value === "true")}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field><Field label="În vigoare de la"><InputText type="date" value={item.effective_from} disabled /></Field><Field label="În vigoare până la"><InputText type="date" value={value.effective_to ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_to", e.target.value || null)} /></Field></div></div></DialogFrame>;
}
function AuthorizationDialog({ value, locations, offerings, saving, onChange, onClose, onSave }: { value: CreateOfferingAuthorizationInput; locations: SchoolLocation[]; offerings: EducationOffering[]; saving: boolean; onChange: (value: CreateOfferingAuthorizationInput) => void; onClose: () => void; onSave: () => void }) {
  const set = (field: keyof CreateOfferingAuthorizationInput, next: unknown) => onChange({ ...value, [field]: next });
  const source = (field: keyof CreateOfferingAuthorizationInput["source"], next: unknown) => onChange({ ...value, source: { ...value.source, [field]: next } });
  const positive = value.status === "provisional" || value.status === "accredited";
  const valid = Boolean(value.offering_id && value.location_id && value.status && value.capacity_unit && value.shift && (!positive || (value.capacity ?? 0) > 0) && value.authority_name.trim() && value.decision_reference.trim() && value.effective_from && value.source.citation.trim() && value.source.source_url.startsWith("https://") && /^[a-f0-9]{64}$/.test(value.source.checksum_sha256));
  return <DialogFrame title={value.replaces_authorization_id ? "Decizie de înlocuire" : "Decizie de autorizare"} saving={saving} valid={valid} onClose={onClose} onSave={onSave}>
    <div className="grid gap-3 sm:grid-cols-2">
      <Field label="Ofertă"><Select.Root value={value.offering_id || null} disabled={Boolean(value.replaces_authorization_id)} options={offerings.filter((item) => item.active).map((item) => ({ label: `${item.code} — ${item.title}`, value: item.id }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("offering_id", String(e.value ?? ""))}><Select.Trigger><Select.Value placeholder="Selectați oferta" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
      <Field label="Locație"><Select.Root value={value.location_id || null} disabled={Boolean(value.replaces_authorization_id)} options={locations.filter((item) => item.active).map((item) => ({ label: `${item.code} — ${item.name}`, value: item.id }))} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("location_id", String(e.value ?? ""))}><Select.Trigger><Select.Value placeholder="Selectați locația" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
      <Field label="Stare"><Select.Root value={value.status} options={authorizationStatuses} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("status", String(e.value))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
      <Field label="Capacitate"><InputNumber.Root value={value.capacity} min={positive ? 1 : 0} onValueChange={(e: InputNumberRootValueChangeEvent) => set("capacity", e.value ?? null)}><InputNumber.Input /></InputNumber.Root></Field>
      <Field label="Unitate capacitate"><Select.Root value={value.capacity_unit} options={capacityUnits} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("capacity_unit", String(e.value))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
      <Field label="Schimb"><Select.Root value={value.shift} options={studyShifts} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => set("shift", String(e.value))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
      <Field label="Autoritate"><InputText value={value.authority_name} onChange={(e: ChangeEvent<HTMLInputElement>) => set("authority_name", e.target.value)} /></Field>
      <Field label="Referință decizie"><InputText value={value.decision_reference} onChange={(e: ChangeEvent<HTMLInputElement>) => set("decision_reference", e.target.value)} /></Field>
      <Field label="În vigoare de la"><InputText type="date" value={value.effective_from} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_from", e.target.value)} /></Field>
      <Field label="În vigoare până la"><InputText type="date" value={value.effective_to ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => set("effective_to", e.target.value || null)} /></Field>
      <Field label="Tip sursă"><Select.Root value={value.source.source_kind} options={sourceKinds} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => source("source_kind", String(e.value))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></Field>
      <Field label="Citare"><InputText value={value.source.citation} onChange={(e: ChangeEvent<HTMLInputElement>) => source("citation", e.target.value)} /></Field>
      <Field label="Emitent"><InputText value={value.source.issuer} onChange={(e: ChangeEvent<HTMLInputElement>) => source("issuer", e.target.value)} /></Field>
      <Field label="Articol / anexă"><InputText value={value.source.article_reference} onChange={(e: ChangeEvent<HTMLInputElement>) => source("article_reference", e.target.value)} /></Field>
      <Field label="URL oficial HTTPS"><InputText value={value.source.source_url} onChange={(e: ChangeEvent<HTMLInputElement>) => source("source_url", e.target.value)} /></Field>
      <Field label="SHA-256"><InputText value={value.source.checksum_sha256} onChange={(e: ChangeEvent<HTMLInputElement>) => source("checksum_sha256", e.target.value.toLowerCase())} /></Field>
      <Field label="Publicat la"><InputText type="date" value={value.source.published_on ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => source("published_on", e.target.value || null)} /></Field>
      <Field label="Consolidat la"><InputText type="date" value={value.source.consolidated_on ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => source("consolidated_on", e.target.value || null)} /></Field>
    </div>
  </DialogFrame>;
}
