package cloud.eguilde.dss;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import java.io.IOException;
import java.nio.charset.StandardCharsets;

final class VerifyHandler implements HttpHandler {
    private static final int MAX_REQUEST_BYTES = 64 * 1024;
    private final AdapterConfiguration config;
    private final VerificationService verifier;
    VerifyHandler(AdapterConfiguration config, VerificationService verifier) { this.config = config; this.verifier = verifier; }
    @Override public void handle(HttpExchange exchange) throws IOException {
        if (!"POST".equals(exchange.getRequestMethod())) { HttpResponses.json(exchange, 405, java.util.Map.of("error", "method_not_allowed")); return; }
        if (!constantTimeEquals(exchange.getRequestHeaders().getFirst("Authorization"), "Bearer " + config.bearerToken())) { HttpResponses.json(exchange, 401, java.util.Map.of("error", "unauthorized")); return; }
        byte[] body = exchange.getRequestBody().readNBytes(MAX_REQUEST_BYTES + 1);
        if (body.length > MAX_REQUEST_BYTES) { HttpResponses.json(exchange, 413, java.util.Map.of("error", "request_too_large")); return; }
        try { HttpResponses.json(exchange, 200, verifier.verify(Json.MAPPER.readValue(body, VerifyRequest.class))); }
        catch (Exception error) { HttpResponses.json(exchange, 400, java.util.Map.of("error", "invalid_request")); }
    }
    private static boolean constantTimeEquals(String left, String right) { return left != null && java.security.MessageDigest.isEqual(left.getBytes(StandardCharsets.UTF_8), right.getBytes(StandardCharsets.UTF_8)); }
}
