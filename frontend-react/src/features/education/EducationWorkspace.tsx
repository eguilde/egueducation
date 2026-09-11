import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Textarea } from "@primereact/ui/textarea";
import { Checkbox } from "@primereact/ui/checkbox";
import { Message } from "@primereact/ui/message";
import { Popover } from "@primereact/ui/popover";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select } from "@primereact/ui/select";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import {
  educationPermissionImplies,
  useAuth,
  type EducationDelegationGrant,
} from "../../auth/AuthProvider";
import type { ContractClient } from "../../api/client";
import { createEducationApi, type AuthenticatedFetcher } from "./api";
import { educationAreas, visibleEducationAreas } from "./catalog";
import { PortfolioArchiveGrantManager } from "./PortfolioArchiveGrantManager";
import {
  createEducationDelegationApi,
  EducationDelegationManager,
  type EducationDelegationApi,
} from "./EducationDelegationManager";
import {
  createIntertenantPortfolioTransferApi,
  PortfolioIntertenantTransfer,
  type IntertenantPortfolioTransferApi,
} from "./PortfolioIntertenantTransfer";
import { PortfolioValorificationPackageManager } from "./PortfolioValorificationPackageManager";
import { PortfolioDocumentVersionHistoryDialog } from "./PortfolioDocumentVersionHistoryDialog";
import { PortfolioProcedureManager, type PortfolioProcedureApi, type PortfolioProcedure as ProcedureView, type PortfolioProcedureRule as ProcedureRuleView } from "./PortfolioProcedureManager";
import { RoleCockpit, type RoleCockpitKind } from "./RoleCockpits";
import { createRoleCockpitsApi, roleCockpitLoader } from "./role-cockpits-api";
import type {
  EducationApi,
  DirectorCockpit,
  EducationArea,
  EducationModule,
  EducationMetadataResource,
  EducationPage,
  EducationPdfRecordsDomain,
  EducationRecord,
  EducationRecordInput,
  EducationRelatedResource,
  EducationRecordsDomain,
  EducationRootCreateInputByDomain,
  EducationRootUpdateInputByDomain,
  EducationRequirement,
  CreatePortfolioChecklistItemInput,
  CreatePortfolioCustodyEventInput,
  CreatePortfolioDocumentInput,
  CreatePortfolioOpisEntryInput,
  CreatePortfolioReviewEventInput,
  GovernanceMeeting,
  GovernanceMeetingInput,
  PortfolioSection,
} from "./types";

const Spinner = () => (
  <ProgressSpinner.Root>
    <ProgressSpinner.Range>
      <ProgressSpinner.Track />
      <ProgressSpinner.Value />
    </ProgressSpinner.Range>
  </ProgressSpinner.Root>
);

export function educationPermissionAllows(
  directPermissions: readonly string[],
  activeDelegations: ReadonlyArray<Pick<EducationDelegationGrant, "permission_code" | "resource_type" | "resource_id">>,
  permission: string,
  resourceType = "institution",
  resourceID?: string,
) {
  if (directPermissions.some((granted) => educationPermissionImplies(granted, permission))) return true;
  return activeDelegations.some((item) =>
    educationPermissionImplies(item.permission_code, permission) &&
    (resourceType === "institution"
      ? item.resource_type === "institution"
      : item.resource_type === resourceType && Boolean(resourceID) && item.resource_id === resourceID),
  );
}

export function effectiveEducationPermissions(
  grantedPermissions: readonly string[],
): string[] {
  const navigationPermissions = educationAreas.flatMap((area) => area.permissions);
  return [...new Set([
    ...grantedPermissions,
    ...navigationPermissions.filter((requested) =>
      grantedPermissions.some((granted) => educationPermissionImplies(granted, requested)),
    ),
  ])];
}

const procedureView = (item: import("./types").PortfolioProcedure): ProcedureView => ({
  id: item.id, code: item.procedure_code, title: item.title, description: item.source_ref,
  status: item.lifecycle_status, version: item.version_no, updated_at: item.updated_at,
});
const procedureRuleView = (item: import("./types").PortfolioProcedureRule): ProcedureRuleView => ({
  id: item.id, legal_section_code: item.section_code, label: item.label_ro,
  label_en: item.label_en, source_catalog_version: item.source_catalog_version, active: item.active,
  required: item.required, sort_order: item.sort_order,
});
function portfolioProcedureAdapter(api: EducationApi): PortfolioProcedureApi {
  return {
    list: async (query) => { const result = await api.portfolioProcedures(query); return { ...result, items: result.items.map(procedureView) }; },
    detail: async (id) => procedureView(await api.portfolioProcedure(id)),
    create: async (input) => procedureView(await api.createPortfolioProcedure({ procedure_code: input.code, title: input.title, source_ref: input.description })),
    update: async (id, input) => {
      const current = await api.portfolioProcedure(id);
      return procedureView(await api.updatePortfolioProcedure(id, { procedure_code: input.code, title: input.title, source_ref: input.description ?? current.source_ref, effective_from: current.effective_from, effective_to: current.effective_to, calendar_rules: current.calendar_rules, access_rules: current.access_rules, accepted_formats: current.accepted_formats, retention_rules: current.retention_rules, transfer_rules: current.transfer_rules, expected_updated_at: input.expected_updated_at ?? current.updated_at }));
    },
    rules: async (id, query) => {
      const result = await api.portfolioProcedureRules(id, query);
      return { ...result, items: result.items.map(procedureRuleView) };
    },
    replaceRules: async (id, input) => api.replacePortfolioProcedureRules(id, {
      expected_updated_at: input.expected_updated_at,
      rules: input.rules.map((rule) => ({
        id: rule.id ?? globalThis.crypto.randomUUID(), procedure_id: id,
        section_code: rule.legal_section_code, label_ro: rule.label ?? "", label_en: rule.label_en ?? "",
        source_catalog_version: rule.source_catalog_version ?? "", required: rule.required,
        sort_order: rule.sort_order ?? 0, active: rule.active ?? true,
      })),
    }),
    transition: async (id, input) => procedureView(await api.transitionPortfolioProcedure(id, input.transition, { expected_updated_at: input.expected_updated_at, evidence: { reference: input.evidence } })),
  };
}

type SchoolRowAction = {
  label: string;
  icon: string;
  onSelect: () => void;
  severity?: "secondary" | "danger" | "warn" | "success";
  disabled?: boolean;
};

/** A compact, accessible action menu shared by School registry tables. */
export function SchoolRowActionMenu({ actions }: { actions: SchoolRowAction[] }) {
  const [open, setOpen] = useState(false);

  const selectAction = (action: SchoolRowAction) => {
    // Close the portalled overlay before the selected action mounts a dialog
    // or refreshes the table. Keeping both mounted lets focus management race
    // the row re-render and can leave the action button detached mid-click.
    setOpen(false);
    // Let the originating pointer/focus event and Popover teardown complete
    // before a selected action mounts another portalled overlay. Otherwise the
    // Popover's outside-interaction cleanup also dismisses the new Dialog.
    window.requestAnimationFrame(action.onSelect);
  };

  return (
    <Popover.Root
      open={open}
      onOpenChange={(event: { value?: boolean }) =>
        setOpen(Boolean(event.value))
      }
    >
      <Popover.Trigger
        as={Button}
        iconOnly
        rounded
        size="small"
        variant="text"
        aria-label="Acțiuni înregistrare"
        title="Acțiuni înregistrare"
      >
        <i className="pi pi-ellipsis-v" aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Positioner side="left" align="start" sideOffset={6}>
          <Popover.Popup>
            <Popover.Content>
              <div className="flex min-w-40 flex-col gap-1" role="menu">
                {actions.map((action) => (
                  <Button
                    key={action.label}
                    size="small"
                    variant="text"
                    severity={action.severity}
                    disabled={action.disabled}
                    aria-label={action.label}
                    title={action.label}
                    onClick={() => selectAction(action)}
                  >
                    <i className={action.icon} aria-hidden="true" />
                    {action.label}
                  </Button>
                ))}
              </div>
            </Popover.Content>
          </Popover.Popup>
        </Popover.Positioner>
      </Popover.Portal>
    </Popover.Root>
  );
}

export interface EducationListPanelProps<T extends { id: string }> {
  title: string;
  description: string;
  load: (
    query: string,
    page?: number,
    pageSize?: number,
    sort?: { field?: string; direction?: "asc" | "desc" },
    filters?: Record<string, string>,
  ) => Promise<EducationPage<T>>;
  columns: Array<{
    field?: string;
    header: string;
    render: (item: T) => ReactNode;
    /** Keep operational controls pinned on narrow, horizontally-scrolled tables. */
    action?: boolean;
  }>;
  emptyMessage: string;
  onAdd?: () => void;
  addLabel?: string;
  /** Only expose documented server-side header filters. Defaults preserve existing registry contracts. */
  filterableFields?: readonly string[];
  /** Only expose sorting where the endpoint contract accepts it. Defaults preserve existing registry contracts. */
  sortableFields?: readonly string[];
}

/** Reusable authenticated list state for all paginated Education resources. */
export function EducationListPanel<T extends { id: string }>({
  title,
  description,
  load,
  columns,
  emptyMessage,
  onAdd,
  addLabel = "înregistrare",
  filterableFields,
  sortableFields,
}: EducationListPanelProps<T>) {
  // Backend list contracts expose documented field filters; `q` is not a
  // supported Education query parameter, so never offer a misleading global
  // search control here.
  const query = "";
  const [pageNumber, setPageNumber] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [sort, setSort] = useState<{
    field?: string;
    direction?: "asc" | "desc";
  }>({});
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [page, setPage] = useState<EducationPage<T>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 50,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const requestSequence = useRef(0);
  const filterEffectReady = useRef(false);
  const refresh = async (
    nextQuery = query,
    nextPage = pageNumber,
    nextPageSize = pageSize,
    nextSort = sort,
    nextFilters = filters,
  ) => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setError(undefined);
    try {
      const result = await load(
        nextQuery,
        nextPage,
        nextPageSize,
        nextSort,
        nextFilters,
      );
      if (sequence === requestSequence.current) {
        setPage(result);
        setPageNumber(result.page ?? nextPage);
        setPageSize(result.pageSize ?? nextPageSize);
      }
    } catch {
      if (sequence === requestSequence.current)
        setError("Datele nu au putut fi încărcate. Încercați din nou.");
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  };
  useEffect(() => {
    void refresh("");
  }, [load]); // load is stable in each resource page.
  // Header filters are server-side. Debouncing prevents a request per keypress
  // while preserving the backend as the source of truth for result sets.
  useEffect(() => {
    if (!filterEffectReady.current) {
      filterEffectReady.current = true;
      return;
    }
    const timer = window.setTimeout(() => {
      setPageNumber(1);
      void refresh(query, 1, pageSize, sort, filters);
    }, 350);
    return () => window.clearTimeout(timer);
  }, [query, filters]);

  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>{title}</Card.Title>
        <Card.Content>
          <div className="flex flex-col gap-4">
            <p>{description}</p>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                variant="outlined"
                severity="secondary"
                disabled={loading || Object.keys(filters).length === 0}
                onClick={() => {
                  setFilters({});
                }}
              >
                Resetează filtrele
              </Button>
            </div>
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
            ) : page.items.length === 0 && !onAdd ? (
              <Message.Root severity="info">
                <Message.Content>
                  <Message.Text>{emptyMessage}</Message.Text>
                </Message.Content>
              </Message.Root>
            ) : (
              <DataTable.Root
                data={page.items as unknown as Record<string, unknown>[]}
                dataKey="id"
                scrollable
                className="max-h-[calc(100dvh-20rem)] min-h-72 overflow-auto"
              >
                <DataTable.Table>
                  <DataTable.THead className="sticky top-0 z-10">
                    <DataTable.THeadRow>
                      {columns.map((column) => (
                        <DataTable.THeadCell
                          key={column.header}
                          frozen={column.action || undefined}
                          alignFrozen={column.action ? "right" : undefined}
                        >
                          {column.field && (sortableFields === undefined || sortableFields.includes(column.field)) ? (
                            <Button
                              variant="text"
                              size="small"
                              onClick={() => {
                                const field = column.field;
                                const direction =
                                  sort.field === field &&
                                  sort.direction === "asc"
                                    ? "desc"
                                    : "asc";
                                setSort({ field, direction });
                                setPageNumber(1);
                                void refresh(
                                  query,
                                  1,
                                  pageSize,
                                  { field, direction },
                                  filters,
                                );
                              }}
                            >
                              {column.header}
                              {sort.field === column.field
                                ? sort.direction === "asc"
                                  ? " ↑"
                                  : " ↓"
                                : ""}
                            </Button>
                          ) : column.action && onAdd ? (
                            <span className="flex items-center justify-between gap-2">
                              <span>{column.header}</span>
                              <Button
                                iconOnly
                                rounded
                                size="small"
                                aria-label={`Adaugă ${addLabel}`}
                                title={`Adaugă ${addLabel}`}
                                onClick={onAdd}
                              >
                                <i className="pi pi-plus" aria-hidden="true" />
                              </Button>
                            </span>
                          ) : (
                            <span>{column.header}</span>
                          )}
                          {column.field && (filterableFields === undefined || filterableFields.includes(column.field)) && (
                            <InputText
                              aria-label={`Filtru ${column.header}`}
                              className="mt-1 w-full"
                              placeholder="Filtru"
                              value={filters[column.field] ?? ""}
                              onChange={(
                                event: ChangeEvent<HTMLInputElement>,
                              ) =>
                                setFilters((current) => ({
                                  ...current,
                                  [column.field as string]: event.target.value,
                                }))
                              }
                            />
                          )}
                        </DataTable.THeadCell>
                      ))}
                    </DataTable.THeadRow>
                  </DataTable.THead>
                  <DataTable.TBody>
                    {({ item, index }) => {
                      const row = item as T;
                      return (
                        <DataTable.Row key={row.id} index={index}>
                          {columns.map((column) => (
                            <DataTable.Cell
                              key={column.header}
                              frozen={column.action || undefined}
                              alignFrozen={column.action ? "right" : undefined}
                            >
                              {column.render(row)}
                            </DataTable.Cell>
                          ))}
                        </DataTable.Row>
                      );
                    }}
                  </DataTable.TBody>
                </DataTable.Table>
              </DataTable.Root>
            )}
            {!loading && page.items.length === 0 && onAdd && (
              <Message.Root severity="info">
                <Message.Content>
                  <Message.Text>{emptyMessage}</Message.Text>
                </Message.Content>
              </Message.Root>
            )}
            <div
              className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2"
              aria-label="Paginare"
            >
              <span>
                {page.total
                  ? `${(pageNumber - 1) * pageSize + 1} - ${Math.min(pageNumber * pageSize, page.total)} din ${page.total}`
                  : "0 rezultate"}
              </span>
              <div className="flex items-center gap-2">
                <Select.Root
                  value={pageSize}
                  options={[10, 20, 50, 100].map((value) => ({
                    label: String(value),
                    value,
                  }))}
                  optionLabel="label"
                  optionValue="value"
                  onValueChange={(event: SelectValueChangeEvent) => {
                    const next = Number(event.value);
                    setPageSize(next);
                    setPageNumber(1);
                    void refresh(query, 1, next, sort, filters);
                  }}
                >
                  <Select.Trigger aria-label="Rânduri pe pagină">
                    <Select.Value />
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
                <Button
                  size="small"
                  variant="outlined"
                  disabled={pageNumber <= 1 || loading}
                  onClick={() => {
                    const next = pageNumber - 1;
                    setPageNumber(next);
                    void refresh(query, next, pageSize, sort, filters);
                  }}
                >
                  Anterior
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  disabled={pageNumber * pageSize >= page.total || loading}
                  onClick={() => {
                    const next = pageNumber + 1;
                    setPageNumber(next);
                    void refresh(query, next, pageSize, sort, filters);
                  }}
                >
                  Următor
                </Button>
              </div>
            </div>
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}

