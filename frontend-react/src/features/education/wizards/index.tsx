import { useEffect, useMemo, useState, type ChangeEvent } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { InputText } from "@primereact/ui/inputtext";
import { Select } from "@primereact/ui/select";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Checkbox } from "@primereact/ui/checkbox";

export type WizardValues = Record<string, string | number | boolean>;
export interface EducationWizardAdapter {
  create(path: string, payload: WizardValues): Promise<unknown>;
  eligibleGovernanceUsers?: () => Promise<Array<{ id: string; name: string }>>;
}
export interface EducationWizardProps {
  adapter: EducationWizardAdapter;
  canManage?: boolean;
  canSelfManage?: boolean;
  onSaved?: (value: unknown) => void;
}
export type WizardDefinition = {
  title: string;
  path: string;
  parentKey?: string;
  permission: "manage" | "self-manage";
  steps: string[];
  fields: Field[];
  initial: WizardValues;
  validate: (v: WizardValues) => string[];
};
type Field = {
  key: string;
  label: string;
  required?: boolean;
  type?: "text" | "number" | "select" | "checkbox";
  options?: { label: string; value: string }[];
};
const opt = (values: string[]) =>
  values.map((value) => ({ label: value, value }));
const required = (v: WizardValues, keys: string[]) =>
  keys
    .filter((k) => String(v[k] ?? "").trim() === "")
    .map((k) => `${k} este obligatoriu`);
const def = (
  title: string,
  path: string,
  permission: WizardDefinition["permission"],
  steps: string[],
  initial: WizardValues,
  fields: Field[],
  keys: string[],
): WizardDefinition => ({
  title,
  path,
  permission,
  steps,
  initial,
  fields,
  validate: (v) => required(v, keys),
});
const currentSchoolYear = () => {
  const now = new Date();
  const year = now.getMonth() >= 7 ? now.getFullYear() : now.getFullYear() - 1;
  return `${year}-${year + 1}`;
};
const governanceStatus = opt(["draft", "scheduled", "held", "published"]);
const personnelStatus = opt(["active", "on_leave", "vacant", "inactive"]);
const evaluationStatus = opt([
  "draft",
  "submitted",
  "reviewed",
  "approved",
  "contested",
]);
const declarationStatus = opt(["draft", "submitted", "validated", "expired"]);
const mobilityStatus = opt([
  "open",
  "pending",
  "approved",
  "rejected",
  "completed",
]);
const meritStatus = opt([
  "draft",
  "submitted",
  "evaluated",
  "approved",
  "funded",
]);

