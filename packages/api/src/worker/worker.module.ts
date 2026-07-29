import { Module, Global } from '@nestjs/common'
import { ClientsModule, Transport } from '@nestjs/microservices'
import { ConfigModule, ConfigService } from '@nestjs/config'
import { WorkerService } from './worker.service'
import { join } from 'path'

@Global()
@Module({
  imports: [
    ClientsModule.registerAsync([
      {
        name: 'WORKER_SERVICE',
        imports: [ConfigModule],
        inject: [ConfigService],
        useFactory: (cfg: ConfigService) => ({
          transport: Transport.GRPC,
          options: {
            package: 'xqecz',
            protoPath: join(__dirname, '../../../../proto/xqecz.proto'),
            url: cfg.get('WORKER_URL', 'localhost:50051'),
            // 使用 snake_case 字段名（与 proto 定义、Go worker、worker.service.ts 映射一致）。
            loader: { keepCase: true },
            // gRPC 全局调用超时（30 秒），防止 worker 不可达时请求长期挂起。
            // 可通过 WORKER_GRPC_TIMEOUT_MS 环境变量覆盖。
            channelOptions: {
              'grpc.keepalive_time_ms': 30_000,
              'grpc.keepalive_timeout_ms': 10_000,
            },
          },
        }),
      },
    ]),
  ],
  providers: [WorkerService],
  exports: [WorkerService],
})
export class WorkerModule {}
