package cloud.eguilde.dss;

import eu.europa.esig.dss.spi.client.http.NativeHTTPDataLoader;
import eu.europa.esig.dss.spi.tsl.TrustedListsCertificateSource;
import eu.europa.esig.dss.spi.validation.CommonCertificateVerifier;
import eu.europa.esig.dss.spi.x509.KeyStoreCertificateSource;
import eu.europa.esig.dss.spi.x509.aia.DefaultAIASource;
import eu.europa.esig.dss.service.crl.OnlineCRLSource;
import eu.europa.esig.dss.service.ocsp.OnlineOCSPSource;
import eu.europa.esig.dss.tsl.function.OfficialJournalSchemeInformationURI;
import eu.europa.esig.dss.tsl.job.TLValidationJob;
import eu.europa.esig.dss.tsl.source.LOTLSource;
import java.time.Instant;
import java.util.List;

/**
 * Loads the EU LOTL using an operator-supplied Official Journal anchor. A
 * keystore path is mandatory precisely so a deployment cannot silently trust
 * an arbitrary downloaded LOTL signing key. A failed refresh leaves this
 * service unavailable rather than retaining an unboundedly stale trust list.
 */
final class TrustListRuntime {
    private final AdapterConfiguration config;
    private volatile CommonCertificateVerifier certificateVerifier;
    private volatile Instant refreshedAt;
    private volatile String failure = "trust_list_not_loaded";
    TrustListRuntime(AdapterConfiguration config) { this.config = config; }
    synchronized void refresh() {
        try {
            KeyStoreCertificateSource officialJournal = new KeyStoreCertificateSource(config.euLotlOjKeystore().toFile(), config.euLotlOjKeystoreType(), config.euLotlOjKeystorePassword());
            LOTLSource europeanLotl = new LOTLSource();
            europeanLotl.setUrl(config.euLotlUrl());
            europeanLotl.setCertificateSource(officialJournal);
            europeanLotl.setSigningCertificatesAnnouncementPredicate(new OfficialJournalSchemeInformationURI(config.euOjUrl()));
            europeanLotl.setPivotSupport(true);
            europeanLotl.setTLVersions(List.of(5, 6));
            NativeHTTPDataLoader loader = new NativeHTTPDataLoader();
            loader.setConnectTimeout(15_000);
            loader.setReadTimeout(45_000);
            loader.setMaxInputSize(25 * 1024 * 1024);
            TrustedListsCertificateSource trusted = new TrustedListsCertificateSource();
            TLValidationJob job = new TLValidationJob();
            NativeDssFileLoader fileLoader = new NativeDssFileLoader(loader);
            job.setOfflineDataLoader(fileLoader);
            job.setOnlineDataLoader(fileLoader);
            job.setTrustedListCertificateSource(trusted);
            job.setListOfTrustedListSources(europeanLotl);
            job.onlineRefresh();
            if (trusted.getNumberOfTrustedEntityKeys() < 1) throw new IllegalStateException("EU LOTL refresh returned no trusted entities");
            CommonCertificateVerifier verifier = new CommonCertificateVerifier();
            verifier.setTrustedCertSources(trusted);
            // Revocation checking is not optional: a chain anchored in a TL is
            // still rejected by the DSS policy when OCSP/CRL evidence is not
            // available or validates negatively.
            verifier.setOcspSource(new OnlineOCSPSource(loader));
            verifier.setCrlSource(new OnlineCRLSource(loader));
            verifier.setAIASource(new DefaultAIASource(loader));
            this.certificateVerifier = verifier;
            this.refreshedAt = Instant.now();
            this.failure = "";
        } catch (Exception error) {
            this.certificateVerifier = null;
            this.refreshedAt = null;
            this.failure = error.getClass().getSimpleName();
        }
    }
    boolean ready() { return certificateVerifier != null && refreshedAt != null && refreshedAt.plus(config.trustMaxAge()).isAfter(Instant.now()); }
    CommonCertificateVerifier certificateVerifier() { if (!ready()) throw new IllegalStateException("trust_list_not_ready"); return certificateVerifier; }
    String failure() { return failure; }
}
