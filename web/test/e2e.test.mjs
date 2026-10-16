// End-to-end checks of the built web app in a browser: real files, the ways
// they go wrong, and the security limits. Run after "go run ./cmd/webbuild".
// E2E_BROWSER picks chromium (default) or webkit, Safari's engine;
// CHROMIUM_PATH uses an installed Chromium instead of Playwright's.
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createServer } from 'node:http';
import { mkdtempSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { extname, join } from 'node:path';
import { chromium, webkit } from 'playwright';

const root = new URL('../../', import.meta.url).pathname;
const dist = join(root, 'dist', 'web');
const dir = mkdtempSync(join(tmpdir(), 'cedolo-e2e-'));
const types = { '.html': 'text/html', '.js': 'text/javascript', '.mjs': 'text/javascript', '.wasm': 'application/wasm', '.css': 'text/css' };

let site, outside, outsideHits = 0, browser, base, other;

// Passwords of the synthetic protected files the tests generate; they
// protect nothing real.
const TEST_PASSWORD = 'segreto'; // ggignore
const OTHER_PASSWORD = 'altra'; // ggignore
const WRONG_PASSWORD = 'sbagliata'; // ggignore

function synth(name, ...args) {
  const path = join(dir, name);
  execFileSync('go', ['run', './cmd/cedolo', 'synth', ...args, '-o', path], { cwd: root });
  return path;
}

before(async () => {
  site = createServer((req, res) => {
    const p = join(dist, decodeURIComponent(req.url.split('?')[0]).replace(/\/$/, '/index.html'));
    try {
      statSync(p);
      res.writeHead(200, { 'content-type': types[extname(p)] || 'application/octet-stream' });
      res.end(readFileSync(p));
    } catch {
      res.writeHead(404);
      res.end();
    }
  }).listen(0, '127.0.0.1');
  outside = createServer((req, res) => {
    outsideHits++;
    res.writeHead(200, { 'access-control-allow-origin': '*' });
    res.end('ok');
  }).listen(0, '127.0.0.1');
  await Promise.all([site, outside].map((s) => new Promise((ok) => s.on('listening', ok))));
  base = `http://127.0.0.1:${site.address().port}/`;
  other = `http://127.0.0.1:${outside.address().port}/leak`;
  browser = process.env.E2E_BROWSER === 'webkit'
    ? await webkit.launch()
    : await chromium.launch(process.env.CHROMIUM_PATH ? { executablePath: process.env.CHROMIUM_PATH } : {});
});

after(async () => {
  await browser?.close();
  site?.close();
  outside?.close();
});

// open loads the app and records what should never happen; before runs in
// the page ahead of the app, to change what the browser offers.
async function open({ before, viewport } = {}) {
  const page = await browser.newPage({ locale: 'en-US', ...(viewport && { viewport }) });
  const seen = { requests: [], errors: [], dialogs: 0, missing: [] };
  page.on('request', (r) => seen.requests.push(r.url()));
  page.on('response', (r) => { if (r.status() >= 400) seen.missing.push(r.url()); });
  page.on('pageerror', (e) => seen.errors.push(e.message));
  page.on('dialog', (d) => { seen.dialogs++; d.dismiss(); });
  await page.addInitScript(() => {
    document.addEventListener('securitypolicyviolation', (e) => (window.__violations ||= []).push(e.violatedDirective));
  });
  if (before) await page.addInitScript(before);
  await page.goto(base);
  await page.waitForFunction(() => window.cedoloReady === true, null, { timeout: 30000 });
  return { page, seen };
}

async function statuses(page) {
  await page.waitForSelector('#result:not([hidden])', { timeout: 60000 });
  return page.$$eval('#file-list li', (ls) => ls.map((l) => `${l.className}: ${l.querySelector('.note').textContent}`));
}

// press clicks a control of the page. The report frame runs in another
// process, and headless Chromium routes a click by where frames were last
// drawn, which can be before the latest layout change or scroll: bring the
// control into view and let a frame be drawn first.
async function press(page, selector) {
  await page.$eval(selector, (e) => e.scrollIntoView({ block: 'center' }));
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
  await page.click(selector);
}

function onlyThisSite(seen) {
  const elsewhere = seen.requests.filter((u) => !u.startsWith(base) && !u.startsWith('data:') && !u.startsWith('about:'));
  assert.deepEqual(elsewhere, []);
}

test('reads payslips in every style and rotation and shows the report', async () => {
  // The classic and signed examples hold the same amounts, so the second is
  // a repeat; the grid example differs and is turned for display.
  const files = [synth('classic.pdf'), synth('signed.pdf', '-style', 'signed'), synth('grid-90.pdf', '-style', 'grid', '-rotate', '90')];
  const { page, seen } = await open();
  await page.setInputFiles('#files', files);
  assert.deepEqual(await statuses(page), [
    'ok: — 1 payslip(s) read',
    'warn: — 1 payslip(s) read, 1 already read in another file and ignored',
    'ok: — 1 payslip(s) read',
  ]);
  const frame = page.frames().find((f) => f !== page.mainFrame());
  await frame.waitForSelector('section.month:not([hidden]) .bar');
  await page.waitForFunction(() => parseInt(document.getElementById('report').style.height, 10) > 600);
  // One payslip at a time: the picker switches, and a segment of the bar
  // opens its explanation.
  const shown = () => frame.$$eval('section.month', (ms) => ms.filter((m) => !m.hidden).map((m) => m.dataset.month));
  assert.deepEqual(await shown(), ['1']);
  await frame.click('.picker button[data-month="0"]');
  assert.deepEqual(await shown(), ['0']);
  await frame.click('section.month:not([hidden]) .bar .seg.part-taxes');
  assert.equal(await frame.$eval('section.month:not([hidden]) details.part-taxes', (d) => d.open), true);
  assert.match(await frame.$eval('section.month:not([hidden]) details.part-taxes', (d) => d.textContent), /IRPEF/);
  assert.deepEqual(seen.errors, []);
  assert.deepEqual(seen.missing, [], 'every file of the app is served');
  assert.equal(seen.dialogs, 0);
  assert.deepEqual(await page.evaluate(() => window.__violations || []), []);
  onlyThisSite(seen);
  await page.close();
});

test('handles damaged, oversized, repeated and protected files', async () => {
  const good = synth('good.pdf');
  const locked1 = synth('locked1.pdf', '-style', 'grid', '-password', TEST_PASSWORD);
  const locked2 = synth('locked2.pdf', '-style', 'signed', '-password', TEST_PASSWORD);
  const locked3 = synth('locked3.pdf', '-password', OTHER_PASSWORD);
  const damaged = join(dir, 'damaged.pdf');
  writeFileSync(damaged, 'not a pdf');
  const big = join(dir, 'big.pdf');
  writeFileSync(big, Buffer.alloc(21 * 1024 * 1024));
  const { page, seen } = await open();
  await page.setInputFiles('#files', [good, damaged, big, good, locked1, locked2, locked3]);

  // A wrong password is refused, the right one opens the file and is reused
  // for the next file; a file can be skipped.
  await page.waitForSelector('#pw-form:not([hidden])');
  await page.fill('#pw-input', WRONG_PASSWORD);
  await page.click('#pw-form button[type=submit]');
  await page.waitForSelector('#pw-error:not([hidden])');
  await page.fill('#pw-input', TEST_PASSWORD);
  await page.click('#pw-form button[type=submit]');
  await page.waitForFunction(() => document.getElementById('pw-title').textContent.startsWith('locked3'));
  await page.click('#pw-skip');

  assert.deepEqual(await statuses(page), [
    'ok: — 1 payslip(s) read',
    'bad: — not a PDF, or a damaged one',
    'bad: — too large: payslips are well under 20 MB',
    'warn: — 1 payslip(s) read, 1 already read in another file and ignored',
    'ok: — 1 payslip(s) read',
    // Opened with the password remembered from the previous file; the
    // signed example repeats the classic one's amounts.
    'warn: — 1 payslip(s) read, 1 already read in another file and ignored',
    'bad: — protected by a password: skipped',
  ]);
  assert.deepEqual(seen.errors, []);
  onlyThisSite(seen);
  await page.close();
});

// Safari cannot iterate a ReadableStream with "for await", which pdf.js
// uses; compat.js fills it in. Removing it reproduces Safari in any engine.
test('works where streams cannot be iterated, as in Safari', async () => {
  const { page, seen } = await open({ before: () => {
    delete ReadableStream.prototype[Symbol.asyncIterator];
    delete ReadableStream.prototype.values;
  } });
  await page.setInputFiles('#files', [synth('safari.pdf')]);
  assert.deepEqual(await statuses(page), ['ok: — 1 payslip(s) read']);
  assert.deepEqual(seen.errors, []);
  await page.close();
});

test('a hidden report frame takes no space', async () => {
  const damaged = join(dir, 'only-damaged.pdf');
  writeFileSync(damaged, 'not a pdf');
  const { page } = await open();
  await page.setInputFiles('#files', [damaged]);
  await statuses(page);
  assert.equal(await page.$eval('#report', (f) => getComputedStyle(f).display), 'none');
  await page.close();
});

test('the report frame fits its content on a phone', async () => {
  // The frame once grew without end: the report measured the document,
  // which is never shorter than the frame, so each resize asked for more.
  const { page } = await open({ viewport: { width: 390, height: 800 } });
  await page.setInputFiles('#files', [synth('phone.pdf')]);
  await statuses(page);
  const frame = page.frames().find((f) => f !== page.mainFrame());
  await frame.waitForSelector('section.month:not([hidden]) .bar');
  const height = () => page.$eval('#report', (f) => parseInt(f.style.height, 10) || 0);
  await page.waitForFunction(() => parseInt(document.getElementById('report').style.height, 10) > 1000);
  const first = await height();
  await page.waitForTimeout(1500);
  assert.equal(await height(), first, 'the frame keeps its height');
  const content = await frame.evaluate(() => document.body.getBoundingClientRect().height);
  assert.ok(Math.abs(first - content) <= 4, `frame ${first}px for ${content}px of content`);
  // Content that grows later, such as text reflowed by a late font, gets room.
  await frame.evaluate(() => { document.querySelector('main').style.paddingBottom = '600px'; });
  await page.waitForFunction((h) => parseInt(document.getElementById('report').style.height, 10) > h, first);
  await page.close();
});

test('makes an anonymized copy that reads as the same payslip', async () => {
  const { page, seen } = await open();
  await page.setInputFiles('#files', [synth('BIANCHI_ANNA_10-2026.pdf')]);
  await statuses(page);
  await page.$eval('#files-box', (d) => { d.open = true; });
  await press(page, '.anon-open');
  await page.waitForSelector('#anon:not([hidden])');
  const text = await page.$eval('#anon-text', (e) => e.textContent);
  for (const secret of ['BIANCHI', 'ANNA', 'ESEMPIO', 'SINTETICO', '000042']) {
    assert.ok(!text.includes(secret), `the preview shows ${secret}`);
  }
  assert.match(text, /TOTALE/);
  assert.match(text, /1\.667,40/, 'amounts are kept by default');
  assert.equal(await page.$eval('#anon-title', (e) => e.textContent), 'Anonymized copy of BIANCHI_ANNA_10-2026.pdf');

  // The rebuilt PDF has a fixed name and nothing personal, and the app reads
  // it back as the same payslip.
  const [file] = await Promise.all([page.waitForEvent('download'), press(page, '#anon-download')]);
  assert.equal(file.suggestedFilename(), 'anonymized-payslip.pdf');
  const copy = join(dir, 'copy.pdf');
  await file.saveAs(copy);
  const bytes = readFileSync(copy, 'latin1');
  assert.ok(bytes.startsWith('%PDF-'));
  for (const secret of ['BIANCHI', 'ANNA', 'ESEMPIO', '000042', '/Info', '/Image']) {
    assert.ok(!bytes.includes(secret), `the copy contains ${secret}`);
  }

  // Without amounts, no amount is left.
  await press(page, '#anon-amounts');
  assert.doesNotMatch(await page.$eval('#anon-text', (e) => e.textContent), /1\.667,40/);
  assert.equal(await page.$eval('#anon-warning', (e) => e.hidden), true);
  await press(page, '#anon-close');
  assert.equal(await page.$eval('#anon', (e) => e.hidden), true);

  await press(page, '#restart');
  await page.setInputFiles('#files', [copy]);
  assert.deepEqual(await statuses(page), ['ok: — 1 payslip(s) read']);
  const frame = page.frames().find((f) => f !== page.mainFrame());
  await frame.waitForSelector('section.month:not([hidden]) .bar');
  assert.match(await frame.content(), /1\.667,40/);
  assert.deepEqual(seen.errors, []);
  onlyThisSite(seen);
  await page.close();
});

test('an unsupported browser gets a plain message', async () => {
  const page = await browser.newPage();
  await page.addInitScript(() => { delete window.WebAssembly; });
  await page.goto(base);
  await page.waitForFunction(() => document.getElementById('status').classList.contains('error'));
  assert.match(await page.$eval('#status', (s) => s.textContent), /cedolo non può funzionare in questo browser/);
  await page.close();
});

test('file names and contents never run as code', async () => {
  const hostile = synth('<img src=x onerror=alert(1)>.pdf');
  const { page, seen } = await open();
  await page.setInputFiles('#files', [hostile]);
  await statuses(page);
  assert.equal(await page.$eval('#file-list li span:nth-child(2)', (s) => s.textContent), '<img src=x onerror=alert(1)>.pdf');
  assert.equal(await page.$$eval('[onerror],[onload]', (els) => els.length), 0);
  assert.equal(seen.dialogs, 0);
  await page.close();
});

test('nothing reaches another site and nothing is stored', async () => {
  const { page } = await open();
  await page.setInputFiles('#files', [synth('one.pdf')]);
  await statuses(page);
  const result = await page.evaluate(async (other) => {
    const blocked = async (fn) => { try { await fn(); return false; } catch { return true; } };
    return {
      fetch: await blocked(() => fetch(other)),
      worker: await blocked(() => new Promise((ok, ko) => { const w = new Worker('pdfjs/pdf.worker.min.mjs', { type: 'module' }); w.onerror = ko; setTimeout(ok, 500); })),
      eval: await blocked(() => eval('1')),
      storage: localStorage.length + sessionStorage.length + (await indexedDB.databases()).length + (await caches.keys()).length + document.cookie.length,
    };
  }, other);
  assert.deepEqual(result, { fetch: true, worker: true, eval: true, storage: 0 });
  const frame = page.frames().find((f) => f !== page.mainFrame());
  const fromFrame = await frame.evaluate(async (other) => {
    const blocked = async (fn) => { try { await fn(); return false; } catch { return true; } };
    return { fetch: await blocked(() => fetch(other)), parent: await blocked(() => window.parent.document.body) };
  }, other);
  assert.deepEqual(fromFrame, { fetch: true, parent: true });
  assert.equal(outsideHits, 0);
  await page.close();
});
