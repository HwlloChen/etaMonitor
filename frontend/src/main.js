import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import 'mdui/mdui.css'
import 'mdui'
import './assets/main.css'
import App from './App.vue'
import Home from './views/Home.vue'
import Servers from './views/Servers.vue'
import ServerDetail from './views/ServerDetail.vue'
import Admin from './views/Admin.vue'
import PlayerDetail from './views/PlayerDetail.vue'
import NotFound from './views/NotFound.vue'
import { initWebSocket } from './utils/ws.js'
import { setTheme } from 'mdui'

const routes = [
  { path: '/', name: 'Home', component: Home },
  { path: '/servers', name: 'Servers', component: Servers },
  { path: '/server/:id', name: 'ServerDetail', component: ServerDetail },
  { path: '/player/:id', name: 'PlayerDetail', component: PlayerDetail },
  { path: '/admin', name: 'Admin', component: Admin },
  { path: '/:pathMatch(.*)*', name: 'NotFound', component: NotFound }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

const pinia = createPinia()

const app = createApp(App)
app.use(router)
app.use(pinia)

if (localStorage.getItem('theme')) {
  setTheme(localStorage.getItem('theme'))
} else {
  setTheme('auto')
  localStorage.setItem('theme', 'auto')
}

app.mount('#app')

// 在应用挂载后初始化WebSocket
setTimeout(() => {
  initWebSocket().catch(error => {
    console.error('WebSocket初始化失败:', error)
  })
}, 100)