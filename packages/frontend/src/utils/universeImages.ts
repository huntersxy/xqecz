import { loadPlanetImage } from './artworkPlanet'
import type { UniverseNode } from './artworkUniverse'

export interface UniverseImageProgress { total: number; ready: number; failed: number; pending: number }
type Entry = { url: string; state: 'pending' | 'ready' | 'failed' }

/**
 * 体验优先：展开期间预加载全作品缩略图，不按视口限量，不淘汰已完成图片。
 * 相同 URL 共享一次请求；取消和过期结果检查属于正确性，不能随性能限制一起移除。
 */
export function createUniverseImages(onImage: (node: UniverseNode, image: HTMLImageElement | null) => void, onProgress: (progress: UniverseImageProgress) => void) {
  const abort = new AbortController()
  const entries = new Map<number, Entry>()
  const requests = new Map<string, Promise<HTMLImageElement | null>>()
  let disposed = false
  function progress() {
    if (disposed) return
    const states = [...entries.values()]
    onProgress({ total: states.length, ready: states.filter(entry => entry.state === 'ready').length, failed: states.filter(entry => entry.state === 'failed').length, pending: states.filter(entry => entry.state === 'pending').length })
  }
  return {
    update(nodes: readonly UniverseNode[]) {
      if (disposed) return
      const current = new Set(nodes.filter(node => node.artwork.thumb).map(node => node.artwork.id))
      entries.forEach((_, id) => { if (!current.has(id)) entries.delete(id) })
      for (const node of nodes) {
        const url = node.artwork.thumb
        if (!url || entries.get(node.artwork.id)?.url === url) continue
        const entry: Entry = { url, state: 'pending' }
        entries.set(node.artwork.id, entry)
        let request = requests.get(url)
        if (!request) {
          request = loadPlanetImage(url, abort.signal, 20000).catch(() => null)
          requests.set(url, request)
        }
        void request.then(image => {
          if (disposed || entries.get(node.artwork.id) !== entry) return
          entry.state = image ? 'ready' : 'failed'
          onImage(node, image)
          progress()
        })
      }
      progress()
    },
    dispose() { disposed = true; abort.abort(); entries.clear(); requests.clear() },
  }
}
