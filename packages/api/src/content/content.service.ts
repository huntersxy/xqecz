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

@Injectable()
export class ContentService implements OnModuleInit {
  private readonly log = new Logger(ContentService.name)
  constructor(
    @InjectRepository(Content) private contentRepo: Repository<Content>,
    @InjectRepository(User) private userRepo: Repository<User>,
    @InjectRepository(Claim) private claimRepo: Repository<Claim>,
    @InjectRepository(ContentLike) private likeRepo: Repository<ContentLike>,
    private worker: WorkerService,
    private redis: RedisService,
    @Inject(ConfigService) private cfg: ConfigService,
  ) {}

  // 模块初始化后立即刷新一次推荐位，并启动每 10 分钟的周期刷新。
  // 推荐刷新链路：api 读 MySQL → 调用 worker 纯计算打分 → api 写 Redis ZSet。
  onModuleInit() {
    void this.runStartupMigration().catch((e) => this.log.warn('Startup migration failed (non-fatal):', (e as Error)?.message))
    void this.refreshRecommend().catch((e) => console.warn('[recommend] initial refresh failed:', (e as Error)?.message))
  }

  @Cron('*/10 * * * *')
  handleRecommendRefresh() {
    void this.refreshRecommend().catch((e) => console.warn('[recommend] scheduled refresh failed:', (e as Error)?.message))
  }

