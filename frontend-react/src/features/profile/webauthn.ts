import type { PasskeyCeremony, PasskeyRegistrationOptions, PasskeyRegistrationResult } from './types';

const decodeBase64Url = (value: string): Uint8Array<ArrayBuffer> => {
    const padding = '='.repeat((4 - value.length % 4) % 4);
    const binary = atob(value.replace(/-/g, '+').replace(/_/g, '/') + padding);
    return Uint8Array.from(binary, (character) => character.charCodeAt(0));
};

const encodeBase64Url = (value: ArrayBuffer): string => {
    const bytes = new Uint8Array(value);
    let binary = '';
    for (const byte of bytes) binary += String.fromCharCode(byte);
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
};

export const supportsWebAuthn = () => (
    typeof window !== 'undefined' &&
    typeof window.PublicKeyCredential !== 'undefined' &&
    typeof navigator.credentials?.create === 'function'
);

export const browserPasskeyCeremony: PasskeyCeremony = {
    async register(options: PasskeyRegistrationOptions): Promise<PasskeyRegistrationResult> {
        if (!supportsWebAuthn()) throw new Error('passkey_browser_unsupported');
        const publicKey: PublicKeyCredentialCreationOptions = {
            challenge: decodeBase64Url(options.challenge),
            rp: options.rp,
            user: {
                id: decodeBase64Url(options.user.id),
                name: options.user.name,
                displayName: options.user.displayName
            },
            pubKeyCredParams: options.pubKeyCredParams,
            timeout: options.timeout,
            attestation: options.attestation
        };
        const credential = await navigator.credentials.create({ publicKey });
        if (!(credential instanceof PublicKeyCredential) || !(credential.response instanceof AuthenticatorAttestationResponse)) {
            throw new Error('passkey_response_invalid');
        }
        return {
            credential_id: encodeBase64Url(credential.rawId),
            device_name: 'Cheie de acces browser',
            challenge: options.challenge,
            response: {
                type: credential.type,
                clientDataJSON: encodeBase64Url(credential.response.clientDataJSON),
                attestationObject: encodeBase64Url(credential.response.attestationObject),
                transports: credential.response.getTransports?.() ?? []
            }
        };
    }
};
