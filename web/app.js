// cedolo web app: reads payslip PDFs with pdf.js and analyzes them with the
// Go core compiled to WebAssembly. Files are read in memory and never sent;
// nothing is stored, not even the chosen language or a password.
import './compat.js';
import * as pdfjs from './pdfjs/pdf.min.mjs';
import { layoutOf, hasText } from './layout.js';

// pdf.js parses PDFs in the page itself, not in a Web Worker: a worker
// loaded from a file does not inherit the page's Content-Security-Policy
// (it is set by a <meta> tag, as static hosting allows), so a flaw in the
// parser exploited by a crafted PDF could reach the network from there.
// worker-src 'none' in the policy keeps it that way.
globalThis.pdfjsWorker = await import('./pdfjs/pdf.worker.min.mjs');

// Motion is skipped for people who asked their system to reduce it.
const calm = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
const wait = (ms) => new Promise((r) => setTimeout(r, calm() ? 0 : ms));

// enter plays the entrance animation on elements just shown.
function enter(...els) {
  for (const el of els) {
    el.classList.remove('enter');
    void el.offsetWidth; // restart the animation
    el.classList.add('enter');
    el.addEventListener('animationend', () => el.classList.remove('enter'), { once: true });
  }
}

// Limits that keep huge, hostile or too many files from freezing the page.
const MAX_BYTES = 20 * 1024 * 1024;
const MAX_PAGES = 50;
const MAX_FILES = 60;

const $ = (id) => document.getElementById(id);
const state = {
  lang: navigator.language?.toLowerCase().startsWith('it') ? 'it' : 'en',
  texts: {},
  // {name, layout, error, refused: [key, ...args], read, repeated}
  files: [],
  // The report tab picked last, kept when the report is built again.
  tab: '',
  tooMany: false,
  report: '',
  // The file whose anonymized copy is open, and the copy's PDF bytes.
  anon: null,
  anonPDF: null,
  // Passwords that opened a file, tried first on the next protected one;
  // payslips from one employer usually share one. Memory only.
  passwords: [],
};

// t returns a text, filling %s and %d in order.
function t(key, ...args) {
  let i = 0;
  return (state.texts[key] ?? key).replace(/%[sd]/g, () => String(args[i++] ?? ''));
}

function applyTexts() {
  state.texts = JSON.parse(globalThis.cedolo.texts(state.lang));
  document.documentElement.lang = state.lang;
  document.title = t('web.title');
  for (const el of document.querySelectorAll('[data-t]')) el.textContent = t(el.dataset.t);
  for (const el of document.querySelectorAll('[data-t-aria]')) el.setAttribute('aria-label', t(el.dataset.tAria));
  for (const b of document.querySelectorAll('[data-lang]')) b.setAttribute('aria-pressed', String(b.dataset.lang === state.lang));
}

function status(text, error = false) {
  $('status').textContent = text;
  $('status').classList.toggle('error', error);
}

async function start() {
  try {
    const go = new globalThis.Go();
    const { instance } = await WebAssembly.instantiateStreaming(fetch(new URL('cedolo.wasm', import.meta.url)), go.importObject);
    go.run(instance);
  } catch (err) {
    status(`${window.cedoloUnsupported} (${err.message})`, true);
    return;
  }
  applyTexts();
  $('version').textContent = globalThis.cedolo.version;
  $('files').disabled = false;
  status('');
  window.cedoloReady = true;
}

// askPassword shows the password form for a file and resolves with the
// password typed, or null when the file is skipped.
function askPassword(name, wrong) {
  const form = $('pw-form');
  $('pw-title').textContent = t('web.password.title', name);
  $('pw-error').hidden = !wrong;
  $('pw-input').value = '';
  form.hidden = false;
  form.scrollIntoView({ block: 'nearest' });
  $('pw-input').focus();
  return new Promise((resolve) => {
    const done = (value) => {
      form.hidden = true;
      $('pw-input').value = '';
      form.removeEventListener('submit', onSubmit);
      $('pw-skip').removeEventListener('click', onSkip);
      resolve(value);
    };
    const onSubmit = (ev) => {
      ev.preventDefault(); // never submitted anywhere
      done($('pw-input').value);
    };
    const onSkip = () => done(null);
    form.addEventListener('submit', onSubmit);
    $('pw-skip').addEventListener('click', onSkip);
  });
}