export const wizardDefinitions: Record<string, WizardDefinition> = {
  caMeeting: def(
    "Ședință CA/CP/CEAC",
    "/education/governance/meetings",
    "manage",
    ["Configurare", "Coordonare", "Rezumat", "Creare"],
    {
      school_year: currentSchoolYear(),
      organism: "ca",
      title: "",
      meeting_type: "ordinary",
      status: "draft",
      quorum_required: 0,
      participants_count: 0,
      meeting_date: "",
      location: "",
      chairperson_user_id: "",
      secretary_user_id: "",
      summary: "",
    },
    [
      { key: "school_year", label: "An școlar", required: true },
      {
        key: "organism",
        label: "Organism",
        required: true,
        type: "select",
        options: opt(["ca", "cp", "ceac", "cfdcd"]),
      },
      { key: "title", label: "Titlu", required: true },
      {
        key: "meeting_type",
        label: "Tip",
        type: "select",
        options: opt(["ordinary", "extraordinary"]),
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: governanceStatus,
      },
      { key: "quorum_required", label: "Cvorum", type: "number" },
      { key: "participants_count", label: "Participanți", type: "number" },
      { key: "meeting_date", label: "Data", required: true },
      { key: "location", label: "Locație" },
      {
        key: "chairperson_user_id",
        label: "Președinte",
        required: true,
        type: "select",
      },
      {
        key: "secretary_user_id",
        label: "Secretar",
        required: true,
        type: "select",
      },
      { key: "summary", label: "Rezumat" },
    ],
    [
      "school_year",
      "organism",
      "title",
      "meeting_date",
      "chairperson_user_id",
      "secretary_user_id",
    ],
  ),
  minute: {
    ...def(
      "Punct de minută",
      "/education/governance/meetings/{parent}/minutes",
      "manage",
      ["Ședință și subiect", "Conținut", "Urmărire", "Confirmare"],
      {
        meeting_id: "",
        agenda_order: 1,
        topic_title: "",
        discussion_summary: "",
        decision_summary: "",
        responsible_party: "",
        due_on: "",
        follow_up_status: "open",
        requires_publication: false,
        notes: "",
      },
      [
        { key: "meeting_id", label: "Ședință", required: true },
        {
          key: "agenda_order",
          label: "Ordine pe agendă",
          required: true,
          type: "number",
        },
        { key: "topic_title", label: "Subiect", required: true },
        {
          key: "discussion_summary",
          label: "Rezumat discuții",
          required: true,
        },
        { key: "decision_summary", label: "Decizie", required: true },
        { key: "responsible_party", label: "Responsabil" },
        { key: "due_on", label: "Termen" },
        {
          key: "follow_up_status",
          label: "Urmărire",
          type: "select",
          options: opt(["open", "in_progress", "completed", "cancelled"]),
        },
        {
          key: "requires_publication",
          label: "Necesită publicare",
          type: "checkbox",
        },
        { key: "notes", label: "Note" },
      ],
      ["meeting_id", "topic_title", "discussion_summary", "decision_summary"],
    ),
    parentKey: "meeting_id",
  },
  vote: {
    ...def(
      "Vot al ședinței",
      "/education/governance/meetings/{parent}/votes",
      "manage",
      ["Ședință și subiect", "Rezultat", "Temei", "Confirmare"],
      {
        meeting_id: "",
        subject_title: "",
        agenda_order: 1,
        decision_type: "resolution",
        votes_for: 0,
        votes_against: 0,
        abstentions: 0,
        outcome: "pending",
        requires_follow_up: false,
        legal_basis: "",
        notes: "",
      },
      [
        { key: "meeting_id", label: "Ședință", required: true },
        { key: "subject_title", label: "Subiect", required: true },
        { key: "agenda_order", label: "Ordine pe agendă", type: "number" },
        {
          key: "decision_type",
          label: "Tip decizie",
          type: "select",
          options: opt(["resolution", "opinion", "approval", "information"]),
        },
        { key: "votes_for", label: "Pentru", type: "number" },
        { key: "votes_against", label: "Împotrivă", type: "number" },
        { key: "abstentions", label: "Abțineri", type: "number" },
        {
          key: "outcome",
          label: "Rezultat",
          type: "select",
          options: opt(["pending", "adopted", "rejected", "tied"]),
        },
        {
          key: "requires_follow_up",
          label: "Necesită urmărire",
          type: "checkbox",
        },
        { key: "legal_basis", label: "Temei legal" },
        { key: "notes", label: "Note" },
      ],
      ["meeting_id", "subject_title"],
    ),
    parentKey: "meeting_id",
  },
  resolution: {
    ...def(
      "Hotărâre",
      "/education/governance/meetings/{parent}/resolutions",
      "manage",
      ["Ședință și vot", "Act", "Publicare", "Confirmare"],
      {
        meeting_id: "",
        vote_id: "",
        title: "",
        resolution_type: "hotarare",
        publication_status: "draft",
        anonymization_state: "not_required",
        issued_on: "",
        signed_by: "",
        notes: "",
      },
      [
        { key: "meeting_id", label: "Ședință", required: true },
        { key: "vote_id", label: "Vot", required: true },
        { key: "title", label: "Titlu", required: true },
        {
          key: "resolution_type",
          label: "Tip act",
          type: "select",
          options: opt(["hotarare", "decizie", "aviz", "proces_verbal"]),
        },
        { key: "issued_on", label: "Data emiterii", required: true },
        {
          key: "publication_status",
          label: "Publicare",
          type: "select",
          options: opt(["draft", "pending", "published", "restricted"]),
        },
        {
          key: "anonymization_state",
          label: "Anonimizare",
          type: "select",
          options: opt(["not_required", "pending", "completed"]),
        },
        { key: "signed_by", label: "Semnat de" },
        { key: "notes", label: "Note" },
      ],
      ["meeting_id", "vote_id", "title", "issued_on"],
    ),
    parentKey: "meeting_id",
  },
  managerial: def(
    "Dosar managerial",
    "/education/managerial/records",
    "manage",
    ["Identificare", "Responsabilitate", "Publicare", "Confirmare"],
    {
      school_year: currentSchoolYear(),
      dossier_type: "director_portfolio",
      title: "",
      status: "draft",
      owner_name: "",
      due_on: "",
      publication_required: false,
      summary: "",
    },
    [
      { key: "school_year", label: "An școlar", required: true },
      {
        key: "dossier_type",
        label: "Tip dosar",
        required: true,
        type: "select",
        options: opt([
          "director_portfolio",
          "adjunct_director_portfolio",
          "pdi",
          "pas",
          "quality_report",
        ]),
      },
      { key: "title", label: "Titlu", required: true },
      {
        key: "status",
        label: "Stare",
        type: "select",
        options: opt([
          "draft",
          "in_review",
          "approved",
          "published",
          "archived",
        ]),
      },
      { key: "owner_name", label: "Responsabil", required: true },
      { key: "due_on", label: "Termen", required: true },
      {
        key: "publication_required",
        label: "Necesită publicare",
        type: "checkbox",
      },
      { key: "summary", label: "Rezumat" },
    ],
    ["school_year", "dossier_type", "title", "owner_name", "due_on"],
  ),
  personnel: def(
    "Cadru didactic",
    "/education/personnel/records",
    "manage",
    ["Identitate", "Urmărire", "Contact", "Confirmare"],
    {
      full_name: "",
      role_title: "",
      employment_type: "titular",
      status: "active",
      evaluation_status: "not_started",
      mobility_stage: "none",
      school_year: currentSchoolYear(),
      assigned_unit: "",
      phone: "",
      email: "",
      has_portfolio: false,
      notes: "",
    },
    [
      { key: "full_name", label: "Nume complet", required: true },
      { key: "role_title", label: "Funcție", required: true },
      {
        key: "employment_type",
        label: "Încadrare",
        type: "select",
        options: opt(["titular", "suplinitor", "plata_cu_ora", "auxiliar"]),
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: personnelStatus,
      },
      {
        key: "evaluation_status",
        label: "Evaluare",
        type: "select",
        options: opt(["not_started", "in_progress", "finalized"]),
      },
      {
        key: "mobility_stage",
        label: "Mobilitate",
        type: "select",
        options: opt(["none", "pretransfer", "transfer", "restrangere"]),
      },
      { key: "school_year", label: "An școlar", required: true },
      { key: "assigned_unit", label: "Structură" },
      { key: "phone", label: "Telefon" },
      { key: "email", label: "Email" },
      { key: "has_portfolio", label: "Are portofoliu", type: "checkbox" },
      { key: "notes", label: "Note" },
    ],
    ["full_name", "role_title", "school_year"],
  ),
  evaluation: def(
    "Evaluare anuală",
    "/education/evaluations/records",
    "manage",
    ["Cadru", "Rezultat", "Evaluator", "Confirmare"],
    {
      employee_code: "",
      full_name: "",
      role_title: "",
      school_year: currentSchoolYear(),
      status: "draft",
      score: 0,
      qualification: "",
      evaluator_name: "",
      finalized_on: "",
      summary: "",
    },
    [
      { key: "employee_code", label: "Cod angajat", required: true },
      { key: "full_name", label: "Nume", required: true },
      { key: "role_title", label: "Funcție" },
      { key: "school_year", label: "An școlar", required: true },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: evaluationStatus,
      },
      { key: "score", label: "Punctaj", type: "number" },
      {
        key: "qualification",
        label: "Calificativ",
        type: "select",
        options: opt(["foarte_bine", "bine", "satisfacator", "nesatisfacator"]),
      },
      { key: "evaluator_name", label: "Evaluator", required: true },
      { key: "finalized_on", label: "Finalizat la" },
      { key: "summary", label: "Rezumat" },
    ],
    ["employee_code", "full_name", "school_year", "evaluator_name"],
  ),
  declaration: def(
    "Declarație",
    "/education/declarations/records",
    "manage",
    ["Titular", "Calendar", "Rezumat", "Confirmare"],
    {
      employee_code: "",
      full_name: "",
      declaration_type: "authenticity",
      status: "draft",
      school_year: currentSchoolYear(),
      submitted_on: "",
      valid_until: "",
      summary: "",
    },
    [
      { key: "employee_code", label: "Cod angajat", required: true },
      { key: "full_name", label: "Nume", required: true },
      {
        key: "declaration_type",
        label: "Tip",
        type: "select",
        options: opt([
          "authenticity",
          "consent",
          "conflict_of_interest",
          "certificate",
        ]),
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: declarationStatus,
      },
      { key: "school_year", label: "An școlar", required: true },
      { key: "submitted_on", label: "Depus la", required: true },
      { key: "valid_until", label: "Valabil până la" },
      { key: "summary", label: "Rezumat" },
    ],
    ["employee_code", "full_name", "school_year", "submitted_on"],
  ),
  mobility: def(
    "Mobilitate",
    "/education/mobility/records",
    "manage",
    ["Titular", "Status", "Unități", "Confirmare"],
    {
      employee_code: "",
      full_name: "",
      school_year: currentSchoolYear(),
      request_type: "transfer",
      stage: "draft",
      status: "open",
      source_school: "",
      destination_school: "",
      submitted_on: "",
      reviewed_by: "",
      notes: "",
    },
    [
      { key: "employee_code", label: "Cod angajat", required: true },
      { key: "full_name", label: "Nume", required: true },
      { key: "school_year", label: "An școlar", required: true },
      {
        key: "request_type",
        label: "Tip solicitare",
        type: "select",
        options: opt(["transfer", "pretransfer", "detasare", "restrangere"]),
      },
      {
        key: "stage",
        label: "Etapă",
        type: "select",
        options: opt(["draft", "submitted", "review", "approved", "completed"]),
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: mobilityStatus,
      },
      { key: "source_school", label: "Unitate sursă", required: true },
      { key: "destination_school", label: "Destinație" },
      { key: "submitted_on", label: "Depus la", required: true },
      { key: "reviewed_by", label: "Analizat de" },
      { key: "notes", label: "Note" },
    ],
    [
      "employee_code",
      "full_name",
      "school_year",
      "source_school",
      "submitted_on",
    ],
  ),
  merit: def(
    "Gradație de merit",
    "/education/gradatii/records",
    "manage",
    ["Candidat", "Evaluare", "Decizie", "Confirmare"],
    {
      full_name: "",
      role_title: "",
      school_year: currentSchoolYear(),
      category: "predare",
      status: "draft",
      score: 0,
      committee_name: "",
      decision_date: "",
      funded: false,
      notes: "",
    },
    [
      { key: "full_name", label: "Nume", required: true },
      { key: "role_title", label: "Funcție", required: true },
      { key: "school_year", label: "An școlar", required: true },
      {
        key: "category",
        label: "Categorie",
        type: "select",
        options: opt(["predare", "management", "consiliere", "auxiliar"]),
      },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: meritStatus,
      },
      { key: "score", label: "Punctaj", type: "number" },
      { key: "committee_name", label: "Comisie", required: true },
      { key: "decision_date", label: "Data deciziei", required: true },
      { key: "funded", label: "Finanțat", type: "checkbox" },
      { key: "notes", label: "Note" },
    ],
    [
      "full_name",
      "role_title",
      "school_year",
      "committee_name",
      "decision_date",
    ],
  ),
  portfolio: def(
    "Portofoliu CD",
    "/education/portfolios/records",
    "manage",
    ["Titular", "Structură", "Conformitate", "Confirmare"],
    {
      owner_name: "",
      owner_role: "",
      school_year: currentSchoolYear(),
      status: "draft",
      section_count: 0,
      last_updated_on: "",
      retention_until: "",
      transfer_status: "none",
      authenticity_declared: false,
      consent_captured: false,
      custodian: "",
      notes: "",
    },
    [
      { key: "owner_name", label: "Titular", required: true },
      { key: "owner_role", label: "Funcție", required: true },
      { key: "school_year", label: "An școlar", required: true },
      {
        key: "status",
        label: "Status",
        type: "select",
        options: opt([
          "draft",
          "submitted",
          "validated",
          "transferred",
          "archived",
        ]),
      },
      { key: "section_count", label: "Secțiuni", type: "number" },
      { key: "last_updated_on", label: "Actualizat la", required: true },
      { key: "retention_until", label: "Retenție până la", required: true },
      {
        key: "transfer_status",
        label: "Transfer",
        type: "select",
        options: opt(["none", "prepared", "sent", "received"]),
      },
      {
        key: "authenticity_declared",
        label: "Autenticitate",
        type: "checkbox",
      },
      { key: "consent_captured", label: "Consimțământ", type: "checkbox" },
      { key: "custodian", label: "Custode" },
      { key: "notes", label: "Note" },
    ],
    [
      "owner_name",
      "owner_role",
      "school_year",
      "last_updated_on",
      "retention_until",
    ],
  ),
};

