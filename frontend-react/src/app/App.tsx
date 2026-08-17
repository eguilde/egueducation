import {
  lazy,
  Suspense,
  useMemo,
  type ComponentType,
  type ReactNode,
} from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "../auth/AuthProvider";
import { AppShell } from "../components/AppShell";
import { AppThemeProvider } from "../components/ThemeMenu";
import { createAdminApi } from "../features/admin/api";
import { createArchiveApi } from "../features/earchiva/api";
import { createEducationApi } from "../features/education/api";
import { createProfileApi } from "../features/profile/api";
import {
  browserPasskeyCeremony,
  supportsWebAuthn,
} from "../features/profile/webauthn";
import { createRegistraturaApi } from "../features/registratura/api";
import { createWorkflowApi } from "../features/workflow/api";
import {
  CallbackPage,
  LandingPage,
  LogoutCallbackPage,
  RegistrationPage,
  RequireAuthenticated,
  RequirePermission,
} from "./Pages";

const AdministrationWorkspace = lazy(() =>
  import("../features/admin/AdministrationWorkspace").then((module) => ({
    default: module.AdministrationWorkspace,
  })),
);
const ArchiveWorkspace = lazy(() =>
  import("../features/earchiva/ArchiveWorkspace").then((module) => ({
    default: module.ArchiveWorkspace,
  })),
);
const EducationWorkspace = lazy(() =>
  import("../features/education/EducationWorkspace").then((module) => ({
    default: module.EducationWorkspace,
  })),
);
const ProfileWorkspace = lazy(() =>
  import("../features/profile/ProfileWorkspace").then((module) => ({
    default: module.ProfileWorkspace,
  })),
);
const RegistraturaWorkspace = lazy(() =>
  import("../features/registratura/RegistraturaWorkspace").then((module) => ({
    default: module.RegistraturaWorkspace,
  })),
);
const WorkflowWorkspace = lazy(() =>
  import("../features/workflow/WorkflowWorkspace").then((module) => ({
    default: module.WorkflowWorkspace,
  })),
);
const CaMeetingWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.CaMeetingWizard,
  })),
);
const MeetingMinuteWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.MeetingMinuteWizard,
  })),
);
const MeetingVoteWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.MeetingVoteWizard,
  })),
);
const MeetingResolutionWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.MeetingResolutionWizard,
  })),
);
const ManagerialDossierWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.ManagerialDossierWizard,
  })),
);
const PersonnelRecordWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.PersonnelRecordWizard,
  })),
);
const EvaluationWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.EvaluationWizard,
  })),
);
const DeclarationWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.DeclarationWizard,
  })),
);
const MobilityWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.MobilityWizard,
  })),
);
const MeritWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.MeritWizard,
  })),
);
const PortfolioRecordWizard = lazy(() =>
  import("../features/education/wizards").then((module) => ({
    default: module.PortfolioRecordWizard,
  })),
);

const deferred = (children: ReactNode) => (
  <Suspense
    fallback={
      <div className="p-6" role="status">
        Se încarcă…
      </div>
    }
  >
    {children}
  </Suspense>
);

const secure = (permission: string, children: ReactNode, module?: string) => (
  <RequirePermission permission={permission} module={module}>
    {children}
  </RequirePermission>
);

const schoolReadPermissions = [
  "education.read",
  "education.governance.read",
  "education.decisions.read",
  "education.managerial.read",
  "education.regulations.read",
  "education.personnel.read",
  "education.evaluations.read",
  "education.declarations.read",
  "education.mobility.read",
  "education.gradatii.read",
  "education.portfolios.read",
  "education.compliance.read",
] as const;

function SchoolAccess({
  permissions,
  children,
}: {
  permissions: readonly string[];
  children: ReactNode;
}) {
  const { has, ready, session } = useAuth();
  if (!ready)
    return (
      <div className="p-6" role="status">
        Se încarcă…
      </div>
    );
  const educationActive = Boolean(
    session?.modules.some(
      (module) => module.code === "education" && module.active,
    ),
  );
  return educationActive && permissions.some(has) ? (
    <>{children}</>
  ) : (
    <Navigate to="/" replace />
  );
}

function SchoolRoute({
  permissions = schoolReadPermissions,
}: {
  permissions?: readonly string[];
}) {
  return (
    <SchoolAccess permissions={permissions}>
      {deferred(<EducationWorkspace />)}
    </SchoolAccess>
  );
}

type EducationWizardComponent = ComponentType<{
  adapter: {
    create(
      path: string,
      payload: Record<string, string | number | boolean>,
    ): Promise<unknown>;
    eligibleGovernanceUsers(): Promise<Array<{ id: string; name: string }>>;
  };
  canManage: boolean;
}>;

