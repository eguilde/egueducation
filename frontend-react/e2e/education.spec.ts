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

async function authenticatedSchool(page: Page) {
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
      body: JSON.stringify({
        user: {
          id: "user-chair",
          sub: "school-subject",
          name: "Director Test",
          email: "director@example.test",
          roles: [],
        },
        institution_id: "inst-test",
        institution_name: "Școala Test",
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
  await page.route("**/api/education/governance/eligible-users", (route) =>
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
  await expect(page.getByText("Director Test")).toBeVisible();
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

  await page.getByLabel("Filtru Organism").fill("ca");
  const filtered = page.waitForRequest((request) =>
    new URL(request.url()).searchParams.get("filter.organism") === "ca",
  );
  await page.getByRole("button", { name: "Aplică filtre" }).click();
  await filtered;

  const pageTwo = page.waitForRequest((request) =>
    new URL(request.url()).searchParams.get("page") === "2",
  );
  await page.getByRole("button", { name: "Următor" }).click();
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
