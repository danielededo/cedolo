// cedolo web app: reads payslip PDFs with pdf.js and analyzes them with the
// Go core compiled to WebAssembly. Files are read in memory and never sent;
// nothing is stored, not even the chosen language.
import * as pdfjs from './pdfjs/pdf.min.mjs';
import { layoutOf, hasText } from './layout.js';

// pdf.js parses PDFs in the page itself, not in a Web Worker: a worker
// loaded from a file does not inherit the page's Content-Security-Policy
// (it is set by a <meta> tag, as static hosting allows), so a flaw in the
// parser exploited by a crafted PDF could reach the network from there.
// worker-src 'none' in the policy keeps it that way.
globalThis.pdfjsWorker = await import('./pdfjs/pdf.worker.min.mjs');

// Limits that keep a huge or hostile file from freezing the page.
const MAX_BYTES = 20 * 1024 * 1024;
const MAX_PAGES = 50;

const $ = (id) => document.getElementById(id);
const state = {
  lang: navigator.language?.toLowerCase().startsWith('it') ? 'it' : 'en',
  texts: {},
  files: [], // {name, layout, error, count}
  report: '',
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
    status(`cedolo: ${err.message}`, true);
    return;
  }
  applyTexts();
  $('version').textContent = globalThis.cedolo.version;
  $('files').disabled = false;
  status('');
}

async function readFile(file) {
  const entry = { name: file.name, layout: null, error: '', refused: null, count: 0 };
  if (file.size > MAX_BYTES) {
    entry.refused = ['web.file.too_big', MAX_BYTES / 1024 / 1024];
    return entry;
  }
  try {
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
    try {
      const doc = await task.promise;
      if (doc.numPages > MAX_PAGES) {
        entry.refused = ['web.file.too_long', MAX_PAGES];
      } else {
        entry.layout = await layoutOf(doc);
      }
    } finally {
      await task.destroy();
    }
  } catch (err) {
    entry.error = err.message || String(err);
  }
  return entry;
}

function analyze() {
  const readable = state.files.filter((f) => f.layout && hasText(f.layout));
  const result = globalThis.cedolo.analyze(JSON.stringify(readable.map((f) => f.layout)), state.lang);
  if (result.error) throw new Error(result.error);
  readable.forEach((f, i) => { f.count = result.perDocument[i]; });
  state.report = result.report;
}

function render() {
  const list = $('file-list');
  list.replaceChildren();
  for (const f of state.files) {
    let cls, icon, note;
    if (f.refused) [cls, icon, note] = ['bad', '!', t(...f.refused)];
    else if (f.error) [cls, icon, note] = ['bad', '!', t('web.file.error', f.error)];
    else if (!hasText(f.layout)) [cls, icon, note] = ['warn', '~', t('web.file.notext')];
    else if (f.count === 0) [cls, icon, note] = ['warn', '~', t('web.file.none')];
    else [cls, icon, note] = ['ok', '✓', t('web.file.ok', f.count)];
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
  $('files-box').open = !any || state.files.some((f) => f.error || f.refused || f.count === 0);
  $('none').hidden = any;
  $('download').hidden = !any;
  $('report').hidden = !any;
  if (any) $('report').srcdoc = state.report;
  $('pick').hidden = true;
  $('result').hidden = false;
}

async function handle(fileList) {
  const files = [...fileList].filter((f) => f.type === 'application/pdf' || /\.pdf$/i.test(f.name));
  if (files.length === 0) return;
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
    $('report').style.height = `${Math.ceil(ev.data.cedoloHeight) + 2}px`;
  }
});

$('files').addEventListener('change', (ev) => handle(ev.target.files));
const drop = $('drop');
drop.addEventListener('dragover', (ev) => { ev.preventDefault(); drop.classList.add('over'); });
drop.addEventListener('dragleave', () => drop.classList.remove('over'));
drop.addEventListener('drop', (ev) => {
  ev.preventDefault();
  drop.classList.remove('over');
  if (!$('files').disabled) handle(ev.dataTransfer.files);
});
$('restart').addEventListener('click', restart);
$('download').addEventListener('click', download);
for (const b of document.querySelectorAll('[data-lang]')) {
  b.addEventListener('click', () => {
    if (!globalThis.cedolo) return;
    state.lang = b.dataset.lang;
    applyTexts();
    if (state.files.length) {
      analyze();
      render();
    }
  });
}

status('…');
start();