  /**
   * 启动时自检迁移：自动识别可安全升级的数据并修复。
   * - 图片类内容有 file_path 但缺 compressed_path → 补压缩
   * - 图片/视频类内容有 file_path 但缺 thumb_path → 补缩略图
   *
   * 所有操作幂等、允许失败，不影响服务启动。
   */
  private async runStartupMigration() {
    // 1. 图片缺压缩
    const needCompress = await this.contentRepo.find({
      where: { type: 'image', file_path: Not(IsNull()), compressed_path: '' },
      select: ['id', 'file_path'],
    })
    if (needCompress.length) {
      this.log.log(`Found ${needCompress.length} images missing compressed version, queuing compression...`)
      for (const row of needCompress) {
        const abs = this.absPath(row.file_path!)
        if (!existsSync(abs)) { this.log.warn(`  #${row.id}: file not found on disk (${abs}), skipping`); continue }
        void this.processMedia(row.id, abs, 'image').catch((e) =>
          this.log.warn(`  #${row.id} compression failed: ${(e as Error)?.message}`))
      }
    }

    // 2. 图片/视频缺缩略图（有原文件但 thumb_path 为空）
    const needThumb = await this.contentRepo.find({
      where: [
        { type: 'image', file_path: Not(IsNull()), thumb_path: IsNull() },
        { type: 'video', file_path: Not(IsNull()), thumb_path: IsNull() },
      ],
      select: ['id', 'type', 'file_path'],
    })
    if (needThumb.length) {
      this.log.log(`Found ${needThumb.length} media items missing thumbnail, queuing generation...`)
      for (const row of needThumb) {
        const abs = this.absPath(row.file_path!)
        if (!existsSync(abs)) { this.log.warn(`  #${row.id}: file not found on disk (${abs}), skipping`); continue }
        void this.processMedia(row.id, abs, row.type).catch((e) =>
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
   * 缩略图存 thumbs（`thumbs/xxx_thumb.jpg` → /thumbs/...），压缩图存 images（`images/xxx.webp` → /images/...）。
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
    const imgUrl = row.type === 'image'
      ? ContentService.fileUrl(row.compressed_path || row.file_path)
      : ''
    // 缩略图优先用生成的 thumb_path；未生成（如 ffmpeg 缺失降级 / 历史数据）时回退到压缩图或原图，避免破图。
    const thumb = ContentService.fileUrl(row.thumb_path) ||
      ContentService.fileUrl(row.compressed_path) ||
      ContentService.fileUrl(row.file_path)
    // 头像：游客用 guest_email，登录用户用 user.email（均为 QQ 邮箱→QQ 头像，其余→Gravatar）
    const emailForAvatar = guestUser ? row.guest_email : user?.email
    const avatarUrl = emailForAvatar ? ContentService.makeAvatarUrl(emailForAvatar) : undefined
    return {
      id: row.id, title: row.title, type: row.type,
      text: row.content || '', url: row.url || '',
      thumb, video: row.type === 'video' ? ContentService.fileUrl(row.file_path) : '',
      img: imgUrl, compressed: row.compressed_path ? ContentService.fileUrl(row.compressed_path) : undefined,
      origin: row.file_path ? ContentService.fileUrl(row.file_path) : undefined,
      platform: row.platform || undefined, file_size: row.file_size || 0,
      ogTitle: row.og_title || undefined, ogImage: row.og_image ? ContentService.fileUrl(row.og_image) : undefined,
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

  async list(opts: { page?: number; pageSize?: number; tag?: string; type?: string; auditStatus?: string; auditStatuses?: string[]; keyword?: string; sortBy?: string; order?: string; userId?: number }) {
    const page = Math.max(1, opts.page || 1)
    const pageSize = Math.min(Math.max(1, opts.pageSize || 20), 100)

    const where: any = {}
    if (opts.auditStatuses?.length) where.audit_status = In(opts.auditStatuses)
    else if (opts.auditStatus) where.audit_status = opts.auditStatus
    if (opts.type) where.type = opts.type
    if (opts.userId) where.user_id = opts.userId
    if (opts.keyword) where.title = Like(`%${opts.keyword}%`)

    const sortBy = ['created_at', 'view_count', 'id'].includes(opts.sortBy || '') ? opts.sortBy! : 'created_at'
    const order = opts.order === 'asc' ? 'ASC' as const : 'DESC' as const

    const [rows, total] = await this.contentRepo.findAndCount({
      where, order: { [sortBy]: order }, skip: (page - 1) * pageSize, take: pageSize,
    })

    const list = await this.decorateRows(rows)
    return { list, total, page, page_size: pageSize, total_page: Math.ceil(total / pageSize) }
  }

  async detail(id: number, silent?: boolean, viewer?: { uid: number | string; is_admin?: boolean }) {
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
    return this.decorateContent(row, undefined, { likeCount })
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
    // 公开可见范围 = 已通过 + 审核中（rejected 不参与标签云）
    const rows = await this.contentRepo.find({ where: { audit_status: In(['approved', 'pending']) }, select: ['tags'] })
    const set = new Set<string>()
    for (const r of rows) for (const t of this.parseTags(r.tags)) set.add(t)
    return [...set]
  }

  async create(input: { title: string; type: string; content?: string; filePath?: string; fileSize?: number; thumbPath?: string; compressedPath?: string; platform?: string; url?: string; tags?: string[]; userId: number; auditStatus?: string; guestNickname?: string; guestEmail?: string }, opts?: { absPath?: string }) {
    const row = this.contentRepo.create({
      title: input.title, type: input.type, content: input.content || '',
      file_path: input.filePath, file_size: input.fileSize || 0,
      thumb_path: input.thumbPath, compressed_path: input.compressedPath || '',
      platform: input.platform, url: input.url,
      tags: JSON.stringify(input.tags || []), user_id: input.userId,
      guest_nickname: input.guestNickname, guest_email: input.guestEmail,
      // 安全默认：pending。仅管理员上传/编辑等显式传 approved。
      audit_status: input.auditStatus || 'pending',
    })
    const saved = await this.contentRepo.save(row)

    // 文件类内容：落库后异步触发缩略图/压缩/S3，不阻塞响应。
    if (opts?.absPath && (input.type === 'image' || input.type === 'video')) {
      void this.processMedia(saved.id, opts.absPath, input.type)
    }
    // 链接类内容：异步抓取 OG 元数据回填。
    if (input.type === 'link' && input.url) {
      void this.processLinkPreview(saved.id, input.url)
    }
    // 仅通过审核的内容进入推荐候选：异步刷新推荐位（不阻塞响应）。
    if ((input.auditStatus || 'pending') === 'approved') {
      void this.refreshRecommend().catch((e) => console.warn('[recommend] refresh after create failed:', (e as Error)?.message))
    }
    return this.decorateContent(saved)
  }

  /** 异步媒体处理：缩略图 → (图片)压缩，逐步回写数据库。任一步失败仅告警不影响其他步骤。 */
  async processMedia(id: number, absPath: string, type: string) {
    try {
      const t = await this.worker.generateThumbnail(absPath, type)
      if (t?.success && t.thumb_path) {
        await this.contentRepo.update(id, { thumb_path: t.thumb_path })
      } else if (t && !t.success) {
        console.warn(`[media] thumbnail failed for #${id}: ${t.error}`)
      }
    } catch (e) {
      console.warn(`[media] thumbnail error #${id}:`, (e as Error)?.message)
    }

    if (type === 'image') {
      try {
        const c = await this.worker.compressImage(absPath)
        if (c?.success && c.compressed_path) {
          await this.contentRepo.update(id, { compressed_path: c.compressed_path })
        } else if (c && !c.success) {
          console.warn(`[media] compress failed for #${id}: ${c.error}`)
        }
      } catch (e) {
        console.warn(`[media] compress error #${id}:`, (e as Error)?.message)
      }
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

  /** 异步抓取链接 OG 元数据并回填 og_title/og_image/platform。 */
  private async processLinkPreview(id: number, url: string) {
    try {
      const r = await this.worker.fetchLinkPreview(url)
      if (r?.success) {
        const update: Record<string, unknown> = {}
        if (r.title) update.og_title = r.title
        if (r.image) update.og_image = r.image
        if (r.platform) update.platform = r.platform
        if (Object.keys(update).length) await this.contentRepo.update(id, update)
      }
    } catch (e) {
      console.warn(`[linkpreview] error #${id}:`, (e as Error)?.message)
    }
  }

  async update(id: number, fields: { title?: string; content?: string; url?: string; filePath?: string; fileSize?: number; thumbPath?: string; compressedPath?: string; platform?: string; tags?: string[] }) {
    const row = await this.contentRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('内容不存在')
    const update: any = {}
    if (fields.title !== undefined) update.title = fields.title
    if (fields.content !== undefined) update.content = fields.content
    if (fields.url !== undefined) update.url = fields.url
    if (fields.filePath !== undefined) update.file_path = fields.filePath
    if (fields.fileSize !== undefined) update.file_size = fields.fileSize
    if (fields.thumbPath !== undefined) update.thumb_path = fields.thumbPath
    if (fields.compressedPath !== undefined) update.compressed_path = fields.compressedPath
    if (fields.platform !== undefined) update.platform = fields.platform
    if (fields.tags !== undefined) update.tags = JSON.stringify(fields.tags)
    if (Object.keys(update).length) await this.contentRepo.update(id, update)
    const updated = await this.contentRepo.findOne({ where: { id } })!
    return this.decorateContent(updated!)
  }

  async softDelete(id: number) {
    await this.contentRepo.softDelete(id)
  }

  async contentExists(id: number): Promise<boolean> {
    return (await this.contentRepo.count({ where: { id } })) > 0
  }

  async getContentRow(id: number) {
    return this.contentRepo.findOne({ where: { id } })
  }

  async setAuditStatus(id: number, status: string) {
    await this.contentRepo.update(id, { audit_status: status })
    // 审核通过意味着内容进入推荐候选，立即刷新推荐位。
    if (status === 'approved') {
      void this.refreshRecommend().catch((e) => console.warn('[recommend] refresh after audit failed:', (e as Error)?.message))
    }
  }

  async updateContentAuthor(id: number, userId: number) {
    const row = await this.contentRepo.findOne({ where: { id } })
    const oldUserId = row?.user_id || 0
    await this.contentRepo.update(id, { user_id: userId })
    const newUser = await this.userRepo.findOne({ where: { id: userId } })
    return { oldUserId, newUsername: newUser?.username || '' }
  }

  async purgeDeleted() {
    const result = await this.contentRepo.createQueryBuilder().delete().where('deleted_at IS NOT NULL').execute()
    return result.affected || 0
  }

  /** 为单条内容重新生成缩略图（仅图片/视频）。 */
  async regenerateThumbnail(id: number) {
    const row = await this.contentRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('内容不存在')
    if (row.type !== 'image' && row.type !== 'video') throw new BadRequestException('仅图片/视频可生成缩略图')
    if (!row.file_path) throw new BadRequestException('无原始文件')
    const t = await this.worker.generateThumbnail(this.absPath(row.file_path), row.type)
    if (!t?.success) throw new BadRequestException(t?.error || '缩略图生成失败')
    await this.contentRepo.update(id, { thumb_path: t.thumb_path })
    const updated = await this.contentRepo.findOne({ where: { id } })
    return this.decorateContent(updated!)
  }

  /** 批量重新生成所有图片/视频的缩略图。 */
  async regenerateAllThumbnails() {
    const rows = await this.contentRepo.find({ where: { type: In(['image', 'video']) } })
    let ok = 0
    let fail = 0
    for (const r of rows) {
      if (!r.file_path) continue
      try {
        const t = await this.worker.generateThumbnail(this.absPath(r.file_path), r.type)
        if (t?.success) {
          await this.contentRepo.update(r.id, { thumb_path: t.thumb_path })
          ok++
        } else {
          fail++
        }
      } catch {
        fail++
      }
    }
    return { ok, fail, total: rows.length }
  }

  /**
   * 2026-07-29 一次性迁移：把 DB 里 type 为 'video' / 'link' 的旧记录统一改为 'text'。
   * 设计要点：
   * - 幂等：再次执行无副作用（剩余 0 条时立刻返回）。
   * - 不动 file_path / url / content 等字段（保留原数据，view 端用 type 区分是否渲染图片/视频）。
   * - 旧 'video' 行有 file_path：仍能渲染为视频，但 type 变成 'text'。
   *   → ContentOverlay.vue 优先看 img / video / url 字段，type 仅用于分类显示/筛选。
   */
  async migrateOldTypes() {
    const oldTypes = ['video', 'link'] as const
    const result = { video: 0, link: 0 }
    for (const t of oldTypes) {
      const r = await this.contentRepo.createQueryBuilder()
        .update(Content)
        .set({ type: 'text' })
        .where('type = :t', { t })
        .execute()
      result[t] = r.affected || 0
    }
    const total = result.video + result.link
    console.log(`[migrate] old-types → text: video=${result.video} link=${result.link} total=${total}`)
    return { ...result, total }
  }
}
