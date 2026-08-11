import { defineStore } from 'pinia'
import { api, getToken, setToken } from '../api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: getToken(),
    username: '',
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
  },
  actions: {
    async login(username, password) {
      const res = await api.post('/auth/login', { username, password })
      this.token = res.token
      this.username = res.username
      setToken(res.token)
    },
    logout() {
      this.token = ''
      this.username = ''
      setToken('')
    },
  },
})
