import type { AdmissionApi } from "./api";
import { AdmissionVertical } from "./AdmissionVertical";

export type AdmissionCapabilities = {
  read: boolean;
  piiRead: boolean;
  manage: boolean;
  contextManage: boolean;
  retentionManage: boolean;
  retentionApprove: boolean;
  decide: boolean;
  appealsManage: boolean;
  /** Certificate trust administration is deliberately split: proposal/revocation and approval are separate rights. */
  signerAuthorizationManage?: boolean;
  signerAuthorizationApprove?: boolean;
};

/** The route shell injects an authenticated, generated-contract adapter. */
export function AdmissionWorkspace({ api, capabilities }: { api: AdmissionApi; capabilities: AdmissionCapabilities }) {
  return <AdmissionVertical api={api} capabilities={capabilities} />;
}
