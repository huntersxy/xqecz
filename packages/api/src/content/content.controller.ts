import { Controller, Get, Post, Put, Delete, Param, Query, Body, Req, Res, UseGuards, UploadedFile, UseInterceptors, BadRequestException, NotFoundException, ForbiddenException } from '@nestjs/common'
import { FileInterceptor } from '@nestjs/platform-express'
import { diskStorage } from 'multer'
import { basename, extname, relative } from 'path'
import { randomBytes } from 'crypto'
import { mkdirSync, unlink } from 'fs'
import type { Request, Response } from 'express'
import { ContentService } from './content.service'
import { AuthGuard } from '../guards/auth.guard'
import { OptionalAuthGuard } from '../guards/optional-auth.guard'
import { ApiKeyPermissionGuard } from '../guards/api-key-permission.guard'
import { CurrentUser } from '../decorators/current-user.decorator'
import { RequireApiKeyPermission } from '../decorators/require-api-key-permission.decorator'
import { UploadContentDto, UpdateContentDto, ListContentDto, ClaimDto, QuickUploadDto } from './dto'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { Claim, ContentLike, ContentFavorite } from '../entities'
import { RedisService } from '../redis/redis.service'
import { UPLOAD_DIR } from '../paths'

// 共享卷根目录（与 Go Worker 挂载同一路径，由项目根相对解析）。multer 把文件落在这里，api 再把它传给 worker 处理。
function uploadRoot(): string {
  return process.env.UPLOAD_DIR || UPLOAD_DIR
}

// 前后端统一：单文件最大 20MB（快速上传/普通上传/富文本插图一致）。
const MAX_UPLOAD_SIZE = 20 * 1024 * 1024

