export interface ProfileUser {
  id: string;
  name: string;
  email: string;
  email_verified: boolean;
  phone_number: string;
  phone_number_verified: boolean;
  locale: "ro" | "en";
}

export interface PasskeyCredential {
  id: string;
  credential_id: string;
  device_name: string;
  created_at: string;
  last_used_at?: string;
}

export interface PasskeyRegistrationOptions {
  challenge: string;
  rp: { id: string; name: string };
  user: { id: string; name: string; displayName: string };
  pubKeyCredParams: Array<{ type: 'public-key'; alg: number }>;
  timeout: number;
  attestation: AttestationConveyancePreference;
}

export interface PasskeyRegistrationResult {
  credential_id: string;
  device_name?: string;
  challenge: string;
  response: Record<string, unknown>;
}

export interface PasskeyCeremony {
  register(options: PasskeyRegistrationOptions): Promise<PasskeyRegistrationResult>;
}

export interface ProfileApi {
  update(input: Pick<ProfileUser, "name" | "phone_number" | "locale">): Promise<ProfileUser>;
  passkeys(): Promise<PasskeyCredential[]>;
  registrationOptions(): Promise<PasskeyRegistrationOptions>;
  finishRegistration(input: PasskeyRegistrationResult): Promise<PasskeyCredential>;
	/** Activates the current user's EUDI Wallet integration; no delete endpoint exists. */
	activateEUDIWallet(): Promise<{ status: string }>;
}
