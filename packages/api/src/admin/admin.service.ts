import { Injectable, NotFoundException, ForbiddenException } from '@nestjs/common'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository, Like } from 'typeorm'
import { Content, User, Claim, CommentReport, Comment } from '../entities'
import { ContentService } from '../content/content.service'
import { CommentService } from '../comment/comment.service'

@Injectable()
export class AdminService {
  constructor(
    @InjectRepository(Content) private contentRepo: Repository<Content>,
    @InjectRepository(User) private userRepo: Repository<User>,
    @InjectRepository(Claim) private claimRepo: Repository<Claim>,
    private contentSvc: ContentService,
    private commentSvc: CommentService,
  ) {}

  private formatUser(u: User) {
    return { id: u.id, username: u.username, is_admin: !!u.is_admin, is_banned: !!u.is_banned, created_at: u.created_at, updated_at: u.updated_at }
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
    await this.userRepo.update(id, { is_banned: 1 })
  }

  async listClaims(page = 1, pageSize = 20, status?: string) {
    const where: any = {}
    if (status) where.status = status
    const [rows, total] = await this.claimRepo.findAndCount({
      where, order: { created_at: 'DESC' }, skip: (page - 1) * pageSize, take: pageSize,
    })
    const totalPage = pageSize > 0 ? Math.ceil(total / pageSize) : 1
    return { list: rows, total, page, page_size: pageSize, total_page: totalPage }
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
