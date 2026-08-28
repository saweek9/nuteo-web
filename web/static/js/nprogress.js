// Top progress bar — shows during HTMX navigation.

(function () {
  'use strict';

  const bar = document.getElementById('nprogress');
  if (!bar) return;

  // Pure CSS animation: bar grows from 0 to 80% then settles at 100%.
  // Use HTMX events (htmx:beforeRequest, htmx:afterRequest) to drive it.
  let timer = null;

  document.body.addEventListener('htmx:beforeRequest', function () {
    bar.classList.add('indeterminate');
    bar.classList.remove('hidden');
    bar.style.transform = 'scaleX(0)';
    // Start progress after a small delay so we don't flash for
    // very fast requests.
    timer = setTimeout(function () {
      bar.style.transition = 'transform 400ms ease';
      bar.style.transform = 'scaleX(0.7)';
    }, 80);
  });

  document.body.addEventListener('htmx:afterRequest', function () {
    if (timer) { clearTimeout(timer); timer = null; }
    bar.style.transition = 'transform 150ms ease';
    bar.style.transform = 'scaleX(1)';
    // Fade out after a moment.
    setTimeout(function () {
      bar.classList.remove('indeterminate');
      bar.classList.add('hidden');
      bar.style.transform = '';
      bar.style.transition = '';
    }, 350);
  });

  // Errors: snap to 100% then fade.
  document.body.addEventListener('htmx:responseError', finish);
  document.body.addEventListener('htmx:sendError', finish);
  document.body.addEventListener('htmx:timeout', finish);
  function finish() {
    if (timer) { clearTimeout(timer); timer = null; }
    bar.style.transition = 'transform 150ms ease';
    bar.style.transform = 'scaleX(1)';
    setTimeout(function () {
      bar.classList.remove('indeterminate');
      bar.classList.add('hidden');
      bar.style.transform = '';
      bar.style.transition = '';
    }, 350);
  }
})();
