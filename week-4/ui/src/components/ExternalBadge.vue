<script setup lang="ts">
// external is optional: older API responses (or a frontend deployed ahead of
// the backend that added this field) may omit it — treat that as internal.
withDefaults(defineProps<{ external?: boolean }>(), { external: false })
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-wide"
    :class="external ? 'bg-accent-soft text-accent' : 'border border-border bg-surface text-faint'"
  >
    {{ external ? 'External' : 'Internal' }}
  </span>
</template>
