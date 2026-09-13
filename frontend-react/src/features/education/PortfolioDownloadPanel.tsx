import { useEffect, useRef, useState } from "react";
import { Download } from "@primeicons/react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Message } from "@primereact/ui/message";

function exportError(error: unknown): string {
  const code = error instanceof Error ? error.cause : undefined;
  if (code === "education_portfolio_export_provenance_incomplete") return "Exportul nu poate fi pregătit: unele documente nu mai sunt disponibile sau integritatea lor nu poate fi confirmată. Contactați administratorul școlii.";
  if (code === "education_portfolio_export_bundle_limit_exceeded") return "Portofoliul depășește limita unui singur export. Contactați administratorul școlii.";
  if (code === "education_portfolio_export_storage_unavailable") return "Arhiva nu este disponibilă momentan. Încercați din nou.";
  if (error instanceof Error && error.message === "education_request_403") return "Nu aveți dreptul de a exporta acest portofoliu.";
  return "Portofoliul nu a putut fi descărcat. Încercați din nou.";
}

export function PortfolioDownloadPanel({ portfolioID, portfolioCode, canExport, disabled = false, download }: {
  portfolioID: string;
  portfolioCode: string;
  canExport: boolean;
  disabled?: boolean;
  download: (id: string, signal?: AbortSignal) => Promise<Blob>;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const [complete, setComplete] = useState(false);
  const controller = useRef<AbortController | undefined>(undefined);

  useEffect(() => {
    controller.current?.abort();
    setBusy(false); setError(undefined); setComplete(false);
    return () => { controller.current?.abort(); };
  }, [portfolioID, canExport]);

  const start = async () => {
    if (!canExport || disabled || busy) return;
    const current = new AbortController();
    controller.current = current;
    setBusy(true); setError(undefined); setComplete(false);
    try {
      const blob = await download(portfolioID, current.signal);
      if (current.signal.aborted) return;
      const url = URL.createObjectURL(blob);
      const revoke = URL.revokeObjectURL.bind(URL);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `${portfolioCode.replace(/[^\p{L}\p{N}._-]+/gu, "_").slice(0, 120) || "portofoliu"}.zip`;
      document.body.appendChild(anchor);
      try { anchor.click(); } finally { anchor.remove(); window.setTimeout(() => revoke(url), 1000); }
      setComplete(true);
    } catch (cause) {
      if (!current.signal.aborted) setError(exportError(cause));
    } finally {
      if (!current.signal.aborted) setBusy(false);
    }
  };

  if (!canExport) return null;
  return <Card.Root><Card.Body><Card.Title>Export portofoliu</Card.Title><Card.Content><div className="flex flex-col gap-2">
    <p className="m-0">Pachetul ZIP include portofoliul, opisul și documentele.</p>
    <div><Button disabled={disabled || busy} onClick={() => void start()}><Download aria-hidden="true" />{busy ? "Se pregătește exportul…" : "Descarcă portofoliul"}</Button></div>
    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
    {complete && <Message.Root severity="success"><Message.Content><Message.Text>Descărcarea a fost pornită.</Message.Text></Message.Content></Message.Root>}
  </div></Card.Content></Card.Body></Card.Root>;
}
