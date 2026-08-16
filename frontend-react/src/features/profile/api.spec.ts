import { describe, expect, it, vi } from "vitest";
import { createProfileApi } from "./api";

describe("profile API adapter", () => {
  it("updates only fields accepted by the current-profile contract", async () => {
    const profile = { id: "u1", name: "Ana", email: "ana@example.test", email_verified: true, phone_number: "", phone_number_verified: false, locale: "ro" };
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ user: profile, institution_id: "inst-1" }), { status: 200 }));
    await expect(createProfileApi(fetcher).update({ name: "Ana", phone_number: "", locale: "ro" })).resolves.toEqual(profile);
    expect(fetcher.mock.calls[0][0]).toBe("/api/profile");
    expect(JSON.parse(fetcher.mock.calls[0][1].body)).toEqual({ name: "Ana", phone_number: "", locale: "ro" });
  });

  it("keeps the browser WebAuthn ceremony outside the transport adapter", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ challenge: "challenge" }), { status: 200 }));
    await expect(createProfileApi(fetcher).registrationOptions()).resolves.toEqual({ challenge: "challenge" });
    expect(fetcher.mock.calls[0][0]).toBe("/api/passkeys/register-options");
  });

	it("activates EUDI Wallet only through the supported current-user endpoint", async () => {
		const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "active" }), { status: 200 }));
		await expect(createProfileApi(fetcher).activateEUDIWallet()).resolves.toEqual({ status: "active" });
		expect(fetcher.mock.calls[0][0]).toBe("/api/eudi-wallet/activate");
		expect(fetcher.mock.calls[0][1].method).toBe("POST");
	});
});
