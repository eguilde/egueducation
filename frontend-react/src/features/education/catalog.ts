import type { EducationArea, EducationModule } from "./types";

/**
 * Navigation is intentionally permission-driven. It mirrors the backend route
 * families recorded in the Education OpenAPI catalog, not an optimistic UI menu.
 */
export const educationAreas: EducationArea[] = [
  { id: "overview", label: "Panou de control", icon: "pi pi-chart-bar", permissions: ["education.read", "education.governance.read", "education.managerial.read", "education.portfolios.read", "education.evaluations.read", "education.personnel.read", "education.compliance.read"], description: "Indicatori și situația operațională a instituției." },
  { id: "governance", label: "Guvernanță", icon: "pi pi-users", permissions: ["education.governance.read"], description: "Ședințe, organisme, membri, voturi și hotărâri." },
  { id: "decisions", label: "Decizii", icon: "pi pi-file-edit", permissions: ["education.decisions.read"], description: "Decizii, emitere și pași de publicare." },
  { id: "managerial", label: "Management", icon: "pi pi-briefcase", permissions: ["education.managerial.read"], description: "Dosare manageriale și fluxuri aferente." },
  { id: "regulations", label: "Regulamente", icon: "pi pi-book", permissions: ["education.regulations.read"], description: "Regulamente, versiuni și aprobare." },
  { id: "committees", label: "Comisii", icon: "pi pi-sitemap", permissions: ["education.committees.read"], description: "Comisii și membri." },
  { id: "personnel", label: "Personal", icon: "pi pi-id-card", permissions: ["education.personnel.read", "education.personnel.files.read", "education.personnel.access.read"], description: "Dosare, funcții, acces și situații disciplinare." },
  { id: "evaluations", label: "Evaluări", icon: "pi pi-check-square", permissions: ["education.evaluations.read"], description: "Evaluări, criterii, contestații și rezultate." },
  { id: "declarations", label: "Declarații", icon: "pi pi-verified", permissions: ["education.declarations.read"], description: "Declarații de interese și conformitate." },
  { id: "mobility", label: "Mobilitate", icon: "pi pi-arrow-right-arrow-left", permissions: ["education.mobility.read"], description: "Mobilitate, punctaje, contestații și decizii." },
  { id: "merit", label: "Gradații de merit", icon: "pi pi-trophy", permissions: ["education.gradatii.read"], description: "Dosare, criterii, punctaje și contestații." },
  { id: "portfolios", label: "Portofolii", icon: "pi pi-folder", permissions: ["education.portfolios.read"], description: "Portofolii, opis, custodie și transfer." },
  { id: "compliance", label: "Conformitate", icon: "pi pi-shield", permissions: ["education.compliance.read"], description: "Publicări și cerințe de conformitate." },
];

export function visibleEducationAreas(
  permissions: readonly string[],
  modules: readonly EducationModule[],
): EducationArea[] {
  const educationModule = modules.find((module) => module.code === "education");
  if (educationModule && !educationModule.active) return [];
  return educationAreas.filter((area) =>
    area.permissions.some((permission) => permissions.includes(permission)),
  );
}
