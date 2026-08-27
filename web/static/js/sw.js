// nuteo-web service worker — network-first for HTML, cache-first
// for static assets. PWA-ready: cache core static assets on install
// so the site works offline after first visit.

const CACHE_VERSION = 'nuteo-v1';
const STATIC_CACHE = `${CACHE_VERSION}-static`;
const RUNTIME_CACHE = `${CACHE_VERSION}-runtime`;

// Critical assets to pre-cache on install (so the site shell loads offline).
const PRECACHE_URLS = [
  '/',
  '/services',
  '/blog',
  '/contact',
  '/static/css/site.css',
  '/static/js/htmx.min.js',
  '/static/js/site.js',
  '/static/images/logo.svg',
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) =>
      cache.addAll(PRECACHE_URLS).catch((err) => {
        // Don't fail install just because a few assets 404'd.
        console.warn('[sw] precache partial:', err);
      })
    ).then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((k) => !k.startsWith(CACHE_VERSION))
                  .map((k) => caches.delete(k)))
    ).then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const req = event.request;
  if (req.method !== 'GET') return;

  const url = new URL(req.url);
  // Only handle same-origin requests.
  if (url.origin !== self.location.origin) return;

  // Skip API/POST endpoints.
  if (url.pathname.startsWith('/admin/') ||
      url.pathname.startsWith('/contact') ||
      url.pathname === '/newsletter') {
    return;
  }

  // Static assets: cache-first.
  if (url.pathname.startsWith('/static/') ||
      url.pathname.endsWith('.svg') ||
      url.pathname === '/favicon.ico') {
    event.respondWith(cacheFirst(req));
    return;
  }

  // HTML pages: network-first, fall back to cache.
  event.respondWith(networkFirst(req));
});

async function cacheFirst(req) {
  const cached = await caches.match(req);
  if (cached) return cached;
  try {
    const resp = await fetch(req);
    if (resp.ok) {
      const cache = await caches.open(RUNTIME_CACHE);
      cache.put(req, resp.clone());
    }
    return resp;
  } catch (err) {
    return new Response('Offline', { status: 503 });
  }
}

async function networkFirst(req) {
  try {
    const resp = await fetch(req);
    if (resp.ok) {
      const cache = await caches.open(RUNTIME_CACHE);
      cache.put(req, resp.clone());
    }
    return resp;
  } catch (err) {
    const cached = await caches.match(req);
    if (cached) return cached;
    // Last-resort: offline fallback page.
    const offline = await caches.match('/');
    if (offline) return offline;
    return new Response('Offline', { status: 503 });
  }
}
