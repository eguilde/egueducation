import { useEffect, useRef, useState } from "react";
import { Card } from "@primereact/ui/card";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";

export type SecretariatCockpit = {
  classes: number;
  students: number;
  active_enrolments: number;
  portfolios_in_review: number;
  institution_id: string;
};

export type HRCockpit = {
  personnel: number;
  expiring_documents: number;
  expired_documents: number;
  pending_evaluations: number;
  institution_id: string;
};

export type CommitteeCockpit = {
  committees: number;
  active_members: number;
  meetings: number;
  evidence_documents: number;
  institution_id: string;
};

export type InspectorCockpit = {
  readiness_open: number;
  pending_publications: number;
  mandatory_publication_pending: number;
  requirements_pending: number;
  evaluations_in_review: number;
  institution_id: string;
};

export type RoleCockpitData =
  | { kind: "secretariat"; value: SecretariatCockpit }
  | { kind: "hr"; value: HRCockpit }
  | { kind: "committee"; value: CommitteeCockpit }
  | { kind: "inspector"; value: InspectorCockpit };

export type RoleCockpitKind = RoleCockpitData["kind"];
export type RoleCockpitLoader = () => Promise<RoleCockpitData>;

type Metric = { label: string; value: number };

const cockpitCopy: Record<RoleCockpitKind, { title: string; description: string; permission: string }> = {
  secretariat: {
    title: "Cockpit secretariat",
    description: "Situația operațională a claselor, elevilor, înscrierilor și portofoliilor din instituția activă.",
    permission: "education.cockpit.secretariat.read",
  },
  hr: {
    title: "Cockpit resurse umane",
    description: "Situația personalului, documentelor și evaluărilor din instituția activă.",
    permission: "education.cockpit.hr.read",
  },
  committee: {
    title: "Cockpit responsabil comisie",
    description: "Situația comisiilor, membrilor, ședințelor și dovezilor instituției active.",
    permission: "education.cockpit.committee.read",
  },
  inspector: {
    title: "Cockpit inspector",
    description: "Indicatori de pregătire, publicare, cerințe și evaluări pentru instituția activă.",
    permission: "education.cockpit.inspector.read",
  },
};

function metrics(data: RoleCockpitData): Metric[] {
  switch (data.kind) {
    case "secretariat":
      return [
        { label: "Clase", value: data.value.classes },
        { label: "Elevi", value: data.value.students },
        { label: "Înscrieri active", value: data.value.active_enrolments },
        { label: "Portofolii în verificare", value: data.value.portfolios_in_review },
      ];
    case "hr":
      return [
        { label: "Personal", value: data.value.personnel },
        { label: "Documente care expiră", value: data.value.expiring_documents },
        { label: "Documente expirate", value: data.value.expired_documents },
        { label: "Evaluări în așteptare", value: data.value.pending_evaluations },
      ];
    case "committee":
      return [
        { label: "Comisii", value: data.value.committees },
        { label: "Membri activi", value: data.value.active_members },
        { label: "Ședințe", value: data.value.meetings },
        { label: "Documente doveditoare", value: data.value.evidence_documents },
      ];
    case "inspector":
      return [
        { label: "Pregătiri deschise", value: data.value.readiness_open },
        { label: "Publicări în așteptare", value: data.value.pending_publications },
        { label: "Publicări obligatorii restante", value: data.value.mandatory_publication_pending },
        { label: "Cerințe neacoperite", value: data.value.requirements_pending },
        { label: "Evaluări în verificare", value: data.value.evaluations_in_review },
      ];
  }
}

export function RoleCockpit({
  kind,
  allowed,
  load,
}: {
  kind: RoleCockpitKind;
  allowed: boolean;
  load: RoleCockpitLoader;
}) {
  const [data, setData] = useState<RoleCockpitData>();
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(allowed);
  const request = useRef(0);
  const copy = cockpitCopy[kind];

  useEffect(() => {
    if (!allowed) {
      setLoading(false);
      return;
    }
    const current = ++request.current;
    setLoading(true);
    setError(undefined);
    void load()
      .then((result) => {
        if (current === request.current && result.kind === kind) setData(result);
      })
      .catch(() => {
        if (current === request.current) setError("Indicatorii nu au putut fi încărcați. Încercați din nou.");
      })
      .finally(() => {
        if (current === request.current) setLoading(false);
      });
  }, [allowed, kind, load]);

  if (!allowed) {
    return <Message.Root severity="warn"><Message.Content><Message.Text>Nu aveți dreptul {copy.permission} pentru instituția activă.</Message.Text></Message.Content></Message.Root>;
  }
  if (loading) {
    return <div className="flex justify-center p-8" role="status"><ProgressSpinner.Root aria-label="Se încarcă indicatorii"><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root></div>;
  }
  if (error) {
    return <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>;
  }
  if (!data) return null;
  const values = metrics(data);
  const empty = values.every((metric) => metric.value === 0);
  return <Card.Root><Card.Body><Card.Title>{copy.title}</Card.Title><Card.Subtitle>{copy.description}</Card.Subtitle><Card.Content><div className="flex flex-col gap-3">{empty && <Message.Root severity="info"><Message.Content><Message.Text>Nu există înregistrări pentru indicatorii acestui cockpit în instituția activă.</Message.Text></Message.Content></Message.Root>}<div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{values.map((metric) => <Card.Root key={metric.label}><Card.Body><Card.Title>{metric.label}</Card.Title><Card.Content><strong>{metric.value}</strong></Card.Content></Card.Body></Card.Root>)}</div></div></Card.Content></Card.Body></Card.Root>;
}