export function buildWizardPayload(values: WizardValues): WizardValues {
  return Object.fromEntries(
    Object.entries(values).map(([key, value]) => [
      key,
      typeof value === "string" ? value.trim() : value,
    ]),
  );
}
function fieldStep(
  index: number,
  fieldCount: number,
  totalSteps: number,
): number {
  const inputSteps = Math.max(1, totalSteps - 1);
  const fieldsPerStep = Math.ceil(fieldCount / inputSteps);
  return Math.min(inputSteps - 1, Math.floor(index / fieldsPerStep));
}

export function EducationWizard({
  definition,
  adapter,
  canManage = false,
  canSelfManage = false,
  onSaved,
}: EducationWizardProps & { definition: WizardDefinition }) {
  const allowed =
    definition.permission === "manage" ? canManage : canManage || canSelfManage;
  const [values, setValues] = useState<WizardValues>(() => {
    const initial = { ...definition.initial };
    if (definition.parentKey && typeof window !== "undefined") {
      const routeValue = new URLSearchParams(window.location.search).get(
        "meetingId",
      );
      if (routeValue) initial[definition.parentKey] = routeValue;
    }
    return initial;
  });
  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [eligibleUsers, setEligibleUsers] = useState<
    Array<{ id: string; name: string }>
  >([]);
  useEffect(() => {
    if (
      definition !== wizardDefinitions.caMeeting ||
      !adapter.eligibleGovernanceUsers
    )
      return;
    let current = true;
    void adapter
      .eligibleGovernanceUsers()
      .then((users) => {
        if (current) setEligibleUsers(users);
      })
      .catch(() => {
        if (current)
          setError("Utilizatorii eligibili nu au putut fi încărcați.");
      });
    return () => {
      current = false;
    };
  }, [adapter, definition]);
  const update = (key: string, value: string | number | boolean) =>
    setValues((v) => ({ ...v, [key]: value }));
  const effectiveFields = useMemo(
    () =>
      definition.fields.map((field) =>
        ["chairperson_user_id", "secretary_user_id"].includes(field.key)
          ? {
              ...field,
              options: eligibleUsers.map((user) => ({
                label: user.name,
                value: user.id,
              })),
            }
          : field,
      ),
    [definition.fields, eligibleUsers],
  );
  const visibleFields = effectiveFields.filter(
    (_, index) =>
      fieldStep(index, effectiveFields.length, definition.steps.length) ===
      step,
  );
  const next = () => {
    const missing = visibleFields.filter(
      (f) => f.required && String(values[f.key] ?? "").trim() === "",
    );
    if (missing.length) {
      setError(missing.map((f) => `${f.label} este obligatoriu`).join("; "));
      return;
    }
    setError("");
    setStep(Math.min(definition.steps.length - 1, step + 1));
  };
  const submit = async () => {
    setError("");
    if (!allowed)
      return setError("Nu aveți drepturi pentru această operațiune.");
    const validationErrors = definition.validate(values);
    if (validationErrors.length) return setError(validationErrors.join("; "));
    setBusy(true);
    try {
      const payload = buildWizardPayload(values);
      let path = definition.path;
      if (definition.parentKey) {
        const parent = String(payload[definition.parentKey] ?? "").trim();
        path = path.replace("{parent}", encodeURIComponent(parent));
        delete payload[definition.parentKey];
      }
      const created = await adapter.create(path, payload);
      onSaved?.(created);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Salvarea a eșuat.");
    } finally {
      setBusy(false);
    }
  };
  return (
    <Card.Root className="w-full">
      <Card.Body>
        <Card.Content>
          <div className="flex flex-col gap-4">
            <div>
              <h2 className="m-0 text-xl font-semibold">{definition.title}</h2>
              <div className="mt-2 flex flex-wrap gap-2">
                {definition.steps.map((label, i) => (
                  <span
                    key={label}
                    className={
                      i === step ? "font-semibold" : "text-muted-color"
                    }
                  >
                    {i + 1}. {label}
                  </span>
                ))}
              </div>
            </div>
            {error && (
              <Message.Root severity="error">
                <Message.Content>
                  <Message.Text>{error}</Message.Text>
                </Message.Content>
              </Message.Root>
            )}
            {!allowed && (
              <Message.Root severity="warn">
                <Message.Content>
                  <Message.Text>
                    Accesul necesită permisiunea corespunzătoare.
                  </Message.Text>
                </Message.Content>
              </Message.Root>
            )}
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              {visibleFields.map((f) => (
                <label key={f.key} className="flex flex-col gap-1">
                  <span>
                    {f.label}
                    {f.required ? " *" : ""}
                  </span>
                  {f.type === "select" ? (
                    <Select.Root
                      value={values[f.key] as string}
                      options={f.options}
                      optionLabel="label"
                      optionValue="value"
                      onValueChange={(e: { value: unknown }) =>
                        update(f.key, String(e.value ?? ""))
                      }
                    >
                      <Select.Trigger>
                        <Select.Value />
                        <Select.Indicator />
                      </Select.Trigger>
                      <Select.Portal>
                        <Select.Positioner>
                          <Select.Popup>
                            <Select.List />
                          </Select.Popup>
                        </Select.Positioner>
                      </Select.Portal>
                    </Select.Root>
                  ) : f.type === "number" ? (
                    <InputText
                      aria-label={f.label}
                      type="number"
                      value={String(values[f.key] ?? "")}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        update(f.key, Number(e.target.value))
                      }
                    />
                  ) : f.type === "checkbox" ? (
                    <Checkbox.Root
                      checked={Boolean(values[f.key])}
                      onCheckedChange={() =>
                        update(f.key, !Boolean(values[f.key]))
                      }
                    >
                      <Checkbox.Box>
                        <Checkbox.Indicator />
                      </Checkbox.Box>
                    </Checkbox.Root>
                  ) : (
                    <InputText
                      aria-label={f.label}
                      value={String(values[f.key] ?? "")}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        update(f.key, e.target.value)
                      }
                    />
                  )}
                </label>
              ))}
            </div>
            <div className="flex flex-wrap justify-end gap-2">
              <Button
                severity="secondary"
                variant="outlined"
                disabled={step === 0}
                onClick={() => setStep(Math.max(0, step - 1))}
              >
                Înapoi
              </Button>
              <Button
                loading={busy || undefined}
                disabled={!allowed}
                onClick={step === definition.steps.length - 1 ? submit : next}
              >
                {step === definition.steps.length - 1 ? "Salvează" : "Continuă"}
              </Button>
              {busy && (
                <ProgressSpinner.Root>
                  <ProgressSpinner.Range>
                    <ProgressSpinner.Track />
                    <ProgressSpinner.Value />
                  </ProgressSpinner.Range>
                </ProgressSpinner.Root>
              )}
            </div>
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}
export const CaMeetingWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.caMeeting} />
);
export const MeetingMinuteWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.minute} />
);
export const MeetingVoteWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.vote} />
);
export const MeetingResolutionWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.resolution} />
);
export const ManagerialDossierWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.managerial} />
);
export const PersonnelRecordWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.personnel} />
);
export const EvaluationWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.evaluation} />
);
export const DeclarationWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.declaration} />
);
export const MobilityWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.mobility} />
);
export const MeritWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.merit} />
);
export const PortfolioRecordWizard = (p: EducationWizardProps) => (
  <EducationWizard {...p} definition={wizardDefinitions.portfolio} />
);
