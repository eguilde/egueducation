import { useCallback, useEffect, useRef, useState, type ChangeEvent, type Dispatch, type SetStateAction } from "react";
import { Button } from "@primereact/ui/button";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { Select } from "@primereact/ui/select";
import { Tag } from "@primereact/ui/tag";
import type { ArchiveRetentionApi, ArchiveSeriesRetentionRule, RetentionQuery } from "./archive-retention-api";

export function ArchiveRetentionWorkspace({ api, canManage, canApprove, actorSubject }: { api: ArchiveRetentionApi; canManage: boolean; canApprove: boolean; actorSubject?: string }) {
  const [query, setQuery] = useState<RetentionQuery>({ page: 1, pageSize: 20, sort: "effective_from", direction: "desc" });
  const [page, setPage] = useState<{ items: ArchiveSeriesRetentionRule[]; total: number; page: number; pageSize: number }>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [error, setError] = useState<string>(); const [open, setOpen] = useState(false); const [saving, setSaving] = useState(false); const [loading, setLoading] = useState(true);
  const [retiring, setRetiring] = useState<ArchiveSeriesRetentionRule>(); const [retirementReason, setRetirementReason] = useState("");
  const [taxonomy, setTaxonomy] = useState<Array<{ id: string; label: string }>>([]); const [sources, setSources] = useState<Array<{ id: string; label: string }>>([]);
  const [form, setForm] = useState({ taxonomy_node_id: "", source_id: "", effective_from: "", effective_to: "", minimum_retention_days: "" });
  const requestSequence = useRef(0);
  const load = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true); setError(undefined);
    try {
      const next = await api.list(query);
      if (sequence === requestSequence.current) setPage(next);
    } catch {
      if (sequence === requestSequence.current) setError("Regulile de retenție nu au putut fi încărcate.");
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [api, query]);
  useEffect(() => { void load(); return () => { requestSequence.current += 1; }; }, [load]);
  useEffect(() => {
    let active = true;
    void Promise.all([api.taxonomy(), api.sources("")]).then(([nodes, legalSources]) => {
      if (active) { setTaxonomy(nodes); setSources(legalSources); }
    }).catch(() => { if (active) setError("Opțiunile pentru regulă nu au putut fi încărcate." ); });
    return () => { active = false; };
  }, [api]);
  const mutate = async (operation: () => Promise<unknown>) => { setSaving(true); try { await operation(); await load(); } catch { setError("Comanda a fost refuzată; reîncărcați și încercați din nou."); } finally { setSaving(false); } };
  const save = () => void mutate(async () => { await api.propose({ taxonomy_node_id: form.taxonomy_node_id, source_id: form.source_id, effective_from: form.effective_from, effective_to: form.effective_to || undefined, anchor_kind: "intake_received_at", duration_model: "minimum_days", minimum_retention_days: Number(form.minimum_retention_days) }); setOpen(false); });
  const retire = () => void mutate(async () => { if (!retiring) return; await api.retire(retiring.id, retiring.expected_version, retirementReason.trim()); setRetiring(undefined); setRetirementReason(""); });
  const last = Math.max(1, Math.ceil(page.total / page.pageSize));
  return <RetentionContents api={api} canManage={canManage} canApprove={canApprove} actorSubject={actorSubject} query={query} setQuery={setQuery} page={page} loading={loading} error={error} taxonomy={taxonomy} sources={sources} form={form} setForm={setForm} open={open} setOpen={setOpen} saving={saving} save={save} mutate={mutate} retiring={retiring} setRetiring={setRetiring} retirementReason={retirementReason} setRetirementReason={setRetirementReason} retire={retire} last={last} />;
}

function retentionStatus(value: unknown): RetentionQuery["status"] {
  return value === "active" || value === "proposed" || value === "retired" || value === "revoked" ? value : undefined;
}

function RetentionContents(props: any) {
  const { api, canManage, canApprove, actorSubject, query, setQuery, page, loading, error, taxonomy, sources, form, setForm, open, setOpen, saving, save, mutate, retiring, setRetiring, retirementReason, setRetirementReason, retire, last } = props;
  const sort = (field: RetentionQuery["sort"]) => setQuery((value: RetentionQuery) => ({ ...value, page: 1, sort: field, direction: value.sort === field && value.direction === "asc" ? "desc" : "asc" }));
  return <RetentionTable {...props} sort={sort} />;
}