async function readFile(file) {
  const entry = { name: file.name, layout: null, error: '', refused: null, read: 0, repeated: 0 };
  if (file.size > MAX_BYTES) {
    entry.refused = ['web.file.too_big', MAX_BYTES / 1024 / 1024];
    return entry;
  }
  const task = pdfjs.getDocument({
    data: new Uint8Array(await file.arrayBuffer()),
    // Only text is read: no fonts handed to the browser, no system fonts,
    // no forms or scripts, no canvas.
    disableFontFace: true,
    useSystemFonts: false,
    enableXfa: false,
    isOffscreenCanvasSupported: false,
    standardFontDataUrl: new URL('pdfjs/standard_fonts/', import.meta.url).href,
    cMapUrl: new URL('pdfjs/cmaps/', import.meta.url).href,
    cMapPacked: true,
    verbosity: 0,
  });
  // Known passwords first, then ask; a skipped file stops here.
  let untried = [...state.passwords];
  let tried = null;
  task.onPassword = async (update, reason) => {
    if (untried.length) {
      tried = untried.shift();
      update(tried);
      return;
    }
    const wrong = reason === pdfjs.PasswordResponses.INCORRECT_PASSWORD && tried !== null && !state.passwords.includes(tried);
    tried = await askPassword(file.name, wrong);
    if (tried === null) {
      entry.refused = ['web.file.locked'];
      task.destroy();
    } else {
      update(tried);
    }
  };
  try {
    const doc = await task.promise;
    if (tried !== null && !state.passwords.includes(tried)) state.passwords.push(tried);
    if (doc.numPages > MAX_PAGES) {
      entry.refused = ['web.file.too_long', MAX_PAGES];
    } else {
      entry.layout = await layoutOf(doc);
    }
  } catch (err) {
    if (entry.refused) {
      // skipped by the person
    } else if (err?.name === 'InvalidPDFException') {
      entry.refused = ['web.file.damaged'];
    } else {
      entry.error = err.message || String(err);
    }
  } finally {
    await task.destroy();
  }
  return entry;
}

function analyze() {
  const readable = state.files.filter((f) => f.layout && hasText(f.layout));
  const result = globalThis.cedolo.analyze(JSON.stringify(readable.map((f) => f.layout)), state.lang);
  if (result.error) throw new Error(result.error);
  readable.forEach((f, i) => {
    const r = result.perDocument[i];
    if (r.error) f.error = r.error;
    f.read = r.read ?? 0;
    f.repeated = r.repeated ?? 0;
    f.periods = Array.isArray(r.periods) ? r.periods.filter((p) => typeof p === 'string') : [];
  });
  state.report = result.report;
}

function describe(f) {
  if (f.refused) return ['bad', '!', t(...f.refused)];
  if (f.error) return ['bad', '!', t('web.file.error', f.error)];
  if (!hasText(f.layout)) return ['warn', '~', t('web.file.notext')];
  if (f.read === 0) return ['warn', '~', t('web.file.none')];
  if (f.repeated > 0) return ['warn', '~', t('web.file.repeated', f.read, f.repeated)];
  return ['ok', '✓', t('web.file.ok', f.read)];
}

