import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers'
import { ViteImageOptimizer } from 'vite-plugin-image-optimizer'

const buildDate = new Date().toISOString().split('T')[0]

export default defineConfig(async ({ mode }) => {
  const isDev = mode === 'development'
  // 开发/预览态代理目标：默认 NestJS API（http://localhost:3000），可用 VITE_PROXY_TARGET 覆盖。
  // vite preview 的代理默认继承 server.proxy，因此生产预览同样走这里的配置。
  const proxyTarget = process.env.VITE_PROXY_TARGET || 'http://localhost:3000'
  const plugins: import('vite').PluginOption[] = [
    vue(),
    tailwindcss(),
    AutoImport({
      imports: ['vue', 'vue-router'],
      dts: 'src/auto-imports.d.ts',
    }),
    Components({
      dts: 'src/components.d.ts',
      resolvers: [AntDesignVueResolver({ importStyle: false })],
    }),
  ]

  if (isDev) {
    plugins.push((await import('vite-plugin-vue-devtools')).default())
  } else {
    plugins.push(
      ViteImageOptimizer({
        png: { quality: 80 },
        jpeg: { quality: 80 },
        jpg: { quality: 80 },
        webp: { quality: 80 },
        avif: { quality: 80 },
        svg: {
          multipass: true,
          plugins: [
            {
              name: 'preset-default',
              params: {
                overrides: {
                  cleanupNumericValues: false,
                  cleanupIds: { minify: false, remove: false },
                  convertPathData: false,
                },
              },
            },
            'sortAttrs',
            {
              name: 'addAttributesToSVGElement',
              params: { attributes: [{ xmlns: 'http://www.w3.org/2000/svg' }] },
            },
          ],
        },
      }),
    )
  }

  return {
    // .env 统一放项目根目录一份（VITE_ 前缀的变量对客户端生效）
    envDir: fileURLToPath(new URL('../..', import.meta.url)),
    plugins,
    define: {
      'import.meta.env.VITE_BUILD_DATE': JSON.stringify(buildDate),
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    build: {
      // 2026-07-29 恢复：此前 emptyOutDir: false 是为规避平台 safe-delete 拦截“批量清空 dist”，
      // 但已引发“公告弹窗残留”事故（过期 chunk 未清理）。现改回 true，构建前整体清空 dist。
      emptyOutDir: true,
      rollupOptions: {
        output: {
          manualChunks(id: string) {
            if (id.includes('node_modules')) {
              if (id.includes('vue') || id.includes('pinia')) return 'vue-vendor'
              if (id.includes('ant-design-vue') || id.includes('@ant-design'))
                return 'antd-vendor'
              if (id.includes('marked') || id.includes('dompurify') || id.includes('ofetch'))
                return 'utils-vendor'
              if (id.includes('motion-v')) return 'motion-vendor'
            }
          },
        },
      },
    },
    server: {
      // 自适应端口：dev 编排器（scripts/dev.mjs）注入预探测的空闲端口；
      // 未注入时用 5173。strictPort 保持 false，被占时 Vite 自动向上顺延。
      port: Number(process.env.VITE_PORT) || 5173,
      strictPort: false,
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/uploads': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/thumbs': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/images': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
  }
})
