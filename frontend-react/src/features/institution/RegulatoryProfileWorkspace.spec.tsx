import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PrimeReactProvider } from "@primereact/core/config";
import { primeTheme } from "../../components/ThemeMenu";
import { RegulatoryProfileWorkspace } from "./RegulatoryProfileWorkspace";
import type { InstitutionCapabilities, InstitutionPolicyApi, PolicyCutoverPreflight, RegulatoryProfile } from "./api";

vi.mock("../../auth/AuthProvider", () => ({ useAuth: () => ({ session: null }) }));

const profile: RegulatoryProfile = {
  id: "profile-1", tenant_code: "tenant-a", institution_id: "inst-a", version: 1, status: "unclassified",
  school_legal_form: null, regulatory_profile: "", authorization_status: "unknown", accreditation_reference: "", authorized_levels: [],
  has_legal_personality: true, tax_identifier: "", founder_name: "", funder_name: "", budget_authority_name: "",
  is_contracting_authority: false, accounting_profile: "", procurement_profile: "", payroll_profile: "", vat_profile: "not_registered",
  treasury_required: false, public_funding: false, program_codes: [], effective_from: null, effective_to: null, source_reference: "",
  approved_by_subject: "", approved_at: null, created_by_subject: "migration", created_at: "2026-09-11T00:00:00Z",
  updated_by_subject: "migration", updated_at: "2026-09-11T00:00:00Z",
};
const capabilities: InstitutionCapabilities = {
  tenant_code: "tenant-a", institution_id: "inst-a", evaluated_at: "2026-09-11T00:00:00Z", evaluation_id: "eval-1",
  profile_id: "profile-1", profile_version: 1, profile_status: "unclassified", school_legal_form: null,
  blocked: true, block_reason: "Profilul instituțional nu este clasificat și aprobat", warnings: ["Citirea rămâne disponibilă"],
  effective_policies: [], capabilities: [],
};
const api = (): InstitutionPolicyApi => ({ profile: vi.fn().mockResolvedValue(profile), capabilities: vi.fn().mockResolvedValue(capabilities), saveProfile: vi.fn() });

const preflight: PolicyCutoverPreflight = {
  tenant_code: "tenant-a", institution_id: "inst-a", phase: "legacy",
  legacy_profiles: 2, unmapped_profiles: 0, legacy_assignments: 3, unmapped_assignments: 0,
  legacy_overrides: 1, unmapped_overrides: 0, legacy_evaluations: 4, unmapped_evaluations: 0,
  missing_pack_provenance: 0, inputs_without_effective_date: 0, inputs_with_multiple_decisions: 0,
  consumer_provenance_mismatches: 0, open_blocking_issues: 0, structurally_ready_for_dual: true,
};

describe("RegulatoryProfileWorkspace", () => {
  it("keeps reads available but explains fail-closed regulated writes for an unclassified school", async () => {
    render(<PrimeReactProvider {...primeTheme}><RegulatoryProfileWorkspace api={api()} canManage={false} /></PrimeReactProvider>);
    expect(await screen.findByText(/nu este clasificat și aprobat/i)).toBeInTheDocument();
    expect(screen.getByText("Neclasificată")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /versiune nouă/i })).not.toBeInTheDocument();
  });

  it("does not preselect a legal form when opening the first classified version", async () => {
    render(<PrimeReactProvider {...primeTheme}><RegulatoryProfileWorkspace api={api()} canManage /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: /versiune nouă/i }));
    await waitFor(() => expect(screen.getByText("Selectați forma juridică")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Salvează versiunea" })).toBeDisabled();
  });

  it("keeps the selected legal form and derived regulatory profile in the same form update", async () => {
    const client = api();
    render(<PrimeReactProvider {...primeTheme}><RegulatoryProfileWorkspace api={client} canManage /></PrimeReactProvider>);
    fireEvent.click(await screen.findByRole("button", { name: /versiune nouă/i }));
    fireEvent.click(screen.getByRole("combobox", { name: "Formă juridică" }));
    fireEvent.click(await screen.findByRole("option", { name: "Școală publică" }));
    fireEvent.change(screen.getByLabelText("Citare act"), { target: { value: "Hotărâre E2E" } });
    const save = screen.getByRole("button", { name: "Salvează versiunea" });
    expect(save).toBeEnabled();
    fireEvent.click(save);
    await waitFor(() => expect(client.saveProfile).toHaveBeenCalledWith(expect.objectContaining({ school_legal_form: "public", regulatory_profile: "ro.public.preuniversity", source: expect.objectContaining({ citation: "Hotărâre E2E" }) })));
  });

  it("shows the scope-bound cutover preflight only to a profile administrator", async () => {
    const client = { ...api(), cutoverPreflight: vi.fn().mockResolvedValue(preflight) };
    render(<PrimeReactProvider {...primeTheme}><RegulatoryProfileWorkspace api={client} canManage /></PrimeReactProvider>);
    expect(await screen.findByText("Pregătire migrare policy v2")).toBeInTheDocument();
    expect(await screen.findByText("Structură reconciliată")).toBeInTheDocument();
    expect(screen.getByText("0/2")).toBeInTheDocument();
    expect(client.cutoverPreflight).toHaveBeenCalledTimes(1);
  });

  it("ignores an older institutional response that completes after a new load", async () => {
    let resolveOldProfile!: (value: RegulatoryProfile) => void;
    let resolveOldCapabilities!: (value: InstitutionCapabilities) => void;
    const oldApi: InstitutionPolicyApi = {
      profile: vi.fn(() => new Promise<RegulatoryProfile>((resolve) => { resolveOldProfile = resolve; })),
      capabilities: vi.fn(() => new Promise<InstitutionCapabilities>((resolve) => { resolveOldCapabilities = resolve; })),
      saveProfile: vi.fn(),
    };
    const nextProfile = { ...profile, id: "profile-new", tenant_code: "tenant-new", institution_id: "inst-new", school_legal_form: "private" as const, regulatory_profile: "ro.private.preuniversity" };
    const nextCapabilities = { ...capabilities, profile_id: nextProfile.id, tenant_code: nextProfile.tenant_code, institution_id: nextProfile.institution_id, school_legal_form: "private" as const };
    const nextApi: InstitutionPolicyApi = { profile: vi.fn().mockResolvedValue(nextProfile), capabilities: vi.fn().mockResolvedValue(nextCapabilities), saveProfile: vi.fn() };
    const view = render(<PrimeReactProvider {...primeTheme}><RegulatoryProfileWorkspace api={oldApi} canManage={false} /></PrimeReactProvider>);

    view.rerender(<PrimeReactProvider {...primeTheme}><RegulatoryProfileWorkspace api={nextApi} canManage={false} /></PrimeReactProvider>);
    expect(await screen.findByText("Școală privată")).toBeInTheDocument();

    resolveOldProfile({ ...profile, school_legal_form: "public", regulatory_profile: "ro.public.preuniversity" });
    resolveOldCapabilities({ ...capabilities, school_legal_form: "public" });
    await Promise.resolve();
    expect(screen.getByText("Școală privată")).toBeInTheDocument();
    expect(screen.queryByText("Școală publică")).not.toBeInTheDocument();
  });
});
