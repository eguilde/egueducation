import { afterEach, describe, expect, it, vi } from 'vitest';
import { browserPasskeyCeremony } from './webauthn';

const originalCredentials = Object.getOwnPropertyDescriptor(navigator, 'credentials');

describe('browser passkey ceremony', () => {
    afterEach(() => {
        vi.unstubAllGlobals();
        if (originalCredentials) Object.defineProperty(navigator, 'credentials', originalCredentials);
        else Reflect.deleteProperty(navigator, 'credentials');
    });

    it('converts WebAuthn binary values to the backend base64url contract', async () => {
        class FakeAttestationResponse {
            clientDataJSON = Uint8Array.from([1, 2]).buffer;
            attestationObject = Uint8Array.from([3, 4]).buffer;
            getTransports = () => ['internal'];
        }
        class FakeCredential {
            rawId = Uint8Array.from([5, 6]).buffer;
            type = 'public-key';
            response = new FakeAttestationResponse();
        }
        vi.stubGlobal('PublicKeyCredential', FakeCredential);
        vi.stubGlobal('AuthenticatorAttestationResponse', FakeAttestationResponse);
        Object.defineProperty(navigator, 'credentials', {
            configurable: true,
            value: { create: vi.fn(async () => new FakeCredential()) }
        });

        const result = await browserPasskeyCeremony.register({
            challenge: 'AQI',
            rp: { id: 'example.test', name: 'Test' },
            user: { id: 'AwQ', name: 'ana@example.test', displayName: 'Ana' },
            pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
            timeout: 60_000,
            attestation: 'none'
        });

        expect(result).toMatchObject({
            credential_id: 'BQY',
            challenge: 'AQI',
            response: {
                type: 'public-key',
                clientDataJSON: 'AQI',
                attestationObject: 'AwQ',
                transports: ['internal']
            }
        });
    });
});
