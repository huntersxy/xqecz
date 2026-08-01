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

app.mount('#app')

// 初始化 Web Vitals 性能监控
if (import.meta.env.PROD) {
  initWebVitals()
}
