const CACHE = 'xqecz-media-v1'
const MAX_ENTRIES = 600
const MEDIA_PATH = /\/(uploads|thumbs|images)\/([^/]+)$/

self.addEventListener('install', () => self.skipWaiting())

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    const names = await caches.keys()
    await Promise.all(names.filter((n) => n.startsWith('xqecz-media-') && n !== CACHE).map((n) => caches.delete(n)))
    await self.clients.claim()
  })())
})

function cacheKey(pathname) {
  const m = MEDIA_PATH.exec(pathname)
  return m ? new Request(`${self.location.origin}/__media__/${m[1]}/${m[2]}`) : null
}

async function trim(cache) {
  const keys = await cache.keys()
  await Promise.all(keys.slice(0, Math.max(0, keys.length - MAX_ENTRIES)).map((k) => cache.delete(k)))
}

function storable(res) {
  return res.status === 200 && (res.headers.get('content-type') || '').startsWith('image/')
}

async function fetchAndStore(event, key) {
  const req = event.request
  const sameOrigin = new URL(req.url).origin === self.location.origin
  let res
  try {
    res = await fetch(req.url, { mode: sameOrigin ? 'same-origin' : 'cors', credentials: 'omit' })
  } catch (err) {
    if (sameOrigin) throw err
    return fetch(req)
  }
  if (storable(res)) {
    const copy = res.clone()
    event.waitUntil((async () => {
      const cache = await caches.open(CACHE)
      await cache.put(key, copy)
      await trim(cache)
    })())
  }
  return res
}

async function serve(event, key) {
  const cache = await caches.open(CACHE)
  const hit = await cache.match(key)
  return hit || fetchAndStore(event, key)
}

self.addEventListener('fetch', (event) => {
  const req = event.request
  if (req.method !== 'GET' || req.destination !== 'image' || req.headers.has('range')) return
  const key = cacheKey(new URL(req.url).pathname)
  if (!key || new URL(req.url).search) return
  event.respondWith(serve(event, key))
})
