import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { PrimeReactProvider } from "@primereact/core";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RoleCockpit } from "./RoleCockpits";

function view(node: ReactNode) {
  return render(<PrimeReactProvider>{node}</PrimeReactProvider>);
}

describe("RoleCockpit", () => {
  afterEach(cleanup);

  it("renders only server-owned secretariat metrics", async () => {
    const load = vi.fn().mockResolvedValue({ kind: "secretariat", value: { classes: 4, students: 92, active_enrolments: 88, portfolios_in_review: 3, institution_id: "school-a" } });
    view(<RoleCockpit kind="secretariat" allowed load={load} />);
    await screen.findByText("92");
    expect(load).toHaveBeenCalledOnce();
    expect(screen.getByText("Portofolii în verificare")).toBeInTheDocument();
  });

  it("does not request data without the exact cockpit capability", () => {
    const load = vi.fn();
    view(<RoleCockpit kind="hr" allowed={false} load={load} />);
    expect(screen.getByText(/education\.cockpit\.hr\.read/)).toBeInTheDocument();
    expect(load).not.toHaveBeenCalled();
  });

  it("renders a recoverable loading failure", async () => {
    view(<RoleCockpit kind="committee" allowed load={async () => { throw new Error("offline"); }} />);
    await waitFor(() => expect(screen.getByText(/nu au putut fi încărcați/i)).toBeInTheDocument());
  });

  it("renders an empty server response without synthesizing metrics", async () => {
    view(<RoleCockpit kind="inspector" allowed load={async () => ({ kind: "inspector", value: { readiness_open: 0, pending_publications: 0, mandatory_publication_pending: 0, requirements_pending: 0, evaluations_in_review: 0, institution_id: "school-a" } })} />);
    await screen.findByText(/Nu există înregistrări/i);
    expect(screen.getByText("Cerințe neacoperite")).toBeInTheDocument();
  });
});
