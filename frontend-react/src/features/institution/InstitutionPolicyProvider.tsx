import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from "react";
import { createContractClient } from "../../api/client";
import { useAuth } from "../../auth/AuthProvider";
import { createInstitutionPolicyApi, type InstitutionCapabilities } from "./api";

interface InstitutionPolicyContextValue {
  ready: boolean;
  blocked: boolean;
  capabilities: InstitutionCapabilities | null;
  canPolicy(code: string): boolean;
  refresh(): Promise<void>;
}

const InstitutionPolicyContext = createContext<InstitutionPolicyContextValue>({ ready: true, blocked: true, capabilities: null, canPolicy: () => false, refresh: async () => undefined });

export function InstitutionPolicyProvider({ children }: PropsWithChildren) {
  const { apiFetch, session } = useAuth();
  const [ready, setReady] = useState(!session);
  const [capabilities, setCapabilities] = useState<InstitutionCapabilities | null>(null);
  const generation = useRef(0);
  const load = useCallback(async () => {
    const currentGeneration = ++generation.current;
    if (!session) { setCapabilities(null); setReady(true); return; }
    setReady(false); setCapabilities(null);
    try {
      const api = createInstitutionPolicyApi(createContractClient(apiFetch));
      const response = await api.capabilities();
      if (response.tenant_code !== session.tenant_code || response.institution_id !== session.institution_id) throw new Error("institution_policy_scope_mismatch");
      if (generation.current === currentGeneration) setCapabilities(response);
    } catch { if (generation.current === currentGeneration) setCapabilities(null); }
    finally { if (generation.current === currentGeneration) setReady(true); }
  }, [apiFetch, session]);
  useEffect(() => { void load(); return () => { generation.current += 1; }; }, [load]);
  useEffect(() => {
    if (!session || typeof window === "undefined") return;
    const refreshWhenVisible = () => { if (document.visibilityState === "visible") void load(); };
    window.addEventListener("focus", refreshWhenVisible);
    document.addEventListener("visibilitychange", refreshWhenVisible);
    const interval = window.setInterval(() => { if (document.visibilityState === "visible") void load(); }, 60_000);
    return () => {
      window.removeEventListener("focus", refreshWhenVisible);
      document.removeEventListener("visibilitychange", refreshWhenVisible);
      window.clearInterval(interval);
    };
  }, [load, session]);
  const enabled = useMemo(() => new Set(capabilities?.capabilities.filter((item) => item.enabled).map((item) => item.code) ?? []), [capabilities]);
  const value = useMemo<InstitutionPolicyContextValue>(() => ({ ready, blocked: capabilities?.blocked ?? true, capabilities, canPolicy: (code) => enabled.has(code), refresh: load }), [capabilities, enabled, load, ready]);
  return <InstitutionPolicyContext.Provider value={value}>{children}</InstitutionPolicyContext.Provider>;
}

export function useInstitutionPolicy() { return useContext(InstitutionPolicyContext); }
