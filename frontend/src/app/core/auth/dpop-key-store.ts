const KEY_NAME = 'egueducation-dpop-es256';

export async function loadOrCreateKeyPair(): Promise<CryptoKeyPair> {
  const stored = localStorage.getItem(KEY_NAME);
  if (stored) {
    const parsed = JSON.parse(stored) as { privateKey: JsonWebKey; publicKey: JsonWebKey };
    return {
      privateKey: await crypto.subtle.importKey(
        'jwk',
        parsed.privateKey,
        { name: 'ECDSA', namedCurve: 'P-256' },
        false,
        ['sign'],
      ),
      publicKey: await crypto.subtle.importKey(
        'jwk',
        parsed.publicKey,
        { name: 'ECDSA', namedCurve: 'P-256' },
        true,
        ['verify'],
      ),
    };
  }

  const keyPair = await crypto.subtle.generateKey(
    { name: 'ECDSA', namedCurve: 'P-256' },
    true,
    ['sign', 'verify'],
  );

  const privateKey = await crypto.subtle.exportKey('jwk', keyPair.privateKey);
  const publicKey = await crypto.subtle.exportKey('jwk', keyPair.publicKey);
  localStorage.setItem(KEY_NAME, JSON.stringify({ privateKey, publicKey }));
  return keyPair;
}
