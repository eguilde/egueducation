import { useCallback, useEffect, useState, type ChangeEvent } from 'react';
import { Button } from '@primereact/ui/button';
import { Card } from '@primereact/ui/card';
import { DataTable } from '@primereact/ui/datatable';
import { InputText } from '@primereact/ui/inputtext';
import { Message } from '@primereact/ui/message';
import { ProgressSpinner } from '@primereact/ui/progressspinner';
import { Select } from '@primereact/ui/select';
import type { SelectValueChangeEvent } from 'primereact/select';
import { Tag } from '@primereact/ui/tag';
import { Key, Refresh } from '@primeicons/react';
import { createProfileApi } from './api';
import type { PasskeyCeremony, PasskeyCredential, ProfileApi, ProfileUser } from './types';

const Spinner = () => (
    <ProgressSpinner.Root>
        <ProgressSpinner.Range><ProgressSpinner.Track /><ProgressSpinner.Value /></ProgressSpinner.Range>
    </ProgressSpinner.Root>
);

export interface ProfileWorkspaceProps {
    user: ProfileUser;
    api?: ProfileApi;
    ceremony?: PasskeyCeremony;
    onUpdated?: (user: ProfileUser) => void;
}

export function ProfileWorkspace({ user, api = createProfileApi(), ceremony, onUpdated }: ProfileWorkspaceProps) {
    const [form, setForm] = useState<ProfileUser>(user);
    const [passkeys, setPasskeys] = useState<PasskeyCredential[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string>();
    const [notice, setNotice] = useState<string>();
	const [walletActive, setWalletActive] = useState(false);

    useEffect(() => setForm(user), [user]);
    const loadPasskeys = useCallback(async () => {
        setLoading(true);
        setError(undefined);
        try {
            setPasskeys(await api.passkeys());
        } catch (reason) {
            setError(readError(reason));
        } finally {
            setLoading(false);
        }
    }, [api]);
    useEffect(() => { void loadPasskeys(); }, [loadPasskeys]);

    const save = async () => {
        if (!form.name.trim()) return;
        setSaving(true);
        setError(undefined);
        setNotice(undefined);
        try {
            const updated = await api.update({
                name: form.name.trim(),
                phone_number: form.phone_number.trim(),
                locale: form.locale
            });
            setForm(updated);
            onUpdated?.(updated);
            setNotice('Profilul a fost actualizat. Dacă telefonul a fost schimbat, acesta trebuie reverificat.');
        } catch (reason) {
            setError(readError(reason));
        } finally {
            setSaving(false);
        }
    };

    const enroll = async () => {
        if (!ceremony) return;
        setError(undefined);
        setNotice(undefined);
        setSaving(true);
        try {
            const options = await api.registrationOptions();
            const result = await ceremony.register(options);
            await api.finishRegistration(result);
            await loadPasskeys();
            setNotice('Cheia de acces a fost înregistrată.');
        } catch (reason) {
            setError(readError(reason));
        } finally {
            setSaving(false);
        }
    };

	const activateWallet = async () => {
		setSaving(true); setError(undefined); setNotice(undefined);
		try { await api.activateEUDIWallet(); setWalletActive(true); setNotice('EUDI Wallet este activ pentru profilul curent.'); }
		catch (reason) { setError(readError(reason)); }
		finally { setSaving(false); }
	};

    return (
        <section aria-label="Profil utilizator" className="flex flex-col gap-4">
            <div><h1>Profil</h1><p>Setările personale sunt separate de drepturile și apartenențele tenantului.</p></div>
            {error && <Message.Root severity="error"><Message.Content><Message.Text>{error}</Message.Text></Message.Content></Message.Root>}
            {notice && <Message.Root severity="info"><Message.Content><Message.Text>{notice}</Message.Text></Message.Content></Message.Root>}

            <Card.Root><Card.Body><Card.Content>
                <div className="flex flex-col gap-3">
                    <InputText aria-label="Nume" value={form.name} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm({ ...form, name: event.target.value })} />
                    <InputText aria-label="E-mail" value={form.email} disabled />
                    <InputText aria-label="Telefon" value={form.phone_number} onChange={(event: ChangeEvent<HTMLInputElement>) => setForm({ ...form, phone_number: event.target.value })} />
                    <Select.Root
                        value={form.locale}
                        options={[{ label: 'Română', value: 'ro' }, { label: 'English', value: 'en' }]}
                        optionLabel="label"
                        optionValue="value"
                        onValueChange={(event: SelectValueChangeEvent) => setForm({ ...form, locale: event.value as 'ro' | 'en' })}
                    >
                        <Select.Trigger><Select.Value /></Select.Trigger>
                        <Select.Portal><Select.Positioner><Select.Popup><Select.List /></Select.Popup></Select.Positioner></Select.Portal>
                    </Select.Root>
                    <div className="flex flex-wrap gap-2">
                        <Tag value={form.email_verified ? 'E-mail verificat' : 'E-mail neverificat'} severity={form.email_verified ? 'success' : 'warn'} />
                        <Tag value={form.phone_number_verified ? 'Telefon verificat' : 'Telefon neverificat'} severity={form.phone_number_verified ? 'success' : 'warn'} />
                    </div>
                    <div><Button disabled={saving || !form.name.trim()} onClick={() => void save()}>{saving ? 'Se salvează…' : 'Salvează profilul'}</Button></div>
                </div>
            </Card.Content></Card.Body></Card.Root>

			<Card.Root><Card.Body><Card.Content><div className="flex flex-col gap-3">
				<div><h2>EUDI Wallet</h2><p>Activarea este disponibilă numai pentru profilul curent. Nu există un endpoint de ștergere, deci interfața nu simulează unul.</p></div>
				{walletActive && <Tag value="Activ" severity="success" />}
				<div><Button onClick={() => void activateWallet()} disabled={saving || walletActive}>{walletActive ? 'EUDI Wallet activ' : 'Activează EUDI Wallet'}</Button></div>
			</div></Card.Content></Card.Body></Card.Root>

            <Card.Root><Card.Body><Card.Content>
                <div className="flex flex-col gap-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                        <div><h2>Chei de acces</h2><p>Cheile de acces sunt asociate exclusiv contului curent.</p></div>
                        <div className="flex gap-2">
                            <Button variant="outlined" severity="secondary" aria-label="Reîncarcă cheile de acces" onClick={() => void loadPasskeys()} disabled={loading}><Refresh /></Button>
                            <Button onClick={() => void enroll()} disabled={saving || loading || !ceremony}><Key />Adaugă cheie</Button>
                        </div>
                    </div>
                    {!ceremony && <Message.Root severity="info"><Message.Content><Message.Text>Înrolarea va fi activată după conectarea ceremoniei WebAuthn; nu este simulată.</Message.Text></Message.Content></Message.Root>}
                    {loading ? <div className="flex justify-center p-6"><Spinner /></div> : <PasskeyTable passkeys={passkeys} />}
                </div>
            </Card.Content></Card.Body></Card.Root>
        </section>
    );
}

