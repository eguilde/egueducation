package cloud.eguilde.dss;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.JsonNode;
import java.util.Map;

record VerifyRequest(@JsonProperty("tenant_code") String tenantCode, @JsonProperty("institution_id") String institutionId, Evidence evidence) { }
record Evidence(String id, @JsonProperty("artifact_type") String artifactType, @JsonProperty("artifact_id") String artifactId,
                @JsonProperty("document_sha256") String documentSha256, @JsonProperty("document_size_bytes") long documentSizeBytes,
                @JsonProperty("expected_canonical_legal_payload_sha256") String expectedCanonicalLegalPayloadSha256,
                @JsonProperty("expected_actor_subject") String expectedActorSubject,
                @JsonProperty("signature_format") String signatureFormat, @JsonProperty("signature_level") String signatureLevel,
                @JsonProperty("signature_subject") String signatureSubject, @JsonProperty("certificate_issuer") String certificateIssuer,
                @JsonProperty("certificate_serial") String certificateSerial, @JsonProperty("certificate_valid_from") String certificateValidFrom,
                @JsonProperty("certificate_valid_until") String certificateValidUntil,
                @JsonProperty("storage_document_id") String storageDocumentId, @JsonProperty("storage_version_id") String storageVersionId,
                @JsonProperty("storage_bucket") String storageBucket, @JsonProperty("storage_object_key") String storageObjectKey,
                @JsonProperty("storage_object_version_id") String storageObjectVersionId,
                @JsonProperty("storage_retention_until") String storageRetentionUntil,
                @JsonProperty("submitted_by_subject") String submittedBySubject, @JsonProperty("submitted_at") String submittedAt,
                @JsonProperty("latest_validation") JsonNode latestValidation) { }

record VerifyResponse(String status, @JsonProperty("signature_format") String signatureFormat,
                      @JsonProperty("signature_level") String signatureLevel, @JsonProperty("signature_subject") String signatureSubject,
                      @JsonProperty("certificate_issuer") String certificateIssuer, @JsonProperty("certificate_serial") String certificateSerial,
                      @JsonProperty("certificate_valid_from") String certificateValidFrom, @JsonProperty("certificate_valid_until") String certificateValidUntil,
                      @JsonProperty("trusted_list_provider") String trustedListProvider, @JsonProperty("validator_provider") String validatorProvider,
                      @JsonProperty("validator_version") String validatorVersion, @JsonProperty("validation_policy") String validationPolicy,
                      @JsonProperty("observed_sha256") String observedSha256, @JsonProperty("observed_size_bytes") long observedSizeBytes,
                      @JsonProperty("signed_payload_sha256") String signedPayloadSha256, @JsonProperty("certificate_sha256") String certificateSha256,
                      @JsonProperty("signed_actor_subject") String signedActorSubject, @JsonProperty("diagnostic_data") Map<String, Object> diagnosticData,
                      @JsonProperty("detailed_report") Map<String, Object> detailedReport, @JsonProperty("simple_report") Map<String, Object> simpleReport,
                      @JsonProperty("etsi_validation_report") Map<String, Object> etsiValidationReport,
                      @JsonProperty("timestamp_token_sha256") String timestampTokenSha256, @JsonProperty("timestamp_at") String timestampAt,
                      @JsonProperty("timestamp_authority") String timestampAuthority, Map<String, Object> findings) {
    static VerifyResponse rejected(String code) { return new VerifyResponse("invalid", "", "", "", "", "", "", "", "", "EU DSS", "6.5", "eIDAS EU LOTL/TL", "", 0, "", "", "", Map.of(), Map.of(), Map.of(), Map.of(), "", "", "", Map.of("code", code)); }
    static VerifyResponse unavailable(String code) { return new VerifyResponse("error", "", "", "", "", "", "", "", "", "EU DSS", "6.5", "eIDAS EU LOTL/TL", "", 0, "", "", "", Map.of(), Map.of(), Map.of(), Map.of(), "", "", "", Map.of("code", code)); }
}
