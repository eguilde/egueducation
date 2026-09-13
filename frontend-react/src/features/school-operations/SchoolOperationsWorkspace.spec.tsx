import { PrimeReactProvider } from "@primereact/core";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SchoolOperationsWorkspace, validLifecycleTransitions } from "./SchoolOperationsWorkspace";

const api = {
  list: vi.fn().mockResolvedValue({ items: [{ id: "contract-1", contract_number: "C-1", title: "Electricitate", supplier_name: "Furnizor Energie", supplier_party_id: "supplier-1", category: "utilități", lifecycle_status: "active", starts_on: "2026-01-01", ends_on: "2026-12-31", total_value: 100, currency: "RON", archive_status: "pending", expected_version: 1 }], total: 1, page: 1, pageSize: 20 }),
  suppliers: vi.fn().mockResolvedValue({ items: [{ id: "supplier-1", display_name: "Furnizor Energie", code: "FE", tax_id: "RO1" }], total: 1, page: 1, pageSize: 20 }),
  detail: vi.fn(), create: vi.fn(), amend: vi.fn(), obligations: vi.fn(), addObligation: vi.fn(), transition: vi.fn(),
};

describe("SchoolOperationsWorkspace", () => {
  it("renders server-side table controls and hides creation without manage RBAC", async () => {
    render(<PrimeReactProvider><SchoolOperationsWorkspace api={api} capabilities={{ read: true, manage: false, approve: false }} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.getByText("Electricitate")).toBeInTheDocument());
    expect(screen.getByLabelText("Filtru Contract")).toBeInTheDocument();
    expect(screen.getByLabelText("Sortează după Nr. contract")).toBeInTheDocument();
    expect(screen.queryByLabelText("Adaugă contract")).not.toBeInTheDocument();
  });

  it("shows the action-header create command only to managers", async () => {
    render(<PrimeReactProvider><SchoolOperationsWorkspace api={api} capabilities={{ read: true, manage: true, approve: false }} /></PrimeReactProvider>);
    await waitFor(() => expect(screen.getByLabelText("Adaugă contract")).toBeInTheDocument());
  });

  it("offers only lifecycle transitions permitted by the local backend-aligned state machine", () => {
    expect(validLifecycleTransitions("draft", "pending")).toEqual(["verified"]);
    expect(validLifecycleTransitions("active", "pending")).toEqual(["suspended", "terminated"]);
    expect(validLifecycleTransitions("active", "pending", true)).toEqual(["suspended", "terminated", "expired"]);
    expect(validLifecycleTransitions("terminated", "pending")).toEqual([]);
    expect(validLifecycleTransitions("terminated", "archived")).toEqual(["archived"]);
  });
});
