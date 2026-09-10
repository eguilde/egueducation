import { describe, expect, it } from "vitest";
import { visibleEducationAreas } from "./catalog";

describe("Education navigation catalog", () => {
  it("shows only domains permitted to the user", () => {
    expect(visibleEducationAreas(["education.governance.read", "education.portfolios.read"], [{ code: "education", active: true }]).map((area) => area.id)).toEqual(["overview", "governance", "committees", "portfolios"]);
  });

  it("hides all domains when the education module is explicitly disabled", () => {
    expect(visibleEducationAreas(["education.governance.read"], [{ code: "education", active: false }])).toEqual([]);
  });

  it("keeps the portfolio area discoverable for a teacher with own-only access", () => {
    expect(visibleEducationAreas(["education.portfolios.read_own"], [{ code: "education", active: true }]).map((area) => area.id)).toEqual(["overview", "portfolios"]);
  });

  it("shows institution portfolios to a director with school-scoped read access", () => {
    expect(visibleEducationAreas(["education.portfolios.school.read"], [{ code: "education", active: true }]).map((area) => area.id)).toEqual(["overview", "portfolios"]);
  });
});
