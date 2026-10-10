import * as THREE from 'three'
import type { UniverseNode } from './artworkUniverse'

const TILE = 512
const COLUMNS = 4
const CAPACITY = COLUMNS * COLUMNS

/** 作品仍保留 512px 的球面效果。16 颗共用一张图集与绘制调用，避免每颗单独提交 GPU。 */
export function createUniverseAtlas(scene: THREE.Scene) {
  const pages: { canvas: HTMLCanvasElement; context: CanvasRenderingContext2D; texture: THREE.CanvasTexture; geometry: THREE.BufferGeometry; material: THREE.MeshBasicMaterial; mesh: THREE.Mesh; ids: number[] }[] = []
  const entries = new Map<number, { page: number; slot: number; node: UniverseNode; url?: string; source?: string }>()
  const changed = new Set<number>()
  function newPage() {
    const canvas = document.createElement('canvas'); canvas.width = canvas.height = TILE * COLUMNS
    const context = canvas.getContext('2d')!
    const texture = new THREE.CanvasTexture(canvas); texture.colorSpace = THREE.SRGBColorSpace
    const geometry = new THREE.BufferGeometry()
    const material = new THREE.MeshBasicMaterial({ map: texture, transparent: true, depthWrite: false })
    const mesh = new THREE.Mesh(geometry, material); scene.add(mesh)
    pages.push({ canvas, context, texture, geometry, material, mesh, ids: [] })
    return pages.length - 1
  }
  function geometryFor(pageIndex: number) {
    const page = pages[pageIndex]!
    const positions: number[] = [], uv: number[] = [], indices: number[] = []
    page.ids.forEach(id => {
      const entry = entries.get(id); if (!entry) return
      const { x, y, radius: r } = entry.node
      const z = r + 1, base = positions.length / 3
      positions.push(x-r,y-r,z, x+r,y-r,z, x+r,y+r,z, x-r,y+r,z)
      const col = entry.slot % COLUMNS, row = Math.floor(entry.slot / COLUMNS)
      // 半像素内缩避免相邻图块在过滤时串色；不削减作品的绘制分辨率。
      const inset = .5 / (TILE * COLUMNS)
      const left = col/COLUMNS+inset, right=(col+1)/COLUMNS-inset
      const top=1-row/COLUMNS-inset, bottom=1-(row+1)/COLUMNS+inset
      uv.push(left,bottom, right,bottom, right,top, left,top)
      indices.push(base,base+1,base+2, base,base+2,base+3)
    })
    page.geometry.setAttribute('position', new THREE.Float32BufferAttribute(positions, 3))
    page.geometry.setAttribute('uv', new THREE.Float32BufferAttribute(uv, 2))
    page.geometry.setIndex(indices); page.geometry.computeBoundingSphere()
  }
  return {
    update(nodes: UniverseNode[]) {
      const current = new Set(nodes.map(node => node.artwork.id))
      entries.forEach((entry,id) => { if (!current.has(id)) {
        const page=pages[entry.page]!, x=entry.slot%COLUMNS*TILE, y=Math.floor(entry.slot/COLUMNS)*TILE
        page.context.clearRect(x,y,TILE,TILE); page.ids[entry.slot] = -1; entries.delete(id); changed.add(entry.page)
      } })
      for (const node of nodes) {
        const existing=entries.get(node.artwork.id)
        if (existing) { existing.node=node; continue }
        let pageIndex=pages.findIndex(page => page.ids.includes(-1) || page.ids.length<CAPACITY)
        if(pageIndex<0) pageIndex=newPage()
        const page=pages[pageIndex]!, hole=page.ids.indexOf(-1), slot=hole<0?page.ids.length:hole
        page.ids[slot]=node.artwork.id
        entries.set(node.artwork.id,{page:pageIndex,slot,node})
      }
      pages.forEach((_,index)=>geometryFor(index))
    },
    get: (id: number) => entries.get(id),
    paint(node: UniverseNode, tile: HTMLCanvasElement, source?: string) {
      const entry=entries.get(node.artwork.id); if (!entry) return
      const page=pages[entry.page]!, x=entry.slot%COLUMNS*TILE, y=Math.floor(entry.slot/COLUMNS)*TILE
      page.context.clearRect(x,y,TILE,TILE); page.context.drawImage(tile,x,y)
      entry.url=node.artwork.atlas_thumb || node.artwork.thumb; entry.source=source; changed.add(entry.page)
    },
    flush() { changed.forEach(index => { pages[index]!.texture.needsUpdate=true }); changed.clear() },
    imageSource: (id: number) => entries.get(id)?.source,
    dispose() { pages.forEach(page=>{ scene.remove(page.mesh); page.geometry.dispose(); page.material.dispose(); page.texture.dispose() }); pages.length=0; entries.clear(); changed.clear() },
  }
}
