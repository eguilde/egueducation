package cloud.eguilde.dss;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Map;

final class CapabilityHandler implements HttpHandler {
    private final AdapterConfiguration config; private final TrustListRuntime trust;
    CapabilityHandler(AdapterConfiguration config, TrustListRuntime trust) { this.config = config; this.trust = trust; }
    @Override public void handle(HttpExchange exchange) throws IOException {
        if (!"GET".equals(exchange.getRequestMethod())) { HttpResponses.json(exchange, 405, Map.of("error", "method_not_allowed")); return; }
        if (exchange.getRequestHeaders().getFirst("Authorization") == null || !java.security.MessageDigest.isEqual(exchange.getRequestHeaders().getFirst("Authorization").getBytes(StandardCharsets.UTF_8), ("Bearer " + config.bearerToken()).getBytes(StandardCharsets.UTF_8))) { HttpResponses.json(exchange, 401, Map.of("error", "unauthorized")); return; }
        if (!trust.ready()) { HttpResponses.json(exchange, 503, Map.of("status", "not_ready", "reason", trust.failure())); return; }
        HttpResponses.json(exchange, 200, Map.of("status", "ready", "protocol_version", AdapterConfiguration.PROTOCOL, "capabilities", List.of("signed-payload-sha256", "leaf-certificate-sha256", "expected-actor-subject", "exact-storage-object-version")));
    }
}
