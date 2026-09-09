import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PortfolioArchiveGrantManager } from "./PortfolioArchiveGrantManager";
import type { EducationApi } from "./types";

describe("PortfolioArchiveGrantManager", () => {
  it("loads tenant-scoped choices and safely revokes an active grant", async () => {
    const api = {
      eligibleAttachmentDocuments: vi.fn().mockResolvedValue({ items: [{ id: "11111111-1111-4111-8111-111111111111", title: "CV profesor", current_version_no: 2 }], total: 1, page: 1, pageSize: 50 }),
      eligibleAttachmentUsers: vi.fn().mockResolvedValue({ items: [{ id: "22222222-2222-4222-8222-222222222222", name: "Ana Pop" }], total: 1, page: 1, pageSize: 100 }),
      attachmentGrants: vi.fn().mockResolvedValue({ items: [{ id: "33333333-3333-4333-8333-333333333333", archive_document_id: "11111111-1111-4111-8111-111111111111", document_title: "CV profesor", grantee_user_id: "22222222-2222-4222-8222-222222222222", grantee_name: "Ana Pop" }], total: 1, page: 1, pageSize: 50 }),
      createAttachmentGrant: vi.fn(),
      deleteAttachmentGrant: vi.fn().mockResolvedValue(undefined),
    } as unknown as EducationApi;

    render(<PrimeReactProvider><PortfolioArchiveGrantManager api={api} /></PrimeReactProvider>);
    expect(await screen.findByText("CV profesor")).toBeInTheDocument();
    expect(screen.getByText("Ana Pop")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Revocă" }));
    await waitFor(() => expect(api.deleteAttachmentGrant).toHaveBeenCalledWith("33333333-3333-4333-8333-333333333333"));
    expect(await screen.findByText("Dreptul de atașare a fost revocat.")).toBeInTheDocument();
  });
});
