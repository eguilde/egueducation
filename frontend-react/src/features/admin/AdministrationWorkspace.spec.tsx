import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimeReactProvider } from "@primereact/core/config";
import { primeTheme } from "../../components/ThemeMenu";
import { AdministrationWorkspace } from "./AdministrationWorkspace";
import type { AdminApi } from "./types";

const api = (): AdminApi => ({
  dashboard: vi.fn().mockResolvedValue({ stats: {}, modules: [], admin_sections: [], warnings: [] }),
  users: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }),
  saveUser: vi.fn(), roles: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }),
  modules: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }),
	 saveModule: vi.fn(), resource: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 50 }), saveResource: vi.fn(),
});

describe("AdministrationWorkspace resource RBAC", () => {
  it("does not expose or fetch a resource merely because generic admin.read is granted", async () => {
    const transport = api();
    render(<PrimeReactProvider {...primeTheme}><AdministrationWorkspace api={transport} institutionName="Școala" permissions={{ dashboard: true }} canAccess={(permission) => permission === "admin.read"} /></PrimeReactProvider>);

    await waitFor(() => expect(transport.dashboard).toHaveBeenCalledOnce());
    expect(screen.queryByRole("button", { name: "Apartenențe" })).not.toBeInTheDocument();
    expect(transport.resource).not.toHaveBeenCalled();
  });

  it("loads only the resource whose exact backend permission is effective", async () => {
    const transport = api();
    render(<PrimeReactProvider {...primeTheme}><AdministrationWorkspace api={transport} institutionName="Școala" permissions={{ dashboard: true }} canAccess={(permission) => permission === "admin.audit.read"} /></PrimeReactProvider>);

    await waitFor(() => expect(transport.resource).toHaveBeenCalledWith("audit"));
    expect(screen.getByRole("button", { name: "Jurnal audit" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Apartenențe" })).not.toBeInTheDocument();
  });

	it("does not expose a mutation control when only the resource read permission is effective", async () => {
		const transport = api();
		render(<PrimeReactProvider {...primeTheme}><AdministrationWorkspace api={transport} institutionName="Școala" permissions={{ dashboard: true }} canAccess={(permission) => permission === "admin.audit.read" || permission === "gdpr.policies.read"} /></PrimeReactProvider>);

		await waitFor(() => expect(transport.resource).toHaveBeenCalledWith("audit"));
		expect(screen.queryByRole("button", { name: "Adaugă sau actualizează" })).not.toBeInTheDocument();
		expect(transport.saveResource).not.toHaveBeenCalled();
	});
});
