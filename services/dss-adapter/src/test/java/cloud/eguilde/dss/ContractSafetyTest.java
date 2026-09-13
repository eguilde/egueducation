package cloud.eguilde.dss;

import static org.junit.jupiter.api.Assertions.*;
import java.nio.charset.StandardCharsets;
import org.apache.pdfbox.pdmodel.interactive.digitalsignature.PDSignature;
import eu.europa.esig.dss.enumerations.SignatureQualification;
import eu.europa.esig.dss.enumerations.TimestampType;
import org.junit.jupiter.api.Test;

class ContractSafetyTest {
    @Test void admitsOnlyPersonSignatureQualificationLevels() {
        assertEquals("qualified", VerificationService.legalSignatureLevel(SignatureQualification.QESIG));
        assertEquals("advanced", VerificationService.legalSignatureLevel(SignatureQualification.ADESIG));
        assertEquals("advanced", VerificationService.legalSignatureLevel(SignatureQualification.ADESIG_QC));
        assertNull(VerificationService.legalSignatureLevel(SignatureQualification.QESEAL));
        assertNull(VerificationService.legalSignatureLevel(SignatureQualification.ADESEAL));
        assertNull(VerificationService.legalSignatureLevel(SignatureQualification.UNKNOWN));
    }
    @Test void onlySignatureTimestampMayAuthorizeAnAdmissionAct() {
        assertTrue(VerificationService.isAdmissionTimestampType(TimestampType.SIGNATURE_TIMESTAMP));
        assertFalse(VerificationService.isAdmissionTimestampType(TimestampType.CONTENT_TIMESTAMP));
        assertFalse(VerificationService.isAdmissionTimestampType(TimestampType.ARCHIVE_TIMESTAMP));
        assertFalse(VerificationService.isAdmissionTimestampType(TimestampType.DOCUMENT_TIMESTAMP));
    }
    @Test void acceptsTheExactGoSignedArtifactEvidenceShape() throws Exception {
        String hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
        String json = """
          {"tenant_code":"tenant-a","institution_id":"institution-a","evidence":{"id":"evidence-1","artifact_type":"admission_decision","artifact_id":"artifact-1","document_sha256":"%s","document_size_bytes":42,"expected_canonical_legal_payload_sha256":"%s","expected_actor_subject":"oidc-subject","signature_format":"PAdES","signature_level":"advanced","signature_subject":"CN=old","certificate_issuer":"CN=issuer","certificate_serial":"01","certificate_valid_from":"2026-01-01T00:00:00Z","certificate_valid_until":"2027-01-01T00:00:00Z","storage_document_id":"doc-1","storage_version_id":"archive-version-1","storage_bucket":"earhive","storage_object_key":"tenant-a/file.pdf","storage_object_version_id":"minio-version-1","storage_retention_until":"2030-01-01T00:00:00Z","submitted_by_subject":"oidc-subject","submitted_at":"2026-01-01T00:00:00Z","latest_validation":{"id":"previous","validation_status":"valid"}}}
          """.formatted(hash, hash);
        VerifyRequest decoded = Json.MAPPER.readValue(json, VerifyRequest.class);
        assertEquals("oidc-subject", decoded.evidence().submittedBySubject());
        assertEquals("PAdES", decoded.evidence().signatureFormat());
        assertTrue(decoded.evidence().latestValidation().isObject());
    }
    @Test void validatesOnlyLowercaseSha256() {
        assertTrue(Hex.sha256("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"));
        assertFalse(Hex.sha256("0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef"));
        assertEquals("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", Hex.sha256("abc".getBytes(StandardCharsets.UTF_8)));
    }
    @Test void rejectsNonWhitespacePostSignatureBytes() {
        PDSignature signature = new PDSignature();
        signature.setByteRange(new int[] {0, 2, 4, 2});
        SecurityException failure = assertThrows(SecurityException.class, () -> LegalPayloadExtractor.assertNoUnsignedIncrement("1234XXx".getBytes(StandardCharsets.US_ASCII), signature));
        assertEquals("pdf_post_signature_modification", failure.getMessage());
    }
    @Test void rejectsWhitespaceAfterCoveredPdf() {
        PDSignature signature = new PDSignature();
        signature.setByteRange(new int[] {0, 2, 4, 2});
        assertThrows(SecurityException.class, () -> LegalPayloadExtractor.assertNoUnsignedIncrement("1234XX \r\n".getBytes(StandardCharsets.US_ASCII), signature));
    }
}
