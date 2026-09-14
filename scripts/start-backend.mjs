// 宝塔 Node 项目用的后端启动器：把 Go 二进制作为子进程拉起，保持「Node 项目」形态无需改面板类型。
//
// 用法：node scripts/start-backend.mjs
// 若不希望多一层 Node 进程，可在面板里直接运行 packages/server/xqecz-server（见 docs/deploy.md）。
import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const serverDir = join(root, 'packages', 'server')
// go build -o xqecz-server 在不同平台上可能带或不带 .exe，两个候选都探一遍。
const candidates = [join(serverDir, 'xqecz-server.exe'), join(serverDir, 'xqecz-server')]
const bin = candidates.find((p) => existsSync(p))

if (!bin) {
  console.error(`[start-backend] 未找到后端二进制，已尝试：${candidates.join('、')}`)
  console.error('[start-backend] 请先执行 pnpm build:server，或确认部署产物已上传。')
  process.exit(1)
}

// stdio: inherit —— 日志直接进面板，信号原样转发，Node 只做进程守护。
const child = spawn(bin, [], { cwd: serverDir, stdio: 'inherit', env: process.env })

for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
  process.on(signal, () => {
    try { child.kill(signal) } catch { /* 子进程已退出 */ }
  })
}

// spawn 失败（如产物名不匹配、缺少执行权限）时给出可读提示，而不是抛未捕获异常。
child.on('error', (err) => {
  console.error(`[start-backend] 启动 ${bin} 失败：${err.message}`)
  process.exit(1)
})

child.on('exit', (code, signal) => {
  console.log(`[start-backend] 后端退出 code=${code} signal=${signal}`)
  process.exit(code ?? 0)
})
