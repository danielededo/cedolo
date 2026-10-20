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
  // Every file has a line in the sheet, in the order picked.
  return page.$$eval('#sheet li[data-i]', (ls) => ls.sort((a, b) => a.dataset.i - b.dataset.i).map((l) => `${l.className}: — ${l.dataset.note}`));
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
  // The page says only how many files need a look; the files that were
  // read before make one line in the sheet.
  assert.equal(await page.$eval('#files-open', (b) => b.textContent), '3 files to look at ›');
  await press(page, '#files-open');
  await page.waitForSelector('#sheet[open]');
  assert.equal(await page.$eval('#problem-list .repeats summary', (s) => s.textContent), '2 files already read, ignored');
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
  await page.waitForFunction(() => parseInt(document.getElementById('report').style.height, 10) > 500);
  const first = await height();
  await page.waitForTimeout(1500);
  assert.equal(await height(), first, 'the frame keeps its height');
  // In the page the report drops its own footer: the page shows the
  // disclaimer once.
  assert.equal(await frame.$eval('footer', (f) => getComputedStyle(f).display), 'none');
  const content = await frame.evaluate(() => document.body.getBoundingClientRect().height);
  assert.ok(Math.abs(first - content) <= 4, `frame ${first}px for ${content}px of content`);
  // Content that grows later, such as text reflowed by a late font, gets room.
  await frame.evaluate(() => { document.querySelector('main').style.paddingBottom = '600px'; });
  await page.waitForFunction((h) => parseInt(document.getElementById('report').style.height, 10) > h, first);
  await page.close();
});

test('shows the latest of many months first', async () => {
  // Files picked in any order: the report opens on the latest month, the
  // one its picker marks.
  // August is in another style, so that months differ in height.
  const files = [9, 1, 5, 3, 7, 2, 8, 4, 6].map((m) => synth(`month-${m}.pdf`, '-month', String(m), ...(m === 8 ? ['-style', 'spaced'] : [])));
  const { page, seen } = await open({ viewport: { width: 390, height: 800 } });
  await page.setInputFiles('#files', files);
  await statuses(page);
  const frame = page.frames().find((f) => f !== page.mainFrame());
  await frame.waitForSelector('section.month:not([hidden]) .bar');
  // The frame is laid out hidden first; what counts is the report shown.
  await page.waitForSelector('#report:not(.pending)');
  const visible = await frame.$$eval('section.month', (ms) => ms.filter((m) => getComputedStyle(m).display !== 'none').map((m) => m.querySelector('.when').textContent));
  assert.equal(visible.length, 1);
  assert.match(visible[0], /^September 2026/);
  assert.equal(await frame.$eval('.picker button[aria-pressed="true"]', (b) => b.textContent), 'Sep 26');
  // On a phone the row of months is wider than the screen: it opens
  // scrolled to the marked month, not to January.
  const inView = await frame.$eval('.picker', (row) => {
    const b = row.querySelector('[aria-pressed="true"]').getBoundingClientRect();
    const r = row.getBoundingClientRect();
    return row.scrollWidth > row.clientWidth && b.left >= r.left && b.right <= r.right;
  });
  assert.ok(inView, 'the marked month is in view');
  // Picking a month in view leaves the row where it is, though the frame
  // changes height with the month shown.
  const left = await frame.$eval('.picker', (row) => row.scrollLeft);
  await press(frame, '.picker button[data-month="7"]');
  await frame.waitForSelector('section.month[data-month="7"]:not([hidden])');
  await page.waitForTimeout(300);
  assert.equal(await frame.$eval('.picker', (row) => row.scrollLeft), left);
  // The charts are in their own tab, reached from the tabs or from the
  // link under the month; arrows move between tabs.
  assert.ok(await frame.$eval('#panel-time', (p) => p.hidden));
  // Inside the app the tabs are the page's, in the bar that stays on top;
  // the report's own are hidden.
  assert.equal(await frame.$eval('.tabs', (t) => getComputedStyle(t).display), 'none');
  const tab = (name) => page.$eval(`#tabs [data-tab="${name}"]`, (t) => t.getAttribute('aria-selected'));
  assert.equal(await tab('month'), 'true');
  await press(page, '#tabs [data-tab="time"]');
  await page.waitForFunction(() => document.querySelector('#tabs [data-tab="time"]').getAttribute('aria-selected') === 'true');
  // The report gets the page's message a moment later.
  await frame.waitForSelector('#panel-time:not([hidden])');
  assert.ok(await frame.$eval('#panel-month', (p) => p.hidden));
  await page.focus('#tabs [data-tab="time"]');
  await page.keyboard.press('ArrowRight');
  // The payslips accrue TFR and leave: the tab after "Over time" is theirs.
  await frame.waitForSelector('#panel-accrued:not([hidden])');
  assert.equal(await tab('accrued'), 'true');
  // A sideways swipe moves between tabs, and the page's tabs follow.
  // Plain events carrying the touch lists: WebKit on a desktop has no Touch
  // constructor.
  const swipe = (dx) => frame.evaluate((dx) => {
    const el = document.querySelector('.panel:not([hidden]) h3');
    const fire = (type, touches, changed) => {
      const ev = new Event(type, { bubbles: true });
      Object.defineProperties(ev, { touches: { value: touches }, changedTouches: { value: changed } });
      el.dispatchEvent(ev);
    };
    fire('touchstart', [{ clientX: 200, clientY: 200 }], [{ clientX: 200, clientY: 200 }]);
    for (const f of [0.25, 0.5, 1]) fire('touchmove', [{ clientX: 200 + dx * f, clientY: 200 }], [{ clientX: 200 + dx * f, clientY: 200 }]);
    fire('touchend', [], [{ clientX: 200 + dx, clientY: 200 }]);
  }, dx);
  await swipe(150);
  await frame.waitForSelector('#panel-time:not([hidden])');
  await page.waitForFunction(() => document.querySelector('#tabs [data-tab="time"]').getAttribute('aria-selected') === 'true');
  // A column of the chart opens its month, marked in the row too. On a
  // phone the chart is the narrow one.
  assert.equal(await frame.$eval('#trend', (svg) => getComputedStyle(svg).display), 'none');
  await press(frame, '#trend-narrow .col-hit[data-i="2"]');
  await frame.waitForSelector('section.month[data-month="2"]:not([hidden])');
  assert.ok(!(await frame.$eval('#panel-month', (p) => p.hidden)), 'the column leads back to the month');
  await page.waitForFunction(() => document.querySelector('#tabs [data-tab="month"]').getAttribute('aria-selected') === 'true');
  assert.equal(await frame.$eval('.picker button[aria-pressed="true"]', (b) => b.dataset.month), '2');
  assert.equal(await frame.$eval('#trend-narrow .month.current', (t) => t.dataset.i), '2');
  // The arrows beside the month step to the one before.
  await press(frame, 'section.month[data-month="2"] .steps button[data-to="1"]');
  await frame.waitForSelector('section.month[data-month="1"]:not([hidden])');
  assert.equal(await frame.$eval('.picker button[aria-pressed="true"]', (b) => b.dataset.month), '1');
  assert.deepEqual(seen.errors, []);
  await page.close();
});

