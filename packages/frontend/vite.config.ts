import { fileURLToPath, URL } from 'node:url'

import { defineConfig, type ProxyOptions } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ArcoResolver } from 'unplugin-vue-components/resolvers'
import { ViteImageOptimizer } from 'vite-plugin-image-optimizer'

const buildDate = new Date().toISOString().split('T')[0]

export default defineConfig(async ({ mode }) => {
  const isDev = mode === 'development'
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
      resolvers: [ArcoResolver({ importStyle: false })],
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
      emptyOutDir: true,
      rollupOptions: {
        output: {
          manualChunks(id: string) {
            if (id.includes('node_modules')) {
              if (id.includes('vue') || id.includes('pinia')) return 'vue-vendor'
              if (id.includes('@arco-design')) return 'arco-vendor'
              if (id.includes('marked') || id.includes('dompurify') || id.includes('ofetch'))
                return 'utils-vendor'
            }
          },
        },
      },
    },
    server: {
      port: Number(process.env.VITE_PORT) || 5173,
      strictPort: false,
      proxy: {
        '^/__planet-media/(thumbs|uploads|images)/': {
          target: 'https://api39.xiey.work',
          changeOrigin: true,
          rewrite: (path: string) => path.replace(/^\/__planet-media/, ''),
          configure(proxy: Parameters<NonNullable<ProxyOptions['configure']>>[0]) {
            proxy.on('proxyReq', request => {
              request.removeHeader('origin')
              request.removeHeader('cookie')
              request.removeHeader('authorization')
            })
          },
        },
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
