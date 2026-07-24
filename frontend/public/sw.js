// Minimal, conservative service worker.
//
// Deliberately NOT caching hashed build assets (to avoid ever serving stale
// JS/CSS). It only precaches a tiny offline shell and answers *navigation*
// requests network-first, falling back to the offline page when the network is
// unavailable. Everything else passes straight through to the network.

const CACHE = 'tv-shell-v1';
const OFFLINE_URL = '/offline.html';
const PRECACHE = [OFFLINE_URL, '/icon.svg', '/manifest.webmanifest'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      .then((c) => c.addAll(PRECACHE))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  if (request.method !== 'GET' || request.mode !== 'navigate') return;
  event.respondWith(fetch(request).catch(() => caches.match(OFFLINE_URL).then((r) => r || Response.error())));
});

// Web Push: internal/push.Payload on the backend is {title, body, url?} --
// see backend/internal/push/push.go. A push event with no data (or
// malformed JSON) still shows a generic notification rather than silently
// doing nothing, since a push service is only allowed to deliver a push at
// all if the page promises to show *something* (browsers penalize/revoke
// "silent push" behavior).
self.addEventListener('push', (event) => {
  let payload = {};
  try {
    if (event.data) payload = event.data.json();
  } catch {
    // Malformed payload -- fall through to the generic notification below.
  }
  const title = payload.title || 'Teamverwaltung';
  const options = {
    body: payload.body || '',
    icon: '/icon.svg',
    badge: '/icon.svg',
    data: { url: payload.url || '/' },
  };
  event.waitUntil(self.registration.showNotification(title, options));
});

// Focuses an already-open app window/tab if one exists (regardless of its
// current path -- the app is a single-page app, so "open" already means the
// right origin), otherwise opens a new one at the notification's target URL.
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = (event.notification.data && event.notification.data.url) || '/';
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ('focus' in client) return client.focus();
      }
      if (self.clients.openWindow) return self.clients.openWindow(targetUrl);
      return undefined;
    }),
  );
});

// A push service may rotate a subscription's endpoint at any time (RFC 8030
// doesn't guarantee an endpoint is permanent) and fires this event when it
// does. Without re-registering, the backend keeps sending to the now-dead
// old endpoint until a delivery attempt eventually 404/410s and prunes it --
// silently losing push for this device in the meantime. VAPID_PUBLIC_KEY
// isn't available inside the service worker's own scope (it's read from
// window.__RUNTIME_CONFIG__ by the page, not passed to the worker), so
// oldSubscription.options.applicationServerKey is reused instead of
// re-deriving it -- valid, since the key hasn't changed, only the endpoint.
self.addEventListener('pushsubscriptionchange', (event) => {
  const oldSub = event.oldSubscription;
  const applicationServerKey = oldSub && oldSub.options && oldSub.options.applicationServerKey;
  if (!applicationServerKey) return;
  event.waitUntil(
    self.registration.pushManager
      .subscribe({ userVisibleOnly: true, applicationServerKey })
      .then((sub) =>
        fetch('/api/v1/users/me/push-subscriptions', {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(sub.toJSON()),
        }),
      )
      .catch(() => {
        // Best-effort: if this fails, the old (now-dead) subscription still
        // gets pruned server-side on its next failed delivery attempt --
        // this handler is an optimization to avoid that lag, not the only
        // path to eventual consistency.
      }),
  );
});
