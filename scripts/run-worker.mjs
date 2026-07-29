// 启动 Go Worker（跨平台，零依赖）。
// 共享上传目录「项目根/data/*」由项目根相对解析（不写死盘符/绝对路径），
// 保证 NestJS 与 Worker 指向同一目录（见 AGENTS.md「共享上传目录」约束）。
//
// 进程销毁要点：不用 `go run`（它会 spawn 编译产物为孙进程，杀掉 go 进程会留下
// 孤儿 gRPC 服务占住端口）。改为先 `go build -o` 再直接运行产物，本进程即服务进程，
// kill 即彻底销毁；收到 SIGINT/SIGTERM 时转发给子进程（Go 侧有 GracefulStop）。
import { spawn, spawnSync } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { mkdirSync } from 'node:fs'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const isWin = process.platform === 'win32'
// 三目录同级：data/uploads（原文件，扁平）、data/thumbs、data/images
const dataDir = join(root, 'data')
const env = {
  ...process.env,
  UPLOAD_DIR: join(dataDir, 'uploads'),
  THUMB_DIR: join(dataDir, 'thumbs'),
  IMAGES_DIR: join(dataDir, 'images'),
}

const workerDir = join(root, 'packages', 'worker')
const binDir = join(root, 'node_modules', '.cache', 'xqecz-dev')
mkdirSync(binDir, { recursive: true })
const binPath = join(binDir, isWin ? 'worker-dev.exe' : 'worker-dev')

// 1) 编译（增量，秒级）
const build = spawnSync('go', ['build', '-o', binPath, './cmd/server/'], {
  cwd: workerDir,
  env,
  stdio: 'inherit',
  shell: isWin,
})
if (build.status !== 0) {
  console.error('[worker] go build 失败')
  process.exit(build.status ?? 1)
}

// 2) 直接运行编译产物（本子进程即 gRPC 服务本体）
const child = spawn(binPath, [], { cwd: workerDir, env, stdio: 'inherit' })

const forward = (sig) => () => {
  if (child.pid) {
    try { child.kill(sig) } catch { /* already gone */ }
  }
}
process.on('SIGINT', forward('SIGINT'))
process.on('SIGTERM', forward('SIGTERM'))
process.on('exit', () => { try { child.kill('SIGKILL') } catch { /* already gone */ } })

child.on('exit', (code, signal) => process.exit(signal ? 0 : (code ?? 0)))
