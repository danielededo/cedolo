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
  tooMany: false,
  report: '',
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
    const { instance } = await WebAssembly.instantiateStreaming(fetch('cedolo.wasm'), go.importObject);
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
    standardFontDataUrl: './pdfjs/standard_fonts/',
    cMapUrl: './pdfjs/cmaps/',
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
    name.textContent = f.name;
    const n = document.createElement('span');
    n.className = 'note';
    n.textContent = `— ${note}`;
    li.append(i, name, n);
    list.append(li);
  }
  const any = state.report !== '';
  $('files-box').open = !any || attention;
  $('none').hidden = any;
  $('too-many').hidden = !state.tooMany;
  $('too-many').textContent = t('web.too_many', MAX_FILES);
  $('download').hidden = !any;
  $('report').hidden = !any;
  if (any) $('report').srcdoc = state.report;
  $('pick').hidden = true;
  $('result').hidden = false;
}

async function handle(fileList) {
  let files = [...fileList].filter((f) => f.type === 'application/pdf' || /\.pdf$/i.test(f.name));
  if (files.length === 0) return;
  state.tooMany = files.length > MAX_FILES;
  files = files.slice(0, MAX_FILES);
  $('drop').classList.add('busy');
  try {
    for (const [i, file] of files.entries()) {
      status(t('web.reading', i + 1, files.length));
      state.files.push(await readFile(file));
    }
    analyze();
    status('');
    render();
  } catch (err) {
    status(t('web.failed', err.message), true);
  } finally {
    $('drop').classList.remove('busy');
  }
}

function restart() {
  state.files = [];
  state.report = '';
  state.tooMany = false;
  state.passwords = [];
  $('report').removeAttribute('srcdoc');
  $('files').value = '';
  $('result').hidden = true;
  $('pick').hidden = false;
  status('');
}

function download() {
  const blob = new Blob([state.report], { type: 'text/html' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = state.lang === 'it' ? 'report-cedolini.html' : 'payslip-report.html';
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
}

// The report frame tells its height, so the page scrolls as one.
window.addEventListener('message', (ev) => {
  if (ev.source === $('report').contentWindow && typeof ev.data?.cedoloHeight === 'number') {
    $('report').style.height = `${Math.min(Math.ceil(ev.data.cedoloHeight) + 2, 100000)}px`;
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
$('restart').addEventListener('click', restart);
$('download').addEventListener('click', download);
for (const b of document.querySelectorAll('[data-lang]')) {
  b.addEventListener('click', () => {
    if (!globalThis.cedolo) return;
    state.lang = b.dataset.lang;
    applyTexts();
    if (state.files.length) {
      try {
        analyze();
        render();
      } catch (err) {
        status(t('web.failed', err.message), true);
      }
    }
  });
}

status('…');
start();
