import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type KeyboardEvent,
} from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Dialog } from "@primereact/ui/dialog";
import { DataTable } from "@primereact/ui/datatable";
import { InputText } from "@primereact/ui/inputtext";
import { Textarea } from "@primereact/ui/textarea";
import { FileUpload } from "@primereact/ui/fileupload";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select } from "@primereact/ui/select";
import { Tabs } from "@primereact/ui/tabs";
import { Search, Send, Inbox, Copy, Times, FilePdf, Upload, Pencil, Ban, Users, Cog, History, ShareAlt, ChevronRight, ChevronDown, SortAlt } from "@primeicons/react";
import { createRegistraturaApi, type RegistraturaApi } from "./api";
import type {
  BatchCreateInput,
  CreateDocumentInput,
  DocumentFilters,
  Registry,
  RegistryDocument,
  DocumentAttachment,
  DocumentVersion,
  WorkflowHistoryEntry,
  WorkflowAction,
  WorkflowAssignees,
  Party,
  RegistryAdminRecord,
  DocumentFilterOptions,
  OrganizationChartNode,
  AdminUser,
  UserAssignment,
  LinkedDocument,
} from "./types";
import {
  calendarDateLabel,
  directionLabel,
  isTerminalStatus,
  permittedActions,
  statusLabel,
} from "./workflow";

type CreateMode = "intrare" | "iesire" | "multiplu" | null;
type DocumentView = "details" | "history" | "edit" | "cancel" | "workflow";
const blank = (
  registryId: number,
  direction: "intrare" | "iesire",
): CreateDocumentInput => ({
  registru_id: registryId,
  subject: "",
  document_type: "DOCUMENT",
  direction,
  status: "INCOMING",
  correspondent: "",
  assigned_to: "",
  confidentiality: "normal",
  summary: "",
});
const storageKey = (tenantKey: string) =>
  `egueducation.registratura.registry.${tenantKey}`;
const validExportRange = (start: string, end: string) => {
  if (!start || !end || start > end) return false;
  return (Date.parse(end) - Date.parse(start)) / 86400000 <= 30;
};
const Spinner = () => (
  <ProgressSpinner.Root>
    <ProgressSpinner.Range>
      <ProgressSpinner.Track />
      <ProgressSpinner.Value />
    </ProgressSpinner.Range>
  </ProgressSpinner.Root>
);
const isRegistryDocument = (
  value: Record<string, unknown>,
): value is RegistryDocument & Record<string, unknown> =>
  typeof value.id === "string" &&
  typeof value.registry_number === "string" &&
  typeof value.subject === "string" &&
  typeof value.document_type === "string" &&
  typeof value.direction === "string" &&
  typeof value.status === "string" &&
  typeof value.correspondent === "string" &&
  typeof value.assigned_to === "string" &&
    typeof value.registered_at === "string";
const ChartTree = ({ nodes }: { nodes: OrganizationChartNode[] }) => <div className="flex flex-col gap-2">{nodes.map((node) => <Card.Root key={node.id}><Card.Body><Card.Content><div className="flex flex-col gap-1"><strong>{node.name}</strong><span>{node.role_tag || "Fără rol"} · {node.user_count} utilizatori</span>{node.users.map((user) => <span key={user.id}>{user.name}{user.email ? ` (${user.email})` : ""}</span>)}{node.children.length > 0 && <div className="ml-4"><ChartTree nodes={node.children} /></div>}</div></Card.Content></Card.Body></Card.Root>)}</div>;

