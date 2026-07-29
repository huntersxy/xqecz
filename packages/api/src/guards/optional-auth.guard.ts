import { Injectable, CanActivate, ExecutionContext } from '@nestjs/common'
import { RedisService } from '../redis/redis.service'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import { User } from '../entities'

@Injectable()
export class OptionalAuthGuard implements CanActivate {
  constructor(
    private redis: RedisService,
    @InjectRepository(User) private userRepo: Repository<User>,
  ) {}

  async canActivate(ctx: ExecutionContext): Promise<boolean> {
    const req = ctx.switchToHttp().getRequest()
    const sessionID = req.cookies?.['session_id']
    if (sessionID) {
      const userID = await this.redis.getSession(sessionID)
      if (userID) {
        const user = await this.userRepo.findOne({ where: { id: userID } })
        if (user && !user.is_banned)
          req.user = { uid: user.id, username: user.username, is_admin: !!user.is_admin }
      }
    }
    return true
  }
}
