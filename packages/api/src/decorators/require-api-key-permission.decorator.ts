import { SetMetadata } from '@nestjs/common'

export const API_KEY_PERMISSION_KEY = 'api-key-permission'

/**
 * 声明该接口需要 API 密钥具备的权限（upload / read / delete）。
 * 仅对通过 X-API-Key 认证的请求生效；Session 登录用户不受此限制。
 */
export const RequireApiKeyPermission = (permission: string) =>
  SetMetadata(API_KEY_PERMISSION_KEY, permission)
