import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core/config";
import { describe, expect, it, vi } from "vitest";
import { PersonnelRecordWizard, CaMeetingWizard } from "./index";

const adapter = () => ({ create: vi.fn().mockResolvedValue({ id: "1" }) });
const renderWizard = (node: React.ReactNode) =>
  render(<PrimeReactProvider>{node}</PrimeReactProvider>);

describe("education wizard behavior", () => {
  it("does not call adapter when RBAC denies creation", async () => {
    const api = adapter();
    renderWizard(<PersonnelRecordWizard adapter={api} />);
    fireEvent.click(screen.getByRole("button", { name: "Continuă" }));
    expect(api.create).not.toHaveBeenCalled();
    expect(screen.getByText(/accesul necesită/i)).toBeInTheDocument();
  });
  it("validates current step before advancing", () => {
    const api = adapter();
    renderWizard(<PersonnelRecordWizard adapter={api} canManage />);
    fireEvent.click(screen.getByRole("button", { name: "Continuă" }));
    expect(
      screen.getByText(/nume complet este obligatoriu/i),
    ).toBeInTheDocument();
  });
  it("submits exact personnel payload after step completion", async () => {
    const api = adapter();
    renderWizard(<PersonnelRecordWizard adapter={api} canManage />);
    fireEvent.change(screen.getByLabelText("Nume complet"), {
      target: { value: "Ana Pop" },
    });
    fireEvent.change(screen.getByLabelText("Funcție"), {
      target: { value: "Profesor" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Continuă" }));
    fireEvent.click(screen.getByRole("button", { name: "Continuă" }));
    fireEvent.click(screen.getByRole("button", { name: "Continuă" }));
    fireEvent.click(screen.getByRole("button", { name: "Salvează" }));
    await waitFor(() => expect(api.create).toHaveBeenCalled());
    expect(api.create.mock.calls[0][0]).toBe("/education/personnel/records");
    expect(api.create.mock.calls[0][1]).toMatchObject({
      full_name: "Ana Pop",
      school_year: expect.any(String),
    });
  });
  it("requires immutable CA user IDs", () => {
    const api = adapter();
    renderWizard(<CaMeetingWizard adapter={api} canManage />);
    expect(screen.getByText(/ședință ca/i)).toBeInTheDocument();
  });
});
