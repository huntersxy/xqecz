import * as THREE from 'three'
import { MapControls } from 'three/addons/controls/MapControls.js'
import { loadPlanetImage } from './artworkPlanet'
import { universeBounds, visibleUniverseNodes, type UniverseNode } from './artworkUniverse'

/**
 * 拥有整个场景的渲染器、GPU 资源、监听器与加载任务；调用方卸载时必须 dispose。
 * 全量作品保留轻量实例，缩略图只为可见候选加载并有上限；禁止逐作品常驻原图纹理。
 * setActive 只暂停帧循环，dispose 才释放资源；图片地址一律交给 loadPlanetImage。
 */
export function createUniverseScene(canvas: HTMLCanvasElement, onSelect: (node: UniverseNode | undefined) => void) {
  const renderer = new THREE.WebGLRenderer({ canvas, alpha: true, antialias: true })
  renderer.setPixelRatio(Math.min(devicePixelRatio, 1.5))
  const scene = new THREE.Scene()
  const camera = new THREE.OrthographicCamera(-1000, 1000, 1000, -1000, .1, 5000)
  camera.position.set(0, 0, 2000)
  const controls = new MapControls(camera, canvas)
  controls.screenSpacePanning = true
  controls.enableRotate = false
  controls.enableDamping = false
  controls.minZoom = .45
  controls.maxZoom = 14
  controls.enabled = false
  const geometry = new THREE.SphereGeometry(1, 16, 12)
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
  let lastLoad = 0
  let loading = 0
  const abort = new AbortController()
  const pending = new Set<number>()
  const failed = new Set<number>()
  const thumbnails = new Map<number, { mesh: THREE.Mesh<THREE.PlaneGeometry, THREE.MeshBasicMaterial>; texture: THREE.CanvasTexture }>()
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
  const stars = new Float32Array(1800 * 3)
  for (let i = 0; i < stars.length; i += 3) {
    stars[i] = (Math.random() - .5) * 14000
    stars[i + 1] = (Math.random() - .5) * 14000
    stars[i + 2] = -200 - Math.random() * 600
  }
  starsGeometry.setAttribute('position', new THREE.BufferAttribute(stars, 3))
  const starsMaterial = new THREE.PointsMaterial({ color: '#e2ddeb', size: 3.5, transparent: true, opacity: .65, sizeAttenuation: false })
  scene.add(new THREE.Points(starsGeometry, starsMaterial))

  function selected(node: UniverseNode | undefined) {
    halo.visible = !!node
    if (node) { halo.position.set(node.x, node.y, 65); halo.scale.setScalar(node.radius) }
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
    if (camera.zoom === 1 && camera.position.x === 0 && camera.position.y === 0 && nodes.length) size = universeBounds(nodes) / Math.min(1, width / height)
    renderer.setSize(width, height, false)
    const aspect = width / height
    camera.left = -size * aspect / 2
    camera.right = size * aspect / 2
    camera.top = size / 2
    camera.bottom = -size / 2
    camera.updateProjectionMatrix()
    dirty = true
  }
  function fit() {
    size = universeBounds(nodes) / Math.min(1, width / height)
    camera.zoom = 1
    camera.position.set(0, 0, 2000)
    controls.target.set(0, 0, 0)
    resize()
    controls.update()
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
  function loadVisible(time: number) {
    if (time - lastLoad < 180 || loading >= 3) return
    lastLoad = time
    const visible = visibleUniverseNodes(nodes, camera.position.x, camera.position.y, size * width / height / camera.zoom, size / camera.zoom)
    const keep = new Set(visible.map(node => node.artwork.id))
    thumbnails.forEach((entry, id) => { entry.mesh.visible = keep.has(id) })
    for (const node of visible) {
      const id = node.artwork.id
      if (!node.artwork.thumb || thumbnails.has(id) || pending.has(id) || failed.has(id)) continue
      if (loading >= 3) break
      pending.add(id)
      loading++
      void loadPlanetImage(node.artwork.thumb, abort.signal).then(image => {
        if (disposed || !image) { if (!disposed) failed.add(id); return }
        // Downsample before GPU upload; at most 64 small textures are resident.
        const tile = document.createElement('canvas')
        tile.width = tile.height = 192
        const ctx = tile.getContext('2d')!
        ctx.beginPath(); ctx.arc(96, 96, 94, 0, Math.PI * 2); ctx.clip()
        const crop = Math.min(image.naturalWidth, image.naturalHeight)
        ctx.drawImage(image, (image.naturalWidth - crop) / 2, (image.naturalHeight - crop) / 2, crop, crop, 0, 0, 192, 192)
        const shade = ctx.createRadialGradient(67, 55, 25, 96, 96, 98)
        shade.addColorStop(0, 'rgba(255,255,255,.06)'); shade.addColorStop(.65, 'rgba(0,0,0,.04)'); shade.addColorStop(1, 'rgba(0,0,0,.65)')
        ctx.fillStyle = shade; ctx.fillRect(0, 0, 192, 192)
        const texture = new THREE.CanvasTexture(tile)
        texture.colorSpace = THREE.SRGBColorSpace
        const mesh = new THREE.Mesh(plane, new THREE.MeshBasicMaterial({ map: texture, transparent: true, depthWrite: false }))
        mesh.position.set(node.x, node.y, node.radius + 1)
        mesh.scale.setScalar(node.radius)
        if (thumbnails.size >= 64) {
          const victim = [...thumbnails].find(([key]) => !keep.has(key)) ?? thumbnails.entries().next().value
          if (victim) release(victim[0], victim[1])
        }
        thumbnails.set(id, { mesh, texture })
        scene.add(mesh)
        dirty = true
      }).finally(() => { pending.delete(id); loading-- })
    }
  }
  function draw(time: number) {
    if (disposed || !active || document.hidden) return
    frame = requestAnimationFrame(draw)
    if (time - lastDraw < 33) return
    lastDraw = time
    controls.update()
    loadVisible(time)
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
    update, fit, zoom, focus, project, setActive,
    dispose() {
      disposed = true; active = false; abort.abort(); cancelAnimationFrame(frame)
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
