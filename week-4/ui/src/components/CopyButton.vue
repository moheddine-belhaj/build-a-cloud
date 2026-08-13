<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(defineProps<{ text: string; label?: string }>(), {
  label: 'copy',
})

const copied = ref(false)
let resetHandle: ReturnType<typeof setTimeout> | undefined

async function onClick() {
  await navigator.clipboard.writeText(props.text)
  copied.value = true
  clearTimeout(resetHandle)
  resetHandle = setTimeout(() => (copied.value = false), 1600)
}
</script>

<template>
  <button
    type="button"
    class="inline-flex shrink-0 items-center gap-1 text-xs font-medium transition-colors"
    :class="copied ? 'text-success' : 'text-faint hover:text-foreground'"
    @click="onClick"
  >
    <svg
      v-if="copied"
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2.4"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M5 13l4 4L19 7" />
    </svg>
    <svg
      v-else
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <rect x="8" y="8" width="12" height="12" rx="2" />
      <path d="M16 8V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3" />
    </svg>
    {{ copied ? 'Copied!' : label }}
  </button>
</template>
