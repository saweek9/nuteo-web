// Cookie consent banner — GDPR / ePrivacy.
//
// Three layers of persistence so the banner doesn't pop up repeatedly:
//   1. localStorage    — primary, survives across sessions (most browsers)
//   2. document.cookie — fallback when localStorage is blocked
//                          (Safari ITP in some modes, private browsing,
//                          Brave shields, file:// previews)
//   3. sessionStorage  — last-resort fallback so the banner doesn't
//                          reappear within the same session even when
//                          all persistent storage is blocked
//
// Also: cross-tab sync via the `storage` event so a decision in one
// tab immediately closes the banner in any other open tab.

(function () {
  'use strict';

  const STORAGE_KEY = 'cookie-consent';
  const COOKIE_NAME = 'cookie_consent';

  function readLS() {
    try { return localStorage.getItem(STORAGE_KEY); }
    catch (e) { return null; }
  }

  function writeLS(value) {
    try { localStorage.setItem(STORAGE_KEY, value); return true; }
    catch (e) { return false; }
  }

  function readCookie() {
    // document.cookie can throw in sandboxed iframes, file://, or contexts
    // where storage is locked down. Swallow + fall back so a single
    // broken context doesn't break the whole consent flow.
    try {
      const m = document.cookie.match(
        new RegExp('(?:^|; )' + COOKIE_NAME + '=([^;]*)')
      );
      return m ? decodeURIComponent(m[1]) : null;
    } catch (e) {
      return null;
    }
  }

  function writeCookie(value) {
    try {
      const oneYear = 60 * 60 * 24 * 365;
      document.cookie =
        COOKIE_NAME + '=' + encodeURIComponent(value) +
        '; max-age=' + oneYear +
        '; path=/; SameSite=Lax';
    } catch (e) { /* storage locked down — give up quietly */ }
  }

  function readSS() {
    try { return sessionStorage.getItem(STORAGE_KEY); }
    catch (e) { return null; }
  }

  function writeSS(value) {
    try { sessionStorage.setItem(STORAGE_KEY, value); }
    catch (e) { /* sessionStorage blocked — give up quietly */ }
  }

  function readDecision() {
    return readLS() || readCookie() || readSS();
  }

  function persist(value) {
    const ls = writeLS(value);
    writeCookie(value);        // always try cookie too (extra durability)
    writeSS(value);            // session fallback
    if (!ls) {
      if (window.console && console.info) {
        console.info('[nuteo] cookie: localStorage blocked, ' +
          'falling back to document.cookie + sessionStorage');
      }
    }
  }

  const banner = document.getElementById('cookie-banner');
  if (!banner) return;

  // Already decided → don't show.
  const existing = readDecision();
  if (existing === 'accept' || existing === 'decline') {
    banner.hidden = true;
    if (existing === 'accept') {
      document.dispatchEvent(new CustomEvent('cookie-consent', {
        detail: { accepted: true, restored: true }
      }));
    }
    return;
  }

  // First visit → show.
  banner.hidden = false;

  banner.addEventListener('click', function (e) {
    const btn = e.target.closest('[data-cookie-action]');
    if (!btn) return;
    const action = btn.getAttribute('data-cookie-action');
    persist(action);
    banner.hidden = true;
    document.dispatchEvent(new CustomEvent('cookie-consent', {
      detail: { accepted: action === 'accepted' }
    }));
  });

  // Cross-tab sync: if user accepts in another tab, hide here too.
  window.addEventListener('storage', function (e) {
    if (e.key !== STORAGE_KEY) return;
    if (!e.newValue) return;
    banner.hidden = true;
  });
})();
