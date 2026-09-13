import { useCallback, useEffect, useRef, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { Select } from "@primereact/ui/select";
import { Textarea } from "@primereact/ui/textarea";
import { Tag } from "@primereact/ui/tag";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import { PortfolioLifecycleDialog, type PortfolioLifecycleDraft } from "./PortfolioLifecycleDialog";
import { PortfolioLifecycleStatus } from "./PortfolioLifecycleStatus";
import { PortfolioLifecycleHistory } from "./PortfolioLifecycleHistory";
import type { PortfolioLifecycleResult } from "./types";
import type { EducationApi, EducationListQuery, EducationPage, EducationRecord, PortfolioManagerialDecisionInput, PortfolioReturnForCorrectionsInput, PortfolioReviewEvent } from "./types";

const today = () => new Date().toISOString().slice(0, 10);
const correctionInitial = (): PortfolioReturnForCorrectionsInput => ({ reviewed_on: today(), missing_documents: 0, compliance_score: 0, notes: "" });
const managerialInitial = (): PortfolioManagerialDecisionInput => ({ reviewed_on: today(), outcome: "acceptat", missing_documents: 0, compliance_score: 0, notes: "" });
const emptyReviews = (): EducationPage<PortfolioReviewEvent> => ({ items: [], total: 0, page: 1, pageSize: 20 });
const reviewQuery = { page: 1, pageSize: 20, sort: "reviewed_on", direction: "desc" as const };
const portfolioFilterFields = ["owner_name", "school_year", "status", "last_updated_on"] as const;
type PortfolioFilterField = typeof portfolioFilterFields[number];

export function PortfolioReviewWorkspace({ api, canManage, canReturn, canManageLifecycle = false, canDecideRetention = false }: { api: EducationApi; canManage: boolean; canReturn: boolean; canManageLifecycle?: boolean; canDecideRetention?: boolean }) {
  const [lifecycle, setLifecycle] = useState<PortfolioLifecycleDraft>();
  const [lifecycleResult, setLifecycleResult] = useState<PortfolioLifecycleResult>();
  const [historyID, setHistoryID] = useState<string>();
  useEffect(() => { if (!canManageLifecycle) setLifecycle(undefined); }, [canManageLifecycle]);
  const [page, setPage] = useState<EducationPage<EducationRecord>>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [query, setQuery] = useState<EducationListQuery>({ page: 1, pageSize: 20, q: "", sort: "", direction: "asc", filters: {} });
  const [selected, setSelected] = useState<EducationRecord>();
  const [detailReady, setDetailReady] = useState(false);
  const [reviews, setReviews] = useState<EducationPage<PortfolioReviewEvent>>({ items: [], total: 0, page: 1, pageSize: 20 });
  const [dialog, setDialog] = useState<"return" | "managerial">();
  const [correction, setCorrection] = useState<PortfolioReturnForCorrectionsInput>(correctionInitial);
  const [managerial, setManagerial] = useState<PortfolioManagerialDecisionInput>(managerialInitial);
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const selectionRequest = useRef(0);
  const selectedID = useRef<string | undefined>(undefined);

  const load = useCallback(async () => { setLoading(true); setError(undefined); try { setPage(await api.records("portfolios", query)); } catch { setError("Portofoliile nu au putut fi încărcate."); } finally { setLoading(false); } }, [api, query]);
  useEffect(() => { void load(); }, [load]);
  const choose = async (item: EducationRecord) => {
    const request = ++selectionRequest.current;
    selectedID.current = item.id;
    setSelected(item);
    setHistoryID(undefined);
    setDetailReady(false);
    setReviews(emptyReviews());
    setError(undefined);
    try {
      const [detail, nextReviews] = await Promise.all([
        api.recordDetail("portfolios", item.id),
        api.portfolioReviews(item.id, reviewQuery),
      ]);
      if (request === selectionRequest.current) {
        setSelected({ ...item, ...detail });
        setDetailReady(true);
        setReviews(nextReviews);
      }
    } catch {
      if (request === selectionRequest.current) setError("Detaliul portofoliului sau istoricul verificării nu a putut fi încărcat.");
    }
  };
  const refresh = async (updated: EducationRecord) => {
    if (selectedID.current === updated.id) await choose(updated);
    await load();
  };
  const returnForCorrections = async () => { if (!selected || !canReturn || !correction.notes.trim() || !correction.reviewed_on) return; const portfolio = selected; setLoading(true); setError(undefined); try { const updated = await api.returnPortfolioForCorrections(portfolio.id, correction); setDialog(undefined); await refresh(updated); } catch { setError("Portofoliul nu a putut fi returnat pentru completări."); } finally { setLoading(false); } };
  const recordManagerialDecision = async () => { if (!selected || !canManage || !managerial.notes.trim() || !managerial.reviewed_on) return; const portfolio = selected; setLoading(true); setError(undefined); try { const updated = await api.recordPortfolioManagerialDecision(portfolio.id, managerial); setDialog(undefined); await refresh(updated); } catch { setError("Decizia managerială nu a putut fi înregistrată."); } finally { setLoading(false); } };
  const setCorrectionField = (key: keyof PortfolioReturnForCorrectionsInput, value: string | number) => setCorrection((current) => ({ ...current, [key]: value }));
  const setManagerialField = (key: keyof PortfolioManagerialDecisionInput, value: string | number) => setManagerial((current) => ({ ...current, [key]: value }));
  const setFilter = (field: PortfolioFilterField, value: string) => setQuery((current) => ({ ...current, page: 1, filters: { ...current.filters, [field]: value || undefined } }));

  return <section aria-label="Verificare portofolii" className="flex flex-col gap-3">
    {selected && detailReady && <Button variant="outlined" onClick={() => setHistoryID(selected.id)}>Istoric operații de protecție</Button>}
    {historyID && selected?.id === historyID && detailReady && <PortfolioLifecycleHistory api={api} portfolioID={historyID} canRetry={canManageLifecycle} canDecideDisposition={canDecideRetention} onClose={() => setHistoryID(undefined)} />}
    {selected && detailReady && canManageLifecycle && <div className="flex flex-wrap gap-2" aria-label="Protecția portofoliului selectat">
      <Button variant="outlined" disabled={Boolean(selected.activity_ceased_on)} onClick={() => setLifecycle({ record: selected, kind: "cessation", date: "", reason: "", active: false })}>Înregistrează încetarea activității</Button>
      <Button variant="outlined" severity="warn" onClick={() => setLifecycle({ record: selected, kind: "legal_hold", date: "", reason: "", active: !Boolean(selected.legal_hold_active) })}>{selected.legal_hold_active ? "Ridică blocarea juridică" : "Aplică blocare juridică"}</Button>
    </div>}
    <PortfolioLifecycleDialog value={canManageLifecycle ? lifecycle : undefined} onClose={() => setLifecycle(undefined)} onSubmit={async input => {
      if (!canManageLifecycle) throw new Error("portfolio_lifecycle_permission_revoked");
      const result = input.kind === "cessation"
        ? await api.recordPortfolioCessation(input.record.id, { activity_ceased_on: input.date, reason: input.reason })
        : await api.setPortfolioLegalHold(input.record.id, { active: input.active, reason: input.reason });
      setLifecycleResult(result);
      await refresh(result.portfolio);
    }} />
    {lifecycleResult && <PortfolioLifecycleStatus api={api} initial={lifecycleResult} canRetry={canManageLifecycle} canDecideDisposition={canDecideRetention} />}
    <Card.Root><Card.Body><Card.Title>Verificare portofolii profesori</Card.Title><Card.Content><div className="flex flex-wrap items-end gap-2"><label className="flex flex-col gap-1"><span>Caută</span><InputText value={query.q ?? ""} onChange={(e: ChangeEvent<HTMLInputElement>) => setQuery((current) => ({ ...current, q: e.target.value, page: 1 }))} /></label><Button variant="outlined" onClick={() => void load()} disabled={loading}>Aplică</Button></div></Card.Content></Card.Body></Card.Root>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    <div className="min-h-64 overflow-hidden rounded-border border-surface"><DataTable.Root data={page.items as Record<string, unknown>[]} dataKey="id" scrollable className="max-h-[min(60dvh,42rem)] overflow-auto"><DataTable.Table><DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow>{([['owner_name','Profesor'],['school_year','An școlar'],['status','Stare'],['last_updated_on','Actualizat']] as const).map(([field, label]) => <DataTable.THeadCell key={field}><Button variant="text" size="small" aria-label={`Sortează după ${label}`} onClick={() => setQuery((current) => ({ ...current, sort: field, direction: current.sort === field && current.direction === "asc" ? "desc" : "asc", page: 1 }))}>{label}{query.sort === field ? (query.direction === "asc" ? " ↑" : " ↓") : ""}</Button></DataTable.THeadCell>)}<DataTable.THeadCell>Acțiuni</DataTable.THeadCell></DataTable.THeadRow><DataTable.THeadRow>{([['owner_name','Profesor'],['school_year','An școlar'],['status','Stare'],['last_updated_on','Actualizat']] as const).map(([field, label]) => <DataTable.THeadCell key={field}><InputText aria-label={`Filtrează ${label}`} type={field === "last_updated_on" ? "date" : "text"} value={query.filters?.[field] ?? ""} onChange={(event: ChangeEvent<HTMLInputElement>) => setFilter(field, event.target.value)} /></DataTable.THeadCell>)}<DataTable.THeadCell aria-label="Filtre acțiuni" /></DataTable.THeadRow></DataTable.THead><DataTable.TBody>{({ item, index }) => { const row = item as EducationRecord; return <DataTable.Row key={row.id} index={index}><DataTable.Cell>{String(row.owner_name ?? "—")}</DataTable.Cell><DataTable.Cell>{String(row.school_year ?? "—")}</DataTable.Cell><DataTable.Cell><Tag value={String(row.status ?? "—")} /></DataTable.Cell><DataTable.Cell>{String(row.last_updated_on ?? "—")}</DataTable.Cell><DataTable.Cell><Button size="small" variant="text" onClick={() => void choose(row)}>Detalii</Button></DataTable.Cell></DataTable.Row>; }}</DataTable.TBody></DataTable.Table></DataTable.Root></div>
    <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-2 border-t border-surface pt-2" aria-label="Paginare portofolii"><span>{page.total ? `${(page.page - 1) * page.pageSize + 1} – ${Math.min(page.page * page.pageSize, page.total)} din ${page.total}` : "0 rezultate"}</span><div className="flex gap-2"><Button size="small" variant="outlined" disabled={page.page <= 1 || loading} onClick={() => setQuery((current) => ({ ...current, page: page.page - 1 }))}>Anterior</Button><Button size="small" variant="outlined" disabled={page.page * page.pageSize >= page.total || loading} onClick={() => setQuery((current) => ({ ...current, page: page.page + 1 }))}>Următor</Button></div></div>
    {selected && <Card.Root><Card.Body><Card.Title>{String(selected.owner_name ?? selected.id)}</Card.Title><Card.Content><div className="flex flex-wrap justify-between gap-2"><Tag aria-label={`Stare portofoliu selectat: ${String(selected.status ?? "—")}`} value={String(selected.status ?? "—")} />{canManage && <Button onClick={() => { setManagerial(managerialInitial()); setDialog("managerial"); }}>Decizie managerială</Button>}{canReturn && <Button severity="warn" variant="outlined" disabled={loading} onClick={() => { setCorrection(correctionInitial()); setDialog("return"); }}>Returnează pentru completări</Button>}</div><div className="mt-3 flex flex-col gap-2">{reviews.items.length ? reviews.items.map((review) => <div className="rounded-border border border-surface p-2" key={review.id}><strong>{review.review_stage}</strong> · {review.outcome} · {review.reviewer_name}<p>{review.notes || "—"}</p></div>) : <Message.Root severity="info"><Message.Content><Message.Text>Nu există verificări înregistrate.</Message.Text></Message.Content></Message.Root>}</div></Card.Content></Card.Body></Card.Root>}
    <Dialog.Root open={dialog === "return"} onOpenChange={(event: { value?: boolean }) => !event.value && setDialog(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,40rem)]"><Dialog.Header><Dialog.Title>Returnează pentru completări</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Evaluatorul și etapa sunt preluate din sesiunea autentificată.</Message.Text></Message.Content></Message.Root><InputText aria-label="Data verificării" type="date" value={correction.reviewed_on} onChange={(e: ChangeEvent<HTMLInputElement>) => setCorrectionField("reviewed_on", e.target.value)} /><InputText aria-label="Documente lipsă" type="number" min="0" value={String(correction.missing_documents)} onChange={(e: ChangeEvent<HTMLInputElement>) => setCorrectionField("missing_documents", Number(e.target.value))} /><InputText aria-label="Scor conformitate" type="number" min="0" max="100" value={String(correction.compliance_score)} onChange={(e: ChangeEvent<HTMLInputElement>) => setCorrectionField("compliance_score", Number(e.target.value))} /><Textarea aria-label="Observații" value={correction.notes} onChange={(e: ChangeEvent<HTMLTextAreaElement>) => setCorrectionField("notes", e.target.value)} /></div></Dialog.Content><Dialog.Footer><Button variant="outlined" severity="secondary" onClick={() => setDialog(undefined)}>Renunță</Button><Button disabled={loading || !correction.notes.trim() || !correction.reviewed_on} onClick={() => void returnForCorrections()}>Returnează</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
    <Dialog.Root open={dialog === "managerial"} onOpenChange={(event: { value?: boolean }) => !event.value && setDialog(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,40rem)]"><Dialog.Header><Dialog.Title>Decizie managerială</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3"><Message.Root severity="info"><Message.Content><Message.Text>Evaluatorul și etapa sunt preluate din sesiunea autentificată.</Message.Text></Message.Content></Message.Root><Select.Root aria-label="Rezultat" value={managerial.outcome} options={[{ label: "Acceptă", value: "acceptat" }, { label: "Respinge", value: "respins" }]} optionLabel="label" optionValue="value" onValueChange={(e: SelectValueChangeEvent) => setManagerialField("outcome", String(e.value) as PortfolioManagerialDecisionInput["outcome"])}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root><InputText aria-label="Data verificării" type="date" value={managerial.reviewed_on} onChange={(e: ChangeEvent<HTMLInputElement>) => setManagerialField("reviewed_on", e.target.value)} /><InputText aria-label="Documente lipsă" type="number" min="0" value={String(managerial.missing_documents)} onChange={(e: ChangeEvent<HTMLInputElement>) => setManagerialField("missing_documents", Number(e.target.value))} /><InputText aria-label="Scor conformitate" type="number" min="0" max="100" value={String(managerial.compliance_score)} onChange={(e: ChangeEvent<HTMLInputElement>) => setManagerialField("compliance_score", Number(e.target.value))} /><Textarea aria-label="Observații" value={managerial.notes} onChange={(e: ChangeEvent<HTMLTextAreaElement>) => setManagerialField("notes", e.target.value)} /></div></Dialog.Content><Dialog.Footer><Button variant="outlined" severity="secondary" onClick={() => setDialog(undefined)}>Renunță</Button><Button disabled={loading || !managerial.notes.trim() || !managerial.reviewed_on} onClick={() => void recordManagerialDecision()}>Înregistrează decizia</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
  </section>;
}
