// Cookie consent banner — GDPR / ePrivacy.
//
// On every page load we check localStorage for a saved decision.
// If neither "accepted" nor "declined" is set, we show the banner.
// Choice is sticky and survives across sessions in the same browser.

(function () {
  'use strict';

  const KEY = 'cookie-consent';

  function decide() {
    try { return localStorage.getItem(KEY); } catch (e) { return null; }
  }

  function persist(value) {
    try { localStorage.setItem(KEY, value); } catch (e) {}
  }

  const banner = document.getElementById('cookie-banner');
  if (!banner) return;

  // Already decided → don't show.
  const existing = decide();
  if (existing === 'accepted' || existing === 'declined') {
    banner.hidden = true;
    return;
  }

  // First visit → show.
  banner.hidden = false;

  banner.addEventListener('click', function (e) {
    var btn = e.target.closest('[data-cookie-action]');
    if (!btn) return;
    var action = btn.getAttribute('data-cookie-action');
    persist(action);
    banner.hidden = true;
    // Trigger a custom event so analytics scripts can opt-in.
    document.dispatchEvent(new CustomEvent('cookie-consent', {
      detail: { accepted: action === 'accepted' }
    }));
  });
})();
