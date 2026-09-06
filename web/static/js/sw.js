const CACHE_VERSION = "v34";
const SHELL_CACHE = `buffetflow-shell-${CACHE_VERSION}`;
const DATA_CACHE = `buffetflow-data-${CACHE_VERSION}`;
const SHELL = [
  "/static/offline.html",
  "/static/css/app.css?v=57",
  "/static/js/icons.js?v=2",
  "/static/js/app.js?v=15",
  "/static/js/offline.js?v=15",
  "/static/js/layout-division.js?v=1",
  "/static/js/layout-editor.js?v=38",
  "/static/icons/icon-192.png?v=3",
  "/static/icons/icon-512.png?v=3",
  "/static/icons/icon-512-maskable.png?v=3",
  "/static/icons/apple-touch-icon.png?v=3",
  "/static/icons/emenys-mark.png",
  "/static/icons/emenys-mark-ink.png",
  "/manifest.webmanifest"
];

self.addEventListener("install", (event) => {
  event.waitUntil(Promise.all([
    caches.open(SHELL_CACHE).then((cache) => cache.addAll(SHELL)),
    self.skipWaiting()
  ]));
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    const keys = await caches.keys();
    await Promise.all(keys.filter((key) => key.startsWith("buffetflow-") && ![SHELL_CACHE, DATA_CACHE].includes(key)).map((key) => caches.delete(key)));
    await self.clients.claim();
  })());
});

self.addEventListener("message", (event) => {
  if (event.data?.type === "SKIP_WAITING") self.skipWaiting();
});

self.addEventListener("sync", (event) => {
  if (event.tag !== "buffetflow-sync") return;
  event.waitUntil(self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => clients.forEach((client) => client.postMessage({ type: "SYNC_REQUESTED" }))));
});

async function cacheFirst(request) {
  const cached = await caches.match(request, { ignoreSearch: false });
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(SHELL_CACHE);
    await cache.put(request, response.clone());
  }
  return response;
}

async function networkFirst(request, navigation = false) {
  const cache = await caches.open(DATA_CACHE);
  try {
    const response = await fetch(request);
    const contentType = response.headers.get("Content-Type") || "";
    const isLogin = response.redirected && new URL(response.url).pathname === "/login";
    if (response.ok && !isLogin && (contentType.includes("text/html") || contentType.includes("application/json"))) {
      await cache.put(request, response.clone());
    }
    return response;
  } catch (_error) {
    const cached = await cache.match(request);
    if (cached) return cached;
    if (navigation) return (await caches.match("/static/offline.html")) || Response.error();
    return new Response(JSON.stringify({ offline: true, error: "Conteúdo indisponível sem conexão." }), { status: 503, headers: { "Content-Type": "application/json" } });
  }
}

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (url.pathname === "/api/health") {
    event.respondWith(fetch(request, { cache: "no-store" }).catch(() => new Response(JSON.stringify({ ok: false }), { status: 503, headers: { "Content-Type": "application/json" } })));
    return;
  }
  if (request.mode === "navigate") {
    event.respondWith(networkFirst(request, true));
    return;
  }
  if (url.pathname.startsWith("/static/") || url.pathname === "/manifest.webmanifest" || url.pathname === "/sw.js") {
    event.respondWith(cacheFirst(request));
    return;
  }
  if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/online") || url.pathname.startsWith("/offline") || url.pathname.startsWith("/events") || url.pathname.startsWith("/layouts") || url.pathname.startsWith("/inventory") || url.pathname.startsWith("/models") || url.pathname.startsWith("/catalog") || url.pathname.startsWith("/settings")) {
    event.respondWith(networkFirst(request));
  }
});
