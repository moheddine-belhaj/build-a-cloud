<script setup lang="ts">
// external is optional: older API responses (or a frontend deployed ahead of
// the backend that added this field) may omit it — treat that as internal.
withDefaults(defineProps<{ external?: boolean }>(), { external: false })
</script>

<template>
  <span
    class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
    :class="external ? 'bg-sky-100 text-sky-800' : 'bg-slate-100 text-slate-600'"
  >
    {{ external ? 'Public' : 'Internal' }}
  </span>
</template>
