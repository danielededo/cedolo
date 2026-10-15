// Positioned words of a PDF read with pdf.js, in the layout JSON of the Go
// core (internal/layout): pages with tokens in top-left coordinates.
//
// pdf.js reports runs of text with the width of the whole run, not of each
// glyph, so words in a run get a share of its width by number of characters.
// The extractor reads columns by zones, which absorbs the difference; see
// docs/design/spike-browser-wasm.md.

// layoutOf returns the layout of an opened pdf.js document.
export async function layoutOf(doc) {
  const pages = [];
  for (let n = 1; n <= doc.numPages; n++) {
    const page = await doc.getPage(n);
    // Unrotated, as PDFium in the CLI reads it: a page stored turned for
    // display keeps the coordinates its text was written in.
    const vp = page.getViewport({ scale: 1, rotation: 0 });
    const content = await page.getTextContent();
    const tokens = [];
    for (const item of content.items) {
      if (!item.str || !item.str.trim()) continue;
      tokens.push(...wordsOf(item, vp.height));
    }
    pages.push({ number: n, width: vp.width, height: vp.height, tokens });
    page.cleanup();
  }
  return { pages };
}

// wordsOf splits one pdf.js text item into word tokens.
export function wordsOf(item, pageHeight) {
  const [a, b, c, d, e, f] = item.transform;
  const vertical = Math.abs(b) > Math.abs(a);
  const size = Math.hypot(c, d) || item.height;
  const step = item.width / Math.max([...item.str].length, 1);
  const out = [];
  let i = 0;
  for (const part of item.str.split(/(\s+)/)) {
    const len = [...part].length;
    if (part.trim()) {
      if (vertical) {
        // Text turned a quarter: words follow each other upwards.
        const bottom = pageHeight - (f + step * i);
        out.push({ text: part, x: e - size, y: bottom - step * len, w: size, h: step * len });
      } else {
        // The top sits about 0.8 of the font size above the baseline.
        out.push({ text: part, x: e + step * i, y: pageHeight - f - size * 0.8, w: step * len, h: size });
      }
    }
    i += len;
  }
  return out;
}

// hasText reports whether a layout holds any word, to tell scans apart.
export function hasText(layout) {
  return layout.pages.some((p) => p.tokens.length > 0);
}
