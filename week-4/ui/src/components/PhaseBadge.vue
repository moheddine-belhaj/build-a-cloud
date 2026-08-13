<script setup lang="ts">
import { computed } from 'vue'
import type { InstancePhase } from '@/lib/api'

const props = defineProps<{ phase: InstancePhase }>()

// The API documents Provisioning/Healthy/Degraded/Deleting, but an
// older/not-yet-redeployed backend can still hand back CNPG's raw,
// free-text phase (e.g. "Cluster in healthy state") instead of the
// normalized enum. Normalize defensively here too, so the badge never
// shows an uncolored raw sentence regardless of backend deploy state.
const normalized = computed<InstancePhase>(() => {
  const lower = props.phase.toLowerCase()
  if (lower.includes('healthy')) return 'Healthy'
  if (lower.includes('unrecoverable') || lower.includes('error') || lower.includes('cannot determine')) {
    return 'Degraded'
  }
  if (lower.includes('delet')) return 'Deleting'
  // Any other phase — "Provisioning" itself, or a raw CNPG transient
  // string like "Setting up primary" — is a working/in-progress state.
  return 'Provisioning'
})

const classes = computed(() => {
  switch (normalized.value) {
    case 'Healthy':
      return 'bg-success-soft text-success'
    case 'Provisioning':
      return 'bg-warning-soft text-warning'
    case 'Degraded':
      return 'bg-danger-soft text-danger'
    case 'Deleting':
      return 'border border-border bg-surface text-faint'
  }
})
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-wide"
    :class="classes"
  >
    <span class="h-1.5 w-1.5 rounded-full bg-current"></span>
    {{ normalized }}
  </span>
</template>
