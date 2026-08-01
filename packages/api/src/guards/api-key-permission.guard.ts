import { Injectable, CanActivate, ExecutionContext, ForbiddenException } from '@nestjs/common'
import { Reflector } from '@nestjs/core'
import { API_KEY_PERMISSION_KEY } from '../decorators/require-api-key-permission.decorator'

/**
 * API 密钥权限校验：当请求通过 X-API-Key 认证时，检查密钥声明的 permissions
 * 是否包含接口要求的权限；Session 登录用户直接放行（走各自业务权限判断）。
 */
@Injectable()
export class ApiKeyPermissionGuard implements CanActivate {
  constructor(private reflector: Reflector) {}

  canActivate(ctx: ExecutionContext): boolean {
    const permission = this.reflector.get<string | undefined>(
      API_KEY_PERMISSION_KEY,
      ctx.getHandler(),
    )
    if (!permission) return true

    const req = ctx.switchToHttp().getRequest()
    const apiKey = req.user?.api_key as { permissions?: string[] } | undefined
    if (apiKey && !apiKey.permissions?.includes(permission)) {
      throw new ForbiddenException(`API 密钥缺少 ${permission} 权限`)
    }
    return true
  }
}