function RetentionTable({ api, canManage, canApprove, actorSubject, query, setQuery, page, loading, error, taxonomy, sources, open, setOpen, saving, form, setForm, save, mutate, retiring, setRetiring, retirementReason, setRetirementReason, retire, last, sort }: any) {
  const set = (patch: Partial<RetentionQuery>) => setQuery((value: RetentionQuery) => ({ ...value, ...patch, page: 1 }));
  if (open) return <ProposalDialog taxonomy={taxonomy} sources={sources} form={form} setForm={setForm} saving={saving} error={error} onClose={() => setOpen(false)} onSave={save} />;
  return <section className="mt-4 flex flex-col gap-3" aria-label="Reguli de retenție arhivă">{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}<div className="max-h-[min(58dvh,34rem)] overflow-auto rounded-border border border-surface">{loading ? <div className="p-8" aria-label="Se încarcă regulile"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track/><ProgressSpinner.Value/></ProgressSpinner.Range></ProgressSpinner.Root></div> : <DataTable.Root data={page.items as Record<string, unknown>[]} dataKey="id"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell>Serie arhivistică</DataTable.THeadCell><DataTable.THeadCell><Button size="small" variant="text" onClick={() => sort("effective_from")}>În vigoare de la</Button></DataTable.THeadCell><DataTable.THeadCell><Button size="small" variant="text" onClick={() => sort("status")}>Stare</Button></DataTable.THeadCell><DataTable.THeadCell><Button size="small" variant="text" onClick={() => sort("minimum_retention_days")}>Zile minime</Button></DataTable.THeadCell><DataTable.THeadCell><span>Acțiuni </span>{canManage && <Button size="small" onClick={() => setOpen(true)}>Adaugă regulă</Button>}</DataTable.THeadCell></DataTable.THeadRow><DataTable.THeadRow><DataTable.THeadCell><Select.Root value={query.taxonomyNodeID ?? ""} options={[{ label: "Toate seriile", value: "" }, ...taxonomy.map((node: any) => ({ label: node.label, value: node.id }))]} optionLabel="label" optionValue="value" onValueChange={(event: any) => set({ taxonomyNodeID: String(event.value || "") || undefined })}><Select.Trigger aria-label="Filtru serie arhivistică"><Select.Value/><Select.Indicator/></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List/></Select.Popup></Select.Positioner></Select.Portal></Select.Root></DataTable.THeadCell><DataTable.THeadCell/><DataTable.THeadCell><Select.Root value={query.status ?? ""} options={[{ label: "Toate stările", value: "" }, { label: "Propusă", value: "proposed" }, { label: "Activă", value: "active" }, { label: "Retrasă", value: "retired" }]} optionLabel="label" optionValue="value" onValueChange={(event: any) => set({ status: retentionStatus(event.value) })}><Select.Trigger aria-label="Filtru stare"><Select.Value/><Select.Indicator/></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List/></Select.Popup></Select.Positioner></Select.Portal></Select.Root></DataTable.THeadCell><DataTable.THeadCell/><DataTable.THeadCell/></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }: any) => { const rule = item as ArchiveSeriesRetentionRule; return <DataTable.Row key={rule.id} index={index}><DataTable.Cell>{taxonomy.find((node: any) => node.id === rule.taxonomy_node_id)?.label ?? "Serie arhivistică"}</DataTable.Cell><DataTable.Cell>{rule.effective_from}</DataTable.Cell><DataTable.Cell><Tag value={rule.status}/></DataTable.Cell><DataTable.Cell>{rule.minimum_retention_days}</DataTable.Cell><DataTable.Cell>{canApprove && rule.status === "proposed" && (!actorSubject || rule.proposed_by_subject !== actorSubject) && <Button size="small" onClick={() => void mutate(() => api.approve(rule.id, rule.expected_version))}>Aprobă</Button>}{canManage && rule.status === "active" && <Button size="small" severity="danger" onClick={() => setRetiring(rule)}>Retrage</Button>}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root>}</div><div className="sticky bottom-0 flex justify-between border-t border-surface pt-2"><span>{page.total} reguli</span><div><Button size="small" disabled={query.page <= 1} onClick={() => setQuery((value: RetentionQuery) => ({ ...value, page: value.page - 1 }))}>Anterior</Button><Button size="small" disabled={query.page >= last} onClick={() => setQuery((value: RetentionQuery) => ({ ...value, page: value.page + 1 }))}>Următor</Button></div></div><Dialog.Root open={Boolean(retiring)} onOpenChange={(event: any) => !event.value && setRetiring(undefined)}><Dialog.Portal><Dialog.Backdrop/><Dialog.Positioner><Dialog.Popup><Dialog.Header><Dialog.Title>Retrage regula de retenție</Dialog.Title></Dialog.Header><Dialog.Content><InputText aria-label="Motiv retragere" value={retirementReason} onChange={(event: ChangeEvent<HTMLInputElement>) => setRetirementReason(event.target.value)}/></Dialog.Content><Dialog.Footer><Button severity="danger" disabled={saving || !retirementReason.trim()} onClick={retire}>Confirmă retragerea</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root></section>;
}


