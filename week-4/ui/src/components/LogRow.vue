<script setup lang="ts">
import { computed } from 'vue'
import type { RequestLogEntry } from '@/lib/api'

const props = defineProps<{ entry: RequestLogEntry }>()

// Matches Go's default `log` package timestamp layout ("2006/01/02
// 15:04:05"), so the line reads exactly like the server's own stdout log.
const timestamp = computed(() => {
  const d = new Date(props.entry.createdAt)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
})

const duration = computed(() => `${props.entry.durationMs.toFixed(3)}ms`)

const statusClass = computed(() => {
  const status = props.entry.status
  if (status >= 500) return 'text-danger'
  if (status >= 400) return 'text-warning'
  if (status >= 200 && status < 300) return 'text-success'
  return 'text-muted'
})

const methodClass = computed(() => {
  switch (props.entry.method) {
    case 'GET':
      return 'bg-accent-soft text-accent'
    case 'POST':
      return 'bg-success-soft text-success'
    case 'PATCH':
    case 'PUT':
      return 'bg-warning-soft text-warning'
    case 'DELETE':
      return 'bg-danger-soft text-danger'
    default:
      return 'bg-surface text-muted'
  }
})
</script>

<template>
  <div class="flex items-center gap-2 px-4 py-1.5 font-mono text-xs">
    <span class="text-faint">{{ timestamp }}</span>
    <span class="shrink-0 rounded px-1.5 py-0.5 text-[11px] font-semibold" :class="methodClass">{{ entry.method }}</span>
    <span class="min-w-0 flex-1 truncate text-foreground">{{ entry.path }}</span>
    <span class="shrink-0 font-semibold" :class="statusClass">{{ entry.status }}</span>
    <span class="shrink-0 text-faint">({{ duration }})</span>
  </div>
</template>
