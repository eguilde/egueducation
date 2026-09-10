import {
    createContext,
    useCallback,
    useContext,
    useEffect,
    useMemo,
    useRef,
    useState,
    type PropsWithChildren
} from 'react';
import {
    beginAuthorization,
    beginLogout,
    completeAuthorization,
    completeLogout,
    refreshWithCookie,
    type Tokens
} from './oidc-client';
import { oidcConfig } from './config';
import { createContractClient, type ContractClient } from '../api/client';
import type { components } from '../api/generated';
import { validateSessionContext } from '../api/runtime-validators';

export type User = components['schemas']['SessionUser'];
export type SessionContext = components['schemas']['SessionContext'];
export type EducationDelegationGrant = components['schemas']['EducationActiveDelegationGrant'];
export type EducationAuthorizationScope =
    | { resourceType: 'institution' }
    | { resourceType: Exclude<EducationDelegationGrant['resource_type'], 'institution'>; resourceId: string };

export function educationPermissionImplies(granted: string, requested: string): boolean {
    if (granted === requested && granted !== '') return true;
    return granted.startsWith('education.')
        && requested.startsWith('education.')
        && granted.endsWith('.manage')
        && requested === `${granted.slice(0, -'.manage'.length)}.read`;
}

interface AuthValue {
    user: User | null;
    session: SessionContext | null;
    ready: boolean;
    authorizationReady: boolean;
    educationGrants: readonly EducationDelegationGrant[];
    login: () => Promise<void>;
    logout: () => Promise<void>;
    complete: () => Promise<string>;
    completeLogout: () => void;
    has: (permission: string) => boolean;
    canEducation: (permission: string, scope?: EducationAuthorizationScope) => boolean;
    refreshEducationAuthorization: () => Promise<void>;
    updateLocalProfile: (profile: Pick<User, 'id' | 'name' | 'email' | 'email_verified' | 'phone_number' | 'phone_number_verified' | 'locale'>) => void;
    apiFetch: typeof fetch;
    apiClient: ContractClient;
}

const AuthContext = createContext<AuthValue | undefined>(undefined);
const config = oidcConfig();

async function loadMe(accessToken: string): Promise<SessionContext> {
    const client = createContractClient(async (request) => {
        const headers = new Headers(request.headers);
        headers.set('Authorization', `Bearer ${accessToken}`);
        return fetch(new Request(request, { credentials: 'include', headers }));
    }, config.apiBaseUrl);
    const { data, response } = await client.GET('/api/me');
    if (!response.ok || !data) {
        throw new Error('Nu s-a putut valida sesiunea utilizatorului.');
    }
    if (!validateSessionContext(data)) {
        throw new Error('Răspuns /api/me invalid conform contractului OpenAPI.');
    }
    return data;
}

function isActiveGrantResponse(value: unknown, session: SessionContext): value is components['schemas']['EducationActiveDelegationGrantsResponse'] {
    if (!value || typeof value !== 'object') return false;
    const response = value as Record<string, unknown>;
    if (response.tenant_code !== session.tenant_code || response.institution_id !== session.institution_id ||
        typeof response.revision !== 'string' || typeof response.evaluated_at !== 'string' || !Array.isArray(response.grants)) return false;
    return response.grants.every((grant) => {
        if (!grant || typeof grant !== 'object') return false;
        const entry = grant as Record<string, unknown>;
        return typeof entry.permission_code === 'string' && typeof entry.resource_id === 'string' &&
            (entry.resource_type === 'institution' || entry.resource_type === 'portfolio' || entry.resource_type === 'meeting' ||
                entry.resource_type === 'decision' || entry.resource_type === 'regulation' || entry.resource_type === 'personnel');
    });
}

async function loadActiveEducationGrants(accessToken: string, session: SessionContext): Promise<EducationDelegationGrant[]> {
    const client = createContractClient(async (request) => {
        const headers = new Headers(request.headers);
        headers.set('Authorization', `Bearer ${accessToken}`);
        return fetch(new Request(request, { credentials: 'include', headers }));
    }, config.apiBaseUrl);
    const { data, response } = await client.GET('/api/education/delegations/active-grants');
    if (!response.ok || !isActiveGrantResponse(data, session)) throw new Error('Snapshot-ul de delegări educaționale nu este valid.');
    return data.grants;
}

