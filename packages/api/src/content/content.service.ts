import { Injectable, NotFoundException, BadRequestException, Inject, OnModuleInit, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { Cron } from '@nestjs/schedule'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository, Like, In, IsNull, Not } from 'typeorm'
import { join, relative } from 'path'
import { createHash } from 'crypto'
import { existsSync } from 'fs'
import { Content, User, ContentLike } from '../entities'
import { WorkerService } from '../worker/worker.service'
import { RedisService } from '../redis/redis.service'
import { Claim } from '../entities'
import { UPLOAD_DIR } from '../paths'
import { convertNonGifToWebp } from './webp.util'

/** 批量缩略图后台任务状态（Redis 持久化，管理端轮询用）。 */
export interface RegenAllStatus {
  status: 'idle' | 'running' | 'done' | 'error'
  total: number
  ok: number
  fail: number
  started_at?: number
  updated_at?: number
  finished_at?: number
  error?: string
}

/** 内容装饰结果（list / detail 缓存的统一形状）。 */
type DecoratedContent = ReturnType<ContentService['decorateContent']>

@Injectable()
export class ContentService implements OnModuleInit {
  private readonly log = new Logger(ContentService.name)

  // 批量缩略图后台任务：Redis 分布式锁（多实例防重）+ 状态记录（供管理端查询进度）。
  // 锁 TTL 与心跳配合：任务逐条串行可能运行很久，心跳每 30s 续期一次；
  // 进程崩溃后锁最多 2 分钟自动释放，不会永久卡死。
  private static readonly REGEN_LOCK_KEY = 'lock:regen-all-thumbs'
  private static readonly REGEN_STATUS_KEY = 'task:regen-all-thumbs:status'
  private static readonly REGEN_LOCK_TTL = 120
  private static readonly REGEN_HEARTBEAT_MS = 30_000
  private static readonly REGEN_STATUS_TTL = 7 * 24 * 3600
  // 内容缓存 TTL（5 分钟安全兜底；所有写路径都会显式失效，保证不出现缓存不更新）。
  private static readonly CONTENT_TTL = 300
  private static readonly LIST_TTL = 300
  private static readonly TAGS_TTL = 300
  private static readonly VIDEO_EXTS = new Set([
    '.mp4', '.webm', '.mov', '.m4v', '.mkv', '.avi', '.flv', '.ogv', '.wmv', '.3gp', '.mpeg', '.mpg', '.ts', '.m2ts',
  ])
  private regenHeartbeat: ReturnType<typeof setInterval> | undefined

  constructor(
    @InjectRepository(Content) private contentRepo: Repository<Content>,
    @InjectRepository(User) private userRepo: Repository<User>,
    @InjectRepository(Claim) private claimRepo: Repository<Claim>,
    @InjectRepository(ContentLike) private likeRepo: Repository<ContentLike>,
    private worker: WorkerService,
    private redis: RedisService,
    @Inject(ConfigService) private cfg: ConfigService,
  ) {}

  /** 按文件扩展名识别视频文件（type 归并后，媒体类型以扩展名为唯一依据）。 */
  static isVideoFile(filePath: string): boolean {
    const dot = filePath.lastIndexOf('.')
    const ext = (dot >= 0 ? filePath.slice(dot) : filePath).toLowerCase()
    return ContentService.VIDEO_EXTS.has(ext)
  }

  /** 供 worker 使用的媒体类型（content_type）：视频扩展名 → video，其余 → image。 */
  static mediaTypeForPath(filePath: string): 'image' | 'video' {
    return ContentService.isVideoFile(filePath) ? 'video' : 'image'
  }

