package cloud.eguilde.dss;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.net.URI;
import java.nio.file.Path;
import java.time.Duration;
import java.util.Map;

record AdapterConfiguration(String bearerToken, String bindHost, int port, int httpWorkers,
                            URI s3Endpoint, String s3AccessKey, String s3SecretKey,
                            String allowedBucket, String reportBucket, int reportRetentionDays, long maxArtifactBytes, Path euLotlOjKeystore, String euLotlOjKeystoreType,
                            char[] euLotlOjKeystorePassword, String euLotlUrl, String euOjUrl,
                            Duration trustMaxAge, Map<String, CertificateBinding> certificateBindings) {
    static final String PROTOCOL = "egueducation-signed-artifact-verifier.v1";

    static AdapterConfiguration fromEnvironment() throws Exception {
        String token = require("DSS_ADAPTER_BEARER_TOKEN");
        String endpoint = require("DSS_ADAPTER_S3_ENDPOINT");
        URI endpointUri = URI.create(endpoint);
        if (!"https".equalsIgnoreCase(endpointUri.getScheme()) && !"localhost".equalsIgnoreCase(endpointUri.getHost()) && !"127.0.0.1".equals(endpointUri.getHost())) throw new IllegalArgumentException("DSS_ADAPTER_S3_ENDPOINT must use HTTPS outside local development");
        String bindings = require("DSS_ADAPTER_CERTIFICATE_BINDINGS_JSON");
        ObjectMapper mapper = Json.MAPPER;
        Map<String, CertificateBinding> parsed = mapper.readValue(bindings, new TypeReference<>() {});
        if (parsed.isEmpty()) throw new IllegalArgumentException("DSS_ADAPTER_CERTIFICATE_BINDINGS_JSON must not be empty");
        parsed.forEach((fingerprint, binding) -> {
            if (!Hex.sha256(fingerprint) || binding == null || binding.tenantCode().isBlank() || binding.institutionId().isBlank() || binding.actorSubject().isBlank())
                throw new IllegalArgumentException("certificate bindings must have SHA-256 fingerprints and a complete scope");
        });
        return new AdapterConfiguration(token, env("DSS_ADAPTER_BIND_HOST", "0.0.0.0"), integer("DSS_ADAPTER_PORT", 8080, 1, 65535),
            integer("DSS_ADAPTER_HTTP_WORKERS", 8, 1, 64), endpointUri, require("DSS_ADAPTER_S3_ACCESS_KEY"), require("DSS_ADAPTER_S3_SECRET_KEY"),
            require("DSS_ADAPTER_ALLOWED_BUCKET"), require("DSS_ADAPTER_REPORT_BUCKET"), integer("DSS_ADAPTER_REPORT_RETENTION_DAYS", 3650, 365, 36500), longValue("DSS_ADAPTER_MAX_ARTIFACT_BYTES", 52_428_800L, 1_024L, 1_073_741_824L), Path.of(require("DSS_ADAPTER_EU_LOTL_OJ_KEYSTORE")),
            env("DSS_ADAPTER_EU_LOTL_OJ_KEYSTORE_TYPE", "JKS"), require("DSS_ADAPTER_EU_LOTL_OJ_KEYSTORE_PASSWORD").toCharArray(),
            env("DSS_ADAPTER_EU_LOTL_URL", "https://ec.europa.eu/tools/lotl/eu-lotl.xml"),
            env("DSS_ADAPTER_EU_OJ_URL", "https://eur-lex.europa.eu/"), Duration.ofHours(integer("DSS_ADAPTER_TRUST_MAX_AGE_HOURS", 24, 1, 168)), Map.copyOf(parsed));
    }
    private static String require(String name) { String value = System.getenv(name); if (value == null || value.isBlank()) throw new IllegalArgumentException(name + " is required"); return value.trim(); }
    private static String env(String name, String fallback) { String value = System.getenv(name); return value == null || value.isBlank() ? fallback : value.trim(); }
    private static int integer(String name, int fallback, int min, int max) { try { int value = Integer.parseInt(env(name, String.valueOf(fallback))); if (value < min || value > max) throw new NumberFormatException(); return value; } catch (NumberFormatException error) { throw new IllegalArgumentException(name + " is outside allowed range", error); } }
    private static long longValue(String name, long fallback, long min, long max) { try { long value = Long.parseLong(env(name, String.valueOf(fallback))); if (value < min || value > max) throw new NumberFormatException(); return value; } catch (NumberFormatException error) { throw new IllegalArgumentException(name + " is outside allowed range", error); } }
}

record CertificateBinding(String tenantCode, String institutionId, String actorSubject) { }
