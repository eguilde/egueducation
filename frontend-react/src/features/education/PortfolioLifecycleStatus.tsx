import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Message } from "@primereact/ui/message";
import { Textarea } from "@primereact/ui/textarea";
import { PortfolioRetentionDispositionPanel } from "./PortfolioRetentionDispositionPanel";
import type { EducationApi, PortfolioLifecycleResult } from "./types";

type Props = {
  api: Pick<EducationApi, "portfolioLifecycleOperation" | "retryPortfolioLifecycleOperation" | "portfolioRetentionDispositions" | "submitPortfolioRetentionDisposition" | "decidePortfolioRetentionDisposition">;
  initial: PortfolioLifecycleResult;
  canRetry: boolean;
  canSubmitDisposition?: boolean;
  canDecideDisposition?: boolean;
  onUpdate?: (result: PortfolioLifecycleResult) => void;
};

const labels = {
  pending: "În așteptare", processing: "În curs de verificare", completed: "Finalizată",
  blocked: "Blocată — necesită verificare", dead_letter: "Oprită după erori repetate",
};

export function PortfolioLifecycleStatus(props: Props) {
  return <OperationStatus key={props.initial.operation.id} {...props} />;
}

function OperationStatus({ api, initial, canRetry, canSubmitDisposition = false, canDecideDisposition = false, onUpdate }: Props) {
  const updateListener = useRef(onUpdate);
  useEffect(() => { updateListener.current = onUpdate; }, [onUpdate]);
  const [result, setResult] = useState(initial);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [refresh, setRefresh] = useState(0);
  const active = useRef(true);
  const submitting = useRef(false);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const operation = result.operation;
  const waiting = operation.status === "pending" || operation.status === "processing";
  useEffect(() => {
    let disposed = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const read = async () => {
      try {
        const next = await api.portfolioLifecycleOperation(initial.operation.portfolio_id, initial.operation.id);
        if (disposed) return;
        setResult(next); setError(undefined);
        updateListener.current?.(next);
        if (next.operation.status === "pending" || next.operation.status === "processing") timer = setTimeout(() => void read(), 3000);
      } catch {
        if (!disposed) setError("Starea nu poate fi verificată. Nu considerați operația finalizată; reîncărcați starea.");
      }
    };
    void read();
    return () => { disposed = true; if (timer) clearTimeout(timer); };
  }, [api, initial.operation.id, initial.operation.portfolio_id, refresh]);
  const retry = async () => {
    if (!canRetry || !reason.trim() || submitting.current) return;
    submitting.current = true; setBusy(true);
    try {
      const next = await api.retryPortfolioLifecycleOperation(operation.portfolio_id, operation.id, { reason: reason.trim() });
      if (!active.current) return;
      setResult(next); setReason(""); setError(undefined); setRefresh(value => value + 1);
      updateListener.current?.(next);
    } catch {
      if (active.current) setError("Reluarea nu a fost confirmată. Verificați starea și drepturile înainte de o nouă încercare.");
    } finally {
      submitting.current = false;
      if (active.current) setBusy(false);
    }
  };
  const blocked = operation.status === "blocked" || operation.status === "dead_letter";
  return <section aria-label="Starea protecției portofoliului" className="flex flex-col gap-2">
    <Message.Root severity={operation.status === "completed" ? "success" : blocked ? "warn" : "info"}>
      <Message.Content><Message.Text>
        {labels[operation.status]}. Versiuni verificate: {operation.completed_versions} / {operation.total_versions}.
        {waiting && " Solicitarea este înregistrată; aplicarea protecției în arhivă nu este încă finalizată."}
        {blocked && ` Versiuni blocate: ${operation.blocked_versions}. Protecțiile existente sunt păstrate.`}
      </Message.Text></Message.Content>
    </Message.Root>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    <div className="flex flex-wrap gap-2"><Button variant="outlined" severity="secondary" disabled={busy} onClick={() => setRefresh(value => value + 1)}>Reîncarcă starea</Button></div>
    {blocked && canRetry && <div className="flex flex-col gap-2">
      <label className="flex flex-col gap-1"><span>Motivul reluării *</span><Textarea value={reason} disabled={busy} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setReason(event.target.value)} /></label>
      <div><Button disabled={busy || !reason.trim()} onClick={() => void retry()}>{busy ? "Se solicită reluarea…" : "Reia verificarea"}</Button></div>
    </div>}
    {(canSubmitDisposition || canDecideDisposition) && <PortfolioRetentionDispositionPanel api={api} portfolioID={operation.portfolio_id} operationID={operation.id} transitions={result.transitions} canSubmit={canSubmitDisposition} canDecide={canDecideDisposition} />}
  </section>;
}