test('a touch on a chart shows its amounts under it', async () => {
  const files = [1, 2, 3].map((m) => synth(`touch-${m}.pdf`, '-month', String(m)));
  const { page, seen } = await open({ viewport: { width: 390, height: 800 } });
  await page.setInputFiles('#files', files);
  await statuses(page);
  await page.waitForSelector('#report:not(.pending)');
  const frame = page.frames().find((f) => f !== page.mainFrame());
  await press(page, '#tabs [data-tab="time"]');
  await frame.waitForSelector('#panel-time:not([hidden])');
  // A finger would cover the amounts beside it: they stay under the chart,
  // and the tab stays where it is. A finger lifted, as plain events: not
  // every browser here emulates touch inside a frame.
  await frame.$eval('#trend-narrow .col-hit[data-i="1"]', (hit) => {
    const box = hit.getBoundingClientRect();
    const at = { bubbles: true, pointerType: 'touch', clientX: box.x + box.width / 2, clientY: box.y + box.height / 2 };
    hit.dispatchEvent(new PointerEvent('pointerdown', at));
    hit.dispatchEvent(new PointerEvent('pointerup', at));
    hit.dispatchEvent(new MouseEvent('click', at));
  });
  await frame.waitForSelector('#trend-narrow ~ .tooltip.pinned:not([hidden])');
  assert.match(await frame.$eval('#trend-narrow ~ .tooltip', (t) => t.textContent), /Feb 26/);
  assert.ok(!(await frame.$eval('#panel-time', (p) => p.hidden)));
  assert.equal(await frame.$$eval('#trend-narrow .col.dim', (cs) => cs.length), 2);
  // Its button opens the month.
  await press(frame, '#trend-narrow ~ .tooltip .open');
  await frame.waitForSelector('section.month[data-month="1"]:not([hidden])');
  await page.waitForFunction(() => document.querySelector('#tabs [data-tab="month"]').getAttribute('aria-selected') === 'true');
  assert.deepEqual(seen.errors, []);
  await page.close();
});

