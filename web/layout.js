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
    const items = content.items.filter((item) => item.str && item.str.trim());
    const tokens = [];
    const spaced = letterSpaced(items);
    for (const item of items) {
      tokens.push(...wordsOf(item, vp.height, spaced));
    }
    pages.push({ number: n, width: vp.width, height: vp.height, tokens });
    page.cleanup();
  }
  return { pages };
}

// letterSpaced tells whether a page prints its words a character at a time,
// as the Go core's layout.JoinSpaced decides: at least 20 items, two in five
// of a single character.
function letterSpaced(items) {
  const single = items.filter((item) => [...item.str.trim()].length === 1).length;
  return items.length >= 20 && single * 5 >= items.length * 2;
}

// spacedParts splits the text of an item on a letter-spaced page, where
// pdf.js puts single spaces between the letters of a word ("ES I", "o r e"):
// a single space next to a lone character does not split words without
// digits. A space next to a digit may be real ("IVS 3%"); the Go core joins
// the pieces of amounts. The spaces removed still count as characters for
// the positions.
function spacedParts(str) {
  const pieces = str.split(/(\s+)/);
  const parts = [];
  const lone = (s) => [...s].length === 1;
  const digit = (s) => /\d/.test(s);
  for (let k = 0; k < pieces.length; k++) {
    const p = pieces[k];
    const prev = parts[parts.length - 1];
    const next = pieces[k + 1];
    if (p === ' ' && prev && next && !digit(prev) && !digit(next) && (lone(prev) || lone(next))) {
      parts[parts.length - 1] += p + pieces[++k];
      continue;
    }
    parts.push(p);
  }
  return parts;
}

// wordsOf splits one pdf.js text item into word tokens; see spacedParts for
// letter-spaced pages.
export function wordsOf(item, pageHeight, spaced = false) {
  const [a, b, c, d, e, f] = item.transform;
  const vertical = Math.abs(b) > Math.abs(a);
  const size = Math.hypot(c, d) || item.height;
  const step = item.width / Math.max([...item.str].length, 1);
  const out = [];
  let i = 0;
  const parts = spaced ? spacedParts(item.str) : item.str.split(/(\s+)/);
  for (const part of parts) {
    const len = [...part].length;
    if (part.trim()) {
      const text = part.includes(' ') ? part.replaceAll(' ', '') : part;
      if (vertical) {
        // Text turned a quarter: words follow each other upwards.
        const bottom = pageHeight - (f + step * i);
        out.push({ text, x: e - size, y: bottom - step * len, w: size, h: step * len });
      } else {
        // The top sits about 0.8 of the font size above the baseline.
        out.push({ text, x: e + step * i, y: pageHeight - f - size * 0.8, w: step * len, h: size });
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
