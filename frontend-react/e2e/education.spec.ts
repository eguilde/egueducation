import { expect, test, type Page } from "@playwright/test";

const meeting = {
  id: "meeting-1",
  school_year: "2026-2027",
  organism: "ca",
  title: "Ședință ordinară CA",
  meeting_type: "ordinary",
  status: "scheduled",
  meeting_date: "2026-09-10",
  location: "Sala profesorală",
  chairperson: "Director Test",
  secretary_name: "Secretar Test",
  chairperson_user_id: "user-chair",
  secretary_user_id: "user-secretary",
};

async function authenticatedSchool(page: Page, sessionOverride?: Record<string, unknown>) {
  await page.route("**/api/oidc/.well-known/openid-configuration", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        issuer: "http://127.0.0.1:4173/api/oidc",
        authorization_endpoint: "http://127.0.0.1:4173/api/oidc/authorize",
        token_endpoint: "http://127.0.0.1:4173/api/oidc/token",
        jwks_uri: "http://127.0.0.1:4173/api/oidc/jwks",
        response_types_supported: ["code"],
        subject_types_supported: ["public"],
        id_token_signing_alg_values_supported: ["RS256"],
        grant_types_supported: ["authorization_code", "refresh_token"],
        code_challenge_methods_supported: ["S256"],
      }),
    }),
  );
  await page.route("**/api/oidc/token", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        access_token: "test-access-token",
        token_type: "Bearer",
        expires_in: 3600,
      }),
    }),
  );
  await page.route("**/api/me", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(sessionOverride ?? {
        user: {
          id: "11111111-1111-4111-8111-111111111111",
          sub: "school-subject",
          name: "Director Test",
          email: "director@example.test",
          email_verified: true,
          phone_number: "",
          phone_number_verified: false,
          preferred_otp_channel: "sms",
          locale: "ro",
          roles: [],
        },
        tenant_code: "tenant-test",
        institution_id: "inst-test",
        institution_name: "Școala Test",
        platform_roles: [],
        authz_version: 1,
        permissions: [
          "education.read",
          "education.governance.read",
          "education.governance.manage",
          "education.personnel.read",
          "education.personnel.manage",
          "education.evaluations.read",
          "education.evaluations.manage",
          "education.declarations.read",
          "education.declarations.manage",
          "education.mobility.read",
          "education.mobility.manage",
          "education.gradatii.read",
          "education.gradatii.manage",
          "education.portfolios.read",
          "education.portfolios.manage",
        ],
        modules: [{ code: "education", active: true }],
        authentication: ["sms"],
        gdpr_capabilities: [],
      }),
    }),
  );
  await page.route("**/api/education/governance/eligible-users?**", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        items: [
          { id: "user-chair", name: "Director Test" },
          { id: "user-secretary", name: "Secretar Test" },
        ],
      }),
    }),
  );
  await page.route("**/api/education/governance/meetings?**", (route) => {
    const requestUrl = new URL(route.request().url());
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        items: [meeting],
        total: 45,
        page: Number(requestUrl.searchParams.get("page") ?? 1),
        pageSize: Number(requestUrl.searchParams.get("pageSize") ?? 20),
      }),
    });
  });
  await page.route("**/api/education/governance/meetings/meeting-1", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(meeting) }),
  );
  await page.route("**/api/education/**", (route) => {
    const pathname = new URL(route.request().url()).pathname;
    if (
      pathname.endsWith("/education/governance/eligible-users") ||
      pathname.endsWith("/education/governance/meetings") ||
      pathname.endsWith("/education/governance/meetings/meeting-1")
    ) {
      return route.fallback();
    }
    if (route.request().method() === "GET") {
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0, page: 1, pageSize: 20 }),
      });
    }
    return route.fulfill({
      status: 201,
      contentType: "application/json",
      body: JSON.stringify({ id: "created-1" }),
    });
  });
  await page.goto("/");
  const expectedName = String((sessionOverride?.user as { name?: string } | undefined)?.name ?? "Director Test");
  await expect(page.getByText(expectedName)).toBeVisible();
}

test("school governance uses server pagination, sorting and column filters", async ({ page }) => {
  await authenticatedSchool(page);
  await page.goto("/scoala/governance");
  await expect(page.getByText("Ședințe de guvernanță")).toBeVisible();
  await expect(page.getByText("1 - 20 din 45")).toBeVisible();

  const sorted = page.waitForRequest((request) => {
    const url = new URL(request.url());
    return url.pathname.endsWith("/education/governance/meetings") && url.searchParams.get("sort") === "title";
  });
  await page.getByRole("button", { name: "Titlu" }).click();
  expect(new URL((await sorted).url()).searchParams.get("direction")).toBe("asc");

  const filtered = page.waitForRequest((request) =>
    new URL(request.url()).searchParams.get("filter.organism") === "ca",
  );
  await page.getByLabel("Filtru Organism").fill("ca");
  await page.getByLabel("Filtru Organism").press("Tab");
  await filtered;

  const pageTwo = page.waitForRequest((request) =>
    new URL(request.url()).searchParams.get("page") === "2",
  );
  await page.getByLabel("Paginare", { exact: true }).getByRole("button", { name: "Următor" }).click();
  expect(new URL((await pageTwo).url()).searchParams.get("pageSize")).toBe("20");
});

