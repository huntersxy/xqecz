import { NestFactory } from '@nestjs/core'
import { ValidationPipe } from '@nestjs/common'
import cookieParser from 'cookie-parser'
import express from 'express'
import { readFileSync, mkdirSync } from 'fs'
import { join } from 'path'
import { AppModule } from './app.module'
import { UPLOAD_DIR, THUMB_DIR, IMAGES_DIR } from './paths'

// 在 ConfigModule 加载前，先把 .env 注入 process.env。
// 这样 main.ts 的静态目录、content.controller 的上传目录（均读 process.env.UPLOAD_DIR）
// 与 content.service（读 ConfigService）会指向同一目录，避免此前「一个写 /app、一个读 /tmp」的错配。
try {
  const envPath = join(__dirname, '..', '.env')
  const txt = readFileSync(envPath, 'utf8')
  for (const line of txt.split(/\r?\n/)) {
    const m = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$/)
    if (m && !line.trimStart().startsWith('#') && process.env[m[1]] === undefined) {
      process.env[m[1]] = m[2]
    }
  }
} catch {
  // .env 缺失时忽略，下方会使用项目相对 data 目录
}

// 把「项目相对」的 data 目录解析结果注入 process.env，
// 确保 ConfigModule（content.service 经 ConfigService 读取）也拿到同一路径。
process.env.UPLOAD_DIR = process.env.UPLOAD_DIR || UPLOAD_DIR

async function bootstrap() {
  const app = await NestFactory.create(AppModule, {
    logger: ['error', 'warn', 'log'],
  })

  app.setGlobalPrefix('api')
  app.use(cookieParser())
  app.enableCors({
    origin: (process.env.CORS_ORIGINS || 'http://localhost:5173').split(','),
    credentials: true,
  })

  // 静态服务上传文件（与 Go Worker 共享卷）。统一存放于项目内 data 目录（相对项目根解析）。
  // 三目录同级：/uploads → data/uploads（原文件，扁平）、/thumbs → data/thumbs、/images → data/images
  mkdirSync(UPLOAD_DIR, { recursive: true })
  mkdirSync(THUMB_DIR, { recursive: true })
  mkdirSync(IMAGES_DIR, { recursive: true })
  app.use('/uploads', express.static(UPLOAD_DIR))
  app.use('/thumbs', express.static(THUMB_DIR))
  app.use('/images', express.static(IMAGES_DIR))
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      transformOptions: { enableImplicitConversion: true },
    }),
  )

  // 优雅关停：SIGINT/SIGTERM 时关闭 HTTP server 与 TypeORM/Redis 连接
  app.enableShutdownHooks()

  // 自适应端口：优先用注入的 PORT（dev 编排器已预先探测空闲），
  // 若仍被占用（竞态/单独启动 dev:api 时），向上顺延重试。
  const preferred = Number(process.env.PORT) || 3000
  let port = preferred
  for (; port < preferred + 50; port++) {
    try {
      await app.listen(port)
      break
    } catch (err: unknown) {
      if ((err as { code?: string })?.code !== 'EADDRINUSE') throw err
      console.warn(`[xqecz-api] port ${port} in use, trying ${port + 1}...`)
    }
  }
  process.env.PORT = String(port)
  console.log(`[xqecz-api] listening on http://localhost:${port}`)
}
bootstrap()
