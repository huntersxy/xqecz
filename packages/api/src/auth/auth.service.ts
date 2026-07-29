import { Injectable, ConflictException, UnauthorizedException, ForbiddenException, OnModuleInit, Logger } from '@nestjs/common'
import { InjectRepository } from '@nestjs/typeorm'
import { Repository } from 'typeorm'
import * as bcrypt from 'bcryptjs'
import { randomBytes } from 'crypto'
import { User } from '../entities'
import { RedisService } from '../redis/redis.service'

@Injectable()
export class AuthService implements OnModuleInit {
  private readonly logger = new Logger(AuthService.name)

  constructor(
    @InjectRepository(User) private userRepo: Repository<User>,
    private redis: RedisService,
  ) {}

  /** 启动时检查：若 users 表为空，自动生成 admin 账号并将随机密码打印到日志。 */
  async onModuleInit() {
    const count = await this.userRepo.count()
    if (count > 0) return

    const password = this.generatePassword()
    const hash = await bcrypt.hash(password, 10)
    const user = this.userRepo.create({ username: 'admin', password: hash, is_admin: 1, is_banned: 0 })
    const saved = await this.userRepo.save(user)
    this.logger.warn(`========================================`)
    this.logger.warn(`管理员账号已自动生成`)
    this.logger.warn(`用户名: admin`)
    this.logger.warn(`密码:   ${password}`)
    this.logger.warn(`请登录后立即修改密码！`)
    this.logger.warn(`========================================`)
  }

  /** 生成 16 字节随机密码（base64url 编码，22 字符）。 */
  private generatePassword(): string {
    return randomBytes(16).toString('base64url')
  }

  private generateSessionID(): string {
    return randomBytes(32).toString('hex')
  }

  async register(username: string, email: string, password: string) {
    const exists = await this.userRepo.findOne({ where: { username } })
    if (exists) throw new ConflictException('用户名已存在')

    // 邮箱唯一性校验
    const emailExists = await this.userRepo.findOne({ where: { email } })
    if (emailExists) throw new ConflictException('邮箱已被注册')

    const hash = await bcrypt.hash(password, 10)
    const user = this.userRepo.create({ username, email, password: hash, is_admin: 0, is_banned: 0 })
    const saved = await this.userRepo.save(user)
    return { user_id: saved.id }
  }

  async login(username: string, password: string) {
    const user = await this.userRepo.findOne({ where: { username }, select: ['id', 'username', 'email', 'password', 'is_admin', 'is_banned'] })
    if (!user || !(await bcrypt.compare(password, user.password)))
      throw new UnauthorizedException('用户名或密码错误')
    if (user.is_banned) throw new ForbiddenException('账号已被封禁')

    const sessionID = this.generateSessionID()
    await this.redis.setSession(sessionID, user.id)
    // 老用户没有邮箱时标记 needs_email，前端弹窗强制补填
    const needsEmail = !user.email
    return {
      sessionID,
      user: { id: user.id, username: user.username, email: user.email || undefined, is_admin: !!user.is_admin },
      needs_email: needsEmail,
    }
  }

  async logout(sessionID: string) {
    if (sessionID) await this.redis.delSession(sessionID)
  }

  async getMe(uid: number) {
    const user = await this.userRepo.findOne({ where: { id: uid } })
    if (!user) throw new UnauthorizedException('用户不存在')
    return { id: user.id, username: user.username, email: user.email || undefined, is_admin: !!user.is_admin, is_banned: !!user.is_banned, created_at: user.created_at, updated_at: user.updated_at }
  }

  async updateEmail(uid: number, email: string) {
    const emailExists = await this.userRepo.findOne({ where: { email } })
    if (emailExists && emailExists.id !== uid) throw new ConflictException('邮箱已被使用')
    await this.userRepo.update(uid, { email })
    return { message: '邮箱更新成功' }
  }

  /** 修改密码（需验证旧密码），新密码打印到日志。 */
  async changePassword(uid: number, oldPassword: string, newPassword: string) {
    const user = await this.userRepo.findOne({ where: { id: uid }, select: ['id', 'username', 'password'] })
    if (!user) throw new UnauthorizedException('用户不存在')
    if (!(await bcrypt.compare(oldPassword, user.password)))
      throw new UnauthorizedException('旧密码错误')

    const hash = await bcrypt.hash(newPassword, 10)
    await this.userRepo.update(uid, { password: hash })
    this.logger.warn(`用户 [${user.username}] 修改了密码，新密码: ${newPassword}`)
    return { message: '密码修改成功' }
  }
}
