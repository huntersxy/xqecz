import { Controller, Get, Post, Put, Delete, Param, Query, Body, Req, Res, UseGuards, UploadedFile, UseInterceptors, BadRequestException } from '@nestjs/common'
import { FileInterceptor } from '@nestjs/platform-express'
import { diskStorage } from 'multer'
import { basename, extname, relative } from 'path'
import { randomBytes } from 'crypto'
import { mkdirSync, unlink } from 'fs'
import type { Request, Response } from 'express'
import { ContentService } from './content.service'
import { AuthGuard } from '../guards/auth.guard'
import { CurrentUser } from '../decorators/current-user.decorator'
import { UploadContentDto, UpdateContentDto, ListContentDto, ClaimDto, QuickUploadDto } from './dto'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { Claim } from '../entities'
import { RedisService } from '../redis/redis.service'
import { UPLOAD_DIR } from '../paths'

// 共享卷根目录（与 Go Worker 挂载同一路径，由项目根相对解析）。multer 把文件落在这里，api 再把它传给 worker 处理。
function uploadRoot(): string {
  return process.env.UPLOAD_DIR || UPLOAD_DIR
}

// multer 磁盘存储：原文件扁平写到 {UPLOAD_DIR}/ 根目录（无子目录）。
// 前端上传前已把文件重命名为 <md5>.<ext>；文件名合规（32 位十六进制）就直接沿用，
// 否则（第三方客户端/旧调用）服务端兜底生成随机名，保证任何来源都能落盘。
const MD5_NAME = /^[a-f0-9]{32}\.[a-z0-9]{1,8}$/i
function uploadStorage() {
  return diskStorage({
    destination: (_req, _file, cb) => {
      const root = uploadRoot()
      mkdirSync(root, { recursive: true })
      cb(null, root)
    },
    filename: (_req, file, cb) => {
      const original = basename(file.originalname || '')
      if (MD5_NAME.test(original)) {
        cb(null, original.toLowerCase())
        return
      }
      const ext = extname(original).toLowerCase() || '.bin'
      cb(null, `${Date.now()}_${randomBytes(6).toString('hex')}${ext}`)
    },
  })
}

// 把 multer 写出的绝对路径转成相对 UPLOAD_DIR 的路径（用于 /uploads 静态服务与 DB 存储）。
function relPath(file: Express.Multer.File): string {
  return relative(uploadRoot(), file.path).split('\\').join('/')
}

@Controller('content')
export class ContentController {
  constructor(
    private svc: ContentService,
    private redis: RedisService,
    @InjectRepository(Claim) private claimRepo: Repository<Claim>,
  ) {}

  @Get('list')
  async list(@Query() q: ListContentDto) {
    const data = await this.svc.list({ page: q.page, pageSize: q.page_size, tag: q.tag, type: q.type, auditStatus: q.audit_status || 'approved', keyword: q.keyword, sortBy: q.sort_by, order: q.order })
    return { code: 200, message: 'ok', data }
  }

  @Get('search')
  async search(@Query() q: ListContentDto) {
    if (!q.keyword?.trim()) return { code: 200, message: 'ok', data: { list: [], total: 0, page: 1, page_size: 20, total_page: 1 } }
    const data = await this.svc.list({ page: q.page, pageSize: q.page_size, keyword: q.keyword, auditStatus: 'approved' })
    return { code: 200, message: 'ok', data }
  }

  @Get('recommend')
  async recommend(@Query('count') count?: string, @Query('page') page?: string) {
    const data = await this.svc.recommend(Number(count) || 20, Number(page) || 1)
    return { code: 200, message: 'ok', data }
  }

  @Get('tags')
  async tags() {
    return { code: 200, message: 'ok', data: await this.svc.getAllTags() }
  }

  @Get('my')
  @UseGuards(AuthGuard)
  async my(@CurrentUser('uid') uid: number, @Query() q: ListContentDto) {
    const data = await this.svc.list({ page: q.page, pageSize: q.page_size, userId: uid, auditStatus: q.audit_status, type: q.type })
    return { code: 200, message: 'ok', data }
  }

  @Get(':id')
  async detail(@Param('id') id: string, @Query('silent') silent?: string) {
    const data = await this.svc.detail(Number(id), silent === '1')
    return { code: 200, message: 'ok', data }
  }

  @Post('upload')
  @UseGuards(AuthGuard)
  @UseInterceptors(FileInterceptor('file', { storage: uploadStorage() }))
  async upload(@UploadedFile() file: Express.Multer.File | undefined, @Body() dto: UploadContentDto, @CurrentUser('uid') uid: number) {
    const filePath = file ? relPath(file) : undefined
    const tags = Array.isArray(dto.tags) ? dto.tags : typeof dto.tags === 'string' ? dto.tags.split(',').map(s => s.trim()).filter(Boolean) : []
    // 2026-07-29 改造：type 不再由调用方指定；按"是否有 file"自动设 image / text（必须二选一）。
    // 兜底校验：title 必填（DTO 已 @Length(1,200)），content 与 file 至少一个。
    if (!dto.content?.trim() && !file) {
      return { code: 400, message: '描述正文与媒体文件至少填一项', data: null }
    }
    const type = file ? 'image' : 'text'
    const result = await this.svc.create(
      {
        title: dto.title, type, content: dto.content, filePath, fileSize: file?.size, tags, userId: uid,
      },
      { absPath: file?.path },
    )
    return { code: 200, message: '上传成功', data: result }
  }

