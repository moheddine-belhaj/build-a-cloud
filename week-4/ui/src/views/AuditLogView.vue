<script setup lang="ts">
import ActivityLog from '@/components/ActivityLog.vue'
</script>

<template>
  <div class="mx-auto max-w-5xl px-6 py-10 lg:px-12">
    <h1 class="mb-1 text-2xl font-semibold text-foreground">Activity</h1>
    <p class="mb-6 text-sm text-muted">
      The HTTP request log for your account — every request across all of your instances, newest first.
    </p>

    <ActivityLog />
  </div>
</template>
