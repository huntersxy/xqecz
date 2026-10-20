import { spawn, spawnSync } from 'node:child_process'
import { createServer } from 'node:net'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { mkdirSync, readFileSync, writeFileSync, rmSync } from 'node:fs'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const isWin = process.platform === 'win32'
const cacheDir = join(root, 'node_modules', '.cache', 'xqecz-dev')
const pidFile = join(cacheDir, 'pids.json')

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

async function findFreePort(preferred, taken) {
  for (let p = preferred; p < preferred + 200; p++) {
    if (taken.has(p)) continue
    if ((await canListen(p, '0.0.0.0')) && (await canListen(p, '::'))) {
      taken.add(p)
      return p
    }
  }
  throw new Error(`no free port near ${preferred}`)
}

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
      try { process.kill(-pid, 'SIGTERM') } catch { process.kill(pid, 'SIGTERM') }
      setTimeout(() => {
        try { process.kill(-pid, 'SIGKILL') } catch { }
      }, 3000).unref()
    }
  } catch { }
}

function isAlive(pid) {
  try { process.kill(pid, 0); return true } catch { return false }
}

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

function cleanupStale() {
  try {
    const stale = JSON.parse(readFileSync(pidFile, 'utf8'))
    for (const { name, pid, tree } of stale) {
      const targets = new Set([pid, ...(tree || [])])
      targets.delete(process.pid)
      const alive = [...targets].filter((p) => p && isAlive(p))
      if (alive.length) {
        console.log(`[dev] 清理上次遗留的 ${name} 进程 (pids: ${alive.join(',')})`)
        killTree(pid)
        for (const p of alive) { try { process.kill(p, 'SIGKILL') } catch { } }
      }
    }
    rmSync(pidFile, { force: true })
  } catch { }
}

const COLORS = { server: '\x1b[34m', fe: '\x1b[35m', dev: '\x1b[36m' }
const RESET = '\x1b[0m'
const children = []
let shuttingDown = false

function savePids() {
  try {
    mkdirSync(cacheDir, { recursive: true })
    const pairs = listPidPairs()
    writeFileSync(pidFile, JSON.stringify(children.map((c) => ({
      name: c.name,
      pid: c.child.pid,
      tree: pairs.length ? descendantsOf(c.child.pid, pairs) : [],
    }))))
  } catch { }
}

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
  const childEnv = { ...process.env, ...extraEnv, FORCE_COLOR: '1', NO_COLOR: '' }
  const child = isWin
    ? spawn(
        [command, ...args].map((a) => (/^[\w./:=@-]+$/.test(a) ? a : JSON.stringify(a))).join(' '),
        {
          cwd: root,
          env: childEnv,
          stdio: ['ignore', 'pipe', 'pipe'],
          shell: true,
        },
      )
    : spawn(command, args, {
        cwd: root,
        env: childEnv,
        stdio: ['ignore', 'pipe', 'pipe'],
        detached: true,
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
  setTimeout(() => process.exit(code), 200)
}

async function main() {
  cleanupStale()

  const taken = new Set()
  const serverPort = await findFreePort(Number(process.env.PORT) || 3000, taken)
  const fePort = await findFreePort(Number(process.env.VITE_PORT) || 5173, taken)

  console.log(`${COLORS.dev}[dev]${RESET} 端口分配: server=${serverPort} fe=${fePort}`)
  console.log(`${COLORS.dev}[dev]${RESET} 前端入口: http://localhost:${fePort}`)

  launch('server', 'go', ['-C', 'packages/server', 'run', './cmd/server'], {
    PORT: String(serverPort),
    CORS_ORIGINS: `http://localhost:${fePort},http://127.0.0.1:${fePort}`,
  })
  launch('fe', 'pnpm', ['--filter', './packages/frontend', 'run', 'dev'], {
    VITE_PORT: String(fePort),
    VITE_PROXY_TARGET: `http://localhost:${serverPort}`,
  })
  savePids()
  setInterval(savePids, 30_000).unref()

  process.on('SIGINT', () => shutdown(0))
  process.on('SIGTERM', () => shutdown(0))
  process.on('SIGHUP', () => shutdown(0))
  process.on('exit', () => {
    if (!shuttingDown) for (const { child } of children) killTree(child.pid)
  })
}

main().catch((err) => {
  console.error('[dev] 启动失败:', err)
  shutdown(1)
})
