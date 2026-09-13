import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { FileUpload } from "@primereact/ui/fileupload";
import { InputText } from "@primereact/ui/inputtext";
import { Message } from "@primereact/ui/message";
import type { EducationApi, OwnPortfolioArchiveDocument } from "./types";

type PortfolioOwnUploadProps = {
  api: Pick<EducationApi, "ownPortfolioArchiveDocuments" | "uploadOwnPortfolioArchiveDocument">; portfolioID: string; onReady: (items: OwnPortfolioArchiveDocument[]) => void;
};

export function PortfolioOwnUpload(props: PortfolioOwnUploadProps) {
  return <PortfolioUploadForm key={props.portfolioID} {...props} />;
}

function uploadError(cause: unknown): string {
  const code = cause instanceof Error ? cause.cause : undefined;
  if (code === "archive_idempotency_conflict") return "Datele diferă de încărcarea inițială. Verificați fișierul, titlul și data înainte de a începe o încărcare nouă.";
  if (code === "portfolio_upload_authority_changed" || code === "portfolio_upload_forbidden" || (cause instanceof Error && cause.message === "education_request_403")) return "Nu mai aveți dreptul de a încărca în acest portofoliu. Reîncărcați pagina pentru a verifica starea și permisiunile.";
  return "Încărcarea nu a fost confirmată. Reîncercați cu același fișier.";
}

function uploadAccessRevoked(cause: unknown): boolean {
  return cause instanceof Error && (cause.message === "education_request_403" || cause.cause === "portfolio_upload_authority_changed" || cause.cause === "portfolio_upload_forbidden");
}

function PortfolioUploadForm({ api, portfolioID, onReady }: PortfolioOwnUploadProps) {
  const [file, setFile] = useState<File>();
  const [title, setTitle] = useState("");
  const [date, setDate] = useState("");
  const [busy, setBusy] = useState(false);
  const [accessRevoked, setAccessRevoked] = useState(false);
  const [error, setError] = useState<string>();
  const [documentID, setDocumentID] = useState<string>();
  const [ready, setReady] = useState(false);
  const [refresh, setRefresh] = useState(0);
  const key = useRef<string | undefined>(undefined);
  const mounted = useRef(true);
  const readyCallback = useRef(onReady);
  readyCallback.current = onReady;
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const reset = () => { key.current = undefined; setDocumentID(undefined); setReady(false); setError(undefined); };
  useEffect(() => {
    if (!documentID || ready) return;
    let active = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let attempts = 0;
    const check = async () => {
      try {
        let page = 1;
        while (active) {
          const result = await api.ownPortfolioArchiveDocuments({ page, pageSize: 100, sort: "title", direction: "asc", filters: { title: title.trim() } });
          if (!active) return;
          if (result.items.some((item) => item.id === documentID && item.current_version_no > 0)) {
            readyCallback.current(result.items); setReady(true); setError(undefined); return;
          }
          if (page * 100 >= result.total) break;
          page += 1;
        }
        if (active && ++attempts < 20) timer = setTimeout(() => void check(), 3000);
      } catch { if (active) setError("Starea procesării nu poate fi verificată. Folosiți Verifică starea."); }
    };
    void check();
    return () => { active = false; if (timer) clearTimeout(timer); };
  }, [api, documentID, ready, refresh, title]);
  const upload = async () => {
    if (!file || !title.trim() || busy || documentID || accessRevoked) return;
    key.current ??= crypto.randomUUID();
    setBusy(true); setError(undefined);
    try {
      const result = await api.uploadOwnPortfolioArchiveDocument(portfolioID, { file, title: title.trim(), ...(date ? { document_date: date } : {}) }, key.current);
      if (mounted.current) setDocumentID(result.id);
    } catch (cause) { if (mounted.current) { setError(uploadError(cause)); setAccessRevoked(uploadAccessRevoked(cause)); } }
    finally { if (mounted.current) setBusy(false); }
  };
  return <div className="flex flex-col gap-2" aria-label="Încărcare PDF propriu">
    <FileUpload.Root customUpload accept="application/pdf,.pdf" maxFileSize={100 * 1024 * 1024} disabled={busy || accessRevoked} onSelect={(event: { files: File[] }) => {
      reset(); setFile(event.files[0]); setTitle(event.files[0]?.name.replace(/\.pdf$/i, "") ?? "");
    }}><FileUpload.Trigger>Alege PDF propriu</FileUpload.Trigger><FileUpload.Content /></FileUpload.Root>
    <InputText aria-label="Titlu PDF încărcat" value={title} maxLength={300} disabled={busy || accessRevoked || Boolean(documentID)} onChange={(event: ChangeEvent<HTMLInputElement>) => { reset(); setTitle(event.target.value); }} />
    <InputText aria-label="Data PDF încărcat" type="date" value={date} disabled={busy || accessRevoked || Boolean(documentID)} onChange={(event: ChangeEvent<HTMLInputElement>) => { reset(); setDate(event.target.value); }} />
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {documentID ? <><Message.Root severity={ready ? "success" : "info"}><Message.Content><Message.Text>{ready ? "PDF-ul este disponibil în lista de documente autorizate." : "PDF primit. Procesarea este în curs; documentul nu poate fi atașat încă."}</Message.Text></Message.Content></Message.Root>{!ready && <Button variant="outlined" onClick={() => setRefresh((value) => value + 1)}>Verifică starea</Button>}</> : <Button disabled={!file || !title.trim() || busy || accessRevoked} onClick={() => void upload()}>{busy ? "Se încarcă…" : accessRevoked ? "Încărcare indisponibilă" : error ? "Reîncearcă încărcarea PDF" : "Încarcă PDF propriu"}</Button>}
  </div>;
}
