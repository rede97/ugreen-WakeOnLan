import { createApp } from 'vue'
import App from './App.vue'

const app = createApp(App)

// UGOS Core SDK — available only inside UGREEN desktop environment
Promise.all([
  import('@ugreen-nas/core'),
  import('@ugreen-nas/core/cloudWindow'),
]).then(([core, cw]) => {
  core.default.init()
  cw.default.setTitle('WakeOnLan')
  app.config.globalProperties.$cloudWindow = cw.default
}).catch(() => {
  // Running outside UGREEN — SDK not available
})

app.mount('#app')
