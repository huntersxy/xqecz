// 开发编排器：自适应端口 + 完善的进程销毁（替代 concurrently）。
//
// 1) 自适应端口：每次启动前探测空闲端口（api 优先 3000、worker 优先 50051、fe 优先 5173，
//    被占用则向上顺延），并通过环境变量把实际端口注入三端，保证互连关系正确：
//      api    ← PORT / WORKER_URL / CORS_ORIGINS
//      worker ← WORKER_PORT
//      fe     ← VITE_PORT / VITE_PROXY_TARGET
// 2) 完善的进程销毁：
//    - Ctrl+C / SIGTERM / 任一子进程退出 → 递归杀掉所有子进程树（Windows 用 PowerShell CIM
//      枚举后代，POSIX 用进程组），不留孤儿进程占端口。
//    - 每次启动先读取上一次的 pid 记录，清理上次异常退出遗留的孤儿进程（自愈）。
import { spawn, spawnSync } from 'node:child_process'
import { createServer } from 'node:net'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { mkdirSync, readFileSync, writeFileSync, rmSync } from 'node:fs'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const isWin = process.platform === 'win32'
const cacheDir = join(root, 'node_modules', '.cache', 'xqecz-dev')
const pidFile = join(cacheDir, 'pids.json')

// ---------- 端口探测 ----------
/** 尝试在指定 host:port 上监听一次，成功即认为可用。 */
function canListen(port, host) {
  return new Promise((resolve) => {
    const srv = createServer()
    srv.unref()
    srv.once('error', () => resolve(false))
    srv.listen({ port, host, exclusive: true }, () => {
      srv.close(() => resolve(true))
    })
  })
}

/** 从 preferred 起向上找一个 IPv4/IPv6 都空闲、且未被本次启动占用的端口。 */
async function findFreePort(preferred, taken) {
  for (let p = preferred; p < preferred + 200; p++) {
    if (taken.has(p)) continue
    // Nest 监听 '::'、Vite 监听全接口，两个族都要空闲才算真空闲
    if ((await canListen(p, '0.0.0.0')) && (await canListen(p, '::'))) {
      taken.add(p)
      return p
    }
  }
  throw new Error(`no free port near ${preferred}`)
}

// ---------- 进程树销毁 ----------
/** 递归杀掉 pid 及其全部后代。Windows 走 PowerShell CIM（本机 taskkill/wmic 被策略禁用）。 */
function killTree(pid) {
  if (!pid) return
  try {
    if (isWin) {
      const script = [
        `$procs = Get-CimInstance Win32_Process | Select-Object ProcessId,ParentProcessId;`,
        `function Get-Desc($p) { $procs | Where-Object { $_.ParentProcessId -eq $p } | ForEach-Object { Get-Desc $_.ProcessId; $_.ProcessId } };`,
        `$ids = @(Get-Desc ${pid}) + ${pid};`,
        `$ids | ForEach-Object { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue }`,
      ].join(' ')
      spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', script], {
        stdio: 'ignore',
        timeout: 15000,
      })
    } else {
      // POSIX：子进程以独立进程组启动，杀整组
      try { process.kill(-pid, 'SIGTERM') } catch { /* 组不存在则杀单个 */ process.kill(pid, 'SIGTERM') }
      setTimeout(() => {
        try { process.kill(-pid, 'SIGKILL') } catch { /* already gone */ }
      }, 3000).unref()
    }
  } catch { /* 进程已不存在 */ }
}

/** 判断 pid 是否仍存活。 */
function isAlive(pid) {
  try { process.kill(pid, 0); return true } catch { return false }
}

/** 一次性获取系统全部 (pid, ppid) 对（Windows），用于在 JS 侧计算后代集合。 */
function listPidPairs() {
  if (!isWin) return []
  const r = spawnSync(
    'powershell.exe',
    ['-NoProfile', '-NonInteractive', '-Command',
      'Get-CimInstance Win32_Process | ForEach-Object { "$($_.ProcessId) $($_.ParentProcessId)" }'],
    { encoding: 'utf8', timeout: 15000 },
  )
  if (r.status !== 0 || !r.stdout) return []
  return r.stdout.split(/\r?\n/).map((l) => l.trim().split(' ').map(Number)).filter((a) => a.length === 2 && !Number.isNaN(a[0]))
}

/** 基于 (pid, ppid) 快照计算 root 的全部后代。 */
function descendantsOf(root, pairs) {
  const byParent = new Map()
  for (const [pid, ppid] of pairs) {
    if (!byParent.has(ppid)) byParent.set(ppid, [])
    byParent.get(ppid).push(pid)
  }
  const out = []
  const stack = [root]
  while (stack.length) {
    const p = stack.pop()
    for (const c of byParent.get(p) || []) { out.push(c); stack.push(c) }
  }
  return out
}

/** 启动前自愈：清理上一次 dev 会话遗留的孤儿进程。
 *  同时按「根 pid 递归」与「后代快照逐个杀」双路清理——
 *  覆盖父进程已死、孙进程（如 vite/nest 的 node、worker 的 exe）仍存活的断链场景。 */
