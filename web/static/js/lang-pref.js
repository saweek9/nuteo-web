// Language preference — sticky picker.
//
// Persists user's chosen language across visits/sessions so the
// choice "sticks" without bloating every URL with ?lang=...
//
// Two layers:
//   1. localStorage (primary) — survives across tabs/sessions
//   2. document.cookie (fallback) — for browsers that block
//      localStorage, or contexts that disable JS
//
// The page also reads ?lang=... (highest priority — for shareable
// URLs and the lang switcher buttons).

(function () {
  'use strict';

  const STORAGE_KEY = 'nuteo-lang';
  const COOKIE_NAME = 'nuteo_lang';
  const VALID = ['en', 'th'];

  function currentLang() {
    // URL query takes priority (shareable, explicit).
    const params = new URLSearchParams(location.search);
    const q = params.get('lang');
    if (q && VALID.includes(q)) return q;
    // localStorage next.
    try {
      const ls = localStorage.getItem(STORAGE_KEY);
      if (ls && VALID.includes(ls)) return ls;
    } catch (e) { /* ignore */ }
    // Cookie next.
    try {
      const m = document.cookie.match(
        new RegExp('(?:^|; )' + COOKIE_NAME + '=([^;]*)')
      );
      if (m) {
        const v = decodeURIComponent(m[1]);
        if (VALID.includes(v)) return v;
      }
    } catch (e) { /* ignore */ }
    return 'en';
  }

  function persist(lang) {
    if (!VALID.includes(lang)) return;
    // localStorage
    try { localStorage.setItem(STORAGE_KEY, lang); } catch (e) {}
    // Cookie (1 year, Lax)
    try {
      const oneYear = 60 * 60 * 24 * 365;
      document.cookie =
        COOKIE_NAME + '=' + encodeURIComponent(lang) +
        '; max-age=' + oneYear +
        '; path=/; SameSite=Lax';
    } catch (e) {}
  }

  // Click handler on the lang switcher — capture the chosen lang
  // and persist it BEFORE the link navigation completes.
  document.addEventListener('click', function (e) {
    const a = e.target.closest('a[href*="lang="]');
    if (!a) return;
    const href = a.getAttribute('href') || '';
    const m = href.match(/[?&]lang=([^&#]+)/);
    if (!m) return;
    const lang = decodeURIComponent(m[1]);
    if (VALID.includes(lang)) {
      persist(lang);
    }
  }, true);

  // Expose for tests / debugging.
  window.__nuteoLang = { current: currentLang, persist: persist };
})();
