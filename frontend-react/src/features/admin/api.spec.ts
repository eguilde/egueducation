import { describe, expect, it, vi } from "vitest";
import { createAdminApi } from "./api";

describe("admin API adapter", () => {
  it("sends a tenant-safe user query and normalizes paged users", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [{ id: "u1", name: "Ana" }], total: 1, page: 1, pageSize: 50 }), { status: 200 }));
    const result = await createAdminApi(fetcher).users("ana@example.test");
    expect(fetcher.mock.calls[0][0]).toContain("/api/admin/users?page=1&pageSize=50&filter.email=ana%40example.test");
    expect(result.items[0]?.name).toBe("Ana");
  });

  it("surfaces server codes instead of treating shared identities as local", async () => {
    const api = createAdminApi(vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "shared_identity_platform_admin_required" }), { status: 403 })));
    await expect(api.saveUser({ name: "Ana", email: "ana@example.test", phone: "", locale: "ro", status: "active", email_verified: false, phone_verified: false, preferred_otp_channel: "sms" })).rejects.toThrow("shared_identity_platform_admin_required");
  });

  it("reads GDPR/admin catalog resources through the authenticated transport without tenant headers", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify([{ id: "r1", name: "Politică" }]), { status: 200 }));
    const result = await createAdminApi(fetcher).resource("gdpr/retention-policies");
    expect(fetcher.mock.calls[0][0]).toBe("/api/gdpr/retention-policies");
    expect(fetcher.mock.calls[0][1].headers).not.toHaveProperty("X-Institution-ID");
    expect(result.items[0]?.name).toBe("Politică");
  });
});
