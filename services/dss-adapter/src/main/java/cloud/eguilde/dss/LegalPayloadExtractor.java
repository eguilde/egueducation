package cloud.eguilde.dss;

import java.io.IOException;
import org.apache.pdfbox.Loader;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.pdmodel.PDDocumentNameDictionary;
import org.apache.pdfbox.pdmodel.common.filespecification.PDComplexFileSpecification;
import org.apache.pdfbox.pdmodel.common.filespecification.PDEmbeddedFile;
import org.apache.pdfbox.pdmodel.interactive.digitalsignature.PDSignature;

final class LegalPayloadExtractor {
    static final String ATTACHMENT_NAME = "egueducation-admission-legal-payload.json";
    private LegalPayloadExtractor() { }
    static byte[] extractCoveredCanonicalPayload(byte[] pdf) throws IOException {
        try (PDDocument document = Loader.loadPDF(pdf)) {
            if (document.getSignatureDictionaries().size() != 1) throw new SecurityException("ambiguous_or_missing_pdf_signature");
            assertNoUnsignedIncrement(pdf, document.getSignatureDictionaries().getFirst());
            PDDocumentNameDictionary names = new PDDocumentNameDictionary(document.getDocumentCatalog());
            var tree = names.getEmbeddedFiles();
            if (tree == null) throw new SecurityException("legal_payload_attachment_missing");
            // getValue traverses nested PDF name-tree nodes. PDFs produced by
            // large signing systems commonly use such nodes, so inspecting
            // only root getNames() would reject legitimate signed records.
            PDComplexFileSpecification file = tree.getValue(ATTACHMENT_NAME);
            if (file == null) throw new SecurityException("legal_payload_attachment_missing");
            PDEmbeddedFile embedded = file.getEmbeddedFileUnicode();
            if (embedded == null) embedded = file.getEmbeddedFile();
            if (embedded == null || !"application/json".equalsIgnoreCase(embedded.getSubtype())) throw new SecurityException("legal_payload_attachment_invalid");
            return embedded.toByteArray();
        }
    }
    // For an incrementally signed PDF every byte following the covered range
    // is a post-sign modification. Even whitespace is forbidden: allowing it
    // creates an unverifiable exception in an otherwise exact-byte contract.
    static void assertNoUnsignedIncrement(byte[] pdf, PDSignature signature) {
        int[] range = signature.getByteRange();
        if (range == null || range.length != 4 || range[0] != 0 || range[1] < 1 || range[2] < range[1] || range[3] < 1 || range[2] + range[3] > pdf.length) throw new SecurityException("pdf_signature_byte_range_invalid");
        if (range[2] + range[3] != pdf.length) throw new SecurityException("pdf_post_signature_modification");
    }
}
