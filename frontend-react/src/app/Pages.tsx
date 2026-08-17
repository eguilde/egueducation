import { Card } from '@primereact/ui/card';
import { Button } from '@primereact/ui/button';
import { InputText } from '@primereact/ui/inputtext';
import { Message } from '@primereact/ui/message';
import { ProgressSpinner } from '@primereact/ui/progressspinner';
import { useEffect, useState, type ChangeEvent, type FormEvent } from 'react';
import { Link, Navigate, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthProvider';

export function LandingPage() {
    const { user, login } = useAuth();
    return (
        <section className="landing">
            <h1>eGuEducation</h1>
            <p>{user ? 'Selectați un modul din navigație.' : 'Autentificarea este necesară pentru acces.'}</p>
            {!user && <Button onClick={() => void login()}>Autentificare</Button>}
        </section>
    );
}
const Panel = ({ title, children }: { title: string; children: React.ReactNode }) => <Card.Root><Card.Body><Card.Title><h1>{title}</h1></Card.Title><Card.Content>{children}</Card.Content></Card.Body></Card.Root>;
export function RegistrationPage() { const { login } = useAuth(); return <main className="flex min-h-screen items-center justify-center p-6"><Panel title="Acces eGuEducation"><div className="flex max-w-xl flex-col gap-4"><p>Conturile sunt create și asociate instituției de un administrator autorizat. Auto-înregistrarea publică nu este disponibilă, pentru a proteja datele școlare și separarea dintre instituții.</p><p>Dacă aveți deja cont, continuați cu autentificarea. Pentru un cont nou, contactați administratorul instituției.</p><div className="flex flex-wrap gap-2"><Button onClick={() => void login()}>Autentificare</Button><Button as={Link} to="/" variant="outlined">Prima pagină</Button></div></div></Panel></main>; }
export function CanaryActivationPage() {
    const { login } = useAuth();
    const [activationKey, setActivationKey] = useState('');
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string>();

    const activate = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        const key = activationKey.trim();
        if (!key || submitting) return;
        setSubmitting(true);
        setError(undefined);
        try {
            const response = await fetch('/api/oidc/e2e-canary/session', {
                method: 'POST',
                headers: { Authorization: `Bearer ${key}` },
                credentials: 'include',
                cache: 'no-store',
                redirect: 'error',
            });
            setActivationKey('');
            if (response.status !== 204) throw new Error('Activarea sesiunii de test a fost refuzată.');
            await login();
        } catch (reason) {
            setActivationKey('');
            setError(reason instanceof Error ? reason.message : 'Sesiunea de test nu a putut fi activată.');
        } finally {
            setSubmitting(false);
        }
    };

    return (
        <main className="flex min-h-screen items-center justify-center p-6">
            <Panel title="Sesiune controlată de verificare">
                <form className="flex w-full max-w-md flex-col gap-4" onSubmit={(event) => void activate(event)}>
                    <p>Introduceți cheia temporară furnizată operatorului autorizat. Cheia rămâne numai în memoria acestei pagini și nu este salvată.</p>
                    {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
                    <label htmlFor="canary-activation-key">Cheie temporară de activare</label>
                    <InputText
                        id="canary-activation-key"
                        aria-label="Cheie temporară de activare"
                        type="password"
                        autoComplete="off"
                        value={activationKey}
                        disabled={submitting}
                        onChange={(event: ChangeEvent<HTMLInputElement>) => setActivationKey(event.target.value)}
                    />
                    <div className="flex flex-wrap gap-2">
                        <Button type="submit" disabled={submitting || !activationKey.trim()}>{submitting ? 'Se activează…' : 'Activează și autentifică'}</Button>
                        <Button as={Link} to="/" variant="outlined" severity="secondary">Anulează</Button>
                    </div>
                </form>
            </Panel>
        </main>
    );
}
export function WorkspacePage({ title, description }: { title: string; description: string }) { return <Panel title={title}><p>{description}</p></Panel>; }
export function ProfilePage() { const { user } = useAuth(); return <Panel title="Profil utilizator"><dl><dt>Nume</dt><dd>{user?.name}</dd><dt>E-mail</dt><dd>{user?.email ?? '—'}</dd></dl></Panel>; }
const Spinner = () => <ProgressSpinner.Root><ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range></ProgressSpinner.Root>;
export function CallbackPage() { const { complete } = useAuth(); const navigate = useNavigate(); const [error, setError] = useState<string>(); useEffect(() => { void complete().then((returnTo) => navigate(returnTo, { replace: true })).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : 'Autentificarea nu a putut fi finalizată.')); }, [complete, navigate]); if (error) return <Panel title="Autentificare nereușită"><p role="alert">{error}</p><Button as={Link} to="/" variant="outlined">Înapoi la prima pagină</Button></Panel>; return <div className="flex justify-center p-8"><Spinner /></div>; }
export function LogoutCallbackPage() { const { completeLogout } = useAuth(); const navigate = useNavigate(); const [error, setError] = useState<string>(); useEffect(() => { try { completeLogout(); navigate('/', { replace: true }); } catch (reason) { setError(reason instanceof Error ? reason.message : 'Deconectarea OIDC nu a putut fi validată.'); } }, [completeLogout, navigate]); if (error) return <Panel title="Deconectare nereușită"><p role="alert">{error}</p><Button as={Link} to="/" variant="outlined">Înapoi la prima pagină</Button></Panel>; return <div className="flex justify-center p-8"><Spinner /></div>; }
export function RequirePermission({ permission, module, children }: { permission: string; module?: string; children: React.ReactNode }) { const { ready, has, session } = useAuth(); if (!ready) return <div className="flex justify-center p-8"><Spinner /></div>; const moduleActive = !module || Boolean(session?.modules.some((item) => item.code === module && item.active)); return has(permission) && moduleActive ? <>{children}</> : <Navigate to="/" replace />; }
export function RequireAuthenticated({ children }: { children: React.ReactNode }) { const { ready, user } = useAuth(); if (!ready) return <div className="flex justify-center p-8"><Spinner /></div>; return user ? <>{children}</> : <Navigate to="/" replace />; }
