import { spawnSync } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const out = process.platform === 'win32' ? 'xqecz-server.exe' : 'xqecz-server'

const result = spawnSync(
  'go',
  ['build', '-ldflags=-s -w', '-o', out, './cmd/server'],
  { cwd: join(root, 'packages', 'server'), stdio: 'inherit' },
)

if (result.error) {
  console.error('[build-server] 无法执行 go：', result.error.message)
  process.exit(1)
}
console.log(`[build-server] 产物：packages/server/${out}`)
process.exit(result.status ?? 1)
