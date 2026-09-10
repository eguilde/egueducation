import type { ContractClient } from "../../api/client";
import type { RoleCockpitData, RoleCockpitKind, RoleCockpitLoader } from "./RoleCockpits";

async function requireData<T>(request: Promise<{ data?: T; response: Response }>): Promise<T> {
  const result = await request;
  if (result.data !== undefined) return result.data;
  throw new Error(`education_role_cockpit_${result.response.status}`);
}

export type RoleCockpitsApi = {
  secretariat: () => Promise<Extract<RoleCockpitData, { kind: "secretariat" }>>;
  hr: () => Promise<Extract<RoleCockpitData, { kind: "hr" }>>;
  committee: () => Promise<Extract<RoleCockpitData, { kind: "committee" }>>;
  inspector: () => Promise<Extract<RoleCockpitData, { kind: "inspector" }>>;
};

export function createRoleCockpitsApi(client: ContractClient): RoleCockpitsApi {
  return {
    secretariat: async () => ({ kind: "secretariat", value: await requireData(client.GET("/api/education/secretariat/cockpit")) }),
    hr: async () => ({ kind: "hr", value: await requireData(client.GET("/api/education/hr/cockpit")) }),
    committee: async () => ({ kind: "committee", value: await requireData(client.GET("/api/education/committee/cockpit")) }),
    inspector: async () => ({ kind: "inspector", value: await requireData(client.GET("/api/education/inspector/cockpit")) }),
  };
}

export function roleCockpitLoader(api: RoleCockpitsApi, kind: RoleCockpitKind): RoleCockpitLoader {
  switch (kind) {
    case "secretariat": return api.secretariat;
    case "hr": return api.hr;
    case "committee": return api.committee;
    case "inspector": return api.inspector;
  }
}