  /** 列表缓存键：规范化参数后 sha1，确保不同筛选条件互不串缓存。 */
  private contentListKey(opts: { page?: number; pageSize?: number; tag?: string; auditStatus?: string; auditStatuses?: string[]; keyword?: string; sortBy?: string; order?: string; userId?: number }): string {
    const norm = {
      page: Math.max(1, opts.page || 1),
      pageSize: Math.min(Math.max(1, opts.pageSize || 20), 100),
      tag: opts.tag || '',
      auditStatus: opts.auditStatus || '',
      auditStatuses: [...(opts.auditStatuses || [])].sort(),
      keyword: opts.keyword || '',
      sortBy: opts.sortBy && ['created_at', 'view_count', 'id'].includes(opts.sortBy) ? opts.sortBy : 'created_at',
      order: opts.order === 'asc' ? 'asc' : 'desc',
      userId: opts.userId || 0,
    }
    return `content_list:${createHash('sha1').update(JSON.stringify(norm)).digest('hex')}`
  }

  /** 写路径缓存失效：单条详情 + 列表/搜索/标签；Redis 不可用时忽略（TTL 兜底）。 */
  private async invalidateContent(id: number) {
    await Promise.all([
      this.redis.clearContentCache(id).catch(() => undefined),
      this.redis.clearContentListCache().catch(() => undefined),
    ])
  }

  // 模块初始化后立即刷新一次推荐位，并启动每 10 分钟的周期刷新。
  // 推荐刷新链路：api 读 MySQL → 调用 worker 纯计算打分 → api 写 Redis ZSet。
  onModuleInit() {
    // 启动时清空非会话缓存，避免代码升级后读到旧格式/旧数据。
    void this.redis.clearCachesOnStartup().catch((e) =>
      console.warn('[redis] startup cache clear failed (non-fatal):', (e as Error)?.message))
    void this.runStartupMigration().catch((e) => this.log.warn('Startup migration failed (non-fatal):', (e as Error)?.message))
    void this.refreshRecommend().catch((e) => console.warn('[recommend] initial refresh failed:', (e as Error)?.message))
  }

  @Cron('*/10 * * * *')
  handleRecommendRefresh() {
    void this.refreshRecommend().catch((e) => console.warn('[recommend] scheduled refresh failed:', (e as Error)?.message))
  }

  /**
   * 启动时自检迁移：自动识别可安全升级的数据并修复。
   * - 有 file_path 但缺 thumb_path → 补缩略图（图片/视频统一按扩展名识别）
   *
   * 所有操作幂等、允许失败，不影响服务启动。
   */
  private async runStartupMigration() {
    // 缺缩略图（有原文件但 thumb_path 为空，覆盖历史 video 记录）
    const needThumb = await this.contentRepo.find({
      where: { file_path: Not(IsNull()), thumb_path: IsNull() },
      select: ['id', 'file_path'],
    })
    if (needThumb.length) {
      this.log.log(`Found ${needThumb.length} media items missing thumbnail, queuing generation...`)
      for (const row of needThumb) {
        const abs = this.absPath(row.file_path!)
        if (!existsSync(abs)) { this.log.warn(`  #${row.id}: file not found on disk (${abs}), skipping`); continue }
        void this.processMedia(row.id, abs, ContentService.mediaTypeForPath(row.file_path!)).catch((e) =>
          this.log.warn(`  #${row.id} thumbnail failed: ${(e as Error)?.message}`))
      }
    }
  }

  /** 共享卷根目录（与 Go Worker 挂载同一路径，由项目根相对解析）。 */
  private uploadDir(): string {
    return this.cfg.get('UPLOAD_DIR', UPLOAD_DIR)
  }

  /** 把相对路径（如 <md5>.jpg，扁平存于 uploads 根）还原为 worker 可读的绝对路径。 */
  private absPath(rel: string): string {
    return join(this.uploadDir(), rel)
  }

  /** 根据 sessionID 解析登录用户 ID，未登录返回 0。 */
  async resolveUserId(sessionID: string): Promise<number> {
    if (!sessionID) return 0
    try {
      const uid = await this.redis.getSession(sessionID)
      return uid ?? 0
    } catch { return 0 }
  }

