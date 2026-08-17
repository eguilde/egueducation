import { expect, test } from '@playwright/test';

const origin = 'http://127.0.0.1:4173';

async function authenticated(page: import('@playwright/test').Page, permissions: string[]) {
	await page.route('**/api/oidc/.well-known/openid-configuration', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ issuer: `${origin}/api/oidc`, authorization_endpoint: `${origin}/api/oidc/authorize`, token_endpoint: `${origin}/api/oidc/token`, jwks_uri: `${origin}/api/oidc/jwks`, response_types_supported: ['code'], subject_types_supported: ['public'], id_token_signing_alg_values_supported: ['RS256'], grant_types_supported: ['authorization_code', 'refresh_token'], code_challenge_methods_supported: ['S256'] }) }));
	await page.route('**/api/oidc/token', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ access_token: 'test-access-token', token_type: 'Bearer', expires_in: 3600 }) }));
	await page.route('**/api/me', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ user: { id: 'user-1', sub: 'subject-1', name: 'Ana', email: 'ana@example.test', email_verified: true, phone_number: '', phone_number_verified: false, preferred_otp_channel: 'sms', locale: 'ro', roles: [] }, institution_id: 'inst-1', institution_name: 'Școala Test', permissions, modules: [{ code: 'admin', active: true }], authentication: ['sms'], gdpr_capabilities: [] }) }));
}

test('admin hides a mutation without its exact manage permission', async ({ page }) => {
	await authenticated(page, ['admin.read', 'admin.roles.read']);
	await page.route('**/api/admin/dashboard', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ stats: {}, modules: [], admin_sections: [], warnings: [] }) }));
	await page.route('**/api/admin/roles**', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }) }));
	await page.goto('/'); await expect(page.getByText('Ana')).toBeVisible();
	await page.goto('/administrare');
	await expect(page.getByRole('button', { name: 'Adaugă sau actualizează' })).toHaveCount(0);
});

test('mobile admin sends the exact role DTO only after its manage permission', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await authenticated(page, ['admin.read', 'admin.roles.read', 'admin.roles.manage']);
	await page.route('**/api/admin/dashboard', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ stats: {}, modules: [], admin_sections: [], warnings: [] }) }));
	await page.route('**/api/admin/roles**', async (route) => {
		if (route.request().method() === 'POST') {
			expect(route.request().postDataJSON()).toEqual({ code: 'reviewer', label: 'Reviewer' });
			await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ code: 'reviewer', label: 'Reviewer' }) }); return;
		}
		await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ items: [], total: 0, page: 1, pageSize: 50 }) });
	});
	await page.goto('/');
	await page.getByRole('button', { name: 'Deschide navigația' }).click();
	await expect(page.getByText('Ana')).toBeVisible();
	await page.goto('/administrare');
	await page.getByRole('button', { name: 'Adaugă sau actualizează' }).click();
	await page.getByLabel('Cod').fill('reviewer');
	await page.getByLabel('Etichetă').fill('Reviewer');
	await page.getByRole('button', { name: 'Salvează' }).click();
	await expect(page.getByRole('dialog', { name: /Configurare instituție/ })).not.toBeVisible();
});

test('profile activates EUDI Wallet through its supported endpoint on mobile', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await authenticated(page, []);
	await page.route('**/api/passkeys', (route) => route.fulfill({ contentType: 'application/json', body: '[]' }));
	await page.route('**/api/eudi-wallet/activate', async (route) => {
		expect(route.request().method()).toBe('POST');
		expect(route.request().postDataJSON()).toEqual({});
		await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ status: 'active' }) });
	});
	await page.goto('/profil');
	await expect(page.getByRole('heading', { name: 'Profil' })).toBeVisible();
	await page.getByRole('button', { name: 'Activează EUDI Wallet' }).click();
	await expect(page.getByText('EUDI Wallet este activ pentru profilul curent.')).toBeVisible();
});
