import { Injectable, NotFoundException, ForbiddenException } from '@nestjs/common'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository, Like, In } from 'typeorm'
import { Content, User, Claim, CommentReport, Comment, Poll } from '../entities'
import { ContentService } from '../content/content.service'
import { CommentService } from '../comment/comment.service'
import { RedisService } from '../redis/redis.service'

/** 仪表盘聚合结果（与 getJSON 缓存共用同一结构） */
export interface DashboardData {
  content: { total: number; pending: number; approved: number; rejected: number; image: number; text: number; today: number }
  users: { total: number; admins: number; banned: number; today: number }
  comments: { total: number; today: number }
  claims: { total: number; pending: number; approved: number; rejected: number }
  reports: { total: number; unhandled: number; handled: number }
  polls: { total: number; votes: number }
  views: number
  topTags: { tag: string; count: number }[]
  recentContents: unknown[]
  recentUsers: unknown[]
}

@Injectable()
export class AdminService {
  constructor(
    @InjectRepository(Content) private contentRepo: Repository<Content>,
    @InjectRepository(User) private userRepo: Repository<User>,
    @InjectRepository(Claim) private claimRepo: Repository<Claim>,
    @InjectRepository(CommentReport) private reportRepo: Repository<CommentReport>,
    @InjectRepository(Comment) private commentRepo: Repository<Comment>,
    @InjectRepository(Poll) private pollRepo: Repository<Poll>,
    private contentSvc: ContentService,
    private commentSvc: CommentService,
    private redis: RedisService,
  ) {}

  private formatUser(u: User) {
    return { id: u.id, username: u.username, email: u.email || undefined, is_admin: !!u.is_admin, is_banned: !!u.is_banned, created_at: u.created_at, updated_at: u.updated_at }
  }

  /** 仪表盘统计：先读 Redis 缓存（60s TTL），未命中或 Redis 不可用时降级直查 MySQL */
  async getDashboard(fresh = false): Promise<DashboardData> {
    if (!fresh) {
      try {
        const cached = await this.redis.getJSON<DashboardData>('admin:dashboard')
        if (cached) return cached
      } catch { /* Redis 不可用 → 直查 MySQL */ }
    }
    const data = await this.computeDashboard()
    try {
      await this.redis.setJSON('admin:dashboard', data, 60)
    } catch { /* 缓存写失败忽略，不影响返回 */ }
    return data
  }

