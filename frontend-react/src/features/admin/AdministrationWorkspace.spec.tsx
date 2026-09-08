import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimeReactProvider } from "@primereact/core/config";
import { primeTheme } from "../../components/ThemeMenu";
import { AdministrationWorkspace, normalizeUserForm, validateUserForm } from "./AdministrationWorkspace";
import type { AdminApi, UpsertUserInput } from "./types";

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

describe("AdministrationWorkspace user identity form", () => {
  const phoneOnlyUser: UpsertUserInput = {
    name: "  Utilizator SMS  ", email: "", phone: " +40712345678 ", locale: "ro", status: "active",
    email_verified: false, phone_verified: true, preferred_otp_channel: "sms",
  };

  it("accepts and sends an SMS-primary phone-only user without inventing an e-mail address", () => {
    expect(validateUserForm(phoneOnlyUser)).toBeUndefined();
    expect(normalizeUserForm(phoneOnlyUser)).toEqual({
      name: "Utilizator SMS",
      email: "",
      phone: "+40712345678",
      locale: "ro",
      status: "active",
      email_verified: false,
      phone_verified: true,
      preferred_otp_channel: "sms",
    });
  });

  it("requires an identifier and makes the SMS channel require a phone number", () => {
    expect(validateUserForm({ ...phoneOnlyUser, phone: "" })).toBe("Introduceți e-mailul sau numărul de telefon al utilizatorului.");
    expect(validateUserForm({ ...phoneOnlyUser, email: "ana@example.test", phone: "" })).toBe("Pentru autentificare prin SMS este necesar un număr de telefon.");
    expect(validateUserForm({ ...phoneOnlyUser, phone_verified: false })).toBe("Numărul de telefon trebuie marcat ca verificat pentru autentificare prin SMS.");
  });

  it("preserves the immutable user id when an existing user is updated", () => {
    expect(normalizeUserForm({ ...phoneOnlyUser, id: "1a2b3c4d" }).id).toBe("1a2b3c4d");
  });
});
