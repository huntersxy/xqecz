import './assets/main.css'
import '@arco-design/web-vue/dist/arco.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { useSiteStore } from './stores/site'
import { initWebVitals } from './utils/webVitals'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

const siteStore = useSiteStore(pinia)
void siteStore.load()

app.mount('#app')

if (import.meta.env.PROD) {
  initWebVitals()
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => { void navigator.serviceWorker.register('/sw.js') })
  }
}
