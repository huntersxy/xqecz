import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const serverDir = join(root, 'packages', 'server')
const candidates = [join(serverDir, 'xqecz-server.exe'), join(serverDir, 'xqecz-server')]
const bin = candidates.find((p) => existsSync(p))

if (!bin) {
  console.error(`[start-backend] 未找到后端二进制，已尝试：${candidates.join('、')}`)
  console.error('[start-backend] 请先执行 pnpm build:server，或确认部署产物已上传。')
  process.exit(1)
}

const child = spawn(bin, [], { cwd: serverDir, stdio: 'inherit', env: process.env })

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
  process.on(signal, () => {
    try { child.kill(signal) } catch { }
  })
}

child.on('error', (err) => {
  console.error(`[start-backend] 启动 ${bin} 失败：${err.message}`)
  process.exit(1)
})

child.on('exit', (code, signal) => {
  console.log(`[start-backend] 后端退出 code=${code} signal=${signal}`)
  process.exit(code ?? 0)
})
