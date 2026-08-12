import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { authApi, getToken, setToken, type Credentials, type RegisterRequest } from '@/lib/api'

const EMAIL_KEY = 'paas_email'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(getToken())
  const email = ref<string | null>(localStorage.getItem(EMAIL_KEY))

  const isAuthenticated = computed(() => token.value !== null)

  function persist(newToken: string, newEmail: string) {
    token.value = newToken
    email.value = newEmail
    setToken(newToken)
    localStorage.setItem(EMAIL_KEY, newEmail)
  }

  async function login(credentials: Credentials) {
    const res = await authApi.login(credentials)
    persist(res.token, credentials.email)
  }

  async function register(body: RegisterRequest) {
    const res = await authApi.register(body)
    persist(res.token, body.email)
  }

  function logout() {
    token.value = null
    email.value = null
    setToken(null)
    localStorage.removeItem(EMAIL_KEY)
  }

  return { token, email, isAuthenticated, login, register, logout }
})