function render() {
  const list = $('file-list');
  list.replaceChildren();
  let attention = false;
  for (const f of state.files) {
    const [cls, icon, note] = describe(f);
    attention ||= cls !== 'ok';
    const li = document.createElement('li');
    li.className = cls;
    const i = document.createElement('span');
    i.className = 'icon';
    i.textContent = icon;
    i.setAttribute('aria-hidden', 'true');
    const name = document.createElement('span');
    name.className = 'name';
    name.textContent = f.name;
    const n = document.createElement('span');
    n.className = 'note';
    n.textContent = `— ${note}`;
    const text = document.createElement('span');
    text.className = 'text';
    text.append(name, ' ', n);
    li.append(i, text);
    if (f.layout && hasText(f.layout)) {
      // Offered up front where nothing was read; elsewhere only after
      // "Something read badly?", to keep the list quiet.
      const b = document.createElement('button');
      b.type = 'button';
      b.className = f.read === 0 ? 'anon-open' : 'anon-open optional';
      b.textContent = t('web.anon.button');
      b.addEventListener('click', () => openAnon(f));
      li.append(b);
    }
    list.append(li);
  }
  const any = state.report !== '';
  // The files show only when one needs attention, or when picking the one
  // read badly.
  list.classList.remove('show-anon');
  $('files-title').textContent = t('web.files.attention');
  $('files-box').hidden = any && !attention;
  $('anon-ask').hidden = !state.files.some((f) => f.layout && hasText(f.layout) && f.read > 0);
  $('none').hidden = any;
  $('too-many').hidden = !state.tooMany;
  $('too-many').textContent = t('web.too_many', MAX_FILES);
  $('download').hidden = !any;
  $('report').hidden = !any;
  if (!any) $('tabs').hidden = true;
  if (any && $('report').srcdoc !== state.report) {
    // A new report tells its tabs once loaded; one payslip has none.
    $('tabs').hidden = true;
    // Until the report tells its height, a placeholder stands in for it,
    // so the page neither flashes an empty frame nor jumps.
    $('report').classList.add('pending');
    $('skeleton').hidden = false;
    $('report').srcdoc = state.report;
  }
  const first = $('result').hidden;
  $('intro').hidden = true;
  $('pick').hidden = true;
  $('result').hidden = false;
  if (first) {
    window.scrollTo({ top: 0, behavior: calm() ? 'auto' : 'smooth' });
    enter($('result'));
  }
}

// added confirms what a pick of files added to the report: the months
// read, or that none was new. Files with problems are listed above it.
function added(files) {
  const months = files.filter((f) => f.read > f.repeated).flatMap((f) => f.periods);
  let text = t('web.added.none');
  if (months.length === 1) text = t('web.added.one', months[0]);
  if (months.length > 1) text = t('web.added.many', months.length, months.join(', '));
  $('added').textContent = text;
  $('added').classList.toggle('none', months.length === 0);
  $('added').hidden = false;
}

// progress shows how many files have been read, or hides the bar.
function progress(done, total) {
  $('progress').hidden = total === 0;
  $('progress-bar').style.width = total ? `${Math.round((done / total) * 100)}%` : '0';
}

async function handle(fileList) {
  let files = [...fileList].filter((f) => f.type === 'application/pdf' || /\.pdf$/i.test(f.name));
  if (files.length === 0) return;
  // Files added to those already read count towards the same limit.
  const room = MAX_FILES - state.files.length;
  state.tooMany = files.length > room;
  files = files.slice(0, Math.max(room, 0));
  if (files.length === 0) {
    render();
    return;
  }
  // Adding to a report: the picker comes back above it while reading, for
  // the progress and any password.
  const adding = !$('result').hidden;
  const before = state.files.length;
  $('added').hidden = true;
  if (adding) {
    $('pick').hidden = false;
    $('pick').scrollIntoView({ block: 'nearest' });
  }
  $('drop').classList.add('busy');
  progress(0, files.length);
  try {
    for (const [i, file] of files.entries()) {
      status(t('web.reading', i + 1, files.length));
      state.files.push(await readFile(file));
      progress(i + 1, files.length);
    }
    analyze();
    status('');
    // The start page fades out before the results come in.
    if (!adding) $('intro').classList.add('leaving');
    $('pick').classList.add('leaving');
    await wait(180);
    render();
    if (adding) added(state.files.slice(before));
  } catch (err) {
    status(t('web.failed', err.message), true);
  } finally {
    // The same file can be picked again, to be told it was already read.
    $('files').value = '';
    $('drop').classList.remove('busy');
    $('intro').classList.remove('leaving');
    $('pick').classList.remove('leaving');
    progress(0, 0);
  }
}

