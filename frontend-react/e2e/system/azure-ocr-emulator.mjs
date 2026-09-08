/**
 * Deliberately small, deterministic contract emulator for the two external
 * protocols required by an archive ingestion: Azure Document Intelligence and
 * ClamAV INSTREAM.  It is only used by the system proof; production always
 * talks to its configured services.
 */
import http from 'node:http';
import net from 'node:net';

const host = process.env.ARCHIVE_EMULATOR_HOST ?? '127.0.0.1';
const httpPort = Number(process.env.ARCHIVE_EMULATOR_PORT ?? 9090);
const clamdPort = Number(process.env.ARCHIVE_CLAMD_EMULATOR_PORT ?? 3310);
const expectedKey = process.env.AZURE_DOCUMENT_INTELLIGENCE_KEY ?? 'system-e2e-azure-key';
const text = 'EguEducation arhiva test OCR. Dosar elev Maria Popescu. Referinta E2E-ARCHIVE-SEARCH-UNIQUE.';
let polls = 0;

function json(response, status, value) {
  response.writeHead(status, { 'content-type': 'application/json' });
  response.end(JSON.stringify(value));
}

const azure = http.createServer(async (request, response) => {
  const url = new URL(request.url ?? '/', `http://${host}:${httpPort}`);
  if (url.pathname === '/healthz') return json(response, 200, { status: 'ok' });
  if (request.method === 'POST' && /^\/documentintelligence\/documentModels\/[^/]+:analyze$/.test(url.pathname)) {
    if (request.headers['ocp-apim-subscription-key'] !== expectedKey || request.headers['content-type'] !== 'application/pdf') return json(response, 401, { error: { code: 'Unauthorized' } });
    let length = 0;
    for await (const chunk of request) length += chunk.length;
    if (!length) return json(response, 400, { error: { code: 'InvalidRequest' } });
    response.writeHead(202, { 'operation-location': `http://${host}:${httpPort}/operations/system-e2e` });
    return response.end();
  }
  if (request.method === 'GET' && url.pathname === '/operations/system-e2e') {
    polls += 1;
    if (polls === 1) return json(response, 200, { status: 'running' });
    return json(response, 200, { status: 'succeeded', analyzeResult: { content: text, pages: [{}] } });
  }
  return json(response, 404, { error: { code: 'NotFound' } });
});

// A protocol-faithful clean scanner is needed because archive uploads are
// fail-closed when ClamAV is unavailable.  It consumes INSTREAM frames and
// returns exactly the response ClamAV returns for a clean document.
const clamd = net.createServer((socket) => {
  let buffer = Buffer.alloc(0);
  let begun = false;
  socket.on('data', (chunk) => {
    buffer = Buffer.concat([buffer, chunk]);
    if (!begun) {
      const marker = Buffer.from('zINSTREAM\0');
      if (buffer.length < marker.length) return;
      if (!buffer.subarray(0, marker.length).equals(marker)) return socket.destroy();
      buffer = buffer.subarray(marker.length);
      begun = true;
    }
    while (buffer.length >= 4) {
      const size = buffer.readUInt32BE(0);
      if (buffer.length < size + 4) return;
      buffer = buffer.subarray(size + 4);
      if (size === 0) {
        socket.end('stream: OK\0');
        return;
      }
    }
  });
});

await Promise.all([
  new Promise((resolve) => azure.listen(httpPort, host, resolve)),
  new Promise((resolve) => clamd.listen(clamdPort, host, resolve)),
]);
process.on('SIGTERM', () => { azure.close(); clamd.close(); });
