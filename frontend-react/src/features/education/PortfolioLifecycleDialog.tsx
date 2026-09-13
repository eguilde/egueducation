import { useEffect, useId, useRef, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Dialog } from "@primereact/ui/dialog";
import { InputText } from "@primereact/ui/inputtext";
import { Textarea } from "@primereact/ui/textarea";
import { Message } from "@primereact/ui/message";
import type { EducationRecord } from "./types";

export type PortfolioLifecycleDraft = {
  record: EducationRecord;
  kind: "cessation" | "legal_hold";
  date: string;
  reason: string;
  active: boolean;
};

type Props = {
  value?: PortfolioLifecycleDraft;
  onClose: () => void;
  onSubmit: (input: PortfolioLifecycleDraft) => Promise<void>;
};

export function PortfolioLifecycleDialog({ value, ...props }: Props) {
  return value ? <LifecycleForm key={`${value.record.id}:${value.kind}:${value.active}`} value={value} {...props} /> : null;
}

function LifecycleForm({ value, onClose, onSubmit }: Omit<Props, "value"> & { value: PortfolioLifecycleDraft }) {
  const titleID = useId();
  const [date, setDate] = useState(value.date);
  const [reason, setReason] = useState(value.reason);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const active = useRef(true);
  const submitting = useRef(false);
  useEffect(() => { active.current = true; return () => { active.current = false; }; }, []);
  const title = value.kind === "cessation" ? "Înregistrează încetarea activității" : value.active ? "Aplică blocare juridică" : "Ridică blocarea juridică";
  const valid = reason.trim().length > 0 && (value.kind !== "cessation" || /^\d{4}-\d{2}-\d{2}$/.test(date));
  const submit = async () => {
    if (!valid || submitting.current) return;
    submitting.current = true; setBusy(true); setError(undefined);
    try {
      await onSubmit({ ...value, date, reason: reason.trim() });
      if (active.current) onClose();
    } catch {
      if (active.current) setError("Solicitarea nu a putut fi confirmată. Verificați starea portofoliului și reîncercați.");
    } finally {
      submitting.current = false;
      if (active.current) setBusy(false);
    }
  };
  return <Dialog.Root open onOpenChange={(event: { value?: boolean }) => { if (!event.value && !submitting.current) onClose(); }}>
    <Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup aria-labelledby={titleID}>
      <Dialog.Header><Dialog.Title id={titleID}>{title}</Dialog.Title><Dialog.Close aria-label="Închide operația de ciclu de viață" disabled={busy} /></Dialog.Header>
      <Dialog.Content><div className="flex flex-col gap-3">
        {value.kind === "cessation" && <label className="flex flex-col gap-1"><span>Data încetării *</span><InputText type="date" value={date} disabled={busy} onChange={(event: ChangeEvent<HTMLInputElement>) => setDate(event.target.value)} /></label>}
        <label className="flex flex-col gap-1"><span>Motiv *</span><Textarea value={reason} disabled={busy} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setReason(event.target.value)} /></label>
        {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
      </div></Dialog.Content>
      <Dialog.Footer><div className="flex flex-wrap justify-end gap-2">
        <Button variant="outlined" severity="secondary" disabled={busy} onClick={onClose}>Renunță</Button>
        <Button disabled={!valid || busy} onClick={() => void submit()}>{busy ? "Se trimite solicitarea…" : "Confirmă"}</Button>
      </div></Dialog.Footer>
    </Dialog.Popup></Dialog.Positioner></Dialog.Portal>
  </Dialog.Root>;
}