// openAnon shows the anonymized copy of a file (ADR 0008): the text it
// contains, for review, and the rebuilt PDF to download.
function openAnon(f) {
  state.anon = f;
  const keep = $('anon-amounts').checked;
  const result = globalThis.cedolo.anonymize(JSON.stringify(f.layout), keep);
  state.anonPDF = result.error ? null : result.pdf;
  $('anon-title').textContent = t('web.anon.title', f.name);
  $('anon-text').textContent = result.error ? '' : result.text;
  $('anon-text').hidden = Boolean(result.error);
  $('anon-error').hidden = !result.error;
  if (result.error && result.error !== 'leak') $('anon-error').textContent = t('web.failed', result.error);
  $('anon-warning').hidden = !keep;
  $('anon-download').disabled = Boolean(result.error);
  $('anon').hidden = false;
  $('anon').scrollIntoView({ block: 'nearest' });
}

function closeAnon() {
  state.anon = null;
  state.anonPDF = null;
  $('anon-text').textContent = '';
  $('anon').hidden = true;
}

function downloadAnon() {
  if (!state.anonPDF) return;
  // A fixed name: the original's file name often holds a person's name.
  save(new Blob([state.anonPDF], { type: 'application/pdf' }), t('web.anon.file'));
}

function save(blob, name) {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
}

function download() {
  save(new Blob([state.report], { type: 'text/html' }), state.lang === 'it' ? 'report-cedolini.html' : 'payslip-report.html');
}

// The report's tabs sit in the bar at the top of the page, which stays in
// view as the page scrolls. The report says which tabs it has and which it
// shows; the page asks it for one. The tab picked stays picked when the
// report is built again with more payslips.
function tabsFrom(v) {
  const ok = Array.isArray(v?.names) && Array.isArray(v.labels) && v.names.length === v.labels.length && v.names.length <= 5 &&
    v.names.every((n) => typeof n === 'string' && /^[a-z]{1,20}$/.test(n)) &&
    v.labels.every((l) => typeof l === 'string' && l.length <= 40);
  if (!ok) return;
  const bar = $('tabs');
  bar.replaceChildren(...v.names.map((name, i) => {
    const b = document.createElement('button');
    b.type = 'button';
    b.setAttribute('role', 'tab');
    b.dataset.tab = name;
    b.textContent = v.labels[i];
    b.addEventListener('click', () => pickTab(name));
    b.addEventListener('keydown', (ev) => {
      const j = { ArrowRight: i + 1, ArrowLeft: i - 1, Home: 0, End: v.names.length - 1 }[ev.key];
      if (j === undefined) return;
      ev.preventDefault();
      const next = v.names[(j + v.names.length) % v.names.length];
      pickTab(next);
      bar.querySelector(`[data-tab="${next}"]`).focus();
    });
    return b;
  }));
  bar.hidden = false;
  const keep = v.names.includes(state.tab) ? state.tab : v.current;
  markTab(keep);
  if (keep !== v.current) $('report').contentWindow.postMessage({ cedoloShowTab: keep }, '*');
}

function markTab(name) {
  state.tab = name;
  for (const b of $('tabs').children) {
    const on = b.dataset.tab === name;
    b.setAttribute('aria-selected', String(on));
    b.tabIndex = on ? 0 : -1;
  }
}

function pickTab(name) {
  markTab(name);
  $('report').contentWindow.postMessage({ cedoloShowTab: name }, '*');
  // Back to the top of the report when it was scrolled past.
  const header = document.querySelector('header.bar').offsetHeight;
  const top = $('report').getBoundingClientRect().top + window.scrollY - header;
  if (window.scrollY > top) window.scrollTo({ top, behavior: calm() ? 'auto' : 'smooth' });
}

