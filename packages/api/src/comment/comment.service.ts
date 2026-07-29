import { Injectable, NotFoundException, ForbiddenException } from '@nestjs/common'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository, IsNull } from 'typeorm'
import { Comment, CommentReport, User } from '../entities'
import { RedisService } from '../redis/redis.service'

@Injectable()
export class CommentService {
  constructor(
    @InjectRepository(Comment) private commentRepo: Repository<Comment>,
    @InjectRepository(CommentReport) private reportRepo: Repository<CommentReport>,
    @InjectRepository(User) private userRepo: Repository<User>,
    private redis: RedisService,
  ) {}

  private async decorateComment(row: Comment) {
    const user = await this.userRepo.findOne({ where: { id: row.user_id } })
    return {
      id: row.id, content_id: row.content_id, user_id: row.user_id, text: row.text,
      parent_id: row.parent_id ?? null, is_banned: !!row.is_banned,
      created_at: row.created_at, updated_at: row.updated_at,
      user: user ? { id: user.id, username: user.username } : { id: row.user_id, username: 'unknown' },
    }
  }

  async list(contentId: number, page = 1, pageSize = 20) {
    // Top-level comments
    const [tops, total] = await this.commentRepo.findAndCount({
      where: { content_id: contentId, parent_id: IsNull() as any, is_banned: 0 },
      order: { created_at: 'ASC' },
      skip: (page - 1) * pageSize, take: pageSize,
    })

    // Replies for those top-level comments
    const topIds = tops.map((t) => t.id)
    let replies: Comment[] = []
    if (topIds.length) {
      replies = await this.commentRepo
        .createQueryBuilder('c')
        .where('c.parent_id IN (:...ids)', { ids: topIds })
        .andWhere('c.is_banned = 0')
        .orderBy('c.created_at', 'ASC')
        .getMany()
    }

    const replyMap = new Map<number, Comment[]>()
    for (const r of replies) {
      if (!replyMap.has(r.parent_id!)) replyMap.set(r.parent_id!, [])
      replyMap.get(r.parent_id!)!.push(r)
    }

    const list = await Promise.all(tops.map(async (t) => {
      const base = await this.decorateComment(t)
      const childReplies = replyMap.get(t.id) || []
      ;(base as any).replies = await Promise.all(childReplies.map((r) => this.decorateComment(r)))
      return base
    }))

    const totalPage = pageSize > 0 ? Math.ceil(total / pageSize) : 1
    return { list, total, page, page_size: pageSize, total_page: totalPage }
  }

  async count(contentId: number) {
    const count = await this.commentRepo.count({ where: { content_id: contentId, is_banned: 0 } as any })
    return { content_id: contentId, count }
  }

  async add(contentId: number, userId: number, text: string, parentId?: number | null) {
    const row = this.commentRepo.create({ content_id: contentId, user_id: userId, text: text.trim(), parent_id: parentId ?? undefined })
    const saved = await this.commentRepo.save(row)
    await this.redis.clearCommentCache(contentId)
    return this.decorateComment(saved)
  }

  async softDelete(id: number, userId: number, isAdmin: boolean) {
    const row = await this.commentRepo.findOne({ where: { id } })
    if (!row) throw new NotFoundException('评论不存在')
    if (userId !== row.user_id && !isAdmin) throw new ForbiddenException('无权删除该评论')
    await this.commentRepo.softDelete(id)
  }

  async report(commentId: number, userId: number, reason: string) {
    const row = this.reportRepo.create({ comment_id: commentId, user_id: userId, reason })
    const saved = await this.reportRepo.save(row)
    return saved
  }

  async listReports() {
    const rows = await this.reportRepo
      .createQueryBuilder('cr')
      .leftJoin('comments', 'c', 'c.id = cr.comment_id')
      .leftJoin('users', 'u', 'u.id = cr.user_id')
      .select(['cr.*', 'c.text AS comment_text', 'u.username AS user_username'])
      .where('cr.handled = 0')
      .orderBy('cr.created_at', 'DESC')
      .getRawMany()
    return rows
  }

  async handleReport(id: number) {
    await this.reportRepo.update(id, { handled: 1 })
  }
}
