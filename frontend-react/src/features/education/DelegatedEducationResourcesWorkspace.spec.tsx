import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { DelegatedEducationResourcesWorkspace, delegatedResources } from "./DelegatedEducationResourcesWorkspace";
import type { EducationDelegationGrant } from "../../auth/AuthProvider";

const grant = (resource_type: EducationDelegationGrant["resource_type"], resource_id: string, permission_code = "education.portfolios.read"): EducationDelegationGrant => ({ resource_type, resource_id, permission_code });

function api() {
  return {
    governanceMeetingDetail: vi.fn().mockResolvedValue({ id: "meeting-1", meeting_code: "CA-01", status: "draft" }),
    recordDetail: vi.fn().mockResolvedValue({ id: "portfolio-1", title: "Portofoliu profesor", status: "active" }),
    saveGovernanceMeeting: vi.fn(),
    updateRecord: vi.fn(),
  };
}

function view(grants: readonly EducationDelegationGrant[], client = api(), onRefresh?: () => Promise<void>, institutionID?: string) {
  return {
    client,
    ...render(<PrimeReactProvider><DelegatedEducationResourcesWorkspace grants={grants} api={client} onRefresh={onRefresh} institutionID={institutionID} /></PrimeReactProvider>),
  };
}

describe("DelegatedEducationResourcesWorkspace", () => {
  it("keeps institution grants out of the exact-resource inbox", () => {
    expect(delegatedResources([grant("institution", "institution-1")])).toEqual([]);
  });

  it("loads only the exact portfolio identifier, never a global collection", async () => {
    const { client } = view([grant("portfolio", "portfolio-1")]);
    await screen.findByText("Portofoliu profesor");
    expect(client.recordDetail).toHaveBeenCalledWith("portfolios", "portfolio-1");
    expect(client.recordDetail).toHaveBeenCalledTimes(1);
    expect(client.governanceMeetingDetail).not.toHaveBeenCalled();
  });

  it("does not fetch an identifier for which no active grant exists", async () => {
    const { client } = view([grant("portfolio", "portfolio-1")]);
    await screen.findByText("Portofoliu profesor");
    expect(client.recordDetail).not.toHaveBeenCalledWith("portfolios", "portfolio-b");
  });

  it("uses the literal meeting detail endpoint for a delegated meeting", async () => {
    const { client } = view([grant("meeting", "meeting-1", "education.governance.read")]);
    await screen.findByText("CA-01");
    expect(client.governanceMeetingDetail).toHaveBeenCalledWith("meeting-1");
    expect(client.recordDetail).not.toHaveBeenCalled();
  });

  it("fails closed when the exact detail request is forbidden", async () => {
    const client = api();
    client.recordDetail.mockRejectedValueOnce(new Error("forbidden"));
    view([grant("portfolio", "portfolio-1")], client);
    await screen.findByText("Nu există resurse delegate active pentru utilizatorul curent.");
    expect(screen.queryByText("Portofoliu profesor")).not.toBeInTheDocument();
  });

  it("clears the view when the refreshed grant snapshot is revoked", async () => {
    const client = api();
    const onRefresh = vi.fn().mockResolvedValue(undefined);
    const result = view([grant("portfolio", "portfolio-1")], client, onRefresh);
    await screen.findByText("Portofoliu profesor");
    result.rerender(<PrimeReactProvider><DelegatedEducationResourcesWorkspace grants={[]} api={client} onRefresh={onRefresh} /></PrimeReactProvider>);
    await screen.findByText("Nu există resurse delegate active pentru utilizatorul curent.");
    expect(screen.queryByText("Portofoliu profesor")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Actualizează" }));
    await waitFor(() => expect(onRefresh).toHaveBeenCalledOnce());
  });

  it("shows details but does not turn a manage grant into a global edit route", async () => {
    const { client } = view([grant("portfolio", "portfolio-1", "education.portfolios.manage")]);
    await screen.findByText("Portofoliu profesor");
    fireEvent.click(screen.getByRole("button", { name: "Detalii" }));
    expect(await screen.findByText(/detaliul nu conține toate câmpurile necesare/i)).toBeInTheDocument();
    expect(client.recordDetail).toHaveBeenCalledWith("portfolios", "portfolio-1");
  });

  it("edits a complete delegated portfolio through its exact update contract only", async () => {
    const client = api();
    const fullPortfolio = {
      id: "portfolio-1", institution_id: "institution-1", portfolio_code: "PF-01", authenticity_declared: true, consent_captured: true,
      custodian: "Secretariat", last_updated_on: "2026-09-10", legal_hold_active: false, notes: "Inițial", owner_name: "Profesor", owner_personnel_id: "personnel-1", owner_role: "Profesor", owner_user_id: "user-1", retention_period_days: 365, retention_until: "2027-09-10", school_year: "2026-2027", section_count: 3, status: "active", transfer_status: "none",
    };
    client.recordDetail.mockResolvedValue(fullPortfolio);
    client.updateRecord.mockResolvedValue({ ...fullPortfolio, notes: "Actualizat" });
    view([grant("portfolio", "portfolio-1", "education.portfolios.manage")], client);
    await screen.findByText("Portofoliu portfolio-1");
    fireEvent.click(screen.getByRole("button", { name: "Detalii" }));
    fireEvent.click(await screen.findByRole("button", { name: "Editează" }));
    fireEvent.change(screen.getByLabelText("Note *"), { target: { value: "Actualizat" } });
    fireEvent.click(screen.getByRole("button", { name: "Salvează" }));
    await waitFor(() => expect(client.updateRecord).toHaveBeenCalledWith("portfolios", "portfolio-1", expect.objectContaining({ notes: "Actualizat", owner_name: "Profesor", owner_personnel_id: "personnel-1" })));
    expect(client.governanceMeetingDetail).not.toHaveBeenCalled();
  });

  it("fails closed before display when the exact response has another institution", async () => {
    const client = api();
    client.recordDetail.mockResolvedValue({ id: "portfolio-1", institution_id: "other-institution", title: "Nu afișa" });
    view([grant("portfolio", "portfolio-1")], client, undefined, "institution-1");
    await screen.findByText("Nu există resurse delegate active pentru utilizatorul curent.");
  });
});
