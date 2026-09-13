import { PrimeReactProvider } from "@primereact/core";
import { act, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AdmissionApi, AdmissionLegalPreparationArtifact } from "./api";
import { LegalPreparationArtifactUpload } from "./AdmissionVertical";

const queued = {
  intent_id: "11111111-1111-4111-8111-111111111111",
  preparation_id: "22222222-2222-4222-8222-222222222222",
  artifact_slot: "primary",
  retention_until: "2026-10-12T10:15:00Z",
  replayed: false,
  document: { id: "doc-1", institution_id: "institution-1", title: "Decizie", original_file_name: "signed.pdf", mime_type: "application/pdf", source_kind: "admission", source_system: "admission", external_reference: "prep-1", status: "queued", current_version_no: 1, received_at: "2026-09-12T10:00:00Z", created_at: "2026-09-12T10:00:00Z", updated_at: "2026-09-12T10:00:00Z" },
  version: { id: "version-1", document_id: "doc-1", version_no: 1, source_sha256: "a".repeat(64), source_size_bytes: 12, page_count: 1, text_status: "queued", created_at: "2026-09-12T10:00:00Z" },
} as AdmissionLegalPreparationArtifact;

const ready = { ...queued, document: { ...queued.document, status: "ready" } } as AdmissionLegalPreparationArtifact;

afterEach(() => vi.useRealTimers());

describe("LegalPreparationArtifactUpload polling", () => {
  it("caps automatic polling, resumes only on manual check, and aborts outstanding requests on unmount", async () => {
    vi.useFakeTimers();
    const getLegalPreparationArtifact = vi.fn().mockResolvedValue(queued);
    const api = { getLegalPreparationArtifact, uploadLegalPreparationArtifact: vi.fn() } as unknown as AdmissionApi;
    const onReady = vi.fn();
    const view = render(<PrimeReactProvider><LegalPreparationArtifactUpload api={api} preparationID="prep-1" slot="primary" label="PDF semnat · test" onReady={onReady}/></PrimeReactProvider>);

    await act(async () => { await Promise.resolve(); await Promise.resolve(); });
    await act(async () => { vi.advanceTimersByTime(800); await Promise.resolve(); });
    for (let attempt = 0; attempt < 20; attempt += 1) {
      await act(async () => { vi.advanceTimersByTime(1_500); await Promise.resolve(); });
    }

    expect(screen.getByText(/Procesarea durează prea mult/)).toBeInTheDocument();
    // One recovery GET plus twenty bounded polling attempts.
    expect(getLegalPreparationArtifact).toHaveBeenCalledTimes(21);
    await act(async () => { vi.advanceTimersByTime(30_000); await Promise.resolve(); });
    expect(getLegalPreparationArtifact).toHaveBeenCalledTimes(21);

    getLegalPreparationArtifact.mockResolvedValueOnce(ready);
    await act(async () => { screen.getByText("Verifică starea").click(); await Promise.resolve(); await Promise.resolve(); });
    expect(getLegalPreparationArtifact).toHaveBeenCalledTimes(22);
    expect(screen.getByText("Versiunea WORM este pregătită pentru finalizare.")).toBeInTheDocument();
    expect(onReady).toHaveBeenLastCalledWith(expect.objectContaining({ document_id: "doc-1", version_id: "version-1" }));

    view.unmount();
    const signals = getLegalPreparationArtifact.mock.calls.map((call) => call[2]).filter(Boolean) as AbortSignal[];
    expect(signals.length).toBeGreaterThan(0);
    expect(signals.every((signal) => signal.aborted)).toBe(true);
  });
});
