import { useEffect, useMemo, useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { Avatar } from '@primereact/ui/avatar';
import { Button } from '@primereact/ui/button';
import { Drawer } from '@primereact/ui/drawer';
import { Toolbar } from '@primereact/ui/toolbar';
import { Bars, SignOut, Times, User } from '@primeicons/react';
import { navigation } from '../app/navigation';
import { useAuth } from '../auth/AuthProvider';
import { ThemeMenu } from './ThemeMenu';

const DESKTOP_QUERY = '(min-width: 768px)';
function useDesktopLayout() {
    const [isDesktop, setIsDesktop] = useState(() => typeof window !== 'undefined' && window.matchMedia(DESKTOP_QUERY).matches);
    useEffect(() => { const media = window.matchMedia(DESKTOP_QUERY); const change = () => setIsDesktop(media.matches); change(); media.addEventListener('change', change); return () => media.removeEventListener('change', change); }, []);
    return isDesktop;
}

function NavigationContent({ tenantTitle, onNavigate }: { tenantTitle: string; onNavigate?: () => void }) {
    const { user, session, login, logout, has } = useAuth();
    const location = useLocation();
    const items = useMemo(() => navigation.filter((item) => (!item.permission || has(item.permission)) && (!item.module || session?.modules.some((module) => module.code === item.module && module.active))), [has, session?.modules]);
    return <div className="flex h-full flex-col gap-4 p-3">
        <Link to="/" className="app-sidebar-brand flex items-center gap-2 px-2 py-1" onClick={onNavigate}><Avatar.Root shape="circle"><Avatar.Fallback>eG</Avatar.Fallback></Avatar.Root><span>{tenantTitle}</span></Link>
        <nav aria-label="Navigație principală" className="flex-1"><p className="px-2 text-sm">Componente</p><ul className="m-0 flex list-none flex-col gap-1 p-0">
            {items.map((item) => <li key={item.to}><Link to={item.to} aria-current={location.pathname === item.to ? 'page' : undefined} className={`block rounded px-2 py-2 no-underline ${location.pathname === item.to ? 'active-nav-item' : ''}`} onClick={onNavigate}>{item.label}</Link></li>)}
        </ul></nav>
        <div className="flex items-center gap-2 px-2">{user ? <><Link to="/profil" className="app-profile-link flex min-w-0 flex-1 items-center gap-2 no-underline" onClick={onNavigate}><Avatar.Root shape="circle"><Avatar.Fallback>{user.name.slice(0, 1).toUpperCase()}</Avatar.Fallback></Avatar.Root><span className="truncate">{user.name}</span></Link><Button aria-label="Deconectare" title="Deconectare" variant="text" rounded iconOnly onClick={() => void logout()}><SignOut /></Button></> : <Button variant="text" onClick={() => void login()}><User /><span>Autentificare</span></Button>}</div>
    </div>;
}

export function AppShell() {
    const { session } = useAuth();
    const isDesktop = useDesktopLayout();
    const [open, setOpen] = useState(false);
    const [tenantTitle, setTenantTitle] = useState('eGuEducation');
    useEffect(() => { if (isDesktop) setOpen(false); }, [isDesktop]);
    useEffect(() => {
        if (session?.institution_name) { setTenantTitle(session.institution_name); return; }
        const controller = new AbortController();
        void fetch('/api/config', { signal: controller.signal }).then(async (response) => response.ok ? response.json() as Promise<unknown> : undefined).then((value) => {
            if (!value || typeof value !== 'object') return;
            const config = value as { institutionName?: unknown; service?: { title?: unknown } };
            const title = typeof config.institutionName === 'string' ? config.institutionName : typeof config.service?.title === 'string' ? config.service.title : undefined;
            if (title?.trim()) setTenantTitle(title.trim());
        }).catch(() => undefined);
        return () => controller.abort();
    }, [session?.institution_name]);
    return <div className="flex min-h-screen">
        {isDesktop && <aside id="main-navigation" aria-label="Navigație principală" className="app-side-navigation w-72 shrink-0"><NavigationContent tenantTitle={tenantTitle} /></aside>}
        {!isDesktop && <Drawer.Root open={open} position="left" onOpenChange={(event: { value?: boolean }) => setOpen(Boolean(event.value))}><Drawer.Portal><Drawer.Backdrop /><Drawer.Popup id="main-navigation" aria-label="Navigație principală" className="w-72"><Drawer.Header><Drawer.Title>{tenantTitle}</Drawer.Title><Button aria-label="Închide navigația" title="Închide navigația" variant="text" rounded iconOnly onClick={() => setOpen(false)}><Times /></Button></Drawer.Header><Drawer.Content><NavigationContent tenantTitle={tenantTitle} onNavigate={() => setOpen(false)} /></Drawer.Content></Drawer.Popup></Drawer.Portal></Drawer.Root>}
        <div className="min-w-0 flex-1"><header><Toolbar.Root className="app-toolbar"><Toolbar.Start><div className="flex items-center gap-2">{!isDesktop && <Button aria-label="Deschide navigația" title="Deschide navigația" variant="text" rounded iconOnly onClick={() => setOpen(true)}><Bars /></Button>}<Link to="/" className="app-title">{tenantTitle}</Link></div></Toolbar.Start><Toolbar.End><ThemeMenu /></Toolbar.End></Toolbar.Root></header><main className="p-4 md:p-6"><Outlet /></main></div>
    </div>;
}
