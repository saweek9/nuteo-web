// Theme toggle + mobile menu + FAQ accordion + scroll-reveal animations.

(function () {
  'use strict';

  // ===== Scroll-reveal animations =====
  // Uses IntersectionObserver to add `.in-view` class to elements with
  // `[data-reveal]` when they enter the viewport, triggering the
  // CSS animation defined in site.css. Honors `prefers-reduced-motion`.
  function initReveal() {
    var els = document.querySelectorAll('[data-reveal]');
    if (!els.length) return;
    var prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced || !('IntersectionObserver' in window)) {
      els.forEach(function (e) { e.classList.add('in-view'); });
      return;
    }
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          entry.target.classList.add('in-view');
          io.unobserve(entry.target);
        }
      });
    }, { threshold: 0.1, rootMargin: '0px 0px -50px 0px' });
    els.forEach(function (el) { io.observe(el); });
  }
  initReveal();

  // ===== Service Worker =====
  // Register /sw.js so the browser can claim it as a service worker.
  // We only register on pages (not on /sw.js itself) and skip when
  // unsupported (very old browsers).
  if ('serviceWorker' in navigator && location.protocol === 'https:') {
    // Don't register from /sw.js (would cause infinite reload).
    if (!location.pathname.endsWith('/sw.js')) {
      navigator.serviceWorker.register('/sw.js').catch(function (err) {
        console.warn('[nuteo] SW registration failed:', err);
      });
    }
  }

  // ===== Theme toggle (light / dark / system) =====
  const THEME_KEY = 'nuteo-theme';
  const themeBtn = document.querySelector('.theme-toggle');
  const root = document.documentElement;

  function applyTheme(theme) {
    // theme: 'light' | 'dark' | 'system'
    if (theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      root.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
      root.setAttribute('data-theme', theme);
    }
  }

  // Read saved choice, fall back to 'system'
  let savedTheme;
  try {
    savedTheme = localStorage.getItem(THEME_KEY) || 'system';
  } catch (e) {
    savedTheme = 'system';
  }
  applyTheme(savedTheme);

  if (themeBtn) {
    themeBtn.addEventListener('click', function () {
      // Cycle: system → light → dark → system
      const cur = root.getAttribute('data-theme');
      let next;
      if (savedTheme === 'system') next = 'light';
      else if (savedTheme === 'light') next = 'dark';
      else next = 'system';

      // Toggle the actual data-theme immediately so the UI feels snappy.
      applyTheme(next === 'system'
        ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
        : next);

      savedTheme = next;
      try { localStorage.setItem(THEME_KEY, next); } catch (e) {}
    });
  }

  // ===== Mobile menu =====
  const mobileBtn = document.querySelector('.mobile-menu-btn');
  const siteNav = document.getElementById('primary-nav');
  if (mobileBtn && siteNav) {
    mobileBtn.setAttribute('aria-expanded', 'false');
    mobileBtn.setAttribute('aria-controls', 'primary-nav');
    mobileBtn.setAttribute('aria-label', 'Open menu');

    function closeMenu() {
      siteNav.classList.remove('is-open');
      mobileBtn.setAttribute('aria-expanded', 'false');
      mobileBtn.setAttribute('aria-label', 'Open menu');
    }
    function openMenu() {
      siteNav.classList.add('is-open');
      mobileBtn.setAttribute('aria-expanded', 'true');
      mobileBtn.setAttribute('aria-label', 'Close menu');
    }

    mobileBtn.addEventListener('click', function () {
      if (siteNav.classList.contains('is-open')) closeMenu();
      else openMenu();
    });
    // Close on link click
    siteNav.querySelectorAll('a').forEach(function (a) {
      a.addEventListener('click', closeMenu);
    });
    // Close on Escape key (a11y)
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && siteNav.classList.contains('is-open')) {
        closeMenu();
        mobileBtn.focus();
      }
    });
    // Close when window resized past mobile breakpoint
    window.addEventListener('resize', function () {
      if (window.innerWidth > 1000 && siteNav.classList.contains('is-open')) {
        closeMenu();
      }
    });
  }

  // ===== Newsletter form: progressive enhancement for HTMX =====
  // HTMX fires hx:configRequest and hx:afterRequest events. We add
  // a `class="sending"` during submit so CSS can dim the form, and
  // toggle the loading skeleton on.
  document.querySelectorAll('.newsletter-form').forEach(function (f) {
    var pending = document.querySelector('.newsletter-pending');
    f.addEventListener('htmx:beforeRequest', function () {
      f.classList.add('is-sending');
      var btn = f.querySelector('button[type="submit"]');
      if (btn) btn.disabled = true;
      if (pending) {
        pending.hidden = false;
        pending.textContent = '\u00a0'; // nbsp so shimmer has height
      }
    });
    f.addEventListener('htmx:afterRequest', function () {
      f.classList.remove('is-sending');
      var btn = f.querySelector('button[type="submit"]');
      if (btn) btn.disabled = false;
      if (pending) pending.hidden = true;
    });
  });

})();
