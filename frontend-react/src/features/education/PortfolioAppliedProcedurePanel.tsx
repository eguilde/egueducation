import { useEffect, useRef, useState } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Dialog } from "@primereact/ui/dialog";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Book, Times } from "@primeicons/react";
import type { OwnPortfolioAppliedProcedure, PortfolioProcedure, PortfolioProcedureRule } from "./types";

export type { OwnPortfolioAppliedProcedure } from "./types";

const formatValue = (value: unknown): string => typeof value === "string" || typeof value === "number" || typeof value === "boolean" ? String(value) : value === null ? "—" : "";
const label = (key: string) => ({ calendar_rules: "Calendar", access_rules: "Acces", accepted_formats: "Formate acceptate", retention_rules: "Păstrare", transfer_rules: "Transfer" }[key] ?? key.replaceAll("_", " "));

function RuleValue({ value }: { value: unknown }) {
  if (Array.isArray(value)) return value.length ? <ol className="m-0 flex list-decimal flex-col gap-1 pl-5">{value.map((item, index) => <li key={index}><RuleValue value={item} /></li>)}</ol> : <span>—</span>;
  if (value && typeof value === "object") return <dl className="m-0 grid gap-2 sm:grid-cols-2">{Object.entries(value as Record<string, unknown>).map(([key, item]) => <div key={key} className="min-w-0 rounded-border border border-surface p-2"><dt className="text-sm text-muted-color">{label(key)}</dt><dd className="m-0 break-words"><RuleValue value={item} /></dd></div>)}</dl>;
  return <span>{formatValue(value) || "—"}</span>;
}

function ProcedureMetadata({ procedure }: { procedure: PortfolioProcedure }) {
  const ruleGroups = [
    ["calendar_rules", procedure.calendar_rules], ["access_rules", procedure.access_rules], ["accepted_formats", procedure.accepted_formats], ["retention_rules", procedure.retention_rules], ["transfer_rules", procedure.transfer_rules],
  ] as const;
  return <div className="flex flex-col gap-4"><dl className="grid gap-2 sm:grid-cols-2"><div><dt>Cod</dt><dd className="m-0">{procedure.procedure_code}</dd></div><div><dt>Versiune</dt><dd className="m-0">v{procedure.version_no}</dd></div><div className="sm:col-span-2"><dt>Titlu</dt><dd className="m-0">{procedure.title}</dd></div><div className="sm:col-span-2"><dt>Sursă</dt><dd className="m-0 break-words">{procedure.source_ref}</dd></div><div><dt>Aplicabilă de la</dt><dd className="m-0">{procedure.effective_from ?? "—"}</dd></div><div><dt>Aplicabilă până la</dt><dd className="m-0">{procedure.effective_to ?? "—"}</dd></div></dl>{ruleGroups.map(([key, value]) => <section key={key} aria-label={label(key)} className="flex flex-col gap-2"><h3 className="m-0 text-base font-semibold">{label(key)}</h3><RuleValue value={value} /></section>)}</div>;
}

function AppliedRules({ rules }: { rules: PortfolioProcedureRule[] }) {
  return <section aria-label="Reguli de secțiune aplicate" className="flex flex-col gap-2"><h3 className="m-0 text-base font-semibold">Reguli de secțiune aplicate</h3><p className="m-0 text-sm text-muted-color">Secțiunile prevăzute de procedura școlii.</p>{rules.length ? <ol className="m-0 flex list-none flex-col gap-2 p-0">{[...rules].sort((left, right) => left.sort_order - right.sort_order).map((rule) => <li key={rule.id} className="rounded-border border border-surface p-2"><dl className="grid gap-1 sm:grid-cols-2"><div><dt>Secțiune</dt><dd className="m-0">{rule.section_code}</dd></div><div><dt>Ordine</dt><dd className="m-0">{rule.sort_order}</dd></div><div><dt>Denumire</dt><dd className="m-0">{rule.label_ro}</dd></div><div><dt>Obligatorie</dt><dd className="m-0">{rule.required ? "Da" : "Nu"}</dd></div><div><dt>Activă</dt><dd className="m-0">{rule.active ? "Da" : "Nu"}</dd></div><div><dt>Catalog sursă</dt><dd className="m-0">{rule.source_catalog_version}</dd></div></dl></li>)}</ol> : <Message.Root severity="info"><Message.Content><Message.Text>Procedura aplicată nu conține reguli de secțiune.</Message.Text></Message.Content></Message.Root>}</section>;
}

/** Read-only procedure snapshot resolved by the server for the authenticated teacher's portfolio. */
export function PortfolioAppliedProcedurePanel({ portfolioID, load, disabled = false }: { portfolioID: string; disabled?: boolean; load: (portfolioID: string) => Promise<OwnPortfolioAppliedProcedure> }) {
  const [open, setOpen] = useState(false);
  const [data, setData] = useState<OwnPortfolioAppliedProcedure>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();
  const request = useRef(0);

  useEffect(() => {
    const current = ++request.current;
    setData(undefined);
    setError(undefined);
    if (!open) { setLoading(false); return; }
    setLoading(true);
    void load(portfolioID).then((result) => {
      if (current === request.current) setData(result);
    }).catch((cause: unknown) => {
      if (current !== request.current) return;
      const code = cause instanceof Error && typeof cause.cause === "string" ? cause.cause : "";
      setError(code === "education_own_portfolio_procedure_not_applied" ? "Acest portofoliu mai vechi nu are încă o procedură aplicată." : "Procedura aplicată nu a putut fi încărcată. Încercați din nou.");
    }).finally(() => {
      if (current === request.current) setLoading(false);
    });
    return () => { request.current++; };
  }, [load, open, portfolioID]);

  return <Card.Root><Card.Body><Card.Title>Procedură aplicată</Card.Title><Card.Subtitle>Consultați procedura școlii aplicată acestui portofoliu.</Card.Subtitle><Card.Content><Button disabled={disabled} variant="outlined" severity="secondary" onClick={() => setOpen(true)}><Book aria-hidden="true" />Procedura aplicată</Button></Card.Content></Card.Body>{open && <Dialog.Root open onOpenChange={(event: { value?: boolean }) => !event.value && setOpen(false)}><Dialog.Portal><Dialog.Backdrop /><Dialog.Positioner><Dialog.Popup className="w-[min(96vw,64rem)]"><Dialog.Header><Dialog.Title>Procedura aplicată portofoliului</Dialog.Title><Dialog.Close aria-label="Închide procedura aplicată"><Times aria-hidden="true" /></Dialog.Close></Dialog.Header><Dialog.Content><div className="flex flex-col gap-4">{loading && <div className="flex justify-center p-4" role="status" aria-label="Se încarcă procedura aplicată"><ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div>}{error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}{data && <><ProcedureMetadata procedure={data.procedure} /><AppliedRules rules={data.rules} /></>}</div></Dialog.Content><Dialog.Footer><Button variant="outlined" severity="secondary" onClick={() => setOpen(false)}>Închide</Button></Dialog.Footer></Dialog.Popup></Dialog.Positioner></Dialog.Portal></Dialog.Root>}</Card.Root>;
}
