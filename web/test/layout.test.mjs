// How pdf.js text items become word tokens; see layout.js.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { wordsOf } from '../layout.js';

const item = (str, x = 100) => ({ str, transform: [7, 0, 0, 7, x, 742], width: 4 * str.length, height: 7 });
const words = (str, spaced) => wordsOf(item(str), 842, spaced).map((t) => t.text);

test('spaces split an item into words', () => {
  assert.deepEqual(words('TOTALE LORDO'), ['TOTALE', 'LORDO']);
  assert.deepEqual(words('ES I'), ['ES', 'I']);
});

test('on a letter-spaced page, spaces between lone letters do not split', () => {
  assert.deepEqual(words('ES I', true), ['ESI']);
  assert.deepEqual(words('godu t a', true), ['goduta']);
  assert.deepEqual(words('S I L I T', true), ['SILIT']);
  assert.deepEqual(words("L '", true), ["L'"]);
});

test('on a letter-spaced page, words and numbers stay apart', () => {
  assert.deepEqual(words('IVS 3%', true), ['IVS', '3%']);
  assert.deepEqual(words('ESON IVS', true), ['ESON', 'IVS']);
  assert.deepEqual(words('0 , 00', true), ['0', ',', '00']);
});

test('removed spaces still count for the positions', () => {
  const [a] = wordsOf(item('o r e'), 842, true);
  assert.equal(a.x, 100);
  assert.equal(a.w, 20);
});