export interface RegistraturaWorkspaceProps {
  api?: RegistraturaApi;
  tenantKey: string;
  canManage?: boolean;
  canManageWorkflow?: boolean;
  canReadAdminUsers?: boolean;
  canReadLinks?: boolean;
  canManageLinks?: boolean;
}
export function RegistraturaWorkspace({
  api = createRegistraturaApi(),
  tenantKey,
  canManage = false,
  canManageWorkflow = false,
  canReadAdminUsers = false,
  canReadLinks = false,
  canManageLinks = false,
}: RegistraturaWorkspaceProps) {
  const [registries, setRegistries] = useState<Registry[]>([]);
  const [registryId, setRegistryId] = useState<number>();
  const [documents, setDocuments] = useState<RegistryDocument[]>([]);
  const [filters, setFilters] = useState<DocumentFilters>({});
  const [appliedFilters, setAppliedFilters] = useState<DocumentFilters>({});
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [filterOptions, setFilterOptions] = useState<DocumentFilterOptions>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [sort, setSort] = useState("registered_at");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("desc");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [mode, setMode] = useState<CreateMode>(null);
  const [exportOpen, setExportOpen] = useState(false);
  const [exportStart, setExportStart] = useState("");
  const [exportEnd, setExportEnd] = useState("");
  const [form, setForm] = useState<CreateDocumentInput>();
  const [count, setCount] = useState("1");
  const [createAttachments, setCreateAttachments] = useState<File[]>([]);
  const [partyOptions, setPartyOptions] = useState<Party[]>([]);
  const [partySearch, setPartySearch] = useState("");
  const [departmentOptions, setDepartmentOptions] = useState<RegistryAdminRecord[]>([]);
  const [saving, setSaving] = useState(false);
  const [selected, setSelected] = useState<RegistryDocument>();
  const [documentView, setDocumentView] = useState<DocumentView>("details");
  const [expandedDocuments, setExpandedDocuments] = useState<Record<string, RegistryDocument>>({});
  const [detailLoading, setDetailLoading] = useState(false);
  const [attachments, setAttachments] = useState<DocumentAttachment[]>([]);
  const [versions, setVersions] = useState<DocumentVersion[]>([]);
  const [history, setHistory] = useState<WorkflowHistoryEntry[]>([]);
  const [assignees, setAssignees] = useState<WorkflowAssignees>();
  const [workflowAction, setWorkflowAction] = useState<WorkflowAction>();
  const [workflowDepartment, setWorkflowDepartment] = useState("");
  const [workflowUser, setWorkflowUser] = useState("");
  const [workflowNote, setWorkflowNote] = useState("");
  const [cancelReason, setCancelReason] = useState("");
  const [linkedDocuments, setLinkedDocuments] = useState<LinkedDocument[]>([]);
  const [linkSourceModule, setLinkSourceModule] = useState("");
  const [linkSourceRecordId, setLinkSourceRecordId] = useState("");
  const [linkRelationType, setLinkRelationType] = useState("supporting");
  const [detailError, setDetailError] = useState<string>();
  const [adminOpen, setAdminOpen] = useState(false);
  const [adminResource, setAdminResource] = useState<"parties" | "departments" | "organizations" | "registries">("parties");
  const [adminItems, setAdminItems] = useState<(Party | RegistryAdminRecord)[]>([]);
  const [adminName, setAdminName] = useState("");
  const [adminEditing, setAdminEditing] = useState<{ item: Party | RegistryAdminRecord; resource: "parties" | "departments" | "organizations" | "registries" }>();
  const [adminDraft, setAdminDraft] = useState<Record<string, unknown>>({});
  const [specialistEditing, setSpecialistEditing] = useState<{ item: Party | RegistryAdminRecord; resource: "parties" | "departments" | "organizations" | "registries" }>();
  const [chart, setChart] = useState<OrganizationChartNode[]>([]);
  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([]);
  const [assignmentUser, setAssignmentUser] = useState<AdminUser>();
  const [assignment, setAssignment] = useState<UserAssignment>();
  const [pendingAdminDeletion, setPendingAdminDeletion] = useState<{ item: Party | RegistryAdminRecord; resource: "parties" | "departments" | "organizations" | "registries" }>();
  const [editing, setEditing] = useState(false);
  const [editForm, setEditForm] = useState<(CreateDocumentInput & { change_notes?: string })>();
  const listRequestSequence = useRef(0);
  const detailRequestSequence = useRef(0);
  const load = useCallback(
    async (activeRegistryId = registryId, activeFilters = appliedFilters) => {
      if (!activeRegistryId) return;
      const requestSequence = ++listRequestSequence.current;
      setLoading(true);
      setError(undefined);
      try {
        const result = await api.documents({
          registryId: activeRegistryId,
          page,
          pageSize,
          filters: activeFilters,
          sort,
          direction: sortDirection,
        });
        if (requestSequence !== listRequestSequence.current) return;
        setDocuments(result.items); setTotal(result.total);
      } catch {
        if (requestSequence !== listRequestSequence.current) return;
        setError("Documentele nu au putut fi încărcate. Încercați din nou.");
      } finally {
        if (requestSequence === listRequestSequence.current) setLoading(false);
      }
    },
    [api, appliedFilters, page, pageSize, registryId, sort, sortDirection],
  );
  useEffect(() => {
    void api
      .registries()
      .then((items) => {
        setRegistries(items);
        const stored = Number(sessionStorage.getItem(storageKey(tenantKey)));
        const next = items.some((item) => item.id === stored)
          ? stored
          : (items.find((item) => item.isDefault)?.id ?? items[0]?.id);
        setRegistryId(next);
      })
      .catch(() => {
        setError("Registrele disponibile nu au putut fi încărcate.");
        setLoading(false);
      });
    void api.filters().then(setFilterOptions).catch(() => undefined);
  }, [api, tenantKey]);
  useEffect(() => () => {
    listRequestSequence.current += 1;
    detailRequestSequence.current += 1;
  }, []);
  useEffect(() => {
    if (registryId) {
      sessionStorage.setItem(storageKey(tenantKey), String(registryId));
      void load(registryId);
    }
  }, [load, registryId, tenantKey]);
  const loadCreateLookups = async () => {
    try {
      const [partyResult, departmentResult] = await Promise.all([
        api.parties(partySearch),
        api.admin("departments"),
      ]);
      setPartyOptions(partyResult.items);
      setDepartmentOptions(departmentResult.items);
    } catch {
      setError("Listele de persoane și compartimente nu au putut fi încărcate.");
    }
  };
  const openCreate = (nextMode: CreateMode) => {
    if (!registryId || !nextMode) return;
    setForm(blank(registryId, nextMode === "iesire" ? "iesire" : "intrare"));
    setCount("1");
    setCreateAttachments([]);
    setPartySearch("");
    setMode(nextMode);
    void loadCreateLookups();
  };
  const submit = async () => {
    if (!form || !registryId) return;
    if (mode !== "multiplu" && (!form.subject.trim() || !form.correspondent.trim() || !form.assigned_to.trim() || !(form.department_ids?.length))) {
      setError("Conținutul, emitentul, destinatarul și compartimentul sunt obligatorii.");
      return;
    }
    setSaving(true);
    try {
      let created: RegistryDocument[] = [];
      if (mode === "multiplu") {
        const numericCount = Number(count);
        if (
          !Number.isInteger(numericCount) ||
          numericCount < 1 ||
          numericCount > 20
        )
          throw new Error("INVALID_COUNT");
        const payload: BatchCreateInput = {
          ...form,
          count: numericCount,
          document_type: "MULTIPLU",
        };
        created = await api.createBatch(payload);
      } else {
        created = [await api.create({
          ...form,
          subject: form.subject.trim(),
          document_type: form.document_type.trim() || "DOCUMENT",
        })];
      }
      if (createAttachments.length > 0) {
        await Promise.all(created.flatMap((document) => createAttachments.map((file) => api.upload(document.id, file, "primary"))));
      }
      setMode(null);
      await load(registryId);
      if (created[0]) await openDetail(created[0]);
    } catch (reason) {
      setError(
        reason instanceof Error && reason.message === "INVALID_COUNT"
          ? "Pentru MULTIPLU sunt admise între 1 și 20 de înregistrări."
          : "Înregistrarea nu a putut fi creată.",
      );
    } finally {
      setSaving(false);
    }
  };
  const actionHint = useMemo(
    () =>
      documents.reduce<Record<string, string>>((result, item) => {
        const actions = permittedActions(item.status);
        result[item.id] = isTerminalStatus(item.status)
          ? "Finalizat"
          : actions.length
            ? actions.join(", ")
            : "Fără acțiuni";
        return result;
      }, {}),
    [documents],
  );
  const openDetail = async (document: RegistryDocument, view: DocumentView = "details") => {
    const requestSequence = ++detailRequestSequence.current;
    setDocumentView(view); setEditing(view === "edit"); setSelected(document); setDetailLoading(true); setDetailError(undefined); setWorkflowAction(undefined); setCancelReason("");
    try {
      const [full, nextVersions, nextAttachments, nextHistory, nextAssignees] = await Promise.all([api.document(document.id), api.versions(document.id), api.attachments(document.id), api.workflowHistory(document.id), api.assignees()]);
      if (requestSequence !== detailRequestSequence.current) return;
      setSelected(full); setEditForm({ registru_id: full.registru_id ?? registryId ?? 0, subject: full.subject, document_type: full.document_type, direction: full.direction === "iesire" ? "iesire" : "intrare", status: full.status, correspondent: full.correspondent, assigned_to: full.assigned_to, confidentiality: full.confidentiality ?? "normal", summary: full.summary ?? "", due_date: full.due_date, external_number: full.external_number, external_number_date: full.external_number_date, entry_at: full.entry_at, exit_at: full.exit_at, activity: full.activity, record_kind: full.record_kind === "dosar" ? "dosar" : "document", department_ids: full.department_ids ?? [] }); setVersions(nextVersions); setAttachments(nextAttachments); setHistory(nextHistory); setAssignees(nextAssignees);
    } catch { if (requestSequence === detailRequestSequence.current) setDetailError("Detaliile documentului nu au putut fi încărcate."); }
    finally { if (requestSequence === detailRequestSequence.current) setDetailLoading(false); }
  };
  const toggleExpanded = async (document: RegistryDocument) => {
    if (expandedDocuments[document.id]) {
      setExpandedDocuments((current) => { const next = { ...current }; delete next[document.id]; return next; });
      return;
    }
    try {
      const full = await api.document(document.id);
      setExpandedDocuments((current) => ({ ...current, [document.id]: full }));
    } catch { setError("Detaliile rândului nu au putut fi încărcate."); }
  };
  const changeSort = (field: string) => {
    setPage(1);
    if (sort === field) setSortDirection((value) => value === "asc" ? "desc" : "asc");
    else { setSort(field); setSortDirection("asc"); }
  };
  const applyFilters = () => {
    setPage(1);
    setAppliedFilters({ ...filters });
  };
  const resetFilters = () => {
    setPage(1);
    setFilters({});
    setAppliedFilters({});
  };
  const sortableHeader = (label: string, field: string) => <Button variant="text" severity="secondary" aria-label={`Sortează după ${label}`} onClick={() => changeSort(field)}>{label}<SortAlt />{sort === field ? <span>{sortDirection === "asc" ? "↑" : "↓"}</span> : null}</Button>;
  const refreshDetail = async () => { if (selected) await openDetail(selected); };
  const saveEdit = async () => { if (!selected || !editForm || !editForm.subject.trim() || !editForm.change_notes?.trim()) { setDetailError("Subiectul și nota modificării sunt obligatorii."); return; } setSaving(true); try { await api.update(selected.id, { ...editForm, subject: editForm.subject.trim(), expected_workflow_version: selected.workflow_version ?? null }); setEditing(false); await refreshDetail(); await load(); } catch { setDetailError("Modificarea a fost respinsă; verificați câmpurile sau versiunea fluxului."); } finally { setSaving(false); } };
  const createVersion = async () => { if (!selected || !editForm || !editForm.change_notes?.trim()) { setDetailError("Nota versiunii este obligatorie."); return; } setSaving(true); try { await api.createVersion(selected.id, { subject: editForm.subject, status: editForm.status, assigned_to: editForm.assigned_to, confidentiality: editForm.confidentiality, summary: editForm.summary, due_date: editForm.due_date, change_notes: editForm.change_notes }); await refreshDetail(); } catch { setDetailError("Versiunea nu a putut fi creată."); } finally { setSaving(false); } };
  const saveBlob = (blob: Blob, name: string) => { const url = URL.createObjectURL(blob); const link = document.createElement("a"); link.href = url; link.download = name; link.click(); URL.revokeObjectURL(url); };
  const runWorkflow = async () => {
    if (!selected || !workflowAction) return;
    if (workflowAction === "assign_department" && !workflowDepartment) { setDetailError("Alegeți compartimentul."); return; }
    if (workflowAction === "assign_user" && !workflowUser) { setDetailError("Alegeți utilizatorul."); return; }
    setSaving(true); setDetailError(undefined);
    try { await api.workflow(selected.id, { action: workflowAction, expected_version: selected.workflow_version ?? null, note: workflowNote.trim() || null, department_id: workflowAction === "assign_department" ? workflowDepartment : undefined, user_id: workflowAction === "assign_user" ? workflowUser : undefined }); await refreshDetail(); await load(); setWorkflowAction(undefined); }
    catch { setDetailError("Acțiunea de flux a fost respinsă sau documentul a fost schimbat între timp."); } finally { setSaving(false); }
  };
  const cancelDocument = async () => { if (!selected || cancelReason.trim().length < 10) { setDetailError("Motivul anulării trebuie să conțină cel puțin 10 caractere."); return; } setSaving(true); try { await api.cancel(selected.id, cancelReason.trim()); await refreshDetail(); await load(); } catch { setDetailError("Documentul nu a putut fi anulat."); } finally { setSaving(false); } };
  const loadLinks = async () => { if (!canReadLinks || !linkSourceModule.trim() || !linkSourceRecordId.trim()) return; try { setLinkedDocuments(await api.links(linkSourceModule.trim(), linkSourceRecordId.trim())); } catch { setDetailError("Legăturile documentului nu au putut fi încărcate."); } };
  const createLink = async () => { if (!selected || !canManageLinks || !linkSourceModule.trim() || !linkSourceRecordId.trim()) return; setSaving(true); try { await api.createLink({ document_id: selected.id, source_module: linkSourceModule.trim(), source_record_id: linkSourceRecordId.trim(), relation_type: linkRelationType }); await loadLinks(); } catch { setDetailError("Legătura documentului nu a putut fi creată."); } finally { setSaving(false); } };
  const deleteLink = async (linkId: string) => { if (!canManageLinks) return; setSaving(true); try { await api.deleteLink(linkId); await loadLinks(); } catch { setDetailError("Legătura documentului nu a putut fi ștearsă."); } finally { setSaving(false); } };
  const upload = async (files: File[]) => { if (!selected || files.length === 0) return; setSaving(true); try { for (const file of files) await api.upload(selected.id, file, "primary"); await refreshDetail(); } catch { setDetailError("Fișierul nu a putut fi încărcat sau verificat."); } finally { setSaving(false); } };
  const loadAdmin = async (resource = adminResource) => { setAdminResource(resource); setAdminName(""); try { const result = resource === "parties" ? await api.parties() : await api.admin(resource); setAdminItems(result.items); } catch { setDetailError("Datele de administrare nu au putut fi încărcate."); } };
  const openAdmin = () => { setAdminOpen(true); void loadAdmin("parties"); void api.admin("departments").then((result) => setDepartmentOptions(result.items)).catch(() => undefined); };
  const createAdmin = async () => { if (!adminName.trim()) return; setSaving(true); try { if (adminResource === "parties") await api.createParty({ party_type: "physical", display_name: adminName.trim(), active: true, is_default_organization: false }); else await api.createAdmin(adminResource, adminResource === "registries" ? { name: adminName.trim(), prefix: "REG", start_number: 1, registry_type: "public", is_default: false, department_ids: [] } : adminResource === "organizations" ? { name: adminName.trim(), description: "", active: true, is_default: false, department_ids: [] } : { name: adminName.trim(), description: "", role_tag: "", parent_id: null, active: true }); await loadAdmin(); } catch { setDetailError("Elementul nu a putut fi creat."); } finally { setSaving(false); } };
  const editAdmin = (item: Party | RegistryAdminRecord, resource: typeof adminResource) => {
    const raw = item as Record<string, unknown>;
    const draft = resource === "registries" ? { name: raw.name ?? raw.nume ?? "", prefix: raw.prefix ?? raw.prefix_nr ?? "", start_number: raw.start_number ?? raw.nr_inceput ?? 1, current_number: raw.current_number ?? raw.nr_curent ?? "", next_number: raw.next_number ?? raw.nr_urmator ?? "", registry_type: raw.registry_type ?? raw.tip_registru ?? "public", is_default: raw.is_default ?? raw.isDefault ?? false, department_ids: raw.department_ids ?? [] } : resource === "parties" ? { ...raw } : { name: raw.name ?? raw.nume ?? "", description: raw.description ?? "", active: raw.active ?? true, ...(resource === "departments" ? { parent_id: raw.parent_id ?? null, role_tag: raw.role_tag ?? "" } : { is_default: raw.is_default ?? false, department_ids: raw.department_ids ?? [] }) };
    setAdminDraft(draft); setSpecialistEditing({ item, resource });
  };
  const saveAdminEdit = async () => { if (!adminEditing || !String(adminDraft.name ?? adminDraft.display_name ?? "").trim()) return; setSaving(true); try { if (adminEditing.resource === "parties") await api.updateParty(String(adminEditing.item.id), adminDraft as Partial<Party>); else await api.updateAdmin(adminEditing.resource, adminEditing.item.id, adminDraft); setAdminEditing(undefined); await loadAdmin(adminEditing.resource); } catch { setDetailError("Modificarea nu a putut fi salvată."); } finally { setSaving(false); } };
  const saveSpecialistEdit = async () => { if (!specialistEditing || !String(adminDraft.name ?? adminDraft.display_name ?? "").trim()) return; setSaving(true); try { if (specialistEditing.resource === "parties") await api.updateParty(String(specialistEditing.item.id), adminDraft as Partial<Party>); else await api.updateAdmin(specialistEditing.resource, specialistEditing.item.id, adminDraft); const resource = specialistEditing.resource; setSpecialistEditing(undefined); await loadAdmin(resource); } catch { setDetailError("Modificarea nu a putut fi salvată."); } finally { setSaving(false); } };
  const deleteAdminItem = async () => {
    const target = pendingAdminDeletion;
    if (!target) return;
    setSaving(true); setDetailError(undefined);
    try {
      if (target.resource === "parties") await api.deleteParty(String(target.item.id));
      else await api.deleteAdmin(target.resource, target.item.id);
      setPendingAdminDeletion(undefined);
      await loadAdmin(target.resource);
    } catch { setDetailError("Elementul nu a putut fi șters."); }
    finally { setSaving(false); }
  };
  const loadChart = async () => { try { setChart(await api.chart()); } catch { setDetailError("Organigrama nu a putut fi încărcată."); } };
  const openAssignments = async () => {
    if (!canReadAdminUsers) return;
    try { setAdminUsers((await api.adminUsers()).items); } catch { setDetailError("Lista utilizatorilor nu a putut fi încărcată."); }
  };
  const selectAssignmentUser = async (id: string) => {
    const user = adminUsers.find((item) => item.id === id); if (!user) return;
    setAssignmentUser(user); try { setAssignment(await api.userAssignments(user.id)); } catch { setDetailError("Atribuirile utilizatorului nu au putut fi încărcate."); }
  };
  const saveAssignments = async () => { if (!canManage || !assignmentUser || !assignment) return; setSaving(true); try { setAssignment(await api.saveUserAssignments(assignmentUser.id, { department_ids: assignment.department_ids, primary_department_id: assignment.primary_department_id ?? null, organization_id: assignment.organization_id ?? null })); } catch { setDetailError("Atribuirile utilizatorului nu au putut fi salvate."); } finally { setSaving(false); } };
  return (
    <section aria-label="Registratură" className="flex flex-col gap-4">
      <Tabs.Root value="registratura">
        <Tabs.List>
          <Tabs.Tab value="registratura">Registratură</Tabs.Tab>
          <Tabs.Tab value="flux" disabled>
            Flux documente
          </Tabs.Tab>
          <Tabs.Tab value="arhiva" disabled>
            eArhivă
          </Tabs.Tab>
          <Tabs.Indicator />
        </Tabs.List>
      </Tabs.Root>
      <Card.Root>
        <Card.Body>
          <Card.Content>
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <div className="flex flex-wrap gap-2">
                  <Button
                    onClick={() => openCreate("intrare")}
                    disabled={!canManage || !registryId}
                  >
                    <Inbox />
                    Intrare
                  </Button>
                  <Button
                    severity="success"
                    onClick={() => openCreate("iesire")}
                    disabled={!canManage || !registryId}
                  >
                    <Send />
                    Ieșire
                  </Button>
                  <Button
                    variant="outlined"
                    severity="secondary"
                    onClick={() => openCreate("multiplu")}
                    disabled={!canManage || !registryId}
                  >
                    <Copy />
                    MULTIPLU
                  </Button>
                  {canManage && <Button variant="outlined" severity="secondary" onClick={openAdmin}><Cog /> Administrare</Button>}
                  <Button variant="outlined" severity="secondary" disabled={!registryId} onClick={() => setExportOpen(true)}><FilePdf /> Export PDF</Button>
                  <Button
                    variant="outlined"
                    severity="secondary"
                    aria-label={filtersOpen ? "Închide căutarea" : "Deschide căutarea"}
                    aria-expanded={filtersOpen}
                    aria-controls="registratura-search-panel"
                    title={filtersOpen ? "Închide căutarea" : "Deschide căutarea"}
                    onClick={() => setFiltersOpen((value) => !value)}
                  >
                    <Search />
                  </Button>
                </div>
                <div className="w-full lg:w-80">
                  <Select.Root
                    value={registryId}
                    options={registries}
                    optionLabel="nume"
                    optionValue="id"
                    fluid
                    onValueChange={(event: { value: unknown }) =>
                      setRegistryId(Number(event.value))
                    }
                  >
                    <Select.Trigger>
                      <Select.Value placeholder="Alege registrul" />
                      <Select.Indicator />
                    </Select.Trigger>
                    <Select.Portal>
                      <Select.Positioner>
                        <Select.Popup>
                          <Select.List />
                        </Select.Popup>
                      </Select.Positioner>
                    </Select.Portal>
                  </Select.Root>
                </div>
              </div>
              {filtersOpen && <div id="registratura-search-panel" aria-label="Căutare documente"><Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-3">
                <div className="flex items-center justify-between gap-2"><strong>Filtrare documente</strong><Button variant="text" severity="secondary" aria-label="Închide filtrarea" onClick={() => setFiltersOpen(false)}><Times /></Button></div>
                <div className="grid gap-3 md:grid-cols-3">
                  <InputText aria-label="Nr. Document" value={filters.registry_number ?? ""} placeholder="Nr. Document (ex: 123)" onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, registry_number: event.target.value }))} />
                  <Select.Root value={filters.document_type ?? ""} options={(filterOptions?.document_types ?? []).map((value) => ({ label: value, value }))} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setFilters((current) => ({ ...current, document_type: String(event.value ?? "") }))}><Select.Trigger><Select.Value placeholder="Tip document: Toate" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>
                  <InputText aria-label="Nr. Extern" value={filters.external_number ?? ""} placeholder="Nr. Extern (ex: ABC-123)" onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, external_number: event.target.value }))} />
                  <InputText aria-label="Emitent" value={filters.correspondent ?? ""} placeholder="Caută emitent" onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, correspondent: event.target.value }))} />
                  <InputText aria-label="Destinatar" value={filters.assigned_to ?? ""} placeholder="Caută destinatar" onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, assigned_to: event.target.value }))} />
                  <InputText aria-label="Conținut" value={filters.subject ?? ""} placeholder="Caută în conținut" onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, subject: event.target.value }))} />
                </div>
                <div className="grid gap-3 md:grid-cols-4">
                  <InputText aria-label="Data intrare de la" type="date" value={filters.entry_at_from ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, entry_at_from: event.target.value }))} />
                  <InputText aria-label="Data intrare până la" type="date" value={filters.entry_at_to ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, entry_at_to: event.target.value }))} />
                  <InputText aria-label="Data ieșire de la" type="date" value={filters.exit_at_from ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, exit_at_from: event.target.value }))} />
                  <InputText aria-label="Data ieșire până la" type="date" value={filters.exit_at_to ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, exit_at_to: event.target.value }))} />
                </div>
                <div className="flex flex-wrap justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={resetFilters}>Resetare</Button><Button disabled={!registryId} onClick={applyFilters}><Search /> Caută documente</Button></div>
              </div></Card.Content></Card.Body></Card.Root></div>}
            </div>
          </Card.Content>
        </Card.Body>
      </Card.Root>
      <Card.Root>
        <Card.Body>
          <Card.Content>
            {error && (
              <Message.Root severity="error">
                <Message.Content>
                  <Message.Text>{error}</Message.Text>
                </Message.Content>
              </Message.Root>
            )}
            {loading ? (
              <div className="flex justify-center p-8">
                <Spinner />
              </div>
            ) : documents.length === 0 ? (
              <Message.Root severity="info">
                <Message.Content>
                  <Message.Text>
                    Niciun document nu corespunde registrului și filtrelor
                    alese.
                  </Message.Text>
                </Message.Content>
              </Message.Root>
            ) : (
              <DataTable.Root
                data={documents as unknown as Record<string, unknown>[]}
                dataKey="id"
                scrollable
              >
                <DataTable.Table className="min-w-[80rem] table-fixed">
                  <DataTable.THead>
                    <DataTable.THeadRow>
                      <DataTable.THeadCell className="w-10"><span className="sr-only">Extindere</span></DataTable.THeadCell>
                      <DataTable.THeadCell className="w-28">{sortableHeader("Nr. Doc", "registry_number")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-24">{sortableHeader("Tip", "document_type")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-56">{sortableHeader("Conținut", "subject")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-36">{sortableHeader("Emitent", "correspondent")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-36">{sortableHeader("Destinatar", "assigned_to")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-28">{sortableHeader("Data intrare", "entry_at")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-28">{sortableHeader("Data ieșire", "exit_at")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-32">{sortableHeader("Status", "status")}</DataTable.THeadCell>
                      <DataTable.THeadCell className="w-40">Acțiuni</DataTable.THeadCell>
                    </DataTable.THeadRow>
                    <DataTable.THeadRow>
                      <DataTable.THeadCell />
                      <DataTable.THeadCell><InputText aria-label="Filtru coloană Nr. Doc" value={filters.registry_number ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, registry_number: event.target.value }))} onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => event.key === "Enter" && applyFilters()} /></DataTable.THeadCell>
                      <DataTable.THeadCell><Select.Root value={filters.document_type ?? ""} options={(filterOptions?.document_types ?? []).map((value) => ({ label: value, value }))} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => { setPage(1); setFilters((value) => ({ ...value, document_type: String(event.value ?? "") })); }}><Select.Trigger aria-label="Filtru coloană Tip"><Select.Value placeholder="Toate" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></DataTable.THeadCell>
                      <DataTable.THeadCell><InputText aria-label="Filtru coloană Conținut" value={filters.subject ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, subject: event.target.value }))} onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => event.key === "Enter" && applyFilters()} /></DataTable.THeadCell>
                      <DataTable.THeadCell><InputText aria-label="Filtru coloană Emitent" value={filters.correspondent ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, correspondent: event.target.value }))} onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => event.key === "Enter" && applyFilters()} /></DataTable.THeadCell>
                      <DataTable.THeadCell><InputText aria-label="Filtru coloană Destinatar" value={filters.assigned_to ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilters((value) => ({ ...value, assigned_to: event.target.value }))} onKeyDown={(event: KeyboardEvent<HTMLInputElement>) => event.key === "Enter" && applyFilters()} /></DataTable.THeadCell>
                      <DataTable.THeadCell><InputText aria-label="Filtru coloană Data intrare" type="date" value={filters.entry_at_from ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => { setPage(1); setFilters((value) => ({ ...value, entry_at_from: event.target.value })); }} /></DataTable.THeadCell>
                      <DataTable.THeadCell><InputText aria-label="Filtru coloană Data ieșire" type="date" value={filters.exit_at_from ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => { setPage(1); setFilters((value) => ({ ...value, exit_at_from: event.target.value })); }} /></DataTable.THeadCell>
                      <DataTable.THeadCell><Select.Root value={filters.status ?? ""} options={(filterOptions?.statuses ?? []).map((value) => ({ label: statusLabel(value), value }))} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => { setPage(1); setFilters((value) => ({ ...value, status: String(event.value ?? "") })); }}><Select.Trigger aria-label="Filtru coloană Status"><Select.Value placeholder="Toate" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></DataTable.THeadCell>
                      <DataTable.THeadCell><Button variant="text" severity="secondary" aria-label="Aplică filtrele din antet" onClick={applyFilters}><Search /></Button></DataTable.THeadCell>
                    </DataTable.THeadRow>
                  </DataTable.THead>
                  <DataTable.TBody>
                    {({ item, index }) => {
              if (!isRegistryDocument(item)) return null;
              const document = item;
                      const expanded = expandedDocuments[document.id];
                      return (<>
                        <DataTable.Row key={document.id} index={index}>
                          <DataTable.Cell className="w-10"><Button variant="text" severity="secondary" aria-label={`${expanded ? "Restrânge" : "Extinde"} ${document.registry_number}`} aria-expanded={Boolean(expanded)} onClick={() => void toggleExpanded(document)}>{expanded ? <ChevronDown /> : <ChevronRight />}</Button></DataTable.Cell>
                          <DataTable.Cell className="w-28 whitespace-nowrap font-medium">
                            {document.registry_number}
                          </DataTable.Cell>
                          <DataTable.Cell className="w-24 whitespace-nowrap font-medium">
                            {directionLabel(document)}
                          </DataTable.Cell>
                          <DataTable.Cell className="w-56"><span className="block truncate" title={document.subject}>{document.subject}</span></DataTable.Cell>
                          <DataTable.Cell className="w-36">
                            <span className="block truncate" title={document.correspondent}>{document.correspondent}</span>
                          </DataTable.Cell>
                          <DataTable.Cell className="w-36">
                            <span className="block truncate" title={document.assigned_to || undefined}>{document.assigned_to || "—"}</span>
                          </DataTable.Cell>
                          <DataTable.Cell className="w-28 whitespace-nowrap">
                            {calendarDateLabel(document.entry_at ?? (document.direction === "intrare" ? document.registered_at : null))}
                          </DataTable.Cell>
                          <DataTable.Cell className="w-28 whitespace-nowrap">
                            {calendarDateLabel(document.exit_at ?? (document.direction === "iesire" ? document.registered_at : null))}
                          </DataTable.Cell>
                          <DataTable.Cell className="w-32 whitespace-nowrap font-medium">
                            {statusLabel(document.status)}
                          </DataTable.Cell>
                          <DataTable.Cell className="w-40"><div className="flex flex-nowrap gap-1">
                            <Button variant="text" severity="info" aria-label={`Istoric ${document.registry_number}`} title="Istoric" onClick={() => void openDetail(document, "history")}><History /></Button>
                            <Button variant="text" aria-label={`Editează ${document.registry_number}`} title="Editare" disabled={!canManage || isTerminalStatus(document.status)} onClick={() => void openDetail(document, "edit")}><Pencil /></Button>
                            <Button variant="text" severity="danger" aria-label={`Anulează ${document.registry_number}`} title="Anulare" disabled={!canManage || isTerminalStatus(document.status)} onClick={() => void openDetail(document, "cancel")}><Ban /></Button>
                            <Button variant="text" severity="secondary" aria-label={`PDF ${document.registry_number}`} title="Tipărire PDF" onClick={() => api.print(document.id).then((blob) => saveBlob(blob, `${document.registry_number}.pdf`)).catch(() => setError("PDF-ul nu a putut fi generat."))}><FilePdf /></Button>
                            <Button variant="text" severity="warn" aria-label={`Flux ${document.registry_number}`} title={actionHint[document.id]} disabled={!canManageWorkflow} onClick={() => void openDetail(document, "workflow")}><ShareAlt /></Button>
                          </div></DataTable.Cell>
                        </DataTable.Row>
                        {expanded && <DataTable.Row key={`${document.id}-details`} index={index}><DataTable.Cell colSpan={10}><div className="grid gap-3 p-3 md:grid-cols-4"><span><strong>Compartimente:</strong> {expanded.department_names?.join(", ") || "Niciunul"}</span><span><strong>Nr. extern:</strong> {expanded.external_number || "—"}</span><span><strong>Data nr. extern:</strong> {calendarDateLabel(expanded.external_number_date)}</span><span><strong>Activitate:</strong> {expanded.activity || "—"}</span></div></DataTable.Cell></DataTable.Row>}
                      </>);
                    }}
                  </DataTable.TBody>
                </DataTable.Table>
              </DataTable.Root>
            )}
            {!loading && total > 0 && <div className="mt-3 flex flex-col items-center justify-between gap-3 md:flex-row"><span>Se afișează {(page - 1) * pageSize + 1} - {Math.min(page * pageSize, total)} din {total} documente</span><div className="flex flex-wrap items-center gap-1"><Button variant="outlined" severity="secondary" aria-label="Prima pagină" disabled={page <= 1} onClick={() => setPage(1)}>«</Button><Button variant="outlined" severity="secondary" aria-label="Pagina anterioară" disabled={page <= 1} onClick={() => setPage((value) => value - 1)}>‹</Button>{Array.from({ length: Math.min(5, Math.max(1, Math.ceil(total / pageSize))) }, (_, offset) => { const pages = Math.max(1, Math.ceil(total / pageSize)); const start = Math.min(Math.max(1, page - 2), Math.max(1, pages - 4)); const number = start + offset; return number <= pages ? <Button key={number} variant={number === page ? undefined : "outlined"} severity="secondary" aria-label={`Pagina ${number}`} aria-current={number === page ? "page" : undefined} onClick={() => setPage(number)}>{number}</Button> : null; })}<Button variant="outlined" severity="secondary" aria-label="Pagina următoare" disabled={page >= Math.ceil(total / pageSize)} onClick={() => setPage((value) => value + 1)}>›</Button><Button variant="outlined" severity="secondary" aria-label="Ultima pagină" disabled={page >= Math.ceil(total / pageSize)} onClick={() => setPage(Math.max(1, Math.ceil(total / pageSize)))}>»</Button><Select.Root value={pageSize} options={[10, 20, 50, 100].map((value) => ({ label: String(value), value }))} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => { setPage(1); setPageSize(Number(event.value)); }}><Select.Trigger aria-label="Rânduri pe pagină"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div></div>}
          </Card.Content>
        </Card.Body>
      </Card.Root>
      <Dialog.Root open={exportOpen} onOpenChange={(event: { value?: boolean }) => !event.value && setExportOpen(false)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Selectați intervalul de date pentru export PDF</Dialog.Title><Dialog.Close aria-label="Închide exportul"><Times /></Dialog.Close></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><InputText aria-label="Data început" type="date" value={exportStart} onChange={(event: ChangeEvent<HTMLInputElement>) => setExportStart(event.target.value)} /><InputText aria-label="Data sfârșit" type="date" value={exportEnd} onChange={(event: ChangeEvent<HTMLInputElement>) => setExportEnd(event.target.value)} /><span>Vor fi incluse documentele cu data înregistrării în intervalul selectat. Intervalul maxim este de 30 de zile.</span>{exportStart && exportEnd && !validExportRange(exportStart, exportEnd) && <Message.Root severity="error"><Message.Content><Message.Text>Intervalul trebuie să fie cronologic și să nu depășească 30 de zile.</Message.Text></Message.Content></Message.Root>}</div></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={() => setExportOpen(false)}>Anulează</Button><Button disabled={!registryId || !validExportRange(exportStart, exportEnd)} onClick={() => { if (!registryId) return; api.exportPdf({ registru_id: registryId, start_date: exportStart, end_date: exportEnd }).then((blob) => { saveBlob(blob, "registratura.pdf"); setExportOpen(false); }).catch(() => setError("Exportul PDF nu a putut fi generat.")); }}><FilePdf /> Generează PDF</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
      <Dialog.Root
        open={mode !== null}
        onOpenChange={(event: { value?: boolean }) =>
          !event.value && setMode(null)
        }
      >
        <Dialog.Portal>
          <Dialog.Backdrop />
          <Dialog.Positioner>
            <Dialog.Popup>
              <Dialog.Header>
                <Dialog.Title>
                  {mode === "intrare"
                    ? "Înregistrare intrare"
                    : mode === "iesire"
                      ? "Înregistrare ieșire"
                      : "Înregistrare MULTIPLU"}
                </Dialog.Title>
                <Dialog.Close aria-label="Închide">
                  <Times />
                </Dialog.Close>
              </Dialog.Header>
              <Dialog.Content>
                <div className="flex flex-col gap-3">
                  {mode === "multiplu" && <>
                    <span><strong>Registru:</strong> {registries.find((item) => item.id === registryId)?.nume ?? "—"}</span>
                    <InputText aria-label="Număr documente" type="number" min="1" max="20" value={count} onChange={(event: ChangeEvent<HTMLInputElement>) => setCount(event.target.value)} />
                    <Textarea aria-label="Conținut opțional" value={form?.subject ?? ""} placeholder='Dacă nu completați, se generează automat „Document multiplu 1/n”.' onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setForm((value) => value ? { ...value, subject: event.target.value } : value)} />
                    <InputText aria-label="Data intrare documente multiple" type="datetime-local" value={form?.entry_at ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, entry_at: event.target.value || null } : value)} />
                    <Message.Root severity="info"><Message.Content><Message.Text>Documentele sunt create cu tip MULTIPLU și pot fi transformate ulterior în Intrare sau Ieșire.</Message.Text></Message.Content></Message.Root>
                  </>}
                  {mode !== "multiplu" && <>
                  <Select.Root value={form?.record_kind ?? "document"} options={[{ label: "Document", value: "document" }, { label: "Dosar", value: "dosar" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setForm((value) => value ? { ...value, record_kind: String(event.value) === "dosar" ? "dosar" : "document" } : value)}><Select.Trigger aria-label="Document sau Dosar"><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>
                  <InputText
                    aria-label="Conținut"
                    value={form?.subject ?? ""}
                    placeholder="Conținut *"
                    onChange={(event: ChangeEvent<HTMLInputElement>) =>
                      setForm((value) =>
                        value
                          ? { ...value, subject: event.target.value }
                          : value,
                      )
                    }
                  />
                  <InputText
                    aria-label="Emitent"
                    value={form?.correspondent ?? ""}
                    placeholder="Emitent *"
                    onChange={(event: ChangeEvent<HTMLInputElement>) =>
                      setForm((value) =>
                        value
                          ? { ...value, correspondent: event.target.value }
                          : value,
                      )
                    }
                  />
                  <InputText aria-label="Destinatar" value={form?.assigned_to ?? ""} placeholder="Destinatar *" onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, assigned_to: event.target.value } : value)} />
                  <InputText
                    aria-label="Tip document"
                    value={form?.document_type ?? ""}
                    placeholder="Tip document"
                    onChange={(event: ChangeEvent<HTMLInputElement>) =>
                      setForm((value) =>
                        value
                          ? { ...value, document_type: event.target.value }
                          : value,
                      )
                    }
                  />
                  <div className="grid gap-2 md:grid-cols-2">
                    <InputText aria-label="Caută persoană sau organizație" value={partySearch} placeholder="Caută persoană / organizație" onChange={(event: ChangeEvent<HTMLInputElement>) => { const query = event.target.value; setPartySearch(query); void api.parties(query).then((result) => setPartyOptions(result.items)).catch(() => undefined); }} />
                    <Select.Root value={form?.correspondent_party_id ?? ""} options={partyOptions} optionLabel="display_name" optionValue="id" onValueChange={(event: { value: unknown }) => { const party = partyOptions.find((item) => item.id === String(event.value)); setForm((value) => value ? { ...value, correspondent_party_id: String(event.value) || null, correspondent: party?.display_name ?? value.correspondent } : value); }}><Select.Trigger><Select.Value placeholder="Selectează persoana" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2">
                    <InputText aria-label="Număr extern" value={form?.external_number ?? ""} placeholder="Număr extern" onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, external_number: event.target.value } : value)} />
                    <InputText aria-label="Data numărului extern" type="date" value={form?.external_number_date ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, external_number_date: event.target.value || null } : value)} />
                    {mode === "intrare" && <InputText aria-label="Data intrării" type="datetime-local" value={form?.entry_at ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, entry_at: event.target.value || null } : value)} />}
                    {mode === "iesire" && <InputText aria-label="Data ieșirii" type="datetime-local" value={form?.exit_at ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, exit_at: event.target.value || null } : value)} />}
                  </div>
                  <InputText aria-label="Activitate" value={form?.activity ?? ""} placeholder="Activitate" onChange={(event: ChangeEvent<HTMLInputElement>) => setForm((value) => value ? { ...value, activity: event.target.value || null } : value)} />
                  <div className="flex flex-col gap-2"><span>Compartimente responsabile</span><div className="flex flex-wrap gap-2">{departmentOptions.map((department) => { const id = String(department.id); const selectedDepartment = form?.department_ids?.includes(id) ?? false; return <Button key={id} type="button" variant={selectedDepartment ? undefined : "outlined"} severity="secondary" onClick={() => setForm((value) => value ? { ...value, department_ids: selectedDepartment ? (value.department_ids ?? []).filter((entry) => entry !== id) : [...(value.department_ids ?? []), id] } : value)}>{String(department.name ?? department.nume ?? department.code ?? id)}</Button>; })}</div></div>
                  <Textarea aria-label="Rezumat" value={form?.summary ?? ""} placeholder="Rezumat" onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setForm((value) => value ? { ...value, summary: event.target.value } : value)} />
                  <div className="flex flex-col gap-2"><span>Atașamente scanate (se încarcă după crearea documentului)</span><FileUpload.Root name="create-files" multiple maxFileSize={104857600} customUpload uploadHandler={(event: { files: File[] }) => setCreateAttachments(event.files)}><FileUpload.Trigger><Upload /> Alege fișiere</FileUpload.Trigger><FileUpload.Upload>Pregătește fișiere</FileUpload.Upload><FileUpload.Content /></FileUpload.Root></div>
                  </>}
                </div>
              </Dialog.Content>
              <Dialog.Footer>
                <div className="flex justify-end gap-2">
                  <Button
                    variant="outlined"
                    severity="secondary"
                    onClick={() => setMode(null)}
                  >
                    Renunță
                  </Button>
                  <Button
                    onClick={() => void submit()}
                    disabled={saving || !form || (mode !== "multiplu" && (!form.subject.trim() || !form.correspondent.trim() || !form.assigned_to.trim() || !(form.department_ids?.length))) || (mode === "multiplu" && (!Number.isInteger(Number(count)) || Number(count) < 1 || Number(count) > 20))}
                  >
                    {saving ? "Se salvează…" : "Salvează"}
                  </Button>
                </div>
              </Dialog.Footer>
            </Dialog.Popup>
          </Dialog.Positioner>
        </Dialog.Portal>
      </Dialog.Root>
      <Dialog.Root open={Boolean(selected)} onOpenChange={(event: { value?: boolean }) => !event.value && setSelected(undefined)}>
        <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup>
          <Dialog.Header><Dialog.Title>{documentView === "history" ? "Istoricul documentului" : documentView === "edit" ? "Editare document" : documentView === "cancel" ? "Anulare document" : documentView === "workflow" ? "Flux document" : "Detalii document"} {selected?.registry_number}</Dialog.Title><Dialog.Close aria-label="Închide"><Times /></Dialog.Close></Dialog.Header>
          <Dialog.Content>
            {detailLoading ? <div className="flex justify-center p-8"><Spinner /></div> : selected && <div className="flex flex-col gap-4">
              {detailError && <Message.Root severity="error"><Message.Content><Message.Text>{detailError}</Message.Text></Message.Content></Message.Root>}
              <div className="grid gap-2 md:grid-cols-2"><span><strong>Subiect:</strong> {selected.subject}</span><span><strong>Status:</strong> {statusLabel(selected.status)}</span><span><strong>Corespondent:</strong> {selected.correspondent || "—"}</span><span><strong>Destinatar:</strong> {selected.assigned_to || "—"}</span><span><strong>Confidențialitate:</strong> {selected.confidentiality ?? "normal"}</span><span><strong>Înregistrat:</strong> {selected.registered_at}</span></div>
              {documentView === "edit" && editing && editForm && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><strong>Editare document</strong><InputText aria-label="Subiect document" value={editForm.subject} onChange={(event: ChangeEvent<HTMLInputElement>) => setEditForm((value) => value && ({ ...value, subject: event.target.value }))} /><InputText aria-label="Corespondent document" value={editForm.correspondent} onChange={(event: ChangeEvent<HTMLInputElement>) => setEditForm((value) => value && ({ ...value, correspondent: event.target.value }))} />{selected.document_type.toUpperCase() === "MULTIPLU" && <Select.Root value={editForm.direction} options={[{ label: "Intrare", value: "intrare" }, { label: "Ieșire", value: "iesire" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setEditForm((value) => value && ({ ...value, direction: String(event.value) as "intrare" | "iesire", document_type: "DOCUMENT" }))}><Select.Trigger><Select.Value placeholder="Convertește MULTIPLU" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>}<div className="grid gap-2 md:grid-cols-2"><InputText aria-label="Număr extern document" value={editForm.external_number ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setEditForm((value) => value && ({ ...value, external_number: event.target.value }))} /><InputText aria-label="Data număr extern document" type="date" value={editForm.external_number_date ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setEditForm((value) => value && ({ ...value, external_number_date: event.target.value || null }))} /><InputText aria-label="Activitate document" value={editForm.activity ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setEditForm((value) => value && ({ ...value, activity: event.target.value }))} /><Select.Root value={editForm.record_kind ?? "document"} options={[{ label: "Document", value: "document" }, { label: "Dosar", value: "dosar" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setEditForm((value) => value && ({ ...value, record_kind: String(event.value) === "dosar" ? "dosar" : "document" }))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div><Textarea aria-label="Rezumat document" value={editForm.summary} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setEditForm((value) => value && ({ ...value, summary: event.target.value }))} /><Textarea aria-label="Notă modificare" value={editForm.change_notes ?? ""} placeholder="Notă modificare obligatorie" onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setEditForm((value) => value && ({ ...value, change_notes: event.target.value }))} /><div className="flex flex-wrap gap-2"><Button disabled={saving || !editForm.change_notes?.trim()} onClick={() => void saveEdit()}>{selected.document_type.toUpperCase() === "MULTIPLU" ? "Convertește și salvează" : "Salvează documentul"}</Button><Button variant="outlined" disabled={saving || !editForm.change_notes?.trim()} onClick={() => void createVersion()}>Creează versiune</Button></div></div></Card.Content></Card.Body></Card.Root>}
              {documentView === "cancel" && canManage && !isTerminalStatus(selected.status) && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><Message.Root severity="warn"><Message.Content><Message.Text>Anularea este ireversibilă. Documentul rămâne în istoric cu motivul introdus.</Message.Text></Message.Content></Message.Root><Textarea aria-label="Motiv anulare" value={cancelReason} placeholder="Motiv anulare obligatoriu (minim 10 caractere)" onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setCancelReason(event.target.value)} /><Button severity="danger" disabled={saving || cancelReason.trim().length < 10} onClick={() => void cancelDocument()}><Ban /> Anulează documentul</Button></div></Card.Content></Card.Body></Card.Root>}
              {documentView === "workflow" && canManageWorkflow && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><strong>Acțiuni flux</strong>{permittedActions(selected.status).length === 0 ? <span>Nu există tranziții disponibile în această stare.</span> : <div className="flex flex-wrap gap-2">{permittedActions(selected.status).map((action) => <Button key={action} variant="outlined" onClick={() => setWorkflowAction(action)}>{action}</Button>)}</div>}{workflowAction && <div className="flex flex-col gap-2"><span>Acțiune: {workflowAction}</span>{workflowAction === "assign_department" && <Select.Root value={workflowDepartment} options={assignees?.departments ?? []} optionLabel="name" optionValue="id" onValueChange={(event: { value: unknown }) => setWorkflowDepartment(String(event.value))}><Select.Trigger><Select.Value placeholder="Compartiment" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>}{workflowAction === "assign_user" && <Select.Root value={workflowUser} options={(assignees?.users ?? []).filter((user) => !workflowDepartment || user.department_ids?.includes(workflowDepartment))} optionLabel="name" optionValue="id" onValueChange={(event: { value: unknown }) => setWorkflowUser(String(event.value))}><Select.Trigger><Select.Value placeholder="Utilizator" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>}<Textarea aria-label="Notă flux" value={workflowNote} placeholder="Notă (opțional)" onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setWorkflowNote(event.target.value)} /><div className="flex gap-2"><Button disabled={saving} onClick={() => void runWorkflow()}><Users /> Aplică acțiunea</Button><Button variant="outlined" severity="secondary" onClick={() => setWorkflowAction(undefined)}>Renunță</Button></div></div>}</div></Card.Content></Card.Body></Card.Root>}
              {canReadLinks && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><strong>Legături documente</strong><span>Contextul sursă este obligatoriu și este verificat de backend pe tenant și vizibilitatea registrului.</span><div className="grid gap-2 md:grid-cols-3"><InputText aria-label="Modul sursă legătură" value={linkSourceModule} placeholder="Modul sursă (ex. education)" onChange={(event: ChangeEvent<HTMLInputElement>) => setLinkSourceModule(event.target.value)} /><InputText aria-label="ID înregistrare sursă legătură" value={linkSourceRecordId} placeholder="ID înregistrare sursă" onChange={(event: ChangeEvent<HTMLInputElement>) => setLinkSourceRecordId(event.target.value)} /><Select.Root value={linkRelationType} options={[{ label: "Principal", value: "primary" }, { label: "Suport", value: "supporting" }, { label: "Decizie", value: "decision" }, { label: "Bază arhivă", value: "archive_basis" }, { label: "Bază GDPR", value: "gdpr_basis" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setLinkRelationType(String(event.value))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div><div className="flex flex-wrap gap-2"><Button variant="outlined" severity="secondary" disabled={!linkSourceModule.trim() || !linkSourceRecordId.trim()} onClick={() => void loadLinks()}>Încarcă legături</Button>{canManageLinks && <Button disabled={saving || !linkSourceModule.trim() || !linkSourceRecordId.trim()} onClick={() => void createLink()}>Adaugă legătură</Button>}</div>{linkedDocuments.map((link) => <div className="flex flex-wrap items-center justify-between gap-2" key={link.link_id}><span>{link.registry_number} · {link.subject} · {link.relation_type}</span>{canManageLinks && <Button severity="danger" variant="text" aria-label={`Șterge legătura ${link.registry_number}`} disabled={saving} onClick={() => void deleteLink(link.link_id)}>Șterge</Button>}</div>)}</div></Card.Content></Card.Body></Card.Root>}
              {(documentView === "details" || documentView === "edit") && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><strong>Atașamente</strong>{attachments.map((attachment) => <div className="flex flex-wrap items-center justify-between gap-2" key={attachment.id}><span>{attachment.file_name} · {attachment.category} · {attachment.status}</span><Button variant="text" onClick={() => api.download(selected.id, attachment.id).then((blob) => saveBlob(blob, attachment.file_name)).catch(() => setDetailError("Descărcarea nu a reușit."))}>Descarcă</Button></div>)}{canManage && <FileUpload.Root name="file" multiple maxFileSize={104857600} customUpload uploadHandler={(event: { files: File[] }) => { void upload(event.files); }}><FileUpload.Trigger><Upload /> Alege fișiere</FileUpload.Trigger><FileUpload.Upload>Încarcă</FileUpload.Upload><FileUpload.Content /></FileUpload.Root>}</div></Card.Content></Card.Body></Card.Root>}
              {(documentView === "history" || documentView === "workflow" || documentView === "details") && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><strong>Istoric și versiuni</strong>{history.length === 0 && versions.length === 0 ? <span>Nu există istoric disponibil.</span> : <>{history.map((entry) => <span key={entry.id}>{entry.created_at}: {entry.action} → {statusLabel(entry.to_status)} {entry.actor_name ? `(${entry.actor_name})` : ""}</span>)}{versions.map((version) => <span key={version.id}>Versiunea {version.version_no}: {version.change_notes || "fără notă"}</span>)}</>}</div></Card.Content></Card.Body></Card.Root>}
            </div>}
          </Dialog.Content>
        </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
      </Dialog.Root>
      <Dialog.Root open={adminOpen} onOpenChange={(event: { value?: boolean }) => !event.value && setAdminOpen(false)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Administrare Registratură</Dialog.Title><Dialog.Close aria-label="Închide"><Times /></Dialog.Close></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Select.Root value={adminResource} options={[{ label: "Persoane", value: "parties" }, { label: "Compartimente", value: "departments" }, { label: "Organizații", value: "organizations" }, { label: "Registre", value: "registries" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => void loadAdmin(String(event.value) as typeof adminResource)}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><div className="flex gap-2"><InputText aria-label="Nume element" value={adminName} placeholder="Nume" onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminName(event.target.value)} /><Button disabled={saving || !adminName.trim()} onClick={() => void createAdmin()}>Adaugă</Button><Button variant="outlined" onClick={() => void loadChart()}>Organigramă</Button>{canReadAdminUsers && <Button variant="outlined" onClick={() => void openAssignments()}>Atribuiri utilizatori</Button>}</div>{canReadAdminUsers && adminUsers.length > 0 && <Select.Root value={assignmentUser?.id ?? ""} options={adminUsers} optionLabel="name" optionValue="id" onValueChange={(event: { value: unknown }) => void selectAssignmentUser(String(event.value))}><Select.Trigger><Select.Value placeholder="Alege utilizator" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>}{assignment && assignmentUser && <Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-2"><strong>Atribuiri: {assignmentUser.name}</strong>{(assignees?.departments ?? []).map((department) => { const selectedDepartment = assignment.department_ids.includes(department.id); return <Button key={department.id} variant={selectedDepartment ? undefined : "outlined"} onClick={() => setAssignment((current) => current && ({ ...current, department_ids: selectedDepartment ? current.department_ids.filter((id) => id !== department.id) : [...current.department_ids, department.id], primary_department_id: current.primary_department_id === department.id && selectedDepartment ? null : current.primary_department_id }))}>{department.name}</Button>; })}<Select.Root value={assignment.primary_department_id ?? ""} options={(assignees?.departments ?? []).filter((department) => assignment.department_ids.includes(department.id))} optionLabel="name" optionValue="id" onValueChange={(event: { value: unknown }) => setAssignment((current) => current && ({ ...current, primary_department_id: String(event.value) || null }))}><Select.Trigger><Select.Value placeholder="Compartiment principal" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>{canManage && <Button disabled={saving} onClick={() => void saveAssignments()}>Salvează atribuiri</Button>}</div></Card.Content></Card.Body></Card.Root>}{chart.length > 0 && <ChartTree nodes={chart} />}{adminItems.map((item) => <div className="flex items-center justify-between gap-2" key={String(item.id)}><span>{"display_name" in item ? String(item.display_name) : String(item.nume ?? item.name ?? item.code ?? item.id)}</span><div className="flex gap-1"><Button variant="text" aria-label={`Editează ${"display_name" in item ? String(item.display_name) : String(item.nume ?? item.name ?? item.code ?? item.id)}`} onClick={() => editAdmin(item, adminResource)}><Pencil /> Editează</Button><Button severity="danger" variant="text" onClick={() => setPendingAdminDeletion({ item, resource: adminResource })}>Șterge</Button></div></div>)}</div></Dialog.Content></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
      <Dialog.Root open={Boolean(adminEditing)} onOpenChange={(event: { value?: boolean }) => !event.value && setAdminEditing(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Editează {adminEditing?.resource === "parties" ? "persoana / organizația" : adminEditing?.resource === "registries" ? "registrul" : adminEditing?.resource === "departments" ? "compartimentul" : "organizația"}</Dialog.Title><Dialog.Close aria-label="Închide editarea" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">{adminEditing?.resource === "parties" ? <><Select.Root value={String(adminDraft.party_type ?? "physical")} options={[{ label: "Persoană fizică", value: "physical" }, { label: "Persoană juridică", value: "legal" }, { label: "Instituție", value: "institution" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setAdminDraft((value) => ({ ...value, party_type: String(event.value) }))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><InputText aria-label="Nume afișat" value={String(adminDraft.display_name ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, display_name: event.target.value }))} /><div className="grid gap-2 md:grid-cols-2"><InputText aria-label="Prenume" value={String(adminDraft.first_name ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, first_name: event.target.value }))} /><InputText aria-label="Nume familie" value={String(adminDraft.last_name ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, last_name: event.target.value }))} /><InputText aria-label="CUI / identificator" value={String(adminDraft.identifier_code ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, identifier_code: event.target.value }))} /><InputText aria-label="Email" type="email" value={String(adminDraft.email ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, email: event.target.value }))} /><InputText aria-label="Telefon" value={String(adminDraft.phone_number ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, phone_number: event.target.value }))} /><InputText aria-label="Adresă" value={String(adminDraft.address_line1 ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, address_line1: event.target.value }))} /></div><Textarea aria-label="Note persoană" value={String(adminDraft.notes ?? "")} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setAdminDraft((value) => ({ ...value, notes: event.target.value }))} /></> : <><InputText aria-label="Nume" value={String(adminDraft.name ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, name: event.target.value }))} />{adminEditing?.resource === "registries" && <div className="grid gap-2 md:grid-cols-2"><InputText aria-label="Prefix" value={String(adminDraft.prefix ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, prefix: event.target.value }))} /><InputText aria-label="Număr început" type="number" value={String(adminDraft.start_number ?? 1)} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, start_number: Number(event.target.value) || 1 }))} /><Select.Root value={String(adminDraft.registry_type ?? "public")} options={[{ label: "Public", value: "public" }, { label: "Privat", value: "private" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setAdminDraft((value) => ({ ...value, registry_type: String(event.value) }))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div>}{adminEditing?.resource !== "registries" && <Textarea aria-label="Descriere" value={String(adminDraft.description ?? "")} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setAdminDraft((value) => ({ ...value, description: event.target.value }))} />}{adminEditing?.resource === "departments" && <InputText aria-label="Rol compartiment" value={String(adminDraft.role_tag ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, role_tag: event.target.value }))} />}</>}</div></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={() => setAdminEditing(undefined)}>Renunță</Button><Button disabled={saving || !String(adminDraft.name ?? adminDraft.display_name ?? "").trim()} onClick={() => void saveAdminEdit()}>{saving ? "Se salvează…" : "Salvează"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
      <Dialog.Root open={Boolean(specialistEditing)} onOpenChange={(event: { value?: boolean }) => !event.value && setSpecialistEditing(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Editează configurarea Registratură</Dialog.Title><Dialog.Close aria-label="Închide editarea avansată" /></Dialog.Header><Dialog.Content>{specialistEditing && <div className="flex flex-col gap-3">{specialistEditing.resource === "parties" ? <><Select.Root value={String(adminDraft.party_type ?? "physical")} options={[{ label: "Persoană fizică", value: "physical" }, { label: "Persoană juridică", value: "legal" }, { label: "Instituție", value: "institution" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setAdminDraft((value) => ({ ...value, party_type: String(event.value) }))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><div className="grid gap-2 md:grid-cols-2">{([['display_name', 'Nume afișat'], ['short_name', 'Nume scurt'], ['first_name', 'Prenume'], ['last_name', 'Nume familie'], ['legal_name', 'Denumire legală'], ['code', 'Cod intern'], ['identifier_code', 'CUI / identificator'], ['tax_id', 'Cod fiscal'], ['email', 'Email'], ['phone_number', 'Telefon'], ['address_line1', 'Adresă'], ['address_line2', 'Adresă suplimentară'], ['locality', 'Localitate'], ['county', 'Județ'], ['country', 'Țară'], ['birth_place', 'Loc naștere'], ['trade_register_number', 'Nr. registru comerțului'], ['legal_representative', 'Reprezentant legal'], ['legal_form', 'Formă juridică'], ['institution_type', 'Tip instituție'], ['website', 'Website']] as const).map(([key, label]) => <InputText key={key} aria-label={label} value={String(adminDraft[key] ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, [key]: event.target.value }))} />)}<InputText aria-label="Data nașterii" type="date" value={String(adminDraft.birth_date ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, birth_date: event.target.value || null }))} /></div><Textarea aria-label="Note persoană" value={String(adminDraft.notes ?? "")} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setAdminDraft((value) => ({ ...value, notes: event.target.value }))} /></> : <><InputText aria-label="Nume" value={String(adminDraft.name ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, name: event.target.value }))} />{specialistEditing.resource === "registries" ? <div className="grid gap-2 md:grid-cols-2"><InputText aria-label="Prefix" value={String(adminDraft.prefix ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, prefix: event.target.value }))} /><InputText aria-label="Număr început" type="number" value={String(adminDraft.start_number ?? 1)} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, start_number: Number(event.target.value) || 1 }))} /><Select.Root value={String(adminDraft.registry_type ?? "public")} options={[{ label: "Public", value: "public" }, { label: "Privat", value: "private" }]} optionLabel="label" optionValue="value" onValueChange={(event: { value: unknown }) => setAdminDraft((value) => ({ ...value, registry_type: String(event.value) }))}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></div> : <Textarea aria-label="Descriere" value={String(adminDraft.description ?? "")} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setAdminDraft((value) => ({ ...value, description: event.target.value }))} />}{specialistEditing.resource === "departments" && <><InputText aria-label="Rol compartiment" value={String(adminDraft.role_tag ?? "")} onChange={(event: ChangeEvent<HTMLInputElement>) => setAdminDraft((value) => ({ ...value, role_tag: event.target.value }))} /><Select.Root value={String(adminDraft.parent_id ?? "")} options={departmentOptions.filter((item) => String(item.id) !== String(specialistEditing.item.id))} optionLabel="name" optionValue="id" onValueChange={(event: { value: unknown }) => setAdminDraft((value) => ({ ...value, parent_id: String(event.value) || null }))}><Select.Trigger><Select.Value placeholder="Compartiment părinte" /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></>}{specialistEditing.resource !== "departments" && <div className="flex flex-col gap-2"><span>Compartimente asociate</span><div className="flex flex-wrap gap-2">{departmentOptions.map((department) => { const id = String(department.id); const checked = ((adminDraft.department_ids as string[] | undefined) ?? []).includes(id); return <Button key={id} type="button" variant={checked ? undefined : "outlined"} severity="secondary" onClick={() => setAdminDraft((value) => ({ ...value, department_ids: checked ? ((value.department_ids as string[] | undefined) ?? []).filter((entry) => entry !== id) : [...((value.department_ids as string[] | undefined) ?? []), id] }))}>{String(department.name ?? department.nume ?? id)}</Button>; })}</div></div>}</>}</div>}</Dialog.Content><Dialog.Footer><div className="flex flex-wrap justify-end gap-2">{specialistEditing?.resource === "parties" || specialistEditing?.resource === "departments" || specialistEditing?.resource === "organizations" ? <Button type="button" variant={adminDraft.active === false ? "outlined" : undefined} severity="secondary" onClick={() => setAdminDraft((value) => ({ ...value, active: value.active === false }))}>{adminDraft.active === false ? "Inactive" : "Activă"}</Button> : null}{specialistEditing?.resource === "parties" || specialistEditing?.resource === "organizations" || specialistEditing?.resource === "registries" ? <Button type="button" variant={adminDraft.is_default || adminDraft.is_default_organization ? undefined : "outlined"} severity="secondary" onClick={() => setAdminDraft((value) => specialistEditing.resource === "parties" ? ({ ...value, is_default_organization: !value.is_default_organization }) : ({ ...value, is_default: !value.is_default }))}>{adminDraft.is_default || adminDraft.is_default_organization ? "Implicită" : "Marchează implicită"}</Button> : null}<Button variant="outlined" severity="secondary" onClick={() => setSpecialistEditing(undefined)}>Renunță</Button><Button disabled={saving || !String(adminDraft.name ?? adminDraft.display_name ?? "").trim()} onClick={() => void saveSpecialistEdit()}>{saving ? "Se salvează…" : "Salvează"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
      <Dialog.Root open={Boolean(pendingAdminDeletion)} onOpenChange={(event: { value?: boolean }) => !event.value && !saving && setPendingAdminDeletion(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup aria-describedby="delete-admin-description"><Dialog.Header><Dialog.Title>Confirmați ștergerea</Dialog.Title><Dialog.Close aria-label="Închide confirmarea" /></Dialog.Header><Dialog.Content><p id="delete-admin-description">Elementul selectat va fi șters definitiv. Această acțiune nu poate fi anulată.</p></Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" disabled={saving} onClick={() => setPendingAdminDeletion(undefined)}>Renunță</Button><Button severity="danger" disabled={saving} onClick={() => void deleteAdminItem()}>{saving ? "Se șterge…" : "Șterge definitiv"}</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
    </section>
  );
}
