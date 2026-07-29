import { Injectable, CanActivate, ExecutionContext, UnauthorizedException } from '@nestjs/common'
import { RedisService } from '../redis/redis.service'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { User } from '../entities'

@Injectable()
export class AuthGuard implements CanActivate {
  constructor(
    private redis: RedisService,
    @InjectRepository(User) private userRepo: Repository<User>,
  ) {}

  async canActivate(ctx: ExecutionContext): Promise<boolean> {
    const req = ctx.switchToHttp().getRequest()
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
}
