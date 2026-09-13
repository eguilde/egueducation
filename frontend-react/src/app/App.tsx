import {
  lazy,
  Suspense,
  useMemo,
  type ComponentType,
  type ReactNode,
} from "react";
import { BrowserRouter, Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { AuthProvider, useAuth } from "../auth/AuthProvider";
import { AppShell } from "../components/AppShell";
import { AppThemeProvider } from "../components/ThemeMenu";
import { createAdminApi } from "../features/admin/api";
import { createArchiveApi } from "../features/earchiva/api";
import { createEducationApi } from "../features/education/api";
import { SignedArtifactEvidenceWorkspace, SchoolReportsWorkspace } from "../features/education/SchoolTrustWorkspaces";
import { createProfileApi } from "../features/profile/api";
import { createContractClient } from "../api/client";
import { createInstitutionPolicyApi } from "../features/institution/api";
import { createArchiveRetentionApi } from "../features/earchiva/archive-retention-api";
import { createRegulatorySourcesApi } from "../features/regulatory-sources/api";
import { InstitutionPolicyProvider } from "../features/institution/InstitutionPolicyProvider";
import { createSchoolClassesApi } from "../features/education/school-classes-api";
import { createSchoolOperationsApi } from "../features/school-operations/api";
import { createAdmissionApi } from "../features/admission/api";
import { createSchoolWizardApi } from "../features/education/school-wizard-api";
import type { WizardKind } from "../features/education/wizards";
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
const RegulatoryProfileWorkspace = lazy(() =>
  import("../features/institution/RegulatoryProfileWorkspace").then((module) => ({ default: module.RegulatoryProfileWorkspace })),
);
const DelegatedEducationResourcesWorkspace = lazy(() =>
  import("../features/education/DelegatedEducationResourcesWorkspace").then((module) => ({
    default: module.DelegatedEducationResourcesWorkspace,
  })),
);
const TeacherPortfolioWorkspace = lazy(() =>
  import("../features/education/TeacherPortfolioWorkspace").then((module) => ({
    default: module.TeacherPortfolioWorkspace,
  })),
);
const PortfolioReviewWorkspace = lazy(() =>
  import("../features/education/PortfolioReviewWorkspace").then((module) => ({ default: module.PortfolioReviewWorkspace })),
);
const SchoolClassesWorkspace = lazy(() =>
  import("../features/education/SchoolClassesWorkspace").then((module) => ({
    default: module.SchoolClassesWorkspace,
  })),
);
const SchoolOperationsWorkspace = lazy(() =>
  import("../features/school-operations/SchoolOperationsWorkspace").then((module) => ({
    default: module.SchoolOperationsWorkspace,
  })),
);
const AdmissionWorkspace = lazy(() =>
  import("../features/admission/AdmissionWorkspace").then((module) => ({
    default: module.AdmissionWorkspace,
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
  "education.portfolios.school.read",
  "education.portfolios.read_own",
  "education.compliance.read",
  "education.cockpit.secretariat.read",
  "education.cockpit.hr.read",
  "education.cockpit.committee.read",
  "education.cockpit.inspector.read",
] as const;

function SchoolAccess({
  permissions,
  children,
  allowDelegatedResources = false,
}: {
  permissions: readonly string[];
  children: ReactNode;
  allowDelegatedResources?: boolean;
}) {
  const { canEducation, ready, authorizationReady, session, educationGrants } = useAuth();
  if (!ready || !authorizationReady)
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
  const hasExactDelegatedResource = educationGrants.some((grant) => grant.resource_type !== "institution");
  return educationActive && (permissions.some((permission) => canEducation(permission)) || (allowDelegatedResources && hasExactDelegatedResource)) ? (
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

function DelegatedEducationResourcesRoute() {
  const { apiFetch, educationGrants, refreshEducationAuthorization, session } = useAuth();
  const api = useMemo(() => createEducationApi(apiFetch), [apiFetch]);
  return <SchoolAccess permissions={[]} allowDelegatedResources>{deferred(<DelegatedEducationResourcesWorkspace grants={educationGrants} api={api} institutionID={session?.institution_id} onRefresh={refreshEducationAuthorization} />)}</SchoolAccess>;
}

function TeacherPortfolioRoute() {
  const { apiFetch, canEducation } = useAuth();
  const api = useMemo(() => createEducationApi(apiFetch), [apiFetch]);
  return (
    <SchoolAccess permissions={["education.portfolios.read_own"]}>
      {deferred(
        <TeacherPortfolioWorkspace
          api={api}
          canManageOwn={canEducation("education.portfolios.manage_own")}
          canExportOwn={canEducation("education.portfolios.read_own") || canEducation("education.portfolios.export_own")}
        />,
      )}
    </SchoolAccess>
  );
}

function PortfolioReviewRoute() {
  const { apiFetch, canEducation } = useAuth();
  const api = useMemo(() => createEducationApi(apiFetch), [apiFetch]);
  const canReturn = canEducation("education.portfolios.request_corrections") || canEducation("education.portfolios.school.manage") || canEducation("education.portfolios.manage");
  const canManage = canEducation("education.portfolios.school.manage") || canEducation("education.portfolios.manage");
  return <SchoolAccess permissions={["education.portfolios.school.read", "education.portfolios.read", "education.portfolios.verify", "education.portfolios.request_corrections", "education.portfolios.school.manage", "education.portfolios.manage"]}>
    {deferred(<PortfolioReviewWorkspace api={api} canManage={canManage} canReturn={canReturn} canManageLifecycle={canEducation("education.portfolios.school.manage")} canDecideRetention={canEducation("education.portfolios.custody.manage") && canEducation("earchiva.manage")} />)}
  </SchoolAccess>;
}

function SchoolClassesRoute() {
  const { apiFetch, canEducation } = useAuth();
  const client = useMemo(() => createContractClient(apiFetch), [apiFetch]);
  const api = useMemo(() => createSchoolClassesApi(client), [client]);
  return <SchoolAccess permissions={["education.classes.read", "education.classes.manage", "education.classes.read_assigned"]}>{deferred(<SchoolClassesWorkspace api={api} capabilities={{ read: canEducation("education.classes.read") || canEducation("education.classes.manage"), assigned: canEducation("education.classes.read_assigned"), manage: canEducation("education.classes.manage") }} />)}</SchoolAccess>;
}

function SchoolOperationsRoute() {
  const { apiFetch, canEducation } = useAuth();
  const client = useMemo(() => createContractClient(apiFetch), [apiFetch]);
  const api = useMemo(() => createSchoolOperationsApi(client), [client]);
  const permissions = ["school_operations.contracts.read", "school_operations.contracts.manage", "school_operations.contracts.approve"] as const;
  return <SchoolAccess permissions={permissions}>{deferred(<SchoolOperationsWorkspace api={api} capabilities={{ read: canEducation("school_operations.contracts.read") || canEducation("school_operations.contracts.manage") || canEducation("school_operations.contracts.approve"), manage: canEducation("school_operations.contracts.manage"), approve: canEducation("school_operations.contracts.approve") }} />)}</SchoolAccess>;
}

function AdmissionRoute() {
  const { apiFetch, canEducation } = useAuth();
  const client = useMemo(() => createContractClient(apiFetch), [apiFetch]);
  const api = useMemo(() => createAdmissionApi(client), [client]);
  const permissions = ["education.admissions.read", "education.admissions.manage", "education.admissions.decide", "education.admissions.appeals.manage", "education.admissions.retention.manage", "education.admissions.retention.approve", "registratura.read", "education.classes.read", "institution.offerings.read"] as const;
  return <SchoolAccess permissions={permissions}>{deferred(<AdmissionWorkspace api={api} capabilities={{
    read: canEducation("education.admissions.read"),
    piiRead: canEducation("education.admissions.read") && canEducation("registratura.read"),
    manage: canEducation("education.admissions.manage"),
    contextManage: canEducation("education.admissions.manage") && canEducation("education.classes.read") && canEducation("institution.offerings.read"),
    retentionManage: canEducation("education.admissions.retention.manage"),
    retentionApprove: canEducation("education.admissions.retention.approve"),
    signerAuthorizationManage: canEducation("education.admissions.signer.manage"),
    signerAuthorizationApprove: canEducation("education.admissions.signer.approve"),
    decide: canEducation("education.admissions.decide"),
    appealsManage: canEducation("education.admissions.appeals.manage"),
  }} />)}</SchoolAccess>;
}

const schoolReportReadPermissions = [
  "education.portfolios.school.read",
  "education.portfolios.read",
  "education.evaluations.read",
  "education.governance.read",
  "education.personnel.files.read",
  "education.compliance.read",
] as const;

function SchoolReportsRoute() {
  const { apiFetch, canEducation } = useAuth();
  const client = useMemo(() => createContractClient(apiFetch), [apiFetch]);
  return (
    <SchoolAccess permissions={schoolReportReadPermissions}>
      {deferred(
        <SchoolReportsWorkspace
          client={client}
          canExport={canEducation("education.reports.export_sensitive")}
        />,
      )}
    </SchoolAccess>
  );
}
function SchoolSignaturesRoute() { const { apiFetch, canEducation } = useAuth(); const client = useMemo(() => createContractClient(apiFetch), [apiFetch]); return <SchoolAccess permissions={["education.signatures.read", "education.signatures.manage", "education.signatures.validate"]}>{deferred(<SignedArtifactEvidenceWorkspace client={client} canManage={canEducation("education.signatures.manage")} canValidate={canEducation("education.signatures.validate")} />)}</SchoolAccess>; }

type EducationWizardComponent = ComponentType<{
  adapter: import("../features/education/wizards").EducationWizardAdapter;
  canManage: boolean;
  onSaved?: (value: unknown) => void;
}>;

function SchoolWizardRoute({
  permission,
  Wizard,
}: {
  permission: string;
  Wizard: EducationWizardComponent;
}) {
  const { apiFetch } = useAuth();
  const navigate = useNavigate();
  const client = useMemo(() => createContractClient(apiFetch), [apiFetch]);
  const adapter = useMemo(() => createSchoolWizardApi(client), [client]);
  const returnTo = ({
    "education.governance.manage": "/scoala/governance",
    "education.managerial.manage": "/scoala/managerial",
    "education.personnel.manage": "/scoala/personnel",
    "education.evaluations.manage": "/scoala/evaluations",
    "education.declarations.manage": "/scoala/declarations",
    "education.mobility.manage": "/scoala/mobility",
    "education.gradatii.manage": "/scoala/merit",
    "education.portfolios.manage": "/scoala/portfolios",
  } as Record<string, string>)[permission] ?? "/scoala";
  return (
    <SchoolAccess permissions={[permission]}>
      {deferred(<Wizard adapter={adapter} canManage onSaved={() => navigate(returnTo, { replace: true })} />)}
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
      canRecoverPortfolioCustody={has("earchiva.manage") && (has("education.portfolios.school.manage") || has("education.portfolios.manage"))}
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
  const institutionPolicyApi = useMemo(() => createInstitutionPolicyApi(createContractClient(apiFetch)), [apiFetch]);
  const archiveRetentionApi = useMemo(() => createArchiveRetentionApi(createContractClient(apiFetch)), [apiFetch]);
  const regulatorySourcesApi = useMemo(() => createRegulatorySourcesApi(createContractClient(apiFetch)), [apiFetch]);
  return secure(
    "admin.read",
    <AdministrationWorkspace
      api={api}
      institutionPolicyApi={institutionPolicyApi}
      archiveRetentionApi={archiveRetentionApi}
      regulatorySourcesApi={regulatorySourcesApi}
      actorSubject={session?.user.sub}
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

function InstitutionProfileRoute() {
  const { apiFetch, has } = useAuth();
  const api = useMemo(() => createInstitutionPolicyApi(createContractClient(apiFetch)), [apiFetch]);
  return secure("institution.regulatory_profile.read", <RegulatoryProfileWorkspace api={api} canManage={has("institution.regulatory_profile.manage")} canReadOfferings={has("institution.offerings.read")} canManageOfferings={has("institution.offerings.manage")} />);
}

export function App() {
  return (
    <AppThemeProvider>
      <AuthProvider>
        <InstitutionPolicyProvider><BrowserRouter>
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
              <Route path="profil-institutie" element={deferred(<InstitutionProfileRoute />)} />
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
              <Route path="scoala/resurse-delegate" element={deferred(<DelegatedEducationResourcesRoute />)} />
              <Route path="scoala/clase" element={deferred(<SchoolClassesRoute />)} />
              <Route path="scoala/operatiuni" element={deferred(<SchoolOperationsRoute />)} />
              <Route path="scoala/admitere" element={deferred(<AdmissionRoute />)} />
              <Route path="scoala/dashboard" element={<SchoolRoute />} />
              <Route
                path="scoala/dashboard/director"
                element={<SchoolRoute />}
              />
              <Route
                path="scoala/dashboard/director/reports"
                element={deferred(<SchoolReportsRoute />)}
              />
              <Route path="scoala/reports" element={deferred(<SchoolReportsRoute />)} />
              <Route path="scoala/signatures" element={deferred(<SchoolSignaturesRoute />)} />
              <Route
                path="scoala/secretariat"
                element={<SchoolRoute permissions={["education.cockpit.secretariat.read"]} />}
              />
              <Route path="scoala/hr" element={<SchoolRoute permissions={["education.cockpit.hr.read"]} />} />
              <Route path="scoala/committee-cockpit" element={<SchoolRoute permissions={["education.cockpit.committee.read"]} />} />
              <Route path="scoala/inspector" element={<SchoolRoute permissions={["education.cockpit.inspector.read"]} />} />
              <Route
                path="scoala/compliance"
                element={
                  <SchoolRoute permissions={["education.compliance.read"]} />
                }
              />
              <Route path="scoala/decisions" element={<SchoolRoute permissions={["education.decisions.read"]} />} />
              <Route path="scoala/managerial" element={<SchoolRoute permissions={["education.managerial.read"]} />} />
              <Route path="scoala/regulations" element={<SchoolRoute permissions={["education.regulations.read"]} />} />
              <Route path="scoala/committees" element={<SchoolRoute permissions={["education.governance.read"]} />} />
              <Route path="scoala/evaluations" element={<SchoolRoute permissions={["education.evaluations.read"]} />} />
              <Route path="scoala/declarations" element={<SchoolRoute permissions={["education.declarations.read"]} />} />
              <Route path="scoala/mobility" element={<SchoolRoute permissions={["education.mobility.read"]} />} />
              <Route path="scoala/merit" element={<SchoolRoute permissions={["education.gradatii.read"]} />} />
              <Route
                path="scoala/teacher"
                element={deferred(<TeacherPortfolioRoute />)}
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
                path="scoala/portfolios"
                element={
                  <PortfolioReviewRoute />
                }
              />
              <Route
                path="scoala/portfolio"
                element={
                  <SchoolRoute permissions={["education.portfolios.school.read", "education.portfolios.read"]} />
                }
              />
              <Route
                path="scoala/portfolio/me"
                element={deferred(<TeacherPortfolioRoute />)}
              />
              <Route
                path="scoala/portfolio/workflow"
                element={
                  <PortfolioReviewRoute />
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
        </BrowserRouter></InstitutionPolicyProvider>
      </AuthProvider>
    </AppThemeProvider>
  );
}
