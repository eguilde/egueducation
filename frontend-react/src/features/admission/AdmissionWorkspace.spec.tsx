import { PrimeReactProvider } from "@primereact/core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AdmissionWorkspace } from "./AdmissionWorkspace";

const page = { total: 1, page: 1, pageSize: 20 };
const api = {
  listCampaigns: vi.fn(),
  currentDSSRetentionPolicy: vi.fn().mockRejectedValue(new Error("not configured")), proposeRetentionRule: vi.fn(), approveRetentionRule: vi.fn(), configureDSSRetentionPolicy: vi.fn(),
  listCampaignContexts: vi.fn().mockResolvedValue([]), listClasses: vi.fn().mockResolvedValue([]), listAuthorizations: vi.fn().mockResolvedValue([]), createCampaignContext: vi.fn(), listRegulatorySources: vi.fn().mockResolvedValue([]),
  createCampaign: vi.fn(), transitionCampaign: vi.fn(), listCriteria: vi.fn().mockResolvedValue({ ...page, items: [] }), addCriterion: vi.fn(), listDocumentRequirements: vi.fn().mockResolvedValue({ ...page, items: [] }), addDocumentRequirement: vi.fn(),
  listApplications: vi.fn().mockResolvedValue({ ...page, items: [{ id: "a1", application_no: "A-1", campaign_id: "c1", candidate_party_id: "party-1", status: "under_review", consent_snapshot: {}, expected_version: 2 }] }),
  listCandidateParties: vi.fn().mockResolvedValue([]), listStudents: vi.fn().mockResolvedValue([]), createApplication: vi.fn(), transitionApplication: vi.fn(),
  getApplication: vi.fn().mockResolvedValue({ application: { id: "a1", application_no: "A-1", campaign_id: "c1", candidate_party_id: "party-1", status: "under_review", consent_snapshot: {}, expected_version: 2 }, documents: [], assessments: [], decisions: [] }),
  listArchiveVersions: vi.fn().mockResolvedValue([]), assessDocument: vi.fn(), assessCriterion: vi.fn(),
  listDecisions: vi.fn().mockResolvedValue({ ...page, items: [] }), prepareDecision: vi.fn(), finalizeDecision: vi.fn(), cancelLegalPreparation: vi.fn(), getLegalPreparationArtifact: vi.fn().mockResolvedValue(undefined), uploadLegalPreparationArtifact: vi.fn(), enrolApplication: vi.fn(), listAppeals: vi.fn().mockResolvedValue({ ...page, items: [] }), createAppeal: vi.fn(), prepareAppealResolution: vi.fn(), finalizeAppealResolution: vi.fn(), resolveAppeal: vi.fn(), listSignerAuthorizations: vi.fn().mockResolvedValue({ ...page, items: [] }), proposeSignerAuthorization: vi.fn(), approveSignerAuthorization: vi.fn(), revokeSignerAuthorization: vi.fn(),
};

