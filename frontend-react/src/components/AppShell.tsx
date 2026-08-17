import { useEffect, useMemo, useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { Avatar } from '@primereact/ui/avatar';
import { Button } from '@primereact/ui/button';
import { Sidebar } from '@primereact/ui/sidebar';
import { Toolbar } from '@primereact/ui/toolbar';
import { Bars, SignOut, User } from '@primeicons/react';
import { navigation } from '../app/navigation';
import { useAuth } from '../auth/AuthProvider';
import { ThemeMenu } from './ThemeMenu';

const DESKTOP_QUERY = '(min-width: 768px)';

function useDesktopLayout() {
    const [isDesktop, setIsDesktop] = useState(
        () => typeof window !== 'undefined' && window.matchMedia(DESKTOP_QUERY).matches
    );

    useEffect(() => {
        const media = window.matchMedia(DESKTOP_QUERY);
        const change = () => setIsDesktop(media.matches);
        change();
        media.addEventListener('change', change);
        return () => media.removeEventListener('change', change);
    }, []);

    return isDesktop;
}

export function AppShell() {
    const { user, session, login, logout, has } = useAuth();
    const location = useLocation();
    const isDesktop = useDesktopLayout();
    const [open, setOpen] = useState(isDesktop);
    const [tenantTitle, setTenantTitle] = useState('eGuEducation');

    useEffect(() => setOpen(isDesktop), [isDesktop]);

    useEffect(() => {
        if (session?.institution_name) {
            setTenantTitle(session.institution_name);
            return;
        }

        const controller = new AbortController();
        void fetch('/api/config', { signal: controller.signal })
            .then(async (response) => response.ok ? response.json() as Promise<unknown> : undefined)
            .then((value) => {
                if (!value || typeof value !== 'object') return;
                const config = value as { institutionName?: unknown; service?: { title?: unknown } };
                const title = typeof config.institutionName === 'string'
                    ? config.institutionName
                    : typeof config.service?.title === 'string' ? config.service.title : undefined;
                if (title?.trim()) setTenantTitle(title.trim());
            })
            .catch(() => undefined);
        return () => controller.abort();
    }, [session?.institution_name]);

    const items = useMemo(
        () => navigation.filter((item) =>
            (!item.permission || has(item.permission)) &&
            (!item.module || session?.modules.some((module) => module.code === item.module && module.active))
        ),
        [has, session?.modules]
    );

    return (
        <Sidebar.Layout className="min-h-screen">
            {!isDesktop && <Sidebar.Backdrop />}
            <Sidebar.Root
                id="main-navigation"
                aria-label="Navigație principală"
                side="left"
                width="18rem"
                collapsible={isDesktop ? 'none' : 'offcanvas'}
                overlay={!isDesktop}
                open={open}
                onOpenChange={(event: { value?: boolean }) => setOpen(Boolean(event.value))}
            >
                <Sidebar.Spacer />
                <Sidebar.Aside>
                    <Sidebar.Panel>
                        <Sidebar.Header>
                            <Sidebar.Menu>
                                <Sidebar.MenuItem>
                                    <Sidebar.MenuButton as={Link} to="/" className="app-sidebar-brand">
                                        <Avatar.Root shape="circle">
                                            <Avatar.Fallback>eG</Avatar.Fallback>
                                        </Avatar.Root>
                                        <span>eGuEducation</span>
                                    </Sidebar.MenuButton>
                                    {!isDesktop && (
                                        <Sidebar.MenuAction
                                            as={Button}
                                            aria-label="Închide navigația"
                                            title="Închide navigația"
                                            variant="text"
                                            rounded
                                            iconOnly
                                            onClick={() => setOpen(false)}
                                        >
                                            <Bars />
                                        </Sidebar.MenuAction>
                                    )}
                                </Sidebar.MenuItem>
                            </Sidebar.Menu>
                        </Sidebar.Header>

                        <Sidebar.Content>
                            <Sidebar.Group>
                                <Sidebar.GroupLabel>Componente</Sidebar.GroupLabel>
                                <Sidebar.GroupContent>
                                    <Sidebar.Menu>
                                        {items.map((item) => (
                                            <Sidebar.MenuItem key={item.to}>
                                                <Sidebar.MenuButton
                                                    as={Link}
                                                    to={item.to}
                                                    isActive={location.pathname === item.to}
                                                    onClick={() => {
                                                        if (!isDesktop) setOpen(false);
                                                    }}
                                                >
                                                    <span>{item.label}</span>
                                                </Sidebar.MenuButton>
                                            </Sidebar.MenuItem>
                                        ))}
                                    </Sidebar.Menu>
                                </Sidebar.GroupContent>
                            </Sidebar.Group>
                        </Sidebar.Content>

                        <Sidebar.Footer>
                            <Sidebar.Menu>
                                <Sidebar.MenuItem>
                                    {user ? (
                                        <Sidebar.MenuButton as={Link} to="/profil" className="app-profile-link">
                                            <Avatar.Root shape="circle">
                                                <Avatar.Fallback>{user.name.slice(0, 1).toUpperCase()}</Avatar.Fallback>
                                            </Avatar.Root>
                                            <span>{user.name}</span>
                                        </Sidebar.MenuButton>
                                    ) : (
                                        <Sidebar.MenuButton as={Button} variant="text" onClick={() => void login()}>
                                            <User />
                                            <span>Autentificare</span>
                                        </Sidebar.MenuButton>
                                    )}
                                    {user && (
                                        <Sidebar.MenuAction
                                            as={Button}
                                            aria-label="Deconectare"
                                            title="Deconectare"
                                            variant="text"
                                            rounded
                                            iconOnly
                                            onClick={() => void logout()}
                                        >
                                            <SignOut />
                                        </Sidebar.MenuAction>
                                    )}
                                </Sidebar.MenuItem>
                            </Sidebar.Menu>
                        </Sidebar.Footer>
                    </Sidebar.Panel>
                </Sidebar.Aside>
            </Sidebar.Root>

            <Sidebar.Main>
                <header>
                    <Toolbar.Root className="app-toolbar">
                        <Toolbar.Start>
                            <div className="flex items-center gap-2">
                                {!isDesktop && (
                                    <Button
                                        aria-label="Deschide navigația"
                                        title="Deschide navigația"
                                        variant="text"
                                        rounded
                                        iconOnly
                                        onClick={() => setOpen(true)}
                                    >
                                        <Bars />
                                    </Button>
                                )}
                                <Link to="/" className="app-title">{tenantTitle}</Link>
                            </div>
                        </Toolbar.Start>
                        <Toolbar.End>
                            <ThemeMenu />
                        </Toolbar.End>
                    </Toolbar.Root>
                </header>
                <main className="p-4 md:p-6">
                    <Outlet />
                </main>
            </Sidebar.Main>
        </Sidebar.Layout>
    );
}
