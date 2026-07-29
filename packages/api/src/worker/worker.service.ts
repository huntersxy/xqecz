import { Inject, Injectable, OnModuleInit } from '@nestjs/common'
import { ClientGrpc } from '@nestjs/microservices'
import { firstValueFrom, timeout, catchError, of } from 'rxjs'

/** gRPC 调用默认超时 30 秒（可通过 WORKER_CALL_TIMEOUT_MS 环境变量覆盖）。 */
const DEFAULT_CALL_TIMEOUT = Number(process.env.WORKER_CALL_TIMEOUT_MS) || 30_000

interface WorkerGrpcService {
  health(data: {}): any
  generateThumbnail(data: { file_path: string; content_type: string }): any
  compressImage(data: { file_path: string }): any
  uploadToS3(data: { file_path: string; content_type: string }): any
  fetchLinkPreview(data: { url: string }): any
  refreshRecommend(data: { items: { content_id: number; created_at_unix: number; view_count: number }[] }): any
}

@Injectable()
export class WorkerService implements OnModuleInit {
  private svc!: WorkerGrpcService

  constructor(@Inject('WORKER_SERVICE') private client: ClientGrpc) {}

  onModuleInit() {
    this.svc = this.client.getService<WorkerGrpcService>('WorkerService')
  }

  private callWithTimeout<T>(observable$: any): Promise<T> {
    return firstValueFrom(
      observable$.pipe(
        timeout(DEFAULT_CALL_TIMEOUT),
        catchError((err) => {
          if (err.name === 'TimeoutError') {
            console.warn(`[worker] gRPC call timed out after ${DEFAULT_CALL_TIMEOUT}ms`)
            return of({ success: false, error: `timeout after ${DEFAULT_CALL_TIMEOUT}ms` } as any)
          }
          throw err
        }),
      ),
    )
  }

  async health(): Promise<any> {
    return this.callWithTimeout(this.svc.health({}))
  }

  async generateThumbnail(filePath: string, contentType: string): Promise<any> {
    return this.callWithTimeout(this.svc.generateThumbnail({ file_path: filePath, content_type: contentType }))
  }

  async compressImage(filePath: string): Promise<any> {
    return this.callWithTimeout(this.svc.compressImage({ file_path: filePath }))
  }

  async uploadToS3(filePath: string, contentType: string): Promise<any> {
    return this.callWithTimeout(this.svc.uploadToS3({ file_path: filePath, content_type: contentType }))
  }

  async fetchLinkPreview(url: string): Promise<any> {
    return this.callWithTimeout(this.svc.fetchLinkPreview({ url }))
  }

  /**
   * 让无状态 worker 对给定内容打分（纯计算，不碰 DB/Redis）。
   * 返回 { contentId, score }[]，由调用方（ContentService）写入 Redis ZSet。
   */
  async refreshRecommend(items: { contentId: number; createdAtUnix: number; viewCount: number }[]): Promise<{ contentId: number; score: number }[]> {
    const resp: any = await this.callWithTimeout(
      this.svc.refreshRecommend({
        items: items.map((i) => ({ content_id: i.contentId, created_at_unix: i.createdAtUnix, view_count: i.viewCount })),
      }),
    )
    const results = (resp?.results || []) as { content_id: number; score: number }[]
    return results.map((r) => ({ contentId: r.content_id, score: r.score }))
  }
}