function SchoolWizardRoute({
  permission,
  Wizard,
}: {
  permission: string;
  Wizard: EducationWizardComponent;
}) {
  const { apiFetch } = useAuth();
  const api = useMemo(() => createEducationApi(apiFetch), [apiFetch]);
  const adapter = useMemo(
    () => ({
      create: (
        path: string,
        payload: Record<string, string | number | boolean>,
      ) => api.saveRelated(path, payload),
      eligibleGovernanceUsers: () => api.eligibleGovernanceUsers(),
    }),
    [api],
  );
  return (
    <SchoolAccess permissions={[permission]}>
      {deferred(<Wizard adapter={adapter} canManage />)}
    </SchoolAccess>
  );
}

function RegistraturaRoute() {
  const { apiFetch, has, session } = useAuth();
  const api = useMemo(() => createRegistraturaApi(apiFetch), [apiFetch]);
  const tenantKey = `${window.location.host}:${session?.institution_id ?? "none"}`;
  return secure(
    "registratura.read",
    <RegistraturaWorkspace
      api={api}
      tenantKey={tenantKey}
      canManage={has("registratura.manage")}
      canManageWorkflow={has("registratura.manage") || has("workflow.manage")}
      canReadAdminUsers={has("admin.users.read")}
      canReadLinks={has("registratura.links.read")}
      canManageLinks={has("registratura.links.manage")}
    />,
    "registratura",
  );
}

function WorkflowRoute() {
  const { apiFetch, has } = useAuth();
  const api = useMemo(() => createWorkflowApi(apiFetch), [apiFetch]);
  return secure(
    "workflow.read",
    <WorkflowWorkspace
      api={api}
      canTransition={has("workflow.transition")}
      canManage={has("workflow.manage")}
    />,
    "workflow",
  );
}

function ArchiveRoute() {
  const { apiFetch, has } = useAuth();
  const api = useMemo(() => createArchiveApi(apiFetch), [apiFetch]);
  // Archivists are granted the dedicated earchiva permission.  Do not
  // additionally require a role-derived module toggle in the client.
  return secure(
    "earchiva.read",
    <ArchiveWorkspace
      api={api}
      canManage={has("earchiva.manage")}
      canReadContent={has("earchiva.content.read")}
      canReview={has("earchiva.review")}
    />,
  );
}

function ProfileRoute() {
  const { apiFetch, updateLocalProfile, user } = useAuth();
  const api = useMemo(() => createProfileApi(apiFetch), [apiFetch]);
  return (
    <RequireAuthenticated>
      {user && (
        <ProfileWorkspace
          user={user}
          api={api}
          ceremony={supportsWebAuthn() ? browserPasskeyCeremony : undefined}
          onUpdated={updateLocalProfile}
        />
      )}
    </RequireAuthenticated>
  );
}

function AdministrationRoute() {
  const { apiFetch, has, session } = useAuth();
  const api = useMemo(() => createAdminApi(apiFetch), [apiFetch]);
  return secure(
    "admin.read",
    <AdministrationWorkspace
      api={api}
      institutionName={session?.institution_name ?? "Instituția curentă"}
      permissions={{
        dashboard: has("admin.read"),
        usersRead: has("admin.users.read"),
        usersManage: has("admin.users.manage"),
        rolesRead: has("admin.roles.read"),
        rolesManage: has("admin.roles.manage"),
        modulesRead: has("admin.modules.read"),
        modulesManage: has("admin.modules.manage"),
      }}
      canAccess={has}
    />,
  );
}

