<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import AppShell from '@/components/AppShell.vue'

const auth = useAuthStore()
</script>

<template>
  <AppShell v-if="auth.isAuthenticated">
    <RouterView />
  </AppShell>
  <RouterView v-else />
</template>
