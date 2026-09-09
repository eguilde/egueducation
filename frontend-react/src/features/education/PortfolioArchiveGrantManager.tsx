import { useCallback, useEffect, useState } from "react";
import { Button } from "@primereact/ui/button";
import { Card } from "@primereact/ui/card";
import { Message } from "@primereact/ui/message";
import { ProgressSpinner } from "@primereact/ui/progressspinner";
import { Select, type SelectValueChangeEvent } from "@primereact/ui/select";
import type {
  EducationApi,
  EligibleGovernanceUser,
  OwnPortfolioArchiveDocument,
  PortfolioAttachmentGrant,
} from "./types";

const Picker = ({
  label,
  value,
  options,
  placeholder,
  onChange,
}: {
  label: string;
  value: string;
  options: Array<{ label: string; value: string }>;
  placeholder: string;
  onChange: (value: string) => void;
}) => (
  <div className="flex min-w-0 flex-1 flex-col gap-1">
    <label>{label}</label>
    <Select.Root
      aria-label={label}
      value={value || null}
      options={options}
      optionLabel="label"
      optionValue="value"
      onValueChange={(event: SelectValueChangeEvent) =>
        onChange(String(event.value ?? ""))
      }
    >
      <Select.Trigger>
        <Select.Value placeholder={placeholder} />
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
  </div>
);

/** Institution-administered, tenant-scoped grants for teacher portfolio evidence. */
export function PortfolioArchiveGrantManager({ api }: { api: EducationApi }) {
  const [documents, setDocuments] = useState<OwnPortfolioArchiveDocument[]>([]);
  const [users, setUsers] = useState<EligibleGovernanceUser[]>([]);
  const [grants, setGrants] = useState<PortfolioAttachmentGrant[]>([]);
  const [documentId, setDocumentId] = useState("");
  const [userId, setUserId] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();

  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const [documentPage, userPage, grantPage] = await Promise.all([
        api.eligibleAttachmentDocuments(),
        api.eligibleAttachmentUsers(),
        api.attachmentGrants(),
      ]);
      setDocuments(documentPage.items);
      setUsers(userPage.items);
      setGrants(grantPage.items);
    } catch {
      setError("Drepturile de atașare nu au putut fi încărcate.");
    } finally {
      setLoading(false);
    }
  }, [api]);

  useEffect(() => {
    void load();
  }, [load]);

  const grant = async () => {
    if (!documentId || !userId) return;
    setSaving(true);
    setError(undefined);
    setNotice(undefined);
    try {
      await api.createAttachmentGrant({
        archive_document_id: documentId,
        grantee_user_id: userId,
      });
      setDocumentId("");
      setUserId("");
      setNotice("Dreptul de atașare a fost acordat.");
      await load();
    } catch {
      setError("Dreptul nu a putut fi acordat. Verificați utilizatorul și starea documentului.");
    } finally {
      setSaving(false);
    }
  };

  const revoke = async (id: string) => {
    setSaving(true);
    setError(undefined);
    setNotice(undefined);
    try {
      await api.deleteAttachmentGrant(id);
      setNotice("Dreptul de atașare a fost revocat.");
      await load();
    } catch {
      setError("Dreptul nu poate fi revocat cât timp este folosit de o dovadă depusă.");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card.Root>
      <Card.Body>
        <Card.Title>Acces documente eArhivă pentru portofolii</Card.Title>
        <Card.Subtitle>
          Acordați explicit unui utilizator dreptul de a folosi un document eligibil ca dovadă în propriul portofoliu.
        </Card.Subtitle>
        <Card.Content>
          <div className="flex flex-col gap-3">
            {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
            {notice && <Message.Root severity="success"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}
            {loading ? (
              <div className="flex justify-center p-4" role="status">
                <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>
              </div>
            ) : (
              <>
                <div className="flex flex-col gap-3 lg:flex-row lg:items-end">
                  <Picker
                    label="Document eArhivă eligibil"
                    value={documentId}
                    options={documents.map((document) => ({ label: `${document.title} · v${document.current_version_no}`, value: document.id }))}
                    placeholder={documents.length ? "Alegeți documentul" : "Nu există documente eligibile"}
                    onChange={setDocumentId}
                  />
                  <Picker
                    label="Utilizator beneficiar"
                    value={userId}
                    options={users.map((user) => ({ label: user.name, value: user.id }))}
                    placeholder={users.length ? "Alegeți utilizatorul" : "Nu există utilizatori eligibili"}
                    onChange={setUserId}
                  />
                  <Button disabled={saving || !documentId || !userId} onClick={() => void grant()}>
                    {saving ? "Se salvează…" : "Acordă acces"}
                  </Button>
                </div>
                <div className="flex flex-col gap-2" aria-label="Drepturi de atașare active">
                  {grants.length === 0 ? (
                    <Message.Root severity="info"><Message.Content><Message.Text>Nu există drepturi de atașare acordate.</Message.Text></Message.Content></Message.Root>
                  ) : grants.map((item) => (
                    <Card.Root key={item.id}>
                      <Card.Body>
                        <Card.Content>
                          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                            <div><strong>{item.document_title}</strong><p>{item.grantee_name}</p></div>
                            <Button size="small" variant="outlined" severity="danger" disabled={saving} onClick={() => void revoke(item.id)}>Revocă</Button>
                          </div>
                        </Card.Content>
                      </Card.Body>
                    </Card.Root>
                  ))}
                </div>
              </>
            )}
          </div>
        </Card.Content>
      </Card.Body>
    </Card.Root>
  );
}
