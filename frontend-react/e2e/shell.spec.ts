import { test, expect } from '@playwright/test';

test('desktop shell keeps the left navigation open, hides bars, and shows the tenant title', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.route('**/api/config', (route) => route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ institutionName: 'Școala Gimnazială nr. 1 Balotești' })
    }));
    await page.goto('/');

    await expect(page.getByRole('heading', { name: 'eGuEducation' })).toBeVisible();
    await expect(page.getByText(/Platformă digitală pentru registratură/)).toBeVisible();
    await expect(page.locator('.landing').getByRole('button', { name: 'Autentificare' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Școala Gimnazială nr. 1 Balotești' })).toBeVisible();
    const navigation = page.locator('#main-navigation');
    await expect(navigation).toHaveAttribute('data-side', 'left');
    await expect(navigation).toHaveAttribute('data-state', 'expanded');
    await expect(page.locator('header').getByRole('button', { name: /navigația/i })).toHaveCount(0);
});

test('mobile shell opens and closes the left off-canvas navigation from bars controls', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/');

    const navigation = page.locator('#main-navigation');
    await expect(navigation).toHaveAttribute('data-side', 'left');
    await expect(navigation).toHaveAttribute('data-collapsible-mode', 'offcanvas');
    await expect(navigation).toHaveAttribute('data-state', 'collapsed');
    await page.locator('header').getByRole('button', { name: 'Deschide navigația' }).click();
    await expect(navigation).toHaveAttribute('data-state', 'expanded');
    await expect(page.getByText('Componente')).toBeVisible();
    await page.getByRole('button', { name: 'Închide navigația' }).click();
    await expect(navigation).toHaveAttribute('data-state', 'collapsed');
});

test('theme popover changes and persists the PrimeReact color scheme', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'Tema aplicației' }).click();
    await expect(page.getByRole('group', { name: 'Mod de culoare' })).toBeVisible();
    await page.getByRole('button', { name: 'Întunecat' }).click();
    await expect(page.locator('html')).toHaveClass(/app-dark/);
    await expect.poll(() => page.evaluate(() => localStorage.getItem('egueducation.scheme'))).toBe('dark');
    await page.reload();
    await expect(page.locator('html')).toHaveClass(/app-dark/);
});

test('OIDC registration hand-off explains audited administrator provisioning', async ({ page }) => {
    await page.goto('/auth/register');
    await expect(page.getByRole('heading', { name: 'Acces eGuEducation' })).toBeVisible();
    await expect(page.getByText(/Auto-înregistrarea publică nu este disponibilă/)).toBeVisible();
    await expect(page.getByRole('button', { name: 'Autentificare' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Prima pagină' })).toBeVisible();
});

test('authenticated navigation requires both effective permission and an active module', async ({ page }) => {
    const origin = 'http://127.0.0.1:4173';
    await page.route('**/api/oidc/.well-known/openid-configuration', (route) => route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
            issuer: `${origin}/api/oidc`,
            authorization_endpoint: `${origin}/api/oidc/authorize`,
            token_endpoint: `${origin}/api/oidc/token`,
            jwks_uri: `${origin}/api/oidc/jwks`,
            response_types_supported: ['code'],
            subject_types_supported: ['public'],
            id_token_signing_alg_values_supported: ['RS256'],
            grant_types_supported: ['authorization_code', 'refresh_token'],
            code_challenge_methods_supported: ['S256']
        })
    }));
    await page.route('**/api/oidc/token', (route) => route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ access_token: 'test-access-token', token_type: 'Bearer', expires_in: 3600 })
    }));
    await page.route('**/api/me', (route) => route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
            user: {
                id: 'user-1', sub: 'subject-1', name: 'Ana', email: 'ana@example.test',
                email_verified: true, phone_number: '', phone_number_verified: false,
                preferred_otp_channel: 'sms', locale: 'ro', roles: ['super_admin']
            },
            institution_id: 'inst-1', institution_name: 'Școala Test',
            permissions: ['registratura.read', 'workflow.read'],
            modules: [{ code: 'registratura', active: false }, { code: 'workflow', active: true }],
            authentication: ['sms'], gdpr_capabilities: []
        })
    }));

    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto('/');

    await expect(page.getByText('Ana')).toBeVisible();
    await expect(page.getByRole('link', { name: 'Flux documente' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Registratură' })).toHaveCount(0);
    await expect(page.getByRole('link', { name: 'Administrare' })).toHaveCount(0);
});