test('shows one person at a time', async () => {
  // Two people, the first with two employers in one month.
  const files = [
    synth('rossi-1.pdf', '-month', '1', '-person', 'ROSSI MARIO', '-employer', 'ALFA SRL'),
    synth('rossi-2a.pdf', '-month', '2', '-person', 'ROSSI MARIO', '-employer', 'ALFA SRL'),
    synth('rossi-2b.pdf', '-month', '2', '-person', 'ROSSI MARIO', '-employer', 'BETA SRL'),
    synth('verdi-2.pdf', '-month', '2', '-person', 'VERDI LUCIA', '-employer', 'ALFA SRL'),
  ];
  const { page, seen } = await open({ viewport: { width: 390, height: 800 } });
  await page.setInputFiles('#files', files);
  await statuses(page);
  await page.waitForSelector('#report:not(.pending)');
  const frame = page.frames().find((f) => f !== page.mainFrame());
  assert.deepEqual(await frame.$$eval('.people button', (bs) => bs.map((b) => b.textContent)), ['Mario Rossi', 'Lucia Verdi']);
  // The month with two employers names them; the other does not need to.
  assert.deepEqual(await frame.$$eval('.picker button', (bs) => bs.map((b) => b.textContent)), ['Jan 26', 'Feb 26 · ALFA', 'Feb 26 · BETA']);
  // The other person's report replaces the first, and the page's tabs
  // still lead into it.
  await press(frame, '.people button[data-person="1"]');
  await frame.waitForSelector('.people button[data-person="1"][aria-pressed="true"]');
  assert.equal(await frame.$$eval('section.month', (ms) => ms.length), 1);
  assert.match(await frame.$eval('.who .sub', (e) => e.textContent), /February 2026/);
  await press(frame, '.people button[data-person="0"]');
  await press(page, '#tabs [data-tab="time"]');
  await frame.waitForSelector('#panel-time:not([hidden])');
  assert.equal(await frame.$$eval('section.month', (ms) => ms.length), 3);
  assert.deepEqual(seen.errors, []);
  await page.close();
});

test('adds payslips to a report', async () => {
  const [one, two, three] = [1, 2, 3].map((m) => synth(`add-${m}.pdf`, '-month', String(m)));
  const { page, seen } = await open();
  await page.setInputFiles('#files', [one, two]);
  await statuses(page);
  await page.waitForSelector('#report:not(.pending)');
  // The report offers to add files: a new month, and one already read.
  const [chooser] = await Promise.all([page.waitForEvent('filechooser'), page.click('#add')]);
  await chooser.setFiles([three, one]);
  await page.waitForFunction(() => document.querySelectorAll('#sheet li[data-i]').length === 4);
  assert.deepEqual(await statuses(page), [
    'ok: — 1 payslip(s) read',
    'ok: — 1 payslip(s) read',
    'ok: — 1 payslip(s) read',
    'warn: — 1 payslip(s) read, 1 already read in another file and ignored',
  ]);
  assert.ok(await page.$eval('#pick', (p) => p.hidden), 'the picker hides again');
  // The page tells which month was added.
  assert.equal(await page.$eval('#added', (p) => p.textContent), 'Added: March 2026.');
  const frame = page.frames().find((f) => f !== page.mainFrame());
  await frame.waitForFunction(() => document.querySelectorAll('.picker button').length === 3);
  // The report opens on the latest month again.
  assert.equal(await frame.$eval('.picker button[aria-pressed="true"]', (b) => b.textContent), 'Mar 26');
  // The confirmation goes by itself after a few seconds.
  await page.waitForSelector('#added', { state: 'hidden', timeout: 8000 });
  assert.deepEqual(seen.errors, []);
  onlyThisSite(seen);
  await page.close();
});

test('makes an anonymized copy that reads as the same payslip', async () => {
  const { page, seen } = await open();
  await page.setInputFiles('#files', [synth('BIANCHI_ANNA_10-2026.pdf')]);
  await statuses(page);
  // Every file was read: the page lists no file, and the files are in a
  // sheet of their own, opened from the link under the report.
  assert.ok(await page.$eval('#files-alert', (b) => b.hidden));
  assert.ok(!(await page.$eval('#sheet', (d) => d.open)));
  await press(page, '#anon-ask');
  await page.waitForSelector('#sheet[open]');
  assert.equal(await page.$eval('#file-list li .when', (e) => e.textContent), 'October 2026');
  await press(page, '#file-list .anon-open');
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
  assert.ok(!(await page.$eval('#sheet-lists', (e) => e.hidden)), 'back to the files');
  // The browser's back button closes the sheet, as it would a page.
  await page.goBack();
  await page.waitForFunction(() => !document.getElementById('sheet').open);

  // A fresh page reads the copy alone: nothing is kept across a reload.
  await page.reload();
  await page.waitForFunction(() => window.cedoloReady === true, null, { timeout: 30000 });
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
  assert.equal(await page.$eval('#file-list li .name', (s) => s.textContent), '<img src=x onerror=alert(1)>.pdf');
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
