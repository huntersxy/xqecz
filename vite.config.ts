import { fileURLToPath, URL } from 'node:url'
import type { IncomingMessage, ClientRequest } from 'node:http'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 获取当前日期作为构建日期
const buildDate = new Date().toISOString().split('T')[0]

// https://vite.dev/config/
export default defineConfig(async ({ mode }) => {
  const isDev = mode === 'development'
  const plugins: any[] = [vue()]

  if (isDev) {
    plugins.push((await import('vite-plugin-vue-devtools')).default())
  } else {
    plugins.push(
      (await import('vite-plugin-imagemin')).default({
        mozjpeg: { quality: 80 },
        pngquant: { quality: [0.8, 0.9] },
        svgo: { plugins: [{ name: 'removeViewBox', active: false }] },
      }),
      (await import('vite-plugin-compression2')).default({
        algorithms: ['gzip'],
        threshold: 1024,
        deleteOriginalAssets: false,
      }),
    )
  }

  return {
    plugins,
    define: {
      'import.meta.env.VITE_BUILD_DATE': JSON.stringify(buildDate),
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
          secure: false,
          configure(proxy: any) {
            proxy.on('error', (err: Error, _req: IncomingMessage, _res: any) => {
              console.log('proxy error', err)
            })
            proxy.on('proxyReq', (_proxyReq: ClientRequest, req: IncomingMessage) => {
              console.log('Sending Request to the Target:', req.method, req.url)
            })
            proxy.on('proxyRes', (proxyRes: IncomingMessage, req: IncomingMessage) => {
              console.log('Received Response from the Target:', proxyRes.statusCode, req.url)
            })
          },
        },
      },
    },
  }
})