export function App() {
  return (
    <AppThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/auth/callback" element={<CallbackPage />} />
            <Route path="/auth/logout" element={<LogoutCallbackPage />} />
            <Route path="/auth/register" element={<RegistrationPage />} />
            <Route
              path="/documente"
              element={<Navigate to="/registratura" replace />}
            />
            <Route
              path="/flux"
              element={<Navigate to="/flux-documente" replace />}
            />
            <Route
              path="/profile"
              element={<Navigate to="/profil" replace />}
            />
            <Route
              path="/admin"
              element={<Navigate to="/administrare" replace />}
            />
            <Route element={<AppShell />}>
              <Route index element={<LandingPage />} />
              <Route path="profil" element={deferred(<ProfileRoute />)} />
              <Route
                path="registratura"
                element={deferred(<RegistraturaRoute />)}
              />
              <Route
                path="flux-documente"
                element={deferred(<WorkflowRoute />)}
              />
              <Route path="earchiva" element={deferred(<ArchiveRoute />)} />
              <Route path="scoala" element={<SchoolRoute />} />
              <Route path="scoala/dashboard" element={<SchoolRoute />} />
              <Route
                path="scoala/dashboard/director"
                element={<SchoolRoute />}
              />
              <Route
                path="scoala/dashboard/director/reports"
                element={<SchoolRoute />}
              />
              <Route path="scoala/reports" element={<SchoolRoute />} />
              <Route
                path="scoala/secretariat"
                element={<SchoolRoute permissions={["education.read"]} />}
              />
              <Route
                path="scoala/compliance"
                element={
                  <SchoolRoute permissions={["education.compliance.read"]} />
                }
              />
              <Route
                path="scoala/teacher"
                element={
                  <SchoolRoute permissions={["education.portfolios.read"]} />
                }
              />
              <Route
                path="scoala/governance"
                element={
                  <SchoolRoute
                    permissions={[
                      "education.governance.read",
                      "education.decisions.read",
                      "education.managerial.read",
                      "education.regulations.read",
                    ]}
                  />
                }
              />
              <Route
                path="scoala/governance/ca-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={CaMeetingWizard}
                  />
                }
              />
              <Route
                path="scoala/governance/minutes-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={MeetingMinuteWizard}
                  />
                }
              />
              <Route
                path="scoala/governance/votes-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={MeetingVoteWizard}
                  />
                }
              />
              <Route
                path="scoala/governance/resolutions-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={MeetingResolutionWizard}
                  />
                }
              />
              <Route
                path="scoala/governance/managerial-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.managerial.manage"
                    Wizard={ManagerialDossierWizard}
                  />
                }
              />
              <Route
                path="scoala/personnel"
                element={
                  <SchoolRoute
                    permissions={[
                      "education.personnel.read",
                      "education.evaluations.read",
                      "education.declarations.read",
                      "education.mobility.read",
                      "education.gradatii.read",
                    ]}
                  />
                }
              />
              <Route
                path="scoala/personnel/wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.personnel.manage"
                    Wizard={PersonnelRecordWizard}
                  />
                }
              />
              <Route
                path="scoala/personnel/evaluations-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.evaluations.manage"
                    Wizard={EvaluationWizard}
                  />
                }
              />
              <Route
                path="scoala/personnel/declarations-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.declarations.manage"
                    Wizard={DeclarationWizard}
                  />
                }
              />
              <Route
                path="scoala/personnel/mobility-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.mobility.manage"
                    Wizard={MobilityWizard}
                  />
                }
              />
              <Route
                path="scoala/personnel/merit-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.gradatii.manage"
                    Wizard={MeritWizard}
                  />
                }
              />
              <Route
                path="scoala/portfolio"
                element={
                  <SchoolRoute permissions={["education.portfolios.read"]} />
                }
              />
              <Route
                path="scoala/portfolio/me"
                element={
                  <SchoolRoute permissions={["education.portfolios.read"]} />
                }
              />
              <Route
                path="scoala/portfolio/workflow"
                element={
                  <SchoolRoute permissions={["education.portfolios.read"]} />
                }
              />
              <Route
                path="scoala/portfolio/wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.portfolios.manage"
                    Wizard={PortfolioRecordWizard}
                  />
                }
              />
              <Route
                path="education/governance/ca-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={CaMeetingWizard}
                  />
                }
              />
              <Route
                path="education/governance/minutes-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={MeetingMinuteWizard}
                  />
                }
              />
              <Route
                path="education/governance/votes-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={MeetingVoteWizard}
                  />
                }
              />
              <Route
                path="education/governance/resolutions-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.governance.manage"
                    Wizard={MeetingResolutionWizard}
                  />
                }
              />
              <Route
                path="education/governance/managerial-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.managerial.manage"
                    Wizard={ManagerialDossierWizard}
                  />
                }
              />
              <Route
                path="education/personnel/wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.personnel.manage"
                    Wizard={PersonnelRecordWizard}
                  />
                }
              />
              <Route
                path="education/personnel/evaluations-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.evaluations.manage"
                    Wizard={EvaluationWizard}
                  />
                }
              />
              <Route
                path="education/personnel/declarations-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.declarations.manage"
                    Wizard={DeclarationWizard}
                  />
                }
              />
              <Route
                path="education/personnel/mobility-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.mobility.manage"
                    Wizard={MobilityWizard}
                  />
                }
              />
              <Route
                path="education/personnel/merit-wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.gradatii.manage"
                    Wizard={MeritWizard}
                  />
                }
              />
              <Route
                path="education/portfolio/wizard"
                element={
                  <SchoolWizardRoute
                    permission="education.portfolios.manage"
                    Wizard={PortfolioRecordWizard}
                  />
                }
              />
              <Route path="education/*" element={<SchoolRoute />} />
              <Route
                path="administrare"
                element={deferred(<AdministrationRoute />)}
              />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </AppThemeProvider>
  );
}
