import { join } from 'path'
import { Module } from '@nestjs/common'
import { ConfigModule, ConfigService } from '@nestjs/config'
import { PROJECT_ROOT } from './paths'
import { TypeOrmModule } from '@nestjs/typeorm'
import { ScheduleModule } from '@nestjs/schedule'
import { AuthModule } from './auth/auth.module'
import { ContentModule } from './content/content.module'
import { CommentModule } from './comment/comment.module'
import { PollModule } from './poll/poll.module'
import { AdminModule } from './admin/admin.module'
import { ApiKeyModule } from './api-key/api-key.module'
import { HealthController } from './health.controller'
import { RedisModule } from './redis/redis.module'
import { WorkerModule } from './worker/worker.module'

@Module({
  imports: [
    // .env 统一放项目根目录一份（API/Worker/Frontend 共用）。
    // Nest 默认只找进程 cwd（packages/api）下的 .env，这里显式指向项目根。
    // 注意：process.env 中已存在的变量优先级更高（dev 编排器注入的端口不会被覆盖）。
    ConfigModule.forRoot({ isGlobal: true, envFilePath: join(PROJECT_ROOT, '.env') }),
    TypeOrmModule.forRootAsync({
      imports: [ConfigModule],
      inject: [ConfigService],
      useFactory: (cfg: ConfigService) => ({
        type: 'mysql',
        host: cfg.get('MYSQL_HOST', 'localhost'),
        port: cfg.get<number>('MYSQL_PORT', 3306),
        username: cfg.get('MYSQL_USER', 'root'),
        password: cfg.get('MYSQL_PASSWORD', ''),
        database: cfg.get('MYSQL_DATABASE', 'xqecz'),
        autoLoadEntities: true,
        synchronize: false,
        charset: 'utf8mb4',
        // 远程 MySQL 连接池显式配置：防网络波动导致连接泄漏或池爆炸。
        // 注意：mysql2 3.x 已移除 acquireTimeout（无效选项，会告警），连接等待超时
        // 由 connectTimeout 覆盖；如需池获取超时可设 waitForConnections=false + queueLimit。
        extra: {
          connectionLimit: cfg.get<number>('MYSQL_POOL_SIZE', 5),
          connectTimeout: cfg.get<number>('MYSQL_CONNECT_TIMEOUT', 10000),
          waitForConnections: true,
          queueLimit: 0,
          idleTimeout: cfg.get<number>('MYSQL_IDLE_TIMEOUT', 60000),
          enableKeepAlive: true,
          keepAliveInitialDelay: 10000,
        },
      }),
    }),
    ScheduleModule.forRoot(),
    RedisModule,
    WorkerModule,
    AuthModule,
    ContentModule,
    CommentModule,
    PollModule,
    AdminModule,
    ApiKeyModule,
  ],
  controllers: [HealthController],
})
export class AppModule {}
