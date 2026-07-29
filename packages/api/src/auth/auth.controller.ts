import { Controller, Post, Get, Put, Body, Res, Req, UseGuards } from '@nestjs/common'
import type { Response, Request } from 'express'
import { AuthService } from './auth.service'
import { RegisterDto, LoginDto, ChangePasswordDto, UpdateEmailDto } from './dto'
import { AuthGuard } from '../guards/auth.guard'
import { CurrentUser } from '../decorators/current-user.decorator'

@Controller('auth')
export class AuthController {
  constructor(private svc: AuthService) {}

  @Post('register')
  async register(@Body() dto: RegisterDto) {
    const result = await this.svc.register(dto.username, dto.email, dto.password)
    return { code: 200, message: '注册成功', data: result }
  }

  @Post('login')
  async login(@Body() dto: LoginDto, @Res({ passthrough: true }) res: Response) {
    const { sessionID, user, needs_email } = await this.svc.login(dto.username, dto.password)
    res.cookie('session_id', sessionID, { httpOnly: true, sameSite: 'lax', path: '/', maxAge: 30 * 24 * 3600 * 1000 })
    return { code: 200, message: '登录成功', data: { user, needs_email: !!needs_email } }
  }

  @Post('logout')
  async logout(@Req() req: Request, @Res({ passthrough: true }) res: Response) {
    await this.svc.logout(req.cookies?.['session_id'] || '')
    res.clearCookie('session_id', { path: '/' })
    return { code: 200, message: '已退出登录', data: null }
  }

  @Post('change-password')
  @UseGuards(AuthGuard)
  async changePassword(@CurrentUser('uid') uid: number, @Body() dto: ChangePasswordDto) {
    const result = await this.svc.changePassword(uid, dto.oldPassword, dto.newPassword)
    return { code: 200, message: '密码修改成功', data: result }
  }

  @Get('me')
  @UseGuards(AuthGuard)
  async me(@CurrentUser('uid') uid: number) {
    const user = await this.svc.getMe(uid)
    return { code: 200, message: 'ok', data: user }
  }

  @Put('email')
  @UseGuards(AuthGuard)
  async updateEmail(@CurrentUser('uid') uid: number, @Body() dto: UpdateEmailDto) {
    const result = await this.svc.updateEmail(uid, dto.email)
    return { code: 200, message: result.message, data: null }
  }
}
