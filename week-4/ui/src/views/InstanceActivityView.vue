<script setup lang="ts">
import { RouterLink } from 'vue-router'
import ActivityLog from '@/components/ActivityLog.vue'

const props = defineProps<{ id: string }>()
</script>

<template>
  <div class="mx-auto max-w-5xl px-6 py-8 lg:px-12">
    <RouterLink
      :to="`/instances/${id}`"
      class="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground"
    >
      <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 5l-7 7 7 7" /></svg>Back to instance
    </RouterLink>

    <h1 class="mb-1 font-mono text-2xl font-semibold text-foreground">{{ id }}</h1>
    <p class="mb-6 text-sm text-muted">The HTTP request log for this instance, newest first.</p>

    <ActivityLog :instance-id="id" />
  </div>
</template>
