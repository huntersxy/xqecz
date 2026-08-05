import { Injectable, OnModuleDestroy, OnModuleInit } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import Redis from 'ioredis'

@Injectable()
export class RedisService implements OnModuleInit, OnModuleDestroy {
  private client!: Redis
  private prefix: string

  constructor(private cfg: ConfigService) {
    this.prefix = this.cfg.get('REDIS_PREFIX', 'xqecz:')
  }

  onModuleInit() {
    this.client = new Redis({
      host: this.cfg.get('REDIS_HOST', 'localhost'),
      port: this.cfg.get<number>('REDIS_PORT', 6379),
      password: this.cfg.get('REDIS_PASSWORD') || undefined,
      db: this.cfg.get<number>('REDIS_DB', 0),
      keyPrefix: this.prefix,
      lazyConnect: true,
    })
    this.client.on('error', (err) => console.error('[redis]', err.message))
    this.client.connect().then(() => console.log('[redis] connected'))
  }

  async onModuleDestroy() {
    await this.client.quit()
  }

  // ---- generic cache ----

  async set(key: string, value: string | number, ttlSeconds: number) {
    await this.client.set(key, String(value), 'EX', ttlSeconds)
  }

  async get(key: string): Promise<string | null> {
    return this.client.get(key)
  }

  async setJSON(key: string, value: unknown, ttlSeconds: number) {
    await this.client.set(key, JSON.stringify(value), 'EX', ttlSeconds)
  }

  async getJSON<T>(key: string): Promise<T | null> {
    const raw = await this.client.get(key)
    if (!raw) return null
    try { return JSON.parse(raw) as T } catch { return null }
  }

  /**
   * 读穿缓存：命中直接返回；未命中执行 loader 并写缓存。
   * Redis 不可用时自动降级为直查（读写均容错），不影响业务。
   */
  async getOrSetJSON<T>(key: string, ttlSeconds: number, loader: () => Promise<T>): Promise<T> {
    try {
      const cached = await this.getJSON<T>(key)
      if (cached !== null) return cached
    } catch { /* Redis 不可用 → 直查 */ }
    const data = await loader()
    try { await this.setJSON(key, data, ttlSeconds) } catch { /* 缓存写失败忽略 */ }
    return data
  }

  async del(...keys: string[]) {
    if (keys.length) await this.client.del(...keys)
  }

  // ---- session ----

  private readonly SESSION_TTL = 30 * 24 * 3600
  private readonly SESSION_RENEW = 15 * 24 * 3600

  async setSession(sessionID: string, userID: number) {
    await this.set(`session:${sessionID}`, userID, this.SESSION_TTL)
  }

  async getSession(sessionID: string): Promise<number | null> {
    const val = await this.get(`session:${sessionID}`)
    if (!val) return null
    const uid = Number(val)
    if (!Number.isFinite(uid) || uid <= 0) return null
    // auto-renew
    const ttl = await this.client.ttl(`session:${sessionID}`)
    if (ttl >= 0 && ttl < this.SESSION_RENEW)
      await this.client.expire(`session:${sessionID}`, this.SESSION_TTL)
    return uid
  }

  async delSession(sessionID: string) {
    await this.del(`session:${sessionID}`)
  }

  // ---- view count ----

  async incrementView(contentID: number): Promise<number> {
    const today = new Date().toISOString().slice(0, 10)
    const key = `views:date:${today}:${contentID}`
    const count = await this.client.incr(key)
    if (count === 1) await this.client.expire(key, 32 * 24 * 3600)
    return count
  }

  // ---- recommend ZSet ----

  // 推荐 ZSet 由 api 侧独占总写（key 为 recommend:hot，经 ioredis keyPrefix 实际为 xqecz:recommend:hot）。
  // worker 仅做无状态打分并返回评分，本方法负责原子写入，避免读取端看到半写入状态。
  // 架构约束见 AGENTS.md：NestJS 独占 DB/Redis，worker 不直连。
  async writeRecommendList(items: { contentId: number; score: number }[]): Promise<void> {
    if (!items.length) {
      // 没有可推荐内容时不清空已有推荐位，避免推荐区变空。
      return
    }
    const key = 'recommend:hot'
    const tempKey = key + ':temp'
    const args: (number | string)[] = []
    for (const it of items) args.push(it.score, it.contentId)
    const pipe = this.client.pipeline()
    pipe.del(tempKey)
    pipe.zadd(tempKey, ...args)
    pipe.rename(tempKey, key)
    const res = await pipe.exec()
    // pipeline 单条命令失败不会抛异常，必须显式检查（此前 zadd 传入非法 member
    // 静默失败，导致推荐位一直是旧数据）。
    const failed = (res || []).filter(([err]) => err)
    if (failed.length) {
      throw new Error(`writeRecommendList pipeline failed: ${failed.map(([e]) => (e as Error).message).join('; ')}`)
    }
  }