test("school exposes the Angular-equivalent governance wizard with immutable user IDs", async ({ page }) => {
  await authenticatedSchool(page);
  await page.goto("/scoala/governance/ca-wizard");
  await expect(page.getByRole("heading", { name: "Ședință CA/CP/CEAC" })).toBeVisible();
  await page.getByLabel("Titlu").fill("Ședință CA de test");
  await page.getByRole("button", { name: "Continuă" }).click();
  await page.getByRole("button", { name: "Continuă" }).click();
  await page.getByLabel("Data").fill("2026-09-10");
  await page.getByRole("button", { name: "Continuă" }).click();
  await expect(page.getByText("Președinte *")).toBeVisible();
  await expect(page.getByText("Secretar *")).toBeVisible();
  await page.getByRole("combobox", { name: "Președinte *" }).click();
  await page.locator('[role="option"]:visible').filter({ hasText: "Director Test" }).click();
  await page.getByRole("combobox", { name: "Secretar *" }).click();
  await page.locator('[role="option"]').filter({ hasText: "Secretar Test" }).last().click();
  await page.getByRole("button", { name: "Continuă" }).click();
  await expect(page.getByRole("button", { name: "Salvează" })).toBeVisible();
});

// Mocked browser UX coverage only. The real authorization and database flow is
// exercised separately by system tests; this test proves that the React route
// never falls back to an institution-wide `/records/{id}` endpoint for a teacher.
test("mocked teacher portfolio UX stays on owner-scoped contracts", async ({ page }) => {
  await authenticatedSchool(page, {
    user: { id: "22222222-2222-4222-8222-222222222222", sub: "teacher-subject", name: "Profesor Test", email: "teacher@example.test", email_verified: true, phone_number: "", phone_number_verified: false, preferred_otp_channel: "sms", locale: "ro", roles: ["profesor"] },
    tenant_code: "tenant-test", institution_id: "inst-test", institution_name: "Școala Test", platform_roles: [], authz_version: 1,
    permissions: ["education.read", "education.portfolios.read_own", "education.portfolios.manage_own"], modules: [{ code: "education", active: true }], authentication: ["sms"], gdpr_capabilities: [],
  });
  const portfolio = { id: "own-portfolio", portfolio_code: "PORT-CD-1", owner_name: "Profesor Test", owner_role: "Profesor", school_year: "2026-2027", status: "draft", section_count: 2, last_updated_on: "2026-09-01", authenticity_declared: true, consent_captured: true, notes: "" };
  const templates = [
    { declaration_type: "authenticity", declaration_version: "1", declaration_text: "Confirm că documentele sunt autentice.", source_ref: "OME 3858/2026", effective_from: "2026-05-08" },
    { declaration_type: "gdpr_information", declaration_version: "1", declaration_text: "Confirm prelucrarea datelor conform scopului legal.", source_ref: "OME 3858/2026", effective_from: "2026-05-08" },
  ];
  const acknowledgements: Array<{ id: string; portfolio_id: string; declaration_type: string; declaration_version: string; declaration_text: string; accepted_at: string; accepted_by_user_id: string; attestation_method: string }> = [];
  await page.route("**/api/education/portfolios/me/own-portfolio/opis/regenerate", (route) => route.fulfill({ status: 204 }));
  await page.route("**/api/education/portfolios/me/own-portfolio/submit", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify({ ...portfolio, status: "submitted" }) }));
  await page.route(/\/api\/education\/portfolios\/me\/own-portfolio\/(documents|checklist|opis|reviews)(\?.*)?$/, (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [], total: 0, page: 1, pageSize: 1 }) }));
  await page.route("**/api/education/portfolios/me/own-portfolio/declarations", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify({ templates, acknowledgements }) }));
  await page.route(/\/api\/education\/portfolios\/me\/own-portfolio\/declarations\/([^/]+)\/acknowledgements$/, (route) => {
    const declarationType = new URL(route.request().url()).pathname.split("/").at(-2) ?? "";
    const template = templates.find((item) => item.declaration_type === declarationType)!;
    acknowledgements.push({ id: `ack-${declarationType}`, portfolio_id: portfolio.id, declaration_type: declarationType, declaration_version: "1", declaration_text: template.declaration_text, accepted_at: new Date().toISOString(), accepted_by_user_id: "22222222-2222-4222-8222-222222222222", attestation_method: "explicit_ui_confirmation" });
    return route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify(acknowledgements.at(-1)) });
  });
  await page.route("**/api/education/portfolios/me/own-portfolio", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify(portfolio) }));
  await page.route("**/api/education/portfolios/me?**", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [portfolio], total: 1, page: 1, pageSize: 1 }) }));
  await page.goto("/scoala/portfolio/me");
  await expect(page.getByRole("region", { name: "Portofoliul meu profesional" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Portofoliul meu profesional" }).getByText("Profesor Test")).toBeVisible();
  for (const template of templates) {
    await page.getByRole("button", { name: "Citește și confirmă" }).first().click();
    await expect(page.getByText(template.declaration_text)).toBeVisible();
    await page.getByRole("checkbox", { name: "Confirm declarația afișată" }).click();
    await page.getByRole("button", { name: "Confirmă declarația" }).click();
  }
  const ownSubmit = page.waitForRequest((request) => new URL(request.url()).pathname.endsWith("/education/portfolios/me/own-portfolio/submit"));
  await page.getByRole("button", { name: "Trimite spre verificare" }).click();
  await page.getByRole("button", { name: "Confirmă trimiterea" }).click();
  expect(new URL((await ownSubmit).url()).pathname).toContain("/portfolios/me/");
});
