package cloud.eguilde.dss;

import eu.europa.esig.dss.model.DSSDocument;
import eu.europa.esig.dss.model.InMemoryDocument;
import eu.europa.esig.dss.spi.client.http.DSSFileLoader;
import eu.europa.esig.dss.spi.client.http.NativeHTTPDataLoader;

/** DSS 6.5 separates byte-oriented HTTP and document-oriented TL loaders. */
final class NativeDssFileLoader implements DSSFileLoader {
    private final NativeHTTPDataLoader http;
    NativeDssFileLoader(NativeHTTPDataLoader http) { this.http = http; }
    @Override public DSSDocument getDocument(String url) { return new InMemoryDocument(http.get(url), url); }
}
