package cloud.eguilde.dss;

import java.security.MessageDigest;
import java.util.HexFormat;

final class Hex {
    private Hex() { }
    static String sha256(byte[] bytes) { try { return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(bytes)); } catch (Exception error) { throw new IllegalStateException("SHA-256 unavailable", error); } }
    static boolean sha256(String value) { return value != null && value.matches("[0-9a-f]{64}"); }
}
