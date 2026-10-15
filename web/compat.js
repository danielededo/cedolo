// Fills gaps in browsers that cedolo supports but that lack recent APIs
// pdf.js relies on. Imported before pdf.js.

// Safari cannot iterate a ReadableStream with "for await", which pdf.js uses
// to read the text of a page and to decompress data.
if (typeof ReadableStream !== 'undefined' && !ReadableStream.prototype[Symbol.asyncIterator]) {
  const values = async function* values() {
    const reader = this.getReader();
    try {
      for (;;) {
        const { done, value } = await reader.read();
        if (done) return;
        yield value;
      }
    } finally {
      reader.releaseLock();
    }
  };
  Object.defineProperty(ReadableStream.prototype, Symbol.asyncIterator, { value: values, configurable: true, writable: true });
  if (!ReadableStream.prototype.values) {
    Object.defineProperty(ReadableStream.prototype, 'values', { value: values, configurable: true, writable: true });
  }
}
