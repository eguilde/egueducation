import { describe, expect, it } from "vitest";
import { navigation } from "./navigation";

describe("school role cockpit navigation", () => {
  it("exposes each cockpit only with its precise capability", () => {
    expect(navigation).toEqual(expect.arrayContaining([
      expect.objectContaining({ to: "/scoala/secretariat", permission: "education.cockpit.secretariat.read" }),
      expect.objectContaining({ to: "/scoala/hr", permission: "education.cockpit.hr.read" }),
      expect.objectContaining({ to: "/scoala/committee-cockpit", permission: "education.cockpit.committee.read" }),
      expect.objectContaining({ to: "/scoala/inspector", permission: "education.cockpit.inspector.read" }),
    ]));
  });

  it("exposes reports for every backend report read permission and signatures for their exact capabilities", () => {
    expect(navigation).toEqual(expect.arrayContaining([
      expect.objectContaining({
        to: "/scoala/reports",
        permissions: [
          "education.portfolios.school.read",
          "education.portfolios.read",
          "education.evaluations.read",
          "education.governance.read",
          "education.personnel.files.read",
          "education.compliance.read",
        ],
      }),
      expect.objectContaining({
        to: "/scoala/signatures",
        permissions: [
          "education.signatures.read",
          "education.signatures.manage",
          "education.signatures.validate",
        ],
      }),
      expect.objectContaining({
        to: "/scoala/portfolios",
        permissions: [
          "education.portfolios.school.read",
          "education.portfolios.read",
        ],
      }),
    ]));
  });
});
