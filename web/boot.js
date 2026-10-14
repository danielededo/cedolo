// Shows a message when cedolo cannot start: a browser without WebAssembly
// or modern JavaScript, or a file that failed to load. It runs before the
// app and needs nothing from it, so its texts are inline in both languages.
(function () {
  var message = 'cedolo non può funzionare in questo browser: aggiornalo o prova con un altro browser recente. / ' +
    'cedolo cannot run in this browser: update it or try another recent browser.';
  window.cedoloUnsupported = message;
  function fail() {
    if (window.cedoloReady) return;
    var status = document.getElementById('status');
    if (status) {
      status.textContent = message;
      status.className = 'status error';
    }
  }
  if (typeof WebAssembly !== 'object') {
    document.addEventListener('DOMContentLoaded', fail);
    return;
  }
  window.addEventListener('error', function (ev) {
    // Errors in loading a script or module before the app is ready.
    if (!window.cedoloReady && (ev.target instanceof HTMLScriptElement || ev.message)) fail();
  }, true);
  setTimeout(fail, 20000);
})();
