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
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select } from "@primereact/ui/select";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import { useAuth } from "../../auth/AuthProvider";
import { createEducationApi, type AuthenticatedFetcher } from "./api";
import { visibleEducationAreas } from "./catalog";
import type {
  EducationApi,
  DirectorCockpit,
  EducationArea,
  EducationModule,
  EducationPage,
  EducationPdfRecordsDomain,
  EducationRecord,
  EducationRecordInput,
  EducationRecordsDomain,
  GovernanceMeeting,
} from "./types";

const Spinner = () => (
  <ProgressSpinner.Root>
    <ProgressSpinner.Range>
      <ProgressSpinner.Track />
      <ProgressSpinner.Value />
    </ProgressSpinner.Range>
  </ProgressSpinner.Root>
);

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
  }>;
  emptyMessage: string;
}

/** Reusable authenticated list state for all paginated Education resources. */
export function EducationListPanel<T extends { id: string }>({
  title,
  description,
  load,
  columns,
  emptyMessage,
}: EducationListPanelProps<T>) {
  const [query, setQuery] = useState("");
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

  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>{title}</Card.Title>
        <Card.Content>
          <div className="flex flex-col gap-4">
            <p>{description}</p>
            <div className="flex flex-col gap-2 sm:flex-row">
              <InputText
                aria-label={`Caută în ${title}`}
                value={query}
                placeholder="Caută"
                onChange={(event: ChangeEvent<HTMLInputElement>) =>
                  setQuery(event.target.value)
                }
              />
              <Button
                variant="outlined"
                severity="secondary"
                onClick={() => {
                  setPageNumber(1);
                  void refresh(query, 1, pageSize, sort, filters);
                }}
                disabled={loading}
              >
                Aplică filtre
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
            ) : page.items.length === 0 ? (
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
              >
                <DataTable.Table>
                  <DataTable.THead>
                    <DataTable.THeadRow>
                      {columns.map((column) => (
                        <DataTable.THeadCell key={column.header}>
                          {column.field ? (
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
                          ) : (
                            <span>{column.header}</span>
                          )}
                          {column.field && (
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
                            <DataTable.Cell key={column.header}>
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
            <div
              className="flex flex-wrap items-center justify-between gap-2"
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
}: {
  api: EducationApi;
  canManage: boolean;
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
  useEffect(() => {
    void api
      .eligibleGovernanceUsers()
      .then(setEligibleUsers)
      .catch(() =>
        setError("Utilizatorii eligibili nu au putut fi încărcați."),
      );
  }, [api]);
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
            onClick={() =>
              setEditing({
                input: inputFromRecord(undefined, meetingFields),
              })
            }
          >
            Ședință nouă
          </Button>
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
            render: (item) => (
              <div className="flex flex-wrap gap-1">
                <Button
                  size="small"
                  variant="text"
                  onClick={() =>
                    void api
                      .governanceMeetingDetail(item.id)
                      .then((value) => {
                        setDetail(value as unknown as EducationRecord);
                        setSelectedMeetingId(item.id);
                      })
                      .catch(() =>
                        setError("Detaliul ședinței nu a putut fi încărcat."),
                      )
                  }
                >
                  Detalii
                </Button>
                {canManage && (
                  <Button
                    size="small"
                    variant="text"
                    onClick={() =>
                      setEditing({
                        id: item.id,
                        input: inputFromRecord(
                          item as unknown as EducationRecord,
                          governanceFields,
                        ),
                      })
                    }
                  >
                    Editează
                  </Button>
                )}
                {canManage && (
                  <Button
                    size="small"
                    severity="danger"
                    variant="text"
                    onClick={() => setPendingDelete(item.id)}
                  >
                    Șterge
                  </Button>
                )}
              </div>
            ),
          },
        ]}
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
            await api.saveGovernanceMeeting(editing.input, editing.id);
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
        canManage={canManage}
        title="Membrii organismelor"
        relations={[
          {
            id: "memberships",
            label: "Membri",
            path: () => "/education/governance/memberships",
            fields: [
              { key: "school_year", label: "An școlar" },
              { key: "organism", label: "Organism" },
              {
                key: "app_user_id",
                label: "Utilizator",
                kind: "select",
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
        paths={["/education/governance/meetings/filters"]}
      />
      <GovernanceMeetingRelations
        api={api}
        meetingId=""
        canManage={canManage}
        title="Organisme de guvernanță"
        relations={[
          {
            id: "bodies",
            label: "Organisme",
            path: () => "/education/governance/bodies",
            summary: (_, id) =>
              `/education/governance/bodies/${encodeURIComponent(id)}/completeness-summary`,
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
          canManage={canManage}
        />
      )}
      {selectedMeetingId && (
        <EducationMetadata
          api={api}
          paths={[
            `/education/governance/meetings/${encodeURIComponent(selectedMeetingId)}/finalization-summary`,
          ]}
        />
      )}
    </div>
  );
}

type RelatedConfig = {
  id: string;
  label: string;
  path: (parentId: string) => string;
  fields: RecordField[];
  pdf?: boolean;
  advance?: (parentId: string, itemId: string) => string;
  summary?: (parentId: string, itemId: string) => string;
};
const meetingRelations: RelatedConfig[] = [
  {
    id: "participants",
    label: "Participanți",
    path: (id) =>
      `/education/governance/meetings/${encodeURIComponent(id)}/participants`,
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
    path: (id) =>
      `/education/governance/meetings/${encodeURIComponent(id)}/documents`,
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
    path: (id) =>
      `/education/governance/meetings/${encodeURIComponent(id)}/votes`,
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
    path: (id) =>
      `/education/governance/meetings/${encodeURIComponent(id)}/minutes`,
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
    path: (id) =>
      `/education/governance/meetings/${encodeURIComponent(id)}/resolutions`,
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
const domainRelations: Partial<
  Record<EducationRecordsDomain, RelatedConfig[]>
> = {
  decisions: [
    {
      id: "issuances",
      label: "Emiteri",
      path: (id) =>
        `/education/decisions/records/${encodeURIComponent(id)}/issuances`,
      fields: [
        { key: "issuance_code", label: "Cod emitere" },
        { key: "issuance_type", label: "Tip" },
        { key: "status", label: "Stare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "signed_by", label: "Semnat de" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "publication-steps",
      label: "Pași publicare",
      path: (id) =>
        `/education/decisions/records/${encodeURIComponent(id)}/publication-steps`,
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
      path: (id) =>
        `/education/regulations/records/${encodeURIComponent(id)}/versions`,
      fields: [
        { key: "version_label", label: "Versiune" },
        { key: "version_status", label: "Stare" },
        { key: "prepared_by", label: "Pregătit de" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "effective_from", label: "Aplicabil de la", kind: "date" },
        { key: "published_on", label: "Publicat la", kind: "date" },
        { key: "file_reference", label: "Referință fișier" },
      ],
    },
    {
      id: "workflow",
      label: "Pași flux",
      path: (id) =>
        `/education/regulations/records/${encodeURIComponent(id)}/workflow`,
      fields: [
        { key: "stage_order", label: "Ordine", kind: "number" },
        { key: "stage_type", label: "Etapă" },
        { key: "status", label: "Stare" },
        { key: "assigned_to", label: "Alocat" },
        { key: "due_on", label: "Termen", kind: "date" },
        { key: "completed_on", label: "Finalizat la", kind: "date" },
        { key: "outcome_note", label: "Rezultat" },
      ],
    },
  ],
  committees: [
    {
      id: "members",
      label: "Membri comisie",
      path: (id) =>
        `/education/committees/records/${encodeURIComponent(id)}/members`,
      fields: [
        { key: "full_name", label: "Nume complet" },
        { key: "role_name", label: "Rol" },
        { key: "member_type", label: "Tip" },
        { key: "mandate_from", label: "Mandat de la", kind: "date" },
        { key: "mandate_to", label: "Mandat până la", kind: "date" },
        { key: "status", label: "Stare" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  managerial: [
    {
      id: "documents",
      label: "Documente dosar",
      path: (id) =>
        `/education/managerial/records/${encodeURIComponent(id)}/documents`,
      pdf: true,
      fields: [
        { key: "document_category", label: "Categorie" },
        { key: "title", label: "Titlu" },
        { key: "document_status", label: "Stare" },
        { key: "version_label", label: "Versiune" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "publication_required", label: "Publicare", kind: "boolean" },
        { key: "registered_on", label: "Înregistrat la", kind: "date" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "owner_name", label: "Responsabil" },
        { key: "file_reference", label: "Referință fișier" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "workflow",
      label: "Pași flux",
      path: (id) =>
        `/education/managerial/records/${encodeURIComponent(id)}/workflow`,
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
      path: (id) =>
        `/education/personnel/records/${encodeURIComponent(id)}/assignments`,
      fields: [
        { key: "position_title", label: "Funcție" },
        { key: "organizational_unit", label: "Unitate" },
        { key: "assignment_type", label: "Tip" },
        { key: "status", label: "Stare" },
        { key: "start_date", label: "De la", kind: "date" },
        { key: "end_date", label: "Până la", kind: "date" },
        { key: "workload", label: "Normă", kind: "number" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "file-documents",
      label: "Documente dosar",
      path: (id) =>
        `/education/personnel/records/${encodeURIComponent(id)}/file-documents`,
      fields: [
        { key: "document_type", label: "Tip document" },
        { key: "title", label: "Titlu" },
        { key: "status", label: "Stare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "expires_on", label: "Expiră la", kind: "date" },
        { key: "file_reference", label: "Referință fișier" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "disciplinary-cases",
      label: "Cazuri disciplinare",
      path: (id) =>
        `/education/personnel/records/${encodeURIComponent(id)}/disciplinary-cases`,
      fields: [
        { key: "case_code", label: "Cod" },
        { key: "status", label: "Stare" },
        { key: "opened_on", label: "Deschis la", kind: "date" },
        { key: "closed_on", label: "Închis la", kind: "date" },
        { key: "summary", label: "Rezumat" },
        { key: "outcome", label: "Rezultat" },
      ],
    },
    {
      id: "access-events",
      label: "Evenimente acces",
      path: (id) =>
        `/education/personnel/records/${encodeURIComponent(id)}/access-events`,
      fields: [
        { key: "event_type", label: "Tip" },
        { key: "occurred_on", label: "Data", kind: "date" },
        { key: "actor_name", label: "Operator" },
        { key: "reason", label: "Motiv" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  evaluations: [
    {
      id: "self-reviews",
      label: "Autoevaluări",
      path: (id) =>
        `/education/evaluations/records/${encodeURIComponent(id)}/self-reviews`,
      fields: [
        { key: "submitted_on", label: "Depus la", kind: "date" },
        { key: "status", label: "Stare" },
        { key: "score", label: "Punctaj", kind: "number" },
        { key: "summary", label: "Rezumat" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "criteria",
      label: "Criterii",
      path: (id) =>
        `/education/evaluations/records/${encodeURIComponent(id)}/criteria`,
      fields: [
        { key: "criterion_code", label: "Cod" },
        { key: "criterion_label", label: "Criteriu" },
        { key: "max_score", label: "Maxim", kind: "number" },
        { key: "awarded_score", label: "Acordat", kind: "number" },
        { key: "status", label: "Stare" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "appeals",
      label: "Contestații",
      path: (id) =>
        `/education/evaluations/records/${encodeURIComponent(id)}/appeals`,
      pdf: true,
      fields: [
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
      id: "result-issues",
      label: "Comunicări rezultat",
      path: (id) =>
        `/education/evaluations/records/${encodeURIComponent(id)}/result-issues`,
      pdf: true,
      fields: [
        { key: "document_type", label: "Tip document" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "delivery_channel", label: "Canal" },
        { key: "delivery_status", label: "Stare livrare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "delivered_on", label: "Livrat la", kind: "date" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  mobility: [
    {
      id: "documents",
      label: "Documente",
      path: (id) =>
        `/education/mobility/records/${encodeURIComponent(id)}/documents`,
      fields: [
        { key: "document_type", label: "Tip" },
        { key: "document_title", label: "Titlu" },
        { key: "registered_on", label: "Înregistrat la", kind: "date" },
        { key: "validation_status", label: "Validare" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "scores",
      label: "Punctaje",
      path: (id) =>
        `/education/mobility/records/${encodeURIComponent(id)}/scores`,
      fields: [
        { key: "criterion_code", label: "Cod criteriu" },
        { key: "criterion_label", label: "Criteriu" },
        { key: "max_score", label: "Maxim", kind: "number" },
        { key: "awarded_score", label: "Acordat", kind: "number" },
        { key: "reviewer_name", label: "Evaluator" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "appeals",
      label: "Contestații",
      path: (id) =>
        `/education/mobility/records/${encodeURIComponent(id)}/appeals`,
      pdf: true,
      fields: [
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
      path: (id) =>
        `/education/mobility/records/${encodeURIComponent(id)}/final-decisions`,
      pdf: true,
      fields: [
        { key: "decision_stage", label: "Etapă" },
        { key: "outcome", label: "Rezultat" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "effective_from", label: "Aplicabil de la", kind: "date" },
        { key: "panel_name", label: "Comisie" },
        { key: "legal_basis", label: "Temei legal" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "result-issues",
      label: "Comunicări rezultat",
      path: (id) =>
        `/education/mobility/records/${encodeURIComponent(id)}/result-issues`,
      pdf: true,
      fields: [
        { key: "document_type", label: "Tip document" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "recipient_role", label: "Funcție" },
        { key: "delivery_channel", label: "Canal" },
        { key: "delivery_status", label: "Stare livrare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "delivered_on", label: "Livrat la", kind: "date" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  merit: [
    {
      id: "documents",
      label: "Documente",
      path: (id) =>
        `/education/gradatii/records/${encodeURIComponent(id)}/documents`,
      fields: [
        { key: "document_type", label: "Tip" },
        { key: "document_title", label: "Titlu" },
        { key: "registered_on", label: "Înregistrat la", kind: "date" },
        { key: "validation_status", label: "Validare" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "scores",
      label: "Punctaje",
      path: (id) =>
        `/education/gradatii/records/${encodeURIComponent(id)}/scores`,
      fields: [
        { key: "criterion_code", label: "Cod criteriu" },
        { key: "criterion_label", label: "Criteriu" },
        { key: "criterion_category", label: "Categorie" },
        { key: "max_score", label: "Maxim", kind: "number" },
        { key: "awarded_score", label: "Acordat", kind: "number" },
        { key: "reviewer_name", label: "Evaluator" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "appeals",
      label: "Contestații",
      path: (id) =>
        `/education/gradatii/records/${encodeURIComponent(id)}/appeals`,
      pdf: true,
      fields: [
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
      path: (id) =>
        `/education/gradatii/records/${encodeURIComponent(id)}/final-decisions`,
      pdf: true,
      fields: [
        { key: "decision_stage", label: "Etapă" },
        { key: "outcome", label: "Rezultat" },
        { key: "approved_on", label: "Aprobat la", kind: "date" },
        { key: "effective_from", label: "Aplicabil de la", kind: "date" },
        { key: "panel_name", label: "Comisie" },
        { key: "funded", label: "Finanțat", kind: "boolean" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "result-issues",
      label: "Comunicări rezultat",
      path: (id) =>
        `/education/gradatii/records/${encodeURIComponent(id)}/result-issues`,
      pdf: true,
      fields: [
        { key: "document_type", label: "Tip document" },
        { key: "recipient_name", label: "Destinatar" },
        { key: "recipient_role", label: "Funcție" },
        { key: "delivery_channel", label: "Canal" },
        { key: "delivery_status", label: "Stare livrare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "delivered_on", label: "Livrat la", kind: "date" },
        { key: "notes", label: "Note" },
      ],
    },
  ],
  portfolios: [
    {
      id: "documents",
      label: "Documente",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/documents`,
      fields: [
        { key: "document_type", label: "Tip" },
        { key: "document_title", label: "Titlu" },
        { key: "section_code", label: "Secțiune" },
        { key: "status", label: "Stare" },
        { key: "issued_on", label: "Emis la", kind: "date" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "checklist",
      label: "Checklist",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/checklist`,
      fields: [
        { key: "requirement_code", label: "Cod cerință" },
        { key: "requirement_label", label: "Cerință" },
        { key: "section_code", label: "Secțiune" },
        { key: "mandatory", label: "Obligatoriu", kind: "boolean" },
        { key: "status", label: "Stare" },
        { key: "document_count", label: "Documente", kind: "number" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "opis",
      label: "Opis",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/opis`,
      fields: [
        { key: "section_code", label: "Secțiune" },
        { key: "component_code", label: "Componentă" },
        { key: "entry_title", label: "Titlu" },
        { key: "chronological_index", label: "Ordine", kind: "number" },
        { key: "document_reference", label: "Referință" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "custody",
      label: "Custodie",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/custody`,
      fields: [
        { key: "event_type", label: "Tip eveniment" },
        { key: "custodian", label: "Custode" },
        { key: "occurred_on", label: "Data", kind: "date" },
        { key: "status", label: "Stare" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "reviews",
      label: "Revizuiri",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/reviews`,
      fields: [
        { key: "review_code", label: "Cod" },
        { key: "review_stage", label: "Etapă" },
        { key: "outcome", label: "Rezultat" },
        { key: "reviewer_name", label: "Evaluator" },
        { key: "reviewed_on", label: "Data", kind: "date" },
        { key: "compliance_score", label: "Punctaj", kind: "number" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "transfers",
      label: "Transferuri",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/transfers`,
      advance: (parentId, itemId) =>
        `/education/portfolios/records/${encodeURIComponent(parentId)}/transfers/${encodeURIComponent(itemId)}/advance`,
      fields: [
        { key: "transfer_code", label: "Cod" },
        { key: "transfer_type", label: "Tip" },
        { key: "status", label: "Stare" },
        { key: "requested_on", label: "Solicitat la", kind: "date" },
        { key: "target_institution", label: "Instituție țintă" },
        { key: "requested_by", label: "Solicitat de" },
        { key: "notes", label: "Note" },
      ],
    },
    {
      id: "valorifications",
      label: "Valorificări",
      path: (id) =>
        `/education/portfolios/records/${encodeURIComponent(id)}/valorifications`,
      fields: [
        { key: "valorification_code", label: "Cod" },
        { key: "scope", label: "Domeniu" },
        { key: "status", label: "Stare" },
        { key: "requested_by", label: "Solicitat de" },
        { key: "target_institution", label: "Instituție țintă" },
        { key: "started_on", label: "Început la", kind: "date" },
        { key: "completed_on", label: "Finalizat la", kind: "date" },
      ],
    },
  ],
};

function GovernanceMeetingRelations({
  api,
  meetingId,
  canManage,
  relations = meetingRelations,
  title = "Ședință selectată — operațiuni",
}: {
  api: EducationApi;
  meetingId: string;
  canManage: boolean;
  relations?: RelatedConfig[];
  title?: string;
}) {
  const [relation, setRelation] = useState(relations[0]);
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
  const path = relation.path(meetingId);
  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      setPage(await api.relatedRecords(path));
    } catch {
      setError("Subresursa nu a putut fi încărcată.");
    } finally {
      setLoading(false);
    }
  }, [api, path]);
  useEffect(() => {
    void load();
  }, [load]);
  const action = async (fn: () => Promise<void>) => {
    try {
      await fn();
      setEditing(undefined);
      setDetail(undefined);
      await load();
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
                  onClick={() => setRelation(item)}
                >
                  {item.label}
                </Button>
              ))}
            </div>
            {canManage && (
              <div>
                <Button
                  size="small"
                  onClick={() =>
                    setEditing({
                      input: inputFromRecord(undefined, relation.fields),
                    })
                  }
                >
                  Adaugă {relation.label.toLowerCase()}
                </Button>
              </div>
            )}
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
            ) : page.items.length === 0 ? (
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
              >
                <DataTable.Table>
                  <DataTable.THead>
                    <DataTable.THeadRow>
                      <DataTable.THeadCell>Înregistrare</DataTable.THeadCell>
                      <DataTable.THeadCell>Stare</DataTable.THeadCell>
                      <DataTable.THeadCell>Acțiuni</DataTable.THeadCell>
                    </DataTable.THeadRow>
                  </DataTable.THead>
                  <DataTable.TBody>
                    {({ item, index }) => {
                      const record = item as EducationRecord;
                      return (
                        <DataTable.Row key={record.id} index={index}>
                          <DataTable.Cell>
                            {displayRecord(record, recordPrimaryKeys)}
                          </DataTable.Cell>
                          <DataTable.Cell>
                            {displayRecord(record, recordStatusKeys)}
                          </DataTable.Cell>
                          <DataTable.Cell>
                            <div className="flex flex-wrap gap-1">
                              <Button
                                size="small"
                                variant="text"
                                onClick={() =>
                                  void api
                                    .relatedDetail(path, record.id)
                                    .then((value) => {
                                      setDetail(value);
                                      setSelectedRelatedId(record.id);
                                    })
                                    .catch(() =>
                                      setError(
                                        "Detaliul nu a putut fi încărcat.",
                                      ),
                                    )
                                }
                              >
                                Detalii
                              </Button>
                              {canManage && relation.advance && (
                                <Button
                                  size="small"
                                  variant="text"
                                  onClick={() =>
                                    void api
                                      .command(
                                        relation.advance?.(
                                          meetingId,
                                          record.id,
                                        ) ?? "",
                                      )
                                      .then(load)
                                      .catch(() =>
                                        setError(
                                          "Transferul nu a putut fi avansat.",
                                        ),
                                      )
                                  }
                                >
                                  Avansează
                                </Button>
                              )}
                              {relation.pdf && (
                                <Button
                                  size="small"
                                  variant="text"
                                  onClick={() =>
                                    viewPdf(
                                      api.relatedPdf(path, record.id),
                                      () =>
                                        setError(
                                          "PDF-ul nu a putut fi încărcat.",
                                        ),
                                    )
                                  }
                                >
                                  PDF
                                </Button>
                              )}
                              {canManage && (
                                <Button
                                  size="small"
                                  variant="text"
                                  onClick={() =>
                                    setEditing({
                                      id: record.id,
                                      input: inputFromRecord(
                                        record,
                                        relation.fields,
                                      ),
                                    })
                                  }
                                >
                                  Editează
                                </Button>
                              )}
                              {canManage && (
                                <Button
                                  size="small"
                                  severity="danger"
                                  variant="text"
                                  onClick={() => setPendingDelete(record.id)}
                                >
                                  Șterge
                                </Button>
                              )}
                            </div>
                          </DataTable.Cell>
                        </DataTable.Row>
                      );
                    }}
                  </DataTable.TBody>
                </DataTable.Table>
              </DataTable.Root>
            )}
            <RecordFormDialog
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
                  await api.saveRelated(path, editing.input, editing.id);
                })
              }
            />
            <RecordDetailDialog
              record={detail}
              title={relation.label}
              fields={relation.fields}
              onClose={() => setDetail(undefined)}
            />
            {selectedRelatedId && relation.summary && (
              <EducationMetadata
                api={api}
                paths={[relation.summary(meetingId, selectedRelatedId)]}
              />
            )}
            <DeleteDialog
              open={pendingDelete}
              onClose={() => setPendingDelete(undefined)}
              onConfirm={() =>
                pendingDelete &&
                void action(async () => {
                  await api.deleteRelated(path, pendingDelete);
                  setPendingDelete(undefined);
                })
              }
            />
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
  kind?: "date" | "number" | "boolean" | "select";
  options?: Array<{ label: string; value: string }>;
};
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
    { key: "retention_until", label: "Retenție până la", kind: "date" },
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
    { key: "publication_code", label: "Cod publicare" },
    { key: "domain", label: "Domeniu" },
    { key: "entity_type", label: "Tip entitate" },
    { key: "entity_label", label: "Entitate" },
    { key: "publication_channel", label: "Canal" },
    { key: "publication_status", label: "Stare" },
    { key: "anonymization_status", label: "Anonimizare" },
    { key: "mandatory", label: "Obligatorie", kind: "boolean" },
    { key: "published_on", label: "Publicat la", kind: "date" },
    { key: "reviewed_by", label: "Revizuit de" },
    { key: "notes", label: "Note" },
  ],
};
function permissionForDomain(domain: EducationRecordsDomain) {
  return `education.${domain === "merit" ? "gradatii" : domain}.manage`;
}
const domainWizardRoutes: Partial<Record<EducationRecordsDomain, string>> = {
  managerial: "/scoala/governance/managerial-wizard",
  personnel: "/scoala/personnel/wizard",
  evaluations: "/scoala/personnel/evaluations-wizard",
  declarations: "/scoala/personnel/declarations-wizard",
  mobility: "/scoala/personnel/mobility-wizard",
  merit: "/scoala/personnel/merit-wizard",
  portfolios: "/scoala/portfolio/wizard",
};
const domainTableFields: Record<
  EducationRecordsDomain,
  { primary: string; status: string }
> = {
  decisions: { primary: "title", status: "status" },
  managerial: { primary: "title", status: "status" },
  regulations: { primary: "title", status: "status" },
  committees: { primary: "title", status: "status" },
  personnel: { primary: "full_name", status: "status" },
  evaluations: { primary: "full_name", status: "status" },
  declarations: { primary: "full_name", status: "status" },
  mobility: { primary: "full_name", status: "status" },
  merit: { primary: "full_name", status: "status" },
  portfolios: { primary: "owner_name", status: "status" },
  compliance: { primary: "entity_label", status: "publication_status" },
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
function downloadBlob(load: Promise<Blob>, name: string, onError: () => void) {
  void load
    .then((blob) => {
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = name;
      link.click();
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
function DomainRecordsPage({
  api,
  area,
  canManage,
}: {
  api: EducationApi;
  area: EducationArea;
  canManage: boolean;
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
  const fields = domainFields[domain];
  const tableFields = domainTableFields[domain];
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
      {canManage && (
        <div className="flex flex-wrap gap-2">
          <Button
            onClick={() =>
              setEditing({ input: inputFromRecord(undefined, fields) })
            }
          >
            Înregistrare nouă
          </Button>
          {domainWizardRoutes[domain] && (
            <Button
              variant="outlined"
              onClick={() => navigate(domainWizardRoutes[domain] as string)}
            >
              Creează prin ghid
            </Button>
          )}
        </div>
      )}
      {metadata && <EducationMetadata api={api} paths={metadata} />}
      <EducationListPanel<EducationRecord>
        title={area.label}
        description={area.description}
        load={wrappedLoad}
        emptyMessage="Nu există înregistrări care corespund filtrului ales."
        columns={[
          {
            field: tableFields.primary,
            header: "Înregistrare",
            render: (item) => displayRecord(item, recordPrimaryKeys),
          },
          {
            field: tableFields.status,
            header: "Stare",
            render: (item) => (
              <Tag
                value={displayRecord(item, recordStatusKeys)}
                severity="secondary"
              />
            ),
          },
          {
            header: "Acțiuni",
            render: (item) => (
              <div className="flex flex-wrap gap-1">
                <Button
                  size="small"
                  variant="text"
                  onClick={() =>
                    void api
                      .recordDetail(domain, item.id)
                      .then((value) => {
                        setDetail(value);
                        setSelectedRecordId(item.id);
                      })
                      .catch(() => setError("Detaliul nu a putut fi încărcat."))
                  }
                >
                  Detalii
                </Button>
                {supportsPdf(domain) && (
                  <Button
                    size="small"
                    variant="text"
                    onClick={() =>
                      viewPdf(api.recordPdf(domain, item.id), () =>
                        setError("PDF-ul nu a putut fi generat."),
                      )
                    }
                  >
                    PDF
                  </Button>
                )}
                {canManage && (
                  <Button
                    size="small"
                    variant="text"
                    onClick={() =>
                      setEditing({
                        id: item.id,
                        input: inputFromRecord(item, fields),
                      })
                    }
                  >
                    Editează
                  </Button>
                )}
                {canManage && (
                  <Button
                    size="small"
                    severity="danger"
                    variant="text"
                    onClick={() => setPendingDelete(item.id)}
                  >
                    Șterge
                  </Button>
                )}
              </div>
            ),
          },
        ]}
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
            await api.saveRecord(domain, editing.input, editing.id);
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
      {selectedRecordId && domain === "portfolios" && canManage && (
        <div>
          <Button
            variant="outlined"
            severity="secondary"
            onClick={() =>
              void api
                .command(
                  `/education/portfolios/records/${encodeURIComponent(selectedRecordId)}/opis/regenerate`,
                )
                .then(() => setRefresh((value) => value + 1))
                .catch(() => setError("Opisul nu a putut fi regenerat."))
            }
          >
            Regenerare opis
          </Button>
        </div>
      )}
      {selectedRecordId && domainRelations[domain] && (
        <GovernanceMeetingRelations
          api={api}
          meetingId={selectedRecordId}
          canManage={canManage}
          title={`${area.label} — operațiuni dosar`}
          relations={domainRelations[domain]}
        />
      )}
      {selectedRecordId && domainDetailMetadata[domain] && (
        <EducationMetadata
          api={api}
          paths={domainDetailMetadata[domain].map(
            (suffix) =>
              `${recordsBasePath(domain)}/${encodeURIComponent(selectedRecordId)}${suffix}`,
          )}
        />
      )}
    </div>
  );
}

const domainMetadata: Partial<Record<EducationRecordsDomain, string[]>> = {
  decisions: [
    "/education/decisions/dashboard",
    "/education/decisions/records/filters",
  ],
  managerial: [
    "/education/managerial/dashboard",
    "/education/managerial/records/filters",
  ],
  regulations: [
    "/education/regulations/dashboard",
    "/education/regulations/records/filters",
  ],
  personnel: [
    "/education/personnel/dashboard",
    "/education/personnel/records/filters",
  ],
  evaluations: [
    "/education/evaluations/dashboard",
    "/education/evaluations/records/filters",
  ],
  declarations: [
    "/education/declarations/dashboard",
    "/education/declarations/records/filters",
  ],
  mobility: [
    "/education/mobility/dashboard",
    "/education/mobility/records/filters",
  ],
  merit: [
    "/education/gradatii/dashboard",
    "/education/gradatii/records/filters",
  ],
  portfolios: [
    "/education/portfolios/dashboard",
    "/education/portfolios/records/filters",
  ],
};
const domainDetailMetadata: Partial<Record<EducationRecordsDomain, string[]>> =
  {
    committees: ["/completeness-summary"],
    managerial: ["/portfolio-summary"],
    personnel: ["/portfolio-dossier-summary"],
    portfolios: ["/transfer-summary"],
    regulations: ["/procedural-summary"],
  };
function recordsBasePath(domain: EducationRecordsDomain) {
  return (
    {
      decisions: "/education/decisions/records",
      managerial: "/education/managerial/records",
      regulations: "/education/regulations/records",
      committees: "/education/committees/records",
      personnel: "/education/personnel/records",
      evaluations: "/education/evaluations/records",
      declarations: "/education/declarations/records",
      mobility: "/education/mobility/records",
      merit: "/education/gradatii/records",
      portfolios: "/education/portfolios/records",
      compliance: "/education/compliance/publications",
    } as Record<EducationRecordsDomain, string>
  )[domain];
}
function readable(value: string) {
  return value
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}
function EducationMetadata({
  api,
  paths,
}: {
  api: EducationApi;
  paths: string[];
}) {
  const [items, setItems] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    void Promise.all(paths.map((path) => api.metadata(path)))
      .then(setItems)
      .catch(() =>
        setError("Indicatorii sau filtrele nu au putut fi încărcate."),
      );
  }, [api, paths]);
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
  const values = items.flatMap((item) =>
    Object.entries(item).filter(
      ([, value]) =>
        typeof value === "string" ||
        typeof value === "number" ||
        Array.isArray(value),
    ),
  );
  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>Indicatori și filtre disponibile</Card.Title>
        <Card.Content>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {values.map(([key, value]) => (
              <div key={key}>
                <strong>{readable(key)}</strong>
                <p>
                  {Array.isArray(value)
                    ? value.join(", ") || "—"
                    : String(value)}
                </p>
              </div>
            ))}
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}

function RecordFormDialog({
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
                {fields.map((field) => (
                  <label className="flex flex-col gap-1" key={field.key}>
                    <span>{field.label}</span>
                    {field.kind === "boolean" || field.kind === "select" ? (
                      <Select.Root
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
                        <Select.Trigger aria-label={field.label}>
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
                        type={
                          field.kind === "date"
                            ? "date"
                            : field.kind === "number"
                              ? "number"
                              : "text"
                        }
                        value={String(open?.input[field.key] ?? "")}
                        onChange={(event: ChangeEvent<HTMLInputElement>) =>
                          set(field, event.target.value)
                        }
                      />
                    )}
                  </label>
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
                <Button onClick={onSave}>Salvează</Button>
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
  endpoint?: string;
  title?: string;
}) {
  const [data, setData] = useState<Record<string, unknown>>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    void (endpoint ? api.dashboardAt(endpoint) : api.directorCockpit())
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
  areas,
  permissions,
}: {
  kind: "secretariat" | "compliance";
  areas: EducationArea[];
  permissions: readonly string[];
}) {
  const title =
    kind === "secretariat" ? "Cockpit secretariat" : "Cockpit conformitate";
  const description =
    kind === "secretariat"
      ? "Acces rapid la registrele școlare și operațiunile administrative permise."
      : "Monitorizare și acces la registrele de conformitate permise pentru instituția curentă.";
  const cards = [
    ["Domenii vizibile", areas.length],
    [
      "Registre disponibile",
      areas.filter((area) => area.id !== "overview").length,
    ],
    [
      "Operațiuni de administrare",
      permissions.filter(
        (permission) =>
          permission.startsWith("education.") && permission.endsWith(".manage"),
      ).length,
    ],
  ];
  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>{title}</Card.Title>
        <Card.Content>
          <div className="flex flex-col gap-4">
            <p>{description}</p>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              {cards.map(([label, value]) => (
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
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
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
      <EducationMetadata api={api} paths={["/education/director/cockpit"]} />
      <EducationCatalogs api={api} />
    </div>
  );
}

function EducationCatalogs({ api }: { api: EducationApi }) {
  const [selected, setSelected] = useState("Taxonomii");
  const catalogs: Record<string, string> = {
    Taxonomii: "/education/taxonomies",
    Cerințe: "/education/requirements",
    "Secțiuni portofoliu": "/education/portfolios/sections",
  };
  const path = catalogs[selected] as string;
  const [items, setItems] = useState<EducationRecord[]>([]);
  const [error, setError] = useState<string>();
  useEffect(() => {
    void api
      .relatedRecords(path)
      .then((page) => {
        setItems(page.items);
        setError(undefined);
      })
      .catch(() => setError("Catalogul nu a putut fi încărcat."));
  }, [api, path]);
  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>Cataloge educaționale</Card.Title>
        <Card.Content>
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap gap-2">
              {Object.keys(catalogs).map((label) => (
                <Button
                  key={label}
                  size="small"
                  variant={label === selected ? undefined : "outlined"}
                  severity={label === selected ? undefined : "secondary"}
                  onClick={() => setSelected(label)}
                >
                  {label}
                </Button>
              ))}
            </div>
            {error ? (
              <Message.Root severity="error">
                <Message.Content>
                  <Message.Text>{error}</Message.Text>
                </Message.Content>
              </Message.Root>
            ) : items.length === 0 ? (
              <Message.Root severity="info">
                <Message.Content>
                  <Message.Text>Nu există elemente în catalog.</Message.Text>
                </Message.Content>
              </Message.Root>
            ) : (
              <DataTable.Root data={items as Record<string, unknown>[]}>
                <DataTable.Table>
                  <DataTable.THead>
                    <DataTable.THeadRow>
                      <DataTable.THeadCell>Element</DataTable.THeadCell>
                      <DataTable.THeadCell>Cod</DataTable.THeadCell>
                    </DataTable.THeadRow>
                  </DataTable.THead>
                  <DataTable.TBody>
                    {({ item, index }) => {
                      const record = item as EducationRecord;
                      return (
                        <DataTable.Row
                          key={record.id ?? String(index)}
                          index={index}
                        >
                          <DataTable.Cell>
                            {displayRecord(record, [
                              "label_ro",
                              "label",
                              "name",
                              "title",
                              "code",
                            ])}
                          </DataTable.Cell>
                          <DataTable.Cell>
                            {String(record.code ?? record.id ?? "—")}
                          </DataTable.Cell>
                        </DataTable.Row>
                      );
                    }}
                  </DataTable.TBody>
                </DataTable.Table>
              </DataTable.Root>
            )}
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}

export interface EducationWorkspaceProps {
  api?: EducationApi;
  apiFetch?: AuthenticatedFetcher;
  institutionId?: string;
  permissions?: readonly string[];
  modules?: readonly EducationModule[];
}

/**
 * Responsive Școală foundation. Routes are backend-authoritative; this module
 * intentionally renders only areas granted by both session context and module state.
 */
export function EducationWorkspace(props: EducationWorkspaceProps) {
  const auth = useAuth();
  const institutionId =
    props.institutionId ?? auth.session?.institution_id ?? "";
  const permissions = props.permissions ?? auth.session?.permissions ?? [];
  const modules = props.modules ?? auth.session?.modules ?? [];
  const fallbackApi = useMemo(
    () =>
      institutionId
        ? createEducationApi(props.apiFetch ?? auth.apiFetch)
        : undefined,
    [auth.apiFetch, institutionId, props.apiFetch],
  );
  const api = props.api ?? fallbackApi;
  const location = useLocation();
  const areas = useMemo(
    () => visibleEducationAreas(permissions, modules),
    [modules, permissions],
  );
  const routeActive = location.pathname.includes("/governance")
    ? "governance"
    : location.pathname.includes("/personnel")
      ? "personnel"
      : location.pathname.includes("/portfolio")
        ? "portfolios"
        : location.pathname.includes("/compliance")
          ? "compliance"
          : "overview";
  const [active, setActive] = useState(routeActive);
  const [exportError, setExportError] = useState<string>();
  useEffect(() => {
    setActive(routeActive);
    if (!areas.some((area) => area.id === active))
      setActive(areas[0]?.id ?? "overview");
  }, [active, areas, routeActive]);

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
  if (areas.length === 0)
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
              <div className="flex flex-wrap gap-2">
                <Button
                  size="small"
                  variant="outlined"
                  severity="secondary"
                  onClick={() =>
                    viewPdf(api.exportFile("pdf"), () =>
                      setExportError("Exportul PDF nu a putut fi generat."),
                    )
                  }
                >
                  Export PDF
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  severity="secondary"
                  onClick={() =>
                    downloadBlob(api.exportFile("csv"), "educatie.csv", () =>
                      setExportError("Exportul CSV nu a putut fi generat."),
                    )
                  }
                >
                  Export CSV
                </Button>
              </div>
              {exportError && (
                <Message.Root severity="error">
                  <Message.Content>
                    <Message.Text>{exportError}</Message.Text>
                  </Message.Content>
                </Message.Root>
              )}
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
          endpoint="/education/portfolios/dashboard"
          title="Dashboard cadru didactic"
        />
      ) : location.pathname.includes("/secretariat") ? (
        <SchoolRoleDashboard
          kind="secretariat"
          areas={areas}
          permissions={permissions}
        />
      ) : location.pathname.includes("/compliance") ? (
        <SchoolRoleDashboard
          kind="compliance"
          areas={areas}
          permissions={permissions}
        />
      ) : active === "overview" ? (
        <Overview
          api={api}
          canReadGovernance={permissions.includes("education.governance.read")}
        />
      ) : active === "governance" ? (
        <GovernanceMeetingsPage
          api={api}
          canManage={permissions.includes("education.governance.manage")}
        />
      ) : (
        <DomainRecordsPage
          api={api}
          area={current}
          canManage={permissions.includes(
            permissionForDomain(current.id as EducationRecordsDomain),
          )}
        />
      )}
    </section>
  );
}
