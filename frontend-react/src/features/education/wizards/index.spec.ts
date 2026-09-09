import { describe, expect, it } from "vitest";
import { buildWizardPayload, wizardDefinitions } from "./index";

describe("education wizard contracts", () => {
  it.each([
    ["caMeeting", "/education/governance/meetings", "school_year"],
    ["personnel", "/education/personnel/records", "full_name"],
    ["evaluation", "/education/evaluations/records", "employee_code"],
    ["declaration", "/education/declarations/records", "declaration_type"],
    ["mobility", "/education/mobility/records", "request_type"],
    ["merit", "/education/gradatii/records", "committee_name"],
    ["portfolio", "/education/portfolios/records", "owner_name"],
  ])("maps %s to the Angular endpoint and fields", (key, path, field) => {
    const definition = wizardDefinitions[key];
    expect(definition.path).toBe(path);
    expect(definition.fields.some(item => item.key === field)).toBe(true);
    expect(definition.steps).toHaveLength(4);
  });
  it("trims strings and preserves typed values", () => expect(buildWizardPayload({ name: " Ana ", score: 8, funded: false })).toEqual({ name: "Ana", score: 8, funded: false }));
  it("declares manage versus self-manage RBAC", () => {
    expect(wizardDefinitions.caMeeting.permission).toBe("manage");
    expect(wizardDefinitions.portfolio.permission).toBe("manage");
    expect(wizardDefinitions.personnel.validate({ full_name: "", role_title: "", school_year: "" })).toHaveLength(3);
  });
  it("uses the exact governance enums accepted by the backend", () => {
    const field = (definition: "minute" | "vote" | "resolution", key: string) =>
      wizardDefinitions[definition].fields.find((item) => item.key === key);

    expect(wizardDefinitions.minute.initial.follow_up_status).toBe("de_stabilit");
    expect(field("minute", "follow_up_status")?.options?.map((item) => item.value)).toEqual([
      "de_stabilit", "in_urmarire", "realizat", "amanat", "inchis",
    ]);
    expect(wizardDefinitions.vote.initial).toMatchObject({
      decision_type: "hotarare",
      outcome: "adoptat",
    });
    expect(field("vote", "decision_type")?.options?.map((item) => item.value)).toEqual([
      "hotarare", "aviz", "informare", "delegare", "aprobare",
    ]);
    expect(field("vote", "outcome")?.options?.map((item) => item.value)).toEqual([
      "adoptat", "respins", "amanat",
    ]);
    expect(wizardDefinitions.resolution.initial).toMatchObject({
      publication_status: "intern",
      anonymization_state: "nu_este_necesara",
    });
    expect(field("resolution", "publication_status")?.options?.map((item) => item.value)).toEqual([
      "intern", "publicat", "pregatit_publicare",
    ]);
    expect(field("resolution", "anonymization_state")?.options?.map((item) => item.value)).toEqual([
      "necesara", "finalizata", "nu_este_necesara",
    ]);
  });
});

