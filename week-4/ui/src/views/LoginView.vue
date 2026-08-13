<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/lib/api'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login({ email: email.value, password: password.value })
    const redirect = (route.query.redirect as string) || '/instances'
    await router.push(redirect)
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Something went wrong. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-bg px-4">
    <div class="w-full max-w-sm rounded-lg border border-border bg-surface-2 p-8 shadow-sm">
      <div class="mb-6 flex items-center gap-2.5">
        <span class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M4 7 L12 3 L20 7 L12 11 Z" />
            <path d="M4 12 L12 16 L20 12" />
            <path d="M4 17 L12 21 L20 17" />
          </svg>
        </span>
        <span class="text-[14.5px] font-semibold tracking-tight text-foreground">PaaS Console</span>
      </div>
      <h1 class="mb-6 text-xl font-semibold text-foreground">Sign in</h1>

      <form class="space-y-4" @submit.prevent="onSubmit">
        <div>
          <label for="email" class="mb-1 block text-sm font-medium text-muted">Email</label>
          <input
            id="email"
            v-model="email"
            type="email"
            required
            autocomplete="email"
            class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
        </div>

        <div>
          <label for="password" class="mb-1 block text-sm font-medium text-muted"
            >Password</label
          >
          <input
            id="password"
            v-model="password"
            type="password"
            required
            autocomplete="current-password"
            class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
        </div>

        <p v-if="error" class="text-sm text-danger">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded-md bg-accent px-3 py-2 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
        >
          {{ loading ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>

      <p class="mt-4 text-center text-sm text-muted">
        No account yet?
        <RouterLink to="/register" class="font-medium text-accent hover:underline"
          >Register</RouterLink
        >
      </p>
    </div>
  </div>
</template>
