import { Controller, Get, Post, Delete, Param, Query, Body, Req, UseGuards, Res } from '@nestjs/common'
import type { Request } from 'express'
import { PollService } from './poll.service'
import { AuthGuard } from '../guards/auth.guard'
import { OptionalAuthGuard } from '../guards/optional-auth.guard'
import { CurrentUser } from '../decorators/current-user.decorator'
import { CreatePollDto, VoteDto } from './dto'
import { randomBytes } from 'crypto'

@Controller('poll')
export class PollController {
  constructor(private svc: PollService) {}

  private getVisitorId(req: Request, res: any): string {
    let vid = req.cookies?.['visitor_id'] as string
    if (!vid) {
      vid = randomBytes(16).toString('hex')
      res.cookie('visitor_id', vid, { httpOnly: true, sameSite: 'lax', path: '/', maxAge: 365 * 24 * 3600 * 1000 })
    }
    return vid
  }

  @Get('list')
  async list(@Query('page') page?: string, @Query('page_size') pageSize?: string) {
    const data = await this.svc.list(Number(page) || 1, Number(pageSize) || 20)
    return { code: 200, message: 'ok', data }
  }

  @Get(':id')
  @UseGuards(OptionalAuthGuard)
  async detail(@Param('id') id: string, @CurrentUser('uid') uid: number | undefined, @Req() req: Request) {
    const data = await this.svc.detail(Number(id), uid, req.cookies?.['visitor_id'])
    return { code: 200, message: 'ok', data }
  }

  @Post(':id/vote')
  @UseGuards(OptionalAuthGuard)
  async vote(@Param('id') id: string, @Body() dto: VoteDto, @CurrentUser('uid') uid: number | undefined, @Req() req: Request, @Res({ passthrough: true }) res: any) {
    const vid = uid ? undefined : this.getVisitorId(req, res)
    const data = await this.svc.vote(Number(id), dto.option_index, uid, vid)
    return { code: 200, message: 'ok', data }
  }

  @Post('create')
  @UseGuards(AuthGuard)
  async create(@Body() dto: CreatePollDto, @CurrentUser('uid') uid: number) {
    const data = await this.svc.create(dto.title, dto.description || '', dto.options, uid)
    return { code: 200, message: '创建成功', data }
  }

  @Delete(':id')
  @UseGuards(AuthGuard)
  async delete(@Param('id') id: string, @CurrentUser('uid') uid: number, @CurrentUser('is_admin') isAdmin: boolean) {
    await this.svc.softDelete(Number(id), uid, isAdmin)
    return { code: 200, message: '已删除', data: null }
  }
}
