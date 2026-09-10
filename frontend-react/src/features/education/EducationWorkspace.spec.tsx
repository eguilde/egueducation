import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { EducationListPanel, RecordFormDialog, SchoolRowActionMenu, domainListCapabilities, domainWizardRoutes, educationPermissionAllows, effectiveEducationPermissions, hasDomainRelations, relationManagePermission } from "./EducationWorkspace";

describe("School overlay lifecycle", () => {
  it("closes the action popover before invoking an action that may mount a dialog", async () => {
    const onSelect = vi.fn();
    render(
      <PrimeReactProvider>
        <SchoolRowActionMenu actions={[{ label: "Detalii", icon: "pi pi-eye", onSelect }]} />
      </PrimeReactProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Acțiuni înregistrare" }));
    fireEvent.click(await screen.findByRole("button", { name: "Detalii" }));
    expect(onSelect).not.toHaveBeenCalled();
    await waitFor(() => expect(onSelect).toHaveBeenCalledOnce());
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("keeps a remote Select disabled until its server options are ready", () => {
    const common = {
      open: { input: { app_user_id: "" } },
      title: "Adaugă membru",
      onClose: vi.fn(),
      onChange: vi.fn(),
      onSave: vi.fn(),
    };
    const { rerender } = render(
      <PrimeReactProvider>
        <RecordFormDialog
          {...common}
          fields={[{ key: "app_user_id", label: "Utilizator", kind: "select", options: [], search: { label: "Caută Utilizator", value: "Ana", onChange: vi.fn(), loading: true } }]}
        />
      </PrimeReactProvider>,
    );
    expect(screen.getByRole("combobox", { name: "Utilizator" })).toBeDisabled();

    rerender(
      <PrimeReactProvider>
        <RecordFormDialog
          {...common}
          fields={[{ key: "app_user_id", label: "Utilizator", kind: "select", options: [{ label: "Ana Pop", value: "user-1" }], search: { label: "Caută Utilizator", value: "Ana", onChange: vi.fn(), loading: false } }]}
        />
      </PrimeReactProvider>,
    );
    expect(screen.getByRole("combobox", { name: "Utilizator" })).toBeEnabled();
  });

  it("enforces a conditional contract requirement before submitting", () => {
    const fields = [
      { key: "document_status", label: "Stare", kind: "select" as const, options: [{ label: "Aprobat", value: "approved" }] },
      { key: "approved_on", label: "Aprobat la", kind: "date" as const, required: (input: Record<string, unknown>) => input.document_status === "approved" },
    ];
    const common = { title: "Document", fields, onClose: vi.fn(), onChange: vi.fn(), onSave: vi.fn() };
    const { rerender } = render(
      <PrimeReactProvider>
        <RecordFormDialog {...common} open={{ input: { document_status: "approved", approved_on: "" } }} />
      </PrimeReactProvider>,
    );
    expect(screen.getByLabelText("Aprobat la")).toBeRequired();
    expect(screen.getByRole("button", { name: "Salvează" })).toBeDisabled();

    rerender(
      <PrimeReactProvider>
        <RecordFormDialog {...common} open={{ input: { document_status: "approved", approved_on: "2026-09-10" } }} />
      </PrimeReactProvider>,
    );
    expect(screen.getByRole("button", { name: "Salvează" })).toBeEnabled();
  });
});

describe("EducationListPanel", () => {
  it("debounces header-row filters and sends them to the server from page one", async () => {
    const load = vi.fn().mockResolvedValue({
      items: [{ id: "record-1", title: "Plan anual" }],
      total: 1,
      page: 1,
      pageSize: 20,
    });
    render(
      <PrimeReactProvider>
        <EducationListPanel<{ id: string; title: string }>
          title="Registre"
          description="Test"
          load={load}
          emptyMessage="Gol"
          columns={[
            { field: "title", header: "Titlu", render: (item) => item.title },
            { header: "Acțiuni", action: true, render: () => "" },
          ]}
        />
      </PrimeReactProvider>,
    );

    await waitFor(() => expect(load).toHaveBeenCalledTimes(1));
    fireEvent.change(await screen.findByLabelText("Filtru Titlu"), {
      target: { value: "plan" },
    });
    await waitFor(() =>
      expect(load).toHaveBeenLastCalledWith(
        "",
        1,
        20,
        {},
        { title: "plan" },
      ),
    );
  });

  it("places the accessible Add control in the frozen Action header", async () => {
    const onAdd = vi.fn();
    render(
      <PrimeReactProvider>
        <EducationListPanel<{ id: string; title: string }>
          title="Registre"
          description="Test"
          load={vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 })}
          emptyMessage="Gol"
          onAdd={onAdd}
          addLabel="înregistrare"
          columns={[
            { field: "title", header: "Titlu", render: (item) => item.title },
            { header: "Acțiuni", action: true, render: () => "" },
          ]}
        />
      </PrimeReactProvider>,
    );

    const add = await screen.findByRole("button", { name: "Adaugă înregistrare" });
    expect(add.closest("th")).toHaveTextContent("Acțiuni");
    fireEvent.click(add);
    expect(onAdd).toHaveBeenCalledOnce();
  });

  it("provides a server-bound filter and sort control for every declared data column", async () => {
    const load = vi.fn().mockResolvedValue({
      items: [{ id: "record-1", title: "Plan anual", status: "draft" }],
      total: 1,
      page: 1,
      pageSize: 20,
    });
    render(
      <PrimeReactProvider>
        <EducationListPanel<{ id: string; title: string; status: string }>
          title="Registre"
          description="Test"
          load={load}
          emptyMessage="Gol"
          columns={[
            { field: "title", header: "Titlu", render: (item) => item.title },
            { field: "status", header: "Stare", render: (item) => item.status },
            { header: "Acțiuni", action: true, render: () => "" },
          ]}
        />
      </PrimeReactProvider>,
    );

    expect(await screen.findByLabelText("Filtru Titlu")).toBeInTheDocument();
    expect(screen.getByLabelText("Filtru Stare")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Stare/ }));
    await waitFor(() =>
      expect(load).toHaveBeenLastCalledWith("", 1, 20, { field: "status", direction: "asc" }, {}),
    );
  });

  it("does not render or emit unsupported Committee filters and sorting", async () => {
    const load = vi.fn().mockResolvedValue({
      items: [{
        id: "committee-1",
        committee_code: "COM-001",
        school_year: "2026-2027",
        committee_type: "CEAC",
        title: "Comisia CEAC",
        status: "active",
        decision_reference: "CA/1",
        starts_on: "2026-09-01",
        ends_on: "2027-08-31",
        evaluation_scope: "Da",
        notes: "Intern",
      }],
      total: 1,
      page: 1,
      pageSize: 20,
    });
    render(
      <PrimeReactProvider>
        <EducationListPanel<{ id: string; [key: string]: string }>
          title="Comisii"
          description="Test"
          load={load}
          emptyMessage="Gol"
          filterableFields={domainListCapabilities.committees.filterableFields}
          sortableFields={domainListCapabilities.committees.sortableFields}
          columns={[
            { field: "committee_code", header: "Cod", render: (item) => item.committee_code },
            { field: "school_year", header: "An școlar", render: (item) => item.school_year },
            { field: "committee_type", header: "Tip", render: (item) => item.committee_type },
            { field: "title", header: "Denumire", render: (item) => item.title },
            { field: "status", header: "Stare", render: (item) => item.status },
            { field: "decision_reference", header: "Act", render: (item) => item.decision_reference },
            { field: "starts_on", header: "Începe la", render: (item) => item.starts_on },
            { field: "ends_on", header: "Se încheie la", render: (item) => item.ends_on },
            { field: "evaluation_scope", header: "Evaluare", render: (item) => item.evaluation_scope },
            { field: "notes", header: "Note", render: (item) => item.notes },
          ]}
        />
      </PrimeReactProvider>,
    );

    await waitFor(() => expect(load).toHaveBeenCalledTimes(1));
    expect(screen.getByLabelText("Filtru An școlar")).toBeInTheDocument();
    expect(screen.queryByLabelText("Filtru Act")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Filtru Se încheie la")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Filtru Evaluare")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Filtru Note")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Cod/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Începe la/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Act/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Se încheie la/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Evaluare/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Note/ })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Începe la/ }));
    await waitFor(() =>
      expect(load).toHaveBeenLastCalledWith("", 1, 20, { field: "starts_on", direction: "asc" }, {}),
    );
  });
});

