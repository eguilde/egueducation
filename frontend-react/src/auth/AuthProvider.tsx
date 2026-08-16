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
    consumeReturnTo,
    refreshWithCookie,
    type Tokens
} from './oidc-client';
import { oidcConfig } from './config';

export interface User {
    id: string;
    sub: string;
    name: string;
    email: string;
    email_verified: boolean;
    phone_number: string;
    phone_number_verified: boolean;
    preferred_otp_channel: string;
    locale: 'ro' | 'en';
    roles: string[];
}

export interface SessionContext {
    user: User;
    institution_id: string;
    institution_name: string;
    permissions: string[];
    modules: Array<{ code: string; active: boolean }>;
    authentication: string[];
    gdpr_capabilities: string[];
}

interface AuthValue {
    user: User | null;
    session: SessionContext | null;
    ready: boolean;
    login: () => Promise<void>;
    logout: () => Promise<void>;
    complete: () => Promise<string>;
    completeLogout: () => void;
    has: (permission: string) => boolean;
    updateLocalProfile: (profile: Pick<User, 'id' | 'name' | 'email' | 'email_verified' | 'phone_number' | 'phone_number_verified' | 'locale'>) => void;
    apiFetch: typeof fetch;
}

const AuthContext = createContext<AuthValue | undefined>(undefined);
const config = oidcConfig();

function isSessionContext(value: unknown): value is SessionContext {
    if (!value || typeof value !== 'object') return false;
    const session = value as Partial<SessionContext>;
    return Boolean(
        session.user?.id &&
        session.user.sub &&
        session.user.name &&
        Array.isArray(session.user.roles) &&
        Array.isArray(session.permissions) &&
        Array.isArray(session.modules)
    );
}

async function loadMe(accessToken: string): Promise<SessionContext> {
    const response = await fetch(`${config.apiBaseUrl}/me`, {
        credentials: 'include',
        headers: { Authorization: `Bearer ${accessToken}` }
    });
    if (!response.ok) throw new Error('Nu s-a putut valida sesiunea utilizatorului.');
    const value: unknown = await response.json();
    if (!isSessionContext(value)) throw new Error('Răspuns /api/me invalid.');
    return value;
}

export function AuthProvider({ children }: PropsWithChildren) {
    const [session, setSession] = useState<SessionContext | null>(null);
    const [ready, setReady] = useState(false);
    const [tokens, setTokens] = useState<Tokens | null>(null);
    const tokensRef = useRef<Tokens | null>(null);

    const apply = useCallback(async (next: Tokens) => {
        const validatedSession = await loadMe(next.accessToken);
        tokensRef.current = next;
        setTokens(next);
        setSession(validatedSession);
    }, []);

    useEffect(() => {
        void refreshWithCookie(config)
            .then((next) => next ? apply(next).catch(() => setSession(null)) : undefined)
            .finally(() => setReady(true));
    }, [apply]);

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
        setSession(null);
        // The internal endpoint is a fail-safe for cookie revocation. Complete
        // the standards flow as well when an ID token is available in memory.
        if (idToken) await beginLogout(config, idToken);
    }, []);
    const complete = useCallback(async () => {
        const next = await completeAuthorization(config);
        await apply(next);
        return consumeReturnTo();
    }, [apply]);
    const finishLogout = useCallback(() => completeLogout(), []);
    const has = useCallback((permission: string) => Boolean(
        session?.permissions.includes(permission)
    ), [session]);
    const updateLocalProfile = useCallback<AuthValue['updateLocalProfile']>((profile) => {
        setSession((current) => current ? {
            ...current,
            user: { ...current.user, ...profile }
        } : current);
    }, []);
    const apiFetch = useCallback<typeof fetch>(async (input, init) => {
        let activeTokens = tokensRef.current;
        if (!activeTokens || activeTokens.expiresAt <= Math.floor(Date.now() / 1000) + 30) {
            activeTokens = await refreshWithCookie(config);
            if (!activeTokens) {
                tokensRef.current = null;
                setTokens(null);
                setSession(null);
                throw new Error('Sesiunea a expirat. Autentificați-vă din nou.');
            }
            try {
                await apply(activeTokens);
            } catch (error) {
                tokensRef.current = null;
                setTokens(null);
                setSession(null);
                throw error;
            }
        }

        const execute = (token: string) => {
            const headers = new Headers(init?.headers);
            headers.set('Authorization', `Bearer ${token}`);
            return fetch(input, { ...init, credentials: 'include', headers });
        };

        let response = await execute(activeTokens.accessToken);
        if (response.status !== 401) return response;

        const refreshed = await refreshWithCookie(config);
        if (!refreshed) {
            tokensRef.current = null;
            setTokens(null);
            setSession(null);
            return response;
        }
        try {
            await apply(refreshed);
        } catch (error) {
            tokensRef.current = null;
            setTokens(null);
            setSession(null);
            throw error;
        }
        response = await execute(refreshed.accessToken);
        return response;
    }, [apply]);

    const value = useMemo<AuthValue>(() => ({
        user: session?.user ?? null,
        session,
        ready,
        login,
        logout,
        complete,
        completeLogout: finishLogout,
        has,
        updateLocalProfile,
        apiFetch
    }), [apiFetch, complete, finishLogout, has, login, logout, ready, session, updateLocalProfile]);

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export const useAuth = () => {
    const value = useContext(AuthContext);
    if (!value) throw new Error('useAuth must be used within AuthProvider');
    return value;
};
