// htmx-head — merge <head> tags between full page navigations.
//
// When hx-boost swaps only #main, the <head> (title, meta, lang
// attribute, etc.) stays old. This extension listens for htmx:afterSwap
// and merges new <head> content into the current <head>.
//
// We use a sibling invisible container to parse the swapped HTML
// without affecting the visible DOM.

(function () {
  'use strict';

  document.body.addEventListener('htmx:afterSwap', function (e) {
    // If the swap target is #main (page navigation), sync the head.
    if (e.detail.target && e.detail.target.id === 'main') {
      var xhr = e.detail.xhr;
      if (!xhr || !xhr.responseText) return;
      try {
        var doc = new DOMParser().parseFromString(
          xhr.responseText, 'text/html');
        var newHead = doc.head;
        var currentHead = document.head;
        // Replace title
        var newTitle = newHead.querySelector('title');
        if (newTitle) {
          var curTitle = currentHead.querySelector('title');
          if (curTitle) curTitle.textContent = newTitle.textContent;
          else currentHead.appendChild(newTitle);
        }
        // Update lang on <html>
        var newLang = doc.documentElement.lang;
        if (newLang) document.documentElement.lang = newLang;
      } catch (err) { /* ignore */ }
    }
  });
})();
