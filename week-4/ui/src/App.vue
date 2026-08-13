<script setup lang="ts">
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-slate-50">
    <header v-if="!auth.isAuthenticated" class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-4xl items-center justify-between px-4 py-3">
        <RouterLink to="/instances" class="text-sm font-semibold text-slate-900">
          PaaS Console
        </RouterLink>
        <div class="flex items-center gap-4 text-sm">
          <span class="text-slate-500">{{ auth.email }}</span>
          <button class="text-slate-500 hover:text-slate-900" @click="logout">Sign out</button>
        </div>
      </div>
    </header>

    <RouterView />
  </div>
</template>
