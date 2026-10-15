// The built page must keep its security limits; see
// docs/design/web-security.md. Run after "go run ./cmd/webbuild".
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const dist = new URL('../../dist/web/', import.meta.url).pathname;
const page = readFileSync(join(dist, 'index.html'), 'utf8');
const app = readFileSync(join(dist, 'app.js'), 'utf8');

function policy() {
  const m = page.match(/http-equiv="Content-Security-Policy" content="([^"]+)"/);
  assert.ok(m, 'the page has a Content-Security-Policy');
  return Object.fromEntries(m[1].split(';').map((d) => d.trim().split(/\s+/)).map(([k, ...v]) => [k, v]));
}

test('the policy is the first thing in the head', () => {
  const head = page.slice(0, page.indexOf('Content-Security-Policy'));
  assert.doesNotMatch(head, /<script|<link/);
});

test('nothing can be fetched, framed or run from elsewhere', () => {
  const p = policy();
  assert.deepEqual(p['default-src'], ["'none'"]);
  assert.deepEqual(p['connect-src'], ["'self'"]);
  for (const d of ['worker-src', 'frame-src', 'object-src', 'font-src', 'base-uri', 'form-action']) {
    assert.deepEqual(p[d], ["'none'"], d);
  }
  assert.deepEqual(p['script-src'].filter((s) => !s.startsWith("'sha256-")), ["'self'", "'wasm-unsafe-eval'"]);
  assert.equal(p['script-src'].filter((s) => s.startsWith("'sha256-")).length, 1, 'only the report script is allowed inline');
  for (const [d, sources] of Object.entries(p)) {
    for (const s of sources) {
      assert.doesNotMatch(s, /^(https?:|\*|'unsafe-eval'$)/, `${d} ${s}`);
    }
  }
});

test('the page loads nothing from other sites', () => {
  assert.match(app, /^import '\.\/compat\.js';/m, 'compat.js comes before pdf.js');
  for (const m of page.matchAll(/(?:src|href)="([^"]+)"/g)) {
    assert.match(m[1], /^(?!\/\/)(?![a-z]+:)|^data:image\/svg|^https:\/\/github\.com\/danielededo\/cedolo$/, m[1]);
  }
});

test('pdf.js runs in the page, under its policy, not in a worker', () => {
  assert.match(app, /globalThis\.pdfjsWorker = await import\('\.\/pdfjs\/pdf\.worker\.min\.mjs'\)/);
  assert.doesNotMatch(app, /workerSrc|new Worker/);
});

test('the report frame is sandboxed without its own origin', () => {
  const frame = page.match(/<iframe[^>]*id="report"[^>]*>/)[0];
  assert.match(frame, /sandbox="allow-scripts"/);
});

test('nothing is stored', () => {
  assert.doesNotMatch(app, /localStorage|sessionStorage|indexedDB|document\.cookie|caches\.|serviceWorker/);
});