  private async computeDashboard(): Promise<DashboardData> {
    const todayStart = new Date()
    todayStart.setHours(0, 0, 0, 0)

    // 每张表只发一条条件聚合查询，避免大并发占满连接池
    const [contentRow, userRow, commentRow, claimRow, reportRow, pollRow, viewsRow] =
      await Promise.all([
        this.contentRepo.createQueryBuilder('c')
          .select('COUNT(*)', 'total')
          .addSelect(`COALESCE(SUM(c.audit_status = 'pending'), 0)`, 'pending')
          .addSelect(`COALESCE(SUM(c.audit_status = 'approved'), 0)`, 'approved')
          .addSelect(`COALESCE(SUM(c.audit_status = 'rejected'), 0)`, 'rejected')
          .addSelect(`COALESCE(SUM(c.type = 'image'), 0)`, 'image')
          .addSelect(`COALESCE(SUM(c.type = 'text'), 0)`, 'text')
          .addSelect('COALESCE(SUM(c.created_at >= :today), 0)', 'today')
          .where('c.deleted_at IS NULL')
          .setParameter('today', todayStart)
          .getRawOne(),
        this.userRepo.createQueryBuilder('u')
          .select('COUNT(*)', 'total')
          .addSelect('COALESCE(SUM(u.is_admin = 1), 0)', 'admins')
          .addSelect('COALESCE(SUM(u.is_banned = 1), 0)', 'banned')
          .addSelect('COALESCE(SUM(u.created_at >= :today), 0)', 'today')
          .where('u.deleted_at IS NULL')
          .setParameter('today', todayStart)
          .getRawOne(),
        this.commentRepo.createQueryBuilder('cm')
          .select('COUNT(*)', 'total')
          .addSelect('COALESCE(SUM(cm.created_at >= :today), 0)', 'today')
          .where('cm.deleted_at IS NULL')
          .setParameter('today', todayStart)
          .getRawOne(),
        this.claimRepo.createQueryBuilder('cl')
          .select('COUNT(*)', 'total')
          .addSelect(`COALESCE(SUM(cl.status = 'pending'), 0)`, 'pending')
          .addSelect(`COALESCE(SUM(cl.status = 'approved'), 0)`, 'approved')
          .addSelect(`COALESCE(SUM(cl.status = 'rejected'), 0)`, 'rejected')
          .getRawOne(),
        this.reportRepo.createQueryBuilder('r')
          .select('COUNT(*)', 'total')
          .addSelect('COALESCE(SUM(r.handled = 0), 0)', 'unhandled')
          .getRawOne(),
        this.pollRepo.createQueryBuilder('p')
          .select('COUNT(*)', 'total')
          .addSelect('COALESCE(SUM(p.vote_count), 0)', 'votes')
          .where('p.deleted_at IS NULL')
          .getRawOne(),
        this.contentRepo.createQueryBuilder('c')
          .select('COALESCE(SUM(c.view_count), 0)', 'views')
          .where('c.deleted_at IS NULL')
          .getRawOne(),
      ])

    // 热门标签：只统计已通过内容，避免 pending/垃圾数据污染
    const approvedRows = await this.contentRepo.find({ where: { audit_status: 'approved' }, select: ['tags'] })
    const tagCount = new Map<string, number>()
    for (const row of approvedRows) {
      let arr: unknown[] = []
      try { const a = JSON.parse(row.tags || '[]'); if (Array.isArray(a)) arr = a } catch { /* 忽略坏 JSON */ }
      for (const t of arr) {
        if (typeof t === 'string' && t.trim()) {
          const k = t.trim()
          tagCount.set(k, (tagCount.get(k) || 0) + 1)
        }
      }
    }
    const topTags = [...tagCount.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 10)
      .map(([tag, count]) => ({ tag, count }))

    // 最新内容 / 最新用户，供仪表盘快速预览
    const recentRows = await this.contentRepo.find({ order: { created_at: 'DESC' }, take: 6 })
    const recentContents = await Promise.all(recentRows.map((r) => this.contentSvc.decorateContentPublic(r)))
    const recentUsers = (await this.userRepo.find({ order: { created_at: 'DESC' }, take: 6 })).map((u) => this.formatUser(u))

    const n = (v: unknown) => Number(v) || 0
    const reportTotal = n(reportRow?.total)
    const data: DashboardData = {
      content: {
        total: n(contentRow?.total), pending: n(contentRow?.pending), approved: n(contentRow?.approved),
        rejected: n(contentRow?.rejected), image: n(contentRow?.image), text: n(contentRow?.text),
        today: n(contentRow?.today),
      },
      users: { total: n(userRow?.total), admins: n(userRow?.admins), banned: n(userRow?.banned), today: n(userRow?.today) },
      comments: { total: n(commentRow?.total), today: n(commentRow?.today) },
      claims: { total: n(claimRow?.total), pending: n(claimRow?.pending), approved: n(claimRow?.approved), rejected: n(claimRow?.rejected) },
      reports: { total: reportTotal, unhandled: n(reportRow?.unhandled), handled: reportTotal - n(reportRow?.unhandled) },
      polls: { total: n(pollRow?.total), votes: n(pollRow?.votes) },
      views: n(viewsRow?.views),
      topTags,
      recentContents,
      recentUsers,
    }
    return data
  }

  async audit(id: number, status: string) {
    const row = await this.contentSvc.getContentRow(id)
    if (!row) throw new NotFoundException('内容不存在')
    await this.contentSvc.setAuditStatus(id, status)
    return this.contentSvc.detail(id)
  }

  async pending(page = 1, pageSize = 20) {
    return this.contentSvc.list({ page, pageSize, auditStatus: 'pending', sortBy: 'created_at', order: 'asc' })
  }

  async allContent(opts: { page?: number; pageSize?: number; auditStatus?: string; type?: string; tag?: string; keyword?: string; sortBy?: string; order?: string }) {
    return this.contentSvc.list(opts)
  }