// The report frame tells its height, so the page scrolls as one.
window.addEventListener('message', (ev) => {
  if (ev.source === $('report').contentWindow && ev.data?.cedoloTabs) {
    tabsFrom(ev.data.cedoloTabs);
    return;
  }
  if (ev.source === $('report').contentWindow && typeof ev.data?.cedoloTab === 'string') {
    if ([...$('tabs').children].some((b) => b.dataset.tab === ev.data.cedoloTab)) markTab(ev.data.cedoloTab);
    return;
  }
  // The report asks to bring a month to the top: y is within the frame.
  if (ev.source === $('report').contentWindow && Number.isFinite(ev.data?.cedoloScroll)) {
    // The header stays on top of the page: land below it.
    const header = document.querySelector('header.bar')?.offsetHeight ?? 0;
    const top = $('report').getBoundingClientRect().top + window.scrollY + Math.max(0, ev.data.cedoloScroll) - header;
    window.scrollTo({ top, behavior: calm() ? 'auto' : 'smooth' });
    return;
  }
  if (ev.source === $('report').contentWindow && typeof ev.data?.cedoloHeight === 'number') {
    $('report').style.height = `${Math.min(Math.ceil(ev.data.cedoloHeight) + 2, 100000)}px`;
    if ($('report').classList.contains('pending')) {
      $('report').classList.remove('pending');
      $('skeleton').hidden = true;
      enter($('report'));
    }
  }
});

$('files').addEventListener('change', (ev) => handle(ev.target.files));
const drop = $('drop');
drop.addEventListener('dragover', (ev) => { ev.preventDefault(); drop.classList.add('over'); });
drop.addEventListener('dragleave', () => drop.classList.remove('over'));
drop.addEventListener('drop', (ev) => {
  ev.preventDefault();
  drop.classList.remove('over');
  if (!$('files').disabled && !drop.classList.contains('busy')) handle(ev.dataTransfer.files);
});
// Payslips can be added to a report: with its button, or dropped on it.
$('add').addEventListener('click', () => $('files').click());
// Files dragged anywhere over the page while a report is shown; the report
// frame stops taking pointer events meanwhile, so that it does not swallow
// the drop.
const result = $('result');
window.addEventListener('dragenter', () => { if (!result.hidden) result.classList.add('over'); });
window.addEventListener('dragleave', (ev) => { if (ev.relatedTarget === null) result.classList.remove('over'); });
window.addEventListener('dragover', (ev) => { if (!result.hidden) ev.preventDefault(); });
window.addEventListener('drop', (ev) => {
  result.classList.remove('over');
  if (result.hidden) return;
  ev.preventDefault();
  if (!drop.classList.contains('busy')) handle(ev.dataTransfer.files);
});
$('download').addEventListener('click', download);
$('anon-amounts').addEventListener('change', () => { if (state.anon) openAnon(state.anon); });
$('anon-download').addEventListener('click', downloadAnon);
$('anon-close').addEventListener('click', closeAnon);
$('anon-ask').addEventListener('click', () => {
  $('file-list').classList.add('show-anon');
  $('files-title').textContent = t('web.files.pick');
  $('files-box').hidden = false;
  $('anon-ask').hidden = true;
  $('files-box').scrollIntoView({ block: 'nearest', behavior: calm() ? 'auto' : 'smooth' });
});
for (const b of document.querySelectorAll('[data-lang]')) {
  b.addEventListener('click', () => {
    if (!globalThis.cedolo) return;
    state.lang = b.dataset.lang;
    applyTexts();
    $('added').hidden = true; // written in the other language
    if (state.files.length) {
      try {
        analyze();
        render();
        if (state.anon) openAnon(state.anon);
      } catch (err) {
        status(t('web.failed', err.message), true);
      }
    }
  });
}

status('…');
start();
