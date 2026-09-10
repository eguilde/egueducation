import { describe, expect, it } from "vitest";
import { createContractClient } from "../../api/client";
import { createRoleCockpitsApi, roleCockpitLoader } from "./role-cockpits-api";

describe("role cockpit contract adapter", () => {
  it("uses the four generated literal cockpit endpoints", async () => {
    const calls: string[] = [];
    const client = createContractClient(async (request) => {
      const path = new URL(request.url).pathname;
      calls.push(path);
      const body = path.endsWith("/secretariat/cockpit")
        ? { classes: 1, students: 2, active_enrolments: 2, portfolios_in_review: 0, institution_id: "school-a" }
        : path.endsWith("/hr/cockpit")
          ? { personnel: 4, expiring_documents: 0, expired_documents: 0, pending_evaluations: 1, institution_id: "school-a" }
          : path.endsWith("/committee/cockpit")
            ? { committees: 1, active_members: 2, meetings: 3, evidence_documents: 4, institution_id: "school-a" }
            : { readiness_open: 0, pending_publications: 1, mandatory_publication_pending: 0, requirements_pending: 2, evaluations_in_review: 3, institution_id: "school-a" };
      return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
    }, "https://example.test/api");
    const api = createRoleCockpitsApi(client);
    await api.secretariat();
    await api.hr();
    await api.committee();
    await roleCockpitLoader(api, "inspector")();
    expect(calls).toEqual([
      "/api/education/secretariat/cockpit",
      "/api/education/hr/cockpit",
      "/api/education/committee/cockpit",
      "/api/education/inspector/cockpit",
    ]);
  });
});