  async updateAuthor(contentId: number, newUserId: number) {
    if (!(await this.contentSvc.getContentRow(contentId))) throw new NotFoundException('内容不存在')
    const newUser = await this.userRepo.findOne({ where: { id: newUserId } })
    if (!newUser) throw new NotFoundException('目标用户不存在')
    return this.contentSvc.updateContentAuthor(contentId, newUserId)
  }

  async purgeDeleted() {
    return this.contentSvc.purgeDeleted()
  }

  async listUsers(page = 1, pageSize = 20, keyword?: string) {
    const where: any = {}
    if (keyword) where.username = Like(`%${keyword}%`)
    const [rows, total] = await this.userRepo.findAndCount({
      where, order: { id: 'DESC' }, skip: (page - 1) * pageSize, take: pageSize,
    })
    const totalPage = pageSize > 0 ? Math.ceil(total / pageSize) : 1
    return { list: rows.map((u) => this.formatUser(u)), total, page, page_size: pageSize, total_page: totalPage }
  }

  async updateRole(id: number, isAdmin: boolean) {
    const u = await this.userRepo.findOne({ where: { id } })
    if (!u) throw new NotFoundException('用户不存在')
    await this.userRepo.update(id, { is_admin: isAdmin ? 1 : 0 })
    const updated = await this.userRepo.findOne({ where: { id } })!
    return this.formatUser(updated!)
  }

  async updateBan(id: number, isBanned: boolean, operatorId: number) {
    const u = await this.userRepo.findOne({ where: { id } })
    if (!u) throw new NotFoundException('用户不存在')
    if (isBanned && operatorId === id) throw new ForbiddenException('不能封禁自己')
    await this.userRepo.update(id, { is_banned: isBanned ? 1 : 0 })
    const updated = await this.userRepo.findOne({ where: { id } })!
    return this.formatUser(updated!)
  }

  async deleteUser(id: number, operatorId: number) {
    if (operatorId === id) throw new ForbiddenException('不能删除自己')
    const u = await this.userRepo.findOne({ where: { id } })
    if (!u) throw new NotFoundException('用户不存在')
    await this.userRepo.softDelete(id)
  }

  async listClaims(page = 1, pageSize = 20, status?: string) {
    const where: any = {}
    if (status) where.status = status
    const [rows, total] = await this.claimRepo.findAndCount({
      where, order: { created_at: 'DESC' }, skip: (page - 1) * pageSize, take: pageSize,
    })
    // 批量关联 content + user，前端需要展示缩略图/标题/认领人
    const contentIds = [...new Set(rows.map((r) => Number(r.content_id)))]
    const userIds = [...new Set(rows.map((r) => Number(r.user_id)))]
    const contents = contentIds.length
      ? await this.contentRepo.find({ where: { id: In(contentIds) } })
      : []
    const users = userIds.length
      ? await this.userRepo.find({ where: { id: In(userIds) } })
      : []
    const contentMap = new Map(contents.map((c) => [Number(c.id), c]))
    const userMap = new Map(users.map((u) => [Number(u.id), u]))
    const list = await Promise.all(rows.map(async (r) => {
      const c = contentMap.get(Number(r.content_id))
      const u = userMap.get(Number(r.user_id))
      return {
        ...r,
        content: c ? await this.contentSvc.decorateContentPublic(c) : null,
        user: u ? { id: u.id, username: u.username } : { id: Number(r.user_id), username: 'unknown' },
      }
    }))
    const totalPage = pageSize > 0 ? Math.ceil(total / pageSize) : 1
    return { list, total, page, page_size: pageSize, total_page: totalPage }
  }

  async handleClaim(id: number, action: 'approve' | 'reject', remark: string, operatorId: number) {
    const claim = await this.claimRepo.findOne({ where: { id } })
    if (!claim) throw new NotFoundException('认领申请不存在')
    await this.claimRepo.update(id, { status: action === 'approve' ? 'approved' : 'rejected', remark, approved_by: operatorId })
    if (action === 'approve') {
      await this.contentSvc.updateContentAuthor(claim.content_id, claim.user_id)
    }
  }

  async listReports() { return this.commentSvc.listReports() }
  async handleReport(id: number) { return this.commentSvc.handleReport(id) }
}
