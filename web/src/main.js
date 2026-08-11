import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)

// Tunggu navigasi awal (termasuk guard redirect login<->dashboard) selesai
// dulu sebelum mount — kalau nggak, App.vue sempat render sekilas pakai
// route.name yang masih kosong (isLoginPage jadi false), jadi layout
// Sidebar+Dashboard nongol sebentar sebelum ke-flip ke halaman yang bener.
router.isReady().then(() => app.mount('#app'))
