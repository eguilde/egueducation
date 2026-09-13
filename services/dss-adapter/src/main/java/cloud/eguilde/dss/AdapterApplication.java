package cloud.eguilde.dss;

import com.sun.net.httpserver.HttpServer;
import java.net.InetSocketAddress;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/** Entrypoint for the deployment-owned, fail-closed DSS verification service. */
public final class AdapterApplication {
    private AdapterApplication() { }

    public static void main(String[] args) throws Exception {
        AdapterConfiguration configuration = AdapterConfiguration.fromEnvironment();
        TrustListRuntime trust = new TrustListRuntime(configuration);
        trust.refresh(); // A failed refresh deliberately leaves readiness false.
        ScheduledExecutorService refresher = Executors.newSingleThreadScheduledExecutor();
        refresher.scheduleWithFixedDelay(trust::refresh, 1, 1, TimeUnit.HOURS);
        VerificationService verifier = new VerificationService(configuration, trust);
        HttpServer server = HttpServer.create(new InetSocketAddress(configuration.bindHost(), configuration.port()), 0);
        server.createContext("/verify", new VerifyHandler(configuration, verifier));
        server.createContext("/verify/capabilities", new CapabilityHandler(configuration, trust));
        server.createContext("/healthz", exchange -> HttpResponses.json(exchange, 200, java.util.Map.of("status", "ok")));
        server.setExecutor(Executors.newFixedThreadPool(configuration.httpWorkers()));
        server.start();
    }
}
