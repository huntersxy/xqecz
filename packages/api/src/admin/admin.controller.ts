import { Controller, Get, Post, Put, Delete, Param, Query, Body, UseGuards } from '@nestjs/common'
import { AdminService } from './admin.service'
import { ContentService } from '../content/content.service'
import { AuthGuard } from '../guards/auth.guard'
import { AdminGuard } from '../guards/admin.guard'
import { CurrentUser } from '../decorators/current-user.decorator'
import { AuditDto, AuthorDto, RoleDto, BanDto, HandleClaimDto } from './dto'

@Controller('admin')
@UseGuards(AuthGuard, AdminGuard)
export class AdminController {
  constructor(
    private svc: AdminService,
    private contentSvc: ContentService,
  ) {}

  @Post('audit/:id')
  async audit(@Param('id') id: string, @Body() dto: AuditDto) {
    const data = await this.svc.audit(Number(id), dto.status)
    return { code: 200, message: '审核完成', data }
  }

  @Get('pending')
  async pending(@Query('page') page?: string, @Query('page_size') pageSize?: string) {
    return { code: 200, message: 'ok', data: await this.svc.pending(Number(page) || 1, Number(pageSize) || 20) }
  }

  @Get('content/all')
  async allContent(@Query() q: any) {
    return { code: 200, message: 'ok', data: await this.svc.allContent({ page: q.page, pageSize: q.page_size, auditStatus: q.audit_status, type: q.type, tag: q.tag, keyword: q.keyword, sortBy: q.sort_by, order: q.order }) }
  }

  @Put('content/:id/author')
  async updateAuthor(@Param('id') id: string, @Body() dto: AuthorDto) {
    const data = await this.svc.updateAuthor(Number(id), dto.user_id)
    return { code: 200, message: 'ok', data: { content_id: Number(id), ...data } }
  }

  @Delete('content/purge')
  async purge() {
    const count = await this.svc.purgeDeleted()
    return { code: 200, message: `已清理 ${count} 条`, data: { count } }
  }

  @Get('users')
  async users(@Query('page') page?: string, @Query('page_size') pageSize?: string, @Query('keyword') keyword?: string) {
    return { code: 200, message: 'ok', data: await this.svc.listUsers(Number(page) || 1, Number(pageSize) || 20, keyword) }
  }

  @Put('users/:id/role')
  async updateRole(@Param('id') id: string, @Body() dto: RoleDto) {
    return { code: 200, message: 'ok', data: await this.svc.updateRole(Number(id), dto.is_admin) }
  }

  @Put('users/:id/ban')
  async updateBan(@Param('id') id: string, @Body() dto: BanDto, @CurrentUser('uid') uid: number) {
    return { code: 200, message: 'ok', data: await this.svc.updateBan(Number(id), dto.is_banned, uid) }
  }

  @Delete('users/:id')
  async deleteUser(@Param('id') id: string, @CurrentUser('uid') uid: number) {
    await this.svc.deleteUser(Number(id), uid)
    return { code: 200, message: '用户已删除', data: null }
  }

  @Get('comments/reports')
  async reports() {
    const rows = await this.svc.listReports()
    return { code: 200, message: 'ok', data: rows.map((r: any) => ({
      id: r.id, comment_id: r.comment_id, user_id: r.user_id, reason: r.reason || '',
      handled: !!r.handled, created_at: r.created_at,
      Comment: r.comment_text ? { id: r.comment_id, text: r.comment_text } : undefined,
      User: r.user_username ? { id: r.user_id, username: r.user_username } : undefined,
    })) }
  }

  @Post('comments/reports/:id/handle')
  async handleReport(@Param('id') id: string) {
    await this.svc.handleReport(Number(id))
    return { code: 200, message: '举报已处理', data: null }
  }

  @Get('claims')
  async claims(@Query('page') page?: string, @Query('page_size') pageSize?: string, @Query('status') status?: string) {
    return { code: 200, message: 'ok', data: await this.svc.listClaims(Number(page) || 1, Number(pageSize) || 20, status) }
  }

  @Post('claims/:id/handle')
  async handleClaim(@Param('id') id: string, @Body() dto: HandleClaimDto, @CurrentUser('uid') uid: number) {
    await this.svc.handleClaim(Number(id), dto.action, dto.remark || '', uid)
    return { code: 200, message: '认领已处理', data: null }
  }

  // ---- 缩略图重建（前端管理后台依赖） ----
  @Post('content/:id/regenerate-thumbnail')
  async regenerateThumbnail(@Param('id') id: string) {
    const data = await this.contentSvc.regenerateThumbnail(Number(id))
    return { code: 200, message: '缩略图已生成', data }
  }

  @Post('content/regenerate-all-thumbnails')
  async regenerateAllThumbnails() {
    const data = await this.contentSvc.regenerateAllThumbnails()
    return { code: 200, message: `成功 ${data.ok} / 失败 ${data.fail}`, data }
  }

  // 手动触发推荐位刷新（api 读 MySQL → worker 打分 → api 写 Redis）。
  @Post('content/refresh-recommend')
  async refreshRecommend() {
    await this.contentSvc.refreshRecommend()
    return { code: 200, message: '推荐位已刷新', data: null }
  }

  // 2026-07-29 一次性迁移：把 type in ('video','link') 的旧记录改为 'text'。幂等，可重复执行。
  @Post('content/migrate-old-types')
  async migrateOldTypes() {
    const data = await this.contentSvc.migrateOldTypes()
    return { code: 200, message: `已迁移 ${data.total} 条（video: ${data.video}，link: ${data.link}）`, data }
  }
}
