import http from 'node:http';
import { createHash } from 'node:crypto';

// Explicit system-test trust-service emulator. It is started only by the
// Playwright system topology; production continues to require a configured
// HTTPS verifier and has no deterministic bypass.
const token = 'system-test-dss-token';
const sha256Pattern = /^[0-9a-f]{64}$/;
const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const certificateSHA256 = createHash('sha256').update('egueducation-system-test-leaf-certificate-v1').digest('hex');
const signedManifestFixtures = new Map();

function reject(res, code) {
  res.writeHead(422, { 'content-type': 'application/json' }).end(JSON.stringify({ status: 'invalid', findings: { code } }));
}

function validatedSignedManifest(request) {
  const evidence = request?.evidence;
  if (!evidence || typeof request.tenant_code !== 'string' || request.tenant_code.trim() === '' || typeof request.institution_id !== 'string' || request.institution_id.trim() === '') return null;
  if (!['admission_decision', 'admission_appeal_resolution'].includes(evidence.artifact_type) || !uuidPattern.test(evidence.artifact_id ?? '')) return null;
  if (!sha256Pattern.test(evidence.document_sha256 ?? '') || !Number.isSafeInteger(evidence.document_size_bytes) || evidence.document_size_bytes <= 0) return null;
  if (!sha256Pattern.test(evidence.expected_canonical_legal_payload_sha256 ?? '') || typeof evidence.expected_actor_subject !== 'string' || evidence.expected_actor_subject.trim() === '') return null;
  if (!uuidPattern.test(evidence.storage_document_id ?? '') || !uuidPattern.test(evidence.storage_version_id ?? '') || typeof evidence.storage_bucket !== 'string' || evidence.storage_bucket.trim() === '' || typeof evidence.storage_object_key !== 'string' || evidence.storage_object_key.trim() === '' || typeof evidence.storage_object_version_id !== 'string' || evidence.storage_object_version_id.trim() === '') return null;
  const retentionUntil = Date.parse(evidence.storage_retention_until ?? '');
  if (!Number.isFinite(retentionUntil) || retentionUntil <= Date.now()) return null;

  // The emulator freezes the first signed manifest seen for an artifact, just
  // like a fixture embedded in an immutable test PDF. A retry that changes the
  // legal payload, actor, document digest, or storage identity is tampering and
  // must not receive a trusted response.
  const fixtureKey = `${request.tenant_code}\u0000${request.institution_id}\u0000${evidence.artifact_type}\u0000${evidence.artifact_id}`;
  const manifest = JSON.stringify({
    schema: 'egueducation.system-test.signed-manifest.v1',
    legal_payload_sha256: evidence.expected_canonical_legal_payload_sha256,
    actor_subject: evidence.expected_actor_subject,
    document_sha256: evidence.document_sha256,
    document_size_bytes: evidence.document_size_bytes,
    storage_document_id: evidence.storage_document_id,
    storage_version_id: evidence.storage_version_id,
    storage_object_version_id: evidence.storage_object_version_id,
    storage_retention_until: evidence.storage_retention_until,
  });
  const embedded = signedManifestFixtures.get(fixtureKey);
  if (embedded !== undefined && embedded !== manifest) return null;
  if (embedded === undefined) signedManifestFixtures.set(fixtureKey, manifest);
  return JSON.parse(manifest);
}

http.createServer((req, res) => {
  if (req.url === '/healthz') { res.writeHead(200).end('ok'); return; }
  if (req.url === '/capabilities') {
    if (req.method !== 'GET' || req.headers.authorization !== `Bearer ${token}`) { res.writeHead(401).end(); return; }
    res.writeHead(200, { 'content-type': 'application/json' }).end(JSON.stringify({
      status: 'ready', protocol_version: 'egueducation-signed-artifact-verifier.v1',
      capabilities: ['signed-payload-sha256', 'leaf-certificate-sha256', 'expected-actor-subject', 'exact-storage-object-version'],
    }));
    return;
  }
  if (req.method !== 'POST' || req.headers.authorization !== `Bearer ${token}`) { res.writeHead(401).end(); return; }
  let body = ''; req.on('data', chunk => { body += chunk; if (body.length > 1024 * 1024) req.destroy(); }); req.on('end', () => {
    let request;
    try { request = JSON.parse(body); } catch { reject(res, 'malformed_verifier_request'); return; }
    const manifest = validatedSignedManifest(request);
    if (!manifest) { reject(res, 'signed_manifest_mismatch'); return; }
    const evidence = request.evidence;
    const report = { emulator: true, fixture: manifest.schema, artifact: evidence.artifact_id };
    res.writeHead(200, { 'content-type': 'application/json' }).end(JSON.stringify({
      status: 'valid', signature_format: 'PAdES', signature_level: 'qualified', signature_subject: 'CN=EguEducation System Test Signer', signed_actor_subject: manifest.actor_subject, certificate_issuer: 'System Test CA', certificate_serial: 'system-test-01', certificate_valid_from: '2026-01-01T00:00:00Z', certificate_valid_until: '2030-01-01T00:00:00Z', certificate_sha256: certificateSHA256, signed_payload_sha256: manifest.legal_payload_sha256, trusted_list_provider: 'system-test-trusted-list', validator_provider: 'system-test-dss', validator_version: '1.0', validation_policy: 'ETSI-EN-319-102-1', observed_sha256: manifest.document_sha256, observed_size_bytes: manifest.document_size_bytes, timestamp_token_sha256: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', timestamp_at: '2026-09-12T00:00:00Z', timestamp_authority: 'system-test-tsa', diagnostic_data: report, detailed_report: report, simple_report: report, etsi_validation_report: report, findings: report,
    }));
  });
}).listen(9091, '127.0.0.1');
