import { Controller, Get, Post, Delete, Param, Query, Body, UseGuards } from '@nestjs/common'
import { CommentService } from './comment.service'
import { AuthGuard } from '../guards/auth.guard'
import { CurrentUser } from '../decorators/current-user.decorator'
import { AddCommentDto, ListCommentDto, ReportCommentDto } from './dto'

@Controller('comment')
export class CommentController {
  constructor(private svc: CommentService) {}

  @Get('list/:content_id')
  async list(@Param('content_id') contentId: string, @Query() q: ListCommentDto) {
    const data = await this.svc.list(Number(contentId), q.page, q.page_size)
    return { code: 200, message: 'ok', data }
  }

  @Get('count/:content_id')
  async count(@Param('content_id') contentId: string) {
    return { code: 200, message: 'ok', data: await this.svc.count(Number(contentId)) }
  }

  @Post('add')
  @UseGuards(AuthGuard)
  async add(@Body() dto: AddCommentDto, @CurrentUser('uid') uid: number) {
    const data = await this.svc.add(dto.content_id, uid, dto.text, dto.parent_id)
    return { code: 200, message: '评论成功', data }
  }

  @Delete(':id')
  @UseGuards(AuthGuard)
  async delete(@Param('id') id: string, @CurrentUser('uid') uid: number, @CurrentUser('is_admin') isAdmin: boolean) {
    await this.svc.softDelete(Number(id), uid, isAdmin)
    return { code: 200, message: '已删除', data: null }
  }

  @Post('report')
  @UseGuards(AuthGuard)
  async report(@Body() dto: ReportCommentDto, @CurrentUser('uid') uid: number) {
    const data = await this.svc.report(dto.comment_id, uid, dto.reason || '')
    return { code: 200, message: '举报已提交', data }
  }
}