export function AuthProvider({ children }: PropsWithChildren) {
    const [session, setSession] = useState<SessionContext | null>(null);
    const [ready, setReady] = useState(false);
    const [authorizationReady, setAuthorizationReady] = useState(false);
    const [educationGrants, setEducationGrants] = useState<EducationDelegationGrant[]>([]);
    const [tokens, setTokens] = useState<Tokens | null>(null);
    const tokensRef = useRef<Tokens | null>(null);
    const sessionRef = useRef<SessionContext | null>(null);
    const clearAuthorization = useCallback(() => { setEducationGrants([]); setAuthorizationReady(true); }, []);

    const apply = useCallback(async (next: Tokens) => {
        const validatedSession = await loadMe(next.accessToken);
        tokensRef.current = next;
        sessionRef.current = validatedSession;
        setTokens(next);
        setAuthorizationReady(false);
        setSession(validatedSession);
        try {
            const grants = await loadActiveEducationGrants(next.accessToken, validatedSession);
            if (sessionRef.current === validatedSession) setEducationGrants(grants);
        } catch {
            if (sessionRef.current === validatedSession) setEducationGrants([]);
        } finally {
            if (sessionRef.current === validatedSession) setAuthorizationReady(true);
        }
    }, []);
    const refreshEducationAuthorization = useCallback(async () => {
        const activeTokens = tokensRef.current;
        const currentSession = sessionRef.current;
        if (!activeTokens || !currentSession) { clearAuthorization(); return; }
        setAuthorizationReady(false);
        try {
            const grants = await loadActiveEducationGrants(activeTokens.accessToken, currentSession);
            if (sessionRef.current === currentSession) setEducationGrants(grants);
        } catch {
            if (sessionRef.current === currentSession) setEducationGrants([]);
        } finally {
            if (sessionRef.current === currentSession) setAuthorizationReady(true);
        }
    }, [clearAuthorization]);

    useEffect(() => {
        void refreshWithCookie(config)
            .then((next) => next ? apply(next).catch(() => { sessionRef.current = null; setSession(null); clearAuthorization(); }) : clearAuthorization())
            .finally(() => setReady(true));
    }, [apply, clearAuthorization]);
    useEffect(() => {
        if (!session || !tokens) return;
        const refresh = () => { void refreshEducationAuthorization(); };
        const onVisibility = () => { if (document.visibilityState === 'visible') refresh(); };
        window.addEventListener('focus', refresh);
        document.addEventListener('visibilitychange', onVisibility);
        const interval = window.setInterval(refresh, 15_000);
        return () => { window.removeEventListener('focus', refresh); document.removeEventListener('visibilitychange', onVisibility); window.clearInterval(interval); };
    }, [refreshEducationAuthorization, session, tokens]);

    const login = useCallback(
        () => beginAuthorization(config, `${location.pathname}${location.search}`),
        []
    );
    const logout = useCallback(async () => {
        const idToken = tokensRef.current?.idToken;
        if (tokensRef.current) {
            await fetch(`${config.apiBaseUrl}/oidc/session/logout`, {
                method: 'POST',
                credentials: 'include',
                headers: { Authorization: `Bearer ${tokensRef.current.accessToken}` }
            }).catch(() => undefined);
        }
        setTokens(null);
        tokensRef.current = null;
        sessionRef.current = null;
        setSession(null);
        clearAuthorization();
        // The internal endpoint is a fail-safe for cookie revocation. Complete
        // the standards flow as well when an ID token is available in memory.
        if (idToken) await beginLogout(config, idToken);
    }, []);
    const complete = useCallback(async () => {
        const next = await completeAuthorization(config);
        await apply(next);
        return next.returnTo ?? '/';
    }, [apply]);
    const finishLogout = useCallback(() => completeLogout(), []);
    const has = useCallback((permission: string) => Boolean(
        session?.permissions.includes(permission)
    ), [session]);
    const canEducation = useCallback<AuthValue['canEducation']>((permission, scope = { resourceType: 'institution' }) => {
        if (session?.permissions.some((granted) => educationPermissionImplies(granted, permission))) return true;
        if (!authorizationReady) return false;
        if (scope.resourceType === 'institution') return educationGrants.some((grant) => educationPermissionImplies(grant.permission_code, permission) && grant.resource_type === 'institution');
        return educationGrants.some((grant) => educationPermissionImplies(grant.permission_code, permission) && grant.resource_type === scope.resourceType && grant.resource_id === scope.resourceId);
    }, [authorizationReady, educationGrants, session?.permissions]);
    const updateLocalProfile = useCallback<AuthValue['updateLocalProfile']>((profile) => {
        setSession((current) => current ? {
            ...current,
            user: { ...current.user, ...profile }
        } : current);
    }, []);
    const apiFetch = useCallback<typeof fetch>(async (input, init) => {
        // openapi-fetch supplies a fully constructed Request. Preserve its
        // contract headers (notably multipart boundaries and Accept) when
        // adding the bearer token. Keep an unused template clone so a 401
        // retry never attempts to reuse an already-consumed request body.
        const requestTemplate = input instanceof Request ? input.clone() : input;
        let activeTokens = tokensRef.current;
        if (!activeTokens || activeTokens.expiresAt <= Math.floor(Date.now() / 1000) + 30) {
            activeTokens = await refreshWithCookie(config);
            if (!activeTokens) {
                tokensRef.current = null;
                sessionRef.current = null;
                setTokens(null);
                setSession(null);
                clearAuthorization();
                throw new Error('Sesiunea a expirat. Autentificați-vă din nou.');
            }
            try {
                await apply(activeTokens);
            } catch (error) {
                tokensRef.current = null;
                sessionRef.current = null;
                setTokens(null);
                setSession(null);
                clearAuthorization();
                throw error;
            }
        }

        const execute = (token: string) => {
            const requestInput = requestTemplate instanceof Request ? requestTemplate.clone() : requestTemplate;
            const headers = new Headers(requestInput instanceof Request ? requestInput.headers : undefined);
            new Headers(init?.headers).forEach((value, name) => headers.set(name, value));
            headers.set('Authorization', `Bearer ${token}`);
            return fetch(requestInput, { ...init, credentials: 'include', headers });
        };

        let response = await execute(activeTokens.accessToken);
        if (response.status === 403 && requestTemplate instanceof Request && new URL(requestTemplate.url, window.location.origin).pathname.startsWith('/api/education/')) void refreshEducationAuthorization();
        if (response.status !== 401) return response;

        const refreshed = await refreshWithCookie(config);
        if (!refreshed) {
            tokensRef.current = null;
            sessionRef.current = null;
            setTokens(null);
            setSession(null);
            clearAuthorization();
            return response;
        }
        try {
            await apply(refreshed);
        } catch (error) {
            tokensRef.current = null;
            sessionRef.current = null;
            setTokens(null);
            setSession(null);
            clearAuthorization();
            throw error;
        }
        response = await execute(refreshed.accessToken);
        return response;
    }, [apply, clearAuthorization, refreshEducationAuthorization]);
    const apiClient = useMemo(
        () => createContractClient((request) => apiFetch(request), config.apiBaseUrl),
        [apiFetch]
    );

    const value = useMemo<AuthValue>(() => ({
        user: session?.user ?? null,
        session,
        ready,
        authorizationReady,
        educationGrants,
        login,
        logout,
        complete,
        completeLogout: finishLogout,
        has,
        canEducation,
        refreshEducationAuthorization,
        updateLocalProfile,
        apiFetch,
        apiClient
    }), [apiClient, apiFetch, authorizationReady, canEducation, complete, educationGrants, finishLogout, has, login, logout, ready, refreshEducationAuthorization, session, updateLocalProfile]);

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export const useAuth = () => {
    const value = useContext(AuthContext);
    if (!value) throw new Error('useAuth must be used within AuthProvider');
    return value;
};