describe("School create routes", () => {
  it("routes portfolio creation through the canonical owner selector wizard", () => {
    expect(domainWizardRoutes.portfolios).toBe("/scoala/portfolio/wizard");
  });
});

describe("School detail relations", () => {
  it("does not mount the generic relation workspace for a portfolio with dedicated relations", () => {
    expect(hasDomainRelations("portfolios")).toBe(false);
    expect(hasDomainRelations("managerial")).toBe(true);
  });
});

describe("domainListCapabilities", () => {
  it("keeps an explicit entry for every root registry and the backend Committee allowlist", () => {
    expect(Object.keys(domainListCapabilities)).toHaveLength(11);
    expect(domainListCapabilities.committees.filterableFields).toEqual([
      "school_year", "committee_type", "title", "status",
    ]);
    expect(domainListCapabilities.committees.sortableFields).toEqual([
      "school_year", "committee_type", "title", "status", "committee_code", "starts_on",
    ]);
  });
});

describe("educationPermissionAllows", () => {
  it("projects manage to read only inside the same education permission family", () => {
    expect(effectiveEducationPermissions(["education.governance.manage"])).toContain(
      "education.governance.read",
    );
    expect(effectiveEducationPermissions(["education.governance.manage"])).not.toContain(
      "education.personnel.read",
    );
    expect(effectiveEducationPermissions(["admin.users.manage"])).not.toContain(
      "education.governance.read",
    );
  });

  it("makes accepted request-time delegations usable without broadening resource scope", () => {
    const grants = [{
      permission_code: "education.portfolios.school.manage",
      resource_type: "portfolio",
      resource_id: "portfolio-1",
    }] as const;

    expect(educationPermissionAllows([], grants, "education.portfolios.school.manage", "portfolio", "portfolio-1")).toBe(true);
    expect(educationPermissionAllows([], grants, "education.portfolios.school.manage", "portfolio", "portfolio-2")).toBe(false);
    expect(educationPermissionAllows([], grants, "education.portfolios.school.manage")).toBe(false);
    expect(educationPermissionAllows(["education.portfolios.school.manage"], [], "education.portfolios.school.manage", "portfolio", "portfolio-2")).toBe(true);
    expect(educationPermissionAllows(["education.governance.manage"], [], "education.governance.read")).toBe(true);
    expect(educationPermissionAllows([], [{ permission_code: "education.governance.manage", resource_type: "institution", resource_id: "inst-1" }], "education.governance.read")).toBe(true);
  });

  it.each([
    ["decisions", "education.decisions.issuance.manage"],
    ["decisions", "education.compliance.manage"],
    ["committees", "education.governance.manage"],
    ["personnel", "education.personnel.files.manage"],
    ["personnel", "education.personnel.access.manage"],
    ["portfolios", "education.portfolios.transfer"],
  ] as const)("uses the backend subresource permission %s → %s", (domain, permission) => {
    expect(relationManagePermission(domain, permission)).toBe(permission);
  });

  it("keeps the parent domain permission only for relations governed by that domain", () => {
    expect(relationManagePermission("evaluations")).toBe("education.evaluations.manage");
    expect(relationManagePermission("merit")).toBe("education.gradatii.manage");
  });
});
