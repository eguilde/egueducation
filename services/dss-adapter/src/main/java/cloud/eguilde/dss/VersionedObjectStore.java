package cloud.eguilde.dss;

import java.net.URI;
import java.time.Instant;
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials;
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider;
import software.amazon.awssdk.core.ResponseBytes;
import software.amazon.awssdk.core.sync.ResponseTransformer;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.GetObjectRequest;
import software.amazon.awssdk.services.s3.model.GetObjectResponse;
import software.amazon.awssdk.services.s3.model.HeadObjectRequest;
import software.amazon.awssdk.services.s3.model.ObjectLockMode;
import software.amazon.awssdk.services.s3.model.PutObjectRequest;
import software.amazon.awssdk.services.s3.model.PutObjectResponse;
import software.amazon.awssdk.core.sync.RequestBody;

/** Exact-version S3 reader. It never accepts a current-key read in place of the supplied immutable version. */
final class VersionedObjectStore implements AutoCloseable {
    private final S3Client s3;
    private final String bucket;
    private final String reportBucket;
    private final int reportRetentionDays;
    private final long configMaxArtifactBytes;
    VersionedObjectStore(AdapterConfiguration config) {
        this.bucket = config.allowedBucket();
        this.reportBucket = config.reportBucket();
        this.reportRetentionDays = config.reportRetentionDays();
        this.configMaxArtifactBytes = config.maxArtifactBytes();
        this.s3 = S3Client.builder().endpointOverride(config.s3Endpoint()).region(Region.US_EAST_1).forcePathStyle(true)
            .credentialsProvider(StaticCredentialsProvider.create(AwsBasicCredentials.create(config.s3AccessKey(), config.s3SecretKey()))).build();
    }
    Snapshot load(Evidence evidence) {
        if (!bucket.equals(evidence.storageBucket()) || blank(evidence.storageObjectKey()) || blank(evidence.storageObjectVersionId())) throw new SecurityException("storage_scope_invalid");
        var head = s3.headObject(HeadObjectRequest.builder().bucket(bucket).key(evidence.storageObjectKey()).versionId(evidence.storageObjectVersionId()).build());
        if (head.contentLength() == null || head.contentLength() < 1 || head.contentLength() > configMaxArtifactBytes) throw new SecurityException("storage_object_size_limit_exceeded");
        ResponseBytes<GetObjectResponse> downloaded = s3.getObject(GetObjectRequest.builder().bucket(bucket).key(evidence.storageObjectKey()).versionId(evidence.storageObjectVersionId()).build(), boundedBytes(configMaxArtifactBytes));
        GetObjectResponse metadata = downloaded.response();
        if (!evidence.storageObjectVersionId().equals(metadata.versionId())) throw new SecurityException("storage_version_mismatch");
        if (metadata.objectLockMode() != ObjectLockMode.COMPLIANCE || metadata.objectLockRetainUntilDate() == null || !metadata.objectLockRetainUntilDate().isAfter(Instant.now())) throw new SecurityException("storage_worm_retention_missing");
        try { if (!metadata.objectLockRetainUntilDate().equals(Instant.parse(evidence.storageRetentionUntil()))) throw new SecurityException("storage_retention_mismatch"); }
        catch (java.time.format.DateTimeParseException error) { throw new SecurityException("storage_retention_invalid"); }
        byte[] body = downloaded.asByteArray();
        if (body.length != evidence.documentSizeBytes() || !Hex.sha256(body).equals(evidence.documentSha256())) throw new SecurityException("storage_content_mismatch");
        return new Snapshot(body, metadata.versionId(), metadata.objectLockMode(), metadata.objectLockRetainUntilDate());
    }
    ReportReference storeReport(VerifyRequest scope, String documentHash, String name, byte[] report, Instant sourceRetentionUntil) {
        String tenant = safeKeySegment(scope.tenantCode(), "tenant");
        String institution = safeKeySegment(scope.institutionId(), "institution");
        String evidence = safeKeySegment(scope.evidence().id(), "evidence");
        String key = "dss-validation-reports/" + tenant + "/" + institution + "/" + evidence + "/" + documentHash + "/" + name + ".xml";
        Instant retainUntil = Instant.now().plusSeconds(86_400L * reportRetentionDays);
        if (retainUntil.isBefore(sourceRetentionUntil)) retainUntil = sourceRetentionUntil;
        PutObjectResponse response = s3.putObject(PutObjectRequest.builder().bucket(reportBucket).key(key)
            .objectLockMode(ObjectLockMode.COMPLIANCE).objectLockRetainUntilDate(retainUntil).contentType("application/xml")
            .metadata(java.util.Map.of("sha256", Hex.sha256(report), "source-document-sha256", documentHash, "tenant-code", tenant, "institution-id", institution, "evidence-id", evidence)).build(), RequestBody.fromBytes(report));
        if (response.versionId() == null || response.versionId().isBlank()) throw new IllegalStateException("report_bucket_versioning_required");
        return new ReportReference(reportBucket, key, response.versionId(), Hex.sha256(report), report.length, retainUntil.toString());
    }
    private static boolean blank(String value) { return value == null || value.isBlank(); }
    private static <T> ResponseTransformer<T, ResponseBytes<T>> boundedBytes(long maximum) {
        return (response, input) -> {
            try (input; java.io.ByteArrayOutputStream output = new java.io.ByteArrayOutputStream()) {
                byte[] buffer = new byte[8192]; long total = 0; int read;
                while ((read = input.read(buffer)) != -1) { total += read; if (total > maximum) throw new SecurityException("storage_object_size_limit_exceeded"); output.write(buffer, 0, read); }
                return ResponseBytes.fromByteArray(response, output.toByteArray());
            }
        };
    }
    private static String safeKeySegment(String value, String field) { if (value == null || !value.matches("[A-Za-z0-9_-]{1,128}")) throw new SecurityException("report_scope_" + field + "_invalid"); return value; }
    @Override public void close() { s3.close(); }
}
record Snapshot(byte[] bytes, String versionId, ObjectLockMode lockMode, Instant retentionUntil) { }
record ReportReference(String bucket, String key, String versionId, String sha256, int sizeBytes, String retentionUntil) { }
