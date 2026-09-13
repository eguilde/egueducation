import { useEffect, useId, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { DataTable } from "@primereact/ui/datatable";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import { Select } from "@primereact/ui/select";
import { Textarea } from "@primereact/ui/textarea";
import type { SelectValueChangeEvent } from "@primereact/ui/select";
import type { EducationApi, PortfolioLifecycleResult, PortfolioRetentionDispositionPage } from "./types";

type Props = {
  api: Pick<EducationApi, "portfolioRetentionDispositions" | "submitPortfolioRetentionDisposition" | "decidePortfolioRetentionDisposition">;
  portfolioID: string;
  operationID: string;
  transitions: PortfolioLifecycleResult["transitions"];
  canSubmit: boolean;
  canDecide?: boolean;
};

const empty = (): PortfolioRetentionDispositionPage => ({ items: [], total: 0, page: 1, pageSize: 10 });
const statusLabels: Record<string, string> = { submitted: "În analiză", approved: "Aprobată", rejected: "Respinsă", blocked: "Revizuire nouă necesară", closed: "Finalizată" };

export function PortfolioRetentionDispositionPanel({ api, portfolioID, operationID, transitions, canSubmit, canDecide = false }: Props) {
  const titleID = useId();
  const decisionTitleID = useId();
  const [data, setData] = useState(empty);
  const [revision, setRevision] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const [success, setSuccess] = useState<string>();
  const [transitionID, setTransitionID] = useState<string>();
  const [statement, setStatement] = useState("");
  const [reference, setReference] = useState("");
  const [decisionID, setDecisionID] = useState<string>();
  const [approve, setApprove] = useState(true);
  const [decisionReason, setDecisionReason] = useState("");
  const selectedDecision = data.items.find(item => item.id === decisionID);
  const eligible = transitions.filter(item => item.status === "blocked" && item.last_error === "portfolio_retention_expired_review_required");
  useEffect(() => {
    let active = true;
    setBusy(true);
    void api.portfolioRetentionDispositions(portfolioID, { page: 1, pageSize: 10, sort: "requested_at", direction: "desc" })
      .then(value => { if (active) setData(value); })
      .catch(() => { if (active) { setData(empty()); setError("Solicitările de eliberare a retenției nu au putut fi încărcate."); } })
      .finally(() => { if (active) setBusy(false); });
    return () => { active = false; };
  }, [api, portfolioID, revision]);
  const submit = async () => {
    if (!transitionID || !statement.trim() || busy) return;
    setBusy(true); setError(undefined); setSuccess(undefined);
    try {
      await api.submitPortfolioRetentionDisposition(portfolioID, operationID, transitionID, { evidence: { statement: statement.trim(), reference: reference.trim() } });
      setTransitionID(undefined); setStatement(""); setReference(""); setSuccess("Solicitarea a fost înregistrată pentru o decizie independentă."); setRevision(value => value + 1);
    } catch {
      setError("Solicitarea nu a fost confirmată. Protecția arhivei rămâne activă.");
    } finally { setBusy(false); }
  };
  const decide = async () => {
    if (!decisionID || !decisionReason.trim() || busy) return;
    setBusy(true); setError(undefined); setSuccess(undefined);
    try {
      await api.decidePortfolioRetentionDisposition(portfolioID, decisionID, { approve, reason: decisionReason.trim() });
      setDecisionID(undefined); setDecisionReason(""); setSuccess(approve ? "Aprobarea a fost înregistrată; eliberarea exactă va fi verificată asincron." : "Respingerea a fost înregistrată; protecția rămâne activă."); setRevision(value => value + 1);
    } catch { setError("Decizia nu a fost confirmată. Protecția arhivei rămâne activă."); }
    finally { setBusy(false); }
  };
  return <section aria-label="Dispoziții de retenție" className="flex flex-col gap-2">
    <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="m-0 text-base">Dispoziții de retenție</h3><Button variant="outlined" size="small" disabled={busy} onClick={() => setRevision(value => value + 1)}>Reîncarcă</Button></div>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {success && <Message.Root severity="success"><Message.Content><Message.Text>{success}</Message.Text></Message.Content></Message.Root>}
    {canSubmit && eligible.length > 0 && <div className="flex flex-wrap gap-2">{eligible.map(item => <Button key={item.id} size="small" onClick={() => setTransitionID(item.id)}>Solicită analizarea expirării</Button>)}</div>}
    <div className="max-h-56 overflow-auto border border-surface rounded-border"><DataTable.Root data={data.items} dataKey="id" scrollable><DataTable.Table>
      <DataTable.THead className="sticky top-0 z-10"><DataTable.THeadRow><DataTable.THeadCell>Solicitată</DataTable.THeadCell><DataTable.THeadCell>Solicitant</DataTable.THeadCell><DataTable.THeadCell>Dovezi</DataTable.THeadCell><DataTable.THeadCell>Stare</DataTable.THeadCell><DataTable.THeadCell>Decizie</DataTable.THeadCell><DataTable.THeadCell>Rezultat</DataTable.THeadCell><DataTable.THeadCell>Acțiuni</DataTable.THeadCell></DataTable.THeadRow></DataTable.THead>
      <DataTable.TBody>{({ item, index }) => { const row = item as PortfolioRetentionDispositionPage["items"][number]; return <DataTable.Row key={row.id} index={index}><DataTable.Cell>{new Date(row.requested_at).toLocaleString("ro-RO")}</DataTable.Cell><DataTable.Cell>{row.requested_by_subject}</DataTable.Cell><DataTable.Cell><div className="max-w-80"><div>{row.evidence.statement}</div><div>{row.evidence.reference || "Fără referință document"}</div></div></DataTable.Cell><DataTable.Cell>{statusLabels[row.status] ?? row.status}</DataTable.Cell><DataTable.Cell>{row.decision || "—"}</DataTable.Cell><DataTable.Cell>{row.outcome || row.last_error_code || "—"}</DataTable.Cell><DataTable.Cell>{canDecide && row.status === "submitted" ? <Button variant="text" size="small" onClick={() => { setDecisionID(row.id); setApprove(true); setDecisionReason(""); }}>Înregistrează decizia</Button> : "—"}</DataTable.Cell></DataTable.Row>; }}</DataTable.TBody>
    </DataTable.Table></DataTable.Root><div className="sticky bottom-0 border-t border-surface p-2">{busy ? "Se încarcă…" : `${data.total} solicitări`}</div></div>
    <Dialog.Root open={Boolean(transitionID)} onOpenChange={(event: { value?: boolean }) => !event.value && !busy && setTransitionID(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup aria-labelledby={titleID} className="w-[min(94vw,36rem)]"><Dialog.Header><Dialog.Title id={titleID}>Solicitare de analizare a expirării</Dialog.Title><Dialog.Close aria-label="Închide" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">
      <Message.Root severity="warn"><Message.Content><Message.Text>Trimiterea nu eliberează arhiva. Este necesară aprobarea unei alte persoane cu drepturi de custodie și eArhivă.</Message.Text></Message.Content></Message.Root>
      <label className="flex flex-col gap-1"><span>Justificare și dovezi *</span><Textarea value={statement} disabled={busy} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setStatement(event.target.value)} /></label>
      <label className="flex flex-col gap-1"><span>Referință document</span><InputText value={reference} disabled={busy} onChange={(event: ChangeEvent<HTMLInputElement>) => setReference(event.target.value)} /></label>
    </div></Dialog.Content><Dialog.Footer><Button variant="outlined" severity="secondary" disabled={busy} onClick={() => setTransitionID(undefined)}>Renunță</Button><Button disabled={busy || !statement.trim()} onClick={() => void submit()}>{busy ? "Se înregistrează…" : "Trimite pentru aprobare"}</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
    <Dialog.Root open={Boolean(decisionID)} onOpenChange={(event: { value?: boolean }) => !event.value && !busy && setDecisionID(undefined)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup aria-labelledby={decisionTitleID} className="w-[min(94vw,36rem)]"><Dialog.Header><Dialog.Title id={decisionTitleID}>Decizie independentă de retenție</Dialog.Title><Dialog.Close aria-label="Închide decizia" /></Dialog.Header><Dialog.Content><div className="flex flex-col gap-3">
      <Message.Root severity="warn"><Message.Content><Message.Text>Aprobarea permite workerului să verifice și să elibereze exclusiv versiunea WORM indicată de dovezile serverului, numai dacă nu există blocaje juridice curente.</Message.Text></Message.Content></Message.Root>
      {selectedDecision && <div role="group" aria-label="Dovezile solicitării" className="flex flex-col gap-2 rounded-border border border-surface p-3">
        <div><strong>Solicitant:</strong> <span>{selectedDecision.requested_by_subject}</span></div>
        <div><strong>Justificare și dovezi:</strong> <span>{selectedDecision.evidence.statement}</span></div>
        <div><strong>Referință document:</strong> <span>{selectedDecision.evidence.reference || "Fără referință document"}</span></div>
      </div>}
      <label className="flex flex-col gap-1"><span>Decizie *</span><Select.Root value={approve ? "approve" : "reject"} options={[{ label: "Aprobă", value: "approve" }, { label: "Respinge", value: "reject" }]} optionLabel="label" optionValue="value" onValueChange={(event: SelectValueChangeEvent) => setApprove(event.value === "approve")}><Select.Trigger><Select.Value /><Select.Indicator /></Select.Trigger><Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal></Select.Root></label>
      <label className="flex flex-col gap-1"><span>Motivul deciziei *</span><Textarea value={decisionReason} disabled={busy} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setDecisionReason(event.target.value)} /></label>
    </div></Dialog.Content><Dialog.Footer><Button variant="outlined" severity="secondary" disabled={busy} onClick={() => setDecisionID(undefined)}>Renunță</Button><Button disabled={busy || !decisionReason.trim()} onClick={() => void decide()}>{busy ? "Se înregistrează…" : "Înregistrează decizia"}</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>
  </section>;
}
