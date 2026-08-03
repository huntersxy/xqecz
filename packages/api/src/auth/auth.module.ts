import { Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { User, ApiKey } from '../entities'
import { AuthService } from './auth.service'
import { AuthController } from './auth.controller'
import { AuthGuard } from '../guards/auth.guard'
import { OptionalAuthGuard } from '../guards/optional-auth.guard'
import { ApiKeyPermissionGuard } from '../guards/api-key-permission.guard'

// 提供 AuthService 及两个有依赖（UserRepository + RedisService）的 guard。
// 导出 TypeOrmModule.forFeature([User])：各使用 guard 的模块 import 本模块后，
// 在其自身语境即可获得 UserRepository，满足 @UseGuards 类引用 guard 的依赖解析。
@Module({
  imports: [TypeOrmModule.forFeature([User, ApiKey])],
  controllers: [AuthController],
  providers: [AuthService, AuthGuard, OptionalAuthGuard, ApiKeyPermissionGuard],
  exports: [AuthService, AuthGuard, OptionalAuthGuard, ApiKeyPermissionGuard, TypeOrmModule],
})
export class AuthModule {}
