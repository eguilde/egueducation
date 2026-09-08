import { describe, expect, it, vi } from "vitest";
import { createProfileApi } from "./api";

const requestAt = (fetcher: ReturnType<typeof vi.fn>, index = 0) => fetcher.mock.calls[index][0] as Request;

describe("profile API adapter", () => {
  it("updates only fields accepted by the current-profile contract", async () => {
    const profile = { id: "u1", name: "Ana", email: "ana@example.test", email_verified: true, phone_number: "", phone_number_verified: false, locale: "ro" };
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ user: profile, institution_id: "inst-1" }), { status: 200 }));
    await expect(createProfileApi(fetcher).update({ name: "Ana", phone_number: "", locale: "ro" })).resolves.toEqual(profile);
    const request = requestAt(fetcher);
    expect(new URL(request.url).pathname).toBe("/api/profile");
    expect(request.credentials).toBe("include");
    expect(await request.json()).toEqual({ name: "Ana", phone_number: "", locale: "ro" });
  });

  it("keeps the browser WebAuthn ceremony outside the transport adapter", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ challenge: "challenge" }), { status: 200 }));
    await expect(createProfileApi(fetcher).registrationOptions()).resolves.toEqual({ challenge: "challenge" });
    const request = requestAt(fetcher);
    expect(new URL(request.url).pathname).toBe("/api/passkeys/register-options");
    expect(request.method).toBe("POST");
  });

	it("activates EUDI Wallet only through the supported current-user endpoint", async () => {
		const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: "active" }), { status: 200 }));
		await expect(createProfileApi(fetcher).activateEUDIWallet()).resolves.toEqual({ status: "active" });
    const request = requestAt(fetcher);
    expect(new URL(request.url).pathname).toBe("/api/eudi-wallet/activate");
		expect(request.method).toBe("POST");
	});
});