function PasskeyTable({ passkeys }: { passkeys: PasskeyCredential[] }) {
    if (passkeys.length === 0) {
        return <Message.Root severity="info"><Message.Content><Message.Text>Nu aveți chei de acces înregistrate.</Message.Text></Message.Content></Message.Root>;
    }
    return <DataTable.Root data={passkeys as unknown as Record<string, unknown>[]} dataKey="id">
        <DataTable.Table>
            <DataTable.THead><DataTable.THeadRow>
                <DataTable.THeadCell>Dispozitiv</DataTable.THeadCell>
                <DataTable.THeadCell>Creată</DataTable.THeadCell>
                <DataTable.THeadCell>Ultima utilizare</DataTable.THeadCell>
            </DataTable.THeadRow></DataTable.THead>
            <DataTable.TBody>{({ item, index }) => {
                const passkey = item as unknown as PasskeyCredential;
                return <DataTable.Row key={passkey.id} index={index}>
                    <DataTable.Cell>{passkey.device_name}</DataTable.Cell>
                    <DataTable.Cell>{passkey.created_at}</DataTable.Cell>
                    <DataTable.Cell>{passkey.last_used_at || '—'}</DataTable.Cell>
                </DataTable.Row>;
            }}</DataTable.TBody>
        </DataTable.Table>
    </DataTable.Root>;
}

function readError(reason: unknown) {
    if (!(reason instanceof Error)) return 'Operația nu a reușit.';
    const known: Record<string, string> = {
        profile_name_required: 'Numele este obligatoriu.',
        profile_locale_invalid: 'Limba aleasă nu este acceptată.',
        passkey_disabled: 'Cheile de acces nu sunt active pentru această implementare.',
        passkey_origin_invalid: 'Cheia de acces nu poate fi înregistrată pentru această origine.',
        passkey_challenge_expired: 'Cererea de înregistrare a expirat. Încercați din nou.'
    };
    return known[reason.message] ?? 'Operația nu a reușit. Încercați din nou.';
}