describe("AdmissionWorkspace", () => {
  beforeEach(() => {
    api.listCampaigns.mockReset().mockResolvedValue({ ...page, items: [{ id: "c1", source_id: "source-1", code: "ADM-2026", title: "Clasa pregătitoare", school_year: "2026-2027", status: "open", capacity_limit: 28, student_place_limit: 28, capacity_unit: "students", capacity_basis: {}, shift: "day", opens_on: "2026-04-01", closes_on: "2026-05-01", expected_version: 1 }] });
  });
  it("renders a lazy campaign table and hides create for read-only RBAC", async () => {
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: false, contextManage: false, retentionManage: false, retentionApprove: false, decide: false, appealsManage: false }} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.getByText("Clasa pregătitoare")).toBeInTheDocument());
    expect(screen.getByLabelText("Filtru Campanie")).toBeInTheDocument();
    expect(screen.getByLabelText("Sortează după Campanie")).toBeInTheDocument();
    expect(screen.queryByLabelText("Adaugă campanie")).not.toBeInTheDocument();
  });

  it("exposes the campaign wizard only to admission managers", async () => {
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: true, contextManage: true, retentionManage: true, retentionApprove: true, decide: true, appealsManage: true }} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.getByLabelText("Adaugă campanie")).toBeInTheDocument());
    expect(screen.getByLabelText("Adaugă context autorizat")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText("Configurează retenția DSS"));
    await waitFor(() => expect(api.currentDSSRetentionPolicy).toHaveBeenCalled());
    expect(screen.getByRole("dialog", { name: "Autoritate și politică retenție DSS" })).toBeInTheDocument();
  });

  it("hides PII workspaces without the additional Registratură read capability", async () => {
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: false, manage: true, contextManage: false, retentionManage: false, retentionApprove: false, decide: true, appealsManage: true }} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.getByText("Clasa pregătitoare")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Aplicații" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Decizii și contestații" })).not.toBeInTheDocument();
  });

  it("does not offer context creation without class and offering selector permissions", async () => {
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: true, contextManage: false, retentionManage: false, retentionApprove: false, decide: false, appealsManage: false }} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.getByLabelText("Adaugă campanie")).toBeInTheDocument());
    expect(screen.queryByLabelText("Adaugă context autorizat")).not.toBeInTheDocument();
  });

  it("opens a published campaign through the generated command boundary", async () => {
    api.listCampaigns.mockResolvedValueOnce({ ...page, items: [{ id: "c2", source_id: "source-1", code: "ADM-PUB", title: "Admitere", school_year: "2026-2027", status: "published", capacity_limit: 28, student_place_limit: 28, capacity_unit: "students", capacity_basis: {}, shift: "day", opens_on: "2026-04-01", closes_on: "2026-05-01", expected_version: 4 }] });
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: true, contextManage: true, retentionManage: true, retentionApprove: true, decide: true, appealsManage: true }} /></PrimeReactProvider>);
    const open = await screen.findByLabelText("open ADM-PUB");
    fireEvent.click(open);
    await waitFor(() => expect(api.transitionCampaign).toHaveBeenCalledWith("c2", { status: "open", expected_version: 4 }));
  });

  it("uses the injected adapter for server-side application loading and detail transition path", async () => {
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: false, contextManage: false, retentionManage: false, retentionApprove: false, decide: true, appealsManage: false }} /></PrimeReactProvider>);
    screen.getByRole("button", { name: "Aplicații" }).click();
    await waitFor(() => expect(screen.getByText("A-1")).toBeInTheDocument());
    expect(api.listApplications).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20, sort: "submitted_at" }));
    fireEvent.click(screen.getByLabelText("Deschide A-1"));
    await waitFor(() => expect(api.getApplication).toHaveBeenCalledWith("a1"));
    expect(screen.getByText("Emite decizie")).toBeInTheDocument();
  });

  it("prepares a decision before showing the WORM signing finalization step", async () => {
api.prepareDecision.mockResolvedValue({ retention_policy_id: "11111111-1111-4111-8111-111111111111", retention_rule_version_id: "22222222-2222-4222-8222-222222222222", retention_source_id: "33333333-3333-4333-8333-333333333333", minimum_retention_days: 30, retention_anchor_at: "2026-09-12T10:15:00Z", required_retention_until: "2026-10-12T10:15:00Z", id: "prep-1", artifact_kind: "decision", artifact_id: "a1", application_id: "a1", canonical_payload: { application_id: "a1" }, canonical_payload_sha256: "a".repeat(64), prepared_by_subject: "director-1", prepared_at: "2026-09-12T10:00:00Z", expires_at: "2026-09-12T10:15:00Z", status: "prepared" });
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: false, contextManage: false, retentionManage: false, retentionApprove: false, decide: true, appealsManage: false }} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Aplicații" }));
    await waitFor(() => expect(screen.getByLabelText("Deschide A-1")).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText("Deschide A-1"));
    await waitFor(() => expect(screen.getByText("Emite decizie")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Emite decizie"));
    fireEvent.change(screen.getByLabelText("Număr decizie"), { target: { value: "D-1" } });
    fireEvent.change(screen.getByLabelText("Motivare decizie"), { target: { value: "criterii îndeplinite" } });
    fireEvent.click(screen.getByText("Pregătește pentru semnare"));
    await waitFor(() => expect(api.prepareDecision).toHaveBeenCalledWith("a1", expect.objectContaining({ decision_no: "D-1" })));
    expect(await screen.findByText("Payload principal · SHA-256")).toBeInTheDocument();
    expect(screen.getByText(/Retenție minimă: 30 zile/)).toHaveTextContent("Termenul este stabilit de politica aprobată");
    expect(screen.queryByRole("spinbutton", { name: /retenție/i })).not.toBeInTheDocument();
    expect(screen.queryByText("Emite decizia")).not.toBeInTheDocument();
  });

  it("retries a selected signed PDF with the same idempotency key and enables finalization only when ready", async () => {
    const preparation = { retention_policy_id: "11111111-1111-4111-8111-111111111111", retention_rule_version_id: "22222222-2222-4222-8222-222222222222", retention_source_id: "33333333-3333-4333-8333-333333333333", minimum_retention_days: 30, retention_anchor_at: "2026-09-12T10:15:00Z", required_retention_until: "2026-10-12T10:15:00Z", id: "prep-upload-1", artifact_kind: "decision", artifact_id: "a1", application_id: "a1", canonical_payload: { application_id: "a1" }, canonical_payload_base64: "eyJhcHBsaWNhdGlvbl9pZCI6ImExIn0=", canonical_payload_sha256: "a".repeat(64), prepared_by_subject: "director-1", prepared_at: "2026-09-12T10:00:00Z", expires_at: "2026-09-12T10:15:00Z", status: "prepared" };
    const readyArtifact = { intent_id: "intent-1", preparation_id: "prep-upload-1", artifact_slot: "primary", retention_until: "2026-10-12T10:15:00Z", replayed: false, document: { id: "doc-1", title: "Decizie semnată", original_file_name: "signed.pdf", mime_type: "application/pdf", status: "ready" }, version: { id: "ver-1", document_id: "doc-1", version_no: 1, source_sha256: "b".repeat(64) } };
    api.prepareDecision.mockResolvedValue(preparation);
    api.getLegalPreparationArtifact.mockResolvedValue(undefined);
    api.uploadLegalPreparationArtifact.mockRejectedValueOnce(new Error("temporary")).mockResolvedValueOnce(readyArtifact);
    const view = render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: false, contextManage: false, retentionManage: false, retentionApprove: false, decide: true, appealsManage: false }} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Aplicații" }));
    await waitFor(() => expect(screen.getByLabelText("Deschide A-1")).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText("Deschide A-1"));
    await waitFor(() => expect(screen.getByText("Emite decizie")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Emite decizie"));
    fireEvent.change(screen.getByLabelText("Număr decizie"), { target: { value: "D-2" } });
    fireEvent.change(screen.getByLabelText("Motivare decizie"), { target: { value: "criterii" } });
    fireEvent.click(screen.getByText("Pregătește pentru semnare"));
    const uploadSection = await screen.findByLabelText("PDF semnat · decizie");
    const input = await waitFor(() => {
      const element = uploadSection.querySelector('input[type="file"]') ?? view.container.querySelector('input[type="file"]');
      expect(element).toBeInstanceOf(HTMLInputElement);
      return element as HTMLInputElement;
    });
    fireEvent.change(input, { target: { files: [new File(["%PDF-1.7"], "signed.pdf", { type: "application/pdf" })] } });
    await waitFor(() => expect(screen.getByText("Încarcă documentul semnat")).toBeEnabled());
    fireEvent.click(screen.getByText("Încarcă documentul semnat"));
    await waitFor(() => expect(screen.getByText("Reîncearcă")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Reîncearcă"));
    await waitFor(() => expect(api.uploadLegalPreparationArtifact).toHaveBeenCalledTimes(2));
    expect(api.uploadLegalPreparationArtifact.mock.calls[0][3]).toBe(api.uploadLegalPreparationArtifact.mock.calls[1][3]);
    await waitFor(() => expect(screen.getByText("Versiunea WORM este pregătită pentru finalizare.")).toBeInTheDocument());
    await waitFor(() => expect(screen.getByText("Finalizează decizia semnată")).toBeEnabled());
  });

  it("renders two independent preparation-bound PDF uploads for a favorable appeal", async () => {
    api.listAppeals.mockResolvedValueOnce({ ...page, items: [{ id: "appeal-1", application_id: "a1", decision_id: "decision-1", appeal_no: "C-1", status: "submitted", expected_version: 4, application_no: "A-1" }] });
    api.getApplication.mockResolvedValue({ application: { id: "a1", application_no: "A-1", campaign_id: "c1", candidate_party_id: "party-1", status: "admitted", consent_snapshot: {}, expected_version: 7 }, documents: [], assessments: [], decisions: [] });
    api.prepareAppealResolution.mockResolvedValue({ retention_policy_id: "11111111-1111-4111-8111-111111111111", retention_rule_version_id: "22222222-2222-4222-8222-222222222222", retention_source_id: "33333333-3333-4333-8333-333333333333", minimum_retention_days: 30, retention_anchor_at: "2026-09-12T10:15:00Z", required_retention_until: "2026-10-12T10:15:00Z", id: "prep-appeal-1", artifact_kind: "appeal_resolution", artifact_id: "appeal-1", application_id: "a1", appeal_id: "appeal-1", resulting_decision_id: "decision-2", policy_evaluation_v2_id: "policy-1", canonical_payload: { appeal_id: "appeal-1" }, canonical_payload_base64: "eyJhcHBlYWxfaWQiOiJhcHBlYWwtMSJ9", canonical_payload_sha256: "a".repeat(64), resulting_decision_payload: { decision_id: "decision-2" }, resulting_decision_payload_base64: "eyJkZWNpc2lvbl9pZCI6ImRlY2lzaW9uLTIifQ==", resulting_decision_payload_sha256: "b".repeat(64), prepared_by_subject: "director-2", prepared_at: "2026-09-12T10:00:00Z", expires_at: "2026-09-12T10:15:00Z", status: "prepared" });
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: false, contextManage: false, retentionManage: false, retentionApprove: false, decide: false, appealsManage: true }} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Decizii și contestații" }));
    await waitFor(() => expect(screen.getByText("C-1")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Soluționează"));
    fireEvent.change(screen.getByLabelText("Motivare soluționare"), { target: { value: "reevaluare favorabilă" } });
    fireEvent.click(screen.getByText("Pregătește pentru semnare"));
    await waitFor(() => expect(api.prepareAppealResolution).toHaveBeenCalledWith("appeal-1", expect.objectContaining({ rationale: "reevaluare favorabilă" })));
    expect(await screen.findByLabelText("PDF semnat · soluție contestație")).toBeInTheDocument();
    expect(screen.getByLabelText("PDF semnat · decizie rezultată")).toBeInTheDocument();
  });

  it("runs the submit and review happy path only through injected commands", async () => {
    api.getApplication.mockReset()
      .mockResolvedValueOnce({ application: { id: "a1", application_no: "A-1", campaign_id: "c1", candidate_party_id: "party-1", status: "draft", consent_snapshot: {}, expected_version: 2 }, documents: [], assessments: [], decisions: [] })
      .mockResolvedValue({ application: { id: "a1", application_no: "A-1", campaign_id: "c1", candidate_party_id: "party-1", status: "submitted", consent_snapshot: {}, expected_version: 3 }, documents: [], assessments: [], decisions: [] });
    render(<PrimeReactProvider><AdmissionWorkspace api={api} capabilities={{ read: true, piiRead: true, manage: true, contextManage: true, retentionManage: true, retentionApprove: true, decide: false, appealsManage: false }} /></PrimeReactProvider>);
    fireEvent.click(screen.getByRole("button", { name: "Aplicații" }));
    await waitFor(() => expect(screen.getByLabelText("Deschide A-1")).toBeInTheDocument());
    fireEvent.click(screen.getByLabelText("Deschide A-1"));
    await waitFor(() => expect(screen.getByText("Depune")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Depune"));
    await waitFor(() => expect(api.transitionApplication).toHaveBeenCalledWith("a1", { status: "submitted", expected_version: 2 }));
    await waitFor(() => expect(screen.getByText("Începe verificarea")).toBeInTheDocument());
    fireEvent.click(screen.getByText("Începe verificarea"));
    await waitFor(() => expect(api.transitionApplication).toHaveBeenLastCalledWith("a1", { status: "under_review", expected_version: 3 }));
  });
});