  /** 根据 session 解析完整用户（快速上传用于判断是否管理员自动过审）。 */
  async resolveUser(sessionID: string) {
    if (!sessionID) return null
    try {
      const uid = await this.redis.getSession(sessionID)
      if (!uid) return null
      return await this.userRepo.findOne({ where: { id: uid } })
    } catch {
      return null
    }
  }

  private parseTags(v: unknown): string[] {
    if (Array.isArray(v)) return v.filter((t) => typeof t === 'string')
    if (typeof v === 'string') {
      try { const a = JSON.parse(v); if (Array.isArray(a)) return a.filter((t) => typeof t === 'string') } catch { /* */ }
      return v.split(',').map((s) => s.trim()).filter(Boolean)
    }
    return []
  }

  /**
   * 将存储的相对路径补全为前端可访问的 URL。
   * 三目录同级：原文件扁平存 uploads（如 `d41d8cd9....png` → /uploads/...），
   * 缩略图存 thumbs（`thumbs/xxx_thumb.webp` → /thumbs/...），压缩图存 images（`images/xxx.webp` → /images/...）。
   */
  static fileUrl(rel?: string): string {
    if (!rel) return ''
    if (rel.startsWith('http://') || rel.startsWith('https://')) return rel
    if (rel.startsWith('/')) return rel
    if (rel.startsWith('thumbs/')) return `/${rel}`
    if (rel.startsWith('images/')) return `/${rel}`
    return `/uploads/${rel}`
  }

  private async decorateContent(
    row: Content,
    userMap?: Map<number, User>,
    opts: { includeViewCount?: boolean; likeCount?: number } = {},
  ) {
    // 游客快速上传：user_id=0，以留档昵称对外展示（邮箱不外露）。
    const guestUser = row.guest_nickname
      ? { id: 0, username: row.guest_nickname }
      : null
    const user = guestUser ? null : (userMap?.get(row.user_id) ?? await this.userRepo.findOne({ where: { id: row.user_id } }))
    // 内容不再分类：媒体类型按 file_path 扩展名识别。
    // 图片 → img 字段；视频文件 → video 字段；无文件的历史行（仅剩缩略图）用缩略图兜底展示。
    const isVideo = !!row.file_path && ContentService.isVideoFile(row.file_path)
    const imgUrl = !isVideo && (row.file_path || row.thumb_path)
      ? ContentService.fileUrl(row.file_path || row.thumb_path)
      : ''
    // 缩略图优先用生成的 thumb_path；未生成时回退到原文件，避免破图。
    const thumb = ContentService.fileUrl(row.thumb_path) ||
      ContentService.fileUrl(row.file_path)
    // 头像：游客用 guest_email，登录用户用 user.email（均为 QQ 邮箱→QQ 头像，其余→Gravatar）
    const emailForAvatar = guestUser ? row.guest_email : user?.email
    const avatarUrl = emailForAvatar ? ContentService.makeAvatarUrl(emailForAvatar) : undefined
    return {
      id: row.id, title: row.title,
      text: row.content || '',
      thumb, video: isVideo ? ContentService.fileUrl(row.file_path!) : '',
      img: imgUrl,
      origin: row.file_path ? ContentService.fileUrl(row.file_path) : undefined,
      file_size: row.file_size || 0,
      user: guestUser || (user ? { id: user.id, username: user.username } : { id: row.user_id, username: 'unknown' }),
      avatar_url: avatarUrl,
      tags: this.parseTags(row.tags),
      // 点赞为公开指标；浏览量不对外展示，仅内部（admin/推荐）使用
      like_count: opts.likeCount ?? 0,
      ...(opts.includeViewCount ? { view_count: row.view_count || 0 } : {}),
      audit_status: row.audit_status, created_at: row.created_at, updated_at: row.updated_at,
    }
  }

