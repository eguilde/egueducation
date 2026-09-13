import { useEffect, useId, useRef, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Refresh } from "@primeicons/react";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { Select } from "@primereact/ui/select";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import { PortfolioLifecycleStatus } from "./PortfolioLifecycleStatus";
import type { EducationApi, PortfolioLifecycleListQuery, PortfolioLifecyclePage, PortfolioLifecycleResult } from "./types";

type Props = { api: EducationApi; portfolioID: string; canRetry: boolean; canSubmitDisposition?: boolean; canDecideDisposition?: boolean; onClose: () => void };
const states = { pending: "În așteptare", processing: "În curs", completed: "Finalizată", blocked: "Blocată", dead_letter: "Oprită după erori" };
const kinds = { cessation_retention: "Retenție la încetare", legal_hold_reconcile: "Blocaj juridic" };
const empty = (): PortfolioLifecyclePage => ({ items: [], total: 0, page: 1, pageSize: 20 });
const dateLabel = (value: string) => { const date = new Date(value); return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString("ro-RO"); };

export function PortfolioLifecycleHistory(props: Props) {
  return <History key={props.portfolioID} {...props} />;
}

function History({ api, portfolioID, canRetry, canSubmitDisposition = false, canDecideDisposition = false, onClose }: Props) {
  const titleID = useId();
  const [query, setQuery] = useState<PortfolioLifecycleListQuery>({ page: 1, pageSize: 20, sort: "requested_at", direction: "desc" });
  const [data, setData] = useState(empty);
  const [revision, setRevision] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const [selected, setSelected] = useState<PortfolioLifecycleResult>();
  const selection = useRef(0);
  useEffect(() => () => { selection.current++; }, []);
  useEffect(() => {
    let active = true;
    setBusy(true); setError(undefined);
    void api.portfolioLifecycleOperations(portfolioID, query).then(value => { if (active) setData(value); }).catch(() => {
      if (active) { selection.current++; setSelected(undefined); setData(empty()); setError("Istoricul operațiilor nu a putut fi încărcat."); }
    }).finally(() => { if (active) setBusy(false); });
    return () => { active = false; };
  }, [api, portfolioID, query, revision]);
  const choose = async (operationID: string) => {
    const request = ++selection.current; setSelected(undefined); setError(undefined);
    try { const value = await api.portfolioLifecycleOperation(portfolioID, operationID); if (selection.current === request) setSelected(value); }
    catch { if (selection.current === request) setError("Starea operației nu a putut fi încărcată."); }
  };
  const sort = (field: PortfolioLifecycleListQuery["sort"]) => setQuery(value => ({ ...value, page: 1, sort: field, direction: value.sort === field && value.direction === "asc" ? "desc" : "asc" }));
  const statusOptions = [{ label: "Toate stările", value: "" }, ...Object.entries(states).map(([value, label]) => ({ value, label }))];
  const typeOptions = [{ label: "Toate tipurile", value: "" }, ...Object.entries(kinds).map(([value, label]) => ({ value, label }))];
  const selectionControl = (label: string, value: string, options: Array<{label: string;value: string}>, onChange: (value: string) => void) => <Select.Root value={value} options={options} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => onChange(String(event.value ?? ""))}><Select.Trigger aria-label={label}><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root>;
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && onClose()}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup aria-labelledby={titleID} className="w-[min(96vw,64rem)]">
    <Dialog.Header><Dialog.Title id={titleID}>Istoric operații de protecție</Dialog.Title><Dialog.Close aria-label="Închide istoricul operațiilor" /></Dialog.Header>
    <Dialog.Content><div className="flex flex-col gap-3">
      {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
      <div className="flex max-h-[55dvh] min-h-48 flex-col overflow-hidden border border-surface rounded-border">
        <DataTable.Root data={data.items} dataKey="id" scrollable className="min-h-0 overflow-auto"><DataTable.Table>
          <DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>
            <DataTable.THeadCell><Button variant="text" size="small" aria-label="Sortează data solicitării" onClick={() => sort("requested_at")}>Solicitată la</Button></DataTable.THeadCell>
            <DataTable.THeadCell><Button variant="text" size="small" aria-label="Sortează tipul operației" onClick={() => sort("type")}>Tip</Button></DataTable.THeadCell>
            <DataTable.THeadCell><Button variant="text" size="small" aria-label="Sortează starea operației" onClick={() => sort("status")}>Stare</Button></DataTable.THeadCell>
            <DataTable.THeadCell>Versiuni verificate</DataTable.THeadCell><DataTable.THeadCell>Acțiuni <Button variant="text" size="small" aria-label="Reîncarcă istoricul" disabled={busy} onClick={() => setRevision(value => value + 1)}><Refresh aria-hidden="true" /></Button></DataTable.THeadCell>
          </DataTable.THeadRow><DataTable.THeadRow>
            <DataTable.THeadCell><InputText type="date" aria-label="Filtru data solicitării UTC" value={query["filter.requested_at"] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setQuery(value => ({ ...value, page: 1, "filter.requested_at": event.target.value || undefined }))} /></DataTable.THeadCell>
            <DataTable.THeadCell>{selectionControl("Filtru tip operație", query["filter.type"] ?? "", typeOptions, filter => setQuery(value => ({ ...value, page: 1, "filter.type": filter as PortfolioLifecycleListQuery["filter.type"] || undefined })))}</DataTable.THeadCell>
            <DataTable.THeadCell>{selectionControl("Filtru stare operație", query["filter.status"] ?? "", statusOptions, filter => setQuery(value => ({ ...value, page: 1, "filter.status": filter as PortfolioLifecycleListQuery["filter.status"] || undefined })))}</DataTable.THeadCell>
            <DataTable.THeadCell /><DataTable.THeadCell />
          </DataTable.THeadRow></DataTable.THead>
          <DataTable.TBody>{({ item, index }) => { const operation = item as PortfolioLifecyclePage["items"][number]; return <DataTable.Row key={operation.id} index={index}><DataTable.Cell>{dateLabel(operation.requested_at)}</DataTable.Cell><DataTable.Cell>{kinds[operation.type]}</DataTable.Cell><DataTable.Cell>{states[operation.status]}</DataTable.Cell><DataTable.Cell>{operation.completed_versions} / {operation.total_versions}</DataTable.Cell><DataTable.Cell><Button variant="text" size="small" aria-label={`Stare operație ${operation.id}`} onClick={() => void choose(operation.id)}>Stare și acțiuni</Button></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody>
        </DataTable.Table></DataTable.Root>
        <div className="sticky bottom-0 flex shrink-0 flex-wrap items-center justify-between gap-2 border-t border-surface p-2" aria-label="Paginare operații">
          <span>{busy ? "Se încarcă…" : `${data.total} operații`}</span><div className="flex flex-wrap items-center gap-2">
            {selectionControl("Operații pe pagină", String(query.pageSize ?? 20), [10,20,50].map(value => ({ label: String(value), value: String(value) })), size => setQuery(value => ({ ...value, page: 1, pageSize: Number(size) })))}
            <Button variant="outlined" size="small" disabled={busy || (query.page ?? 1) <= 1} onClick={() => setQuery(value => ({ ...value, page: (value.page ?? 1) - 1 }))}>Anterior</Button>
            <Button variant="outlined" size="small" disabled={busy || (query.page ?? 1) * (query.pageSize ?? 20) >= data.total} onClick={() => setQuery(value => ({ ...value, page: (value.page ?? 1) + 1 }))}>Următor</Button>
          </div>
        </div>
      </div>
      {selected && <PortfolioLifecycleStatus api={api} initial={selected} canRetry={canRetry} canSubmitDisposition={canSubmitDisposition} canDecideDisposition={canDecideDisposition} onUpdate={() => setRevision(value => value + 1)} />}
    </div></Dialog.Content>
  </Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>;
}