function cleanupStale() {
  try {
    const stale = JSON.parse(readFileSync(pidFile, 'utf8'))
    for (const { name, pid, tree } of stale) {
      const targets = new Set([pid, ...(tree || [])])
      targets.delete(process.pid)
      const alive = [...targets].filter((p) => p && isAlive(p))
      if (alive.length) {
        console.log(`[dev] 清理上次遗留的 ${name} 进程 (pids: ${alive.join(',')})`)
        killTree(pid) // 先按树递归（还活着的父子链）
        for (const p of alive) { try { process.kill(p, 'SIGKILL') } catch { /* 已退出 */ } }
      }
    }
    rmSync(pidFile, { force: true })
  } catch { /* 无残留记录 */ }
}

// ---------- 子进程管理 ----------
const COLORS = { api: '\x1b[34m', worker: '\x1b[32m', fe: '\x1b[35m', dev: '\x1b[36m' }
const RESET = '\x1b[0m'
const children = [] // { name, child }
let shuttingDown = false

function savePids() {
  try {
    mkdirSync(cacheDir, { recursive: true })
    // 附带整棵后代 pid 快照：即使编排器被强杀（无信号机会）、父子链断裂，
    // 下次启动也能按快照逐个清理孤儿（vite/nest 的 node、worker exe 等孙进程）。
    const pairs = listPidPairs()
    writeFileSync(pidFile, JSON.stringify(children.map((c) => ({
      name: c.name,
      pid: c.child.pid,
      tree: pairs.length ? descendantsOf(c.child.pid, pairs) : [],
    }))))
  } catch { /* 缓存目录不可写时忽略（仅影响自愈） */ }
}

/** 带前缀转发子进程输出。 */
function pipePrefixed(name, stream, out) {
  let buf = ''
  stream.on('data', (chunk) => {
    buf += chunk.toString()
    let i
    while ((i = buf.indexOf('\n')) >= 0) {
      out.write(`${COLORS[name] || ''}[${name}]${RESET} ${buf.slice(0, i + 1)}`)
      buf = buf.slice(i + 1)
    }
  })
  stream.on('end', () => { if (buf) out.write(`${COLORS[name] || ''}[${name}]${RESET} ${buf}\n`) })
}

function launch(name, command, args, extraEnv) {
  const child = spawn(command, args, {
    cwd: root,
    env: { ...process.env, ...extraEnv, FORCE_COLOR: '1' },
    stdio: ['ignore', 'pipe', 'pipe'],
    shell: isWin, // pnpm/go 在 Windows 需经 shell 解析
    detached: !isWin, // POSIX 独立进程组，便于整组销毁
  })
  pipePrefixed(name, child.stdout, process.stdout)
  pipePrefixed(name, child.stderr, process.stderr)
  child.on('exit', (code, signal) => {
    if (shuttingDown) return
    console.log(`${COLORS.dev}[dev]${RESET} ${name} 退出 (code=${code} signal=${signal})，正在销毁其余进程...`)
    shutdown(code ?? 1)
  })
  children.push({ name, child })
  return child
}

function shutdown(code = 0) {
  if (shuttingDown) return
  shuttingDown = true
  console.log(`${COLORS.dev}[dev]${RESET} 正在销毁全部子进程树...`)
  for (const { name, child } of children) {
    if (child.pid && isAlive(child.pid)) {
      killTree(child.pid)
      console.log(`${COLORS.dev}[dev]${RESET} 已销毁 ${name} (pid ${child.pid})`)
    }
  }
  rmSync(pidFile, { force: true })
  // 给 stdout 冲刷留一点时间
  setTimeout(() => process.exit(code), 200)
}

// ---------- 主流程 ----------
async function main() {
  cleanupStale()

  const taken = new Set()
  const apiPort = await findFreePort(Number(process.env.PORT) || 3000, taken)
  const workerPort = await findFreePort(Number(process.env.WORKER_PORT) || 50051, taken)
  const fePort = await findFreePort(Number(process.env.VITE_PORT) || 5173, taken)

  console.log(`${COLORS.dev}[dev]${RESET} 端口分配: api=${apiPort} worker=${workerPort} fe=${fePort}`)
  console.log(`${COLORS.dev}[dev]${RESET} 前端入口: http://localhost:${fePort}`)

  launch('worker', 'node', [join(root, 'scripts', 'run-worker.mjs')], {
    WORKER_PORT: String(workerPort),
  })
  launch('api', 'pnpm', ['--filter', './packages/api', 'run', 'start:dev'], {
    PORT: String(apiPort),
    WORKER_URL: `localhost:${workerPort}`,
    CORS_ORIGINS: `http://localhost:${fePort},http://127.0.0.1:${fePort}`,
  })
  launch('fe', 'pnpm', ['--filter', './packages/frontend', 'run', 'dev'], {
    VITE_PORT: String(fePort),
    VITE_PROXY_TARGET: `http://localhost:${apiPort}`,
  })
  savePids()
  // nest --watch 重编译会更换孙进程 pid，定期刷新快照保证自愈精准
  setInterval(savePids, 30_000).unref()

  process.on('SIGINT', () => shutdown(0))
  process.on('SIGTERM', () => shutdown(0))
  process.on('SIGHUP', () => shutdown(0))
  process.on('exit', () => {
    // 最后兜底：同步杀一遍（正常路径 shutdown 已处理，这里覆盖异常崩溃）
    if (!shuttingDown) for (const { child } of children) killTree(child.pid)
  })
}

main().catch((err) => {
  console.error('[dev] 启动失败:', err)
  shutdown(1)
})
