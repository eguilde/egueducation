import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { EducationDelegationManager, type EducationDelegationApi } from "./EducationDelegationManager";

const row = {
  id: "00000000-0000-0000-0000-000000000001",
  tenant_code: "tenant-test",
  institution_id: "school-test",
  delegator_user_id: "00000000-0000-0000-0000-000000000002",
  delegate_user_id: "00000000-0000-0000-0000-000000000003",
  delegator_name: "Director",
  delegate_name: "Director adjunct",
  permission_code: "education.portfolios.transfer",
  resource_type: "institution" as const,
  resource_id: "",
  status: "offered" as const,
  valid_from: "2026-09-10",
  valid_until: "",
  offered_by_user_id: "00000000-0000-0000-0000-000000000002",
  offered_at: "2026-09-10T10:00:00Z",
  accepted_by_user_id: "",
  accepted_at: "",
  revoked_by_user_id: "",
  revoked_at: "",
  expired_by_user_id: "",
  expired_at: "",
  notes: "",
};

function api(): EducationDelegationApi {
  return {
    list: vi.fn().mockResolvedValue({ items: [row], total: 24, page: 1, pageSize: 20 }),
    eligibleAdjuncts: vi.fn().mockResolvedValue([{ user_id: row.delegate_user_id, name: row.delegate_name, email: "adjunct@example.test" }]),
    eligiblePermissions: vi.fn().mockResolvedValue([row.permission_code]),
    create: vi.fn().mockResolvedValue(row),
    accept: vi.fn().mockResolvedValue({ ...row, status: "accepted" }),
    revoke: vi.fn().mockResolvedValue({ ...row, status: "revoked" }),
    expire: vi.fn().mockResolvedValue({ ...row, status: "expired" }),
  };
}

const allCapabilities = { read: true, offer: true, accept: true, revoke: true, expire: true };
function view(client: EducationDelegationApi, capabilities = allCapabilities, onChanged?: () => void) {
  return render(<PrimeReactProvider><EducationDelegationManager api={client} capabilities={capabilities} onChanged={onChanged} /></PrimeReactProvider>);
}

describe("EducationDelegationManager", () => {
  it("loads delegation rows with server pagination and advances pages", async () => {
    const client = api(); view(client);
    await screen.findByText(row.permission_code);
    expect(client.list).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 20 }));
    fireEvent.click(screen.getByRole("button", { name: "Următor" }));
    await waitFor(() => expect(client.list).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, pageSize: 20 })));
  });

  it("uses only server-authorized offer options and keeps submit disabled until selection", async () => {
    const client = api(); view(client);
    const add = await screen.findByRole("button", { name: "Oferă delegare" });
    await waitFor(() => expect(add).toBeEnabled());
    fireEvent.click(add);
    expect(screen.getByRole("button", { name: "Oferă delegarea" })).toBeDisabled();
    expect(client.eligibleAdjuncts).toHaveBeenCalledTimes(1);
    expect(client.eligiblePermissions).toHaveBeenCalledTimes(1);
  });

  it("gates lifecycle actions and does not fetch without read access", async () => {
    const client = api();
    view(client, { read: false, offer: false, accept: false, revoke: false, expire: false });
    expect(await screen.findByText("Nu aveți drept de consultare a delegărilor.")).toBeInTheDocument();
    expect(client.list).not.toHaveBeenCalled();
    expect(client.eligibleAdjuncts).not.toHaveBeenCalled();
  });

  it("runs acceptance only for an authorized offered delegation", async () => {
    const client = api();
    const onChanged = vi.fn();
    view(client, allCapabilities, onChanged);
    fireEvent.click(await screen.findByRole("button", { name: `Acceptă ${row.permission_code}` }));
    await waitFor(() => expect(client.accept).toHaveBeenCalledWith(row.id));
    await waitFor(() => expect(onChanged).toHaveBeenCalledOnce());
  });

  it("refreshes the evaluated authorization snapshot after revocation", async () => {
    const client = api();
    const onChanged = vi.fn();
    view(client, allCapabilities, onChanged);
    fireEvent.click(await screen.findByRole("button", { name: `Revocă ${row.permission_code}` }));
    await waitFor(() => expect(client.revoke).toHaveBeenCalledWith(row.id));
    await waitFor(() => expect(onChanged).toHaveBeenCalledOnce());
  });

  it("filters the displayed delegate column by name on the server", async () => {
    const client = api(); view(client);
    fireEvent.change(await screen.findByLabelText("Filtru Delegat"), { target: { value: "adjunct" } });
    await waitFor(() => expect(client.list).toHaveBeenLastCalledWith(expect.objectContaining({
      page: 1,
      filters: expect.objectContaining({ delegate_name: "adjunct" }),
    })));
  });
});
