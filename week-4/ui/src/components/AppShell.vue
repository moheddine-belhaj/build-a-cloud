<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'
import DatabaseIcon from '@/components/DatabaseIcon.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { theme, toggle } = useTheme()

const isInstances = computed(() => route.path.startsWith('/instances'))
const isUsers = computed(() => route.path.startsWith('/users'))
const isActivity = computed(() => route.path.startsWith('/activity'))

const initials = computed(() => {
  const email = auth.email ?? ''
  return email.slice(0, 2).toUpperCase() || '?'
})

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-screen bg-bg text-foreground">
    <aside class="flex w-56 flex-shrink-0 flex-col border-r border-border bg-surface px-3 py-4">
      <RouterLink to="/instances" class="flex items-center gap-2.5 px-2 pb-5 pt-1.5">
        <span class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M4 7 L12 3 L20 7 L12 11 Z" />
            <path d="M4 12 L12 16 L20 12" />
            <path d="M4 17 L12 21 L20 17" />
          </svg>
        </span>
        <span class="text-[14.5px] font-semibold tracking-tight text-foreground">OurSQL</span>
      </RouterLink>

      <nav class="flex flex-col gap-0.5">
        <RouterLink
          to="/instances"
          class="flex items-center gap-2.5 rounded-md border-l-2 px-2.5 py-2 text-[13.5px] font-medium transition-colors"
          :class="isInstances
            ? 'border-accent bg-accent-soft font-semibold text-accent'
            : 'border-transparent text-muted hover:bg-surface-hover hover:text-foreground'"
        >
          <DatabaseIcon :size="16" :class="isInstances ? 'opacity-100' : 'opacity-85'" />
          Instances
        </RouterLink>
        <RouterLink
          to="/users"
          class="flex items-center gap-2.5 rounded-md border-l-2 px-2.5 py-2 text-[13.5px] font-medium transition-colors"
          :class="isUsers
            ? 'border-accent bg-accent-soft font-semibold text-accent'
            : 'border-transparent text-muted hover:bg-surface-hover hover:text-foreground'"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="flex-shrink-0" :class="isUsers ? 'opacity-100' : 'opacity-85'">
            <circle cx="9" cy="8" r="3.2" />
            <path d="M3.5 19c0-3.3 2.5-5.5 5.5-5.5s5.5 2.2 5.5 5.5" />
            <circle cx="17.5" cy="9" r="2.4" />
            <path d="M15.7 13.6c2.3.3 4 2.1 4.1 4.9" />
          </svg>
          Users
        </RouterLink>
        <RouterLink
          to="/activity"
          class="flex items-center gap-2.5 rounded-md border-l-2 px-2.5 py-2 text-[13.5px] font-medium transition-colors"
          :class="isActivity
            ? 'border-accent bg-accent-soft font-semibold text-accent'
            : 'border-transparent text-muted hover:bg-surface-hover hover:text-foreground'"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" class="flex-shrink-0" :class="isActivity ? 'opacity-100' : 'opacity-85'">
            <circle cx="12" cy="12" r="9" />
            <path d="M12 7v5l3.5 2" />
          </svg>
          Activity
        </RouterLink>
      </nav>

      <div class="flex-1"></div>

      <div class="flex flex-col gap-2.5 border-t border-border pt-3">
        <button
          type="button"
          class="flex items-center justify-center gap-2 rounded-md border border-border bg-surface-2 px-2.5 py-1.5 text-xs font-semibold text-muted transition-colors hover:border-border-strong hover:text-foreground"
          @click="toggle"
        >
          <svg v-if="theme === 'dark'" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="12" cy="12" r="4.5" />
            <path d="M12 2.5v2.2M12 19.3v2.2M4.2 4.2l1.6 1.6M18.2 18.2l1.6 1.6M2.5 12h2.2M19.3 12h2.2M4.2 19.8l1.6-1.6M18.2 5.8l1.6-1.6" />
          </svg>
          <svg v-else width="13" height="13" viewBox="0 0 24 24" fill="currentColor" stroke="none">
            <path d="M20 14.5A8.5 8.5 0 0 1 9.5 4 8.5 8.5 0 1 0 20 14.5Z" />
          </svg>
          {{ theme === 'dark' ? 'Dark mode' : 'Light mode' }}
        </button>

        <div class="flex items-center gap-2.5 px-1">
          <span class="flex h-6.5 w-6.5 flex-shrink-0 items-center justify-center rounded-full border border-border-strong bg-surface-2 text-[11px] font-bold text-muted">
            {{ initials }}
          </span>
          <div class="flex min-w-0 flex-col leading-tight">
            <span class="truncate font-mono text-xs font-medium text-foreground">{{ auth.email }}</span>
            <button type="button" class="text-left text-[11.5px] text-faint hover:text-danger" @click="logout">
              Log out
            </button>
          </div>
        </div>
      </div>
    </aside>

    <main class="min-w-0 flex-1 overflow-y-auto">
      <slot />
    </main>
  </div>
</template>
