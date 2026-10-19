import './assets/main.css'
import '@arco-design/web-vue/dist/arco.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { initWebVitals } from './utils/webVitals'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// 主题初始化、登录态刷新、整页过渡都归 App.vue；这里只负责装配与挂载。
app.mount('#app')

// 生产环境才采样 Web Vitals，开发态交给浏览器面板。
if (import.meta.env.PROD) {
  initWebVitals()
}