type RetentionForm = {
  taxonomy_node_id: string;
  source_id: string;
  effective_from: string;
  effective_to: string;
  minimum_retention_days: string;
};

function ProposalDialog({ taxonomy, sources, form, setForm, saving, error, onClose, onSave }: {
  taxonomy: Array<{ id: string; label: string }>;
  sources: Array<{ id: string; label: string }>;
  form: RetentionForm;
  setForm: Dispatch<SetStateAction<RetentionForm>>;
  saving: boolean;
  error?: string;
  onClose: () => void;
  onSave: () => void;
}) {
  const days = Number(form.minimum_retention_days);
  const valid = Boolean(form.taxonomy_node_id && form.source_id && form.effective_from)
    && Number.isInteger(days) && days >= 1 && days <= 36500
    && (!form.effective_to || form.effective_to >= form.effective_from);
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => { if (!event.value && !saving) onClose(); }}>
    <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(94vw,36rem)]">
      <Dialog.Header><Dialog.Title>Regulă de retenție arhivă</Dialog.Title><Dialog.Close aria-label="Închide" disabled={saving} /></Dialog.Header>
      <Dialog.Content><div className="flex flex-col gap-3">
        <Message.Root severity="info"><Message.Content><Message.Text>Termenul se stabilește din sursa documentată. Acest flux acceptă numai calculul de la data primirii documentului; nu reprezintă o certificare juridică.</Message.Text></Message.Content></Message.Root>
        {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
        <Select.Root value={form.taxonomy_node_id || null} options={taxonomy} optionLabel="label" optionValue="id" disabled={saving}
          onValueChange={(event: { value: unknown }) => setForm(value => ({ ...value, taxonomy_node_id: String(event.value ?? "") }))}>
          <Select.Trigger aria-label="Serie arhivistică"><Select.Value placeholder="Selectați seria" /><Select.Indicator /></Select.Trigger>
          <Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
        </Select.Root>
        <Select.Root value={form.source_id || null} options={sources} optionLabel="label" optionValue="id" disabled={saving}
          onValueChange={(event: { value: unknown }) => setForm(value => ({ ...value, source_id: String(event.value ?? "") }))}>
          <Select.Trigger aria-label="Sursă documentată"><Select.Value placeholder="Selectați sursa" /><Select.Indicator /></Select.Trigger>
          <Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
        </Select.Root>
        <InputText aria-label="Dată intrare în vigoare" type="date" value={form.effective_from} disabled={saving}
          onChange={(event: ChangeEvent<HTMLInputElement>) => setForm(value => ({ ...value, effective_from: event.target.value }))} />
        <InputText aria-label="Dată expirare" type="date" value={form.effective_to} disabled={saving}
          onChange={(event: ChangeEvent<HTMLInputElement>) => setForm(value => ({ ...value, effective_to: event.target.value }))} />
        <InputText aria-label="Zile retenție" type="number" min="1" max="36500" step="1" value={form.minimum_retention_days} disabled={saving}
          onChange={(event: ChangeEvent<HTMLInputElement>) => setForm(value => ({ ...value, minimum_retention_days: event.target.value }))} />
      </div></Dialog.Content>
      <Dialog.Footer><Button variant="outlined" disabled={saving} onClick={onClose}>Renunță</Button><Button disabled={saving || !valid} onClick={onSave}>{saving ? "Se salvează…" : "Propune regula"}</Button></Dialog.Footer>
    </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
  </Dialog.Root>;
}
