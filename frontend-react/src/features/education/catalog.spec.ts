import { describe, expect, it } from "vitest";
import { visibleEducationAreas } from "./catalog";

describe("Education navigation catalog", () => {
  it("shows only domains permitted to the user", () => {
    expect(visibleEducationAreas(["education.governance.read", "education.portfolios.read"], [{ code: "education", active: true }]).map((area) => area.id)).toEqual(["overview", "governance", "portfolios"]);
  });

  it("hides all domains when the education module is explicitly disabled", () => {
    expect(visibleEducationAreas(["education.governance.read"], [{ code: "education", active: false }])).toEqual([]);
  });
});