  /** 批量统计点赞数（按内容分组，避免 N+1）。key 统一用字符串：TypeORM bigint 主键返回字符串。 */
  private async getLikeCountMap(ids: number[]): Promise<Map<string, number>> {
    const map = new Map<string, number>()
    if (!ids.length) return map
    const rows = await this.likeRepo
      .createQueryBuilder('l')
      .select('l.content_id', 'cid')
      .addSelect('COUNT(*)', 'cnt')
      .where('l.content_id IN (:...ids)', { ids })
      .groupBy('l.content_id')
      .getRawMany<{ cid: string; cnt: string }>()
    for (const r of rows) map.set(String(r.cid), Number(r.cnt))
    return map
  }

  /** 批量查询用户，消除 N+1 查询 */
  private async buildUserMap(rows: { user_id: number }[]) {
    const userIds = [...new Set(rows.map((r) => r.user_id).filter((id) => id > 0))]
    const users = userIds.length ? await this.userRepo.find({ where: { id: In(userIds) } }) : []
    return new Map(users.map((u) => [u.id, u]))
  }

  /** 批量装饰内容：批量用户 + 点赞数，消除 N+1 查询 */
  private async decorateRows(rows: Content[]) {
    const userMap = await this.buildUserMap(rows)
    const likeMap = await this.getLikeCountMap(rows.map((r) => r.id))
    return Promise.all(rows.map((r) => this.decorateContent(r, userMap, { likeCount: likeMap.get(String(r.id)) })))
  }

