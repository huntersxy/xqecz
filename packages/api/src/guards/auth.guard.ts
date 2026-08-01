import { Injectable, CanActivate, ExecutionContext, UnauthorizedException } from '@nestjs/common'
import { createHash } from 'crypto'
import { RedisService } from '../redis/redis.service'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { User, ApiKey } from '../entities'

@Injectable()
export class AuthGuard implements CanActivate {
  constructor(
    private redis: RedisService,
    @InjectRepository(User) private userRepo: Repository<User>,
    @InjectRepository(ApiKey) private apiKeyRepo: Repository<ApiKey>,
  ) {}

  async canActivate(ctx: ExecutionContext): Promise<boolean> {
    const req = ctx.switchToHttp().getRequest()

    // API 密钥认证：请求头 X-API-Key
    const apiKeyHeader = req.headers?.['x-api-key']
    if (typeof apiKeyHeader === 'string' && apiKeyHeader.trim()) {
      return this.authenticateApiKey(req, apiKeyHeader.trim())
    }

    // Session 认证（web 前端）
    const sessionID = req.cookies?.['session_id']
    if (!sessionID) throw new UnauthorizedException('未登录')

    const userID = await this.redis.getSession(sessionID)
    if (!userID) throw new UnauthorizedException('登录已过期')

    const user = await this.userRepo.findOne({ where: { id: userID } })
    if (!user) throw new UnauthorizedException('用户不存在')
    if (user.is_banned) throw new UnauthorizedException('账号已被封禁')

    req.user = { uid: user.id, username: user.username, is_admin: !!user.is_admin }
    return true
  }

  private async authenticateApiKey(req: any, raw: string): Promise<boolean> {
    const hash = createHash('sha256').update(raw).digest('hex')
    const row = await this.apiKeyRepo.findOne({ where: { key_hash: hash, is_active: 1 } })
    if (!row) throw new UnauthorizedException('API 密钥无效')

    let permissions: string[] = []
    try {
      const parsed = JSON.parse(row.permissions || '[]')
      if (Array.isArray(parsed)) permissions = parsed
    } catch { /* 权限字段异常按空权限处理 */ }

    req.user = {
      // bigint 主键返回字符串，与 Session 路径保持一致（业务里 uid 与 row.user_id 直接比较）
      uid: row.user_id,
      username: 'API',
      is_admin: false,
      api_key: { id: row.id, permissions },
    }

    // 记录最后使用时间（每分钟至多写一次，避免高频请求打爆 DB）
    const lastUsed = row.last_used_at ? new Date(row.last_used_at).getTime() : 0
    if (Date.now() - lastUsed > 60_000) {
      void this.apiKeyRepo.update(row.id, { last_used_at: new Date() }).catch(() => {})
    }
    return true
  }
}
