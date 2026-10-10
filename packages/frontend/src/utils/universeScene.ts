import * as THREE from 'three'
import { MapControls } from 'three/addons/controls/MapControls.js'
import { createUniverseImages, type UniverseImageProgress } from './universeImages'
import { universeOpeningView, type UniverseNode } from './artworkUniverse'

/**
 * 拥有整个场景的渲染器、GPU 资源、监听器与加载任务；调用方卸载时必须 dispose。
 * 当前版本体验优先：预加载并保留全作品缩略图，不再限制可见贴图数和驻留数。
 * setActive 只暂停帧循环，dispose 才释放资源；图片地址一律交给 loadPlanetImage。
 */
export function createUniverseScene(canvas: HTMLCanvasElement, onSelect: (node: UniverseNode | undefined) => void, onProgress: (progress: UniverseImageProgress) => void = () => {}) {
  const renderer = new THREE.WebGLRenderer({ canvas, alpha: true, antialias: true })
  renderer.setPixelRatio(Math.min(devicePixelRatio, 2))
  const scene = new THREE.Scene()
  const camera = new THREE.OrthographicCamera(-1000, 1000, 1000, -1000, .1, 5000)
  camera.position.set(0, 0, 2000)
  const controls = new MapControls(camera, canvas)
  controls.screenSpacePanning = true
  controls.enableRotate = false
  controls.enableDamping = false
  controls.minZoom = .2
  controls.maxZoom = 14
  controls.enabled = false
  const geometry = new THREE.SphereGeometry(1, 32, 24)
  const material = new THREE.MeshStandardMaterial({ roughness: 1 })
  let planets: THREE.InstancedMesh | undefined
  let nodes: UniverseNode[] = []
  let size = 1200
  let width = 1
  let height = 1
  let disposed = false
  let active = false
  let dirty = true
  let frame = 0
  let lastDraw = 0
  const thumbnails = new Map<number, { mesh: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>; texture: THREE.CanvasTexture; url: string; source?: string }>()
  const plane = new THREE.PlaneGeometry(2, 2)
  const matrix = new THREE.Matrix4()
  const position = new THREE.Vector3()
  const rotation = new THREE.Quaternion()
  const scale = new THREE.Vector3()
  const color = new THREE.Color()
  const ray = new THREE.Raycaster()
  const pointer = new THREE.Vector2()
  const haloMaterial = new THREE.MeshBasicMaterial({ color: '#f6cf9a', transparent: true, opacity: .8, side: THREE.DoubleSide })
  const halo = new THREE.Mesh(new THREE.RingGeometry(1.17, 1.2, 64), haloMaterial)
  halo.visible = false
  scene.add(halo, new THREE.AmbientLight('#f5ebff', 2))
  const light = new THREE.DirectionalLight('#fff0d9', 3)
  light.position.set(-500, 800, 1500)
  scene.add(light)
  const starsGeometry = new THREE.BufferGeometry()
  const stars = new Float32Array(1100 * 3)
  for (let i = 0; i < stars.length; i += 3) {
    stars[i] = (Math.random() - .5) * 14000
    stars[i + 1] = (Math.random() - .5) * 14000
    stars[i + 2] = -200 - Math.random() * 600
  }
  starsGeometry.setAttribute('position', new THREE.BufferAttribute(stars, 3))
  const starsMaterial = new THREE.PointsMaterial({ color: '#e2ddeb', size: 2, transparent: true, opacity: .32, sizeAttenuation: false })
  scene.add(new THREE.Points(starsGeometry, starsMaterial))

  function selected(node: UniverseNode | undefined) {
    halo.visible = !!node
    if (node) { halo.position.set(node.x, node.y, node.radius + 8); halo.scale.setScalar(node.radius) }
    dirty = true
    onSelect(node)
  }
  function update(items: UniverseNode[]) {
    nodes = items
    if (planets) { scene.remove(planets); planets.dispose() }
    planets = new THREE.InstancedMesh(geometry, material, nodes.length)
    nodes.forEach((node, index) => {
      position.set(node.x, node.y, 0)
      scale.setScalar(node.radius)
      planets!.setMatrixAt(index, matrix.compose(position, rotation, scale))
      planets!.setColorAt(index, color.setHSL(node.hue / 360, .25, .39))
    })
    planets.computeBoundingSphere()
    scene.add(planets)
    const current = new Set(nodes.map(node => node.artwork.id))
    thumbnails.forEach((entry, id) => { if (!current.has(id)) release(id, entry) })
    nodes.forEach(node => {
      const entry = thumbnails.get(node.artwork.id)
      if (!entry || entry.url !== node.artwork.thumb) paintArtwork(node, null, !!node.artwork.thumb)
      else { entry.mesh.position.set(node.x, node.y, node.radius + 1); entry.mesh.scale.setScalar(node.radius) }
    })
    images.update(nodes)
    dirty = true
  }
  function release(id: number, entry: (typeof thumbnails extends Map<number, infer T> ? T : never)) {
    scene.remove(entry.mesh)
    entry.mesh.material.dispose()
    entry.texture.dispose()
    thumbnails.delete(id)
  }
  function resize() {
    width = canvas.clientWidth || innerWidth
    height = canvas.clientHeight || innerHeight
    renderer.setSize(width, height, false)
    const aspect = width / height
    camera.left = -size * aspect / 2
    camera.right = size * aspect / 2
    camera.top = size / 2
    camera.bottom = -size / 2
    camera.updateProjectionMatrix()
    dirty = true
  }
  function moveTo(x: number, y: number, viewHeight: number) {
    size = viewHeight
    camera.zoom = 1
    camera.position.set(x, y, 2000)
    controls.target.set(x, y, 0)
    resize()
    controls.update()
  }
  function enter() {
    const view = universeOpeningView(nodes, width / height)
    moveTo(view.x, view.y, view.height)
  }
  function fit() {
    if (!nodes.length) { moveTo(0, 0, 1200); return }
    const left = Math.min(...nodes.map(node => node.x - node.radius))
    const right = Math.max(...nodes.map(node => node.x + node.radius))
    const bottom = Math.min(...nodes.map(node => node.y - node.radius))
    const top = Math.max(...nodes.map(node => node.y + node.radius))
    moveTo((left + right) / 2, (bottom + top) / 2, Math.max(top - bottom, (right - left) / (width / height)) * 1.12 + 200)
  }
  function zoom(factor: number) {
    camera.zoom = THREE.MathUtils.clamp(camera.zoom * factor, controls.minZoom, controls.maxZoom)
    camera.updateProjectionMatrix()
    dirty = true
  }
  function focus(node: UniverseNode) {
    camera.position.set(node.x, node.y, 2000)
    controls.target.set(node.x, node.y, 0)
    camera.zoom = Math.max(camera.zoom, Math.min(8, size / 650))
    camera.updateProjectionMatrix()
    controls.update()
    selected(node)
  }
  function project(node: UniverseNode) {
    const point = new THREE.Vector3(node.x, node.y, 0).project(camera)
    return { x: (point.x + 1) * width / 2, y: (1 - point.y) * height / 2, diameter: node.radius * 2 * height / size * camera.zoom }
  }
  function paintArtwork(node: UniverseNode, image: HTMLImageElement | null, loading = false) {
    if (disposed) return
    const tile = document.createElement('canvas')
    tile.width = tile.height = 512
    const ctx = tile.getContext('2d')
    if (!ctx) return
    const glow = ctx.createRadialGradient(256, 256, 218, 256, 256, 256)
    glow.addColorStop(0, 'rgba(246,197,220,.26)'); glow.addColorStop(1, 'rgba(246,197,220,0)')
    ctx.fillStyle = glow; ctx.fillRect(0, 0, 512, 512)
    ctx.save(); ctx.beginPath(); ctx.arc(256, 256, 235, 0, Math.PI * 2); ctx.clip()
    const paper = ctx.createLinearGradient(0, 0, 512, 512)
    paper.addColorStop(0, image ? '#f6e8ec' : '#302838'); paper.addColorStop(1, image ? '#dbcedb' : '#514457')
    ctx.fillStyle = paper; ctx.fillRect(0, 0, 512, 512)
    if (image) {
      const crop = Math.min(image.naturalWidth, image.naturalHeight)
      ctx.drawImage(image, (image.naturalWidth - crop) / 2, (image.naturalHeight - crop) / 2, crop, crop, 20, 20, 472, 472)
      // 淡球面光影保留作品色彩；不像旧版重阴影把作品压成灰色小点。
      const shade = ctx.createRadialGradient(192, 155, 50, 256, 256, 246)
      shade.addColorStop(0, 'rgba(255,255,255,.12)'); shade.addColorStop(.68, 'rgba(0,0,0,0)'); shade.addColorStop(1, 'rgba(19,10,28,.42)')
      ctx.fillStyle = shade; ctx.fillRect(0, 0, 512, 512)
    } else {
      ctx.textAlign = 'center'; ctx.fillStyle = '#edd5e2'; ctx.font = '80px serif'
      ctx.fillText(node.artwork.thumb ? '✦' : '“', 256, 195)
      ctx.font = '28px sans-serif'
      const title = node.artwork.title || '一段宇宙来信'
      for (let line = 0; line < Math.min(3, Math.ceil(title.length / 9)); line++) ctx.fillText(title.slice(line * 9, line * 9 + 9), 256, 254 + line * 39)
      ctx.fillStyle = '#c7aebe'; ctx.font = '19px sans-serif'
      ctx.fillText(loading ? '正在显影 ···' : node.artwork.thumb ? '图片暂不可达' : '文字星球', 256, 392)
    }
    ctx.restore()
    ctx.beginPath(); ctx.arc(256, 256, 235, 0, Math.PI * 2)
    ctx.strokeStyle = 'rgba(245,222,236,.5)'; ctx.lineWidth = 2; ctx.stroke()
    const texture = new THREE.CanvasTexture(tile)
    texture.colorSpace = THREE.SRGBColorSpace
    const existing = thumbnails.get(node.artwork.id)
    if (existing) { existing.texture.dispose(); existing.texture = texture; existing.mesh.material.map = texture; existing.mesh.material.needsUpdate = true; existing.url = node.artwork.thumb; existing.source = image?.src }
    else {
      const mesh = new THREE.Mesh(plane, new THREE.MeshBasicMaterial({ map: texture, transparent: true, depthWrite: false }))
      mesh.position.set(node.x, node.y, node.radius + 1); mesh.scale.setScalar(node.radius)
      thumbnails.set(node.artwork.id, { mesh, texture, url: node.artwork.thumb, source: image?.src }); scene.add(mesh)
    }
    dirty = true
  }
  // 在开场动画期间就启动全部缩略图，拖动/缩放不卸载作品，不再出现 64 张的比例天花板。
  const images = createUniverseImages((node, image) => paintArtwork(node, image), onProgress)
  function draw(time: number) {
    if (disposed || !active || document.hidden) return
    frame = requestAnimationFrame(draw)
    if (time - lastDraw < 33) return
    lastDraw = time
    controls.update()
    if (dirty) { renderer.render(scene, camera); dirty = false }
  }
  function setActive(value: boolean) {
    active = value
    controls.enabled = value
    cancelAnimationFrame(frame)
    if (value && !document.hidden) { dirty = true; frame = requestAnimationFrame(draw) }
  }
  function visibility() { setActive(active) }
  function change() { dirty = true }
  function key(event: KeyboardEvent) {
    const delta = size / camera.zoom / 10
    const moves: Record<string, [number, number]> = { ArrowLeft: [-delta, 0], ArrowRight: [delta, 0], ArrowUp: [0, delta], ArrowDown: [0, -delta] }
    const move = moves[event.key]
    if (move) { event.preventDefault(); camera.position.x += move[0]; camera.position.y += move[1]; controls.target.x += move[0]; controls.target.y += move[1]; dirty = true }
    if (event.key === '+' || event.key === '=') zoom(1.3)
    if (event.key === '-') zoom(1 / 1.3)
    if (event.key === 'Home') fit()
  }
  let down = { x: 0, y: 0 }
  let moved = false
  function pointerDown(event: PointerEvent) { down = { x: event.clientX, y: event.clientY }; moved = false }
  function pointerMove(event: PointerEvent) { if (Math.hypot(event.clientX - down.x, event.clientY - down.y) > 5) moved = true }
  function pick(event: PointerEvent) {
    if (moved || !active || !planets) return
    const rect = canvas.getBoundingClientRect()
    pointer.set((event.clientX - rect.left) / rect.width * 2 - 1, 1 - (event.clientY - rect.top) / rect.height * 2)
    ray.setFromCamera(pointer, camera)
    const hit = ray.intersectObject(planets)[0]
    selected(hit?.instanceId === undefined ? undefined : nodes[hit.instanceId])
  }
  controls.addEventListener('change', change)
  canvas.addEventListener('pointerdown', pointerDown)
  canvas.addEventListener('pointermove', pointerMove)
  canvas.addEventListener('pointerup', pick)
  canvas.addEventListener('keydown', key)
  window.addEventListener('resize', resize)
  document.addEventListener('visibilitychange', visibility)
  resize()
  return {
    update, enter, fit, zoom, focus, project, setActive,
    // 预览复用成功加载的最终地址，避免缩略图已回退成功、详情卡却再次请求失效源。
    imageSource: (id: number) => thumbnails.get(id)?.source,
    dispose() {
      disposed = true; active = false; images.dispose(); cancelAnimationFrame(frame)
      window.removeEventListener('resize', resize)
      document.removeEventListener('visibilitychange', visibility)
      canvas.removeEventListener('pointerdown', pointerDown)
      canvas.removeEventListener('pointermove', pointerMove)
      canvas.removeEventListener('pointerup', pick)
      canvas.removeEventListener('keydown', key)
      controls.dispose()
      thumbnails.forEach((entry, id) => release(id, entry))
      planets?.dispose(); geometry.dispose(); material.dispose(); plane.dispose()
      halo.geometry.dispose(); haloMaterial.dispose(); starsGeometry.dispose(); starsMaterial.dispose()
      renderer.dispose()
    },
  }
}
