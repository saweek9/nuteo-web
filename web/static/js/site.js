/* Theme toggle + mobile menu + FAQ accordion JS */

(function () {
  'use strict';

  // ===== Theme toggle (light / dark / system) =====
  const THEME_KEY = 'nuteo-theme';
  const themeBtn = document.querySelector('.theme-toggle');
  const root = document.documentElement;

  function applyTheme(theme) {
    // theme: 'light' | 'dark' | 'system'
    if (theme === 'system') {
      root.removeAttribute('data-theme');
    } else {
      root.setAttribute('data-theme', theme);
    }
    if (themeBtn) {
      themeBtn.setAttribute('aria-label',
        theme === 'light' ? 'Switch to dark mode'
        : theme === 'dark' ? 'Switch to system mode'
        : 'Switch to light mode');
      themeBtn.setAttribute('title', themeBtn.getAttribute('aria-label'));
    }
  }

  let savedTheme;
  try { savedTheme = localStorage.getItem(THEME_KEY) || 'system'; }
  catch (e) { savedTheme = 'system'; }
  applyTheme(savedTheme);

  if (themeBtn) {
    themeBtn.addEventListener('click', () => {
      const current = root.getAttribute('data-theme') ||
        (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
      const next = current === 'light' ? 'dark'
                 : current === 'dark' ? 'system'
                 : 'light';
      try { localStorage.setItem(THEME_KEY, next); } catch (e) { /* ignore */ }
      applyTheme(next);
    });
  }

  // ===== Mobile menu toggle =====
  const menuBtn = document.querySelector('.mobile-menu-btn');
  const nav = document.querySelector('.site-nav');
  if (menuBtn && nav) {
    menuBtn.addEventListener('click', () => {
      const isOpen = nav.classList.toggle('is-open');
      menuBtn.setAttribute('aria-expanded', isOpen ? 'true' : 'false');
    });
    // Close on link click
    nav.addEventListener('click', (e) => {
      if (e.target.tagName === 'A' && nav.classList.contains('is-open')) {
        nav.classList.remove('is-open');
        menuBtn.setAttribute('aria-expanded', 'false');
      }
    });
  }

  // ===== Fade-up animation on scroll =====
  if ('IntersectionObserver' in window) {
    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('fade-up');
          observer.unobserve(entry.target);
        }
      });
    }, { threshold: 0.1, rootMargin: '0px 0px -50px 0px' });

    document.querySelectorAll('[data-animate]').forEach((el) => observer.observe(el));
  }

  // ===== Newsletter form (placeholder — real impl in Phase 3) =====
  const newsletterForm = document.querySelector('.newsletter-form');
  if (newsletterForm) {
    newsletterForm.addEventListener('submit', (e) => {
      e.preventDefault();
      const email = newsletterForm.querySelector('input[type="email"]').value;
      // Demo: just show success
      const msg = document.createElement('div');
      msg.className = 'alert success';
      msg.style.marginTop = '1rem';
      msg.textContent = `Thanks! Confirmation sent to ${email}.`;
      newsletterForm.replaceWith(msg);
    });
  }
})();
