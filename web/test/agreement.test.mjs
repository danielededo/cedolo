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
    const doc = core.read(JSON.stringify(layout));
    assert.equal(doc.error, undefined);
    const got = core.analyze(JSON.stringify([doc.doc]), 'en');
    assert.equal(got.error, undefined);
    assert.deepEqual(got.perDocument, [{ read: want.length, repeated: 0, periods: ['October 2026'], latest: 202610, people: ['Without fiscal code'] }]);
    // The page gets no payslips from analyze, only the report.
    assert.equal(got.payslips, undefined);
    const read = core.payslips(JSON.stringify([layout]));
    assert.equal(read.error, undefined);
    assert.deepEqual(JSON.parse(read.payslips).map(summary), want.map(summary));
    assert.match(got.report, /<section class="month"/);
  });
}

test('texts exist in both languages', () => {
  const it = JSON.parse(core.texts('it'));
  const en = JSON.parse(core.texts('en'));
  assert.deepEqual(Object.keys(it).sort(), Object.keys(en).sort());
  assert.ok(it['web.heading'] && it['report.disclaimer']);
});

test('each document is read once and kept for every report after', () => {
  const layouts = JSON.parse(core.example(2025).layouts).slice(0, 3);
  const docs = layouts.map((l) => core.read(JSON.stringify(l)).doc);
  assert.ok(docs.every((d) => Number.isInteger(d) && d > 0) && new Set(docs).size === 3);
  // The same documents give the same report, in either language, without
  // being read again; the same payslip read twice counts once.
  const first = core.analyze(JSON.stringify(docs), 'en');
  assert.equal(first.error, undefined);
  assert.deepEqual(core.analyze(JSON.stringify(docs), 'en'), first);
  assert.match(core.analyze(JSON.stringify(docs), 'it').report, /<html lang="it"/);
  const again = core.read(JSON.stringify(layouts[0])).doc;
  const twice = core.analyze(JSON.stringify([...docs, again]), 'en');
  assert.deepEqual(twice.perDocument.at(-1).repeated, 1);
  assert.equal(twice.report, first.report);
  // Forgotten documents are no longer there to report.
  assert.deepEqual(core.forget(JSON.stringify([...docs, again])), {});
  const gone = core.analyze(JSON.stringify(docs), 'en');
  assert.ok(gone.perDocument.every((r) => /not read/.test(r.error)));
  assert.equal(gone.report, '');
});
