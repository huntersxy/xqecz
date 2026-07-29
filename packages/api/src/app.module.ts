import { Module } from '@nestjs/common'
import { ConfigModule, ConfigService } from '@nestjs/config'
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
    ConfigModule.forRoot({ isGlobal: true }),
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