  // 游客快速上传（免登录）：邮箱 + 昵称标识上传者，落库 user_id=0。
  // 2026-07-29：放开文件类型到 image/* + video/*，单文件 ≤ 20MB，按 IP 每小时限 20 次。
  @Post('quick-upload')
  @UseInterceptors(FileInterceptor('file', {
    storage: uploadStorage(),
    limits: { fileSize: 20 * 1024 * 1024 },
    fileFilter: (_req, file, cb) => {
      if (file.mimetype?.startsWith('image/') || file.mimetype?.startsWith('video/')) cb(null, true)
      else cb(new BadRequestException('仅支持图片或视频文件'), false)
    },
  }))
  async quickUpload(@UploadedFile() file: Express.Multer.File | undefined, @Body() dto: QuickUploadDto, @Req() req: Request) {
    // 兜底：title 必填 + content 与 file 至少一个
    if (!dto.content?.trim() && !file) {
      return { code: 400, message: '描述正文与媒体文件至少填一项', data: null }
    }

    // 检测登录态：已登录用户用真实 id，未登录用游客信息
    const sessionID = req.cookies?.['session_id'] || ''
    const uid = await this.svc.resolveUserId(sessionID)

    const ip = req.ip || req.socket?.remoteAddress || 'unknown'
    // 仅在有文件时按 IP 限频（纯描述没有占用带宽的风控必要）
    if (file) {
      const count = await this.redis.incrWithTTL(`quick_upload:ip:${ip}`, 3600)
      if (count > 20) {
        // 限频拒绝时清理 multer 已落盘的文件，避免孤儿文件堆积。
        unlink(file.path, () => {})
        return { code: 429, message: '上传过于频繁，请一小时后再试', data: null }
      }
    }

    const tags = Array.isArray(dto.tags) ? dto.tags : typeof dto.tags === 'string' ? dto.tags.split(',').map(s => s.trim()).filter(Boolean) : []
    const type = file ? 'image' : 'text'
    const result = await this.svc.create(
      {
        title: dto.title, type, content: dto.content,
        filePath: file ? relPath(file) : undefined, fileSize: file?.size, tags,
        userId: uid || 0,
        guestNickname: uid ? undefined : (dto.nickname || '').trim(),
        guestEmail: uid ? undefined : (dto.email || '').trim().toLowerCase(),
      },
      { absPath: file?.path },
    )
    return { code: 200, message: '上传成功', data: result }
  }

  // 富文本图片上传（前端 Markdown 编辑器的图片插入）。返回 image_url 供插入。
  @Post('upload-image')
  @UseGuards(AuthGuard)
  @UseInterceptors(FileInterceptor('file', { storage: uploadStorage() }))
  async uploadImage(@UploadedFile() file: Express.Multer.File | undefined, @CurrentUser('uid') uid: number) {
    if (!file) return { code: 400, message: '未收到文件', data: null }
    const rel = relPath(file)
    const saved = await this.svc.create(
      { title: file.originalname, type: 'image', filePath: rel, fileSize: file.size, tags: [], userId: uid },
      { absPath: file.path },
    )
    return {
      code: 200,
      message: 'ok',
      data: {
        id: saved.id,
        filename: file.originalname,
        file_size: file.size,
        image_url: ContentService.fileUrl(rel),
        upload_time: new Date().toISOString(),
      },
    }
  }

  @Put(':id')
  @UseGuards(AuthGuard)
  @UseInterceptors(FileInterceptor('file', { storage: uploadStorage() }))
  async update(@Param('id') id: string, @Body() dto: UpdateContentDto, @CurrentUser('uid') uid: number, @CurrentUser('is_admin') isAdmin: boolean) {
    const row = await this.svc.getContentRow(Number(id))
    if (!row) return { code: 404, message: '内容不存在' }
    if (uid !== row.user_id && !isAdmin) return { code: 403, message: '无权修改该内容' }
    const tags = Array.isArray(dto.tags) ? dto.tags : typeof dto.tags === 'string' ? dto.tags.split(',').map(s => s.trim()).filter(Boolean) : undefined
    const data = await this.svc.update(Number(id), { title: dto.title, content: dto.content, url: dto.url, tags })
    return { code: 200, message: '更新成功', data }
  }

  @Delete(':id')
  @UseGuards(AuthGuard)
  async delete(@Param('id') id: string, @CurrentUser('uid') uid: number, @CurrentUser('is_admin') isAdmin: boolean) {
    const row = await this.svc.getContentRow(Number(id))
    if (!row) return { code: 404, message: '内容不存在' }
    if (uid !== row.user_id && !isAdmin) return { code: 403, message: '无权删除该内容' }
    await this.svc.softDelete(Number(id))
    return { code: 200, message: '已删除', data: null }
  }

  @Post(':content_id/claim')
  @UseGuards(AuthGuard)
  async claim(@Param('content_id') contentId: string, @Body() dto: ClaimDto, @CurrentUser('uid') uid: number) {
    if (!(await this.svc.contentExists(Number(contentId)))) return { code: 404, message: '内容不存在' }
    const claim = this.claimRepo.create({ content_id: Number(contentId), user_id: uid, reason: dto.reason || '', status: 'pending' })
    const saved = await this.claimRepo.save(claim)
    return { code: 200, message: '认领申请已提交', data: { id: saved.id } }
  }
}
