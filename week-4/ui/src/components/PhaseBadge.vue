<script setup lang="ts">
import { computed } from 'vue'
import type { InstancePhase } from '@/lib/api'

const props = defineProps<{ phase: InstancePhase }>()

const classes = computed(() => {
  switch (props.phase) {
    case 'Healthy':
      return 'bg-green-100 text-green-800'
    case 'Provisioning':
      return 'bg-amber-100 text-amber-800'
    case 'Degraded':
      return 'bg-red-100 text-red-800'
    case 'Deleting':
      return 'bg-slate-200 text-slate-600'
    default:
      return 'bg-slate-100 text-slate-600'
  }
})
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
    :class="classes"
  >
    {{ phase }}
  </span>
</template>