const governanceFields: RecordField[] = [
  { key: "school_year", label: "An școlar" },
  { key: "organism", label: "Organism" },
  { key: "title", label: "Titlu" },
  { key: "meeting_type", label: "Tip ședință" },
  { key: "status", label: "Stare" },
  { key: "quorum_required", label: "Cvorum necesar", kind: "number" },
  { key: "participants_count", label: "Participanți", kind: "number" },
  { key: "meeting_date", label: "Data", kind: "date" },
  { key: "location", label: "Loc" },
  { key: "chairperson_user_id", label: "Președinte", kind: "select" },
  { key: "secretary_user_id", label: "Secretar", kind: "select" },
  { key: "summary", label: "Rezumat" },
];
function GovernanceMeetingsPage({
  api,
  canManage,
  canManageMeeting,
}: {
  api: EducationApi;
  canManage: boolean;
  canManageMeeting: (meetingID: string) => boolean;
}) {
  const navigate = useNavigate();
  const [editing, setEditing] = useState<{
    id?: string;
    input: EducationRecordInput;
  }>();
  const [detail, setDetail] = useState<EducationRecord>();
  const [selectedRelatedId, setSelectedRelatedId] = useState<string>();
  const [error, setError] = useState<string>();
  const [refresh, setRefresh] = useState(0);
  const [pendingDelete, setPendingDelete] = useState<string>();
  const [selectedMeetingId, setSelectedMeetingId] = useState<string>();
  const [eligibleUsers, setEligibleUsers] = useState<
    Array<{ id: string; name: string }>
  >([]);
  const [eligibleUserQuery, setEligibleUserQuery] = useState("");
  const [eligibleUsersLoading, setEligibleUsersLoading] = useState(true);
  useEffect(() => {
    let current = true;
    setEligibleUsersLoading(true);
    const timer = window.setTimeout(() => void api
      .eligibleGovernanceUsers({ q: eligibleUserQuery, page: 1, pageSize: 100 })
      .then((items) => current && setEligibleUsers(items))
      .catch(() =>
        current && setError("Utilizatorii eligibili nu au putut fi încărcați."),
      )
      .finally(() => current && setEligibleUsersLoading(false)), 250);
    return () => {
      current = false;
      window.clearTimeout(timer);
    };
  }, [api, eligibleUserQuery]);
  const meetingFields = useMemo(
    () =>
      governanceFields.map((field) =>
        field.kind === "select"
          ? {
              ...field,
              options: eligibleUsers.map((user) => ({
                value: user.id,
                label: user.name,
              })),
            }
          : field,
      ),
    [eligibleUsers],
  );
  const load = useMemo(
    () =>
      (
        q: string,
        page = 1,
        pageSize = 20,
        sort?: { field?: string; direction?: "asc" | "desc" },
        filters?: Record<string, string>,
      ) =>
        api.governanceMeetings({
          q,
          page,
          pageSize,
          sort: sort?.field,
          direction: sort?.direction,
          filters,
        }),
    [api, refresh],
  );
  const action = async (fn: () => Promise<void>) => {
    setError(undefined);
    try {
      await fn();
      setEditing(undefined);
      setDetail(undefined);
      setRefresh((value) => value + 1);
    } catch {
      setError("Operația pentru ședință nu a putut fi finalizată.");
    }
  };
  return (
    <div className="flex flex-col gap-3">
      {error && (
        <Message.Root severity="error">
          <Message.Content>
            <Message.Text>{error}</Message.Text>
          </Message.Content>
        </Message.Root>
      )}
      {canManage && (
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outlined"
            onClick={() => navigate("/scoala/governance/ca-wizard")}
          >
            Ghid ședință CA/CP
          </Button>
          {selectedMeetingId && (
            <>
              <Button
                variant="outlined"
                onClick={() =>
                  navigate(
                    `/scoala/governance/minutes-wizard?meetingId=${encodeURIComponent(selectedMeetingId)}`,
                  )
                }
              >
                Ghid minută
              </Button>
              <Button
                variant="outlined"
                onClick={() =>
                  navigate(
                    `/scoala/governance/votes-wizard?meetingId=${encodeURIComponent(selectedMeetingId)}`,
                  )
                }
              >
                Ghid vot
              </Button>
              <Button
                variant="outlined"
                onClick={() =>
                  navigate(
                    `/scoala/governance/resolutions-wizard?meetingId=${encodeURIComponent(selectedMeetingId)}`,
                  )
                }
              >
                Ghid hotărâre
              </Button>
            </>
          )}
        </div>
      )}
      <EducationListPanel<GovernanceMeeting>
        title="Ședințe de guvernanță"
        description="Ședințe și starea lor curentă pentru instituția selectată."
        load={load}
        emptyMessage="Nu există ședințe care corespund filtrului ales."
        columns={[
          { field: "title", header: "Titlu", render: (item) => item.title },
          {
            field: "organism",
            header: "Organism",
            render: (item) => item.organism,
          },
          {
            field: "meeting_date",
            header: "Data",
            render: (item) => item.meeting_date,
          },
          {
            field: "chairperson",
            header: "Președinte",
            render: (item) => item.chairperson || "—",
          },
          {
            field: "status",
            header: "Status",
            render: (item) => <Tag value={item.status} />,
          },
          {
            header: "Acțiuni",
            action: true,
            render: (item) => (
              <SchoolRowActionMenu
                actions={[
                  {
                    label: "Detalii",
                    icon: "pi pi-eye",
                    onSelect: () =>
                    void api
                      .governanceMeetingDetail(item.id)
                      .then((value) => {
                        setDetail(value as unknown as EducationRecord);
                        setSelectedMeetingId(item.id);
                      })
                      .catch(() =>
                        setError("Detaliul ședinței nu a putut fi încărcat."),
                      ),
                  },
                  ...(canManage
                    ? [{
                        label: "Editează",
                        icon: "pi pi-pencil",
                        onSelect: () =>
                      setEditing({
                        id: item.id,
                        input: inputFromRecord(
                          item as unknown as EducationRecord,
                          governanceFields,
                        ),
                      }),
                      }, {
                        label: "Șterge",
                        icon: "pi pi-trash",
                        severity: "danger" as const,
                        onSelect: () => setPendingDelete(item.id),
                      }]
                    : []),
                ]}
              />
            ),
          },
        ]}
        onAdd={
          canManage
            ? () =>
                setEditing({
                  input: inputFromRecord(undefined, meetingFields),
                })
            : undefined
        }
        addLabel="ședință"
      />
      <RecordFormDialog
        open={editing}
        title={`${editing?.id ? "Editează" : "Adaugă"} — ședință`}
        fields={meetingFields}
        onClose={() => setEditing(undefined)}
        onChange={(input) =>
          setEditing((current) => (current ? { ...current, input } : current))
        }
        onSave={() =>
          editing &&
          action(async () => {
            await api.saveGovernanceMeeting(governanceMeetingInput(editing.input), editing.id);
          })
        }
      />
      <RecordDetailDialog
        record={detail}
        title="Ședință de guvernanță"
        fields={governanceFields}
        onClose={() => setDetail(undefined)}
      />
      <DeleteDialog
        open={pendingDelete}
        onClose={() => setPendingDelete(undefined)}
        onConfirm={() =>
          pendingDelete &&
          action(async () => {
            await api.deleteGovernanceMeeting(pendingDelete);
            setPendingDelete(undefined);
          })
        }
      />
      <GovernanceMeetingRelations
        api={api}
        meetingId={selectedMeetingId ?? ""}
        canManage={() => canManage}
        title="Membrii organismelor"
        relations={[
          {
            id: "memberships",
            label: "Membri",
            resource: "governance-memberships",
            fields: [
              { key: "school_year", label: "An școlar" },
              { key: "organism", label: "Organism" },
              {
                key: "app_user_id",
                label: "Utilizator",
                kind: "select",
                search: {
                  label: "Caută Utilizator",
                  value: eligibleUserQuery,
                  onChange: setEligibleUserQuery,
                  loading: eligibleUsersLoading,
                },
                options: eligibleUsers.map((user) => ({
                  value: user.id,
                  label: user.name,
                })),
              },
              { key: "role_name", label: "Rol" },
              { key: "mandate_from", label: "Mandat de la", kind: "date" },
              { key: "mandate_to", label: "Mandat până la", kind: "date" },
              { key: "voting_right", label: "Drept vot", kind: "boolean" },
              { key: "status", label: "Stare" },
              { key: "notes", label: "Note" },
            ],
          },
        ]}
      />
      <EducationMetadata
        api={api}
        resources={["governance-meeting-filters"]}
      />
      <GovernanceMeetingRelations
        api={api}
        meetingId=""
        canManage={() => canManage}
        title="Organisme de guvernanță"
        relations={[
          {
            id: "bodies",
            label: "Organisme",
            resource: "governance-bodies",
            readOnly: true,
            summary: "governance-body-completeness",
            fields: [
              { key: "school_year", label: "An școlar" },
              { key: "organism", label: "Denumire" },
              { key: "body_type", label: "Tip organism" },
              { key: "status", label: "Stare" },
              { key: "chairperson", label: "Președinte" },
              { key: "secretary_name", label: "Secretar" },
              { key: "mandate_from", label: "Mandat de la", kind: "date" },
              { key: "mandate_to", label: "Mandat până la", kind: "date" },
              { key: "notes", label: "Note" },
            ],
          },
        ]}
      />
      {selectedMeetingId && (
        <GovernanceMeetingRelations
          api={api}
          meetingId={selectedMeetingId}
          canManage={() => canManageMeeting(selectedMeetingId)}
        />
      )}
      {selectedMeetingId && (
        <EducationMetadata
          api={api}
          resources={["governance-meeting-finalization"]}
          parentID={selectedMeetingId}
        />
      )}
    </div>
  );
}

type RelatedConfig = {
  id: string;
  label: string;
  resource: EducationRelatedResource;
  fields: RecordField[];
  /**
   * A subresource may deliberately be governed by a narrower permission than
   * its parent dossier.  Keeping this alongside its endpoint prevents the UI
   * from offering an action that the backend will reject.
   */
  managePermission?: string;
  /** A governed projection may be inspected but not edited generically. */
  readOnly?: boolean;
  pdf?: boolean;
  /** The sole handler-declared server-side filter for this related endpoint. */
  filterKey?: string;
  summary?: EducationMetadataResource;
};
const meetingRelations: RelatedConfig[] = [
  {
    id: "participants",
    label: "Participanți",
    resource: "meeting-participants",
    fields: [
      { key: "full_name", label: "Nume" },
      { key: "role_name", label: "Rol" },
      { key: "member_type", label: "Tip membru" },
      { key: "attendance_status", label: "Prezență" },
      { key: "voting_right", label: "Drept vot", kind: "boolean" },
      { key: "signature_present", label: "Semnătură", kind: "boolean" },
      { key: "notes", label: "Note" },
    ],
  },
  {
    id: "documents",
    label: "Documente",
    resource: "meeting-documents",
    pdf: true,
    fields: [
      { key: "document_type", label: "Tip document" },
      { key: "title", label: "Titlu" },
      { key: "document_number", label: "Număr" },
      { key: "registry_number", label: "Nr. registratură" },
      { key: "publication_status", label: "Publicare" },
      { key: "custody_owner", label: "Custode" },
      { key: "signed_by", label: "Semnat de" },
      { key: "issued_on", label: "Emis la", kind: "date" },
      { key: "summary", label: "Rezumat" },
    ],
  },
  {
    id: "votes",
    label: "Voturi",
    resource: "meeting-votes",
    fields: [
      { key: "subject_title", label: "Subiect" },
      { key: "agenda_order", label: "Ordine", kind: "number" },
      { key: "decision_type", label: "Tip decizie" },
      { key: "votes_for", label: "Pentru", kind: "number" },
      { key: "votes_against", label: "Împotrivă", kind: "number" },
      { key: "abstentions", label: "Abțineri", kind: "number" },
      { key: "outcome", label: "Rezultat" },
      { key: "requires_follow_up", label: "Urmărire", kind: "boolean" },
      { key: "legal_basis", label: "Temei legal" },
      { key: "notes", label: "Note" },
    ],
  },
  {
    id: "minutes",
    label: "Minute",
    resource: "meeting-minutes",
    pdf: true,
    fields: [
      { key: "agenda_order", label: "Ordine", kind: "number" },
      { key: "topic_title", label: "Subiect" },
      { key: "discussion_summary", label: "Discuții" },
      { key: "decision_summary", label: "Decizie" },
      { key: "responsible_party", label: "Responsabil" },
      { key: "due_on", label: "Termen", kind: "date" },
      { key: "follow_up_status", label: "Urmărire" },
      { key: "requires_publication", label: "Publicare", kind: "boolean" },
      { key: "notes", label: "Note" },
    ],
  },
  {
    id: "resolutions",
    label: "Hotărâri",
    resource: "meeting-resolutions",
    pdf: true,
    fields: [
      { key: "vote_id", label: "Vot" },
      { key: "title", label: "Titlu" },
      { key: "resolution_type", label: "Tip" },
      { key: "publication_status", label: "Publicare" },
      { key: "anonymization_state", label: "Anonimizare" },
      { key: "issued_on", label: "Emis la", kind: "date" },
      { key: "signed_by", label: "Semnat de" },
      { key: "notes", label: "Note" },
    ],
  },
];
export const domainRelations: Partial<
  Record<EducationRecordsDomain, RelatedConfig[]>
