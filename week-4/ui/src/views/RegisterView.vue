<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ApiError } from '@/lib/api'

const auth = useAuthStore()
const router = useRouter()

const firstName = ref('')
const lastName = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.register({
      firstName: firstName.value,
      lastName: lastName.value,
      email: email.value,
      password: password.value,
    })
    await router.push('/instances')
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
        <span class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
          <svg width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M4 7 L12 3 L20 7 L12 11 Z" />
            <path d="M4 12 L12 16 L20 12" />
            <path d="M4 17 L12 21 L20 17" />
          </svg>
        </span>
        <span class="text-[30px] font-semibold tracking-tight text-foreground">OurSQL</span>
      </div>
      <h1 class="mb-6 text-xl font-semibold text-foreground">Create an account</h1>

      <form class="space-y-4" @submit.prevent="onSubmit">
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="firstName" class="mb-1 block text-sm font-medium text-muted"
              >First name</label
            >
            <input
              id="firstName"
              v-model="firstName"
              type="text"
              required
              autocomplete="given-name"
              class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
            />
          </div>
          <div>
            <label for="lastName" class="mb-1 block text-sm font-medium text-muted"
              >Last name</label
            >
            <input
              id="lastName"
              v-model="lastName"
              type="text"
              required
              autocomplete="family-name"
              class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
            />
          </div>
        </div>

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
            minlength="8"
            autocomplete="new-password"
            class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-2 focus:ring-accent-soft"
          />
          <p class="mt-1 text-xs text-faint">At least 8 characters.</p>
        </div>

        <p v-if="error" class="text-sm text-danger">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded-md bg-accent px-3 py-2 text-sm font-semibold text-accent-contrast hover:bg-accent-strong disabled:opacity-50"
        >
          {{ loading ? 'Creating account…' : 'Create account' }}
        </button>
      </form>

      <p class="mt-4 text-center text-sm text-muted">
        Already have an account?
        <RouterLink to="/login" class="font-medium text-accent hover:underline"
          >Sign in</RouterLink
        >
      </p>
    </div>
  </div>
</template>