// 媒体类型校验：仅允许图片/视频。
function mediaFileFilter(
  _req: Request,
  file: Express.Multer.File,
  cb: (error: Error | null, acceptFile: boolean) => void,
) {
  if (file.mimetype?.startsWith('image/') || file.mimetype?.startsWith('video/')) cb(null, true)
  else cb(new BadRequestException('仅支持图片或视频文件'), false)
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
    @InjectRepository(ContentLike) private likeRepo: Repository<ContentLike>,
    @InjectRepository(ContentFavorite) private favRepo: Repository<ContentFavorite>,
  ) {}

  @Get('list')
  async list(@Query() q: ListContentDto) {
    const data = await this.svc.list({
      page: q.page, pageSize: q.page_size, tag: q.tag, type: q.type, keyword: q.keyword, sortBy: q.sort_by, order: q.order,
      // 公开可见范围：已通过 + 审核中；rejected 不对外展示。
      auditStatuses: q.audit_status ? undefined : ['approved', 'pending'],
      auditStatus: q.audit_status,
    })
    return { code: 200, message: 'ok', data }
  }

  @Get('search')
  async search(@Query() q: ListContentDto) {
    if (!q.keyword?.trim()) return { code: 200, message: 'ok', data: { list: [], total: 0, page: 1, page_size: 20, total_page: 1 } }
    const data = await this.svc.list({ page: q.page, pageSize: q.page_size, keyword: q.keyword, auditStatuses: ['approved', 'pending'] })
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
  @UseGuards(OptionalAuthGuard)
  async detail(
    @Param('id') id: string,
    @Query('silent') silent?: string,
    @CurrentUser() viewer?: { uid: number | string; is_admin?: boolean },
  ) {
    const data = await this.svc.detail(Number(id), silent === '1', viewer)
    return { code: 200, message: 'ok', data }
  }

  @Post('upload')
  @UseGuards(AuthGuard, ApiKeyPermissionGuard)
  @RequireApiKeyPermission('upload')
  @UseInterceptors(FileInterceptor('file', { storage: uploadStorage(), limits: { fileSize: MAX_UPLOAD_SIZE }, fileFilter: mediaFileFilter }))
  async upload(@UploadedFile() file: Express.Multer.File | undefined, @Body() dto: UploadContentDto, @CurrentUser('uid') uid: number, @CurrentUser('is_admin') isAdmin: boolean) {
    const tags = Array.isArray(dto.tags) ? dto.tags : typeof dto.tags === 'string' ? dto.tags.split(',').map(s => s.trim()).filter(Boolean) : []
    let filePath = file ? relPath(file) : undefined
    let fileSize = file?.size
    let absPath = file?.path
    if (file) {
      const prepared = await this.svc.prepareOriginalFile(file)
      if (prepared) {
        filePath = prepared.relPath
        fileSize = prepared.size
        absPath = prepared.absPath
      }
    }
    // 2026-07-29 改造：type 不再由调用方指定；按"是否有 file"自动设 image / text（必须二选一）。
    // 兜底校验：title 必填（DTO 已 @Length(1,200)），content 与 file 至少一个。
    if (!dto.content?.trim() && !file) {
      return { code: 400, message: '描述正文与媒体文件至少填一项', data: null }
    }
    const type = file ? 'image' : 'text'
    const result = await this.svc.create(
      {
        title: dto.title, type, content: dto.content, filePath, fileSize, tags, userId: uid,
        // 管理员上传直接过审；其余用户进入审核列表。
        auditStatus: isAdmin ? 'approved' : 'pending',
      },
      { absPath },
    )
    return { code: 200, message: '上传成功', data: result }
  }

  // 游客快速上传（免登录）：邮箱 + 昵称标识上传者，落库 user_id=0。
  // 2026-07-29：放开文件类型到 image/* + video/*，单文件 ≤ 20MB，按 IP 每小时限 20 次。
  @Post('quick-upload')
  @UseInterceptors(FileInterceptor('file', {
    storage: uploadStorage(),
    limits: { fileSize: MAX_UPLOAD_SIZE },
    fileFilter: mediaFileFilter,
  }))
  async quickUpload(@UploadedFile() file: Express.Multer.File | undefined, @Body() dto: QuickUploadDto, @Req() req: Request) {
    // 兜底：title 必填 + content 与 file 至少一个
    if (!dto.content?.trim() && !file) {
      return { code: 400, message: '描述正文与媒体文件至少填一项', data: null }
    }

    // 检测登录态：已登录用户用真实 id，未登录用游客信息
    const sessionID = req.cookies?.['session_id'] || ''
    const user = await this.svc.resolveUser(sessionID)
    const uid = user ? Number(user.id) : 0

    // 反向代理后 req.ip 可能是代理 IP；优先取 X-Forwarded-For 最左段（真实客户端 IP）。
    const forwarded = req.headers['x-forwarded-for']
    const ip = (typeof forwarded === 'string' ? forwarded.split(',')[0].trim() : null)
      || req.ip
      || req.socket?.remoteAddress
      || 'unknown'
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
    // 按 mimetype 自动推导类型：video/* → video，image/* → image，无文件 → text
    const type = !file ? 'text' : file.mimetype?.startsWith('video/') ? 'video' : 'image'
    let filePath = file ? relPath(file) : undefined
    let fileSize = file?.size
    let absPath = file?.path
    if (file) {
      const prepared = await this.svc.prepareOriginalFile(file)
      if (prepared) {
        filePath = prepared.relPath
        fileSize = prepared.size
        absPath = prepared.absPath
      }
    }
    const result = await this.svc.create(
      {
        title: dto.title, type, content: dto.content,
        filePath, fileSize, tags,
        userId: uid || 0,
        // 仅管理员登录上传自动过审；游客与普通用户一律进入审核列表。
        auditStatus: uid && user?.is_admin ? 'approved' : 'pending',
        guestNickname: uid ? undefined : (dto.nickname || '').trim(),
        guestEmail: uid ? undefined : (dto.email || '').trim().toLowerCase(),
      },
      { absPath },
    )
    return { code: 200, message: '上传成功', data: result }
  }

  @Put(':id')
  @UseGuards(AuthGuard, ApiKeyPermissionGuard)
  @RequireApiKeyPermission('upload')
  @UseInterceptors(FileInterceptor('file', { storage: uploadStorage() }))
  async update(
    @Param('id') id: string,
    @Body() dto: UpdateContentDto,
    @UploadedFile() file: Express.Multer.File | undefined,
    @CurrentUser('uid') uid: number,
    @CurrentUser('is_admin') isAdmin: boolean,
  ) {
    const row = await this.svc.getContentRow(Number(id))
    if (!row) throw new NotFoundException('内容不存在')
    if (uid !== row.user_id && !isAdmin) throw new ForbiddenException('无权修改该内容')
    const tags = Array.isArray(dto.tags) ? dto.tags : typeof dto.tags === 'string' ? dto.tags.split(',').map(s => s.trim()).filter(Boolean) : undefined
    let filePath = file ? relPath(file) : undefined
    let fileSize = file?.size
    let absPath = file?.path
    if (file) {
      const prepared = await this.svc.prepareOriginalFile(file)
      if (prepared) {
        filePath = prepared.relPath
        fileSize = prepared.size
        absPath = prepared.absPath
      }
    }
    // 内容被编辑后重新进入审核列表；管理员编辑视为过审，并刷新推荐位。
    await this.svc.setAuditStatus(Number(id), isAdmin ? 'approved' : 'pending')
    const data = await this.svc.update(Number(id), {
      title: dto.title, content: dto.content, url: dto.url, tags,
      filePath,
      fileSize,
    })
    // 替换了文件则异步重建缩略图与压缩图
    if (file && filePath && (row.type === 'image' || row.type === 'video')) {
      void this.svc.processMedia(Number(id), absPath!, row.type)
    }
    return { code: 200, message: '更新成功', data }
  }

  @Delete(':id')
  @UseGuards(AuthGuard, ApiKeyPermissionGuard)
  @RequireApiKeyPermission('delete')
  async delete(@Param('id') id: string, @CurrentUser('uid') uid: number, @CurrentUser('is_admin') isAdmin: boolean) {
    const row = await this.svc.getContentRow(Number(id))
    if (!row) throw new NotFoundException('内容不存在')
    if (uid !== row.user_id && !isAdmin) throw new ForbiddenException('无权删除该内容')
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

  // ── 点赞 ──
  @Post(':content_id/like')
  @UseGuards(AuthGuard)
  async toggleLike(@Param('content_id') contentId: string, @CurrentUser('uid') uid: number) {
    const cid = Number(contentId)
    const existing = await this.likeRepo.findOne({ where: { content_id: cid, user_id: uid } })
    if (existing) {
      await this.likeRepo.remove(existing)
      const likeCount = await this.likeRepo.count({ where: { content_id: cid } })
      // 点赞变化影响推荐打分（like 权重高），立即刷新推荐位（Redis 分布式锁防抖）
      void this.svc.refreshRecommend().catch((e) => console.warn('[recommend] refresh after unlike failed:', (e as Error)?.message))
      return { code: 200, message: '已取消点赞', data: { liked: false, like_count: likeCount } }
    }
    const row = this.likeRepo.create({ content_id: cid, user_id: uid })
    await this.likeRepo.save(row)
    const likeCount = await this.likeRepo.count({ where: { content_id: cid } })
    // 点赞变化影响推荐打分（like 权重高），立即刷新推荐位（Redis 分布式锁防抖）
    void this.svc.refreshRecommend().catch((e) => console.warn('[recommend] refresh after like failed:', (e as Error)?.message))
    return { code: 200, message: '已点赞', data: { liked: true, like_count: likeCount } }
  }

  @Get(':content_id/like-status')
  @UseGuards(AuthGuard)
  async likeStatus(@Param('content_id') contentId: string, @CurrentUser('uid') uid: number) {
    const cid = Number(contentId)
    const [liked, favorited, likeCount] = await Promise.all([
      this.likeRepo.findOne({ where: { content_id: cid, user_id: uid } }).then(Boolean),
      this.favRepo.findOne({ where: { content_id: cid, user_id: uid } }).then(Boolean),
      this.likeRepo.count({ where: { content_id: cid } }),
    ])
    return { code: 200, message: 'ok', data: { liked, favorited, like_count: likeCount } }
  }

  // ── 收藏 ──
  @Post(':content_id/favorite')
  @UseGuards(AuthGuard)
  async toggleFavorite(@Param('content_id') contentId: string, @CurrentUser('uid') uid: number) {
    const cid = Number(contentId)
    const existing = await this.favRepo.findOne({ where: { content_id: cid, user_id: uid } })
    if (existing) {
      await this.favRepo.remove(existing)
      return { code: 200, message: '已取消收藏', data: { favorited: false } }
    }
    const row = this.favRepo.create({ content_id: cid, user_id: uid })
    await this.favRepo.save(row)
    return { code: 200, message: '已收藏', data: { favorited: true } }
  }
}