> = {
  decisions: [
  {
    id: "issuances",
    label: "Emiteri",
    managePermission: "education.decisions.issuance.manage",
      resource: "decision-issuances",
      fields: [
        { key: "document_type", label: "Tip document" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "recipient_role", label: "Funcție destinatar" },
        { key: "delivery_channel", label: "Canal transmitere" },
        { key: "delivery_status", label: "Stare transmitere" },
        { key: "signed_on", label: "Semnat la", kind: "date" },
        { key: "delivered_on", label: "Predat la", kind: "date" },
        { key: "acknowledged_on", label: "Confirmat la", kind: "date" },
        { key: "file_reference", label: "Referință fișier" },
        { key: "notes", label: "Note" },
      ],
    },
  {
    id: "publication-steps",
    label: "Pași publicare",
    managePermission: "education.compliance.manage",
      resource: "decision-publication-steps",
      fields: [
        { key: "step_order", label: "Ordine", kind: "number" },
        { key: "step_type", label: "Tip pas" },
        { key: "status", label: "Stare" },
        { key: "responsible_name", label: "Responsabil" },
        { key: "publication_channel", label: "Canal" },
        { key: "due_on", label: "Termen", kind: "date" },
        { key: "completed_on", label: "Finalizat la", kind: "date" },
        { key: "publication_reference", label: "Referință" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  regulations: [
    {
      id: "versions",
      label: "Versiuni",
      resource: "regulation-versions",
      fields: [
        { key: "version_label", label: "Versiune" },
        { key: "version_status", label: "Stare" },
        { key: "prepared_by", label: "Pregătit de" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "effective_from", label: "Aplicabil de la", kind: "date" },
        { key: "published_on", label: "Publicat la", kind: "date" },
        { key: "file_reference", label: "Referință fișier" },
        { key: "change_summary", label: "Sinteza modificărilor" },
      ],
    },
    {
      id: "workflow",
      label: "Pași flux",
      resource: "regulation-workflow",
      fields: [
        { key: "phase_order", label: "Ordine", kind: "number" },
        { key: "phase_type", label: "Etapă" },
        { key: "audience", label: "Destinatari" },
        { key: "started_on", label: "Început la", kind: "date" },
        { key: "status", label: "Stare" },
        { key: "due_on", label: "Termen", kind: "date" },
        { key: "completed_on", label: "Finalizat la", kind: "date" },
        { key: "decision_reference", label: "Referință decizie" },
        { key: "feedback_count", label: "Observații", kind: "number" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  committees: [
  {
    id: "members",
    label: "Membri comisie",
    managePermission: "education.governance.manage",
      resource: "committee-members",
      fields: [
        { key: "full_name", label: "Nume complet" },
        { key: "role_name", label: "Rol" },
        {
          key: "member_type",
          label: "Tip",
          kind: "select",
          options: [
            { value: "presedinte", label: "Președinte" },
            { value: "secretar", label: "Secretar" },
            { value: "membru", label: "Membru" },
            { value: "observator", label: "Observator" },
            { value: "invitat", label: "Invitat" },
          ],
        },
        { key: "appointed_on", label: "Numit la", kind: "date" },
        { key: "released_on", label: "Eliberat la", kind: "date" },
        { key: "voting_right", label: "Drept vot", kind: "boolean" },
        {
          key: "status",
          label: "Stare",
          kind: "select",
          options: [
            { value: "active", label: "Activ" },
            { value: "inactive", label: "Inactiv" },
            { value: "replaced", label: "Înlocuit" },
          ],
        },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  managerial: [
    {
      id: "documents",
      label: "Documente dosar",
      resource: "managerial-documents",
      pdf: true,
      fields: [
        {
          key: "document_category",
          label: "Categorie",
          kind: "select",
          options: ["diagnoza", "prognoza", "evidenta", "planificare", "raport", "anexa", "hotarare", "procedura"].map((value) => ({ value, label: value })),
        },
        { key: "title", label: "Titlu" },
        {
          key: "document_status",
          label: "Stare",
          kind: "select",
          options: ["draft", "in_review", "approved", "published", "archived"].map((value) => ({ value, label: value })),
        },
        { key: "version_label", label: "Versiune" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "publication_required", label: "Publicare", kind: "boolean" },
        { key: "registered_on", label: "Înregistrat la", kind: "date" },
        {
          key: "approved_on",
          label: "Aprobat la",
          kind: "date",
          required: (input) => ["approved", "published", "archived"].includes(String(input.document_status ?? "")),
        },
        { key: "owner_name", label: "Responsabil" },
        { key: "file_reference", label: "Referință fișier" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "workflow",
      label: "Pași flux",
      resource: "managerial-workflow",
      fields: [
        { key: "stage_order", label: "Ordine", kind: "number" },
        { key: "stage_type", label: "Etapă" },
        { key: "status", label: "Stare" },
        { key: "assigned_to", label: "Alocat" },
        { key: "due_on", label: "Termen", kind: "date" },
        { key: "completed_on", label: "Finalizat la", kind: "date" },
        { key: "requires_signature", label: "Semnătură", kind: "boolean" },
        { key: "decision_reference", label: "Referință decizie" },
        { key: "outcome_note", label: "Rezultat" },
      ],
    },
  ],
  personnel: [
    {
      id: "assignments",
      label: "Încadrări",
      resource: "personnel-assignments",
      fields: [
        { key: "assignment_code", label: "Cod încadrare", form: false },
        { key: "assignment_title", label: "Titlu încadrare" },
        { key: "assignment_type", label: "Tip", kind: "select", options: ["diriginte", "coordonator_proiect", "responsabil_comisie", "mentor", "membru_comisie", "administrator_structura"].map((value) => ({ value, label: value })) },
        { key: "status", label: "Stare", kind: "select", options: ["propus", "activ", "suspendat", "incetat"].map((value) => ({ value, label: value })) },
        { key: "assigned_on", label: "Atribuit la", kind: "date" },
        { key: "ended_on", label: "Încheiat la", kind: "date" },
        { key: "weekly_hours", label: "Ore săptămânale", kind: "number" },
        { key: "decision_reference", label: "Referință decizie" },
        { key: "notes", label: "Note" },
      ],
    },
  {
    id: "file-documents",
    label: "Documente dosar",
    managePermission: "education.personnel.files.manage",
      resource: "personnel-file-documents",
      fields: [
        { key: "document_code", label: "Cod document", form: false },
        { key: "document_category", label: "Categorie document", kind: "select", options: ["identificare", "studii", "cariera", "evaluare", "declaratie", "medical", "disciplina", "management"].map((value) => ({ value, label: value })) },
        { key: "document_title", label: "Titlu" },
        { key: "confidentiality_level", label: "Nivel confidențialitate", kind: "select", options: ["intern", "confidential", "strict_confidential"].map((value) => ({ value, label: value })) },
        { key: "file_scope", label: "Domeniu fișier", kind: "select", options: ["dosar_personal", "dosar_director", "dosar_director_adjunct"].map((value) => ({ value, label: value })) },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "expires_on", label: "Expiră la", kind: "date" },
        { key: "file_reference", label: "Referință fișier" },
        { key: "included_in_portfolio", label: "Inclus în portofoliu", kind: "boolean" },
        { key: "sensitive_data", label: "Date sensibile", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "disciplinary-cases",
      label: "Cazuri disciplinare",
      resource: "personnel-disciplinary-cases",
      fields: [
        { key: "case_code", label: "Cod caz", form: false },
        { key: "case_type", label: "Tip caz", kind: "select", options: ["sesizare", "cercetare", "sanctiune", "contestatie"].map((value) => ({ value, label: value })) },
        { key: "status", label: "Stare", kind: "select", options: ["deschis", "in_cercetare", "solutionat", "contestat", "inchis"].map((value) => ({ value, label: value })) },
        { key: "reported_on", label: "Raportat la", kind: "date" },
        { key: "hearing_on", label: "Audiere la", kind: "date" },
        { key: "resolved_on", label: "Soluționat la", kind: "date" },
        { key: "committee_name", label: "Comisie" },
        { key: "legal_basis", label: "Temei legal" },
        { key: "sanction", label: "Sancțiune" },
        { key: "notes", label: "Note" },
      ],
    },
  {
    id: "access-events",
    label: "Evenimente acces",
    managePermission: "education.personnel.access.manage",
      resource: "personnel-access-events",
      fields: [
        { key: "event_type", label: "Tip", kind: "select", options: ["consultare", "predare", "actualizare", "arhivare", "export"].map((value) => ({ value, label: value })) },
        { key: "accessed_on", label: "Accesat la", kind: "date" },
        { key: "closed_on", label: "Închis la", kind: "date" },
        { key: "actor_name", label: "Operator" },
        { key: "actor_role", label: "Rol operator" },
        { key: "access_channel", label: "Canal acces", kind: "select", options: ["fizic", "digital", "mixt"].map((value) => ({ value, label: value })) },
        { key: "purpose", label: "Scop" },
        { key: "sensitive_scope", label: "Domeniu sensibil", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  evaluations: [
    {
      id: "self-reviews",
      label: "Autoevaluări",
      resource: "evaluation-self-reviews",
      fields: [
        { key: "review_code", label: "Cod autoevaluare", form: false },
        { key: "completed_on", label: "Finalizat la", kind: "date" },
        { key: "narrative_type", label: "Tip relatare", kind: "select", options: ["autoevaluare", "performanta", "dezvoltare", "impact"].map((value) => ({ value, label: value })) },
        { key: "section_title", label: "Secțiune" },
        { key: "status", label: "Stare", kind: "select", options: ["draft", "submitted", "validated", "returned"].map((value) => ({ value, label: value })) },
        { key: "assumed_score", label: "Punctaj asumat", kind: "number" },
        { key: "evidence_summary", label: "Sinteză dovezi" },
        { key: "strengths", label: "Puncte forte" },
        { key: "improvement_needs", label: "Nevoi îmbunătățire" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "criteria",
      label: "Criterii",
      resource: "evaluation-criteria",
      fields: [
        { key: "criterion_code", label: "Cod criteriu", form: false },
        { key: "criterion_category", label: "Categorie", kind: "select", options: ["proiectare", "predare", "evaluare", "management_clasa", "dezvoltare", "parteneriat"].map((value) => ({ value, label: value })) },
        { key: "criterion_label", label: "Criteriu" },
        { key: "max_score", label: "Maxim", kind: "number" },
        { key: "self_score", label: "Autoevaluare", kind: "number" },
        { key: "reviewer_score", label: "Evaluator", kind: "number" },
        { key: "final_score", label: "Final", kind: "number" },
        { key: "status", label: "Stare", kind: "select", options: ["draft", "reviewed", "validated", "contested"].map((value) => ({ value, label: value })) },
        { key: "evidence_summary", label: "Sinteză dovezi" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "appeals",
      label: "Contestații",
      resource: "evaluation-appeals",
      pdf: true,
      fields: [
        { key: "appeal_code", label: "Cod contestație", form: false },
        { key: "submitted_by", label: "Depus de" },
        { key: "submitted_on", label: "Depus la", kind: "date" },
        { key: "status", label: "Stare", kind: "select", options: ["submitted", "review", "accepted", "rejected", "resolved"].map((value) => ({ value, label: value })) },
        { key: "grounds", label: "Motive" },
        { key: "hearing_on", label: "Audiere la", kind: "date" },
        { key: "resolved_on", label: "Soluționat la", kind: "date" },
        { key: "decision_summary", label: "Decizie" },
        { key: "committee_note", label: "Notă comisie" },
        { key: "attached_to_personnel_file", label: "Atașat dosar personal", kind: "boolean" },
      ],
    },
    {
      id: "result-issues",
      label: "Comunicări rezultat",
      resource: "evaluation-result-issues",
      pdf: true,
      fields: [
        { key: "issue_code", label: "Cod comunicare", form: false },
        { key: "document_type", label: "Tip document", kind: "select", options: ["fisa_evaluare", "comunicare", "decizie", "raport_final"].map((value) => ({ value, label: value })) },
        { key: "recipient_name", label: "Destinatar" },
        { key: "recipient_role", label: "Rol destinatar" },
        { key: "delivery_channel", label: "Canal", kind: "select", options: ["registratura", "email", "intern", "posta"].map((value) => ({ value, label: value })) },
        { key: "delivery_status", label: "Stare livrare", kind: "select", options: ["pregatit", "emis", "transmis", "confirmat"].map((value) => ({ value, label: value })) },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "delivered_on", label: "Livrat la", kind: "date" },
        { key: "acknowledged_on", label: "Confirmat la", kind: "date" },
        { key: "registry_reference", label: "Referință registratură" },
        { key: "attached_to_personnel_file", label: "Atașat dosar personal", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  mobility: [
    {
      id: "documents",
      label: "Documente",
      resource: "mobility-documents",
      filterKey: "document_code",
      fields: [
        { key: "document_code", label: "Cod document", form: false },
        { key: "document_type", label: "Tip" },
        { key: "document_title", label: "Titlu" },
        { key: "registered_on", label: "Înregistrat la", kind: "date" },
        { key: "validation_status", label: "Validare" },
        { key: "stage_scope", label: "Etapă document" },
        { key: "submitted_by", label: "Depus de" },
        { key: "verified_by", label: "Verificat de" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "scores",
      label: "Punctaje",
      resource: "mobility-scores",
      filterKey: "criterion_code",
      fields: [
        { key: "criterion_category", label: "Categorie criteriu" },
        { key: "criterion_code", label: "Cod criteriu" },
        { key: "criterion_label", label: "Criteriu" },
        { key: "max_score", label: "Maxim", kind: "number" },
        { key: "awarded_score", label: "Acordat", kind: "number" },
        { key: "contested", label: "Contestat", kind: "boolean" },
        { key: "evidence_reference", label: "Referință dovezi" },
        { key: "validated_by", label: "Validat de" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "appeals",
      label: "Contestații",
      resource: "mobility-appeals",
      pdf: true,
      filterKey: "appeal_code",
      fields: [
        { key: "appeal_code", label: "Cod contestație", form: false },
        { key: "submitted_by", label: "Depus de" },
        { key: "submitted_on", label: "Depus la", kind: "date" },
        { key: "status", label: "Stare" },
        { key: "grounds", label: "Motive" },
        { key: "hearing_on", label: "Audiere la", kind: "date" },
        { key: "resolved_on", label: "Soluționat la", kind: "date" },
        { key: "decision_summary", label: "Decizie" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "final-decisions",
      label: "Decizii finale",
      resource: "mobility-final-decisions",
      pdf: true,
      filterKey: "decision_code",
      fields: [
        { key: "decision_code", label: "Cod decizie", form: false },
        { key: "decision_type", label: "Tip decizie" },
        { key: "outcome", label: "Rezultat" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "effective_from", label: "Aplicabil de la", kind: "date" },
        { key: "panel_name", label: "Comisie" },
        { key: "destination_unit", label: "Unitate destinație" },
        { key: "legal_basis", label: "Temei legal" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "result-issues",
      label: "Comunicări rezultat",
      resource: "mobility-result-issues",
      pdf: true,
      filterKey: "issue_code",
      fields: [
        { key: "issue_code", label: "Cod comunicare", form: false },
        { key: "document_type", label: "Tip document" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "recipient_role", label: "Funcție" },
        { key: "delivery_channel", label: "Canal" },
        { key: "delivery_status", label: "Stare livrare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "delivered_on", label: "Livrat la", kind: "date" },
        { key: "registry_reference", label: "Referință registratură" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  merit: [
    {
      id: "documents",
      label: "Documente",
      resource: "merit-documents",
      filterKey: "document_code",
      fields: [
        { key: "document_code", label: "Cod document", form: false },
        { key: "document_type", label: "Tip" },
        { key: "document_title", label: "Titlu" },
        { key: "registered_on", label: "Înregistrat la", kind: "date" },
        { key: "validation_status", label: "Validare" },
        { key: "submitted_by", label: "Depus de" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "scores",
      label: "Punctaje",
      resource: "merit-scores",
      filterKey: "criterion_code",
      fields: [
        { key: "criterion_code", label: "Cod criteriu" },
        { key: "criterion_label", label: "Criteriu" },
        {
          key: "criterion_category",
          label: "Categorie",
          kind: "select",
          options: [
            { label: "Performanță", value: "performanta" },
            { label: "Impact", value: "impact" },
            { label: "Dezvoltare", value: "dezvoltare" },
            { label: "Management", value: "management" },
            { label: "Incluziune", value: "incluziune" },
          ],
        },
        { key: "max_score", label: "Maxim", kind: "number" },
        { key: "awarded_score", label: "Acordat", kind: "number" },
        { key: "reviewer_name", label: "Evaluator" },
        {
          key: "panel_stage",
          label: "Etapă comisie",
          kind: "select",
          options: [
            { label: "Autoevaluare", value: "autoevaluare" },
            { label: "Evaluare comisie", value: "evaluare_comisie" },
            { label: "Validare finală", value: "validare_finala" },
          ],
        },
        { key: "contested", label: "Contestat", kind: "boolean" },
        { key: "evidence_reference", label: "Referință dovezi" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "appeals",
      label: "Contestații",
      resource: "merit-appeals",
      pdf: true,
      filterKey: "appeal_code",
      fields: [
        { key: "appeal_code", label: "Cod contestație", form: false },
        { key: "submitted_by", label: "Depus de" },
        { key: "submitted_on", label: "Depus la", kind: "date" },
        { key: "status", label: "Stare" },
        { key: "grounds", label: "Motive" },
        { key: "resolved_on", label: "Soluționat la", kind: "date" },
        { key: "decision_summary", label: "Decizie" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "final-decisions",
      label: "Decizii finale",
      resource: "merit-final-decisions",
      pdf: true,
      filterKey: "decision_code",
      fields: [
        { key: "decision_code", label: "Cod decizie", form: false },
        { key: "decision_stage", label: "Etapă" },
        { key: "outcome", label: "Rezultat" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "effective_from", label: "Aplicabil de la", kind: "date" },
        { key: "panel_name", label: "Comisie" },
        { key: "funded", label: "Finanțat", kind: "boolean" },
        { key: "legal_basis", label: "Temei legal" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "result-issues",
      label: "Comunicări rezultat",
      resource: "merit-result-issues",
      pdf: true,
      filterKey: "issue_code",
      fields: [
        { key: "issue_code", label: "Cod comunicare", form: false },
        { key: "document_type", label: "Tip document" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "recipient_role", label: "Funcție" },
        { key: "delivery_channel", label: "Canal" },
        { key: "delivery_status", label: "Stare livrare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "delivered_on", label: "Livrat la", kind: "date" },
        { key: "registry_reference", label: "Referință registratură" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  // Portfolio relations use their own generated contracts below. They cannot
  // use the generic related-record editor because each has a distinct body,
  // query allow-list and lifecycle policy.
  portfolios: [],
};

export function hasDomainRelations(domain: EducationRecordsDomain): boolean {
  return Boolean(domainRelations[domain]?.length);
}

function GovernanceMeetingRelations({
  api,
  meetingId,
  canManage,
  relations = meetingRelations,
  title = "Ședință selectată — operațiuni",
}: {
  api: EducationApi;
  meetingId: string;
  canManage: (relation: RelatedConfig) => boolean;
  relations?: RelatedConfig[];
  title?: string;
}) {
  // Keep only the stable key in local state. Relation definitions are rebuilt
  // by the parent when async selector data changes; storing the whole object
  // would freeze fields such as `search.loading` and `options` at their first
  // render, leaving a successfully loaded Select permanently disabled.
  const [relationID, setRelationID] = useState(relations[0].id);
  const relation = relations.find((item) => item.id === relationID) ?? relations[0];
  const [page, setPage] = useState<EducationPage<EducationRecord>>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 50,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [editing, setEditing] = useState<{
    id?: string;
    input: EducationRecordInput;
  }>();
  const [detail, setDetail] = useState<EducationRecord>();
  const [pendingDelete, setPendingDelete] = useState<string>();
  const [selectedRelatedId, setSelectedRelatedId] = useState<string>();
  const [pageNumber, setPageNumber] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [sort, setSort] = useState<{ field?: string; direction?: "asc" | "desc" }>({});
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [refresh, setRefresh] = useState(0);
  const filterEffectReady = useRef(false);
  const relationCanManage = !relation.readOnly && canManage(relation);
  const parentID = meetingId || undefined;
  const load = useCallback(async (
    nextPage = 1,
    nextPageSize = 20,
    nextSort: { field?: string; direction?: "asc" | "desc" } = {},
    nextFilters: Record<string, string> = {},
  ) => {
    setLoading(true);
    setError(undefined);
    try {
      const result = await api.relatedRecords(relation.resource, parentID, {
        page: nextPage,
        pageSize: nextPageSize,
        sort: nextSort.field,
        direction: nextSort.direction,
        filters: nextFilters,
      });
      setPage(result);
      setPageNumber(result.page ?? nextPage);
      setPageSize(result.pageSize ?? nextPageSize);
    } catch {
      setError("Subresursa nu a putut fi încărcată.");
    } finally {
      setLoading(false);
    }
  }, [api, parentID, relation.resource]);
  useEffect(() => {
    void load(1, 20, {}, {});
  }, [load, refresh]);
  useEffect(() => {
    if (!filterEffectReady.current) {
      filterEffectReady.current = true;
      return;
    }
    const timer = window.setTimeout(() => {
      void load(1, pageSize, sort, filters);
    }, 350);
    return () => window.clearTimeout(timer);
  }, [filters, load]);
  const action = async (fn: () => Promise<void>) => {
    try {
      await fn();
      setEditing(undefined);
      setDetail(undefined);
      setRefresh((value) => value + 1);
    } catch {
      setError("Operația nu a putut fi finalizată.");
    }
  };
  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>{title}</Card.Title>
        <Card.Content>
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap gap-2">
              {relations.map((item) => (
                <Button
                  key={item.id}
                  size="small"
                  variant={item.id === relation.id ? undefined : "outlined"}
                  severity={item.id === relation.id ? undefined : "secondary"}
                  onClick={() => {
                    setRelationID(item.id);
                    setFilters({});
                    setSort({});
                    setPageNumber(1);
                  }}
                >
                  {item.label}
                </Button>
              ))}
            </div>
            {error && (
              <Message.Root severity="error">
                <Message.Content>
                  <Message.Text>{error}</Message.Text>
                </Message.Content>
              </Message.Root>
            )}
            {loading ? (
              <div className="flex justify-center p-6">
                <Spinner />
              </div>
            ) : page.items.length === 0 && !relationCanManage ? (
              <Message.Root severity="info">
                <Message.Content>
                  <Message.Text>
                    Nu există {relation.label.toLowerCase()}.
                  </Message.Text>
                </Message.Content>
              </Message.Root>
            ) : (
              <DataTable.Root
                data={page.items as Record<string, unknown>[]}
                dataKey="id"
                scrollable
                className="max-h-[calc(100dvh-22rem)] min-h-64 overflow-auto"
              >
                <DataTable.Table>
                  <DataTable.THead className="sticky top-0 z-10">
                    <DataTable.THeadRow>
                      {relation.fields.map((field) => (
                        <DataTable.THeadCell key={field.key}>
                          <div className="flex min-w-36 flex-col gap-1">
                            <Button
                              size="small"
                              variant="text"
                              onClick={() => {
                                const direction = sort.field === field.key && sort.direction === "asc" ? "desc" : "asc";
                                setSort({ field: field.key, direction });
                                void load(1, pageSize, { field: field.key, direction }, filters);
                              }}
                            >
                              {field.label}{sort.field === field.key ? sort.direction === "asc" ? " ↑" : " ↓" : ""}
                            </Button>
                            {relation.filterKey === field.key && (
                              <InputText
                                aria-label={`Filtru ${field.label}`}
                                className="w-full"
                                value={filters[field.key] ?? ""}
                                onChange={(event: ChangeEvent<HTMLInputElement>) =>
                                  setFilters((current) => ({ ...current, [field.key]: event.target.value }))
                                }
                              />
                            )}
                          </div>
                        </DataTable.THeadCell>
                      ))}
                      <DataTable.THeadCell frozen alignFrozen="right">
                        <span className="flex items-center justify-between gap-2">
                          <span>Acțiuni</span>
                          {relationCanManage && (
                            <Button
                              iconOnly
                              rounded
                              size="small"
                              aria-label={`Adaugă ${relation.label.toLowerCase()}`}
                              title={`Adaugă ${relation.label.toLowerCase()}`}
                              onClick={() =>
                                setEditing({
                                  input: inputFromRecord(undefined, relation.fields),
                                })
                              }
                            >
                              <i className="pi pi-plus" aria-hidden="true" />
                            </Button>
                          )}
                        </span>
                      </DataTable.THeadCell>
                    </DataTable.THeadRow>
                  </DataTable.THead>
                  <DataTable.TBody>
                    {({ item, index }) => {
                      const record = item as EducationRecord;
                      return (
                        <DataTable.Row key={record.id} index={index}>
                          {relation.fields.map((field) => (
                            <DataTable.Cell key={field.key}>
                              {field.kind === "boolean"
                                ? <Tag value={record[field.key] ? "Da" : "Nu"} severity={record[field.key] ? "success" : "secondary"} />
                                : field.key.includes("status")
                                  ? <Tag value={String(record[field.key] ?? "—")} severity="secondary" />
                                  : displayRecord(record, [field.key])}
                            </DataTable.Cell>
                          ))}
                          <DataTable.Cell frozen alignFrozen="right">
                            <SchoolRowActionMenu
                              actions={[
                                {
                                  label: "Detalii",
                                  icon: "pi pi-eye",
                                  onSelect: () =>
                                  void api
                                    .relatedDetail(relation.resource, parentID, record.id)
                                    .then((value) => {
                                      setDetail(value);
                                      setSelectedRelatedId(record.id);
                                    })
                                    .catch(() =>
                                      setError(
                                        "Detaliul nu a putut fi încărcat.",
                                      ),
                                    ),
                                  },
                                ...(relation.pdf
                                  ? [{
                                      label: "PDF",
                                      icon: "pi pi-file-pdf",
                                      onSelect: () =>
                                    viewPdf(
                                      api.relatedPdf(relation.resource, parentID, record.id),
                                      () =>
                                        setError(
                                          "PDF-ul nu a putut fi încărcat.",
                                      ),
                                    ),
                                      }]
                                  : []),
                                ...(relationCanManage
                                  ? [{
                                      label: "Editează",
                                      icon: "pi pi-pencil",
                                      onSelect: () =>
                                    setEditing({
                                      id: record.id,
                                      input: inputFromRecord(
                                        record,
                                        relation.fields,
                                      ),
                                      }),
                                      }, {
                                        label: "Șterge",
                                        icon: "pi pi-trash",
                                        severity: "danger" as const,
                                        onSelect: () => setPendingDelete(record.id),
                                      }]
                                  : []),
                              ]}
                            />
                          </DataTable.Cell>
                        </DataTable.Row>
                      );
                    }}
                  </DataTable.TBody>
                </DataTable.Table>
              </DataTable.Root>
            )}
            {!loading && (
              <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2" aria-label="Paginare subresursă">
                <span>{page.total ? `${(pageNumber - 1) * pageSize + 1} - ${Math.min(pageNumber * pageSize, page.total)} din ${page.total}` : "0 rezultate"}</span>
                <div className="flex items-center gap-2">
                  <Select.Root
                    value={pageSize}
                    options={[10, 20, 50, 100].map((value) => ({ label: String(value), value }))}
                    optionLabel="label"
                    optionValue="value"
                    onValueChange={(event: SelectValueChangeEvent) => {
                      const next = Number(event.value);
                      setPageSize(next);
                      setPageNumber(1);
                      void load(1, next, sort, filters);
                    }}
                  >
                    <Select.Trigger aria-label="Rânduri pe pagină subresursă"><Select.Value /><Select.Indicator /></Select.Trigger>
                    <Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
                  </Select.Root>
                  <Button size="small" variant="outlined" disabled={pageNumber <= 1} onClick={() => { const next = pageNumber - 1; setPageNumber(next); void load(next, pageSize, sort, filters); }}>Anterior</Button>
                  <Button size="small" variant="outlined" disabled={pageNumber * pageSize >= page.total} onClick={() => { const next = pageNumber + 1; setPageNumber(next); void load(next, pageSize, sort, filters); }}>Următor</Button>
                </div>
              </div>
            )}
            {!loading && page.items.length === 0 && relationCanManage && (
              <Message.Root severity="info">
                <Message.Content>
                  <Message.Text>
                    Nu există {relation.label.toLowerCase()}.
                  </Message.Text>
                </Message.Content>
              </Message.Root>
            )}
            {relationCanManage && <RecordFormDialog
              open={editing}
              title={`${editing?.id ? "Editează" : "Adaugă"} ${relation.label.toLowerCase()}`}
              fields={relation.fields}
              onClose={() => setEditing(undefined)}
              onChange={(input) =>
                setEditing((current) =>
                  current ? { ...current, input } : current,
                )
              }
              onSave={() =>
                editing &&
                void action(async () => {
                  await api.saveRelated(relation.resource, parentID, editing.input, editing.id);
                })
              }
            />}
            <RecordDetailDialog
              record={detail}
              title={relation.label}
              fields={relation.fields}
              onClose={() => setDetail(undefined)}
            />
            {selectedRelatedId && relation.summary && (
              <EducationMetadata
                api={api}
                resources={[relation.summary]}
                parentID={selectedRelatedId}
              />
            )}
            {relationCanManage && <DeleteDialog
              open={pendingDelete}
              onClose={() => setPendingDelete(undefined)}
              onConfirm={() =>
                pendingDelete &&
                void action(async () => {
                  await api.deleteRelated(relation.resource, parentID, pendingDelete);
                  setPendingDelete(undefined);
                })
              }
            />
            }
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}

const recordPrimaryKeys = [
  "title",
  "name",
  "number",
  "code",
  "subject",
  "person_name",
  "school_year",
];
const recordStatusKeys = ["status", "state", "stage", "publication_status"];
function displayRecord(record: EducationRecord, keys: readonly string[]) {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" || typeof value === "number")
      return String(value);
  }
  return record.id;
}

/** A real list endpoint is wired for every catalogue area. Domain-specific
 * detail/editor views can be added only against their documented DTOs. */
type RecordField = {
  key: string;
  label: string;
  kind?: "text" | "date" | "number" | "boolean" | "select";
  min?: number;
  max?: number;
  options?: Array<{ label: string; value: string }>;
  /** Optional server-side search control for a remotely populated Select. */
  search?: {
    label: string;
    value: string;
    onChange: (value: string) => void;
    loading?: boolean;
  };
  /** Static or state-dependent client requirement mirrored from the API contract. */
  required?: boolean | ((input: EducationRecordInput) => boolean);
  /** Server-generated/list-only values are visible but never editable. */
  form?: boolean;
};

function isRecordFieldRequired(field: RecordField, input: EducationRecordInput = {}): boolean {
  if (typeof field.required === "function" ? field.required(input) : Boolean(field.required)) return true;
  if (field.key === "resolved_on") return ["accepted", "rejected", "resolved"].includes(String(input.status ?? ""));
  if (field.key === "decision_summary") return ["accepted", "rejected"].includes(String(input.status ?? ""));
  if (field.key === "delivered_on") return ["transmis", "confirmat"].includes(String(input.delivery_status ?? ""));
  if (field.key === "acknowledged_on") return String(input.delivery_status ?? "") === "confirmat";
  return false;
}

function isRecordFieldMissing(field: RecordField, input: EducationRecordInput = {}): boolean {
  if (!isRecordFieldRequired(field, input)) return false;
  const value = input[field.key];
  return value === undefined || value === null || (typeof value === "string" && !value.trim());
}
const domainFields: Record<EducationRecordsDomain, RecordField[]> = {
  decisions: [
    { key: "school_year", label: "An școlar" },
    { key: "organism", label: "Organism" },
    { key: "title", label: "Titlu" },
    { key: "status", label: "Stare" },
    { key: "publication_status", label: "Publicare" },
    { key: "decision_date", label: "Data deciziei", kind: "date" },
    { key: "legal_basis", label: "Temei legal" },
    { key: "signed_by", label: "Semnat de" },
    { key: "summary", label: "Rezumat" },
  ],
  managerial: [
    { key: "school_year", label: "An școlar" },
    { key: "dossier_type", label: "Tip dosar" },
    { key: "title", label: "Titlu" },
    { key: "status", label: "Stare" },
    { key: "owner_name", label: "Responsabil" },
    { key: "due_on", label: "Termen", kind: "date" },
    {
      key: "publication_required",
      label: "Necesită publicare",
      kind: "boolean",
    },
    { key: "summary", label: "Rezumat" },
  ],
  regulations: [
    { key: "school_year", label: "An școlar" },
    { key: "regulation_type", label: "Tip regulament" },
    { key: "title", label: "Titlu" },
    { key: "status", label: "Stare" },
    { key: "approval_status", label: "Aprobare" },
    { key: "owner_name", label: "Responsabil" },
    { key: "review_due_on", label: "Revizuire până la", kind: "date" },
    { key: "approved_on", label: "Aprobat la", kind: "date" },
    { key: "summary", label: "Rezumat" },
  ],
  committees: [
    { key: "committee_code", label: "Cod comisie", form: false },
    { key: "school_year", label: "An școlar" },
    { key: "committee_type", label: "Tip" },
    { key: "title", label: "Denumire comisie" },
    { key: "status", label: "Stare" },
    { key: "decision_reference", label: "Act de constituire" },
    { key: "starts_on", label: "Începe la", kind: "date" },
    { key: "ends_on", label: "Se încheie la", kind: "date" },
    { key: "evaluation_scope", label: "Comisie de evaluare", kind: "boolean" },
    { key: "notes", label: "Note" },
  ],
  personnel: [
    { key: "employee_code", label: "Cod angajat" },
    { key: "full_name", label: "Nume complet" },
    { key: "role_title", label: "Funcție" },
    { key: "employment_type", label: "Tip angajare" },
    { key: "status", label: "Stare" },
    { key: "evaluation_status", label: "Stare evaluare" },
    { key: "mobility_stage", label: "Etapă mobilitate" },
    { key: "school_year", label: "An școlar" },
    { key: "assigned_unit", label: "Unitate" },
    { key: "phone", label: "Telefon" },
    { key: "email", label: "E-mail" },
    { key: "has_portfolio", label: "Are portofoliu", kind: "boolean" },
    { key: "notes", label: "Note" },
  ],
  evaluations: [
    { key: "employee_code", label: "Cod angajat" },
    { key: "full_name", label: "Angajat" },
    { key: "role_title", label: "Funcție" },
    { key: "school_year", label: "An școlar" },
    { key: "status", label: "Stare" },
    { key: "score", label: "Punctaj", kind: "number" },
    { key: "evaluator_name", label: "Evaluator" },
    { key: "finalized_on", label: "Finalizat la", kind: "date" },
    { key: "summary", label: "Rezumat" },
  ],
  declarations: [
    { key: "employee_code", label: "Cod angajat" },
    { key: "full_name", label: "Declarant" },
    { key: "school_year", label: "An școlar" },
    { key: "declaration_type", label: "Tip declarație" },
    { key: "status", label: "Stare" },
    { key: "submitted_on", label: "Depus la", kind: "date" },
    { key: "valid_until", label: "Valabil până la", kind: "date" },
    { key: "summary", label: "Rezumat" },
  ],
  mobility: [
    { key: "employee_code", label: "Cod angajat" },
    { key: "full_name", label: "Nume complet" },
    { key: "school_year", label: "An școlar" },
    { key: "request_type", label: "Tip mobilitate" },
    { key: "stage", label: "Etapă" },
    { key: "status", label: "Stare" },
    { key: "source_school", label: "Unitate sursă" },
    { key: "destination_school", label: "Unitate destinație" },
    { key: "submitted_on", label: "Depus la", kind: "date" },
    { key: "reviewed_by", label: "Analizat de" },
    { key: "notes", label: "Note" },
  ],
  merit: [
    { key: "full_name", label: "Nume complet" },
    { key: "role_title", label: "Funcție" },
    { key: "school_year", label: "An școlar" },
    { key: "category", label: "Categorie" },
    { key: "status", label: "Stare" },
    { key: "score", label: "Punctaj", kind: "number" },
    { key: "committee_name", label: "Comisie" },
    { key: "decision_date", label: "Data deciziei", kind: "date" },
    { key: "funded", label: "Finanțat", kind: "boolean" },
  ],
  portfolios: [
    { key: "portfolio_code", label: "Cod portofoliu" },
    { key: "owner_name", label: "Titular" },
    { key: "owner_role", label: "Funcție" },
    { key: "school_year", label: "An școlar" },
    { key: "status", label: "Stare" },
    { key: "section_count", label: "Secțiuni", kind: "number" },
    { key: "last_updated_on", label: "Actualizat la", kind: "date" },
    { key: "transfer_status", label: "Transfer" },
    {
      key: "authenticity_declared",
      label: "Autenticitate declarată",
      kind: "boolean",
    },
    { key: "consent_captured", label: "Consimțământ", kind: "boolean" },
    { key: "custodian", label: "Custode" },
    { key: "notes", label: "Note" },
  ],
  compliance: [
    { key: "publication_code", label: "Cod publicare", form: false },
    { key: "domain", label: "Domeniu", kind: "select", options: ["guvernanta", "documente_manageriale", "portofolii", "regulamente", "conformitate"].map((value) => ({ value, label: value })) },
    { key: "entity_type", label: "Tip entitate", kind: "select", options: ["hotarare", "proces_verbal", "procedura_portofoliu", "rof", "roi", "pdi_pas", "raport", "anunt"].map((value) => ({ value, label: value })) },
    { key: "entity_label", label: "Entitate" },
    { key: "publication_channel", label: "Canal", kind: "select", options: ["site_public", "avizier", "intranet", "registratura"].map((value) => ({ value, label: value })) },
    { key: "publication_status", label: "Stare", kind: "select", options: ["pregatit", "publicat", "retras"].map((value) => ({ value, label: value })) },
    { key: "anonymization_status", label: "Anonimizare", kind: "select", options: ["necesara", "finalizata", "nu_este_necesara"].map((value) => ({ value, label: value })) },
    { key: "mandatory", label: "Obligatorie", kind: "boolean" },
    { key: "published_on", label: "Publicat la", kind: "date", required: (input) => String(input.publication_status ?? "") === "publicat" },
    { key: "reviewed_by", label: "Revizuit de" },
    { key: "notes", label: "Note" },
  ],
};

type DomainListCapabilities = {
  /** Fields that the list endpoint documents as accepted header filters. */
  filterableFields: readonly string[];
  /** Fields that the list endpoint documents as accepted sorting keys. */
  sortableFields: readonly string[];
};

/**
 * Header controls are an API capability, not a presentation default.  Keeping
 * this beside the field catalogue makes an unsupported query impossible to
 * emit from a root registry.  Domains which have not yet tightened their
 * generated list contract deliberately retain their current full-column UI;
 * Comisii mirrors its backend allowlist exactly.
 */
export const domainListCapabilities: Record<
  EducationRecordsDomain,
  DomainListCapabilities
> = {
  decisions: {
    filterableFields: domainFields.decisions.map((field) => field.key),
    sortableFields: domainFields.decisions.map((field) => field.key),
  },
  managerial: {
    filterableFields: domainFields.managerial.map((field) => field.key),
    sortableFields: domainFields.managerial.map((field) => field.key),
  },
  regulations: {
    filterableFields: domainFields.regulations.map((field) => field.key),
    sortableFields: domainFields.regulations.map((field) => field.key),
  },
  committees: {
    filterableFields: ["school_year", "committee_type", "title", "status"],
    sortableFields: [
      "school_year",
      "committee_type",
      "title",
      "status",
      "committee_code",
      "starts_on",
    ],
  },
  personnel: {
    filterableFields: domainFields.personnel.map((field) => field.key),
    sortableFields: domainFields.personnel.map((field) => field.key),
  },
  evaluations: {
    filterableFields: domainFields.evaluations.map((field) => field.key),
    sortableFields: domainFields.evaluations.map((field) => field.key),
  },
  declarations: {
    filterableFields: domainFields.declarations.map((field) => field.key),
    sortableFields: domainFields.declarations.map((field) => field.key),
  },
  mobility: {
    filterableFields: domainFields.mobility.map((field) => field.key),
    sortableFields: domainFields.mobility.map((field) => field.key),
  },
  merit: {
    filterableFields: domainFields.merit.map((field) => field.key),
    sortableFields: domainFields.merit.map((field) => field.key),
  },
  portfolios: {
    filterableFields: domainFields.portfolios.map((field) => field.key),
    sortableFields: domainFields.portfolios.map((field) => field.key),
  },
  compliance: {
    filterableFields: domainFields.compliance.map((field) => field.key),
    sortableFields: domainFields.compliance.map((field) => field.key),
  },
};
function permissionForDomain(domain: EducationRecordsDomain) {
  if (domain === "committees") return "education.governance.manage";
  return `education.${domain === "merit" ? "gradatii" : domain}.manage`;
}
export function relationManagePermission(
  domain: EducationRecordsDomain,
  configuredPermission?: string,
) {
  return configuredPermission ?? permissionForDomain(domain);
}
function delegationResourceTypeForDomain(domain: EducationRecordsDomain) {
  switch (domain) {
    case "portfolios": return "portfolio";
    case "decisions": return "decision";
    case "regulations": return "regulation";
    case "personnel": return "personnel";
    default: return "institution";
  }
}
export const domainWizardRoutes: Partial<Record<EducationRecordsDomain, string>> = {
  managerial: "/scoala/governance/managerial-wizard",
  personnel: "/scoala/personnel/wizard",
  evaluations: "/scoala/personnel/evaluations-wizard",
  declarations: "/scoala/personnel/declarations-wizard",
  mobility: "/scoala/personnel/mobility-wizard",
  merit: "/scoala/personnel/merit-wizard",
  portfolios: "/scoala/portfolio/wizard",
};
function supportsPdf(
  domain: EducationRecordsDomain,
): domain is EducationPdfRecordsDomain {
  return [
    "managerial",
    "evaluations",
    "mobility",
    "merit",
    "portfolios",
  ].includes(domain);
}
function viewPdf(load: Promise<Blob>, onError: () => void) {
  const popup = window.open("", "_blank", "noopener");
  void load
    .then((blob) => {
      const url = URL.createObjectURL(blob);
      if (popup) popup.location.href = url;
      else window.location.assign(url);
      window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
    })
    .catch(onError);
}
function inputFromRecord(
  record: EducationRecord | undefined,
  fields: RecordField[],
): EducationRecordInput {
  return Object.fromEntries(
    fields.map(({ key, kind }) => [
      key,
      kind === "boolean" ? Boolean(record?.[key]) : (record?.[key] ?? ""),
    ]),
  ) as EducationRecordInput;
}

const stringInput = (input: EducationRecordInput, key: string) => typeof input[key] === "string" ? input[key] as string : "";
const optionalStringInput = (input: EducationRecordInput, key: string) => typeof input[key] === "string" && input[key] !== "" ? input[key] as string : undefined;
const optionalBooleanInput = (input: EducationRecordInput, key: string) => typeof input[key] === "boolean" ? input[key] as boolean : undefined;
const optionalNumberInput = (input: EducationRecordInput, key: string) => typeof input[key] === "number" ? input[key] as number : undefined;
const enumStringInput = <const T extends readonly string[]>(input: EducationRecordInput, key: string, allowed: T): T[number] => {
  const value = stringInput(input, key);
  if (!(allowed as readonly string[]).includes(value)) throw new Error(`education_invalid_${key}`);
  return value as T[number];
};
const governanceMeetingInput = (input: EducationRecordInput): GovernanceMeetingInput => ({
  chairperson: optionalStringInput(input, "chairperson"),
  chairperson_user_id: stringInput(input, "chairperson_user_id"),
  location: optionalStringInput(input, "location"),
  meeting_date: stringInput(input, "meeting_date"),
  meeting_type: stringInput(input, "meeting_type"),
  organism: stringInput(input, "organism"),
  participants_count: optionalNumberInput(input, "participants_count"),
  quorum_required: optionalNumberInput(input, "quorum_required"),
  school_year: stringInput(input, "school_year"),
  secretary_name: optionalStringInput(input, "secretary_name"),
  secretary_user_id: stringInput(input, "secretary_user_id"),
  status: stringInput(input, "status"),
  summary: optionalStringInput(input, "summary"),
  title: stringInput(input, "title"),
});

/**
 * A generic form may display a record projection, but it never becomes a
 * transport body. Each case deliberately picks the create DTO allow-list, so
 * ids, generated codes, retention and lifecycle fields cannot be submitted.
 */
export function createInputForDomain<D extends EducationRecordsDomain>(domain: D, input: EducationRecordInput): EducationRootCreateInputByDomain[D] {
  switch (domain) {
    case "decisions": { const dto: EducationRootCreateInputByDomain["decisions"] = { decision_date: stringInput(input, "decision_date"), legal_basis: optionalStringInput(input, "legal_basis"), organism: stringInput(input, "organism"), publication_status: stringInput(input, "publication_status"), school_year: stringInput(input, "school_year"), signed_by: optionalStringInput(input, "signed_by"), status: stringInput(input, "status"), summary: optionalStringInput(input, "summary"), title: stringInput(input, "title") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "managerial": { const dto: EducationRootCreateInputByDomain["managerial"] = { dossier_type: stringInput(input, "dossier_type"), due_on: stringInput(input, "due_on"), owner_name: optionalStringInput(input, "owner_name"), publication_required: optionalBooleanInput(input, "publication_required"), school_year: stringInput(input, "school_year"), status: stringInput(input, "status"), summary: optionalStringInput(input, "summary"), title: stringInput(input, "title") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "regulations": { const dto: EducationRootCreateInputByDomain["regulations"] = { approval_status: stringInput(input, "approval_status"), approved_on: optionalStringInput(input, "approved_on"), owner_name: optionalStringInput(input, "owner_name"), regulation_type: stringInput(input, "regulation_type"), review_due_on: stringInput(input, "review_due_on"), school_year: stringInput(input, "school_year"), status: stringInput(input, "status"), summary: optionalStringInput(input, "summary"), title: stringInput(input, "title") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "committees": { const dto: EducationRootCreateInputByDomain["committees"] = { committee_type: stringInput(input, "committee_type"), decision_reference: optionalStringInput(input, "decision_reference"), ends_on: optionalStringInput(input, "ends_on"), evaluation_scope: optionalBooleanInput(input, "evaluation_scope"), notes: optionalStringInput(input, "notes"), school_year: stringInput(input, "school_year"), starts_on: stringInput(input, "starts_on"), status: stringInput(input, "status"), title: stringInput(input, "title") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "personnel": { const dto: EducationRootCreateInputByDomain["personnel"] = { app_user_id: optionalStringInput(input, "app_user_id"), assigned_unit: optionalStringInput(input, "assigned_unit"), email: optionalStringInput(input, "email"), employment_type: enumStringInput(input, "employment_type", ["titular", "suplinitor", "plata_cu_ora", "auxiliar"] as const), evaluation_status: enumStringInput(input, "evaluation_status", ["draft", "in_review", "finalized"] as const), full_name: stringInput(input, "full_name"), has_portfolio: optionalBooleanInput(input, "has_portfolio"), mobility_stage: enumStringInput(input, "mobility_stage", ["none", "transfer", "detasare", "restrangere"] as const), notes: optionalStringInput(input, "notes"), phone: optionalStringInput(input, "phone"), role_title: stringInput(input, "role_title"), school_year: stringInput(input, "school_year"), status: enumStringInput(input, "status", ["active", "on_leave", "vacant", "inactive"] as const) }; return dto as EducationRootCreateInputByDomain[D]; }
    case "evaluations": { const dto: EducationRootCreateInputByDomain["evaluations"] = { employee_code: stringInput(input, "employee_code"), evaluator_name: optionalStringInput(input, "evaluator_name"), finalized_on: optionalStringInput(input, "finalized_on"), full_name: stringInput(input, "full_name"), role_title: stringInput(input, "role_title"), school_year: stringInput(input, "school_year"), score: optionalNumberInput(input, "score"), status: stringInput(input, "status"), summary: optionalStringInput(input, "summary") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "declarations": { const dto: EducationRootCreateInputByDomain["declarations"] = { declaration_type: stringInput(input, "declaration_type"), employee_code: stringInput(input, "employee_code"), full_name: stringInput(input, "full_name"), school_year: stringInput(input, "school_year"), status: stringInput(input, "status"), submitted_on: stringInput(input, "submitted_on"), summary: optionalStringInput(input, "summary"), valid_until: optionalStringInput(input, "valid_until") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "mobility": { const dto: EducationRootCreateInputByDomain["mobility"] = { destination_school: optionalStringInput(input, "destination_school"), employee_code: stringInput(input, "employee_code"), full_name: stringInput(input, "full_name"), notes: optionalStringInput(input, "notes"), request_type: stringInput(input, "request_type"), reviewed_by: optionalStringInput(input, "reviewed_by"), school_year: stringInput(input, "school_year"), source_school: optionalStringInput(input, "source_school"), stage: stringInput(input, "stage"), status: stringInput(input, "status"), submitted_on: stringInput(input, "submitted_on") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "merit": { const dto: EducationRootCreateInputByDomain["merit"] = { category: stringInput(input, "category"), committee_name: optionalStringInput(input, "committee_name"), decision_date: stringInput(input, "decision_date"), full_name: stringInput(input, "full_name"), funded: optionalBooleanInput(input, "funded"), notes: optionalStringInput(input, "notes"), role_title: stringInput(input, "role_title"), school_year: stringInput(input, "school_year"), score: optionalNumberInput(input, "score"), status: stringInput(input, "status") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "portfolios": { const dto: EducationRootCreateInputByDomain["portfolios"] = { authenticity_declared: optionalBooleanInput(input, "authenticity_declared"), consent_captured: optionalBooleanInput(input, "consent_captured"), custodian: optionalStringInput(input, "custodian"), last_updated_on: stringInput(input, "last_updated_on"), notes: optionalStringInput(input, "notes"), owner_name: stringInput(input, "owner_name"), owner_personnel_id: stringInput(input, "owner_personnel_id"), owner_role: stringInput(input, "owner_role"), owner_user_id: stringInput(input, "owner_user_id"), school_year: stringInput(input, "school_year"), section_count: optionalNumberInput(input, "section_count"), status: stringInput(input, "status"), transfer_status: stringInput(input, "transfer_status") }; return dto as EducationRootCreateInputByDomain[D]; }
    case "compliance": { const dto: EducationRootCreateInputByDomain["compliance"] = { anonymization_status: enumStringInput(input, "anonymization_status", ["necesara", "finalizata", "nu_este_necesara"] as const), domain: enumStringInput(input, "domain", ["guvernanta", "documente_manageriale", "portofolii", "regulamente", "conformitate"] as const), entity_label: stringInput(input, "entity_label"), entity_type: enumStringInput(input, "entity_type", ["hotarare", "proces_verbal", "procedura_portofoliu", "rof", "roi", "pdi_pas", "raport", "anunt"] as const), mandatory: optionalBooleanInput(input, "mandatory"), notes: optionalStringInput(input, "notes"), publication_channel: enumStringInput(input, "publication_channel", ["site_public", "avizier", "intranet", "registratura"] as const), publication_status: enumStringInput(input, "publication_status", ["pregatit", "publicat", "retras"] as const), published_on: optionalStringInput(input, "published_on"), reviewed_by: optionalStringInput(input, "reviewed_by") }; if (dto.publication_status === "publicat" && !dto.published_on) throw new Error("education_required_published_on"); return dto as EducationRootCreateInputByDomain[D]; }
  }
}

/** Portfolio PATCH has a deliberately smaller allow-list than its lifecycle record. */
export function updateInputForDomain<D extends EducationRecordsDomain>(domain: D, input: EducationRecordInput): EducationRootUpdateInputByDomain[D] {
  if (domain !== "portfolios") return createInputForDomain(domain, input) as EducationRootUpdateInputByDomain[D];
  const dto: EducationRootUpdateInputByDomain["portfolios"] = { authenticity_declared: optionalBooleanInput(input, "authenticity_declared"), consent_captured: optionalBooleanInput(input, "consent_captured"), custodian: optionalStringInput(input, "custodian"), last_updated_on: stringInput(input, "last_updated_on"), notes: optionalStringInput(input, "notes"), owner_name: stringInput(input, "owner_name"), owner_personnel_id: optionalStringInput(input, "owner_personnel_id"), owner_role: stringInput(input, "owner_role"), owner_user_id: optionalStringInput(input, "owner_user_id"), school_year: stringInput(input, "school_year"), section_count: optionalNumberInput(input, "section_count"), status: stringInput(input, "status"), transfer_status: stringInput(input, "transfer_status") };
  return dto as EducationRootUpdateInputByDomain[D];
}
function DomainRecordsPage({
  api,
  area,
  canManage,
  canManageRecord,
  canManageRelation,
  canVerifyPortfolio = false,
  canVerifyPortfolioRecord,
  canManageSchoolPortfolios = false,
  canManageSchoolPortfolioRecord,
  portfolioTransferApi,
  canSendPortfolioTransfer = false,
  canReceivePortfolioTransfer = false,
  canSendPortfolioTransferRecord,
  canReceivePortfolioTransferRecord,
  valorificationClient,
  canReadPortfolioValorification = false,
  canManagePortfolioValorification = false,
  canManagePortfolioValorificationRecord,
}: {
  api: EducationApi;
  area: EducationArea;
  canManage: boolean;
  canManageRecord?: (recordID: string) => boolean;
  canManageRelation?: (relation: RelatedConfig, recordID: string) => boolean;
  canVerifyPortfolio?: boolean;
  canVerifyPortfolioRecord?: (recordID: string) => boolean;
  canManageSchoolPortfolios?: boolean;
  canManageSchoolPortfolioRecord?: (recordID: string) => boolean;
  portfolioTransferApi?: IntertenantPortfolioTransferApi;
  canSendPortfolioTransfer?: boolean;
  canReceivePortfolioTransfer?: boolean;
  canSendPortfolioTransferRecord?: (recordID: string) => boolean;
  canReceivePortfolioTransferRecord?: (recordID: string) => boolean;
  valorificationClient?: ContractClient;
  canReadPortfolioValorification?: boolean;
  canManagePortfolioValorification?: boolean;
  canManagePortfolioValorificationRecord?: (recordID: string) => boolean;
}) {
  const navigate = useNavigate();
  const domain = area.id as EducationRecordsDomain;
  const load = useMemo(
    () =>
      (
        q: string,
        page = 1,
        pageSize = 20,
        sort?: { field?: string; direction?: "asc" | "desc" },
        filters?: Record<string, string>,
      ) =>
        api.records(domain, {
          q,
          page,
          pageSize,
          sort: sort?.field,
          direction: sort?.direction,
          filters,
        }),
    [api, domain],
  );
  const [editing, setEditing] = useState<{
    id?: string;
    input: EducationRecordInput;
  }>();
  const [detail, setDetail] = useState<EducationRecord>();
  const [error, setError] = useState<string>();
  const [refresh, setRefresh] = useState(0);
  const [pendingDelete, setPendingDelete] = useState<string>();
  const [selectedRecordId, setSelectedRecordId] = useState<string>();
  const [portfolioLifecycle, setPortfolioLifecycle] = useState<{
    record: EducationRecord;
    kind: "cessation" | "legal_hold";
    date: string;
    reason: string;
    active: boolean;
  }>();
  const fields = domainFields[domain];
  const metadata = domainMetadata[domain];
  const action = async (fn: () => Promise<void>) => {
    setError(undefined);
    try {
      await fn();
      setEditing(undefined);
      setDetail(undefined);
      setRefresh((value) => value + 1);
    } catch {
      setError(
        "Operația nu a putut fi finalizată. Verificați datele și drepturile de acces.",
      );
    }
  };
  const wrappedLoad = useMemo(
    () =>
      (
        q: string,
        page = 1,
        pageSize = 20,
        sort?: { field?: string; direction?: "asc" | "desc" },
        filters?: Record<string, string>,
      ) =>
        load(q, page, pageSize, sort, filters),
    [load, refresh],
  );
  return (
    <div className="flex flex-col gap-3">
      {error && (
        <Message.Root severity="error">
          <Message.Content>
            <Message.Text>{error}</Message.Text>
          </Message.Content>
        </Message.Root>
      )}
      {metadata && <EducationMetadata api={api} resources={metadata} />}
      <EducationListPanel<EducationRecord>
        title={area.label}
        description={area.description}
        load={wrappedLoad}
        emptyMessage="Nu există înregistrări care corespund filtrului ales."
        filterableFields={domainListCapabilities[domain].filterableFields}
        sortableFields={domainListCapabilities[domain].sortableFields}
        columns={[
          ...fields.map((field) => ({
            field: field.key,
            header: field.label,
            render: (item: EducationRecord) => {
              const value = item[field.key];
              if (field.kind === "boolean") {
                return <Tag value={value ? "Da" : "Nu"} severity={value ? "success" : "secondary"} />;
              }
              if (field.key.includes("status")) {
                return <Tag value={String(value ?? "—")} severity="secondary" />;
              }
              return displayRecord(item, [field.key]);
            },
          })),
          {
            header: "Acțiuni",
            action: true,
            render: (item) => {
              const rowCanManage = canManage || Boolean(canManageRecord?.(item.id));
              const rowCanVerifyPortfolio = canVerifyPortfolio || Boolean(canVerifyPortfolioRecord?.(item.id));
              const rowCanManageSchoolPortfolio = canManageSchoolPortfolios || Boolean(canManageSchoolPortfolioRecord?.(item.id));
              return (
              <SchoolRowActionMenu
                actions={[
                  {
                    label: "Detalii",
                    icon: "pi pi-eye",
                    onSelect: () =>
                    void api
                      .recordDetail(domain, item.id)
                      .then((value) => {
                        setDetail(value);
                        setSelectedRecordId(item.id);
                      })
                      .catch(() => setError("Detaliul nu a putut fi încărcat.")),
                  },
                  ...(supportsPdf(domain)
                    ? [{
                        label: "PDF",
                        icon: "pi pi-file-pdf",
                        onSelect: () =>
                      viewPdf(api.recordPdf(domain, item.id), () =>
                        setError("PDF-ul nu a putut fi generat."),
                      ),
                      }]
                    : []),
                  ...(rowCanManage
                    ? [{
                        label: "Editează",
                        icon: "pi pi-pencil",
                        onSelect: () =>
                      setEditing({
                        id: item.id,
                        input: inputFromRecord(item, fields),
                      }),
                      }, {
                        label: "Șterge",
                        icon: "pi pi-trash",
                        severity: "danger" as const,
                        onSelect: () => setPendingDelete(item.id),
                      }]
                    : []),
                  ...(domain === "portfolios" && (rowCanManage || rowCanManageSchoolPortfolio)
                    ? [{
                        label: "Regenerare opis",
                        icon: "pi pi-refresh",
                        onSelect: () => void action(async () => {
                          await api.command("portfolio-opis-regenerate", item.id);
                        }),
                      }, {
                        label: "Solicită completări",
                        icon: "pi pi-replay",
                        severity: "warn" as const,
                        disabled: String(item.status ?? "") !== "submitted",
                        onSelect: () => void action(async () => {
                          await api.command("portfolio-return", item.id);
                        }),
                      }]
                    : []),
                  ...(domain === "portfolios" && rowCanVerifyPortfolio
                    ? [{
                        label: "Validează portofoliul",
                        icon: "pi pi-check-circle",
                        severity: "success" as const,
                        disabled: String(item.status ?? "") !== "submitted",
                        onSelect: () => void action(async () => {
                          await api.command("portfolio-verify", item.id);
                        }),
                      }]
                    : []),
                  ...(domain === "portfolios" && rowCanManageSchoolPortfolio
                    ? [{
                        label: "Înregistrează încetarea activității",
                        icon: "pi pi-calendar-times",
                        onSelect: () => setPortfolioLifecycle({ record: item, kind: "cessation", date: new Date().toISOString().slice(0, 10), reason: "", active: false }),
                      }, {
                        label: Boolean(item.legal_hold_active) ? "Ridică blocarea juridică" : "Aplică blocare juridică",
                        icon: "pi pi-lock",
                        severity: "warn" as const,
                        onSelect: () => setPortfolioLifecycle({ record: item, kind: "legal_hold", date: "", reason: "", active: !Boolean(item.legal_hold_active) }),
                      }]
                    : []),
                ]}
              />
              );
            },
          },
        ]}
        onAdd={
          canManage
            ? () => {
                const wizardRoute = domainWizardRoutes[domain];
                if (wizardRoute) {
                  navigate(wizardRoute);
                  return;
                }
                setEditing({ input: inputFromRecord(undefined, fields) });
              }
            : undefined
        }
        addLabel="înregistrare"
      />
      <RecordFormDialog
        open={editing}
        title={`${editing?.id ? "Editează" : "Adaugă"} — ${area.label}`}
        fields={fields}
        onClose={() => setEditing(undefined)}
        onChange={(input) =>
          setEditing((current) => (current ? { ...current, input } : current))
        }
        onSave={() =>
          editing &&
          action(async () => {
            if (editing.id) {
              await api.updateRecord(domain, editing.id, updateInputForDomain(domain, editing.input));
            } else {
              await api.createRecord(domain, createInputForDomain(domain, editing.input));
            }
          })
        }
      />
      <RecordDetailDialog
        record={detail}
        title={area.label}
        fields={fields}
        onClose={() => setDetail(undefined)}
      />
      <DeleteDialog
        open={pendingDelete}
        onClose={() => setPendingDelete(undefined)}
        onConfirm={() =>
          pendingDelete &&
          action(async () => {
            await api.deleteRecord(domain, pendingDelete);
            setPendingDelete(undefined);
          })
        }
      />
      <Dialog.Root open={Boolean(portfolioLifecycle)} onOpenChange={(event: { value?: boolean }) => !event.value && setPortfolioLifecycle(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>{portfolioLifecycle?.kind === "cessation" ? "Înregistrează încetarea activității" : portfolioLifecycle?.active ? "Aplică blocare juridică" : "Ridică blocarea juridică"}</Dialog.Title><Dialog.Close aria-label="Închide operația de ciclu de viață" /></Dialog.Header><Dialog.Content>{portfolioLifecycle && <div className="flex flex-col gap-3">{portfolioLifecycle.kind === "cessation" && <label className="flex flex-col gap-1"><span>Data încetării *</span><InputText type="date" value={portfolioLifecycle.date} onChange={(event: ChangeEvent<HTMLInputElement>) => setPortfolioLifecycle((current) => current ? { ...current, date: event.target.value } : current)} /></label>}{portfolioLifecycle.kind === "legal_hold" && <label className="flex items-center gap-2"><Checkbox.Root checked={portfolioLifecycle.active} disabled><Checkbox.Box><Checkbox.Indicator /></Checkbox.Box></Checkbox.Root><span>{portfolioLifecycle.active ? "Blocarea juridică va fi activată." : "Blocarea juridică va fi ridicată."}</span></label>}<label className="flex flex-col gap-1"><span>Motiv {portfolioLifecycle.kind === "cessation" || portfolioLifecycle.active ? "*" : ""}</span><Textarea value={portfolioLifecycle.reason} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setPortfolioLifecycle((current) => current ? { ...current, reason: event.target.value } : current)} /></label></div>}</Dialog.Content><Dialog.Footer><div className="flex justify-end gap-2"><Button variant="outlined" severity="secondary" onClick={() => setPortfolioLifecycle(undefined)}>Renunță</Button><Button disabled={!portfolioLifecycle || (portfolioLifecycle.kind === "cessation" && (!portfolioLifecycle.date || !portfolioLifecycle.reason.trim())) || (portfolioLifecycle.kind === "legal_hold" && portfolioLifecycle.active && !portfolioLifecycle.reason.trim())} onClick={() => portfolioLifecycle && void action(async () => { if (portfolioLifecycle.kind === "cessation") await api.recordPortfolioCessation(portfolioLifecycle.record.id, { activity_ceased_on: portfolioLifecycle.date, reason: portfolioLifecycle.reason }); else await api.setPortfolioLegalHold(portfolioLifecycle.record.id, { active: portfolioLifecycle.active, reason: portfolioLifecycle.reason }); setPortfolioLifecycle(undefined); })}>Confirmă</Button></div></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
      {selectedRecordId && hasDomainRelations(domain) && (
        <GovernanceMeetingRelations
          api={api}
          meetingId={selectedRecordId}
          canManage={(relation) => Boolean(canManageRelation?.(relation, selectedRecordId))}
          title={`${area.label} — operațiuni dosar`}
          relations={domainRelations[domain] ?? []}
        />
      )}
      {domain === "portfolios" && selectedRecordId && (
        <PortfolioRelationsPanel api={api} recordID={selectedRecordId} canManage={canManageSchoolPortfolios || Boolean(canManageSchoolPortfolioRecord?.(selectedRecordId))} />
      )}
      {selectedRecordId && domainDetailMetadata[domain] && (
        <EducationMetadata
          api={api}
          resources={domainDetailMetadata[domain]}
          parentID={selectedRecordId}
        />
      )}
      {domain === "portfolios" && selectedRecordId && portfolioTransferApi && (
        canSendPortfolioTransfer || canReceivePortfolioTransfer ||
        Boolean(canSendPortfolioTransferRecord?.(selectedRecordId)) ||
        Boolean(canReceivePortfolioTransferRecord?.(selectedRecordId))
      ) && (
        <PortfolioIntertenantTransfer
          portfolioID={selectedRecordId}
          api={portfolioTransferApi}
          capabilities={{
            send: canSendPortfolioTransfer || Boolean(canSendPortfolioTransferRecord?.(selectedRecordId)),
            receive: canReceivePortfolioTransfer || Boolean(canReceivePortfolioTransferRecord?.(selectedRecordId)),
          }}
        />
      )}
      {domain === "portfolios" && selectedRecordId && valorificationClient && canReadPortfolioValorification && (
        <PortfolioValorificationPackageManager
          portfolioID={selectedRecordId}
          client={valorificationClient}
          capabilities={{
            read: canReadPortfolioValorification,
            manage: canManagePortfolioValorification || Boolean(canManagePortfolioValorificationRecord?.(selectedRecordId)),
          }}
        />
      )}
    </div>
  );
}

const domainMetadata: Partial<Record<EducationRecordsDomain, EducationMetadataResource[]>> = {
  decisions: [
    "decisions-dashboard", "decisions-filters",
  ],
  managerial: [
    "managerial-dashboard", "managerial-filters",
  ],
  regulations: [
    "regulations-dashboard", "regulations-filters",
  ],
  personnel: [
    "personnel-dashboard", "personnel-filters",
  ],
  evaluations: [
    "evaluations-dashboard", "evaluations-filters",
  ],
  declarations: [
    "declarations-dashboard", "declarations-filters",
  ],
  mobility: [
    "mobility-dashboard", "mobility-filters",
  ],
  merit: [
    "merit-dashboard", "merit-filters",
  ],
  portfolios: [
    "portfolios-dashboard", "portfolios-filters",
  ],
};
const domainDetailMetadata: Partial<Record<EducationRecordsDomain, EducationMetadataResource[]>> =
  {
    committees: ["committee-completeness"], managerial: ["managerial-portfolio-summary"],
    personnel: ["personnel-portfolio-dossier-summary"], portfolios: ["portfolio-transfer-summary"], regulations: ["regulation-procedural-summary"],
  };
function readable(value: string) {
  const labels: Record<string, string> = {
    school_year: "An școlar",
    committee: "Comisie",
    membership: "Componență",
    readiness: "Pregătire",
    active_members: "Membri activi",
    voting_members: "Membri cu drept de vot",
    member_names: "Membri",
    chairperson_covered: "Președinte desemnat",
    secretary_covered: "Secretar desemnat",
    ready_for_operation: "Pregătită pentru funcționare",
    blockers: "Blocaje",
  };
  if (labels[value]) return labels[value];
  return value
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

type DisplayableMetadataValue =
  | string
  | number
  | boolean
  | Array<string | number | boolean>;

function isDisplayableMetadataValue(
  value: unknown,
): value is DisplayableMetadataValue {
  return (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean" ||
    (Array.isArray(value) &&
      value.every(
        (entry) =>
          typeof entry === "string" ||
          typeof entry === "number" ||
          typeof entry === "boolean",
      ))
  );
}

export function metadataDisplayEntries(items: Record<string, unknown>[]) {
  const entries: Array<{
    key: string;
    label: string;
    value: DisplayableMetadataValue;
  }> = [];
  const visit = (value: unknown, path: string[], itemIndex: number) => {
    if (isDisplayableMetadataValue(value)) {
      entries.push({
        key: `${itemIndex}.${path.join(".")}`,
        label: path.map(readable).join(" · "),
        value,
      });
      return;
    }
    if (Array.isArray(value)) {
      value.forEach((child, index) => visit(child, [...path, String(index + 1)], itemIndex));
      return;
    }
    if (!value || typeof value !== "object") return;
    for (const [key, child] of Object.entries(value)) {
      if (key !== "id" && key !== "institution_id")
        visit(child, [...path, key], itemIndex);
    }
  };
  items.forEach((item, itemIndex) => visit(item, [], itemIndex));
  return entries;
}

function formatMetadataValue(value: DisplayableMetadataValue) {
  if (Array.isArray(value))
    return value.map((entry) => (typeof entry === "boolean" ? (entry ? "Da" : "Nu") : String(entry))).join(", ") || "—";
  if (typeof value === "boolean") return value ? "Da" : "Nu";
  return String(value);
}
export function EducationMetadata({
  api,
  resources,
  parentID,
}: {
  api: EducationApi;
  resources: EducationMetadataResource[];
  parentID?: string;
}) {
  const [items, setItems] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    void Promise.all(resources.map((resource) => api.metadata(resource, parentID)))
      .then(setItems)
      .catch(() =>
        setError("Indicatorii sau filtrele nu au putut fi încărcate."),
      );
  }, [api, parentID, resources]);
  if (error)
    return (
      <Message.Root severity="warn">
        <Message.Content>
          <Message.Text>{error}</Message.Text>
        </Message.Content>
      </Message.Root>
    );
  if (items.length === 0)
    return (
      <div className="flex justify-center p-4">
        <Spinner />
      </div>
    );
  const values = metadataDisplayEntries(items);
  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>Indicatori și filtre disponibile</Card.Title>
        <Card.Content>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {values.map(({ key, label, value }) => (
              <div key={key}>
                <strong>{label}</strong>
                <p>
                  {typeof value === "boolean" ? (
                    <Tag
                      value={value ? "Da" : "Nu"}
                      severity={value ? "success" : "secondary"}
                    />
                  ) : (
                    formatMetadataValue(value)
                  )}
                </p>
              </div>
            ))}
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}

export function RecordFormDialog({
  open,
  title,
  fields,
  onClose,
  onChange,
  onSave,
}: {
  open?: { id?: string; input: EducationRecordInput };
  title: string;
  fields: RecordField[];
  onClose: () => void;
  onChange: (input: EducationRecordInput) => void;
  onSave: () => void;
}) {
  const saveDisabled = Boolean(open && fields.some((field) => field.form !== false && isRecordFieldMissing(field, open.input)));
  const set = (field: RecordField, raw: string) =>
    onChange({
      ...(open?.input ?? {}),
      [field.key]:
        field.kind === "number"
          ? Number(raw)
          : field.kind === "boolean"
            ? raw === "true"
            : raw,
    });
  return (
    <Dialog.Root
      open={Boolean(open)}
      onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}
    >
      <Dialog.Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Popup>
            <Dialog.Header>
              <Dialog.Title>{title}</Dialog.Title>
              <Dialog.Close aria-label="Închide" />
            </Dialog.Header>
            <Dialog.Content>
              <div className="flex flex-col gap-3">
                {fields.filter((field) => field.form !== false).map((field) => (
                  <div className="flex flex-col gap-1" key={field.key}>
                    <span>{field.label}{isRecordFieldRequired(field, open?.input) ? " *" : ""}</span>
                    {field.kind === "select" && field.search && (
                      <InputText
                        aria-label={field.search.label}
                        value={field.search.value}
                        onChange={(event: ChangeEvent<HTMLInputElement>) =>
                          field.search?.onChange(event.target.value)
                        }
                      />
                    )}
                    {field.kind === "boolean" || field.kind === "select" ? (
                      <Select.Root
                        disabled={field.search?.loading}
                        value={
                          field.kind === "boolean"
                            ? String(Boolean(open?.input[field.key]))
                            : String(open?.input[field.key] ?? "")
                        }
                        options={
                          field.kind === "boolean"
                            ? [
                                { label: "Nu", value: "false" },
                                { label: "Da", value: "true" },
                              ]
                            : (field.options ?? [])
                        }
                        optionLabel="label"
                        optionValue="value"
                        onValueChange={(event: { value: unknown }) =>
                          set(field, String(event.value))
                        }
                      >
                        <Select.Trigger aria-label={field.label} aria-required={isRecordFieldRequired(field, open?.input)}>
                          <Select.Value />
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
                    ) : (
                      <InputText
                        aria-label={field.label}
                        required={isRecordFieldRequired(field, open?.input)}
                        type={
                          field.kind === "date"
                            ? "date"
                            : field.kind === "number"
                              ? "number"
                              : "text"
                        }
                        min={field.min ?? (field.kind === "number" && ["weekly_hours", "assumed_score", "max_score", "self_score", "reviewer_score", "final_score"].includes(field.key) ? 0 : undefined)}
                        max={field.max ?? (field.kind === "number" && ["assumed_score", "max_score", "self_score", "reviewer_score", "final_score"].includes(field.key) ? 100 : undefined)}
                        value={String(open?.input[field.key] ?? "")}
                        onChange={(event: ChangeEvent<HTMLInputElement>) =>
                          set(field, event.target.value)
                        }
                      />
                    )}
                  </div>
                ))}
              </div>
            </Dialog.Content>
            <Dialog.Footer>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outlined"
                  severity="secondary"
                  onClick={onClose}
                >
                  Renunță
                </Button>
                <Button disabled={saveDisabled} onClick={onSave}>Salvează</Button>
              </div>
            </Dialog.Footer>
          </Dialog.Popup>
        </Dialog.Positioner>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
function RecordDetailDialog({
  record,
  title,
  fields,
  onClose,
}: {
  record?: EducationRecord;
  title: string;
  fields: RecordField[];
  onClose: () => void;
}) {
  return (
    <Dialog.Root
      open={Boolean(record)}
      onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}
    >
      <Dialog.Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Popup>
            <Dialog.Header>
              <Dialog.Title>{title}</Dialog.Title>
              <Dialog.Close aria-label="Închide" />
            </Dialog.Header>
            <Dialog.Content>
              <dl className="grid gap-3 sm:grid-cols-2">
                {fields
                  .filter((field) => record?.[field.key] !== undefined)
                  .map((field) => (
                    <div key={field.key}>
                      <dt>{field.label}</dt>
                      <dd>{String(record?.[field.key])}</dd>
                    </div>
                  ))}
              </dl>
            </Dialog.Content>
          </Dialog.Popup>
        </Dialog.Positioner>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
function DeleteDialog({
  open,
  onClose,
  onConfirm,
}: {
  open?: string;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog.Root
      open={Boolean(open)}
      onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}
    >
      <Dialog.Portal>
        <Dialog.Backdrop />
        <Dialog.Positioner>
          <Dialog.Popup>
            <Dialog.Header>
              <Dialog.Title>Confirmă ștergerea</Dialog.Title>
              <Dialog.Close aria-label="Închide" />
            </Dialog.Header>
            <Dialog.Content>
              <p>Această înregistrare va fi ștearsă definitiv.</p>
            </Dialog.Content>
            <Dialog.Footer>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outlined"
                  severity="secondary"
                  onClick={onClose}
                >
                  Renunță
                </Button>
                <Button severity="danger" onClick={onConfirm}>
                  Șterge
                </Button>
              </div>
            </Dialog.Footer>
          </Dialog.Popup>
        </Dialog.Positioner>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function dashboardMetrics(value: Record<string, unknown>) {
  const labels: Record<string, string> = {
    total_meetings: "Ședințe de guvernanță",
    validated_portfolios: "Portofolii validate",
    contested_evaluations: "Evaluări contestate",
    managerial_dossiers: "Dosare manageriale",
    active_personnel: "Personal activ",
    overdue_publications: "Publicări restante",
    total_records: "Total înregistrări",
    review_records: "În revizuire",
    validated_records: "Validate",
  };
  return Object.entries(value).flatMap(([key, item]) => {
    if (["string", "number"].includes(typeof item))
      return [
        { label: labels[key] ?? readable(key), value: item as string | number },
      ];
    if (!item || typeof item !== "object" || Array.isArray(item)) return [];
    return Object.entries(item as Record<string, unknown>)
      .filter(([, nested]) => ["string", "number"].includes(typeof nested))
      .map(([nestedKey, nested]) => ({
        label: labels[nestedKey] ?? readable(nestedKey),
        value: nested as string | number,
      }));
  });
}

function DirectorDashboard({
  api,
  reports,
  endpoint,
  title,
}: {
  api: EducationApi;
  reports: boolean;
  endpoint?: EducationMetadataResource;
  title?: string;
}) {
  const [data, setData] = useState<Record<string, unknown>>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    void (endpoint ? api.metadata(endpoint) : api.directorCockpit())
      .then(setData)
      .catch(() => setError("Dashboard-ul nu a putut fi încărcat."));
  }, [api, endpoint]);
  if (error)
    return (
      <Message.Root severity="error">
        <Message.Content>
          <Message.Text>{error}</Message.Text>
        </Message.Content>
      </Message.Root>
    );
  if (!data)
    return (
      <div className="flex justify-center p-8">
        <Spinner />
      </div>
    );
  const entries = dashboardMetrics(data);
  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>
          {title ?? (reports ? "Rapoarte standard" : "Cockpit director")}
        </Card.Title>
        <Card.Content>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {entries.map(({ label, value }) => (
              <Card.Root key={label}>
                <Card.Body>
                  <Card.Title>{label}</Card.Title>
                  <Card.Content>
                    <strong>{String(value)}</strong>
                  </Card.Content>
                </Card.Body>
              </Card.Root>
            ))}
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}

function SchoolRoleDashboard({
  kind,
  permissions,
  client,
}: {
  kind: RoleCockpitKind;
  permissions: readonly string[];
  client: ContractClient;
}) {
  const permission = `education.cockpit.${kind}.read`;
  const api = useMemo(() => createRoleCockpitsApi(client), [client]);
  const load = useMemo(() => roleCockpitLoader(api, kind), [api, kind]);
  return <RoleCockpit kind={kind} allowed={permissions.includes(permission)} load={load} />;
}

function Overview({
  api,
  canReadGovernance,
}: {
  api: EducationApi;
  canReadGovernance: boolean;
}) {
  const [dashboard, setDashboard] = useState<{
    total_meetings: number;
    scheduled_meetings: number;
    held_meetings: number;
    published_meetings: number;
  }>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    if (!canReadGovernance) return;
    void api
      .governanceDashboard()
      .then((result) => setDashboard(result.stats))
      .catch(() => setError("Indicatorii nu au putut fi încărcați."));
  }, [api, canReadGovernance]);
  if (!canReadGovernance)
    return (
      <Message.Root severity="info">
        <Message.Content>
          <Message.Text>
            Selectați un domeniu disponibil din navigație. Indicatorii de
            guvernanță necesită dreptul dedicat.
          </Message.Text>
        </Message.Content>
      </Message.Root>
    );
  if (error)
    return (
      <Message.Root severity="error">
        <Message.Content>
          <Message.Text>{error}</Message.Text>
        </Message.Content>
      </Message.Root>
    );
  if (!dashboard)
    return (
      <div className="flex justify-center p-8">
        <Spinner />
      </div>
    );
  const stats = [
    ["Total ședințe", dashboard.total_meetings],
    ["Planificate", dashboard.scheduled_meetings],
    ["Desfășurate", dashboard.held_meetings],
    ["Publicate", dashboard.published_meetings],
  ];
  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map(([label, value]) => (
          <Card.Root key={String(label)}>
            <Card.Body>
              <Card.Title>{label}</Card.Title>
              <Card.Content>
                <strong>{value}</strong>
              </Card.Content>
            </Card.Body>
          </Card.Root>
        ))}
      </div>
      <EducationMetadata api={api} resources={["director-cockpit"]} />
      <EducationCatalogs api={api} />
    </div>
  );
}

function EducationCatalogs({ api }: { api: EducationApi }) {
  const [selected, setSelected] = useState("Taxonomii");
  return (
    <div className="flex flex-col gap-3">
      <nav aria-label="Cataloge educaționale" className="flex flex-wrap gap-2">
        {["Taxonomii", "Cerințe", "Secțiuni portofoliu"].map((label) => <Button key={label} size="small" variant={label === selected ? undefined : "outlined"} severity={label === selected ? undefined : "secondary"} onClick={() => setSelected(label)}>{label}</Button>)}
      </nav>
      {selected === "Taxonomii" && <TaxonomyCatalogPanel api={api} />}
      {selected === "Cerințe" && <EducationListPanel<EducationRequirement> title="Cerințe" description="Catalog legal în regim de consultare. Filtrarea permisă este numai după domeniu." load={(_query, page, pageSize, sort, filters) => api.educationRequirements({ page, pageSize, sort: sort?.field, direction: sort?.direction, domain: filters?.domain })} columns={[{ field: "domain", header: "Domeniu", render: (item) => item.domain }, { field: "code", header: "Cod", render: (item) => item.code }, { field: "title_ro", header: "Cerință", render: (item) => item.title_ro }, { field: "implementation_status", header: "Stare", render: (item) => <Tag value={item.implementation_status} severity={item.implementation_status === "implemented" ? "success" : "secondary"} /> }]} emptyMessage="Nu există cerințe pentru filtrul curent." />}
      {selected === "Secțiuni portofoliu" && <EducationListPanel<PortfolioSection> title="Secțiuni portofoliu" description="Catalog oficial în regim de consultare. Filtrarea permisă este numai după secțiune." load={(_query, page, pageSize, sort, filters) => api.portfolioSections({ page, pageSize, sort: sort?.field, direction: sort?.direction, sectionCode: filters?.section_code })} columns={[{ field: "section_code", header: "Secțiune", render: (item) => item.section_code }, { field: "component_code", header: "Componentă", render: (item) => item.component_code }, { field: "label_ro", header: "Denumire", render: (item) => item.label_ro }, { field: "required", header: "Obligatoriu", render: (item) => <Tag value={item.required ? "Da" : "Nu"} severity={item.required ? "info" : "secondary"} /> }]} emptyMessage="Nu există secțiuni pentru filtrul curent." />}
    </div>
  );
}

function TaxonomyCatalogPanel({ api }: { api: EducationApi }) {
  const [items, setItems] = useState<Record<string, Array<{ id: string; code: string; label_ro: string; active: boolean }>>>( {} );
  const [error, setError] = useState(false);
  useEffect(() => { let live = true; void api.taxonomyCatalog().then((result) => { if (live) setItems(result.items); }).catch(() => { if (live) setError(true); }); return () => { live = false; }; }, [api]);
  if (error) return <Message.Root severity="error"><Message.Content><Message.Text>Taxonomiile nu au putut fi încărcate.</Message.Text></Message.Content></Message.Root>;
  return <div className="flex flex-col gap-3">{Object.entries(items).map(([domain, entries]) => <Card.Root key={domain}><Card.Body><Card.Title>{domain}</Card.Title><Card.Content><ul className="m-0 flex list-none flex-col gap-1 p-0">{entries.map((entry) => <li key={entry.id}>{entry.code} — {entry.label_ro} {!entry.active && <Tag value="inactiv" severity="secondary" />}</li>)}</ul></Card.Content></Card.Body></Card.Root>)}{Object.keys(items).length === 0 && <Message.Root severity="info"><Message.Content><Message.Text>Nu există taxonomii disponibile.</Message.Text></Message.Content></Message.Root>}</div>;
}

type PortfolioRelationTab = "documents" | "checklist" | "opis" | "custody" | "reviews" | "transfers" | "valorifications";

type PortfolioRelationItem = { id: string; [key: string]: string | number | boolean | undefined };
type PortfolioRelationInput = Record<string, string | number | boolean | undefined>;

const portfolioRelationString = (input: PortfolioRelationInput, key: string) => String(input[key] ?? "").trim();
const portfolioRelationOptionalString = (input: PortfolioRelationInput, key: string) => {
  const value = portfolioRelationString(input, key);
  return value || undefined;
};
const portfolioRelationOptionalNumber = (input: PortfolioRelationInput, key: string) => {
  const value = portfolioRelationString(input, key);
  return value ? Number(value) : undefined;
};
const portfolioRelationOptionalBoolean = (input: PortfolioRelationInput, key: string) =>
  typeof input[key] === "boolean" ? input[key] : undefined;

function portfolioDocumentInput(input: PortfolioRelationInput): CreatePortfolioDocumentInput {
  return {
    added_on: portfolioRelationString(input, "added_on"),
    applicable_class: portfolioRelationString(input, "applicable_class"),
    authenticity_status: portfolioRelationString(input, "authenticity_status"),
    chronological_index: portfolioRelationOptionalNumber(input, "chronological_index"),
    component_code: portfolioRelationString(input, "component_code"),
    competencies: portfolioRelationString(input, "competencies").split(",").map((value) => value.trim()).filter(Boolean),
    description: portfolioRelationString(input, "description"),
    document_title: portfolioRelationString(input, "document_title"),
    evidence_type: portfolioRelationString(input, "evidence_type"),
    file_reference: portfolioRelationString(input, "file_reference"),
    issued_on: portfolioRelationString(input, "issued_on"),
    notes: portfolioRelationOptionalString(input, "notes"),
    section_code: portfolioRelationString(input, "section_code"),
    sensitive_data: portfolioRelationOptionalBoolean(input, "sensitive_data"),
    school_year: portfolioRelationString(input, "school_year"),
    source_scope: portfolioRelationString(input, "source_scope"),
    subject_discipline: portfolioRelationString(input, "subject_discipline"),
  };
}
function portfolioChecklistInput(input: PortfolioRelationInput): CreatePortfolioChecklistItemInput {
  return {
    checked_by: portfolioRelationOptionalString(input, "checked_by"),
    document_count: portfolioRelationOptionalNumber(input, "document_count"),
    last_checked_on: portfolioRelationString(input, "last_checked_on"),
    mandatory: portfolioRelationOptionalBoolean(input, "mandatory"),
    notes: portfolioRelationOptionalString(input, "notes"),
    requirement_code: portfolioRelationString(input, "requirement_code"),
    requirement_label: portfolioRelationString(input, "requirement_label"),
    section_code: portfolioRelationString(input, "section_code"),
    source_scope: portfolioRelationString(input, "source_scope"),
    status: portfolioRelationString(input, "status"),
  };
}
function portfolioOpisInput(input: PortfolioRelationInput): CreatePortfolioOpisEntryInput {
  return {
    checked_by: portfolioRelationOptionalString(input, "checked_by"),
    checked_on: portfolioRelationString(input, "checked_on"),
    chronological_index: portfolioRelationOptionalNumber(input, "chronological_index"),
    component_code: portfolioRelationString(input, "component_code"),
    document_reference: portfolioRelationString(input, "document_reference"),
    entry_title: portfolioRelationString(input, "entry_title"),
    included_in_transfer: portfolioRelationOptionalBoolean(input, "included_in_transfer"),
    notes: portfolioRelationOptionalString(input, "notes"),
    section_code: portfolioRelationString(input, "section_code"),
    source_scope: portfolioRelationString(input, "source_scope"),
  };
}
function portfolioCustodyInput(input: PortfolioRelationInput): CreatePortfolioCustodyEventInput {
  return {
    access_mode: portfolioRelationString(input, "access_mode"),
    access_reason: portfolioRelationString(input, "access_reason"),
    ended_on: portfolioRelationOptionalString(input, "ended_on"),
    event_type: portfolioRelationString(input, "event_type"),
    holder_name: portfolioRelationString(input, "holder_name"),
    holder_role: portfolioRelationString(input, "holder_role"),
    location_label: portfolioRelationString(input, "location_label"),
    notes: portfolioRelationOptionalString(input, "notes"),
    sensitive_data_access: portfolioRelationOptionalBoolean(input, "sensitive_data_access"),
    started_on: portfolioRelationString(input, "started_on"),
  };
}
function portfolioReviewInput(input: PortfolioRelationInput): CreatePortfolioReviewEventInput {
  return {
    compliance_score: portfolioRelationOptionalNumber(input, "compliance_score"),
    missing_documents: portfolioRelationOptionalNumber(input, "missing_documents"),
    notes: portfolioRelationOptionalString(input, "notes"),
    outcome: portfolioRelationString(input, "outcome"),
    review_stage: portfolioRelationString(input, "review_stage"),
    reviewed_on: portfolioRelationString(input, "reviewed_on"),
    reviewer_name: portfolioRelationString(input, "reviewer_name"),
  };
}

type PortfolioRelationManagerConfig = {
  title: string;
  itemLabel: string;
  fields: RecordField[];
  /** Header filters are limited to the backend allowlist for this relation. */
  filterableFields: readonly string[];
  /** Exact server-declared sort allowlist; no browser-side sorting. */
  sortableFields: readonly string[];
  columns: EducationListPanelProps<PortfolioRelationItem>["columns"];
  readOnlyActions?: (item: PortfolioRelationItem) => SchoolRowAction[];
  load: (page: number, pageSize: number, sort?: { field?: string; direction?: "asc" | "desc" }, filters?: Record<string, string>) => Promise<EducationPage<{ id: string }>>;
  create: (input: PortfolioRelationInput) => Promise<{ id: string }>;
  update: (id: string, input: PortfolioRelationInput) => Promise<{ id: string }>;
  remove: (id: string) => Promise<void>;
};

type PortfolioRelationManagerBaseConfig = Omit<PortfolioRelationManagerConfig, "filterableFields" | "load"> & {
  filterField: string;
  load: (page: number, pageSize: number, sort?: { field?: string; direction?: "asc" | "desc" }, filter?: string) => Promise<EducationPage<{ id: string }>>;
};

function relationInputFromItem(item: PortfolioRelationItem): PortfolioRelationInput {
  const input: PortfolioRelationInput = {};
  for (const [key, value] of Object.entries(item) as Array<[string, unknown]>) {
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") input[key] = value;
    else if (Array.isArray(value) && value.every((entry) => typeof entry === "string")) input[key] = value.join(", ");
  }
  return input;
}

function PortfolioRelationManager({ config, canManage }: { config: PortfolioRelationManagerConfig; canManage: boolean }) {
  const [editing, setEditing] = useState<{ id?: string; input: PortfolioRelationInput }>();
  const [detail, setDetail] = useState<PortfolioRelationItem>();
  const [pendingDelete, setPendingDelete] = useState<string>();
  const [revision, setRevision] = useState(0);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const action = async (run: () => Promise<void>) => {
    setSaving(true); setError(undefined);
    try { await run(); setRevision((current) => current + 1); setEditing(undefined); }
    catch { setError("Operația nu a putut fi finalizată. Verificați datele și drepturile de acces."); }
    finally { setSaving(false); }
  };
  const load = useCallback(async (_query: string, page = 1, pageSize = 20, sort?: { field?: string; direction?: "asc" | "desc" }, filters?: Record<string, string>) => config.load(page, pageSize, sort, filters), [config]);
  const columns: EducationListPanelProps<PortfolioRelationItem>["columns"] = [
    ...config.columns,
    { header: "Acțiuni", action: true, render: (item) => <SchoolRowActionMenu actions={[
      { label: "Detalii", icon: "pi pi-eye", onSelect: () => setDetail(item) },
      ...(config.readOnlyActions?.(item) ?? []),
      ...(canManage ? [{ label: "Editează", icon: "pi pi-pencil", onSelect: () => setEditing({ id: item.id, input: relationInputFromItem(item) }) }, { label: "Șterge", icon: "pi pi-trash", severity: "danger" as const, onSelect: () => setPendingDelete(item.id) }] : []),
    ]} /> },
  ];
  return <div className="flex flex-col gap-3">
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    <EducationListPanel key={revision} title={config.title} description="Date contractuale ale portofoliului; filtrarea, sortarea și paginarea sunt executate de server." emptyMessage="Nu există înregistrări pentru filtrul curent." load={load} columns={columns} onAdd={canManage ? () => setEditing({ input: {} }) : undefined} addLabel={config.itemLabel} filterableFields={config.filterableFields} sortableFields={config.sortableFields} />
    <RecordFormDialog open={editing ? { id: editing.id, input: editing.input as EducationRecordInput } : undefined} title={`${editing?.id ? "Editează" : "Adaugă"} — ${config.title}`} fields={config.fields} onClose={() => !saving && setEditing(undefined)} onChange={(input) => setEditing((current) => current ? { ...current, input: input as PortfolioRelationInput } : current)} onSave={() => editing && void action(async () => { if (editing.id) await config.update(editing.id, editing.input); else await config.create(editing.input); })} />
    <PortfolioRelationDetailDialog title={config.title} record={detail} fields={config.fields} onClose={() => setDetail(undefined)} />
    <DeleteDialog open={pendingDelete} onClose={() => !saving && setPendingDelete(undefined)} onConfirm={() => pendingDelete && void action(async () => { await config.remove(pendingDelete); setPendingDelete(undefined); })} />
  </div>;
}

function PortfolioRelationDetailDialog({ title, record, fields, onClose }: { title: string; record?: PortfolioRelationItem; fields: RecordField[]; onClose: () => void }) {
  return <Dialog.Root open={Boolean(record)} onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>{title}</Dialog.Title><Dialog.Close aria-label="Închide detaliile" /></Dialog.Header><Dialog.Content><dl className="grid gap-3 sm:grid-cols-2">{fields.filter((field) => record?.[field.key as keyof PortfolioRelationItem] !== undefined).map((field) => <div key={field.key}><dt>{field.label.replace(" *", "")}</dt><dd>{String(record?.[field.key as keyof PortfolioRelationItem] ?? "")}</dd></div>)}</dl></Dialog.Content></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

function PortfolioHistoryDetailDialog({ title, record, onClose }: { title: string; record?: PortfolioRelationItem; onClose: () => void }) {
  return <Dialog.Root open={Boolean(record)} onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,44rem)]"><Dialog.Header><Dialog.Title>{title}</Dialog.Title><Dialog.Close aria-label="Închide detaliile" /></Dialog.Header><Dialog.Content><dl className="grid gap-3 sm:grid-cols-2">{Object.entries(record ?? {}).filter(([key]) => key !== "id").map(([key, value]) => <div key={key}><dt>{readable(key)}</dt><dd className="break-words">{String(value ?? "—")}</dd></div>)}</dl></Dialog.Content></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}

const portfolioDocumentRelationFields: RecordField[] = [
  { key: "document_title", label: "Titlu", required: true }, { key: "description", label: "Descriere pedagogică", required: true }, { key: "school_year", label: "An școlar", required: true }, { key: "subject_discipline", label: "Disciplina", required: true }, { key: "applicable_class", label: "Clasa aplicabilă", required: true }, { key: "competencies", label: "Competențe (separate prin virgulă)", required: true }, { key: "evidence_type", label: "Tip dovadă", required: true }, { key: "section_code", label: "Secțiune", required: true }, { key: "component_code", label: "Componentă", required: true }, { key: "source_scope", label: "Domeniu sursă", required: true }, { key: "authenticity_status", label: "Autenticitate", required: true }, { key: "issued_on", label: "Data emiterii", kind: "date", required: true }, { key: "added_on", label: "Data adăugării", kind: "date", required: true }, { key: "chronological_index", label: "Ordine cronologică", kind: "number" }, { key: "file_reference", label: "Referință arhivă", required: true }, { key: "sensitive_data", label: "Conține date sensibile", kind: "boolean" }, { key: "notes", label: "Observații" }, { key: "archive_version_no", label: "Versiune arhivă", form: false }, { key: "archive_sha256", label: "Hash SHA-256 arhivă", form: false },
];
const portfolioDocumentRelationColumns: EducationListPanelProps<PortfolioRelationItem>["columns"] = [
  { field: "document_title", header: "Titlu", render: (item) => item.document_title }, { field: "description", header: "Descriere", render: (item) => item.description }, { field: "school_year", header: "An școlar", render: (item) => item.school_year }, { field: "subject_discipline", header: "Disciplină", render: (item) => item.subject_discipline }, { field: "applicable_class", header: "Clasă", render: (item) => item.applicable_class }, { field: "competencies", header: "Competențe", render: (item) => Array.isArray((item as Record<string, unknown>).competencies) ? ((item as Record<string, unknown>).competencies as string[]).join(", ") : String(item.competencies ?? "") }, { field: "evidence_type", header: "Tip dovadă", render: (item) => item.evidence_type }, { field: "archive_version_no", header: "Versiune arhivă", render: (item) => item.archive_version_no ? `v${item.archive_version_no}` : "—" },
];

/**
 * The managerial Portfolio projections deliberately select named generated
 * operations.  Transfers stay history-only here: their creation and state
 * transitions are handled by the dedicated inter-tenant workflow below.
 */
export function PortfolioRelationsPanel({ api, recordID, canManage = false }: { api: EducationApi; recordID: string; canManage?: boolean }) {
  const [tab, setTab] = useState<PortfolioRelationTab>("documents");
  const [historyDetail, setHistoryDetail] = useState<{ title: string; record: PortfolioRelationItem }>();
  const [documentVersion, setDocumentVersion] = useState<{ id: string; title: string }>();
  const tabs: Array<{ id: PortfolioRelationTab; label: string }> = [
    { id: "documents", label: "Documente" }, { id: "checklist", label: "Checklist" }, { id: "opis", label: "Opis" }, { id: "custody", label: "Custodie" }, { id: "reviews", label: "Revizuiri" }, { id: "transfers", label: "Istoric transferuri" }, { id: "valorifications", label: "Istoric valorificări" },
  ];
  const common = { description: "Date preluate din contractul Portfolio; filtrarea, sortarea și paginarea sunt executate de server.", emptyMessage: "Nu există înregistrări pentru filtrul curent." };
  const baseManagers: Record<Extract<PortfolioRelationTab, "documents" | "checklist" | "opis" | "custody" | "reviews">, PortfolioRelationManagerBaseConfig> = {
    documents: { title: "Documente", itemLabel: "document", filterField: "section_code", fields: [{ key: "document_title", label: "Titlu *", kind: "text" }, { key: "evidence_type", label: "Tip dovadă *", kind: "text" }, { key: "section_code", label: "Secțiune *", kind: "text" }, { key: "component_code", label: "Componentă *", kind: "text" }, { key: "source_scope", label: "Domeniu sursă *", kind: "text" }, { key: "authenticity_status", label: "Autenticitate *", kind: "text" }, { key: "issued_on", label: "Data emiterii *", kind: "date" }, { key: "added_on", label: "Data adăugării *", kind: "date" }, { key: "chronological_index", label: "Ordine cronologică", kind: "number" }, { key: "file_reference", label: "Referință arhivă", kind: "text" }, { key: "sensitive_data", label: "Conține date sensibile", kind: "boolean" }, { key: "notes", label: "Observații", kind: "text" }], columns: [{ field: "document_title", header: "Titlu", render: (item) => item.document_title }, { field: "evidence_type", header: "Tip dovadă", render: (item) => item.evidence_type }, { field: "section_code", header: "Secțiune", render: (item) => item.section_code }, { field: "authenticity_status", header: "Autenticitate", render: (item) => <Tag value={item.authenticity_status} severity="secondary" /> }, { field: "issued_on", header: "Emis la", render: (item) => item.issued_on }], sortableFields: ["document_title", "evidence_type", "section_code", "authenticity_status", "issued_on"], load: (page, pageSize, sort, sectionCode) => api.portfolioDocuments(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, sectionCode }), create: async (input) => api.createPortfolioDocument(recordID, portfolioDocumentInput(input)), update: async (id, input) => api.updatePortfolioDocument(recordID, id, portfolioDocumentInput(input)), remove: (id) => api.deletePortfolioDocument(recordID, id) },
    checklist: { title: "Checklist", itemLabel: "cerință", filterField: "requirement_code", fields: [{ key: "requirement_code", label: "Cod cerință *", kind: "text" }, { key: "requirement_label", label: "Cerință *", kind: "text" }, { key: "section_code", label: "Secțiune *", kind: "text" }, { key: "source_scope", label: "Domeniu sursă *", kind: "text" }, { key: "status", label: "Stare *", kind: "text" }, { key: "last_checked_on", label: "Ultima verificare *", kind: "date" }, { key: "checked_by", label: "Verificat de", kind: "text" }, { key: "document_count", label: "Număr documente", kind: "number" }, { key: "mandatory", label: "Obligatoriu", kind: "boolean" }, { key: "notes", label: "Observații", kind: "text" }], columns: [{ field: "requirement_code", header: "Cod cerință", render: (item) => item.requirement_code }, { field: "requirement_label", header: "Cerință", render: (item) => item.requirement_label }, { field: "section_code", header: "Secțiune", render: (item) => item.section_code }, { field: "status", header: "Stare", render: (item) => <Tag value={item.status} severity="secondary" /> }, { field: "document_count", header: "Documente", render: (item) => String(item.document_count) }], sortableFields: ["requirement_code", "requirement_label", "section_code", "status", "document_count"], load: (page, pageSize, sort, requirementCode) => api.portfolioChecklist(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, requirementCode }), create: async (input) => api.createPortfolioChecklistItem(recordID, portfolioChecklistInput(input)), update: async (id, input) => api.updatePortfolioChecklistItem(recordID, id, portfolioChecklistInput(input)), remove: (id) => api.deletePortfolioChecklistItem(recordID, id) },
    opis: { title: "Opis", itemLabel: "poziție opis", filterField: "section_code", fields: [{ key: "section_code", label: "Secțiune *", kind: "text" }, { key: "component_code", label: "Componentă *", kind: "text" }, { key: "entry_title", label: "Titlu *", kind: "text" }, { key: "document_reference", label: "Referință document *", kind: "text" }, { key: "source_scope", label: "Domeniu sursă *", kind: "text" }, { key: "checked_on", label: "Data verificării *", kind: "date" }, { key: "checked_by", label: "Verificat de", kind: "text" }, { key: "chronological_index", label: "Ordine cronologică", kind: "number" }, { key: "included_in_transfer", label: "Inclus în transfer", kind: "boolean" }, { key: "notes", label: "Observații", kind: "text" }], columns: [{ field: "section_code", header: "Secțiune", render: (item) => item.section_code }, { field: "component_code", header: "Componentă", render: (item) => item.component_code }, { field: "entry_title", header: "Titlu", render: (item) => item.entry_title }, { field: "chronological_index", header: "Ordine", render: (item) => String(item.chronological_index) }, { field: "document_reference", header: "Referință", render: (item) => item.document_reference }], sortableFields: ["section_code", "component_code", "entry_title", "chronological_index", "document_reference"], load: (page, pageSize, sort, sectionCode) => api.portfolioOpis(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, sectionCode }), create: async (input) => api.createPortfolioOpisEntry(recordID, portfolioOpisInput(input)), update: async (id, input) => api.updatePortfolioOpisEntry(recordID, id, portfolioOpisInput(input)), remove: (id) => api.deletePortfolioOpisEntry(recordID, id) },
    custody: { title: "Custodie", itemLabel: "eveniment de custodie", filterField: "event_type", fields: [{ key: "event_type", label: "Tip eveniment *", kind: "text" }, { key: "holder_name", label: "Custode *", kind: "text" }, { key: "holder_role", label: "Rol custode *", kind: "text" }, { key: "location_label", label: "Locație *", kind: "text" }, { key: "access_mode", label: "Mod acces *", kind: "text" }, { key: "access_reason", label: "Motiv acces *", kind: "text" }, { key: "started_on", label: "Început *", kind: "date" }, { key: "ended_on", label: "Sfârșit", kind: "date" }, { key: "sensitive_data_access", label: "Acces date sensibile", kind: "boolean" }, { key: "notes", label: "Observații", kind: "text" }], columns: [{ field: "event_type", header: "Tip eveniment", render: (item) => item.event_type }, { field: "holder_name", header: "Custode", render: (item) => item.holder_name }, { field: "holder_role", header: "Rol custode", render: (item) => item.holder_role }, { field: "started_on", header: "Început", render: (item) => item.started_on }, { field: "ended_on", header: "Sfârșit", render: (item) => item.ended_on }], sortableFields: ["event_type", "holder_name", "holder_role", "started_on", "ended_on"], load: (page, pageSize, sort, eventType) => api.portfolioCustody(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, eventType }), create: async (input) => api.createPortfolioCustodyEvent(recordID, portfolioCustodyInput(input)), update: async (id, input) => api.updatePortfolioCustodyEvent(recordID, id, portfolioCustodyInput(input)), remove: (id) => api.deletePortfolioCustodyEvent(recordID, id) },
    reviews: { title: "Revizuiri", itemLabel: "revizuire", filterField: "review_code", fields: [{ key: "review_stage", label: "Etapă *", kind: "text" }, { key: "outcome", label: "Rezultat *", kind: "text" }, { key: "reviewer_name", label: "Evaluator *", kind: "text" }, { key: "reviewed_on", label: "Data revizuirii *", kind: "date" }, { key: "compliance_score", label: "Scor conformitate", kind: "number" }, { key: "missing_documents", label: "Documente lipsă", kind: "number" }, { key: "notes", label: "Observații", kind: "text" }], columns: [{ field: "review_code", header: "Cod", render: (item) => item.review_code }, { field: "review_stage", header: "Etapă", render: (item) => item.review_stage }, { field: "outcome", header: "Rezultat", render: (item) => item.outcome }, { field: "reviewer_name", header: "Evaluator", render: (item) => item.reviewer_name }, { field: "reviewed_on", header: "Data", render: (item) => item.reviewed_on }], sortableFields: ["review_code", "review_stage", "outcome", "reviewer_name", "reviewed_on"], load: (page, pageSize, sort, reviewCode) => api.portfolioReviews(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, reviewCode }), create: async (input) => api.createPortfolioReview(recordID, portfolioReviewInput(input)), update: async (id, input) => api.updatePortfolioReview(recordID, id, portfolioReviewInput(input)), remove: (id) => api.deletePortfolioReview(recordID, id) },
  };
  const managers: Record<Extract<PortfolioRelationTab, "documents" | "checklist" | "opis" | "custody" | "reviews">, PortfolioRelationManagerConfig> = {
    documents: { ...baseManagers.documents, fields: portfolioDocumentRelationFields, columns: portfolioDocumentRelationColumns, filterableFields: ["section_code", "component_code", "document_title", "description", "school_year", "subject_discipline", "applicable_class", "competencies", "evidence_type", "authenticity_status", "archive_version_no"], sortableFields: ["section_code", "component_code", "document_title", "description", "school_year", "subject_discipline", "applicable_class", "evidence_type", "authenticity_status", "archive_version_no"], load: (page, pageSize, sort, filters) => api.portfolioDocuments(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, sectionCode: filters?.section_code, componentCode: filters?.component_code, documentTitle: filters?.document_title, description: filters?.description, schoolYear: filters?.school_year, subjectDiscipline: filters?.subject_discipline, applicableClass: filters?.applicable_class, competencies: filters?.competencies, evidenceType: filters?.evidence_type, authenticityStatus: filters?.authenticity_status, archiveVersionNo: filters?.archive_version_no }), readOnlyActions: (item) => [{ label: "Istoric versiuni", icon: "pi pi-history", onSelect: () => setDocumentVersion({ id: item.id, title: String(item.document_title ?? "Document") }) }] },
    checklist: { ...baseManagers.checklist, filterableFields: ["requirement_code", "section_code", "status"], load: (page, pageSize, sort, filters) => api.portfolioChecklist(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, requirementCode: filters?.requirement_code, sectionCode: filters?.section_code, status: filters?.status }) },
    opis: { ...baseManagers.opis, filterableFields: ["section_code", "component_code", "entry_title", "chronological_index", "document_reference"], load: (page, pageSize, sort, filters) => api.portfolioOpis(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, sectionCode: filters?.section_code, componentCode: filters?.component_code, entryTitle: filters?.entry_title, chronologicalIndex: filters?.chronological_index, documentReference: filters?.document_reference }) },
    custody: { ...baseManagers.custody, filterableFields: ["event_type", "holder_name", "holder_role", "started_on", "ended_on"], load: (page, pageSize, sort, filters) => api.portfolioCustody(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, eventType: filters?.event_type, holderName: filters?.holder_name, holderRole: filters?.holder_role, startedOn: filters?.started_on, endedOn: filters?.ended_on }) },
    reviews: { ...baseManagers.reviews, filterableFields: ["review_code", "review_stage", "outcome", "reviewer_name"], load: (page, pageSize, sort, filters) => api.portfolioReviews(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, reviewCode: filters?.review_code, reviewStage: filters?.review_stage, outcome: filters?.outcome, reviewerName: filters?.reviewer_name }) },
  };
  return <Card.Root><Card.Body><Card.Title>Portofoliu — operațiuni dosar</Card.Title><Card.Content><div className="flex flex-col gap-3"><nav aria-label="Relații portofoliu" className="flex flex-wrap gap-2">{tabs.map((item) => <Button key={item.id} size="small" variant={tab === item.id ? undefined : "outlined"} severity={tab === item.id ? undefined : "secondary"} onClick={() => setTab(item.id)}>{item.label}</Button>)}</nav>
    {tab === "documents" && <PortfolioRelationManager config={managers.documents} canManage={canManage} />}
    {tab === "checklist" && <PortfolioRelationManager config={managers.checklist} canManage={canManage} />}
    {tab === "opis" && <PortfolioRelationManager config={managers.opis} canManage={canManage} />}
    {tab === "custody" && <PortfolioRelationManager config={managers.custody} canManage={canManage} />}
    {tab === "reviews" && <PortfolioRelationManager config={managers.reviews} canManage={canManage} />}
    {tab === "transfers" && <EducationListPanel title="Istoric transferuri" {...common} filterableFields={["transfer_code", "transfer_type", "status", "destination_institution", "handover_on"]} sortableFields={["transfer_code", "transfer_type", "status", "destination_institution"]} load={(_q, page, pageSize, sort, filters) => api.portfolioTransferHistory(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, transferCode: filters?.transfer_code, transferType: filters?.transfer_type, status: filters?.status, destinationInstitution: filters?.destination_institution, handoverOn: filters?.handover_on })} columns={[{ field: "transfer_code", header: "Cod", render: (item) => item.transfer_code }, { field: "transfer_type", header: "Tip", render: (item) => item.transfer_type }, { field: "status", header: "Stare", render: (item) => <Tag value={item.status} severity="secondary" /> }, { field: "destination_institution", header: "Instituție destinație", render: (item) => item.destination_institution }, { field: "handover_on", header: "Predare", render: (item) => item.handover_on }, { header: "Acțiuni", action: true, render: (item) => <SchoolRowActionMenu actions={[{ label: "Detalii", icon: "pi pi-eye", onSelect: () => setHistoryDetail({ title: "Detalii transfer", record: item as unknown as PortfolioRelationItem }) }]} /> }]} />}
    {tab === "valorifications" && <EducationListPanel title="Istoric valorificări" {...common} filterableFields={["valorification_code", "scope", "status", "target_institution", "started_on"]} sortableFields={["valorification_code", "scope", "status", "target_institution", "started_on"]} load={(_q, page, pageSize, sort, filters) => api.portfolioValorifications(recordID, { page, pageSize, sort: sort?.field, direction: sort?.direction, valorificationCode: filters?.valorification_code, scope: filters?.scope, status: filters?.status, targetInstitution: filters?.target_institution, startedOn: filters?.started_on })} columns={[{ field: "valorification_code", header: "Cod", render: (item) => item.valorification_code }, { field: "scope", header: "Domeniu", render: (item) => item.scope }, { field: "status", header: "Stare", render: (item) => <Tag value={item.status} severity="secondary" /> }, { field: "target_institution", header: "Instituție țintă", render: (item) => item.target_institution }, { field: "started_on", header: "Început", render: (item) => item.started_on }, { header: "Acțiuni", action: true, render: (item) => <SchoolRowActionMenu actions={[{ label: "Detalii", icon: "pi pi-eye", onSelect: () => setHistoryDetail({ title: "Detalii valorificare", record: item as unknown as PortfolioRelationItem }) }]} /> }]} />}
    <PortfolioHistoryDetailDialog title={historyDetail?.title ?? "Detalii"} record={historyDetail?.record} onClose={() => setHistoryDetail(undefined)} />
    {documentVersion && <PortfolioDocumentVersionHistoryDialog title={`Istoric versiuni — ${documentVersion.title}`} load={(query) => api.portfolioDocumentVersions(recordID, documentVersion.id, query)} onClose={() => setDocumentVersion(undefined)} />}
  </div></Card.Content></Card.Body></Card.Root>;
}

export interface EducationWorkspaceProps {
  api?: EducationApi;
  apiFetch?: AuthenticatedFetcher;
  institutionId?: string;
  permissions?: readonly string[];
  modules?: readonly EducationModule[];
  delegationApi?: EducationDelegationApi;
}

/**
 * Responsive Școală foundation. Routes are backend-authoritative; this module
 * intentionally renders only areas granted by both session context and module state.
 */
export function EducationWorkspace(props: EducationWorkspaceProps) {
  const auth = useAuth();
  const institutionId =
    props.institutionId ?? auth.session?.institution_id ?? "";
  const directPermissions = props.permissions ?? auth.session?.permissions ?? [];
  const modules = props.modules ?? auth.session?.modules ?? [];
  const fallbackApi = useMemo(
    () =>
      institutionId
        ? createEducationApi(props.apiFetch ?? auth.apiFetch)
        : undefined,
    [auth.apiFetch, institutionId, props.apiFetch],
  );
  const api = props.api ?? fallbackApi;
  const portfolioTransferApi = useMemo(
    () => createIntertenantPortfolioTransferApi(auth.apiClient),
    [auth.apiClient],
  );
  const delegationApi = useMemo(
    () => props.delegationApi ?? createEducationDelegationApi(auth.apiClient),
    [auth.apiClient, props.delegationApi],
  );
  // Delegation validity is evaluated by the backend for the current tenant and
  // subject. Do not re-create this decision from the paginated delegation
  // ledger or browser time.
  const activeDelegations = props.permissions ? [] : auth.educationGrants;
  const permissions = useMemo(() => {
    const granted = [...new Set([...directPermissions, ...activeDelegations.filter((item) => item.resource_type === "institution").map((item) => item.permission_code)])];
    return effectiveEducationPermissions(granted);
  }, [activeDelegations, directPermissions]);
  const allows = useCallback((permission: string, resourceType = "institution", resourceID?: string) => {
    if (props.permissions) {
      return educationPermissionAllows(
        directPermissions,
        [],
        permission,
        resourceType,
        resourceID,
      );
    }
    return resourceType === "institution"
      ? auth.canEducation(permission)
      : auth.canEducation(permission, { resourceType: resourceType as Exclude<typeof activeDelegations[number]["resource_type"], "institution">, resourceId: resourceID ?? "" });
  }, [activeDelegations, auth, directPermissions, props.permissions]);
  const location = useLocation();
  const areas = useMemo(
    () => visibleEducationAreas(permissions, modules),
    [modules, permissions],
  );
  const routeSegment = ([
    "governance", "decisions", "managerial", "regulations", "committees",
    "personnel", "evaluations", "declarations", "mobility", "merit",
    "portfolio", "compliance",
  ] as const).find((segment) => location.pathname.includes(`/${segment}`));
  const routeActive = routeSegment === "portfolio" ? "portfolios" : routeSegment ?? "overview";
  const roleCockpitKind: RoleCockpitKind | undefined = location.pathname.includes("/secretariat")
    ? "secretariat"
    : location.pathname.includes("/hr")
      ? "hr"
      : location.pathname.includes("/committee-cockpit")
        ? "committee"
        : location.pathname.includes("/inspector")
          ? "inspector"
          : undefined;
  const [active, setActive] = useState(routeActive);
  useEffect(() => {
    setActive((current) => {
      if (areas.some((area) => area.id === routeActive)) return routeActive;
      if (areas.some((area) => area.id === current)) return current;
      return areas[0]?.id ?? "overview";
    });
  }, [areas, routeActive]);

  if (!institutionId)
    return (
      <Message.Root severity="warn">
        <Message.Content>
          <Message.Text>
            Nu este selectată o instituție. Alegeți contextul instituției
            înainte de a accesa modulul Școală.
          </Message.Text>
        </Message.Content>
      </Message.Root>
    );
  if (!api)
    return (
      <Message.Root severity="error">
        <Message.Content>
          <Message.Text>
            Clientul autentificat pentru Școală nu este disponibil.
          </Message.Text>
        </Message.Content>
      </Message.Root>
    );
  if (areas.length === 0 && !roleCockpitKind)
    return (
      <Message.Root severity="warn">
        <Message.Content>
          <Message.Text>
            Nu aveți drepturi active pentru funcționalitățile Școală în această
            instituție.
          </Message.Text>
        </Message.Content>
      </Message.Root>
    );
  const current = areas.find((area) => area.id === active) as EducationArea;
  return (
    <section aria-label="Școală" className="flex flex-col gap-4">
      <Card.Root>
        <Card.Body>
          <Card.Title>Școală</Card.Title>
          <Card.Content>
            <div className="flex flex-col gap-3">
              <p>Operațiuni școlare pentru instituția curentă.</p>
              <nav aria-label="Domenii Școală" className="flex flex-wrap gap-2">
                {areas.map((area) => (
                  <Button
                    key={area.id}
                    variant={area.id === active ? undefined : "outlined"}
                    severity={area.id === active ? undefined : "secondary"}
                    onClick={() => setActive(area.id)}
                  >
                    <i className={area.icon} aria-hidden="true" />
                    {area.label}
                  </Button>
                ))}
              </nav>
            </div>
          </Card.Content>
        </Card.Body>
      </Card.Root>
      {location.pathname.includes("/dashboard/director") ||
      location.pathname.endsWith("/reports") ? (
        <DirectorDashboard
          api={api}
          reports={location.pathname.endsWith("/reports")}
        />
      ) : location.pathname.includes("/teacher") ? (
        <DirectorDashboard
          api={api}
          reports={false}
          endpoint="portfolios-dashboard"
          title="Dashboard cadru didactic"
        />
      ) : roleCockpitKind ? (
        <SchoolRoleDashboard
          kind={roleCockpitKind}
          permissions={permissions}
          client={auth.apiClient}
        />
      ) : active === "overview" ? (
        <>
          <Overview
            api={api}
            canReadGovernance={permissions.includes("education.governance.read")}
          />
          {directPermissions.includes("education.delegations.read") && (
            <EducationDelegationManager
              api={delegationApi}
              onChanged={auth.refreshEducationAuthorization}
              capabilities={{
                read: true,
                offer: directPermissions.includes("education.delegations.offer"),
                accept: directPermissions.includes("education.delegations.accept"),
                revoke: directPermissions.includes("education.delegations.revoke"),
                expire: directPermissions.includes("education.delegations.revoke"),
              }}
            />
          )}
        </>
      ) : active === "governance" ? (
        <GovernanceMeetingsPage
          api={api}
          canManage={allows("education.governance.manage")}
          canManageMeeting={(meetingID) =>
            allows("education.governance.manage", "meeting", meetingID)
          }
        />
      ) : (
        <>
          {active === "portfolios" && allows("education.portfolios.school.manage") && (
            <PortfolioProcedureManager api={portfolioProcedureAdapter(api)} />
          )}
          {active === "portfolios" && allows("education.portfolios.archive_grants.manage") && (
            <PortfolioArchiveGrantManager api={api} />
          )}
          <DomainRecordsPage
            api={api}
            area={current}
            canManage={allows(permissionForDomain(current.id as EducationRecordsDomain))}
            canManageRecord={(recordID) => allows(
              permissionForDomain(current.id as EducationRecordsDomain),
              delegationResourceTypeForDomain(current.id as EducationRecordsDomain),
              recordID,
            )}
            canManageRelation={(relation, recordID) => allows(
              relationManagePermission(
                current.id as EducationRecordsDomain,
                relation.managePermission,
              ),
              delegationResourceTypeForDomain(current.id as EducationRecordsDomain),
              recordID,
            )}
            canVerifyPortfolio={allows("education.portfolios.verify")}
            canVerifyPortfolioRecord={(recordID) => allows("education.portfolios.verify", "portfolio", recordID)}
            canManageSchoolPortfolios={allows("education.portfolios.school.manage")}
            canManageSchoolPortfolioRecord={(recordID) => allows("education.portfolios.school.manage", "portfolio", recordID)}
            portfolioTransferApi={portfolioTransferApi}
            canSendPortfolioTransfer={allows("education.portfolios.transfer")}
            canReceivePortfolioTransfer={allows("education.portfolios.transfer.receive")}
            canSendPortfolioTransferRecord={(recordID) => allows("education.portfolios.transfer", "portfolio", recordID)}
            canReceivePortfolioTransferRecord={(recordID) => allows("education.portfolios.transfer.receive", "portfolio", recordID)}
            valorificationClient={auth.apiClient}
            canReadPortfolioValorification={
              permissions.includes("education.portfolios.read") ||
              permissions.includes("education.portfolios.school.read")
            }
            canManagePortfolioValorification={
              allows("education.portfolios.manage") ||
              allows("education.portfolios.school.manage")
            }
            canManagePortfolioValorificationRecord={(recordID) =>
              allows("education.portfolios.manage", "portfolio", recordID) ||
              allows("education.portfolios.school.manage", "portfolio", recordID)
            }
          />
        </>
      )}
    </section>
  );
}
