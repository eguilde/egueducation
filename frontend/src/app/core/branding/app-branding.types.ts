export interface AppBootstrapConfig {
  institutionId?: string;
  institutionName?: string;
  customer?: {
    name?: string;
    shortName?: string;
    websiteLabel?: string;
  };
  service?: {
    title?: string;
  };
}
