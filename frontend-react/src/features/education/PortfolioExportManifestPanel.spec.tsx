import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioExportManifestPanel } from "./PortfolioExportManifestPanel";
import type { EducationApi } from "./types";

const result = { export_manifest_id: "m-1", generated_at: "2026-09-09T12:00:00Z", manifest: { manifest_version: "egueducation.portfolio-export-manifest/v1", hash_algorithm: "SHA-256", manifest_sha256: "abc123", tenant_code: "tenant", institution_id: "school", portfolio: { id: "p-1", portfolio_code: "PORT-2026", school_year: "2026-2027", status: "validated" }, documents: [{ evidence_record_id: "e-1", section_code: "identificare_profesionala", component_code: "structura_cadru", document_title: "CV", chronological_no: 1, issued_on: "2026-09-01", evidence_type: "document", archive_document_id: "d-1", archive_version_id: "v-1", archive_version_no: 2, source_bucket: "archive", source_object_key: "tenant/cv.pdf", source_sha256: "hash-source" }] } };
describe("PortfolioExportManifestPanel", () => {
  it("posts only the portfolio identity and renders the authoritative manifest", async () => { const api: Pick<EducationApi, "createPortfolioExportManifest"> = { createPortfolioExportManifest: vi.fn().mockResolvedValue(result) }; render(<PrimeReactProvider><PortfolioExportManifestPanel portfolioID="p-1" api={api} canExport /></PrimeReactProvider>); fireEvent.click(screen.getByRole("button", { name: "Generează manifest probatoriu" })); await waitFor(() => expect(api.createPortfolioExportManifest).toHaveBeenCalledWith("p-1")); expect(await screen.findByText("PORT-2026")).toBeInTheDocument(); expect(screen.getByText("hash-source")).toBeInTheDocument(); });
  it("does not render controls when export is not authorized by the parent permission gate", () => { const api: Pick<EducationApi, "createPortfolioExportManifest"> = { createPortfolioExportManifest: vi.fn() }; render(<PrimeReactProvider><PortfolioExportManifestPanel portfolioID="p-1" api={api} canExport={false} /></PrimeReactProvider>); expect(screen.queryByText("Export probatoriu")).not.toBeInTheDocument(); });
});