  /** 根据邮箱生成头像 URL（不暴露原始邮箱），QQ 邮箱 → QQ 头像接口，其余 → Gravatar */
  static makeAvatarUrl(email: string, size: number = 80): string {
    if (!email) return ''
    const trimmed = email.trim().toLowerCase()
    const qqMatch = /^(\d{5,11})@qq\.com$/.exec(trimmed)
    if (qqMatch) {
      return `https://q.qlogo.cn/headimg_dl?dst_uin=${qqMatch[1]}&spec=100`
    }
    const hash = createHash('md5').update(trimmed).digest('hex')
    return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=${size}`
  }

  /** 公开版 decorateContent，供 AdminService 等内部服务复用格式化逻辑 */
  async decorateContentPublic(row: Content, opts?: { includeViewCount?: boolean; likeCount?: number }) {
    return this.decorateContent(row, undefined, opts)
  }

  async list(opts: { page?: number; pageSize?: number; tag?: string; auditStatus?: string; auditStatuses?: string[]; keyword?: string; sortBy?: string; order?: string; userId?: number }) {
    // 列表/搜索/我的内容：按规范化参数哈希缓存，写路径统一失效。
    const key = this.contentListKey(opts)
    return this.redis.getOrSetJSON(key, ContentService.LIST_TTL, async () => {
      const page = Math.max(1, opts.page || 1)
      const pageSize = Math.min(Math.max(1, opts.pageSize || 20), 100)

      const where: any = {}
      if (opts.auditStatuses?.length) where.audit_status = In(opts.auditStatuses)
      else if (opts.auditStatus) where.audit_status = opts.auditStatus
      if (opts.userId) where.user_id = opts.userId
      if (opts.keyword) where.title = Like(`%${opts.keyword}%`)

      const sortBy = ['created_at', 'view_count', 'id'].includes(opts.sortBy || '') ? opts.sortBy! : 'created_at'
      const order = opts.order === 'asc' ? 'ASC' as const : 'DESC' as const

      const [rows, total] = await this.contentRepo.findAndCount({
        where, order: { [sortBy]: order }, skip: (page - 1) * pageSize, take: pageSize,
      })

      const list = await this.decorateRows(rows)
      return { list, total, page, page_size: pageSize, total_page: Math.ceil(total / pageSize) }
    })
  }

  async detail(id: number, silent?: boolean, viewer?: { uid: number | string; is_admin?: boolean }) {
    // 匿名非 silent 的公开详情走缓存；登录/内部请求直查，保证权限判断与新鲜度。
    if (!viewer && !silent) {
      const cached = await this.redis.getJSON<DecoratedContent>(`content:${id}`).catch(() => null)
      if (cached) {
        // 缓存不包含浏览量：命中时照常累计，保证统计不丢。
        await this.contentRepo.increment({ id }, 'view_count', 1)
        await this.redis.incrementView(id)
        return cached
      }
    }
    const row = await this.contentRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('内容不存在')
    // 审核中（pending）内容公开可见；已拒绝（rejected）不对外展示，仅作者本人或管理员可见。
    if (row.audit_status === 'rejected') {
      const isOwner = viewer && String(viewer.uid) === String(row.user_id)
      if (!isOwner && !viewer?.is_admin) {
        throw new NotFoundException('内容不存在或未通过审核')
      }
    }
    if (!silent) {
      await this.contentRepo.increment({ id }, 'view_count', 1)
      await this.redis.incrementView(id)
    }
    const likeCount = await this.likeRepo.count({ where: { content_id: id } })
    const data = this.decorateContent(row, undefined, { likeCount })
    // 仅缓存公开可见（非 rejected）内容；rejected 的访问权限需实时判断，不缓存。
    if (!viewer && row.audit_status !== 'rejected') {
      await this.redis.setJSON(`content:${id}`, data, ContentService.CONTENT_TTL).catch(() => undefined)
    }
    return data
  }

  async recommend(count: number, page: number) {
    const limit = Math.max(1, Math.min(count || 20, 100))
    const pageNum = Math.max(1, page)

    // 优先读 api 写入的 Redis ZSet（recommend:hot，由 refreshRecommend 维护）。
    try {
      const ids = await this.redis.getRecommendList(pageNum, limit)
      if (ids.length) {
        const rows = await this.contentRepo.find({
          where: { id: In(ids), audit_status: 'approved' },
        })
        // TypeORM 将 bigint 主键返回为字符串，而 ZSet 读出的是数字，需统一为字符串再查表。
        const byId = new Map(rows.map((r) => [String(r.id), r]))
        const ordered = ids.map((id) => byId.get(String(id))).filter((r): r is Content => !!r)

        const list = await this.decorateRows(ordered)
        const total = await this.redis.getRecommendTotal()
        return { list, count: total }
      }
    } catch (e) {
      console.warn('[recommend] redis read failed, fallback to db:', (e as Error)?.message)
    }

    // 降级：Worker 还没跑过，退回按浏览量排序。
    const offset = (pageNum - 1) * limit
    const rows = await this.contentRepo.find({
      where: { audit_status: 'approved' },
      order: { view_count: 'DESC', created_at: 'DESC' },
      skip: offset, take: limit,
    })
    const total = await this.contentRepo.count({ where: { audit_status: 'approved' } })

    const list = await this.decorateRows(rows)
    return { list, count: total }
  }

  /**
   * 重新计算推荐位（api 侧主导，符合 AGENTS.md「NestJS 独占 DB/Redis」约束）：
   * 1) 从 MySQL 读取所有已审核通过的内容（id + 创建时间 + 浏览量）；
   * 2) 通过 gRPC 交给 worker 纯计算打分；
   * 3) 把评分结果原子写入 Redis ZSet（recommend:hot），供 recommend() 读取。
   * worker 不直接访问 DB/Redis。
   *
   * 多实例防抖：通过 Redis 分布式锁确保同一时刻只有一个实例在执行刷新，
   * 锁 TTL 5 分钟（正常刷新应在 10 秒内完成）。
   */
  async refreshRecommend() {
    const lockKey = 'lock:refresh-recommend'
    const locked = await this.redis.acquireLock(lockKey, 300)
    if (!locked) {
      console.log('[recommend] another instance is refreshing, skip')
      return
    }
    try {
      const rows = await this.contentRepo.find({
        where: { audit_status: 'approved' },
        select: ['id', 'created_at', 'view_count'],
      })
      if (!rows.length) {
        console.log('[recommend] no approved contents, skip refresh')
        return
      }
      const likeMap = await this.getLikeCountMap(rows.map((r) => r.id))
      const items = rows.map((r) => ({
        contentId: r.id,
        createdAtUnix: Math.floor(new Date(r.created_at).getTime() / 1000),
        viewCount: r.view_count || 0,
        likeCount: likeMap.get(String(r.id)) || 0,
      }))
      const scored = await this.worker.refreshRecommend(items)
      await this.redis.writeRecommendList(scored)
      console.log(`[recommend] refreshed ${scored.length} items`)
    } finally {
      await this.redis.releaseLock(lockKey)
    }
  }

  async getAllTags() {
    return this.redis.getOrSetJSON('tags', ContentService.TAGS_TTL, async () => {
      // 公开可见范围 = 已通过 + 审核中（rejected 不参与标签云）
      const rows = await this.contentRepo.find({ where: { audit_status: In(['approved', 'pending']) }, select: ['tags'] })
      const set = new Set<string>()
      for (const r of rows) for (const t of this.parseTags(r.tags)) set.add(t)
      return [...set]
    })
  }

  async create(input: { title: string; content?: string; filePath?: string; fileSize?: number; thumbPath?: string; tags?: string[]; userId: number; auditStatus?: string; guestNickname?: string; guestEmail?: string }, opts?: { absPath?: string }) {
    const row = this.contentRepo.create({
      title: input.title, content: input.content || '',
      file_path: input.filePath, file_size: input.fileSize || 0,
      thumb_path: input.thumbPath,
      tags: JSON.stringify(input.tags || []), user_id: input.userId,
      guest_nickname: input.guestNickname, guest_email: input.guestEmail,
      // 安全默认：pending。仅管理员上传/编辑等显式传 approved。
      audit_status: input.auditStatus || 'pending',
    })
    const saved = await this.contentRepo.save(row)
    // 新内容改变所有列表/搜索/标签结果。
    await this.redis.clearContentListCache().catch(() => undefined)

    // 文件类内容：落库后异步生成缩略图，不阻塞响应。媒体类型按扩展名识别。
    if (opts?.absPath) {
      void this.processMedia(saved.id, opts.absPath, ContentService.mediaTypeForPath(opts.absPath))
    }
    // 仅通过审核的内容进入推荐候选：异步刷新推荐位（不阻塞响应）。
    if ((input.auditStatus || 'pending') === 'approved') {
      void this.refreshRecommend().catch((e) => console.warn('[recommend] refresh after create failed:', (e as Error)?.message))
    }
    return this.decorateContent(saved)
  }

  /** 异步媒体处理：生成缩略图并回写 thumb_path（失败仅告警不影响其他步骤）。 */
  async processMedia(id: number, absPath: string, type: string) {
    try {
      const t = await this.worker.generateThumbnail(absPath, type)
      if (t?.success && t.thumb_path) {
        await this.contentRepo.update(id, { thumb_path: t.thumb_path })
        await this.invalidateContent(id)
      } else if (t && !t.success) {
        console.warn(`[media] thumbnail failed for #${id}: ${t.error}`)
      }
    } catch (e) {
      console.warn(`[media] thumbnail error #${id}:`, (e as Error)?.message)
    }
  }

  /**
   * 上传时统一处理原图：非 GIF 图片在本地无损转为 WebP，
   * 转换成功后删除源文件，并以 WebP 作为新原图（file_path / file_size 均指向它）。
   * 跳过或转换失败时返回 null，调用方沿用原始文件。
   */
  async prepareOriginalFile(file: { path: string; mimetype?: string }): Promise<{ absPath: string; relPath: string; size: number } | null> {
    const converted = await convertNonGifToWebp(file.path, file.mimetype)
    if (!converted) return null
    return {
      absPath: converted.absPath,
      relPath: relative(this.uploadDir(), converted.absPath).split('\\').join('/'),
      size: converted.size,
    }
  }

  async update(id: number, fields: { title?: string; content?: string; filePath?: string; fileSize?: number; thumbPath?: string; tags?: string[] }) {
    const row = await this.contentRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('内容不存在')
    const update: any = {}
    if (fields.title !== undefined) update.title = fields.title
    if (fields.content !== undefined) update.content = fields.content
    if (fields.filePath !== undefined) update.file_path = fields.filePath
    if (fields.fileSize !== undefined) update.file_size = fields.fileSize
    if (fields.thumbPath !== undefined) update.thumb_path = fields.thumbPath
    if (fields.tags !== undefined) update.tags = JSON.stringify(fields.tags)
    if (Object.keys(update).length) {
      await this.contentRepo.update(id, update)
      await this.invalidateContent(id)
    }
    const updated = await this.contentRepo.findOne({ where: { id } })!
    return this.decorateContent(updated!)
  }

  async softDelete(id: number) {
    await this.contentRepo.softDelete(id)
    await this.invalidateContent(id)
  }

  async contentExists(id: number): Promise<boolean> {
    return (await this.contentRepo.count({ where: { id } })) > 0
  }

  async getContentRow(id: number) {
    return this.contentRepo.findOne({ where: { id } })
  }

  async setAuditStatus(id: number, status: string) {
    await this.contentRepo.update(id, { audit_status: status })
    await this.invalidateContent(id)
    // 审核通过意味着内容进入推荐候选，立即刷新推荐位。
    if (status === 'approved') {
      void this.refreshRecommend().catch((e) => console.warn('[recommend] refresh after audit failed:', (e as Error)?.message))
    }
  }

  async updateContentAuthor(id: number, userId: number) {
    const row = await this.contentRepo.findOne({ where: { id } })
    const oldUserId = row?.user_id || 0
    await this.contentRepo.update(id, { user_id: userId })
    await this.invalidateContent(id)
    const newUser = await this.userRepo.findOne({ where: { id: userId } })
    return { oldUserId, newUsername: newUser?.username || '' }
  }

  async purgeDeleted() {
    const result = await this.contentRepo.createQueryBuilder().delete().where('deleted_at IS NOT NULL').execute()
    await this.redis.clearContentListCache().catch(() => undefined)
    return result.affected || 0
  }

  /** 为单条内容重新生成缩略图（有原始文件即可，媒体类型按扩展名识别）。 */
  async regenerateThumbnail(id: number) {
    const row = await this.contentRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('内容不存在')
    if (!row.file_path) throw new BadRequestException('无原始文件')
    const t = await this.worker.generateThumbnail(this.absPath(row.file_path), ContentService.mediaTypeForPath(row.file_path))
    if (!t?.success) throw new BadRequestException(t?.error || '缩略图生成失败')
    await this.contentRepo.update(id, { thumb_path: t.thumb_path })
    await this.invalidateContent(id)
    const updated = await this.contentRepo.findOne({ where: { id } })
    return this.decorateContent(updated!)
  }

  /**
   * 批量重新生成所有有原始文件的缩略图（接口立即返回，后台异步执行，避免超时）。
   * 多实例防重：Redis 分布式锁 + 心跳续期；进程崩溃时锁最多 2 分钟自动释放。
   * 进度/结果写入 Redis 状态，管理端可经 getRegenerateAllStatus() 轮询。
   */
  async regenerateAllThumbnails() {
    if (!(await this.redis.acquireLock(ContentService.REGEN_LOCK_KEY, ContentService.REGEN_LOCK_TTL))) {
      return { ok: 0, fail: 0, total: 0, count: 0, running: true }
    }
    try {
      const rows = await this.contentRepo.find({
        where: { file_path: Not(IsNull()) },
        select: ['id', 'file_path'],
      })
      const targets = rows.filter((r): r is Content & { file_path: string } => Boolean(r.file_path))
      if (!targets.length) {
        await this.redis.del(ContentService.REGEN_STATUS_KEY)
        await this.redis.releaseLock(ContentService.REGEN_LOCK_KEY).catch(() => undefined)
        return { ok: 0, fail: 0, total: 0, count: 0 }
      }
      const startedAt = Date.now()
      await this.redis.setJSON(ContentService.REGEN_STATUS_KEY, {
        status: 'running', total: targets.length, ok: 0, fail: 0, started_at: startedAt,
      }, ContentService.REGEN_STATUS_TTL)
      // 心跳续期：长任务（数千条逐条串行）不会因 TTL 到期被其他实例重复触发。
      this.regenHeartbeat = setInterval(() => {
        void this.redis.renewLock(ContentService.REGEN_LOCK_KEY, ContentService.REGEN_LOCK_TTL)
          .catch((e) => this.log.warn('批量缩略图锁续期失败:', (e as Error)?.message || e))
      }, ContentService.REGEN_HEARTBEAT_MS)
      void this.runRegenerateAll(targets, startedAt)
        .catch((e) => {
          this.log.error('批量缩略图后台任务异常:', (e as Error)?.message || e)
          void this.redis.setJSON(ContentService.REGEN_STATUS_KEY, {
            status: 'error', total: targets.length, ok: 0, fail: 0,
            started_at: startedAt, finished_at: Date.now(),
            error: (e as Error)?.message || String(e),
          }, ContentService.REGEN_STATUS_TTL).catch(() => undefined)
        })
        .finally(() => {
          if (this.regenHeartbeat) {
            clearInterval(this.regenHeartbeat)
            this.regenHeartbeat = undefined
          }
          void this.redis.releaseLock(ContentService.REGEN_LOCK_KEY).catch(() => undefined)
        })
      return { ok: 0, fail: 0, total: targets.length, count: targets.length }
    } catch (e) {
      // 查询/状态写入失败：立即释放锁并恢复心跳，避免占用 TTL。
      if (this.regenHeartbeat) {
        clearInterval(this.regenHeartbeat)
        this.regenHeartbeat = undefined
      }
      await this.redis.releaseLock(ContentService.REGEN_LOCK_KEY).catch(() => undefined)
      throw e
    }
  }

  /** 后台批量重生成：逐条调 worker 并更新 thumb_path（单条失败不中断），每 10 条与结束时写一次状态。 */
  private async runRegenerateAll(targets: Array<Content & { file_path: string }>, startedAt: number) {
    let ok = 0
    let fail = 0
    for (let i = 0; i < targets.length; i++) {
      const r = targets[i]
      try {
        const t = await this.worker.generateThumbnail(this.absPath(r.file_path), ContentService.mediaTypeForPath(r.file_path))
        if (t?.success) {
          await this.contentRepo.update(r.id, { thumb_path: t.thumb_path })
          await this.redis.clearContentCache(r.id).catch(() => undefined)
          ok++
        } else {
          fail++
        }
      } catch {
        fail++
      }
      if ((i + 1) % 10 === 0 || i === targets.length - 1) {
        await this.redis.setJSON(ContentService.REGEN_STATUS_KEY, {
          status: 'running', total: targets.length, ok, fail,
          started_at: startedAt, updated_at: Date.now(),
        }, ContentService.REGEN_STATUS_TTL).catch(() => undefined)
      }
    }
    await this.redis.setJSON(ContentService.REGEN_STATUS_KEY, {
      status: 'done', total: targets.length, ok, fail,
      started_at: startedAt, finished_at: Date.now(),
    }, ContentService.REGEN_STATUS_TTL).catch(() => undefined)
    // 全部更新完统一失效列表/标签缓存（缩略图嵌入列表卡片）。
    await this.redis.clearContentListCache().catch(() => undefined)
    this.log.log(`批量缩略图完成: 成功 ${ok} / 失败 ${fail} / 共 ${targets.length}`)
  }

  /** 查询批量缩略图任务状态（供管理端轮询；无任务/已过期返回 idle）。 */
  async getRegenerateAllStatus(): Promise<RegenAllStatus> {
    const st = await this.redis.getJSON<RegenAllStatus>(ContentService.REGEN_STATUS_KEY)
    return st ?? { status: 'idle', total: 0, ok: 0, fail: 0 }
  }
}
