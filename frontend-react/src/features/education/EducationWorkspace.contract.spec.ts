import { describe, expect, it } from "vitest";
import { createInputForDomain, updateInputForDomain } from "./EducationWorkspace";

describe("EducationWorkspace root DTO mappers", () => {
  it("maps required personnel create fields and drops generated/output-only fields", () => {
    const dto = createInputForDomain("personnel", {
      full_name: "Ana Pop",
      employment_type: "permanent",
      evaluation_status: "current",
      mobility_stage: "none",
      role_title: "Profesor",
      school_year: "2026-2027",
      status: "active",
      employee_code: "GENERATED-ONLY",
      id: "output-only",
    });

    expect(dto).toMatchObject({ full_name: "Ana Pop", employment_type: "permanent", school_year: "2026-2027" });
    expect(dto).not.toHaveProperty("id");
    expect(dto).not.toHaveProperty("employee_code");
  });

  it("uses the portfolio PATCH allow-list and excludes lifecycle/retention output", () => {
    const dto = updateInputForDomain("portfolios", {
      owner_name: "Ana Pop",
      owner_role: "Profesor",
      school_year: "2026-2027",
      status: "draft",
      transfer_status: "none",
      last_updated_on: "2026-09-10",
      retention_until: "2099-01-01",
      lifecycle_status: "archived",
      portfolio_code: "GENERATED-ONLY",
      id: "output-only",
    });

    expect(dto).toEqual(expect.objectContaining({ owner_name: "Ana Pop", last_updated_on: "2026-09-10", transfer_status: "none" }));
    expect(dto).not.toHaveProperty("retention_until");
    expect(dto).not.toHaveProperty("lifecycle_status");
    expect(dto).not.toHaveProperty("portfolio_code");
    expect(dto).not.toHaveProperty("id");
  });
});
