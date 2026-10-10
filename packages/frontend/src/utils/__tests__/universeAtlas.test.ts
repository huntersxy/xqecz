import { afterEach, describe, expect, it, vi } from 'vitest'
import * as THREE from 'three'
import { createUniverseAtlas } from '../universeAtlas'
import type { UniverseNode } from '../artworkUniverse'

const nodes=(count:number):UniverseNode[]=>Array.from({length:count},(_,id)=>({artwork:{id,thumb:'/thumbs/'+id+'.webp'},x:id*200,y:0,radius:100,hue:30}))
afterEach(()=>vi.restoreAllMocks())
function setup() {
  vi.spyOn(HTMLCanvasElement.prototype,'getContext').mockReturnValue({clearRect:vi.fn(),drawImage:vi.fn()} as unknown as CanvasRenderingContext2D)
  const scene=new THREE.Scene(), atlas=createUniverseAtlas(scene)
  return {scene,atlas}
}
describe('batched artwork atlas',()=>{
  it('draws hundreds of fully resident artworks with one mesh per 16 instead of one per work',()=>{
    const {scene,atlas}=setup(); atlas.update(nodes(309))
    expect(scene.children).toHaveLength(20)
    const first=scene.children[0] as THREE.Mesh<THREE.BufferGeometry,THREE.MeshBasicMaterial>
    expect(first.geometry.getAttribute('position').count).toBe(16*4)
    expect(first.geometry.index!.count).toBe(16*6)
    const entry=atlas.get(0)!; expect(entry.node.radius).toBe(100)
    const tile=document.createElement('canvas');tile.width=tile.height=512
    const texture=first.material.map!, before=texture.version
    for(const node of nodes(16)) atlas.paint(node,tile,'/loaded/'+node.artwork.id)
    expect(texture.version).toBe(before)
    atlas.flush();expect(texture.version).toBe(before+1)
    atlas.flush();expect(texture.version).toBe(before+1)
    expect(atlas.imageSource(0)).toBe('/loaded/0')
    atlas.dispose();expect(scene.children).toHaveLength(0)
  })
  it('reuses vacant slots, updates coordinates and disposes each GPU resource',()=>{
    const {scene,atlas}=setup();atlas.update(nodes(16))
    const first=scene.children[0] as THREE.Mesh<THREE.BufferGeometry,THREE.MeshBasicMaterial>
    const disposeGeometry=vi.fn(),disposeMaterial=vi.fn(),disposeTexture=vi.fn()
    first.geometry.addEventListener('dispose',disposeGeometry)
    first.material.addEventListener('dispose',disposeMaterial)
    first.material.map!.addEventListener('dispose',disposeTexture)
    const revised=nodes(17).slice(1);revised[0]!.x=900
    atlas.update(revised)
    expect(atlas.get(0)).toBeUndefined();expect(atlas.get(1)!.node.x).toBe(900)
    expect(scene.children).toHaveLength(1)
    atlas.dispose()
    expect(disposeGeometry).toHaveBeenCalledOnce();expect(disposeMaterial).toHaveBeenCalledOnce();expect(disposeTexture).toHaveBeenCalledOnce()
  })
})
