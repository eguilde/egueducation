import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { EducationListPanel, educationPermissionAllows } from "./EducationWorkspace";

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
    fireEvent.change(screen.getByLabelText("Filtru Titlu"), {
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
});

describe("educationPermissionAllows", () => {
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
  });
});
