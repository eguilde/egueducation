import { lazy, Suspense, useMemo, type ReactNode } from 'react';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { AuthProvider, useAuth } from '../auth/AuthProvider';
import { AppShell } from '../components/AppShell';
import { AppThemeProvider } from '../components/ThemeMenu';
import { createAdminApi } from '../features/admin/api';
import { createArchiveApi } from '../features/earchiva/api';
import { createProfileApi } from '../features/profile/api';
import { browserPasskeyCeremony, supportsWebAuthn } from '../features/profile/webauthn';
import { createRegistraturaApi } from '../features/registratura/api';
import { createWorkflowApi } from '../features/workflow/api';
import {
    CallbackPage,
    CanaryActivationPage,
    LandingPage,
    LogoutCallbackPage,
    RegistrationPage,
    RequireAuthenticated,
    RequirePermission,
} from './Pages';

const AdministrationWorkspace = lazy(() => import('../features/admin/AdministrationWorkspace').then((module) => ({ default: module.AdministrationWorkspace })));
const ArchiveWorkspace = lazy(() => import('../features/earchiva/ArchiveWorkspace').then((module) => ({ default: module.ArchiveWorkspace })));
const EducationWorkspace = lazy(() => import('../features/education/EducationWorkspace').then((module) => ({ default: module.EducationWorkspace })));
const ProfileWorkspace = lazy(() => import('../features/profile/ProfileWorkspace').then((module) => ({ default: module.ProfileWorkspace })));
const RegistraturaWorkspace = lazy(() => import('../features/registratura/RegistraturaWorkspace').then((module) => ({ default: module.RegistraturaWorkspace })));
const WorkflowWorkspace = lazy(() => import('../features/workflow/WorkflowWorkspace').then((module) => ({ default: module.WorkflowWorkspace })));

const deferred = (children: ReactNode) => (
    <Suspense fallback={<div className="p-6" role="status">Se încarcă…</div>}>
        {children}
    </Suspense>
);

const secure = (permission: string, children: ReactNode, module?: string) => (
    <RequirePermission permission={permission} module={module}>{children}</RequirePermission>
);

function RegistraturaRoute() {
    const { apiFetch, has, session } = useAuth();
    const api = useMemo(() => createRegistraturaApi(apiFetch), [apiFetch]);
    const tenantKey = `${window.location.host}:${session?.institution_id ?? 'none'}`;
    return secure('registratura.read', (
        <RegistraturaWorkspace
            api={api}
            tenantKey={tenantKey}
            canManage={has('registratura.manage')}
            canManageWorkflow={has('registratura.manage') || has('workflow.manage')}
            canReadAdminUsers={has('admin.users.read')}
            canReadLinks={has('registratura.links.read')}
            canManageLinks={has('registratura.links.manage')}
        />
    ), 'registratura');
}

function WorkflowRoute() {
    const { apiFetch, has } = useAuth();
    const api = useMemo(
        () => createWorkflowApi(apiFetch),
        [apiFetch]
    );
    return secure('workflow.read', (
        <WorkflowWorkspace
            api={api}
            canTransition={has('workflow.transition')}
            canManage={has('workflow.manage')}
        />
    ), 'workflow');
}

function ArchiveRoute() {
    const { apiFetch, has } = useAuth();
    const api = useMemo(
        () => createArchiveApi(apiFetch),
        [apiFetch]
    );
    // Archivists are granted the dedicated earchiva permission.  Do not
    // additionally require a role-derived module toggle in the client.
    return secure('earchiva.read', (
        <ArchiveWorkspace api={api} canManage={has('earchiva.manage')} canReadContent={has('earchiva.content.read')} canReview={has('earchiva.review')} />
    ));
}

function ProfileRoute() {
    const { apiFetch, updateLocalProfile, user } = useAuth();
    const api = useMemo(() => createProfileApi(apiFetch), [apiFetch]);
    return (
        <RequireAuthenticated>
            {user && <ProfileWorkspace
                user={user}
                api={api}
                ceremony={supportsWebAuthn() ? browserPasskeyCeremony : undefined}
                onUpdated={updateLocalProfile}
            />}
        </RequireAuthenticated>
    );
}

function AdministrationRoute() {
    const { apiFetch, has, session } = useAuth();
    const api = useMemo(() => createAdminApi(apiFetch), [apiFetch]);
    return secure('admin.read', (
        <AdministrationWorkspace
            api={api}
            institutionName={session?.institution_name ?? 'Instituția curentă'}
            permissions={{
                dashboard: has('admin.read'),
                usersRead: has('admin.users.read'),
                usersManage: has('admin.users.manage'),
                rolesRead: has('admin.roles.read'),
                rolesManage: has('admin.roles.manage'),
                modulesRead: has('admin.modules.read'),
                modulesManage: has('admin.modules.manage')
            }}
            canAccess={has}
        />
    ));
}

export function App() {
    return (
        <AppThemeProvider>
            <AuthProvider>
                <BrowserRouter>
                    <Routes>
                        <Route path="/auth/callback" element={<CallbackPage />} />
                        <Route path="/auth/e2e-canary" element={<CanaryActivationPage />} />
                        <Route path="/auth/logout" element={<LogoutCallbackPage />} />
                        <Route path="/auth/register" element={<RegistrationPage />} />
                        <Route path="/documente" element={<Navigate to="/registratura" replace />} />
                        <Route path="/flux" element={<Navigate to="/flux-documente" replace />} />
                        <Route path="/profile" element={<Navigate to="/profil" replace />} />
                        <Route path="/admin" element={<Navigate to="/administrare" replace />} />
                        <Route path="/education/*" element={<Navigate to="/scoala" replace />} />
                        <Route element={<AppShell />}>
                            <Route index element={<LandingPage />} />
                            <Route path="profil" element={deferred(<ProfileRoute />)} />
                            <Route path="registratura" element={deferred(<RegistraturaRoute />)} />
                            <Route path="flux-documente" element={deferred(<WorkflowRoute />)} />
                            <Route path="earchiva" element={deferred(<ArchiveRoute />)} />
                            <Route path="scoala" element={secure('education.read', deferred(<EducationWorkspace />), 'education')} />
                            <Route path="administrare" element={deferred(<AdministrationRoute />)} />
                        </Route>
                    </Routes>
                </BrowserRouter>
            </AuthProvider>
        </AppThemeProvider>
    );
}
