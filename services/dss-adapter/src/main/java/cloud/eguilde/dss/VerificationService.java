package cloud.eguilde.dss;

import eu.europa.esig.dss.model.InMemoryDocument;
import eu.europa.esig.dss.model.x509.CertificateToken;
import eu.europa.esig.dss.enumerations.SignatureForm;
import eu.europa.esig.dss.enumerations.SignatureQualification;
import eu.europa.esig.dss.enumerations.TimestampQualification;
import eu.europa.esig.dss.enumerations.TimestampType;
import eu.europa.esig.dss.spi.signature.AdvancedSignature;
import eu.europa.esig.dss.spi.x509.tsp.TimestampToken;
import eu.europa.esig.dss.validation.SignedDocumentValidator;
import eu.europa.esig.dss.validation.reports.Reports;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;

/** Performs the object provenance, covered-payload, PAdES and trust checks in that order. */
final class VerificationService implements AutoCloseable {
    private final AdapterConfiguration config;
    private final TrustListRuntime trust;
    private final VersionedObjectStore objects;
    VerificationService(AdapterConfiguration config, TrustListRuntime trust) { this.config = config; this.trust = trust; this.objects = new VersionedObjectStore(config); }

    VerifyResponse verify(VerifyRequest request) {
        try {
            assertRequest(request);
            if (!trust.ready()) return VerifyResponse.unavailable("trust_list_not_ready");
            Snapshot snapshot = objects.load(request.evidence());
            byte[] canonicalPayload = LegalPayloadExtractor.extractCoveredCanonicalPayload(snapshot.bytes());
            String payloadHash = Hex.sha256(canonicalPayload);
            if (!payloadHash.equals(request.evidence().expectedCanonicalLegalPayloadSha256())) return VerifyResponse.rejected("canonical_legal_payload_hash_mismatch");
            SignedDocumentValidator validator = SignedDocumentValidator.fromDocument(new InMemoryDocument(snapshot.bytes(), "signed-legal-artifact.pdf"));
            validator.setCertificateVerifier(trust.certificateVerifier());
            validator.setEnableEtsiValidationReport(true);
            Reports reports = validator.validateDocument();
            if (validator.getSignatures().size() != 1) return VerifyResponse.rejected("ambiguous_or_missing_dss_signature");
            AdvancedSignature signature = validator.getSignatures().getFirst();
            String signatureId = signature.getId();
            if (signature.getSignatureForm() != SignatureForm.PAdES || !reports.getSimpleReport().isValid(signatureId)) return VerifyResponse.rejected("dss_validation_invalid");
            String legalLevel = legalSignatureLevel(reports.getSimpleReport().getSignatureQualification(signatureId));
            if (legalLevel == null) return VerifyResponse.rejected("dss_signature_not_advanced_or_qualified");
            TimestampFacts timestamp = qualifiedTimestamp(signature, reports);
            if (timestamp == null) return VerifyResponse.rejected("dss_qualified_timestamp_missing");
            CertificateToken certificate = signature.getSigningCertificateToken();
            if (certificate == null) return VerifyResponse.rejected("dss_signing_certificate_missing");
            String certificateHash = Hex.sha256(certificate.getEncoded());
            CertificateBinding binding = config.certificateBindings().get(certificateHash);
            if (binding == null || !binding.tenantCode().equals(request.tenantCode()) || !binding.institutionId().equals(request.institutionId()) || !binding.actorSubject().equals(request.evidence().expectedActorSubject())) return VerifyResponse.rejected("certificate_actor_scope_binding_mismatch");
            String documentHash = Hex.sha256(snapshot.bytes());
            ReportReference diagnostic = objects.storeReport(request, documentHash, "diagnostic", reports.getXmlDiagnosticData().getBytes(java.nio.charset.StandardCharsets.UTF_8), snapshot.retentionUntil());
            ReportReference detailed = objects.storeReport(request, documentHash, "detailed", reports.getXmlDetailedReport().getBytes(java.nio.charset.StandardCharsets.UTF_8), snapshot.retentionUntil());
            ReportReference simple = objects.storeReport(request, documentHash, "simple", reports.getXmlSimpleReport().getBytes(java.nio.charset.StandardCharsets.UTF_8), snapshot.retentionUntil());
            ReportReference etsi = objects.storeReport(request, documentHash, "etsi", reports.getXmlValidationReport().getBytes(java.nio.charset.StandardCharsets.UTF_8), snapshot.retentionUntil());
            return new VerifyResponse("valid", "PAdES", legalLevel, certificate.getSubject().getRFC2253(),
                certificate.getIssuer().getRFC2253(), certificate.getSerialNumber().toString(16), certificate.getNotBefore().toInstant().toString(), certificate.getNotAfter().toInstant().toString(),
                "EU LOTL/TL", "EU DSS", "6.5", "eIDAS EU LOTL/TL", documentHash, snapshot.bytes().length, payloadHash, certificateHash,
                binding.actorSubject(), reference(diagnostic), reference(detailed), reference(simple), reference(etsi), timestamp.sha256(), timestamp.at(), timestamp.authority(), Map.of("storage_object_version_id", snapshot.versionId(), "storage_retention_until", snapshot.retentionUntil().toString(), "validated_at", Instant.now().toString()));
        } catch (SecurityException error) { return VerifyResponse.rejected(error.getMessage()); }
          catch (IllegalStateException error) { return VerifyResponse.unavailable(error.getMessage()); }
          catch (Exception error) { return VerifyResponse.unavailable("dss_validation_unavailable"); }
    }
    private static Map<String, Object> reference(ReportReference report) { return Map.of("storage_bucket", report.bucket(), "storage_object_key", report.key(), "storage_object_version_id", report.versionId(), "sha256", report.sha256(), "size_bytes", report.sizeBytes(), "retention_until", report.retentionUntil()); }
    private static void assertRequest(VerifyRequest request) {
        if (request == null || blank(request.tenantCode()) || blank(request.institutionId()) || request.evidence() == null) throw new SecurityException("verification_request_invalid");
        Evidence evidence = request.evidence();
        if (!Hex.sha256(evidence.documentSha256()) || !Hex.sha256(evidence.expectedCanonicalLegalPayloadSha256()) || blank(evidence.expectedActorSubject()) || evidence.documentSizeBytes() < 1) throw new SecurityException("verification_request_contract_invalid");
    }
    private static boolean blank(String value) { return value == null || value.isBlank(); }
    static String legalSignatureLevel(SignatureQualification qualification) {
        // An admission decision is an actor-bound act. Legal-seal certificates
        // authenticate an organisation, not the OIDC person being authorised.
        if (qualification == SignatureQualification.QESIG) return "qualified";
        if (qualification == SignatureQualification.ADESIG || qualification == SignatureQualification.ADESIG_QC) return "advanced";
        return null;
    }
    private static TimestampFacts qualifiedTimestamp(AdvancedSignature signature, Reports reports) {
        for (TimestampToken token : signature.getAllTimestamps()) {
            String id = token.getDSSIdAsString();
            if (!isAdmissionTimestampType(token.getTimeStampType()) || !token.isValid() || token.getGenerationTime() == null || token.getTSTInfoTsa() == null || reports.getSimpleReport().getTimestampQualification(id) != TimestampQualification.QTSA) continue;
            return new TimestampFacts(Hex.sha256(token.getEncoded()), token.getGenerationTime().toInstant().toString(), token.getTSTInfoTsa().getName());
        }
        return null;
    }
    static boolean isAdmissionTimestampType(TimestampType type) { return type == TimestampType.SIGNATURE_TIMESTAMP && type.coversSignature(); }
    private record TimestampFacts(String sha256, String at, String authority) { }
    @Override public void close() { objects.close(); }
}
