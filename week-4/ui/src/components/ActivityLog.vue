<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { requestLogsApi, ApiError, type RequestLogEntry } from '@/lib/api'
import LogRow from '@/components/LogRow.vue'

const props = defineProps<{ instanceId?: string }>()

const PAGE_SIZE = 20

const entries = ref<RequestLogEntry[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const loadError = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const offset = (page.value - 1) * PAGE_SIZE
    const result = props.instanceId
      ? await requestLogsApi.listForInstance(props.instanceId, { limit: PAGE_SIZE, offset })
      : await requestLogsApi.list({ limit: PAGE_SIZE, offset })
    entries.value = result.items
    total.value = result.total
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Failed to load activity.'
  } finally {
    loading.value = false
  }
}

function prevPage() {
  if (page.value > 1) page.value -= 1
}

function nextPage() {
  if (page.value < totalPages.value) page.value += 1
}

watch(() => props.instanceId, () => { page.value = 1; load() })
watch(page, load)
load()
</script>

<template>
  <div>
    <p v-if="loadError" class="mb-4 text-sm text-danger">{{ loadError }}</p>

    <div v-if="loading && entries.length === 0" class="text-sm text-muted">Loading…</div>

    <div
      v-else-if="entries.length === 0"
      class="rounded-lg border border-dashed border-border-strong px-6 py-12 text-center text-sm text-muted"
    >
      No requests recorded yet.
    </div>

    <template v-else>
      <div class="max-h-[32rem] overflow-y-auto rounded-lg border border-border bg-surface-2 shadow-sm">
        <LogRow v-for="entry in entries" :key="entry.id" :entry="entry" class="border-b border-border last:border-0" />
      </div>

      <div class="mt-4 flex items-center justify-center gap-3 text-sm">
        <button
          :disabled="page <= 1"
          class="rounded-md border border-border px-3 py-1.5 font-medium text-foreground hover:bg-surface-hover disabled:opacity-40"
          @click="prevPage"
        >
          Prev
        </button>
        <span class="text-muted">Page {{ page }} of {{ totalPages }}</span>
        <button
          :disabled="page >= totalPages"
          class="rounded-md border border-border px-3 py-1.5 font-medium text-foreground hover:bg-surface-hover disabled:opacity-40"
          @click="nextPage"
        >
          Next
        </button>
      </div>
    </template>
  </div>
</template>
