import { onMounted, onUnmounted, ref } from 'vue'

// useOnline: status online/offline BROWSER (navigator.onLine + event
// online/offline). Ini deteksi "kabel jaringan putus", BUKAN pengganti
// timeout di api/client.js — koneksi bisa "online" secara OS tapi tetap
// gak nyampe ke server (DNS/proxy/server down), makanya dua-duanya dipakai
// bareng: banner ini buat sinyal cepat, timeout buat jaring pengaman.
export function useOnline() {
  const isOnline = ref(navigator.onLine)

  function setOnline() {
    isOnline.value = true
  }
  function setOffline() {
    isOnline.value = false
  }

  onMounted(() => {
    window.addEventListener('online', setOnline)
    window.addEventListener('offline', setOffline)
  })
  onUnmounted(() => {
    window.removeEventListener('online', setOnline)
    window.removeEventListener('offline', setOffline)
  })

  return { isOnline }
}
