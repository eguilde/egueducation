import { describe, expect, it, vi } from "vitest";
import { createAdminApi } from "./api";

const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "content-type": "application/json" } });

describe("admin API adapter", () => {
  it("sends a tenant-safe user query and normalizes paged users", async () => {
    const fetcher = vi.fn().mockResolvedValue(json({ items: [{ id: "u1", name: "Ana" }], total: 1, page: 1, pageSize: 50 }));
    const result = await createAdminApi(fetcher).users("ana@example.test");
    const request = fetcher.mock.calls[0][0] as Request;
    expect(request.url).toContain("/api/admin/users?page=1&pageSize=50&filter.email=ana%40example.test");
    expect(request.credentials).toBe("include");
    expect(request.headers.get("X-Institution-ID")).toBeNull();
    expect(result.items[0]?.name).toBe("Ana");
  });

  it("surfaces server codes instead of treating shared identities as local", async () => {
    const api = createAdminApi(vi.fn().mockResolvedValue(json({ code: "shared_identity_platform_admin_required" }, 403)));
    await expect(api.saveUser({ name: "Ana", email: "ana@example.test", phone: "", locale: "ro", status: "active", email_verified: false, phone_verified: false, preferred_otp_channel: "sms" })).rejects.toThrow("shared_identity_platform_admin_required");
  });

  it("reads GDPR/admin catalog resources through the authenticated transport without tenant headers", async () => {
    const fetcher = vi.fn().mockResolvedValue(json([{ id: "r1", name: "Politică" }]));
    const result = await createAdminApi(fetcher).resource("gdpr/retention-policies");
    const request = fetcher.mock.calls[0][0] as Request;
    expect(new URL(request.url).pathname).toBe("/api/gdpr/retention-policies");
    expect(request.headers.get("X-Institution-ID")).toBeNull();
    expect(result.items[0]?.name).toBe("Politică");
  });
});
