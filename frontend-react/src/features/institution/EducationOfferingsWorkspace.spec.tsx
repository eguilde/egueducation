import { fireEvent, render, screen } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { primeTheme } from "../../components/ThemeMenu";
import { EducationOfferingsWorkspace, loadAllCatalogOptions } from "./EducationOfferingsWorkspace";
import type { EducationOffering, InstitutionPolicyApi, OfferingAuthorization, SchoolLocation } from "./api";

const location: SchoolLocation = { id: "location-1", code: "sediu", name: "Sediu", address: "Balotești", active: true, effective_from: "2026-09-11", effective_to: null, expected_version: 1 };
const offering: EducationOffering = { id: "offering-1", code: "gimnazial", education_level: "gimnazial", specialization_code: "", language_code: "ro", title: "Gimnaziu", active: true, effective_from: "2026-09-11", effective_to: null, expected_version: 1 };
const authorization: OfferingAuthorization = { id: "authorization-1", offering_id: offering.id, offering_code: offering.code, offering_title: offering.title, location_id: location.id, location_code: location.code, location_name: location.name, status: "accredited", authority_name: "ARACIP", decision_reference: "H-1", capacity: 120, capacity_unit: "students", shift: "day", effective_from: "2026-09-11", effective_to: null, expected_version: 1, source_citation: "Hotărârea H-1", source_url: "https://example.test/h-1", replaces_authorization_id: null };

function api(): InstitutionPolicyApi {
  return {
    profile: vi.fn(), capabilities: vi.fn(), saveProfile: vi.fn(),
    locations: vi.fn().mockResolvedValue({ items: [location], total: 1, page: 1, pageSize: 10 }),
    offerings: vi.fn().mockResolvedValue({ items: [offering], total: 1, page: 1, pageSize: 10 }),
    authorizations: vi.fn().mockResolvedValue({ items: [authorization], total: 1, page: 1, pageSize: 10 }),
    createLocation: vi.fn(), updateLocation: vi.fn(), createOffering: vi.fn(), updateOffering: vi.fn(), createAuthorization: vi.fn(),
  };
}

describe("EducationOfferingsWorkspace", () => {
  it("loads every server page for authorization selectors", async () => {
    const all = Array.from({ length: 51 }, (_, index) => ({ ...location, id: `location-${index + 1}`, code: `sediu-${index + 1}` }));
    const loader = vi.fn(async ({ page, pageSize }: { page: number; pageSize: number }) => ({ items: all.slice((page - 1) * pageSize, page * pageSize), total: all.length, page, pageSize }));
    await expect(loadAllCatalogOptions(loader)).resolves.toHaveLength(51);
    expect(loader).toHaveBeenNthCalledWith(1, { page: 1, pageSize: 50, direction: "asc", filters: {} });
    expect(loader).toHaveBeenNthCalledWith(2, { page: 2, pageSize: 50, direction: "asc", filters: {} });
  });

  it("renders the server-backed authorization matrix and opens the create dialog for a manager", async () => {
    const client = api();
    render(<PrimeReactProvider {...primeTheme}><EducationOfferingsWorkspace api={client} canManage /></PrimeReactProvider>);
    expect(await screen.findByText("H-1")).toBeInTheDocument();
    expect(screen.getByText("accredited")).toBeInTheDocument();
    expect(client.locations).toHaveBeenCalledWith(expect.objectContaining({ page: 1, pageSize: 10 }));
    expect(client.offerings).toHaveBeenCalled();
    expect(client.authorizations).toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Adaugă decizie" }));
    expect(await screen.findByRole("dialog", { name: "Decizie de autorizare" })).toBeInTheDocument();
  });

  it("does not expose mutation actions to a read-only user", async () => {
    render(<PrimeReactProvider {...primeTheme}><EducationOfferingsWorkspace api={api()} canManage={false} /></PrimeReactProvider>);
    expect(await screen.findByText("H-1")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Adaugă decizie" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Înlocuiește decizia/ })).not.toBeInTheDocument();
  });

  it("opens optimistic-concurrency editors from the offering and location action columns", async () => {
    render(<PrimeReactProvider {...primeTheme}><EducationOfferingsWorkspace api={api()} canManage /></PrimeReactProvider>);
    expect(await screen.findByText("H-1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Oferte" }));
    fireEvent.click(await screen.findByRole("button", { name: "Editează oferta Gimnaziu" }));
    expect(await screen.findByRole("dialog", { name: "Editează oferta gimnazial" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Închide" }));
    fireEvent.click(screen.getByRole("tab", { name: "Locații" }));
    fireEvent.click(await screen.findByRole("button", { name: "Editează locația Sediu" }));
    expect(await screen.findByRole("dialog", { name: "Editează locația sediu" })).toBeInTheDocument();
  });
});
