import { describe, expect, it } from "vitest";
import { createInputForDomain, updateInputForDomain } from "./EducationWorkspace";

describe("EducationWorkspace root DTO mappers", () => {
  it("maps required personnel create fields and drops generated/output-only fields", () => {
    const dto = createInputForDomain("personnel", {
      full_name: "Ana Pop",
      employment_type: "titular",
      evaluation_status: "draft",
      mobility_stage: "none",
      role_title: "Profesor",
      school_year: "2026-2027",
      status: "active",
      employee_code: "GENERATED-ONLY",
      id: "output-only",
    });

    expect(dto).toMatchObject({ full_name: "Ana Pop", employment_type: "titular", school_year: "2026-2027" });
    expect(dto).not.toHaveProperty("id");
    expect(dto).not.toHaveProperty("employee_code");
  });

  it("rejects personnel lifecycle values outside the database contract", () => {
    expect(() => createInputForDomain("personnel", {
      full_name: "Ana Pop",
      employment_type: "permanent",
      evaluation_status: "current",
      mobility_stage: "none",
      role_title: "Profesor",
      school_year: "2026-2027",
      status: "active",
    })).toThrow("education_invalid_employment_type");
  });

  it("maps only canonical publication taxonomies and enforces the published date", () => {
    expect(createInputForDomain("compliance", {
      anonymization_status: "nu_este_necesara",
      domain: "conformitate",
      entity_label: "Anunț",
      entity_type: "anunt",
      publication_channel: "site_public",
      publication_status: "pregatit",
      publication_code: "SERVER-ONLY",
    })).toEqual({
      anonymization_status: "nu_este_necesara",
      domain: "conformitate",
      entity_label: "Anunț",
      entity_type: "anunt",
      publication_channel: "site_public",
      publication_status: "pregatit",
    });
    expect(() => createInputForDomain("compliance", {
      anonymization_status: "nu_este_necesara",
      domain: "education",
      entity_label: "Anunț",
      entity_type: "anunt",
      publication_channel: "site_public",
      publication_status: "pregatit",
    })).toThrow("education_invalid_domain");
    expect(() => createInputForDomain("compliance", {
      anonymization_status: "finalizata",
      domain: "conformitate",
      entity_label: "Anunț",
      entity_type: "anunt",
      publication_channel: "site_public",
      publication_status: "publicat",
    })).toThrow("education_required_published_on");
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