  async getRecommendList(page: number, pageSize: number): Promise<number[]> {
    const start = (page - 1) * pageSize
    const end = start + pageSize - 1
    const vals = await this.client.zrevrange('recommend:hot', start, end)
    return vals.map(Number).filter((id) => Number.isFinite(id) && id > 0)
  }

  async getRecommendTotal(): Promise<number> {
    const n = await this.client.zcard('recommend:hot')
    return n
  }

  // ---- cache invalidation ----

  /** 按模式删除键（SCAN 精确匹配带前缀的真实 key，再去前缀删除）。 */
  async delByPattern(pattern: string) {
    const fullPattern = this.prefix + pattern
    const keys: string[] = []
    let cursor = '0'
    do {
      const [next, found] = await this.client.scan(cursor, 'MATCH', fullPattern, 'COUNT', 100)
      cursor = next
      keys.push(...found)
    } while (cursor !== '0')
    if (keys.length) {
      const stripped = keys.map((k) => k.startsWith(this.prefix) ? k.slice(this.prefix.length) : k)
      await this.client.del(...stripped)
    }
  }

  async clearCommentCache(contentID: number) {
    await this.delByPattern(`comments:${contentID}:*`)
    await this.del(`comment_count:${contentID}`)
  }

  /** 单条内容详情缓存失效（content:{id}）。 */
  async clearContentCache(contentID: number) {
    await this.del(`content:${contentID}`)
  }

  /** 内容列表/搜索/标签缓存失效（content_list:* + tags）。 */
  async clearContentListCache() {
    await this.delByPattern('content_list:*')
    await this.del('tags')
  }

  /** 全部内容相关缓存失效（detail + list + tags，用户资料变更影响装饰结果时用）。 */
  async clearAllContentCaches() {
    await this.delByPattern('content*')
    await this.del('tags')
  }

  async clearCachesOnStartup() {
    //保留 session 和 view count
    const pattern = this.prefix + '*'
    let cursor = '0'
    do {
      const [next, keys] = await this.client.scan(cursor, 'MATCH', pattern, 'COUNT', 200)
      cursor = next
      for (const fullKey of keys) {
        const k = fullKey.startsWith(this.prefix) ? fullKey.slice(this.prefix.length) : fullKey
        if (k.includes('session:') || k.includes('views:date:')) continue
        await this.client.del(k)
      }
    } while (cursor !== '0')
    console.log('[redis] caches cleared on startup')
  }

  get raw() { return this.client }

  // ---- rate limit ----

  /** 计数器自增；首次创建时设置 TTL。用于游客上传等简单限频。 */
  async incrWithTTL(key: string, ttlSeconds: number): Promise<number> {
    const count = await this.client.incr(key)
    if (count === 1) await this.client.expire(key, ttlSeconds)
    return count
  }

  // ---- distributed lock ----

  /**
   * 尝试获取分布式锁（SET NX EX 原子操作）。
   * @returns true 表示成功获取锁，false 表示锁已被其他实例持有。
   */
  async acquireLock(key: string, ttlSeconds: number): Promise<boolean> {
    const result = await this.client.set(key, '1', 'EX', ttlSeconds, 'NX')
    return result === 'OK'
  }

  /** 释放分布式锁（DEL，仅在锁仍存在时删除）。 */
  async releaseLock(key: string): Promise<void> {
    await this.client.del(key)
  }

  /** 续期分布式锁 TTL（长任务心跳用；锁不存在或已过期返回 false）。 */
  async renewLock(key: string, ttlSeconds: number): Promise<boolean> {
    const result = await this.client.expire(key, ttlSeconds)
    return result === 1
  }
}
