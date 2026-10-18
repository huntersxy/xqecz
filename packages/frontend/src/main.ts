import './assets/main.css'
import '@arco-design/web-vue/dist/arco.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/theme'
import { initWebVitals } from './utils/webVitals'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 主题状态必须在任何路由挂载前落到 html 与 body 上：详情页不经过 PublicShell，
// 若只在侧栏里实例化 store，直接刷新详情会只剩 index.html 预置的 html.dark，
// body 缺少 arco-theme，Arco 弹层与详情调色板仍是浅色。
useThemeStore(pinia)

app.mount('#app')

// 初始化 Web Vitals 性能监控
if (import.meta.env.PROD) {
  initWebVitals()
}
