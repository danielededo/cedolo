// The web app must read payslips as the CLI does: for every synthetic
// layout style, the payslips found by pdf.js and the Go core compiled to
// WebAssembly must match those found by the CLI with PDFium. Run from the
// repository root after building the app:
//
//	go run ./cmd/webbuild -o dist/web && (cd web && npm test)
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import * as pdfjs from 'pdfjs-dist/legacy/build/pdf.mjs';
import { layoutOf } from '../layout.js';

const root = new URL('../../', import.meta.url).pathname;
const site = join(root, 'dist', 'web');
const dist = join(site, readFileSync(join(site, 'index.html'), 'utf8').match(/src="(a\/[0-9a-f]{12})\/app\.js"/)[1]);
const fonts = new URL('../node_modules/pdfjs-dist/standard_fonts/', import.meta.url).pathname;

async function startCore() {
  createRequire(import.meta.url)(join(dist, 'wasm_exec.js'));
  const go = new globalThis.Go();
  const { instance } = await WebAssembly.instantiate(readFileSync(join(dist, 'cedolo.wasm')), go.importObject);
  go.run(instance);
  return globalThis.cedolo;
}

function cli(...args) {
  return execFileSync('go', ['run', './cmd/cedolo', ...args], { cwd: root, encoding: 'utf8' });
}

// summary keeps what a reader must agree on.
function summary(p) {
  return {
    period: [p.year, p.month, p.kind],
    ccnl: [p.ccnl, p.ccnl_code, p.level],
    totals: ['gross', 'deductions', 'net', 'tfr'].map((k) => [p[k]?.cents, p[k]?.status]),
    items: p.items.map((it) => [it.key, it.cents, it.side]),
    leave: (p.leave ?? []).map((l) => [l.kind, l.remaining]),
  };
}

const core = await startCore();
const dir = mkdtempSync(join(tmpdir(), 'cedolo-web-'));

const cases = ['classic', 'signed', 'grid', 'spaced'].map((style) => ({ style, rotate: 0 }))
  .concat([90, 180, 270].map((rotate) => ({ style: 'grid', rotate })));

for (const { style, rotate } of cases) {
  test(`pdf.js and PDFium agree on the ${style} style, turned ${rotate}°`, async () => {
    const pdf = join(dir, `${style}-${rotate}.pdf`);
    cli('synth', '-style', style, '-rotate', String(rotate), '-o', pdf);
    const want = JSON.parse(cli('extract', '-json', pdf));

    const task = pdfjs.getDocument({ data: new Uint8Array(readFileSync(pdf)), isEvalSupported: false, standardFontDataUrl: fonts });
    const layout = await layoutOf(await task.promise);
    await task.destroy();
    const got = core.analyze(JSON.stringify([layout]), 'en');
    assert.equal(got.error, undefined);
    assert.deepEqual(got.perDocument, [{ read: want.length, repeated: 0 }]);
    assert.deepEqual(JSON.parse(got.payslips).map(summary), want.map(summary));
    assert.match(got.report, /<section class="month"/);
  });
}

test('texts exist in both languages', () => {
  const it = JSON.parse(core.texts('it'));
  const en = JSON.parse(core.texts('en'));
  assert.deepEqual(Object.keys(it).sort(), Object.keys(en).sort());
  assert.ok(it['web.heading'] && it['report.disclaimer']);
});
